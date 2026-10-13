package biz

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/pkg/outbox"
)

// stubOutboxStore records enqueued messages in place of PostgreSQL.
type stubOutboxStore struct {
	messages []outbox.Message
	enqueErr error
}

func (s *stubOutboxStore) Enqueue(ctx context.Context, tx outbox.Execer, msgs ...outbox.Message) error {
	if s.enqueErr != nil {
		return s.enqueErr
	}
	s.messages = append(s.messages, msgs...)
	return nil
}

func (s *stubOutboxStore) Claim(ctx context.Context, limit int) ([]*outbox.Message, error) {
	return nil, nil
}

func (s *stubOutboxStore) MarkDispatched(ctx context.Context, id uuid.UUID) error { return nil }

func (s *stubOutboxStore) MarkFailed(ctx context.Context, id uuid.UUID, lastErr error, retryAt time.Time, dead bool) error {
	return nil
}

// TestSettleOrderPublishesOutboxEvent locks in the atomicity contract: the
// order.settled event is produced inside the same settlement transaction, so a
// crash can never leave a paid order without its gate-opening event.
func TestSettleOrderPublishesOutboxEvent(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, RecordID: uuid.New(), LotID: uuid.New(),
		FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodWechat),
	}

	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, log.NewStdLogger(nopWriter{}))
	store := &stubOutboxStore{}
	uc.SetOutbox(store)

	outcome, err := uc.settleOrder(context.Background(), orderID, MethodWechat, "tx-1", 1500)
	if err != nil {
		t.Fatalf("settleOrder failed: %v", err)
	}
	if outcome.Duplicate {
		t.Fatal("the first settlement must not be a duplicate")
	}
	if repo.Orders[orderID].Status != string(StatusPaid) {
		t.Errorf("order status = %s, want paid", repo.Orders[orderID].Status)
	}
	if len(store.messages) != 1 {
		t.Fatalf("expected exactly one outbox message, got %d", len(store.messages))
	}

	msg := store.messages[0]
	if msg.Type != EventOrderSettled {
		t.Errorf("event type = %q, want %s", msg.Type, EventOrderSettled)
	}
	if msg.AggregateID != orderID.String() {
		t.Errorf("aggregate id = %q, want the order id", msg.AggregateID)
	}

	var event OrderSettledEvent
	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		t.Fatalf("payload is not valid JSON: %v", err)
	}
	if event.OrderID != orderID.String() || event.TransactionID != "tx-1" || event.AmountCents != 1500 {
		t.Errorf("event payload = %+v", event)
	}
}

// TestSettleOrderReportsOutboxFailure proves a failed event write fails the
// settlement rather than silently dropping the gate-opening side effect.
func TestSettleOrderReportsOutboxFailure(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodWechat),
	}

	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, log.NewStdLogger(nopWriter{}))
	uc.SetOutbox(&stubOutboxStore{enqueErr: errors.New("db down")})

	if _, err := uc.settleOrder(context.Background(), orderID, MethodWechat, "tx-1", 1500); err == nil {
		t.Fatal("an outbox failure must fail the settlement")
	}
}

// TestHandleOutboxMessageDispatchesSettledEvent covers the dispatcher handler:
// it reloads the order and applies the settlement side effects.
func TestHandleOutboxMessageDispatchesSettledEvent(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, RecordID: uuid.New(), LotID: uuid.New(),
		FinalAmount: 1500, Status: string(StatusPaid), PayMethod: string(MethodWechat),
		OrderType: OrderTypeParking,
	}

	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, log.NewStdLogger(nopWriter{}))

	payload, err := json.Marshal(OrderSettledEvent{
		OrderID: orderID.String(), AmountCents: 1500, Method: "wechat", TransactionID: "tx-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := uc.HandleOutboxMessage(context.Background(), &outbox.Message{
		Type: EventOrderSettled, AggregateID: orderID.String(), Payload: payload,
	}); err != nil {
		t.Fatalf("HandleOutboxMessage failed: %v", err)
	}
}

// TestHandleOutboxMessageGuards covers the malformed and unknown-message paths,
// which must never be retried forever.
func TestHandleOutboxMessageGuards(t *testing.T) {
	uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, log.NewStdLogger(nopWriter{}))

	t.Run("malformed payload dead-letters", func(t *testing.T) {
		err := uc.HandleOutboxMessage(context.Background(), &outbox.Message{
			Type: EventOrderSettled, Payload: []byte("{not json"),
		})
		if err == nil {
			t.Error("a malformed payload must fail so the message dead-letters")
		}
	})

	t.Run("invalid order id dead-letters", func(t *testing.T) {
		payload, _ := json.Marshal(OrderSettledEvent{OrderID: "not-a-uuid"})
		err := uc.HandleOutboxMessage(context.Background(), &outbox.Message{
			Type: EventOrderSettled, Payload: payload,
		})
		if err == nil {
			t.Error("an invalid order id must fail so the message dead-letters")
		}
	})

	t.Run("unknown event type is dropped", func(t *testing.T) {
		err := uc.HandleOutboxMessage(context.Background(), &outbox.Message{
			Type: "some.other.event", Payload: []byte("{}"),
		})
		if err != nil {
			t.Errorf("an unknown event type must be dropped without error, got %v", err)
		}
	})

	t.Run("missing order is dropped", func(t *testing.T) {
		payload, _ := json.Marshal(OrderSettledEvent{OrderID: uuid.New().String()})
		err := uc.HandleOutboxMessage(context.Background(), &outbox.Message{
			Type: EventOrderSettled, Payload: payload,
		})
		if err != nil {
			t.Errorf("a vanished order must be dropped without error, got %v", err)
		}
	})
}

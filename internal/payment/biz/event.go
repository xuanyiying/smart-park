package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/pkg/database"
	"github.com/xuanyiying/smart-park/pkg/outbox"
)

// Event types produced by the payment service.
const EventOrderSettled = "order.settled"

// OrderSettledEvent is published to the outbox in the same transaction that
// marks an order paid.
//
// Downstream side effects of a settlement — opening the gate, informing the
// owner — must not run inline on the callback path: a crash between the order
// update and the side effect used to lose it for good. With the event in the
// outbox, a dispatcher redelivers until the side effect succeeds, and the
// handlers stay idempotent so redelivery is safe.
type OrderSettledEvent struct {
	OrderID       string    `json:"order_id"`
	RecordID      string    `json:"record_id"`
	PlateNumber   string    `json:"plate_number"`
	LotID         string    `json:"lot_id"`
	AmountCents   int64     `json:"amount_cents"`
	Method        string    `json:"method"`
	TransactionID string    `json:"transaction_id"`
	PaidAt        time.Time `json:"paid_at"`
}

// SetOutbox attaches a transactional outbox to the use case.
//
// When no outbox is attached (tests, minimal deployments) settlements fall back
// to running the gate-open side effect inline, preserving the previous
// behaviour. When one is attached, the settlement transaction publishes
// order.settled and a dispatcher applies the side effects with retries.
func (uc *PaymentUseCase) SetOutbox(store outbox.Store) {
	uc.outbox = store
}

// SetChargingClient attaches the charging service used to confirm sessions
// after charging orders settle.
func (uc *PaymentUseCase) SetChargingClient(client ChargingPaymentClient) {
	uc.chargingClient = client
}

// entTxSource is the subset of an Ent transaction the outbox needs. Declared as
// an interface so the biz layer keeps its distance from generated Ent types:
// database.TxFromCtx hands back an *ent.Tx, whose raw handle comes through
// UnderlyingSQLTx.
type entTxSource interface {
	UnderlyingSQLTx() *sql.Tx
}

// publishOrderSettled appends an order.settled event inside the caller's
// transaction. The handle comes from the context the transaction manager
// installed; if it cannot be reached the event is written on its own
// connection, which degrades the guarantee from "atomic with the settlement" to
// "written shortly after" and is logged for visibility.
func (uc *PaymentUseCase) publishOrderSettled(ctx context.Context, order *Order, method PayMethod, transactionID string) error {
	payload, err := json.Marshal(OrderSettledEvent{
		OrderID:       order.ID.String(),
		RecordID:      order.RecordID.String(),
		PlateNumber:   order.PlateNumber,
		LotID:         order.LotID.String(),
		// order.FinalAmount is already cents (分).
		AmountCents:   order.FinalAmount,
		Method:        string(method),
		TransactionID: transactionID,
		PaidAt:        time.Now(),
	})
	if err != nil {
		return fmt.Errorf("encode order.settled event: %w", err)
	}

	msg := outbox.NewMessage("order", order.ID.String(), EventOrderSettled,
		payload, map[string]string{"lot_id": order.LotID.String()})

	var tx *sql.Tx
	if raw := database.TxFromCtx(ctx); raw != nil {
		if source, ok := raw.(entTxSource); ok {
			tx = source.UnderlyingSQLTx()
		}
	}
	if tx == nil {
		uc.log.WithContext(ctx).Warn("outbox event written outside the settlement transaction")
	}
	return outbox.Enqueue(ctx, uc.outbox, tx, msg)
}

// HandleOutboxMessage dispatches one claimed outbox message. It is the handler
// registered with the outbox Dispatcher at service start-up.
func (uc *PaymentUseCase) HandleOutboxMessage(ctx context.Context, msg *outbox.Message) error {
	switch msg.Type {
	case EventOrderSettled:
		var event OrderSettledEvent
		if err := json.Unmarshal(msg.Payload, &event); err != nil {
			// A malformed payload can never succeed, so retrying only burns
			// attempts. Fail the delivery so the message dead-letters where an
			// operator can inspect it.
			return fmt.Errorf("decode order.settled event: %w", err)
		}
		orderID, err := uuid.Parse(event.OrderID)
		if err != nil {
			return fmt.Errorf("order.settled event carries an invalid order id %q: %w", event.OrderID, err)
		}
		order, err := uc.orderRepo.GetOrder(ctx, orderID)
		if err != nil {
			return fmt.Errorf("load settled order %s: %w", event.OrderID, err)
		}
		if order == nil {
			// The order vanished (purged, or the event predates a reset);
			// redelivery cannot fix that, so treat it as handled.
			uc.log.WithContext(ctx).Warnf("order %s not found for settled event, dropping", event.OrderID)
			return nil
		}
		return uc.applyOrderSettledSideEffects(ctx, order)
	default:
		uc.log.WithContext(ctx).Warnf("outbox message of unknown type %q dropped", msg.Type)
		return nil
	}
}

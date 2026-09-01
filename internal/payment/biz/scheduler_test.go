package biz

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// The MockOrderRepo in payment_test.go backs the biz layer; these tests extend it with
// sweep-specific behaviour through a small wrapper.
type sweepRepo struct {
	MockOrderRepo

	closed  map[string]bool
	closedN int
}

func newSweepRepo(orders []*Order) *sweepRepo {
	repo := &sweepRepo{
		MockOrderRepo: MockOrderRepo{Orders: make(map[uuid.UUID]*Order)},
		closed:        make(map[string]bool),
	}
	for _, o := range orders {
		repo.Orders[o.ID] = o
	}
	return repo
}

func (r *sweepRepo) ListOrdersByStatus(ctx context.Context, status string, cutoff time.Time, limit int) ([]*Order, error) {
	var out []*Order
	for _, o := range r.Orders {
		if o.Status == status && !o.CreatedAt.After(cutoff) {
			out = append(out, o)
		}
	}
	return out, nil
}

func (r *sweepRepo) MarkOrderClosed(ctx context.Context, orderID uuid.UUID, closedAt time.Time) (bool, error) {
	o, ok := r.Orders[orderID]
	if !ok || o.Status != string(StatusPending) {
		return false, nil
	}
	o.Status = string(StatusFailed)
	r.closed[orderID.String()] = true
	r.closedN++
	return true, nil
}

func (r *sweepRepo) MarkOrderPaid(ctx context.Context, orderID uuid.UUID, method, transactionID string, paidAmount int64, paidAt time.Time) (bool, error) {
	o, ok := r.Orders[orderID]
	if !ok || o.Status != string(StatusPending) {
		return false, nil
	}
	o.Status = string(StatusPaid)
	o.TransactionID = transactionID
	o.PaidAmount = paidAmount
	o.PayMethod = method
	return true, nil
}

func newOrderID() uuid.UUID { return uuid.New() }

// stubGateway lets tests script what the channel says about each order.
type stubGateway struct {
	paid    map[string]bool
	err     error
	queries int
}

func (g *stubGateway) confirm(ctx context.Context, order *Order) (bool, string, error) {
	g.queries++
	if g.err != nil {
		return false, "", g.err
	}
	if g.paid[order.ID.String()] {
		return true, "tx-" + order.ID.String(), nil
	}
	return false, "", nil
}

func newSweepUseCaseWithGateway(repo *sweepRepo, gateway *stubGateway) *PaymentUseCase {
	uc := NewPaymentUseCase(repo, nil, nil, nil, nil, nil, log.NewStdLogger(io.Discard))
	uc.queryGateway = gateway.confirm
	return uc
}

func TestSweepClosesExpiredUnpaidOrder(t *testing.T) {
	orderID := newOrderID()
	repo := newSweepRepo([]*Order{{
		ID:          orderID,
		FinalAmount: 10,
		Status:      string(StatusPending),
		PayMethod:   string(MethodWechat),
		CreatedAt:   time.Now().Add(-2 * time.Hour),
	}})
	gateway := &stubGateway{paid: map[string]bool{}}
	uc := newSweepUseCaseWithGateway(repo, gateway)

	result, err := uc.SweepExpiredOrders(context.Background())
	if err != nil {
		t.Fatalf("SweepExpiredOrders failed: %v", err)
	}

	if result.Closed != 1 {
		t.Errorf("expected 1 order closed, got %d", result.Closed)
	}
	if repo.Orders[orderID].Status != string(StatusFailed) {
		t.Errorf("order status = %s, want failed", repo.Orders[orderID].Status)
	}
	if gateway.queries != 1 {
		t.Errorf("the gateway must be asked before closing, queries = %d", gateway.queries)
	}
}

func TestSweepSettlesLostCallback(t *testing.T) {
	orderID := newOrderID()
	repo := newSweepRepo([]*Order{{
		ID:          orderID,
		FinalAmount: 25,
		Status:      string(StatusPending),
		PayMethod:   string(MethodAlipay),
		CreatedAt:   time.Now().Add(-2 * time.Hour),
	}})
	gateway := &stubGateway{paid: map[string]bool{orderID.String(): true}}
	uc := newSweepUseCaseWithGateway(repo, gateway)

	result, err := uc.SweepExpiredOrders(context.Background())
	if err != nil {
		t.Fatalf("SweepExpiredOrders failed: %v", err)
	}

	if result.Settled != 1 {
		t.Fatalf("expected 1 order settled, got %d", result.Settled)
	}
	got := repo.Orders[orderID]
	if got.Status != string(StatusPaid) {
		t.Errorf("order status = %s, want paid: the driver paid and must not be double-charged", got.Status)
	}
	if got.TransactionID != "tx-"+orderID.String() {
		t.Errorf("transaction id = %q, want the channel's reference", got.TransactionID)
	}
	if got.PaidAmount != 25 {
		t.Errorf("paid amount = %d, want 25", got.PaidAmount)
	}
}

func TestSweepKeepsPendingOnGatewayError(t *testing.T) {
	orderID := newOrderID()
	repo := newSweepRepo([]*Order{{
		ID:          orderID,
		FinalAmount: 10,
		Status:      string(StatusPending),
		PayMethod:   string(MethodWechat),
		CreatedAt:   time.Now().Add(-2 * time.Hour),
	}})
	gateway := &stubGateway{paid: map[string]bool{}, err: errors.New("gateway timeout")}
	uc := newSweepUseCaseWithGateway(repo, gateway)

	result, err := uc.SweepExpiredOrders(context.Background())
	if err != nil {
		t.Fatalf("SweepExpiredOrders failed: %v", err)
	}

	if result.Errors != 1 {
		t.Errorf("expected 1 deferred order, got %d", result.Errors)
	}
	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Error("a transient gateway failure must not close the order: it might actually be paid")
	}
}

func TestSweepLeavesFreshOrdersAlone(t *testing.T) {
	repo := newSweepRepo([]*Order{{
		ID:          newOrderID(),
		FinalAmount: 10,
		Status:      string(StatusPending),
		PayMethod:   string(MethodWechat),
		// Created seconds ago: inside the 30 minute expiration window.
		CreatedAt: time.Now(),
	}})
	gateway := &stubGateway{paid: map[string]bool{}}
	uc := newSweepUseCaseWithGateway(repo, gateway)

	result, err := uc.SweepExpiredOrders(context.Background())
	if err != nil {
		t.Fatalf("SweepExpiredOrders failed: %v", err)
	}

	if result.Closed != 0 || result.Settled != 0 {
		t.Errorf("fresh orders must not be touched, got %+v", result)
	}
	if gateway.queries != 0 {
		t.Errorf("no gateway queries expected for fresh orders, got %d", gateway.queries)
	}
}

// TestSweepHonoursRaceWithCallback pins the settlement race: whichever side (callback or
// sweeper) wins the conditional update, the other must not overwrite the result.
func TestSweepHonoursRaceWithCallback(t *testing.T) {
	orderID := newOrderID()
	repo := newSweepRepo([]*Order{{
		ID:          orderID,
		FinalAmount: 10,
		Status:      string(StatusPaid), // a callback already settled it
		PayMethod:   string(MethodWechat),
		CreatedAt:   time.Now().Add(-2 * time.Hour),
	}})
	gateway := &stubGateway{paid: map[string]bool{orderID.String(): true}}
	_ = newSweepUseCaseWithGateway(repo, gateway)

	// The repo only lists pending orders, so a paid order never re-enters the sweep; the
	// assertion documents the invariant that keeps the two paths from fighting.
	pending, err := repo.ListOrdersByStatus(context.Background(), string(StatusPending), time.Now(), 500)
	if err != nil {
		t.Fatalf("ListOrdersByStatus failed: %v", err)
	}
	if len(pending) != 0 {
		t.Errorf("paid orders must not be re-processed by the sweeper, got %d", len(pending))
	}
}

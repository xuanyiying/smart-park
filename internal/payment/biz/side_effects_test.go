package biz

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	alipaysdk "github.com/smartwalle/alipay/v3"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	"github.com/xuanyiying/smart-park/internal/payment/alipay"
)

// failingChargingClient fails every confirmation.
type failingChargingClient struct{}

func (failingChargingClient) ConfirmChargingPayment(ctx context.Context, sessionID, transactionID, paymentMethod string, paidAmount int64) error {
	return errors.New("charging service down")
}

// recordingGate records gate-release calls.
type recordingGate struct {
	opened  []string
	openErr error
}

func (g *recordingGate) OpenGate(ctx context.Context, deviceID string, recordID string) error {
	g.opened = append(g.opened, deviceID)
	return g.openErr
}

// TestTriggerAutoGateOpen covers the post-settlement gate release: the record
// status is flipped to paid before the barrier is released, and every failure
// mode degrades safely.
func TestTriggerAutoGateOpen(t *testing.T) {
	recordID := uuid.New()
	order := &Order{
		ID: uuid.New(), RecordID: recordID, FinalAmount: 1500,
		Status: string(StatusPaid), PayMethod: string(MethodWechat), TransactionID: "tx-1",
	}
	record := &ParkingRecordInfo{ID: recordID.String(), ExitDeviceID: "CAM003", FinalAmount: 1500}

	t.Run("no gate client skips silently", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), &scriptedRecordRepo{record: record}, nil, &PaymentConfig{}, nil, nil, testLogger())
		if err := uc.triggerAutoGateOpen(context.Background(), order); err != nil {
			t.Errorf("a deployment without gate control must not fail the settlement: %v", err)
		}
	})

	t.Run("record missing skips silently", func(t *testing.T) {
		gate := &recordingGate{}
		uc := NewPaymentUseCase(NewMockOrderRepo(), &scriptedRecordRepo{}, gate, &PaymentConfig{}, nil, nil, testLogger())
		if err := uc.triggerAutoGateOpen(context.Background(), order); err != nil {
			t.Errorf("a vanished record must not fail the settlement: %v", err)
		}
		if len(gate.opened) != 0 {
			t.Error("no gate may open for a missing record")
		}
	})

	t.Run("record without exit device skips silently", func(t *testing.T) {
		gate := &recordingGate{}
		noDevice := &ParkingRecordInfo{ID: recordID.String(), FinalAmount: 1500}
		uc := NewPaymentUseCase(NewMockOrderRepo(), &scriptedRecordRepo{record: noDevice}, gate, &PaymentConfig{}, nil, nil, testLogger())
		if err := uc.triggerAutoGateOpen(context.Background(), order); err != nil {
			t.Errorf("a record without an exit device must not fail: %v", err)
		}
		if len(gate.opened) != 0 {
			t.Error("no gate may open without an exit device")
		}
	})

	t.Run("happy path flips status then opens", func(t *testing.T) {
		gate := &recordingGate{}
		recordRepo := &scriptedRecordRepo{record: record}
		uc := NewPaymentUseCase(NewMockOrderRepo(), recordRepo, gate, &PaymentConfig{}, nil, nil, testLogger())

		if err := uc.triggerAutoGateOpen(context.Background(), order); err != nil {
			t.Fatalf("triggerAutoGateOpen failed: %v", err)
		}
		if len(recordRepo.statuses) != 1 || recordRepo.statuses[0] != "paid" {
			t.Errorf("record statuses = %v, want [paid] written before the gate opens", recordRepo.statuses)
		}
		if len(gate.opened) != 1 || gate.opened[0] != "CAM003" {
			t.Errorf("gate opened for %v, want [CAM003]", gate.opened)
		}
	})

	t.Run("gate failure surfaces for retry", func(t *testing.T) {
		gate := &recordingGate{openErr: errors.New("device offline")}
		recordRepo := &scriptedRecordRepo{record: record}
		uc := NewPaymentUseCase(NewMockOrderRepo(), recordRepo, gate, &PaymentConfig{}, nil, nil, testLogger())

		if err := uc.triggerAutoGateOpen(context.Background(), order); err == nil {
			t.Error("a gate failure must surface so the dispatcher can retry")
		}
		// The status flip must still have happened: the driver paid.
		if len(recordRepo.statuses) != 1 || recordRepo.statuses[0] != "paid" {
			t.Error("the paid status must be written even when the gate command fails")
		}
	})

	t.Run("record lookup error surfaces", func(t *testing.T) {
		gate := &recordingGate{}
		uc := NewPaymentUseCase(NewMockOrderRepo(), &scriptedRecordRepo{err: errors.New("db down")}, gate, &PaymentConfig{}, nil, nil, testLogger())
		if err := uc.triggerAutoGateOpen(context.Background(), order); err == nil {
			t.Error("a record lookup failure must surface")
		}
	})
}

// TestConfirmChargingSession covers the charging side effect.
func TestConfirmChargingSession(t *testing.T) {
	sessionID := uuid.New()
	order := &Order{
		ID: uuid.New(), RecordID: sessionID, OrderType: OrderTypeCharging,
		PayMethod: "wechat", TransactionID: "tx-1", PaidAmount: 2500,
	}

	t.Run("no charging client skips silently", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())
		if err := uc.confirmChargingSession(context.Background(), order); err != nil {
			t.Errorf("a deployment without charging must not fail: %v", err)
		}
	})

	t.Run("confirmation failure surfaces", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())
		uc.SetChargingClient(failingChargingClient{})
		if err := uc.confirmChargingSession(context.Background(), order); err == nil {
			t.Error("a failed charging confirmation must surface")
		}
	})
}

// TestGenerateAlipayPayment drives the alipay QR-code payment against a
// redirected gateway: the amount must be converted from cents to a yuan string.
func TestGenerateAlipayPayment(t *testing.T) {
	key, privatePEM, publicPEM := alipayKeyPairPEM(t)

	var form map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		form = map[string]interface{}{"biz_content": r.PostForm.Get("biz_content")}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(signedAlipayGatewayResponse(t, key, "alipay_trade_precreate_response", map[string]interface{}{
			"code": "10000", "msg": "Success", "out_trade_no": "order-1", "qr_code": "https://qr.alipay.com/bax00001",
		})))
	}))
	defer srv.Close()

	sdk, err := alipaysdk.New("2021000000000000", privatePEM, false)
	if err != nil {
		t.Fatalf("failed to build sdk client: %v", err)
	}
	if err := sdk.LoadAliPayPublicKey(publicPEM); err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}
	sdk.Client = &http.Client{Transport: gatewayRedirect{base: srv.URL}}

	client := alipay.NewClientWithGateway(sdk, &alipay.Config{
		AppID: "2021000000000000", PrivateKey: privatePEM, AlipayPublicKey: publicPEM,
		SignType: alipay.SignTypeRSA2,
	})

	uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(),
		&PaymentConfig{AlipayPublicKey: publicPEM, AlipaySignType: alipay.SignTypeRSA2}, nil, client, testLogger())

	data, err := uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
		RecordId: uuid.New().String(), Amount: 1500, PayMethod: "alipay",
	})
	if err != nil {
		t.Fatalf("CreatePayment failed: %v", err)
	}
	if data.QrCode != "https://qr.alipay.com/bax00001" {
		t.Errorf("qr code = %q, want the gateway's code url", data.QrCode)
	}
	if biz, _ := form["biz_content"].(string); !contains(biz, `"total_amount":"15.00"`) {
		t.Errorf("biz_content = %q, want total_amount 15.00 (cents converted to yuan)", biz)
	}
}

// TestRunOrderSweeperLifecycle drives the background loop: it performs a sweep
// on each tick and exits cleanly when stopped.
func TestRunOrderSweeperLifecycle(t *testing.T) {
	repo := newSweepRepo([]*Order{{
		ID: newOrderID(), FinalAmount: 10, Status: string(StatusPending),
		PayMethod: string(MethodWechat), CreatedAt: time.Now().Add(-2 * time.Hour),
	}})
	gateway := &stubGateway{paid: map[string]bool{}}
	uc := newSweepUseCaseWithGateway(repo, gateway)

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		uc.RunOrderSweeper(context.Background(), 20*time.Millisecond, stop)
		close(done)
	}()

	// Let at least one tick fire.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if gateway.queryCount() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if gateway.queryCount() == 0 {
		t.Fatal("the sweeper never queried the gateway")
	}

	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the sweeper did not stop after the stop signal")
	}
}

// TestRunOrderSweeperDefaultsInterval proves a non-positive interval falls back
// to a sane default instead of spinning.
func TestRunOrderSweeperDefaultsInterval(t *testing.T) {
	repo := newSweepRepo(nil)
	uc := newSweepUseCaseWithGateway(repo, &stubGateway{paid: map[string]bool{}})

	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		uc.RunOrderSweeper(context.Background(), 0, stop)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	close(stop)
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("the sweeper did not stop")
	}
}

// TestConfirmWithGatewayOrderDefaultPath covers the sweeper's own "channel not
// configured" decision: with no scripted gateway and no clients, orders stay
// pending rather than being closed.
func TestConfirmWithGatewayOrderDefaultPath(t *testing.T) {
	repo := newSweepRepo([]*Order{{
		ID: newOrderID(), FinalAmount: 10, Status: string(StatusPending),
		PayMethod: string(MethodWechat), CreatedAt: time.Now().Add(-2 * time.Hour),
	}})
	uc := NewPaymentUseCase(repo, nil, nil, nil, nil, nil, testLogger())

	result, err := uc.SweepExpiredOrders(context.Background())
	if err != nil {
		t.Fatalf("SweepExpiredOrders failed: %v", err)
	}
	if result.Errors != 1 || result.Closed != 0 {
		t.Errorf("an unconfigured channel must defer the order, got %+v", result)
	}
}

// TestSettleFromSweepOpensGate proves the sweep's补单 path also releases the
// barrier after recording the payment.
func TestSettleFromSweepOpensGate(t *testing.T) {
	recordID := uuid.New()
	repo := newSweepRepo(nil)
	order := &Order{
		ID: newOrderID(), RecordID: recordID, FinalAmount: 1500,
		Status: string(StatusPending), PayMethod: string(MethodWechat),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	repo.Orders[order.ID] = order

	gate := &recordingGate{}
	recordRepo := &scriptedRecordRepo{record: &ParkingRecordInfo{ID: recordID.String(), ExitDeviceID: "CAM003", FinalAmount: 1500}}
	uc := NewPaymentUseCase(repo, recordRepo, gate, nil, nil, nil, testLogger())
	uc.queryGateway = func(ctx context.Context, o *Order) (bool, string, error) {
		return true, "tx-swept", nil
	}

	result, err := uc.SweepExpiredOrders(context.Background())
	if err != nil {
		t.Fatalf("SweepExpiredOrders failed: %v", err)
	}
	if result.Settled != 1 {
		t.Fatalf("expected 1 settled order, got %+v", result)
	}
	if len(gate.opened) != 1 || gate.opened[0] != "CAM003" {
		t.Errorf("gate opened for %v, want [CAM003]", gate.opened)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

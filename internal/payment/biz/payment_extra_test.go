package biz

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

// scriptedRecordRepo returns a configurable parking record for the server-side
// amount authority check.
type scriptedRecordRepo struct {
	record   *ParkingRecordInfo
	err      error
	statuses []string
}

func (r *scriptedRecordRepo) GetRecord(ctx context.Context, recordID string) (*ParkingRecordInfo, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.record, nil
}

func (r *scriptedRecordRepo) UpdateRecordStatus(ctx context.Context, recordID, status string) error {
	r.statuses = append(r.statuses, status)
	return nil
}

// failingOrderRepo fails order creation to cover that path.
type failingOrderRepo struct {
	MockOrderRepo
	createErr error
}

func (r *failingOrderRepo) CreateOrder(ctx context.Context, order *Order) error {
	if r.createErr != nil {
		return r.createErr
	}
	return r.MockOrderRepo.CreateOrder(ctx, order)
}

func testLogger() log.Logger { return log.NewStdLogger(nopWriter{}) }

// TestCreatePaymentServerAmountAuthority locks in the money-integrity rule: the
// order amount is the server-side exit fee, never the client-reported one.
func TestCreatePaymentServerAmountAuthority(t *testing.T) {
	recordID := uuid.New()
	repo := NewMockOrderRepo()
	recordRepo := &scriptedRecordRepo{record: &ParkingRecordInfo{
		ID: recordID.String(), ExitDeviceID: "CAM003", FinalAmount: 1500,
	}}

	uc := NewPaymentUseCase(repo, recordRepo, NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())

	// Client claims 1 cent; the server computed 15.00 at exit time.
	_, err := uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
		RecordId: recordID.String(), Amount: 1, PayMethod: "alipay",
	})
	if err == nil {
		t.Fatal("expected the alipay client to be missing")
	}

	order, err := repo.GetOrderByRecordID(context.Background(), recordID)
	if err != nil || order == nil {
		t.Fatalf("the order must have been created before payment URL generation: %v", err)
	}
	if order.FinalAmount != 1500 {
		t.Errorf("order amount = %d, want the server-side 1500 cents", order.FinalAmount)
	}
}

// TestCreatePaymentIdempotentForExistingOrders covers both reuse branches: a
// paid order returns its existing details, a pending one is reused.
func TestCreatePaymentIdempotentForExistingOrders(t *testing.T) {
	recordID := uuid.New()

	t.Run("paid order returns existing details", func(t *testing.T) {
		repo := NewMockOrderRepo()
		existing := &Order{
			ID: uuid.New(), RecordID: recordID, FinalAmount: 1500,
			Status: string(StatusPaid), PayMethod: string(MethodWechat),
		}
		repo.Orders[existing.ID] = existing

		uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())
		data, err := uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
			RecordId: recordID.String(), Amount: 1500, PayMethod: "wechat",
		})
		if err != nil {
			t.Fatalf("CreatePayment failed: %v", err)
		}
		if data.OrderId != existing.ID.String() {
			t.Errorf("order id = %q, want the existing order", data.OrderId)
		}
		if data.Amount != 1500 {
			t.Errorf("amount = %d, want 1500", data.Amount)
		}
		if data.PayUrl != "" {
			t.Error("a paid order must not generate a new payment URL")
		}
		if len(repo.Orders) != 1 {
			t.Error("no new order may be created for a paid record")
		}
	})

	t.Run("pending order is reused", func(t *testing.T) {
		repo := NewMockOrderRepo()
		existing := &Order{
			ID: uuid.New(), RecordID: recordID, FinalAmount: 1500,
			Status: string(StatusPending), PayMethod: string(MethodWechat),
		}
		repo.Orders[existing.ID] = existing

		uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())
		// WeChat client missing: the reused order still surfaces before the gateway call.
		if _, err := uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
			RecordId: recordID.String(), Amount: 1500, PayMethod: "wechat",
		}); err == nil {
			t.Fatal("expected the missing wechat client to fail the URL generation")
		}
		if len(repo.Orders) != 1 {
			t.Error("a pending order must be reused, not duplicated")
		}
	})
}

// TestCreatePaymentChargingOrder covers the charging session variant, where the
// record id carries the session id and the client-reported amount is trusted
// (the charging service computes it).
func TestCreatePaymentChargingOrder(t *testing.T) {
	sessionID := uuid.New()
	repo := NewMockOrderRepo()

	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())

	_, err := uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
		SessionId: sessionID.String(), Amount: 2500, PayMethod: "wechat", OrderType: "charging",
	})
	if err == nil {
		t.Fatal("expected the missing wechat client to fail")
	}

	order, err := repo.GetOrderByRecordID(context.Background(), sessionID)
	if err != nil || order == nil {
		t.Fatalf("the charging order must be created: %v", err)
	}
	if order.OrderType != OrderTypeCharging {
		t.Errorf("order type = %q, want charging", order.OrderType)
	}
	if order.FinalAmount != 2500 {
		t.Errorf("amount = %d, want the client-reported 2500 for charging", order.FinalAmount)
	}
}

func TestCreatePaymentInvalidChargingSession(t *testing.T) {
	uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())

	if _, err := uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
		SessionId: "not-a-uuid", Amount: 2500, PayMethod: "wechat", OrderType: "charging",
	}); err == nil {
		t.Error("an invalid session id must be rejected")
	}
}

func TestCreatePaymentRepoFailure(t *testing.T) {
	repo := &failingOrderRepo{
		MockOrderRepo: MockOrderRepo{Orders: make(map[uuid.UUID]*Order)},
		createErr:     errors.New("db down"),
	}
	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())

	if _, err := uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
		RecordId: uuid.New().String(), Amount: 1500, PayMethod: "alipay",
	}); err == nil {
		t.Error("a repository failure must surface")
	}
}

// TestServerAmountForRecord covers every branch of the authority lookup.
func TestServerAmountForRecord(t *testing.T) {
	recordID := uuid.New()

	t.Run("nil repo", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), nil, nil, &PaymentConfig{}, nil, nil, testLogger())
		if _, ok := uc.serverAmountForRecord(context.Background(), recordID); ok {
			t.Error("no record repo must mean no authoritative amount")
		}
	})

	t.Run("record missing", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), &scriptedRecordRepo{}, nil, &PaymentConfig{}, nil, nil, testLogger())
		if _, ok := uc.serverAmountForRecord(context.Background(), recordID); ok {
			t.Error("a missing record must mean no authoritative amount")
		}
	})

	t.Run("record without amount", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), &scriptedRecordRepo{record: &ParkingRecordInfo{ID: recordID.String()}}, nil, &PaymentConfig{}, nil, nil, testLogger())
		if _, ok := uc.serverAmountForRecord(context.Background(), recordID); ok {
			t.Error("a zero amount must not be treated as authoritative")
		}
	})

	t.Run("lookup error", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), &scriptedRecordRepo{err: errors.New("db down")}, nil, &PaymentConfig{}, nil, nil, testLogger())
		if _, ok := uc.serverAmountForRecord(context.Background(), recordID); ok {
			t.Error("a lookup failure must not yield an amount")
		}
	})

	t.Run("record with amount", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), &scriptedRecordRepo{record: &ParkingRecordInfo{ID: recordID.String(), FinalAmount: 1500}}, nil, &PaymentConfig{}, nil, nil, testLogger())
		amount, ok := uc.serverAmountForRecord(context.Background(), recordID)
		if !ok || amount != 1500 {
			t.Errorf("amount = %d, ok = %v; want 1500, true", amount, ok)
		}
	})
}

// TestGenerateWechatPaymentURLs drives both WeChat payment flavours against a
// redirected gateway.
func TestGenerateWechatPaymentURLs(t *testing.T) {
	repo := NewMockOrderRepo()
	recordID := uuid.New()

	newUC := func(t *testing.T, srv *httptest.Server) *PaymentUseCase {
		uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(),
			&PaymentConfig{}, wechatGatewayClient(t, srv), nil, testLogger())
		return uc
	}

	t.Run("native pay", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code_url": "weixin://wxpay/bizpayurl?pr=abc"})
		}))
		defer srv.Close()

		data, err := newUC(t, srv).CreatePayment(context.Background(), &v1.CreatePaymentRequest{
			RecordId: recordID.String(), Amount: 1500, PayMethod: "wechat",
		})
		if err != nil {
			t.Fatalf("CreatePayment failed: %v", err)
		}
		if data.PayUrl != "weixin://wxpay/bizpayurl?pr=abc" {
			t.Errorf("pay url = %q, want the code_url", data.PayUrl)
		}
		if data.QrCode == "" {
			t.Error("the QR code must be populated for a native payment")
		}
	})

	t.Run("jsapi pay", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"prepay_id": "wx1234567890"})
		}))
		defer srv.Close()

		data, err := newUC(t, srv).CreatePayment(context.Background(), &v1.CreatePaymentRequest{
			RecordId: recordID.String(), Amount: 1500, PayMethod: "wechat", OpenId: "openid-1",
		})
		if err != nil {
			t.Fatalf("CreatePayment failed: %v", err)
		}
		if data.PayUrl == "" || data.QrCode == "" {
			t.Errorf("jsapi payment must return pay parameters, got url=%q qr=%q", data.PayUrl, data.QrCode)
		}
	})
}

// TestGetPaymentStatus covers the status endpoint including the unknown-order
// guard added after a nil-deref was found while testing.
func TestGetPaymentStatus(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	payTime := time.Now()
	repo.Orders[orderID] = &Order{
		ID: orderID, Status: string(StatusPaid), PayMethod: string(MethodWechat), PayTime: &payTime,
	}

	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())

	data, err := uc.GetPaymentStatus(context.Background(), orderID.String())
	if err != nil {
		t.Fatalf("GetPaymentStatus failed: %v", err)
	}
	if data.Status != string(StatusPaid) || data.PayMethod != string(MethodWechat) {
		t.Errorf("status = %+v", data)
	}
	if data.PayTime != payTime.Format(time.RFC3339) {
		t.Errorf("pay time = %q, want the RFC3339 form", data.PayTime)
	}

	if _, err := uc.GetPaymentStatus(context.Background(), "not-a-uuid"); err == nil {
		t.Error("an invalid order id must be rejected")
	}
	// Unknown order: must error, not panic.
	if _, err := uc.GetPaymentStatus(context.Background(), uuid.New().String()); err == nil {
		t.Error("an unknown order must return an error")
	}
}

// TestRefund covers the refund endpoint's branches.
func TestRefund(t *testing.T) {
	t.Run("unpaid order fails without touching the gateway", func(t *testing.T) {
		repo := NewMockOrderRepo()
		orderID := uuid.New()
		repo.Orders[orderID] = &Order{
			ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodWechat),
		}
		uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())

		data, err := uc.Refund(context.Background(), orderID.String(), "test")
		if err != nil {
			t.Fatalf("Refund returned error: %v", err)
		}
		if data.Status != "failed" {
			t.Errorf("status = %q, want failed", data.Status)
		}
	})

	t.Run("unknown order fails", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())
		data, err := uc.Refund(context.Background(), uuid.New().String(), "test")
		if err != nil {
			t.Fatalf("Refund returned error: %v", err)
		}
		if data.Status != "failed" {
			t.Errorf("status = %q, want failed", data.Status)
		}
	})

	t.Run("invalid order id", func(t *testing.T) {
		uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())
		if _, err := uc.Refund(context.Background(), "not-a-uuid", "test"); err == nil {
			t.Error("an invalid order id must be rejected")
		}
	})

	t.Run("paid order without a gateway fails closed", func(t *testing.T) {
		repo := NewMockOrderRepo()
		orderID := uuid.New()
		repo.Orders[orderID] = &Order{
			ID: orderID, FinalAmount: 1500, Status: string(StatusPaid), PayMethod: string(MethodWechat),
		}
		uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())

		data, err := uc.Refund(context.Background(), orderID.String(), "test")
		if err != nil {
			t.Fatalf("Refund returned error: %v", err)
		}
		if data.Status != "failed" {
			t.Errorf("status = %q, want failed", data.Status)
		}
		if repo.Orders[orderID].Status != string(StatusPaid) {
			t.Error("the order must stay paid when the gateway refund fails")
		}
	})
}

// TestRefundOrderValidation pins the guards that stop nonsensical refunds.
func TestRefundOrderValidation(t *testing.T) {
	uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, testLogger())
	ctx := context.Background()

	if _, err := uc.RefundOrder(ctx, nil, "r-1", 100); err == nil {
		t.Error("a nil order must be rejected")
	}
	if _, err := uc.RefundOrder(ctx, &Order{PayMethod: "wechat"}, "r-1", 0); err == nil {
		t.Error("a non-positive amount must be rejected")
	}
	if _, err := uc.RefundOrder(ctx, &Order{PayMethod: "wechat", FinalAmount: 100}, "r-1", 200); err == nil {
		t.Error("an over-order refund must be rejected")
	}
	if _, err := uc.RefundOrder(ctx, &Order{PayMethod: "wechat", FinalAmount: 100}, "r-1", 50); err == nil {
		t.Error("a missing wechat client must fail loudly")
	}
	if _, err := uc.RefundOrder(ctx, &Order{PayMethod: "alipay", FinalAmount: 100}, "r-1", 50); err == nil {
		t.Error("a missing alipay client must fail loudly")
	}
	if _, err := uc.RefundOrder(ctx, &Order{PayMethod: "cash", FinalAmount: 100}, "r-1", 50); err == nil {
		t.Error("an unknown payment method must be rejected")
	}
}

// TestRefundAgainstGateway proves a channel-confirmed refund persists the
// terminal state, and a rejected one does not.
func TestRefundAgainstGateway(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPaid), PayMethod: string(MethodWechat),
	}

	t.Run("confirmed", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "SUCCESS"})
		}))
		defer srv.Close()

		uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(),
			&PaymentConfig{}, wechatGatewayClient(t, srv), nil, testLogger())

		data, err := uc.Refund(context.Background(), orderID.String(), "customer request")
		if err != nil {
			t.Fatalf("Refund failed: %v", err)
		}
		if data.Status != "success" {
			t.Fatalf("status = %q, want success", data.Status)
		}
		if repo.Orders[orderID].Status != string(StatusRefunded) {
			t.Errorf("order status = %s, want refunded", repo.Orders[orderID].Status)
		}
		if repo.Orders[orderID].RefundedAt == nil {
			t.Error("RefundedAt must be set")
		}
	})

	t.Run("rejected", func(t *testing.T) {
		repo.Orders[orderID].Status = string(StatusPaid)
		repo.Orders[orderID].RefundedAt = nil
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": "PARAM_ERROR", "message": "invalid"})
		}))
		defer srv.Close()

		uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(),
			&PaymentConfig{}, wechatGatewayClient(t, srv), nil, testLogger())

		data, err := uc.Refund(context.Background(), orderID.String(), "customer request")
		if err != nil {
			t.Fatalf("Refund returned error: %v", err)
		}
		if data.Status != "failed" {
			t.Errorf("status = %q, want failed", data.Status)
		}
		if repo.Orders[orderID].Status != string(StatusPaid) {
			t.Error("the order must stay paid when the gateway rejects the refund")
		}
	})
}

// TestFormatTime covers the nil-pointer formatting used by the status response.
func TestFormatTime(t *testing.T) {
	if got := formatTime(nil); got != "" {
		t.Errorf("formatTime(nil) = %q, want empty", got)
	}
	now := time.Now()
	if got := formatTime(&now); got != now.Format(time.RFC3339) {
		t.Errorf("formatTime(now) = %q, want RFC3339", got)
	}
}

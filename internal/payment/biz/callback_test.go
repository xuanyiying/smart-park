package biz

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

// newTestUseCase builds a PaymentUseCase wired to the in-memory order repository.
func newTestUseCase(repo OrderRepo, config *PaymentConfig) *PaymentUseCase {
	return NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), config, nil, nil, log.NewStdLogger(os.Stdout))
}

func TestParseYuanToCents(t *testing.T) {
	tests := []struct {
		input   string
		want    int64
		wantErr bool
	}{
		{input: "10.00", want: 1000},
		{input: "0.01", want: 1},
		{input: "0", want: 0},
		{input: "123.45", want: 12345},
		{input: " 12.30 ", want: 1230},
		{input: "-5.50", want: -550},
		{input: "1.5", want: 150},
		{input: "", wantErr: true},
		{input: "abc", wantErr: true},
		{input: "1.234", wantErr: true}, // sub-cent precision must be rejected, not rounded
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := parseYuanToCents(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseYuanToCents(%q) expected error, got %d", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseYuanToCents(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseYuanToCents(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestParseYuanToCentsIsExact guards the reason the helper exists: binary floating point
// cannot represent decimal fractions such as 0.1 or 19.99 exactly, so money must never be
// parsed through float64.
func TestParseYuanToCentsIsExact(t *testing.T) {
	tests := map[string]int64{
		"19.99":   1999,
		"0.10":    10,
		"1.15":    115,
		"2.70":    270,
		"1234.55": 123455,
		"0.29":    29,
		"8.07":    807,
	}

	for amount, want := range tests {
		got, err := parseYuanToCents(amount)
		if err != nil {
			t.Fatalf("parseYuanToCents(%q) unexpected error: %v", amount, err)
		}
		if got != want {
			t.Errorf("parseYuanToCents(%q) = %d, want %d", amount, got, want)
		}
	}
}

func TestSettleOrderRejectsAmountMismatch(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID:          orderID,
		FinalAmount: 20.00,
		Status:      string(StatusPending),
	}

	uc := newTestUseCase(repo, &PaymentConfig{})

	// Gateway reports 100 cents while the order is worth 2000 cents.
	if _, err := uc.settleOrder(context.Background(), orderID, MethodWechat, "tx-1", 100); err == nil {
		t.Fatal("expected amount mismatch to be rejected")
	}

	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Errorf("order must stay pending after a rejected settlement, got %s", repo.Orders[orderID].Status)
	}
}

func TestSettleOrderIsIdempotentAcrossDuplicateCallbacks(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID:          orderID,
		FinalAmount: 1500, // 15.00 元
		Status:      string(StatusPending),
	}

	uc := newTestUseCase(repo, &PaymentConfig{})

	first, err := uc.settleOrder(context.Background(), orderID, MethodWechat, "tx-1", 1500)
	if err != nil {
		t.Fatalf("first settlement failed: %v", err)
	}
	if first.Duplicate {
		t.Error("first settlement must not be reported as duplicate")
	}
	if repo.Orders[orderID].Status != string(StatusPaid) {
		t.Fatalf("expected order to be paid, got %s", repo.Orders[orderID].Status)
	}

	// A retried callback (or a second replica receiving the same notification) must be
	// acknowledged as a duplicate instead of settling the order again.
	second, err := uc.settleOrder(context.Background(), orderID, MethodWechat, "tx-1", 1500)
	if err != nil {
		t.Fatalf("duplicate settlement should not error: %v", err)
	}
	if !second.Duplicate {
		t.Error("second settlement must be reported as duplicate")
	}
	if repo.Orders[orderID].PaidAmount != 1500 {
		t.Errorf("duplicate callback changed paid amount: %d", repo.Orders[orderID].PaidAmount)
	}
}

func TestSettleOrderRejectsUnknownOrder(t *testing.T) {
	uc := newTestUseCase(NewMockOrderRepo(), &PaymentConfig{})

	if _, err := uc.settleOrder(context.Background(), uuid.New(), MethodWechat, "tx-1", 100); err == nil {
		t.Fatal("expected settlement of an unknown order to fail")
	}
}

func TestSettleOrderRequiresTransactionID(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{ID: orderID, FinalAmount: 10, Status: string(StatusPending)}

	uc := newTestUseCase(repo, &PaymentConfig{})

	if _, err := uc.settleOrder(context.Background(), orderID, MethodWechat, "", 1000); err == nil {
		t.Fatal("expected settlement without a transaction id to fail")
	}
}

// TestUnsignedCallbacksAreRejected documents the security property: a callback that
// arrives as a decoded protobuf message carries no signature, so it must never settle
// an order.
func TestUnsignedCallbacksAreRejected(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID:          orderID,
		FinalAmount: 10.00,
		Status:      string(StatusPending),
	}

	uc := newTestUseCase(repo, &PaymentConfig{})

	wechatResp, err := uc.HandleWechatCallback(context.Background(), &v1.WechatCallbackRequest{
		OutTradeNo:    orderID.String(),
		TransactionId: "attacker-tx",
		TotalFee:      "1000",
		ReturnCode:    "SUCCESS",
	})
	if err != nil {
		t.Fatalf("HandleWechatCallback returned error: %v", err)
	}
	if wechatResp.ReturnCode != string(WechatStatusFail) {
		t.Errorf("unsigned wechat callback must be rejected, got %q", wechatResp.ReturnCode)
	}

	alipayResp, err := uc.HandleAlipayCallback(context.Background(), &v1.AlipayCallbackRequest{
		OutTradeNo:  orderID.String(),
		TradeNo:     "attacker-tx",
		TotalAmount: "10.00",
		TradeStatus: "TRADE_SUCCESS",
	})
	if err != nil {
		t.Fatalf("HandleAlipayCallback returned error: %v", err)
	}
	if alipayResp.Code == "success" {
		t.Error("unsigned alipay callback must be rejected")
	}

	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Errorf("order must remain pending after unsigned callbacks, got %s", repo.Orders[orderID].Status)
	}
}

func TestIsAlipaySuccessStatus(t *testing.T) {
	tests := map[string]bool{
		"TRADE_SUCCESS":  true,
		"TRADE_FINISHED": true,
		"TRADE_CLOSED":   false,
		"WAIT_BUYER_PAY": false,
		"":               false,
	}

	for status, want := range tests {
		if got := isAlipaySuccessStatus(status); got != want {
			t.Errorf("isAlipaySuccessStatus(%s) = %v, want %v", status, got, want)
		}
	}
}

func TestParseAmount(t *testing.T) {
	tests := map[string]float64{
		"1000":  10.00,
		"0":     0.00,
		"100":   1.00,
		"12345": 123.45,
	}

	for input, want := range tests {
		if got := parseAmount(input); got != want {
			t.Errorf("parseAmount(%s) = %.2f, want %.2f", input, got, want)
		}
	}
}

func TestParseAmountFloat(t *testing.T) {
	tests := map[string]float64{
		"10.00":  10.00,
		"0.00":   0.00,
		"123.45": 123.45,
		"999.99": 999.99,
	}

	for input, want := range tests {
		if got := parseAmountFloat(input); got != want {
			t.Errorf("parseAmountFloat(%s) = %.2f, want %.2f", input, got, want)
		}
	}
}

func TestFixMismatchedOrdersRefusesWithoutRefunder(t *testing.T) {
	uc := NewReconciliationUseCase(NewMockOrderRepo(), nil, nil, nil, nil, log.NewStdLogger(os.Stdout))

	// Without a real refunder the fixer must fail loudly rather than marking orders as
	// refunded, which is what silently corrupts the ledger.
	if err := uc.FixMismatchedOrders(context.Background(), []string{uuid.New().String()}); err == nil {
		t.Fatal("expected FixMismatchedOrders to refuse to run without a refunder")
	}
}

// stubRefunder records what reconciliation asked the gateway to refund.
type stubRefunder struct {
	calls       []int64
	err         error
	transaction string
}

func (s *stubRefunder) RefundOrder(ctx context.Context, order *Order, refundID string, amountCents int64) (string, error) {
	s.calls = append(s.calls, amountCents)
	if s.err != nil {
		return "", s.err
	}
	if s.transaction == "" {
		return refundID, nil
	}
	return s.transaction, nil
}

// stubReconciliationRepo keeps reconciliation records in memory for tests.
type stubReconciliationRepo struct {
	records map[uuid.UUID]*ReconciliationRecord
}

func newStubReconciliationRepo(records ...*ReconciliationRecord) *stubReconciliationRepo {
	repo := &stubReconciliationRepo{records: make(map[uuid.UUID]*ReconciliationRecord)}
	for _, r := range records {
		repo.records[r.OrderID] = r
	}
	return repo
}

func (s *stubReconciliationRepo) CreateReconciliation(ctx context.Context, record *ReconciliationRecord) error {
	s.records[record.OrderID] = record
	return nil
}

func (s *stubReconciliationRepo) GetReconciliationByOrderID(ctx context.Context, orderID uuid.UUID) (*ReconciliationRecord, error) {
	return s.records[orderID], nil
}

func (s *stubReconciliationRepo) GetReconciliationByTimeRange(ctx context.Context, start, end time.Time) ([]*ReconciliationRecord, error) {
	return nil, nil
}

func (s *stubReconciliationRepo) UpdateReconciliation(ctx context.Context, record *ReconciliationRecord) error {
	s.records[record.OrderID] = record
	return nil
}

func (s *stubReconciliationRepo) GetMismatchedReconciliations(ctx context.Context, start, end time.Time) ([]*ReconciliationRecord, error) {
	return nil, nil
}

func TestFixMismatchedOrdersOnlySettlesAfterGatewayConfirms(t *testing.T) {
	orderID := uuid.New()
	repo := NewMockOrderRepo()
	// The order was worth 25.00 but the customer was charged 30.00, so 5.00 must be
	// refunded. Note this is the reverse of the (incorrect) original direction.
	repo.Orders[orderID] = &Order{
		ID:          orderID,
		FinalAmount: 2500, // 25.00 元
		PaidAmount:  3000, // 30.00 元
		Status:      string(StatusPaid),
		PayMethod:   string(MethodWechat),
	}

	reconRepo := newStubReconciliationRepo(&ReconciliationRecord{OrderID: orderID})

	refunder := &stubRefunder{err: context.DeadlineExceeded}
	uc := NewReconciliationUseCase(repo, reconRepo, nil, nil, refunder, log.NewStdLogger(os.Stdout))

	// The gateway refund fails: the order must be left untouched.
	_ = uc.FixMismatchedOrders(context.Background(), []string{orderID.String()})

	if repo.Orders[orderID].Status != string(StatusPaid) {
		t.Errorf("order must stay paid when the refund failed, got %s", repo.Orders[orderID].Status)
	}
	if repo.Orders[orderID].RefundTransactionID != "" {
		t.Errorf("no refund transaction id may be recorded, got %q", repo.Orders[orderID].RefundTransactionID)
	}
	if len(refunder.calls) != 1 {
		t.Fatalf("expected exactly one refund attempt, got %d", len(refunder.calls))
	}
	if refunder.calls[0] != 500 {
		t.Errorf("expected the 5.00 overcharge (500 cents) to be refunded, got %d cents", refunder.calls[0])
	}
}

// TestFixMismatchedOrdersDoesNotSilentlyRewriteShortfalls locks in the rule that a
// shortfall is never "fixed" by lowering the order amount: that hides the missing money
// instead of recovering it.
func TestFixMismatchedOrdersDoesNotSilentlyRewriteShortfalls(t *testing.T) {
	orderID := uuid.New()
	repo := NewMockOrderRepo()
	repo.Orders[orderID] = &Order{
		ID:          orderID,
		FinalAmount: 3000, // should have been charged 30.00
		PaidAmount:  2500, // only 25.00 was collected
		Status:      string(StatusPaid),
		PayMethod:   string(MethodWechat),
	}

	reconRepo := newStubReconciliationRepo(&ReconciliationRecord{OrderID: orderID})
	refunder := &stubRefunder{}

	uc := NewReconciliationUseCase(repo, reconRepo, nil, nil, refunder, log.NewStdLogger(os.Stdout))

	if err := uc.FixMismatchedOrders(context.Background(), []string{orderID.String()}); err != nil {
		t.Fatalf("FixMismatchedOrders returned error: %v", err)
	}

	if len(refunder.calls) != 0 {
		t.Errorf("a shortfall must not trigger a refund, got %d attempts", len(refunder.calls))
	}
	if repo.Orders[orderID].FinalAmount != 3000 {
		t.Errorf("order amount must not be rewritten to hide the shortfall, got %d", repo.Orders[orderID].FinalAmount)
	}
	if reconRepo.records[orderID].Status != ReconciliationStatusMismatch {
		t.Errorf("shortfall must remain flagged as mismatch, got %s", reconRepo.records[orderID].Status)
	}
}

func TestReconcileOrderReportsMismatchWhenGatewayDisagrees(t *testing.T) {
	order := &Order{
		ID:          uuid.New(),
		FinalAmount: 12.00,
		PaidAmount:  12.00,
		Status:      string(StatusPaid),
		PayMethod:   string(MethodWechat),
	}

	uc := NewReconciliationUseCase(NewMockOrderRepo(), nil, nil, nil, nil, log.NewStdLogger(os.Stdout))

	// No WeChat client configured: reconciliation must not claim success.
	record, err := uc.reconcileOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("reconcileOrder returned error: %v", err)
	}
	if record.Status == ReconciliationStatusMatched {
		t.Errorf("reconciliation without a gateway client must not report matched, got %s (%s)", record.Status, record.Notes)
	}
}

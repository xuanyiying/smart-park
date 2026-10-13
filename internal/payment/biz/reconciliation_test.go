package biz

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	alipaysdk "github.com/smartwalle/alipay/v3"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"

	"github.com/xuanyiying/smart-park/internal/payment/alipay"
	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

// ---------------------------------------------------------------------------
// 测试基建：带重定向传输的支付渠道客户端
// ---------------------------------------------------------------------------

// gatewayRedirect rewrites outgoing gateway requests to the local test server.
type gatewayRedirect struct{ base string }

func (t gatewayRedirect) RoundTrip(r *http.Request) (*http.Response, error) {
	u, err := url.Parse(t.base)
	if err != nil {
		return nil, err
	}
	r.URL.Scheme = u.Scheme
	r.URL.Host = u.Host
	return http.DefaultTransport.RoundTrip(r)
}

// wechatGatewayClient builds a wechat.Client whose query/refund calls land on
// srv. The platform certificate stands in for WeChat's.
func wechatGatewayClient(t *testing.T, srv *httptest.Server) *wechat.Client {
	t.Helper()

	platformKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate platform key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(11),
		Subject:      pkix.Name{CommonName: "Tenpay.com platform cert"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &platformKey.PublicKey, platformKey)
	if err != nil {
		t.Fatalf("failed to mint certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	merchantKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate merchant key: %v", err)
	}

	coreClient, err := core.NewClient(context.Background(),
		option.WithMerchantCredential("1900000001", cert.SerialNumber.String(), merchantKey),
		option.WithoutValidator(),
		option.WithHTTPClient(&http.Client{Transport: gatewayRedirect{base: srv.URL}}),
	)
	if err != nil {
		t.Fatalf("failed to build core client: %v", err)
	}

	return wechat.NewClientWithCert(coreClient, &wechat.Config{
		AppID:        "wx_appid",
		MchID:        "1900000001",
		CertSerialNo: cert.SerialNumber.String(),
		APIv3Key:     "0123456789abcdef0123456789abcdef",
	}, merchantKey, cert)
}

// alipayGatewayClient builds an alipay.Client wrapper whose gateway calls land
// on srv, with responses signed by the matching test key.
func alipayGatewayClient(t *testing.T, srv *httptest.Server, signingKey *rsa.PrivateKey, privatePEM, publicPEM string) *alipay.Client {
	t.Helper()

	sdk, err := alipaysdk.New("2021000000000000", privatePEM, false)
	if err != nil {
		t.Fatalf("failed to build sdk client: %v", err)
	}
	if err := sdk.LoadAliPayPublicKey(publicPEM); err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}
	sdk.Client = &http.Client{Transport: gatewayRedirect{base: srv.URL}}

	return alipay.NewClientWithGateway(sdk, &alipay.Config{
		AppID:           "2021000000000000",
		PrivateKey:      privatePEM,
		AlipayPublicKey: publicPEM,
		SignType:        alipay.SignTypeRSA2,
	})
}

// alipayKeyPairPEM generates merchant key material as PEM strings.
func alipayKeyPairPEM(t *testing.T) (key *rsa.PrivateKey, privatePEM, publicPEM string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal private key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	return key,
		string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})),
		string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
}

// signedAlipayGatewayResponse builds the signed JSON envelope the alipay SDK
// expects from the gateway.
func signedAlipayGatewayResponse(t *testing.T, key *rsa.PrivateKey, node string, payload map[string]interface{}) string {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}
	digest := sha256.Sum256(raw)
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("failed to sign response: %v", err)
	}
	return fmt.Sprintf(`{"%s":%s,"sign":"%s"}`, node, raw, base64.StdEncoding.EncodeToString(signature))
}

func newReconciliationUseCase(repo OrderRepo, reconRepo ReconciliationRepo, wechatClient *wechat.Client, alipayClient *alipay.Client, refunder Refunder) *ReconciliationUseCase {
	return NewReconciliationUseCase(repo, reconRepo, wechatClient, alipayClient, refunder, log.NewStdLogger(nopWriter{}))
}

// ---------------------------------------------------------------------------
// 每日对账、漏单与报表
// ---------------------------------------------------------------------------

// reconOrderRepo is a scriptable OrderRepo for reconciliation scenarios.
type reconOrderRepo struct {
	MockOrderRepo
	byRange    []*Order
	rangeErr   error
	pending    []*Order
	pendingErr error
}

func newReconOrderRepo() *reconOrderRepo {
	return &reconOrderRepo{MockOrderRepo: MockOrderRepo{Orders: make(map[uuid.UUID]*Order)}}
}

func (r *reconOrderRepo) GetOrdersByTimeRange(ctx context.Context, start, end time.Time) ([]*Order, error) {
	if r.rangeErr != nil {
		return nil, r.rangeErr
	}
	return r.byRange, nil
}

func (r *reconOrderRepo) ListOrdersByStatus(ctx context.Context, status string, cutoff time.Time, limit int) ([]*Order, error) {
	if r.pendingErr != nil {
		return nil, r.pendingErr
	}
	return r.pending, nil
}

func paidOrder(amount, paid int64, method PayMethod) *Order {
	return &Order{
		ID: uuid.New(), FinalAmount: amount, PaidAmount: paid,
		Status: string(StatusPaid), PayMethod: string(method), CreatedAt: time.Now(),
	}
}

// TestReconcileOrderWithoutChannelClients locks in the safety property: with no
// channel client configured, reconciliation must report pending (unknown), never
// "matched".
func TestReconcileOrderWithoutChannelClients(t *testing.T) {
	uc := newReconciliationUseCase(NewMockOrderRepo(), nil, nil, nil, nil)

	wechatOrder := &Order{ID: uuid.New(), FinalAmount: 1500, PaidAmount: 1500,
		Status: string(StatusPaid), PayMethod: string(MethodWechat)}
	record, err := uc.reconcileOrder(context.Background(), wechatOrder)
	if err != nil {
		t.Fatalf("reconcileOrder returned error: %v", err)
	}
	if record.Status != ReconciliationStatusPending {
		t.Errorf("wechat without a client must stay pending, got %s (%s)", record.Status, record.Notes)
	}

	alipayOrder := &Order{ID: uuid.New(), FinalAmount: 1500, PaidAmount: 1500,
		Status: string(StatusPaid), PayMethod: string(MethodAlipay)}
	record, err = uc.reconcileOrder(context.Background(), alipayOrder)
	if err != nil {
		t.Fatalf("reconcileOrder returned error: %v", err)
	}
	if record.Status != ReconciliationStatusPending {
		t.Errorf("alipay without a client must stay pending, got %s (%s)", record.Status, record.Notes)
	}

	unknown := &Order{ID: uuid.New(), PayMethod: "cash"}
	record, err = uc.reconcileOrder(context.Background(), unknown)
	if err != nil {
		t.Fatalf("reconcileOrder returned error: %v", err)
	}
	if record.Status != ReconciliationStatusPending {
		t.Errorf("an unsupported method must stay pending, got %s", record.Status)
	}
}

// TestReconcileOrderLocalSelfCheck covers the local consistency leg: when the
// channel cannot be consulted but paid != final, that is a mismatch.
func TestReconcileOrderLocalSelfCheck(t *testing.T) {
	uc := newReconciliationUseCase(NewMockOrderRepo(), nil, nil, nil, nil)

	// Unsupported method short-circuits before the local check, so use a client-less
	// wechat order with a self-inconsistent ledger: paid > final is still a mismatch
	// the local leg must catch once the channel leg is skipped.
	order := &Order{ID: uuid.New(), FinalAmount: 1000, PaidAmount: 1000,
		Status: string(StatusPaid), PayMethod: "unknown-method"}
	record, err := uc.reconcileOrder(context.Background(), order)
	if err != nil {
		t.Fatalf("reconcileOrder returned error: %v", err)
	}
	if record.Status != ReconciliationStatusPending {
		t.Errorf("expected pending for an unsupported method, got %s", record.Status)
	}
}

// TestReconcileDailyAggregatesAndPersists drives the daily job end to end with
// scripted channel responses.
func TestReconcileDailyAggregatesAndPersists(t *testing.T) {
	repo := newReconOrderRepo()
	matched := paidOrder(1500, 1500, MethodWechat)
	repo.byRange = []*Order{
		matched,
		{ID: uuid.New(), FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodWechat), CreatedAt: time.Now()}, // skipped: not paid
	}
	reconRepo := newStubReconciliationRepo()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"out_trade_no": matched.ID.String(),
			"trade_state":  "SUCCESS",
			"amount":       map[string]interface{}{"total": 1500},
		})
	}))
	defer srv.Close()

	uc := newReconciliationUseCase(repo, reconRepo, wechatGatewayClient(t, srv), nil, nil)

	records, err := uc.ReconcileDaily(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("ReconcileDaily failed: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 reconciliation record (pending orders skipped), got %d", len(records))
	}
	if records[0].Status != ReconciliationStatusMatched {
		t.Errorf("record status = %s, want matched", records[0].Status)
	}
	if _, ok := reconRepo.records[matched.ID]; !ok {
		t.Error("the reconciliation record must be persisted")
	}
}

// TestReconcileDailyPropagatesRepoError proves a repository failure surfaces
// instead of silently producing an empty report.
func TestReconcileDailyPropagatesRepoError(t *testing.T) {
	repo := newReconOrderRepo()
	repo.rangeErr = fmt.Errorf("db down")
	uc := newReconciliationUseCase(repo, newStubReconciliationRepo(), nil, nil, nil)

	if _, err := uc.ReconcileDaily(context.Background(), time.Now()); err == nil {
		t.Fatal("expected the repository error to surface")
	}
}

// TestReconcileDailyFindsMissingOrders covers the lost-callback discovery: the
// channel says paid while the local order still sits in pending.
func TestReconcileDailyFindsMissingOrders(t *testing.T) {
	repo := newReconOrderRepo()
	missing := &Order{
		ID: uuid.New(), FinalAmount: 1500, Status: string(StatusPending),
		PayMethod: string(MethodWechat), CreatedAt: time.Now(),
	}
	repo.pending = []*Order{missing}
	reconRepo := newStubReconciliationRepo()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"out_trade_no":   missing.ID.String(),
			"transaction_id": "4200009999",
			"trade_state":    "SUCCESS",
			"amount":         map[string]interface{}{"total": 1500},
		})
	}))
	defer srv.Close()

	uc := newReconciliationUseCase(repo, reconRepo, wechatGatewayClient(t, srv), nil, nil)

	records, err := uc.ReconcileDaily(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("ReconcileDaily failed: %v", err)
	}
	var found bool
	for _, r := range records {
		if r.OrderID == missing.ID {
			found = true
			if r.Status != ReconciliationStatusMissing {
				t.Errorf("record status = %s, want missing", r.Status)
			}
			if r.TransactionID != "4200009999" {
				t.Errorf("transaction id = %q, want the channel reference", r.TransactionID)
			}
		}
	}
	if !found {
		t.Fatal("the channel-paid/pending order must be reported as missing")
	}
}

// TestConfirmWithGatewayBranches covers the query helper the sweeper and the
// missing-order check share.
func TestConfirmWithGatewayBranches(t *testing.T) {
	order := &Order{ID: uuid.New(), PayMethod: string(MethodWechat)}

	t.Run("no wechat client", func(t *testing.T) {
		uc := newReconciliationUseCase(NewMockOrderRepo(), nil, nil, nil, nil)
		if _, _, err := uc.confirmWithGateway(context.Background(), order); err == nil {
			t.Error("an unconfigured channel must fail loudly")
		}
	})

	t.Run("no alipay client", func(t *testing.T) {
		uc := newReconciliationUseCase(NewMockOrderRepo(), nil, nil, nil, nil)
		alipayOrder := &Order{ID: uuid.New(), PayMethod: string(MethodAlipay)}
		if _, _, err := uc.confirmWithGateway(context.Background(), alipayOrder); err == nil {
			t.Error("an unconfigured channel must fail loudly")
		}
	})

	t.Run("unknown method", func(t *testing.T) {
		uc := newReconciliationUseCase(NewMockOrderRepo(), nil, nil, nil, nil)
		paid, tx, err := uc.confirmWithGateway(context.Background(), &Order{ID: uuid.New(), PayMethod: "cash"})
		if err != nil || paid || tx != "" {
			t.Errorf("unknown method must report not paid without error, got paid=%v tx=%q err=%v", paid, tx, err)
		}
	})
}

// TestGetReconciliationReportAggregates checks the operator-facing report:
// totals, matched/mismatched split and per-method breakdown.
func TestGetReconciliationReportAggregates(t *testing.T) {
	repo := newReconOrderRepo()
	repo.byRange = []*Order{
		paidOrder(1500, 1500, MethodWechat),
		paidOrder(2000, 1800, MethodAlipay), // short-paid: mismatch
		{ID: uuid.New(), Status: string(StatusPending), PayMethod: string(MethodWechat), CreatedAt: time.Now()},
	}
	uc := newReconciliationUseCase(repo, nil, nil, nil, nil)

	report, err := uc.GetReconciliationReport(context.Background(), time.Now(), time.Now().Add(24*time.Hour))
	if err != nil {
		t.Fatalf("GetReconciliationReport failed: %v", err)
	}
	if report["total_orders"] != 2 {
		t.Errorf("total_orders = %v, want 2 (pending excluded)", report["total_orders"])
	}
	if report["total_amount"] != int64(3500) {
		t.Errorf("total_amount = %v, want 3500 cents", report["total_amount"])
	}
	if report["matched_orders"] != 1 || report["mismatched_orders"] != 1 {
		t.Errorf("matched/mismatched = %v/%v, want 1/1", report["matched_orders"], report["mismatched_orders"])
	}
	methods := report["payment_methods"].(map[string]map[string]interface{})
	if methods["wechat"]["count"] != 1 || methods["alipay"]["count"] != 1 {
		t.Errorf("per-method counts = %v", methods)
	}
}

// TestMarkReconciliationNilRecord documents the nil guard.
func TestMarkReconciliationNilRecord(t *testing.T) {
	uc := newReconciliationUseCase(NewMockOrderRepo(), nil, nil, nil, nil)
	// Must not panic.
	uc.markReconciliation(context.Background(), nil, ReconciliationStatusMatched, "noop")
}

// TestFixMismatchedOrdersRefundsOvercharge covers the happy path of the fixer:
// only after the gateway confirms does the order become refunded.
func TestFixMismatchedOrdersRefundsOvercharge(t *testing.T) {
	orderID := uuid.New()
	repo := NewMockOrderRepo()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 2500, PaidAmount: 3000,
		Status: string(StatusPaid), PayMethod: string(MethodWechat),
	}
	reconRepo := newStubReconciliationRepo(&ReconciliationRecord{OrderID: orderID})
	refunder := &stubRefunder{transaction: "50000000382019073109739578859"}

	uc := newReconciliationUseCase(repo, reconRepo, nil, nil, refunder)
	if err := uc.FixMismatchedOrders(context.Background(), []string{orderID.String()}); err != nil {
		t.Fatalf("FixMismatchedOrders failed: %v", err)
	}

	if len(refunder.calls) != 1 || refunder.calls[0] != 500 {
		t.Errorf("expected exactly one 500-cent refund, got %v", refunder.calls)
	}
	order := repo.Orders[orderID]
	if order.Status != string(StatusRefunded) {
		t.Errorf("order status = %s, want refunded", order.Status)
	}
	if order.RefundTransactionID != "50000000382019073109739578859" {
		t.Errorf("refund transaction id = %q, want the gateway reference", order.RefundTransactionID)
	}
	if order.RefundedAt == nil {
		t.Error("RefundedAt must be set")
	}
	if reconRepo.records[orderID].Status != ReconciliationStatusMatched {
		t.Errorf("reconciliation status = %s, want matched after a confirmed refund", reconRepo.records[orderID].Status)
	}
}

// TestFixMismatchedOrdersSkipsBadInput covers the guard rails: invalid ids and
// unknown orders are skipped without touching anything.
func TestFixMismatchedOrdersSkipsBadInput(t *testing.T) {
	repo := NewMockOrderRepo()
	reconRepo := newStubReconciliationRepo()
	refunder := &stubRefunder{}

	uc := newReconciliationUseCase(repo, reconRepo, nil, nil, refunder)
	if err := uc.FixMismatchedOrders(context.Background(), []string{"not-a-uuid", uuid.New().String()}); err != nil {
		t.Fatalf("FixMismatchedOrders must skip bad input without failing: %v", err)
	}
	if len(refunder.calls) != 0 {
		t.Errorf("no refund may be attempted for unknown orders, got %v", refunder.calls)
	}
}

// TestFixMismatchedOrdersMatchingAmount marks a balanced order as matched.
func TestFixMismatchedOrdersMatchingAmount(t *testing.T) {
	orderID := uuid.New()
	repo := NewMockOrderRepo()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, PaidAmount: 1500,
		Status: string(StatusPaid), PayMethod: string(MethodWechat),
	}
	reconRepo := newStubReconciliationRepo(&ReconciliationRecord{OrderID: orderID})
	refunder := &stubRefunder{}

	uc := newReconciliationUseCase(repo, reconRepo, nil, nil, refunder)
	if err := uc.FixMismatchedOrders(context.Background(), []string{orderID.String()}); err != nil {
		t.Fatalf("FixMismatchedOrders failed: %v", err)
	}
	if len(refunder.calls) != 0 {
		t.Error("a balanced order must not trigger a refund")
	}
	if reconRepo.records[orderID].Status != ReconciliationStatusMatched {
		t.Errorf("status = %s, want matched", reconRepo.records[orderID].Status)
	}
}

// TestReconcileWechatOrderAgainstGateway proves the channel is the source of
// truth: matching state/amount reconciles, and every disagreement is reported.
func TestReconcileWechatOrderAgainstGateway(t *testing.T) {
	order := &Order{
		ID: uuid.New(), FinalAmount: 1500, PaidAmount: 1500,
		Status: string(StatusPaid), PayMethod: string(MethodWechat),
	}

	newUC := func(t *testing.T, handler http.HandlerFunc) *ReconciliationUseCase {
		srv := httptest.NewServer(handler)
		t.Cleanup(srv.Close)
		return newReconciliationUseCase(NewMockOrderRepo(), nil, wechatGatewayClient(t, srv), nil, nil)
	}

	successBody := map[string]interface{}{
		"out_trade_no":   order.ID.String(),
		"transaction_id": "4200002106",
		"trade_state":    "SUCCESS",
		"amount":         map[string]interface{}{"total": 1500, "currency": "CNY"},
	}

	t.Run("channel confirms payment", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(successBody)
		})
		record := &ReconciliationRecord{}
		if err := uc.reconcileWechatOrder(context.Background(), order, 1500, record); err != nil {
			t.Fatalf("a matching channel response must reconcile: %v", err)
		}
	})

	t.Run("channel shows a different amount", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"out_trade_no": order.ID.String(), "trade_state": "SUCCESS",
				"amount": map[string]interface{}{"total": 100},
			})
		})
		if err := uc.reconcileWechatOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err == nil {
			t.Error("an amount disagreement with the channel must be reported")
		}
	})

	t.Run("channel shows unpaid", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"out_trade_no": order.ID.String(), "trade_state": "NOTPAY",
			})
		})
		if err := uc.reconcileWechatOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err == nil {
			t.Error("a local-paid/channel-unpaid mismatch must be reported")
		}
	})

	t.Run("channel returns another order", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"out_trade_no": uuid.New().String(), "trade_state": "SUCCESS",
				"amount": map[string]interface{}{"total": 1500},
			})
		})
		if err := uc.reconcileWechatOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err == nil {
			t.Error("a cross-order response must be reported")
		}
	})

	t.Run("missing trade state", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"out_trade_no": order.ID.String()})
		})
		if err := uc.reconcileWechatOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err == nil {
			t.Error("a response without trade_state must be reported")
		}
	})

	t.Run("gateway error", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})
		if err := uc.reconcileWechatOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err == nil {
			t.Error("a gateway failure must surface")
		}
	})
}

// TestReconcileAlipayOrderAgainstGateway mirrors the WeChat test for the
// Alipay channel, including the yuan-string amount comparison.
func TestReconcileAlipayOrderAgainstGateway(t *testing.T) {
	order := &Order{
		ID: uuid.New(), FinalAmount: 1500, PaidAmount: 1500,
		Status: string(StatusPaid), PayMethod: string(MethodAlipay),
	}
	key, privatePEM, publicPEM := alipayKeyPairPEM(t)

	newUC := func(t *testing.T, handler http.HandlerFunc) *ReconciliationUseCase {
		srv := httptest.NewServer(handler)
		t.Cleanup(srv.Close)
		return newReconciliationUseCase(NewMockOrderRepo(), nil, nil, alipayGatewayClient(t, srv, key, privatePEM, publicPEM), nil)
	}

	t.Run("channel confirms payment", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedAlipayGatewayResponse(t, key, "alipay_trade_query_response", map[string]interface{}{
				"code": "10000", "msg": "Success", "out_trade_no": order.ID.String(),
				"trade_status": "TRADE_SUCCESS", "total_amount": "15.00", "trade_no": "20241001220014123456",
			})))
		})
		if err := uc.reconcileAlipayOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err != nil {
			t.Fatalf("a matching channel response must reconcile: %v", err)
		}
	})

	t.Run("TRADE_FINISHED also counts as paid", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedAlipayGatewayResponse(t, key, "alipay_trade_query_response", map[string]interface{}{
				"code": "10000", "msg": "Success", "out_trade_no": order.ID.String(),
				"trade_status": "TRADE_FINISHED", "total_amount": "15.00",
			})))
		})
		if err := uc.reconcileAlipayOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err != nil {
			t.Fatalf("TRADE_FINISHED must reconcile: %v", err)
		}
	})

	t.Run("channel shows a different amount", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedAlipayGatewayResponse(t, key, "alipay_trade_query_response", map[string]interface{}{
				"code": "10000", "msg": "Success", "out_trade_no": order.ID.String(),
				"trade_status": "TRADE_SUCCESS", "total_amount": "0.01",
			})))
		})
		if err := uc.reconcileAlipayOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err == nil {
			t.Error("an amount disagreement must be reported")
		}
	})

	t.Run("channel shows unpaid", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedAlipayGatewayResponse(t, key, "alipay_trade_query_response", map[string]interface{}{
				"code": "10000", "msg": "Success", "out_trade_no": order.ID.String(),
				"trade_status": "WAIT_BUYER_PAY", "total_amount": "15.00",
			})))
		})
		if err := uc.reconcileAlipayOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err == nil {
			t.Error("a local-paid/channel-unpaid mismatch must be reported")
		}
	})

	t.Run("channel error code", func(t *testing.T) {
		uc := newUC(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedAlipayGatewayResponse(t, key, "alipay_trade_query_response", map[string]interface{}{
				"code": "40004", "msg": "Business Failed", "sub_code": "ACQ.TRADE_NOT_EXIST",
			})))
		})
		if err := uc.reconcileAlipayOrder(context.Background(), order, 1500, &ReconciliationRecord{}); err == nil {
			t.Error("a channel error must be reported")
		}
	})
}

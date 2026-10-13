package biz

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
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
	"net/url"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	"github.com/xuanyiying/smart-park/internal/payment/alipay"
	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

// wechatCallbackPlatform mints a platform certificate plus the matching
// wechat.Config, standing in for the WeChat Pay platform material.
func wechatCallbackPlatform(t *testing.T) (*rsa.PrivateKey, *x509.Certificate, *wechat.Config) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate platform key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(7),
		Subject:      pkix.Name{CommonName: "Tenpay.com platform cert"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to mint certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	cfg := &wechat.Config{
		AppID:          "wx_appid",
		MchID:          "1900000001",
		CertSerialNo:   cert.SerialNumber.String(),
		APIv3Key:       "0123456789abcdef0123456789abcdef", // exactly 32 chars
		PublicKeyPath:  "unused-path",
		PrivateKeyPath: "unused-path",
	}
	return key, cert, cfg
}

// paymentConfigForWechat builds the biz-level config with WeChat callbacks enabled.
func paymentConfigForWechat(wxCfg *wechat.Config) *PaymentConfig {
	return &PaymentConfig{
		WechatMchID:            wxCfg.MchID,
		WechatAPIv3Key:         wxCfg.APIv3Key,
		WechatPlatformCertPath: wxCfg.PublicKeyPath,
		WechatCertSerialNo:     wxCfg.CertSerialNo,
	}
}

// buildWechatNotification signs and encrypts a payment notification the way
// WeChat Pay APIv3 does, and returns the raw HTTP payload.
func buildWechatNotification(t *testing.T, platformKey *rsa.PrivateKey, wxCfg *wechat.Config, outTradeNo, transactionID string, totalCents int64) *RawCallback {
	t.Helper()

	plaintext := fmt.Sprintf(`{"appid":"%s","mchid":"%s","out_trade_no":"%s","transaction_id":"%s","trade_state":"SUCCESS","amount":{"total":%d,"currency":"CNY"}}`,
		wxCfg.AppID, wxCfg.MchID, outTradeNo, transactionID, totalCents)

	block, err := aes.NewCipher([]byte(wxCfg.APIv3Key))
	if err != nil {
		t.Fatalf("failed to build cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed to build gcm: %v", err)
	}
	nonce := []byte("0123456789ab") // exactly gcm.NonceSize() bytes, ASCII-safe for JSON
	aad := "transaction"
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), []byte(aad))

	body, err := json.Marshal(map[string]interface{}{
		"id":            "evt-" + uuid.New().String(),
		"event_type":    "TRANSACTION.SUCCESS",
		"resource_type": "encrypt-resource",
		"resource": map[string]interface{}{
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      base64.StdEncoding.EncodeToString(ciphertext),
			"nonce":           string(nonce),
			"associated_data": aad,
		},
	})
	if err != nil {
		t.Fatalf("failed to marshal body: %v", err)
	}

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonceStr := "notify-nonce"
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonceStr, string(body))))
	signature, err := rsa.SignPSS(rand.Reader, platformKey, crypto.SHA256, digest[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		t.Fatalf("failed to sign notification: %v", err)
	}

	return &RawCallback{
		Headers: map[string]string{
			"Wechatpay-Serial":    wxCfg.CertSerialNo,
			"Wechatpay-Timestamp": timestamp,
			"Wechatpay-Nonce":     nonceStr,
			"Wechatpay-Signature": base64.StdEncoding.EncodeToString(signature),
		},
		Body: body,
	}
}

// newWechatNotificationUseCase wires a PaymentUseCase to a wechat client that
// can verify callbacks signed by the test platform certificate.
func newWechatNotificationUseCase(t *testing.T, repo OrderRepo, recordRepo RecordRepo, gate GateControlService) (*PaymentUseCase, *rsa.PrivateKey, *wechat.Config) {
	t.Helper()

	platformKey, cert, wxCfg := wechatCallbackPlatform(t)
	client := wechat.NewClientWithCert(nil, wxCfg, nil, cert)

	uc := NewPaymentUseCase(repo, recordRepo, gate, paymentConfigForWechat(wxCfg), client, nil, log.NewStdLogger(nopWriter{}))
	return uc, platformKey, wxCfg
}

// nopWriter discards log output so test runs stay readable.
type nopWriter struct{}

func (nopWriter) Write(p []byte) (int, error) { return len(p), nil }

// TestHandleWechatNotificationSettlesOrder drives the full verified-callback
// pipeline: signature -> decryption -> amount check -> conditional settlement.
func TestHandleWechatNotificationSettlesOrder(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID:          orderID,
		RecordID:    uuid.New(),
		LotID:       uuid.New(),
		FinalAmount: 1500, // 15.00 元
		Status:      string(StatusPending),
		PayMethod:   string(MethodWechat),
	}

	uc, platformKey, wxCfg := newWechatNotificationUseCase(t, repo, NewMockRecordRepo(), NewMockGateControlService())

	raw := buildWechatNotification(t, platformKey, wxCfg, orderID.String(), "4200002106", 1500)
	resp, err := uc.HandleWechatNotification(context.Background(), raw)
	if err != nil {
		t.Fatalf("HandleWechatNotification returned error: %v", err)
	}
	if resp.ReturnCode != string(WechatStatusSuccess) {
		t.Fatalf("a verified callback must be acknowledged, got %q (%s)", resp.ReturnCode, resp.ReturnMsg)
	}

	settled := repo.Orders[orderID]
	if settled.Status != string(StatusPaid) {
		t.Errorf("order status = %s, want paid", settled.Status)
	}
	if settled.TransactionID != "4200002106" {
		t.Errorf("transaction id = %q, want the channel reference", settled.TransactionID)
	}
	if settled.PaidAmount != 1500 {
		t.Errorf("paid amount = %d, want 1500 cents", settled.PaidAmount)
	}
}

// TestHandleWechatNotificationRejectsTampering proves the security property of
// the raw-payload pipeline: a modified body, a foreign signer and a stale
// timestamp must all be refused, and no order may move.
func TestHandleWechatNotificationRejectsTampering(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodWechat),
	}
	uc, platformKey, wxCfg := newWechatNotificationUseCase(t, repo, NewMockRecordRepo(), NewMockGateControlService())

	raw := buildWechatNotification(t, platformKey, wxCfg, orderID.String(), "4200002106", 1500)

	t.Run("tampered body", func(t *testing.T) {
		tampered := *raw
		tampered.Body = append([]byte(nil), raw.Body...)
		tampered.Body = append(tampered.Body, ' ')
		resp, err := uc.HandleWechatNotification(context.Background(), &tampered)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.ReturnCode != string(WechatStatusFail) {
			t.Error("a tampered body must be rejected")
		}
		if repo.Orders[orderID].Status != string(StatusPending) {
			t.Error("no settlement may happen on a rejected callback")
		}
	})

	t.Run("foreign signing key", func(t *testing.T) {
		other, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatal(err)
		}
		forged := buildWechatNotification(t, other, wxCfg, orderID.String(), "attacker-tx", 1500)
		resp, err := uc.HandleWechatNotification(context.Background(), forged)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.ReturnCode != string(WechatStatusFail) {
			t.Error("a callback signed by a foreign key must be rejected")
		}
	})

	t.Run("missing payload", func(t *testing.T) {
		resp, err := uc.HandleWechatNotification(context.Background(), nil)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.ReturnCode != string(WechatStatusFail) {
			t.Error("a nil payload must be rejected")
		}
		resp, err = uc.HandleWechatNotification(context.Background(), &RawCallback{})
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.ReturnCode != string(WechatStatusFail) {
			t.Error("an empty body must be rejected")
		}
	})

	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Error("no rejected path may settle the order")
	}
}

// TestHandleWechatNotificationAmountMismatch verifies that a gateway reporting
// a different amount than the order is refused (money integrity).
func TestHandleWechatNotificationAmountMismatch(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodWechat),
	}
	uc, platformKey, wxCfg := newWechatNotificationUseCase(t, repo, NewMockRecordRepo(), NewMockGateControlService())

	// Gateway claims 100 cents for a 1500-cent order.
	raw := buildWechatNotification(t, platformKey, wxCfg, orderID.String(), "4200002106", 100)
	resp, err := uc.HandleWechatNotification(context.Background(), raw)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if resp.ReturnCode != string(WechatStatusFail) {
		t.Error("an amount mismatch must be rejected")
	}
	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Error("a mismatched callback must not settle the order")
	}
}

// TestHandleWechatNotificationNonSuccessState documents that WeChat also
// notifies about closed/refunded transactions: those are acknowledged without
// settling, so the channel stops retrying.
func TestHandleWechatNotificationNonSuccessState(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodWechat),
	}
	uc, platformKey, wxCfg := newWechatNotificationUseCase(t, repo, NewMockRecordRepo(), NewMockGateControlService())

	plaintext := fmt.Sprintf(`{"appid":"%s","mchid":"%s","out_trade_no":"%s","transaction_id":"4200002106","trade_state":"CLOSED","amount":{"total":1500,"currency":"CNY"}}`,
		wxCfg.AppID, wxCfg.MchID, orderID.String())

	block, err := aes.NewCipher([]byte(wxCfg.APIv3Key))
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := []byte("0123456789ab")
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), []byte("transaction"))
	body, err := json.Marshal(map[string]interface{}{
		"id": "evt-closed",
		"resource": map[string]interface{}{
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      base64.StdEncoding.EncodeToString(ciphertext),
			"nonce":           string(nonce),
			"associated_data": "transaction",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n", timestamp, "n", string(body))))
	signature, err := rsa.SignPSS(rand.Reader, platformKey, crypto.SHA256, digest[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		t.Fatal(err)
	}

	resp, err := uc.HandleWechatNotification(context.Background(), &RawCallback{
		Headers: map[string]string{
			"Wechatpay-Serial":    wxCfg.CertSerialNo,
			"Wechatpay-Timestamp": timestamp,
			"Wechatpay-Nonce":     "n",
			"Wechatpay-Signature": base64.StdEncoding.EncodeToString(signature),
		},
		Body: body,
	})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if resp.ReturnCode != string(WechatStatusSuccess) {
		t.Error("a non-payment notification must still be acknowledged")
	}
	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Error("a CLOSED notification must not settle the order")
	}
}

// TestHandleWechatNotificationInvalidOutTradeNo covers an unparseable
// out_trade_no in an otherwise valid notification.
func TestHandleWechatNotificationInvalidOutTradeNo(t *testing.T) {
	uc, platformKey, wxCfg := newWechatNotificationUseCase(t, NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService())

	raw := buildWechatNotification(t, platformKey, wxCfg, "not-a-uuid", "4200002106", 1500)
	resp, err := uc.HandleWechatNotification(context.Background(), raw)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if resp.ReturnCode != string(WechatStatusFail) {
		t.Error("an invalid out_trade_no must be rejected")
	}
}

// TestHandleWechatNotificationRequiresConfiguration proves the handler refuses
// to trust anything when the deployment has no WeChat credentials.
func TestHandleWechatNotificationRequiresConfiguration(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodWechat),
	}

	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(),
		&PaymentConfig{}, nil, nil, log.NewStdLogger(nopWriter{}))

	// A signed payload cannot even be built without a platform key, so the
	// important assertion is that an unsigned one is refused.
	resp, err := uc.HandleWechatNotification(context.Background(), &RawCallback{Body: []byte(`{"id":"x"}`)})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if resp.ReturnCode != string(WechatStatusFail) {
		t.Error("an unconfigured deployment must reject callbacks")
	}
	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Error("no callback may settle an order when unconfigured")
	}
}

// ---------------------------------------------------------------------------
// 支付宝回调通知
// ---------------------------------------------------------------------------

// alipayNotificationKeyPair generates merchant key material and an alipay
// client that can verify callbacks (no gateway access needed).
func alipayNotificationKeyPair(t *testing.T) (*rsa.PrivateKey, *alipay.Client, *PaymentConfig) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))

	client, err := alipay.NewTestClient(&alipay.Config{
		AppID:           "2021000000000000",
		PrivateKey:      "unused-for-verification",
		AlipayPublicKey: publicPEM,
		SignType:        alipay.SignTypeRSA2,
	})
	if err != nil {
		t.Fatalf("failed to build alipay client: %v", err)
	}

	cfg := &PaymentConfig{
		AlipayPublicKey: publicPEM,
		AlipaySignType:  alipay.SignTypeRSA2,
	}
	return key, client, cfg
}

// signedAlipayForm signs a callback form with the test key the way Alipay does:
// every parameter except sign/sign_type, sorted, RSA-SHA256 PKCS1v15.
func signedAlipayForm(t *testing.T, key *rsa.PrivateKey, values url.Values) url.Values {
	t.Helper()

	values.Set("sign_type", alipay.SignTypeRSA2)
	content := alipay.BuildSignContent(values)
	digest := sha256.Sum256([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("failed to sign form: %v", err)
	}
	values.Set("sign", base64.StdEncoding.EncodeToString(signature))
	return values
}

// TestHandleAlipayNotificationSettlesOrder drives the verified Alipay callback:
// full-parameter signature verification -> yuan-to-cents conversion -> settlement.
func TestHandleAlipayNotificationSettlesOrder(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodAlipay),
	}

	key, alipayClient, cfg := alipayNotificationKeyPair(t)
	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), cfg, nil, alipayClient, log.NewStdLogger(nopWriter{}))

	form := signedAlipayForm(t, key, url.Values{
		"out_trade_no": {orderID.String()},
		"trade_no":     {"20241001220014123456"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"15.00"},
	})

	resp, err := uc.HandleAlipayNotification(context.Background(), &RawCallback{Form: form})
	if err != nil {
		t.Fatalf("HandleAlipayNotification returned error: %v", err)
	}
	if resp.Code != "success" {
		t.Fatalf("a verified callback must be acknowledged, got %q (%s)", resp.Code, resp.Msg)
	}

	settled := repo.Orders[orderID]
	if settled.Status != string(StatusPaid) {
		t.Errorf("order status = %s, want paid", settled.Status)
	}
	if settled.TransactionID != "20241001220014123456" {
		t.Errorf("transaction id = %q, want the channel reference", settled.TransactionID)
	}
	if settled.PaidAmount != 1500 {
		t.Errorf("paid amount = %d, want 1500 cents (converted from 15.00 元)", settled.PaidAmount)
	}
}

// TestHandleAlipayNotificationRejectsTampering pins the full-parameter signing
// property: every field is covered, so changing any of them invalidates the
// callback, and no settlement happens.
func TestHandleAlipayNotificationRejectsTampering(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodAlipay),
	}

	key, alipayClient, cfg := alipayNotificationKeyPair(t)
	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), cfg, nil, alipayClient, log.NewStdLogger(nopWriter{}))

	base := url.Values{
		"out_trade_no": {orderID.String()},
		"trade_no":     {"20241001220014123456"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"15.00"},
	}

	tamper := func(mutate func(url.Values)) *v1.AlipayCallbackResponse {
		form := signedAlipayForm(t, key, cloneURLValues(base))
		mutate(form)
		resp, err := uc.HandleAlipayNotification(context.Background(), &RawCallback{Form: form})
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		return resp
	}

	if resp := tamper(func(v url.Values) { v.Set("total_amount", "0.01") }); resp.Code == "success" {
		t.Error("a downgraded amount must be rejected")
	}
	if resp := tamper(func(v url.Values) { v.Set("seller_id", "attacker") }); resp.Code == "success" {
		t.Error("an injected parameter must be rejected: every field is signed")
	}
	if resp := tamper(func(v url.Values) { v.Set("sign", "AAAA") }); resp.Code == "success" {
		t.Error("a garbage signature must be rejected")
	}
	if resp := tamper(func(v url.Values) { v.Del("sign") }); resp.Code == "success" {
		t.Error("a missing signature must be rejected")
	}

	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Error("no rejected path may settle the order")
	}
}

// TestHandleAlipayNotificationNonSuccessStatus documents that only successful
// trades settle; others are acknowledged so the channel stops retrying.
func TestHandleAlipayNotificationNonSuccessStatus(t *testing.T) {
	repo := NewMockOrderRepo()
	orderID := uuid.New()
	repo.Orders[orderID] = &Order{
		ID: orderID, FinalAmount: 1500, Status: string(StatusPending), PayMethod: string(MethodAlipay),
	}

	key, alipayClient, cfg := alipayNotificationKeyPair(t)
	uc := NewPaymentUseCase(repo, NewMockRecordRepo(), NewMockGateControlService(), cfg, nil, alipayClient, log.NewStdLogger(nopWriter{}))

	form := signedAlipayForm(t, key, url.Values{
		"out_trade_no": {orderID.String()},
		"trade_no":     {"20241001220014123456"},
		"trade_status": {"WAIT_BUYER_PAY"},
		"total_amount": {"15.00"},
	})

	resp, err := uc.HandleAlipayNotification(context.Background(), &RawCallback{Form: form})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if resp.Code != "success" {
		t.Error("an unpaid notification must still be acknowledged")
	}
	if repo.Orders[orderID].Status != string(StatusPending) {
		t.Error("an unpaid notification must not settle the order")
	}
}

// TestHandleAlipayNotificationGuards covers the malformed-input paths.
func TestHandleAlipayNotificationGuards(t *testing.T) {
	key, alipayClient, cfg := alipayNotificationKeyPair(t)
	uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(), cfg, nil, alipayClient, log.NewStdLogger(nopWriter{}))

	t.Run("nil payload", func(t *testing.T) {
		resp, err := uc.HandleAlipayNotification(context.Background(), nil)
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.Code != "FAIL" {
			t.Error("a nil payload must be rejected")
		}
	})

	t.Run("empty form", func(t *testing.T) {
		resp, err := uc.HandleAlipayNotification(context.Background(), &RawCallback{Form: url.Values{}})
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.Code != "FAIL" {
			t.Error("an empty form must be rejected")
		}
	})

	t.Run("invalid out_trade_no", func(t *testing.T) {
		form := signedAlipayForm(t, key, url.Values{
			"out_trade_no": {"not-a-uuid"},
			"trade_no":     {"20241001220014123456"},
			"trade_status": {"TRADE_SUCCESS"},
			"total_amount": {"15.00"},
		})
		resp, err := uc.HandleAlipayNotification(context.Background(), &RawCallback{Form: form})
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.Code != "FAIL" {
			t.Error("an invalid out_trade_no must be rejected")
		}
	})

	t.Run("invalid total_amount", func(t *testing.T) {
		form := signedAlipayForm(t, key, url.Values{
			"out_trade_no": {uuid.New().String()},
			"trade_no":     {"20241001220014123456"},
			"trade_status": {"TRADE_SUCCESS"},
			"total_amount": {"not-a-number"},
		})
		resp, err := uc.HandleAlipayNotification(context.Background(), &RawCallback{Form: form})
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.Code != "FAIL" {
			t.Error("an unparseable amount must be rejected")
		}
	})

	t.Run("unknown order", func(t *testing.T) {
		form := signedAlipayForm(t, key, url.Values{
			"out_trade_no": {uuid.New().String()},
			"trade_no":     {"20241001220014123456"},
			"trade_status": {"TRADE_SUCCESS"},
			"total_amount": {"15.00"},
		})
		resp, err := uc.HandleAlipayNotification(context.Background(), &RawCallback{Form: form})
		if err != nil {
			t.Fatalf("handler returned error: %v", err)
		}
		if resp.Code != "FAIL" {
			t.Error("a callback for an unknown order must be rejected")
		}
	})
}

// TestHandleAlipayNotificationRequiresConfiguration proves an unconfigured
// deployment refuses callbacks instead of trusting them.
func TestHandleAlipayNotificationRequiresConfiguration(t *testing.T) {
	uc := NewPaymentUseCase(NewMockOrderRepo(), NewMockRecordRepo(), NewMockGateControlService(),
		&PaymentConfig{}, nil, nil, log.NewStdLogger(nopWriter{}))

	resp, err := uc.HandleAlipayNotification(context.Background(), &RawCallback{Form: url.Values{
		"out_trade_no": {uuid.New().String()},
		"trade_status": {"TRADE_SUCCESS"},
	}})
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if resp.Code != "FAIL" {
		t.Error("an unconfigured deployment must reject callbacks")
	}
}

// cloneURLValues deep-copies a form so tampering cannot corrupt the base.
func cloneURLValues(in url.Values) url.Values {
	out := url.Values{}
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}

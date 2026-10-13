package wechat_test

import (
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"

	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

// ---------------------------------------------------------------------------
// 测试基建：密钥与"重定向传输"
//
// wechat.NewClient（生产构造器）内部会向微信服务器下载平台证书，无法离线运行；
// 因此这里用 SDK 的 WithMerchantCredential + WithHTTPClient 直接构造 core.Client，
// 把发往 api.mch.weixin.qq.com 的请求改写到本地 httptest 服务器。覆盖的是本仓库
// 客户端代码（URL 构造、响应解析、金额单位），而非 SDK 本身。
// ---------------------------------------------------------------------------

// redirectTransport rewrites every outgoing request to the local test server,
// preserving the path so handlers can route on it.
type redirectTransport struct{ base string }

func (t redirectTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	u, err := url.Parse(t.base)
	if err != nil {
		return nil, err
	}
	r.URL.Scheme = u.Scheme
	r.URL.Host = u.Host
	return http.DefaultTransport.RoundTrip(r)
}

// merchantKeyMaterial generates the merchant RSA key plus the platform
// certificate that stands in for WeChat's.
func merchantKeyMaterial(t *testing.T) (*rsa.PrivateKey, *x509.Certificate, *wechat.Config) {
	t.Helper()

	_, cert, certPath := testPlatform(t)

	cfg := testConfig(certPath)
	cfg.CertSerialNo = cert.SerialNumber.String()

	// A merchant key that is distinct from the platform key.
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate merchant key: %v", err)
	}
	return key, cert, cfg
}

// newRedirectedClient builds a wechat.Client whose HTTP traffic lands on srv.
func newRedirectedClient(t *testing.T, srv *httptest.Server, cfg *wechat.Config, merchantKey *rsa.PrivateKey, platformCert *x509.Certificate) *wechat.Client {
	t.Helper()

	coreClient, err := core.NewClient(context.Background(),
		option.WithMerchantCredential(cfg.MchID, cfg.CertSerialNo, merchantKey),
		option.WithoutValidator(),
		option.WithHTTPClient(&http.Client{Transport: redirectTransport{base: srv.URL}}),
	)
	if err != nil {
		t.Fatalf("failed to build core client: %v", err)
	}
	return wechat.NewClientWithCert(coreClient, cfg, merchantKey, platformCert)
}

func writeMerchantKey(t *testing.T) string {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatalf("failed to marshal key: %v", err)
	}
	path := filepath.Join(t.TempDir(), "merchant_key.pem")
	if err := os.WriteFile(path, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}), 0o600); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}
	return path
}

func newGatewayServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv
}

// ---------------------------------------------------------------------------
// 生产构造器的快速失败路径
// ---------------------------------------------------------------------------

// TestNewClientFailsFastOnBadKeyMaterial covers everything the production
// constructor can check before it reaches the network: validation, the merchant
// private key file and the platform certificate file. The remaining happy path
// of NewClient downloads platform certificates from WeChat at construction time
// and is therefore exercised by the channel sandbox smoke test (tests/sandbox),
// not by unit tests.
func TestNewClientFailsFastOnBadKeyMaterial(t *testing.T) {
	_, _, certPath := testPlatform(t)

	t.Run("missing private key file", func(t *testing.T) {
		cfg := testConfig(certPath)
		cfg.PrivateKeyPath = filepath.Join(t.TempDir(), "absent.pem")
		if _, err := wechat.NewClient(cfg); err == nil {
			t.Error("a missing private key must fail fast")
		}
	})

	t.Run("invalid private key PEM", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad_key.pem")
		if err := os.WriteFile(path, []byte("not a pem"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := testConfig(certPath)
		cfg.PrivateKeyPath = path
		if _, err := wechat.NewClient(cfg); err == nil {
			t.Error("an unparseable private key must fail fast")
		}
	})

	t.Run("missing platform certificate", func(t *testing.T) {
		cfg := testConfig(certPath)
		cfg.PrivateKeyPath = writeMerchantKey(t)
		cfg.PublicKeyPath = filepath.Join(t.TempDir(), "absent_cert.pem")
		if _, err := wechat.NewClient(cfg); err == nil {
			t.Error("a missing platform certificate must fail fast")
		}
	})

	t.Run("invalid platform certificate", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad_cert.pem")
		if err := os.WriteFile(path, []byte("-----BEGIN CERTIFICATE-----\nnope\n-----END CERTIFICATE-----\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg := testConfig(certPath)
		cfg.PrivateKeyPath = writeMerchantKey(t)
		cfg.PublicKeyPath = path
		if _, err := wechat.NewClient(cfg); err == nil {
			t.Error("an unparseable platform certificate must fail fast")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		cfg := testConfig(certPath)
		cfg.APIv3Key = "too-short"
		if _, err := wechat.NewClient(cfg); err == nil {
			t.Error("an invalid APIv3 key must be rejected before any network access")
		}
	})
}

// ---------------------------------------------------------------------------
// 回调验签时间窗口与证书缺失
// ---------------------------------------------------------------------------

// TestVerifyCallbackTimestampWindow pins the replay window on both sides.
func TestVerifyCallbackTimestampWindow(t *testing.T) {
	now := time.Now()

	if err := wechat.VerifyCallbackTimestamp("not-a-number", now); err == nil {
		t.Error("a non-numeric timestamp must be rejected")
	}
	if err := wechat.VerifyCallbackTimestamp(strconv.FormatInt(now.Add(-4*time.Minute).Unix(), 10), now); err != nil {
		t.Errorf("a recent callback must be accepted: %v", err)
	}
	if err := wechat.VerifyCallbackTimestamp(strconv.FormatInt(now.Add(-10*time.Minute).Unix(), 10), now); err == nil {
		t.Error("a stale callback must be rejected")
	}
	if err := wechat.VerifyCallbackTimestamp(strconv.FormatInt(now.Add(10*time.Minute).Unix(), 10), now); err == nil {
		t.Error("a future-dated callback must be rejected")
	}
}

// TestVerifyCallbackSignatureWithoutCertificate proves the verifier refuses to
// "succeed" when no platform certificate is configured.
func TestVerifyCallbackSignatureWithoutCertificate(t *testing.T) {
	_, _, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, nil)

	if err := client.VerifyCallbackSignature(cfg.CertSerialNo, "1", "n", "body", "c2ln"); err == nil {
		t.Error("signature verification without a platform certificate must fail")
	}
	if client.PlatformCertificate() != nil {
		t.Error("no certificate should be exposed when none was configured")
	}
}

// TestParseAndVerifyCallbackHappyPath drives the complete APIv3 pipeline:
// timestamp freshness -> RSA-PSS signature -> AES-256-GCM decryption.
func TestParseAndVerifyCallbackHappyPath(t *testing.T) {
	platformKey, cert, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, cert)

	plaintext := `{"appid":"wx_appid","mchid":"1900000001","out_trade_no":"order-9","transaction_id":"4200009999","trade_state":"SUCCESS","amount":{"total":1500,"currency":"CNY"}}`

	block, err := aes.NewCipher([]byte(cfg.APIv3Key))
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

	body := map[string]interface{}{
		"id":            "evt-9",
		"event_type":    wechat.CallbackEventTypePayment,
		"resource_type": "encrypt-resource",
		"resource": map[string]interface{}{
			"algorithm":       "AEAD_AES_256_GCM",
			"ciphertext":      base64.StdEncoding.EncodeToString(ciphertext),
			"nonce":           string(nonce),
			"associated_data": aad,
		},
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonceStr := "callback-nonce"
	message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonceStr, string(bodyJSON))
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPSS(rand.Reader, platformKey, crypto.SHA256, digest[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		t.Fatal(err)
	}

	headers := map[string]string{
		"Wechatpay-Serial":    cfg.CertSerialNo,
		"Wechatpay-Timestamp": timestamp,
		"Wechatpay-Nonce":     nonceStr,
		"Wechatpay-Signature": base64.StdEncoding.EncodeToString(signature),
	}

	transaction, err := client.ParseAndVerifyCallback(headers, bodyJSON, time.Now())
	if err != nil {
		t.Fatalf("a correctly signed and encrypted callback must be accepted: %v", err)
	}
	if transaction.OutTradeNo != "order-9" {
		t.Errorf("out_trade_no = %q, want order-9", transaction.OutTradeNo)
	}
	if transaction.TransactionID != "4200009999" {
		t.Errorf("transaction_id = %q, want 4200009999", transaction.TransactionID)
	}
	if transaction.TradeState != wechat.TradeStateSuccess {
		t.Errorf("trade_state = %q, want SUCCESS", transaction.TradeState)
	}
	if transaction.Amount.Total != 1500 {
		t.Errorf("amount.total = %d, want 1500 (cents)", transaction.Amount.Total)
	}
}

// TestDecryptResourceRejectsBadInput covers the guard rails around AES-GCM
// decryption: wrong algorithm, bad base64, wrong nonce length and tampered
// ciphertext must all fail closed.
func TestDecryptResourceRejectsBadInput(t *testing.T) {
	_, cert, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, cert)

	valid := wechat.Resource{
		Algorithm:      "AEAD_AES_256_GCM",
		Ciphertext:     "AAAA",
		Nonce:          "0123456789ab",
		AssociatedData: "transaction",
	}

	wrongAlgo := valid
	wrongAlgo.Algorithm = "AES_256_CBC"
	if _, err := client.DecryptResource(wrongAlgo); err == nil {
		t.Error("an unsupported algorithm must be rejected")
	}

	badBase64 := valid
	badBase64.Ciphertext = "!!!not-base64!!!"
	if _, err := client.DecryptResource(badBase64); err == nil {
		t.Error("invalid base64 ciphertext must be rejected")
	}

	badNonce := valid
	badNonce.Nonce = "tooshort"
	if _, err := client.DecryptResource(badNonce); err == nil {
		t.Error("an invalid nonce length must be rejected")
	}

	// Undecryptable payload (valid base64, not valid GCM ciphertext).
	if _, err := client.DecryptResource(valid); err == nil {
		t.Error("garbage ciphertext must fail to decrypt")
	}
}

// TestConfigAPIv3KeyAccessor documents the accessor operators use to sanity
// check key material.
func TestConfigAPIv3KeyAccessor(t *testing.T) {
	_, cert, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, cert)

	if client.APIv3Key() != cfg.APIv3Key {
		t.Errorf("APIv3Key() = %q, want %q", client.APIv3Key(), cfg.APIv3Key)
	}
}

// ---------------------------------------------------------------------------
// 网关调用（查单/关单/退款/退款查询/下单）——全部经重定向传输打到本地服务器
// ---------------------------------------------------------------------------

// TestQueryOrderAgainstGateway drives the wrapper's parsing of the channel's
// query response, including the map keys reconciliation depends on.
func TestQueryOrderAgainstGateway(t *testing.T) {
	merchantKey, platformCert, cfg := merchantKeyMaterial(t)

	t.Run("successful payment", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/pay/transactions/out-trade-no/order-1" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("mchid"); got != cfg.MchID {
				t.Errorf("mchid = %q, want %q", got, cfg.MchID)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"out_trade_no":   "order-1",
				"transaction_id": "4200002106",
				"trade_state":    "SUCCESS",
				"amount":         map[string]interface{}{"total": 1500, "currency": "CNY"},
			})
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		result, err := client.QueryOrder(context.Background(), "order-1")
		if err != nil {
			t.Fatalf("QueryOrder failed: %v", err)
		}
		if result["trade_state"] != "SUCCESS" {
			t.Errorf("trade_state = %v, want SUCCESS", result["trade_state"])
		}
		if result["transaction_id"] != "4200002106" {
			t.Errorf("transaction_id = %v, want 4200002106", result["transaction_id"])
		}
		// Reconciliation compares this against the order amount in cents.
		if result["total_fee"] != int64(1500) {
			t.Errorf("total_fee = %v (%T), want int64 1500", result["total_fee"], result["total_fee"])
		}
	})

	t.Run("unpaid order reports no transaction id", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"out_trade_no": "order-2",
				"trade_state":  "NOTPAY",
			})
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		result, err := client.QueryOrder(context.Background(), "order-2")
		if err != nil {
			t.Fatalf("QueryOrder failed: %v", err)
		}
		if result["trade_state"] != "NOTPAY" {
			t.Errorf("trade_state = %v, want NOTPAY", result["trade_state"])
		}
		if _, ok := result["transaction_id"]; ok {
			t.Error("an unpaid order must not carry a transaction id")
		}
		if _, ok := result["total_fee"]; ok {
			t.Error("an unpaid order must not carry an amount")
		}
	})

	t.Run("gateway error surfaces", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		if _, err := client.QueryOrder(context.Background(), "order-3"); err == nil {
			t.Error("a gateway failure must surface as an error")
		}
	})
}

func TestCloseOrderAgainstGateway(t *testing.T) {
	merchantKey, platformCert, cfg := merchantKeyMaterial(t)

	srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/pay/transactions/out-trade-no/order-1/close" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
	if err := client.CloseOrder(context.Background(), "order-1"); err != nil {
		t.Fatalf("CloseOrder failed: %v", err)
	}
}

// TestRefundAgainstGateway covers the money path: a confirmed refund returns no
// error, a rejected one fails loudly (never silently "succeeds").
func TestRefundAgainstGateway(t *testing.T) {
	merchantKey, platformCert, cfg := merchantKeyMaterial(t)

	t.Run("refund confirmed", func(t *testing.T) {
		var body map[string]interface{}
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/refund/domestic/refunds" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"refund_id":     "50000000382019073109739578859",
				"out_refund_no": "refund-1",
				"status":        "PROCESSING",
			})
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		if err := client.Refund(context.Background(), "order-1", "refund-1", 1500, 500); err != nil {
			t.Fatalf("Refund failed: %v", err)
		}

		amount, _ := body["amount"].(map[string]interface{})
		if amount["total"] != float64(1500) || amount["refund"] != float64(500) {
			t.Errorf("refund amounts = %v, want total 1500 / refund 500 (cents)", body["amount"])
		}
	})

	t.Run("refund rejected", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code": "PARAM_ERROR", "message": "invalid"})
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		if err := client.Refund(context.Background(), "order-1", "refund-1", 1500, 500); err == nil {
			t.Error("a rejected refund must surface as an error")
		}
	})
}

func TestQueryRefundAgainstGateway(t *testing.T) {
	merchantKey, platformCert, cfg := merchantKeyMaterial(t)

	t.Run("status returned", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/refund/domestic/refunds/refund-1" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "SUCCESS"})
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		status, err := client.QueryRefund(context.Background(), "order-1", "refund-1")
		if err != nil {
			t.Fatalf("QueryRefund failed: %v", err)
		}
		if status != "SUCCESS" {
			t.Errorf("status = %q, want SUCCESS", status)
		}
	})

	t.Run("empty status", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{})
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		status, err := client.QueryRefund(context.Background(), "order-1", "refund-1")
		if err != nil {
			t.Fatalf("QueryRefund failed: %v", err)
		}
		if status != "" {
			t.Errorf("status = %q, want empty", status)
		}
	})
}

// TestCreatePaymentsAgainstGateway covers order creation (native + JSAPI),
// which also exercises request signing.
func TestCreatePaymentsAgainstGateway(t *testing.T) {
	merchantKey, platformCert, cfg := merchantKeyMaterial(t)

	t.Run("native pay returns the code url", func(t *testing.T) {
		var body map[string]interface{}
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/pay/transactions/native" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("failed to decode request: %v", err)
			}
			if r.Header.Get("Authorization") == "" {
				t.Error("requests must carry a signed Authorization header")
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"code_url": "weixin://wxpay/bizpayurl?pr=abc"})
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		codeURL, err := client.CreateNativePay(context.Background(), "order-1", 1500, "停车费")
		if err != nil {
			t.Fatalf("CreateNativePay failed: %v", err)
		}
		if codeURL != "weixin://wxpay/bizpayurl?pr=abc" {
			t.Errorf("code_url = %q", codeURL)
		}
		if body["amount"].(map[string]interface{})["total"] != float64(1500) {
			t.Errorf("amount must be sent in cents, got %v", body["amount"])
		}
		if body["out_trade_no"] != "order-1" {
			t.Errorf("out_trade_no = %v, want order-1", body["out_trade_no"])
		}
	})

	t.Run("jsapi pay returns mini-program parameters", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/v3/pay/transactions/jsapi" {
				t.Errorf("unexpected path %s", r.URL.Path)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"prepay_id": "wx1234567890"})
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		params, err := client.CreateJSAPIPay(context.Background(), "order-1", 1500, "openid-1", "停车费")
		if err != nil {
			t.Fatalf("CreateJSAPIPay failed: %v", err)
		}
		if params["package"] != "prepay_id=wx1234567890" {
			t.Errorf("package = %v, want prepay_id=wx1234567890", params["package"])
		}
		if params["appId"] != cfg.AppID {
			t.Errorf("appId = %v, want %s", params["appId"], cfg.AppID)
		}
		if params["paySign"] == "" || params["paySign"] == nil {
			t.Error("paySign must be produced from the merchant key")
		}
		if params["timeStamp"] == "" || params["nonceStr"] == "" {
			t.Error("timeStamp and nonceStr must be populated")
		}
	})

	t.Run("order creation failure surfaces", func(t *testing.T) {
		srv := newGatewayServer(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		})

		client := newRedirectedClient(t, srv, cfg, merchantKey, platformCert)
		if _, err := client.CreateNativePay(context.Background(), "order-1", 1500, "停车费"); err == nil {
			t.Error("a failed order creation must surface as an error")
		}
		if _, err := client.CreateJSAPIPay(context.Background(), "order-1", 1500, "openid-1", "停车费"); err == nil {
			t.Error("a failed JSAPI creation must surface as an error")
		}
	})
}

package alipay

import (
	"context"
	"crypto"
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
	"strings"
	"testing"

	alipaysdk "github.com/smartwalle/alipay/v3"
)

// ---------------------------------------------------------------------------
// 测试基建
//
// 与微信侧同样的思路：生产构造器 wechat.NewClient 会联网下载平台证书，支付宝侧的
// NewClient 则不联网（只做密钥解析），因此本文件既测 NewClient 的密钥装载，也用
// 重定向传输把 SDK 的网关请求打到本地 httptest 服务器，覆盖查单/退款/关单等
// 网关方法的 URL 构造、金额单位（分→元串）与响应解析。
// ---------------------------------------------------------------------------

// gatewayRedirect rewrites SDK requests aimed at the Alipay gateway to the
// local test server, preserving method and body.
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

// testKeyPairPEM generates merchant key material: a PKCS8 private key PEM and
// the matching PKIX public key PEM (the "Alipay public key" a merchant loads).
func testKeyPairPEM(t *testing.T) (key *rsa.PrivateKey, privatePEM, publicPEM string) {
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

// signedGatewayResponse builds the JSON envelope the SDK expects, with a valid
// RSA-SHA256 signature over the response node (the SDK verifies responses when
// the public key is loaded).
func signedGatewayResponse(t *testing.T, key *rsa.PrivateKey, node string, payload map[string]interface{}) string {
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

// newRealClient builds a wrapper Client backed by a real SDK client whose HTTP
// traffic is redirected to srv.
func newRealClient(t *testing.T, srv *httptest.Server, key *rsa.PrivateKey, privatePEM, publicPEM string) *Client {
	t.Helper()

	sdk, err := alipaysdk.New("2021000000000000", privatePEM, false)
	if err != nil {
		t.Fatalf("failed to build sdk client: %v", err)
	}
	if err := sdk.LoadAliPayPublicKey(publicPEM); err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}
	sdk.Client = &http.Client{Transport: gatewayRedirect{base: srv.URL}}

	return &Client{
		client: sdk,
		config: &Config{
			AppID:           "2021000000000000",
			PrivateKey:      privatePEM,
			AlipayPublicKey: publicPEM,
			NotifyURL:       "https://example.com/notify",
			SignType:        SignTypeRSA2,
		},
	}
}

// ---------------------------------------------------------------------------
// 构造器
// ---------------------------------------------------------------------------

// TestNewClientWithRealKeyMaterial covers the production constructor: it parses
// the merchant key and loads the Alipay public key for callback verification.
func TestNewClientWithRealKeyMaterial(t *testing.T) {
	_, privatePEM, publicPEM := testKeyPairPEM(t)

	client, err := NewClient(&Config{
		AppID:           "2021000000000000",
		PrivateKey:      privatePEM,
		AlipayPublicKey: publicPEM,
		NotifyURL:       "https://example.com/notify",
		SignType:        SignTypeRSA2,
	})
	if err != nil {
		t.Fatalf("NewClient failed with valid key material: %v", err)
	}
	if _, err := client.gateway(); err != nil {
		t.Errorf("a client built from valid keys must expose a gateway: %v", err)
	}
}

func TestNewClientRejectsBadKeyMaterial(t *testing.T) {
	_, privatePEM, publicPEM := testKeyPairPEM(t)

	t.Run("invalid private key", func(t *testing.T) {
		if _, err := NewClient(&Config{
			AppID: "2021000000000000", PrivateKey: "not-a-key", AlipayPublicKey: publicPEM,
		}); err == nil {
			t.Error("an unparseable private key must fail fast")
		}
	})

	t.Run("invalid public key", func(t *testing.T) {
		if _, err := NewClient(&Config{
			AppID: "2021000000000000", PrivateKey: privatePEM, AlipayPublicKey: "not-a-key",
		}); err == nil {
			t.Error("an unparseable public key must fail fast")
		}
	})

	t.Run("missing fields", func(t *testing.T) {
		if _, err := NewClient(&Config{PrivateKey: privatePEM, AlipayPublicKey: publicPEM}); err == nil {
			t.Error("a missing app id must fail fast")
		}
	})
}

// ---------------------------------------------------------------------------
// 未配置网关时的失败路径
// ---------------------------------------------------------------------------

// TestGatewayMethodsRequireConfiguredClient proves that a client built without
// merchant credentials (NewTestClient, used for callback verification only)
// fails loudly on every gateway call instead of panicking or pretending.
func TestGatewayMethodsRequireConfiguredClient(t *testing.T) {
	_, _, publicPEM := testKeyPairPEM(t)

	client, err := NewTestClient(&Config{
		AppID:           "2021000000000000",
		PrivateKey:      "unused",
		AlipayPublicKey: publicPEM,
	})
	if err != nil {
		t.Fatalf("NewTestClient failed: %v", err)
	}

	ctx := context.Background()
	if _, err := client.CreateTradePagePay(ctx, "order-1", 1500, "停车费"); err == nil {
		t.Error("page pay without merchant credentials must fail")
	}
	if _, err := client.CreateTradeWapPay(ctx, "order-1", 1500, "停车费"); err == nil {
		t.Error("wap pay without merchant credentials must fail")
	}
	if _, err := client.CreateTradePreCreate(ctx, "order-1", 1500, "停车费"); err == nil {
		t.Error("precreate without merchant credentials must fail")
	}
	if _, err := client.QueryOrder(ctx, "order-1"); err == nil {
		t.Error("query without merchant credentials must fail")
	}
	if err := client.CloseOrder(ctx, "order-1"); err == nil {
		t.Error("close without merchant credentials must fail")
	}
	if err := client.Refund(ctx, "order-1", "refund-1", 500); err == nil {
		t.Error("refund without merchant credentials must fail")
	}
	if _, err := client.QueryRefund(ctx, "order-1", "refund-1"); err == nil {
		t.Error("refund query without merchant credentials must fail")
	}
	if _, err := client.DecodeNotification(ctx, url.Values{}); err == nil {
		t.Error("notification decoding without merchant credentials must fail")
	}
}

// ---------------------------------------------------------------------------
// 本地构造的支付 URL（不经过网络）
// ---------------------------------------------------------------------------

// TestCreateTradePayURLs covers the two redirect-style payments, which are pure
// local URL construction: the amount must appear in yuan with two decimals.
func TestCreateTradePayURLs(t *testing.T) {
	_, privatePEM, publicPEM := testKeyPairPEM(t)

	sdk, err := alipaysdk.New("2021000000000000", privatePEM, false)
	if err != nil {
		t.Fatalf("failed to build sdk client: %v", err)
	}
	client := &Client{client: sdk, config: &Config{
		AppID: "2021000000000000", PrivateKey: privatePEM, AlipayPublicKey: publicPEM,
		NotifyURL: "https://example.com/notify",
	}}

	for name, call := range map[string]func() (string, error){
		"page pay": func() (string, error) {
			return client.CreateTradePagePay(context.Background(), "order-1", 1500, "停车费")
		},
		"wap pay": func() (string, error) {
			return client.CreateTradeWapPay(context.Background(), "order-1", 1500, "停车费")
		},
	} {
		t.Run(name, func(t *testing.T) {
			payURL, err := call()
			if err != nil {
				t.Fatalf("%s failed: %v", name, err)
			}
			parsed, err := url.Parse(payURL)
			if err != nil {
				t.Fatalf("returned value is not a URL: %v", err)
			}
			q := parsed.Query()
			if q.Get("app_id") != "2021000000000000" {
				t.Errorf("app_id = %q, want the configured app id", q.Get("app_id"))
			}
			if q.Get("sign") == "" {
				t.Error("the payment URL must carry a signature")
			}

			// Business parameters travel inside the signed biz_content JSON.
			var biz map[string]interface{}
			if err := json.Unmarshal([]byte(q.Get("biz_content")), &biz); err != nil {
				t.Fatalf("biz_content is not valid JSON: %v", err)
			}
			if biz["out_trade_no"] != "order-1" {
				t.Errorf("out_trade_no = %v, want order-1", biz["out_trade_no"])
			}
			// 1500 分必须渲染为 "15.00" 元，而不是 "1500"。
			if biz["total_amount"] != "15.00" {
				t.Errorf("total_amount = %v, want 15.00", biz["total_amount"])
			}
			if biz["subject"] != "停车费" {
				t.Errorf("subject = %v, want 停车费", biz["subject"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 网关调用（重定向传输）
// ---------------------------------------------------------------------------

// TestQueryOrderAgainstGateway drives query parsing: a successful trade returns
// the status and yuan amount the reconciliation path compares against.
func TestQueryOrderAgainstGateway(t *testing.T) {
	key, privatePEM, publicPEM := testKeyPairPEM(t)

	t.Run("paid order", func(t *testing.T) {
		var form url.Values
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil {
				t.Errorf("failed to parse form: %v", err)
			}
			form = r.PostForm
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedGatewayResponse(t, key, "alipay_trade_query_response", map[string]interface{}{
				"code":         "10000",
				"msg":          "Success",
				"out_trade_no": "order-1",
				"trade_no":     "20241001220014123456",
				"trade_status": "TRADE_SUCCESS",
				"total_amount": "15.00",
			})))
		}))
		defer srv.Close()

		client := newRealClient(t, srv, key, privatePEM, publicPEM)
		rsp, err := client.QueryOrder(context.Background(), "order-1")
		if err != nil {
			t.Fatalf("QueryOrder failed: %v", err)
		}
		if rsp.Code != "10000" {
			t.Errorf("code = %q, want 10000", rsp.Code)
		}
		if rsp.TradeStatus != "TRADE_SUCCESS" {
			t.Errorf("trade_status = %q, want TRADE_SUCCESS", rsp.TradeStatus)
		}
		if rsp.TotalAmount != "15.00" {
			t.Errorf("total_amount = %q, want 15.00", rsp.TotalAmount)
		}
		if !strings.Contains(form.Get("biz_content"), `"out_trade_no":"order-1"`) {
			t.Errorf("biz_content = %q, want out_trade_no order-1", form.Get("biz_content"))
		}
	})

	t.Run("channel error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedGatewayResponse(t, key, "alipay_trade_query_response", map[string]interface{}{
				"code": "40004", "msg": "Business Failed", "sub_code": "ACQ.TRADE_NOT_EXIST",
			})))
		}))
		defer srv.Close()

		client := newRealClient(t, srv, key, privatePEM, publicPEM)
		rsp, err := client.QueryOrder(context.Background(), "order-1")
		if err != nil {
			t.Fatalf("QueryOrder must return the channel response, got error: %v", err)
		}
		if rsp.Code == "10000" {
			t.Error("a channel error must not be reported as success")
		}
	})

	t.Run("unsigned response rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"alipay_trade_query_response":{"code":"10000","msg":"Success"}}`))
		}))
		defer srv.Close()

		client := newRealClient(t, srv, key, privatePEM, publicPEM)
		if _, err := client.QueryOrder(context.Background(), "order-1"); err == nil {
			t.Error("an unsigned gateway response must be rejected")
		}
	})
}

// TestRefundAgainstGateway covers the refund money path in both directions.
func TestRefundAgainstGateway(t *testing.T) {
	key, privatePEM, publicPEM := testKeyPairPEM(t)

	t.Run("refund confirmed", func(t *testing.T) {
		var form url.Values
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = r.ParseForm()
			form = r.PostForm
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedGatewayResponse(t, key, "alipay_trade_refund_response", map[string]interface{}{
				"code": "10000", "msg": "Success", "trade_no": "20241001220014123456", "refund_fee": "5.00",
			})))
		}))
		defer srv.Close()

		client := newRealClient(t, srv, key, privatePEM, publicPEM)
		if err := client.Refund(context.Background(), "order-1", "refund-1", 500); err != nil {
			t.Fatalf("Refund failed: %v", err)
		}
		// 500 分必须渲染为 "5.00" 元。
		if !strings.Contains(form.Get("biz_content"), `"refund_amount":"5.00"`) {
			t.Errorf("biz_content = %q, want refund_amount 5.00", form.Get("biz_content"))
		}
	})

	t.Run("refund rejected", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedGatewayResponse(t, key, "alipay_trade_refund_response", map[string]interface{}{
				"code": "40004", "msg": "Business Failed", "sub_code": "ACQ.REFUND_AMT_NOT_EQUAL_TOTAL",
			})))
		}))
		defer srv.Close()

		client := newRealClient(t, srv, key, privatePEM, publicPEM)
		if err := client.Refund(context.Background(), "order-1", "refund-1", 500); err == nil {
			t.Error("a rejected refund must surface as an error")
		}
	})
}

// TestCloseOrderAndQueryRefundAgainstGateway covers the remaining maintenance
// calls used by the reconciliation sweeper.
func TestCloseOrderAndQueryRefundAgainstGateway(t *testing.T) {
	key, privatePEM, publicPEM := testKeyPairPEM(t)

	t.Run("close order", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedGatewayResponse(t, key, "alipay_trade_close_response", map[string]interface{}{
				"code": "10000", "msg": "Success", "trade_no": "20241001220014123456",
			})))
		}))
		defer srv.Close()

		client := newRealClient(t, srv, key, privatePEM, publicPEM)
		if err := client.CloseOrder(context.Background(), "order-1"); err != nil {
			t.Fatalf("CloseOrder failed: %v", err)
		}
	})

	t.Run("query refund", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(signedGatewayResponse(t, key, "alipay_trade_fastpay_refund_query_response", map[string]interface{}{
				"code": "10000", "msg": "Success", "refund_status": "REFUND_SUCCESS",
			})))
		}))
		defer srv.Close()

		client := newRealClient(t, srv, key, privatePEM, publicPEM)
		status, err := client.QueryRefund(context.Background(), "order-1", "refund-1")
		if err != nil {
			t.Fatalf("QueryRefund failed: %v", err)
		}
		if status != "REFUND_SUCCESS" {
			t.Errorf("status = %q, want REFUND_SUCCESS", status)
		}
	})
}

// TestCreateTradePreCreateAgainstGateway covers QR-code order creation: the
// wrapper must return the code_url the gateway produced.
func TestCreateTradePreCreateAgainstGateway(t *testing.T) {
	key, privatePEM, publicPEM := testKeyPairPEM(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(signedGatewayResponse(t, key, "alipay_trade_precreate_response", map[string]interface{}{
			"code": "10000", "msg": "Success", "out_trade_no": "order-1", "qr_code": "https://qr.alipay.com/bax00001",
		})))
	}))
	defer srv.Close()

	client := newRealClient(t, srv, key, privatePEM, publicPEM)
	qrCode, err := client.CreateTradePreCreate(context.Background(), "order-1", 1500, "停车费")
	if err != nil {
		t.Fatalf("CreateTradePreCreate failed: %v", err)
	}
	if qrCode != "https://qr.alipay.com/bax00001" {
		t.Errorf("qr_code = %q, want the gateway's code url", qrCode)
	}
}

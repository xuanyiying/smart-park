package alipay_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"net/url"
	"testing"

	"github.com/xuanyiying/smart-park/internal/payment/alipay"
)

// testKeyPair generates an RSA key and its PEM encoded public key, standing in for the
// merchant/AliPay key material without needing real credentials on disk.
func testKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}
	encoded := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})

	return key, string(encoded)
}

// newTestClient builds a Client whose gateway transport is unused; only the callback
// verification path is exercised.
func newTestClient(t *testing.T, publicKey string, signType string) *alipay.Client {
	t.Helper()

	cfg := &alipay.Config{
		AppID:           "2021000000000000",
		PrivateKey:      "unused-private-key",
		AlipayPublicKey: publicKey,
		NotifyURL:       "https://example.com/notify",
		IsProduction:    false,
		SignType:        signType,
	}

	client, err := alipay.NewTestClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

// signedParams builds a callback form and signs it with the given key.
func signedParams(t *testing.T, key *rsa.PrivateKey, signType string, values url.Values) url.Values {
	t.Helper()

	if values == nil {
		values = url.Values{}
	}
	values.Set("sign_type", signType)

	content := alipay.BuildSignContent(values)
	h := crypto.SHA256
	if signType == alipay.SignTypeRSA {
		h = crypto.SHA1
	}

	digest := h.New()
	digest.Write([]byte(content))

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, h, digest.Sum(nil))
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}
	values.Set("sign", base64.StdEncoding.EncodeToString(signature))
	return values
}

func TestNewClientValidation(t *testing.T) {
	_, publicKey := testKeyPair(t)

	tests := []struct {
		name    string
		config  *alipay.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &alipay.Config{
				AppID:           "2021000000000000",
				PrivateKey:      "private-key-material",
				AlipayPublicKey: publicKey,
				NotifyURL:       "https://example.com/notify",
				SignType:        alipay.SignTypeRSA2,
			},
			wantErr: false,
		},
		{
			name: "missing appid",
			config: &alipay.Config{
				PrivateKey:      "private-key-material",
				AlipayPublicKey: publicKey,
			},
			wantErr: true,
		},
		{
			name: "missing private key",
			config: &alipay.Config{
				AppID:           "2021000000000000",
				AlipayPublicKey: publicKey,
			},
			wantErr: true,
		},
		{
			name: "missing alipay public key",
			config: &alipay.Config{
				AppID:      "2021000000000000",
				PrivateKey: "private-key-material",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := alipay.NewTestClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTestClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestVerifyNotificationAcceptsValidSignature covers the happy path for both supported
// signature algorithms.
func TestVerifyNotificationAcceptsValidSignature(t *testing.T) {
	key, publicKey := testKeyPair(t)

	for _, signType := range []string{alipay.SignTypeRSA2, alipay.SignTypeRSA} {
		t.Run(signType, func(t *testing.T) {
			client := newTestClient(t, publicKey, signType)

			params := signedParams(t, key, signType, url.Values{
				"out_trade_no": {"order-1"},
				"trade_no":     {"2024040122001"},
				"trade_status": {"TRADE_SUCCESS"},
				"total_amount": {"15.00"},
				"gmt_payment":  {"2024-04-01 12:00:00"},
			})

			if err := client.VerifyNotification(params); err != nil {
				t.Fatalf("valid signature rejected: %v", err)
			}
		})
	}
}

// TestVerifyNotificationRejectsTampering is the security-critical case: Alipay signs every
// parameter, so changing any signed field (not just the handful the older implementation
// looked at) must invalidate the callback.
func TestVerifyNotificationRejectsTampering(t *testing.T) {
	key, publicKey := testKeyPair(t)
	client := newTestClient(t, publicKey, alipay.SignTypeRSA2)

	params := signedParams(t, key, alipay.SignTypeRSA2, url.Values{
		"out_trade_no": {"order-1"},
		"trade_no":     {"2024040122001"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"15.00"},
		"gmt_payment":  {"2024-04-01 12:00:00"},
	})

	// An attacker downgrading the amount must be detected. Note that total_amount is one
	// of the fields the previous implementation did verify; the important property below
	// is that *every* field is covered, not just a chosen few.
	tampered := cloneValues(params)
	tampered.Set("total_amount", "0.01")
	if err := client.VerifyNotification(tampered); err == nil {
		t.Error("tampered total_amount must be rejected")
	}

	tampered = cloneValues(params)
	tampered.Set("trade_status", "TRADE_CLOSED")
	if err := client.VerifyNotification(tampered); err == nil {
		t.Error("tampered trade_status must be rejected")
	}

	tampered = cloneValues(params)
	tampered.Set("seller_id", "attacker-owned-account")
	if err := client.VerifyNotification(tampered); err == nil {
		t.Error("an injected extra parameter must be rejected: every parameter is signed")
	}

	tampered = cloneValues(params)
	tampered.Set("sign", "AAAA")
	if err := client.VerifyNotification(tampered); err == nil {
		t.Error("garbage signature must be rejected")
	}

	tampered = cloneValues(params)
	tampered.Del("sign")
	if err := client.VerifyNotification(tampered); err == nil {
		t.Error("missing signature must be rejected")
	}
}

// TestVerifyNotificationRejectsForeignKey ensures only the configured Alipay public key
// is trusted, not any key that happens to produce a parseable signature.
func TestVerifyNotificationRejectsForeignKey(t *testing.T) {
	_, publicKey := testKeyPair(t)
	client := newTestClient(t, publicKey, alipay.SignTypeRSA2)

	attacker, _ := testKeyPair(t)
	params := signedParams(t, attacker, alipay.SignTypeRSA2, url.Values{
		"out_trade_no": {"order-1"},
		"total_amount": {"15.00"},
	})

	if err := client.VerifyNotification(params); err == nil {
		t.Error("callback signed by a foreign key must be rejected")
	}
}

// TestBuildSignContent pins the exact signing string: parameters sorted by key, joined
// with '&', with sign and sign_type excluded.
func TestBuildSignContent(t *testing.T) {
	params := url.Values{
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"15.00"},
		"out_trade_no": {"order-1"},
		"sign_type":    {"RSA2"},
		"sign":         {"should-be-excluded"},
	}

	got := alipay.BuildSignContent(params)
	want := "out_trade_no=order-1&total_amount=15.00&trade_status=TRADE_SUCCESS"
	if got != want {
		t.Errorf("BuildSignContent() = %q, want %q", got, want)
	}
}

// TestNormalizeSignType documents that an unset or unknown sign type falls back to RSA2,
// which is what modern Alipay apps use, and that the decision never depends on inspecting
// key material.
func TestNormalizeSignType(t *testing.T) {
	tests := map[string]string{
		"":      alipay.SignTypeRSA2,
		"RSA":   alipay.SignTypeRSA,
		"rsa":   alipay.SignTypeRSA,
		"RSA2":  alipay.SignTypeRSA2,
		"rsa2":  alipay.SignTypeRSA2,
		"bogus": alipay.SignTypeRSA2,
	}

	for input, want := range tests {
		if got := alipay.NormalizeSignType(input); got != want {
			t.Errorf("NormalizeSignType(%q) = %q, want %q", input, got, want)
		}
	}
}

func cloneValues(in url.Values) url.Values {
	out := url.Values{}
	for k, v := range in {
		out[k] = append([]string(nil), v...)
	}
	return out
}

func TestCreateTradePreCreate(t *testing.T) {
	t.Skip("需要真实的支付宝商户私钥和网关访问才能运行")
}

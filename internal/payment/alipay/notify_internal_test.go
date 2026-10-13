package alipay

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"net/url"
	"testing"

	alipaysdk "github.com/smartwalle/alipay/v3"
)

// signCallbackForm signs a callback form the way Alipay does: every parameter
// except sign/sign_type, sorted, joined, RSA-SHA256 (or SHA1) PKCS1v15.
func signCallbackForm(t *testing.T, key *rsa.PrivateKey, signType string, params url.Values) url.Values {
	t.Helper()

	signed := url.Values{}
	for k, v := range params {
		signed[k] = append([]string(nil), v...)
	}
	signed.Set("sign_type", signType)

	content := BuildSignContent(signed)
	h := crypto.SHA256
	if signType == SignTypeRSA {
		h = crypto.SHA1
	}
	digest := h.New()
	digest.Write([]byte(content))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, h, digest.Sum(nil))
	if err != nil {
		t.Fatalf("failed to sign callback: %v", err)
	}
	signed.Set("sign", base64.StdEncoding.EncodeToString(signature))
	return signed
}

// TestParsePublicKeyFormats covers both ways Alipay presents its public key in
// the merchant console: a PEM block and a bare base64 DER blob.
func TestParsePublicKeyFormats(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		t.Fatalf("failed to marshal public key: %v", err)
	}

	t.Run("pem", func(t *testing.T) {
		pub, err := parsePublicKey("-----BEGIN PUBLIC KEY-----\n" +
			base64.StdEncoding.EncodeToString(der) + "\n-----END PUBLIC KEY-----\n")
		if err != nil {
			t.Fatalf("parsePublicKey rejected a PEM key: %v", err)
		}
		if pub.N.Cmp(key.N) != 0 {
			t.Error("parsed key does not match the source key")
		}
	})

	t.Run("bare base64 der", func(t *testing.T) {
		pub, err := parsePublicKey(base64.StdEncoding.EncodeToString(der))
		if err != nil {
			t.Fatalf("parsePublicKey rejected a bare base64 key: %v", err)
		}
		if pub.N.Cmp(key.N) != 0 {
			t.Error("parsed key does not match the source key")
		}
	})

	t.Run("empty", func(t *testing.T) {
		if _, err := parsePublicKey("   "); err == nil {
			t.Error("an empty key must be rejected")
		}
	})

	t.Run("garbage", func(t *testing.T) {
		if _, err := parsePublicKey("definitely not a key"); err == nil {
			t.Error("garbage must be rejected")
		}
	})
}

// TestHashForSignType pins the algorithm mapping: it is driven by the explicit
// configuration value, never by inspecting the key material.
func TestHashForSignType(t *testing.T) {
	if got := hashForSignType(SignTypeRSA); got != crypto.SHA1 {
		t.Errorf("RSA must map to SHA1, got %v", got)
	}
	if got := hashForSignType(SignTypeRSA2); got != crypto.SHA256 {
		t.Errorf("RSA2 must map to SHA256, got %v", got)
	}
	if got := hashForSignType(""); got != crypto.SHA256 {
		t.Errorf("an unset sign type must default to SHA256, got %v", got)
	}
	if got := hashForSignType("bogus"); got != crypto.SHA256 {
		t.Errorf("an unknown sign type must default to SHA256, got %v", got)
	}
}

// TestVerifyNotificationMissingPublicKey documents that verification fails
// closed when no public key is configured.
func TestVerifyNotificationMissingPublicKey(t *testing.T) {
	client := &Client{config: &Config{}}
	if err := client.VerifyNotification(url.Values{"sign": {"x"}}); err == nil {
		t.Error("verification without a public key must fail")
	}
}

// TestParseNotificationRejectsUnsigned proves the convenience wrapper refuses
// to hand unverified parameters back to callers.
func TestParseNotificationRejectsUnsigned(t *testing.T) {
	client := &Client{config: &Config{AlipayPublicKey: "unused"}}
	if _, err := client.ParseNotification(url.Values{"out_trade_no": {"order-1"}}); err == nil {
		t.Error("parameters without a signature must be rejected")
	}
}

// TestParseAndDecodeNotificationSuccess drives the verified happy path: a
// correctly signed callback form is accepted and decoded into its fields.
func TestParseAndDecodeNotificationSuccess(t *testing.T) {
	key, privatePEM, publicPEM := testKeyPairPEM(t)

	sdk, err := alipaysdk.New("2021000000000000", privatePEM, false)
	if err != nil {
		t.Fatalf("failed to build sdk client: %v", err)
	}
	if err := sdk.LoadAliPayPublicKey(publicPEM); err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}
	client := &Client{client: sdk, config: &Config{
		AppID: "2021000000000000", PrivateKey: privatePEM, AlipayPublicKey: publicPEM,
		SignType: SignTypeRSA2,
	}}

	params := url.Values{
		"app_id":       {"2021000000000000"},
		"notify_id":    {"ac8f5b17e33b4ee49cf905efe49c9355"},
		"notify_type":  {"trade_status_sync"},
		"trade_no":     {"20241001220014123456"},
		"out_trade_no": {"order-1"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"15.00"},
	}
	signed := signCallbackForm(t, key, SignTypeRSA2, params)

	verified, err := client.ParseNotification(signed)
	if err != nil {
		t.Fatalf("ParseNotification rejected a validly signed callback: %v", err)
	}
	if verified.Get("out_trade_no") != "order-1" {
		t.Errorf("out_trade_no = %q, want order-1", verified.Get("out_trade_no"))
	}

	notification, err := client.DecodeNotification(context.Background(), signed)
	if err != nil {
		t.Fatalf("DecodeNotification failed: %v", err)
	}
	if notification.OutTradeNo != "order-1" {
		t.Errorf("notification out_trade_no = %q, want order-1", notification.OutTradeNo)
	}
	if notification.TradeStatus != alipaysdk.TradeStatusSuccess {
		t.Errorf("notification trade_status = %q, want TRADE_SUCCESS", notification.TradeStatus)
	}
	if notification.TotalAmount != "15.00" {
		t.Errorf("notification total_amount = %q, want 15.00", notification.TotalAmount)
	}
}

// TestDecodeNotificationRejectsTampering pins the security property on the
// decode path: a modified amount must invalidate the callback.
func TestDecodeNotificationRejectsTampering(t *testing.T) {
	key, privatePEM, publicPEM := testKeyPairPEM(t)

	sdk, err := alipaysdk.New("2021000000000000", privatePEM, false)
	if err != nil {
		t.Fatalf("failed to build sdk client: %v", err)
	}
	if err := sdk.LoadAliPayPublicKey(publicPEM); err != nil {
		t.Fatalf("failed to load public key: %v", err)
	}
	client := &Client{client: sdk, config: &Config{
		AppID: "2021000000000000", PrivateKey: privatePEM, AlipayPublicKey: publicPEM,
		SignType: SignTypeRSA2,
	}}

	params := url.Values{
		"app_id":       {"2021000000000000"},
		"out_trade_no": {"order-1"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"15.00"},
	}
	signed := signCallbackForm(t, key, SignTypeRSA2, params)

	// An attacker downgrading the amount must be caught by the signature check.
	signed.Set("total_amount", "0.01")
	if _, err := client.DecodeNotification(context.Background(), signed); err == nil {
		t.Error("a tampered callback must be rejected")
	}
}

// TestCentsToYuanString pins the money formatting used on every outbound
// gateway call: integer cents, two decimals, no floating point.
func TestCentsToYuanString(t *testing.T) {
	cases := map[int64]string{
		0:      "0.00",
		5:      "0.05",
		1500:   "15.00",
		12345:  "123.45",
		99:     "0.99",
		100:    "1.00",
		-550:   "-5.50",
		199999: "1999.99",
	}
	for cents, want := range cases {
		if got := centsToYuanString(cents); got != want {
			t.Errorf("centsToYuanString(%d) = %q, want %q", cents, got, want)
		}
	}
}

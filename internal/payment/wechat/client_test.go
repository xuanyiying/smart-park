package wechat_test

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

// testPlatform generates an RSA key and a self-signed certificate standing in for the
// WeChat Pay platform certificate, plus the matching PEM file on disk.
func testPlatform(t *testing.T) (*rsa.PrivateKey, *x509.Certificate, string) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "Tenpay.com Root CA"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
	}

	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	path := filepath.Join(t.TempDir(), "platform_cert.pem")
	encoded := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatalf("failed to write certificate: %v", err)
	}

	return key, cert, path
}

func testConfig(certPath string) *wechat.Config {
	return &wechat.Config{
		AppID:          "wx_appid",
		MchID:          "1900000001",
		APIKey:         "legacy-v2-key",
		CertSerialNo:   "5157F09EFDC096DE15EBE81A47057A72",
		PrivateKeyPath: "unused-for-validation",
		PublicKeyPath:  certPath,
		APIv3Key:       "0123456789abcdef0123456789abcdef", // exactly 32 chars
		NotifyURL:      "https://example.com/notify",
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*wechat.Config)
		wantErr bool
	}{
		{name: "valid config", mutate: func(c *wechat.Config) {}},
		{
			name:    "missing appid",
			mutate:  func(c *wechat.Config) { c.AppID = "" },
			wantErr: true,
		},
		{
			name:    "missing mchid",
			mutate:  func(c *wechat.Config) { c.MchID = "" },
			wantErr: true,
		},
		{
			name:    "missing api v3 key",
			mutate:  func(c *wechat.Config) { c.APIv3Key = "" },
			wantErr: true,
		},
		{
			name:    "api v3 key wrong length",
			mutate:  func(c *wechat.Config) { c.APIv3Key = "tooshort" },
			wantErr: true,
		},
		{
			name:    "missing platform certificate",
			mutate:  func(c *wechat.Config) { c.PublicKeyPath = "" },
			wantErr: true,
		},
		{
			name:    "missing cert serial",
			mutate:  func(c *wechat.Config) { c.CertSerialNo = "" },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, certPath := testPlatform(t)
			cfg := testConfig(certPath)
			tt.mutate(cfg)

			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestVerifyCallbackSignatureAcceptsValidSignature proves that a correctly signed APIv3
// notification is accepted, which is the behaviour production depends on.
func TestVerifyCallbackSignatureAcceptsValidSignature(t *testing.T) {
	key, cert, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, cert)

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "abc123"
	body := `{"id":"evt-1","resource":{"algorithm":"AEAD_AES_256_GCM"}}`

	message := fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, body)
	digest := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	if err := client.VerifyCallbackSignature(cfg.CertSerialNo, timestamp, nonce, body,
		base64.StdEncoding.EncodeToString(signature)); err != nil {
		t.Fatalf("valid signature was rejected: %v", err)
	}
}

// TestVerifyCallbackSignatureRejectsTampering is the security-critical counterpart: a
// modified body, a wrong certificate serial or a foreign key must all be refused.
func TestVerifyCallbackSignatureRejectsTampering(t *testing.T) {
	key, cert, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, cert)

	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "nonce-1"
	body := `{"id":"evt-1"}`

	sign := func(payload string) string {
		digest := sha256.Sum256([]byte(payload))
		sig, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
			Hash:       crypto.SHA256,
		})
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}
		return base64.StdEncoding.EncodeToString(sig)
	}

	validSignature := sign(fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, body))

	t.Run("tampered body", func(t *testing.T) {
		if err := client.VerifyCallbackSignature(cfg.CertSerialNo, timestamp, nonce, body+" ", validSignature); err == nil {
			t.Error("tampered body must be rejected")
		}
	})

	t.Run("wrong serial", func(t *testing.T) {
		if err := client.VerifyCallbackSignature("DEADBEEF", timestamp, nonce, body, validSignature); err == nil {
			t.Error("callback signed by an unknown certificate must be rejected")
		}
	})

	t.Run("foreign signing key", func(t *testing.T) {
		other, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			t.Fatalf("failed to generate key: %v", err)
		}
		digest := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, body)))
		sig, err := rsa.SignPSS(rand.Reader, other, crypto.SHA256, digest[:], &rsa.PSSOptions{
			SaltLength: rsa.PSSSaltLengthEqualsHash,
			Hash:       crypto.SHA256,
		})
		if err != nil {
			t.Fatalf("failed to sign: %v", err)
		}
		if err := client.VerifyCallbackSignature(cfg.CertSerialNo, timestamp, nonce, body,
			base64.StdEncoding.EncodeToString(sig)); err == nil {
			t.Error("callback signed by a foreign key must be rejected")
		}
	})
}

// TestDecryptResource round-trips a realistic encrypted APIv3 notification.
func TestDecryptResource(t *testing.T) {
	_, cert, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, cert)

	plaintext := `{"appid":"wx_appid","mchid":"1900000001","out_trade_no":"order-1","transaction_id":"4200001","trade_state":"SUCCESS","amount":{"total":1500,"currency":"CNY"}}`

	block, err := aes.NewCipher([]byte(cfg.APIv3Key))
	if err != nil {
		t.Fatalf("failed to build cipher: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("failed to build gcm: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		t.Fatalf("failed to read nonce: %v", err)
	}
	aad := "transaction"
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), []byte(aad))

	resource := wechat.Resource{
		Algorithm:      "AEAD_AES_256_GCM",
		Ciphertext:     base64.StdEncoding.EncodeToString(ciphertext),
		Nonce:          string(nonce),
		AssociatedData: aad,
	}

	decrypted, err := client.DecryptResource(resource)
	if err != nil {
		t.Fatalf("failed to decrypt resource: %v", err)
	}
	if decrypted.OutTradeNo != "order-1" {
		t.Errorf("out_trade_no = %q, want order-1", decrypted.OutTradeNo)
	}
	if decrypted.Amount.Total != 1500 {
		t.Errorf("amount.total = %d, want 1500", decrypted.Amount.Total)
	}
	if decrypted.TradeState != wechat.TradeStateSuccess {
		t.Errorf("trade_state = %q, want SUCCESS", decrypted.TradeState)
	}
}

// TestParseAndVerifyCallbackRejectsReplay guards the freshness window that stops an
// attacker from replaying a previously captured, validly signed notification.
func TestParseAndVerifyCallbackRejectsReplay(t *testing.T) {
	key, cert, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, cert)

	old := time.Now().Add(-30 * time.Minute)
	timestamp := fmt.Sprintf("%d", old.Unix())
	nonce := "nonce"
	body := `{"id":"evt","resource":{}}`

	digest := sha256.Sum256([]byte(fmt.Sprintf("%s\n%s\n%s\n", timestamp, nonce, body)))
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, digest[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		t.Fatalf("failed to sign: %v", err)
	}

	headers := map[string]string{
		"Wechatpay-Serial":    cfg.CertSerialNo,
		"Wechatpay-Timestamp": timestamp,
		"Wechatpay-Nonce":     nonce,
		"Wechatpay-Signature": base64.StdEncoding.EncodeToString(signature),
	}

	if _, err := client.ParseAndVerifyCallback(headers, []byte(body), time.Now()); err == nil {
		t.Error("a replayed callback outside the freshness window must be rejected")
	}
}

func TestParseAndVerifyCallbackRejectsMissingHeaders(t *testing.T) {
	_, cert, certPath := testPlatform(t)
	cfg := testConfig(certPath)
	client := wechat.NewClientWithCert(nil, cfg, nil, cert)

	if _, err := client.ParseAndVerifyCallback(map[string]string{}, []byte(`{}`), time.Now()); err == nil {
		t.Error("callback without Wechatpay-* headers must be rejected")
	}
}

func TestCreateNativePay(t *testing.T) {
	t.Skip("需要真实的微信支付商户证书和网络环境才能运行")
}

func TestCreateJSAPIPay(t *testing.T) {
	t.Skip("需要真实的微信支付商户证书和网络环境才能运行")
}

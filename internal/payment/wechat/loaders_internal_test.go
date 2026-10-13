package wechat

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writePEM(t *testing.T, name string, block *pem.Block) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, pem.EncodeToMemory(block), 0o600); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
	return path
}

// TestLoadPrivateKey covers both PEM encodings WeChat merchants use plus the
// failure modes that must surface at startup rather than at the first payment.
func TestLoadPrivateKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	t.Run("pkcs8", func(t *testing.T) {
		der, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := loadPrivateKey(writePEM(t, "pkcs8.pem", &pem.Block{Type: "PRIVATE KEY", Bytes: der}))
		if err != nil {
			t.Fatalf("loadPrivateKey rejected a valid PKCS8 key: %v", err)
		}
		if loaded.N.Cmp(key.N) != 0 {
			t.Error("loaded key does not match the source key")
		}
	})

	t.Run("pkcs1", func(t *testing.T) {
		loaded, err := loadPrivateKey(writePEM(t, "pkcs1.pem", &pem.Block{
			Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key),
		}))
		if err != nil {
			t.Fatalf("loadPrivateKey rejected a valid PKCS1 key: %v", err)
		}
		if loaded.N.Cmp(key.N) != 0 {
			t.Error("loaded key does not match the source key")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := loadPrivateKey(filepath.Join(t.TempDir(), "absent.pem")); err == nil {
			t.Error("a missing key file must fail")
		}
	})

	t.Run("not pem", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "garbage.pem")
		if err := os.WriteFile(path, []byte("definitely not pem"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadPrivateKey(path); err == nil {
			t.Error("a non-PEM file must fail")
		}
	})

	t.Run("unsupported key type", func(t *testing.T) {
		ecdsaKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		ecDer, err := x509.MarshalPKCS8PrivateKey(ecdsaKey)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := loadPrivateKey(writePEM(t, "ec.pem", &pem.Block{Type: "PRIVATE KEY", Bytes: ecDer})); err == nil {
			t.Error("a non-RSA key must be rejected, not silently coerced")
		}
	})
}

// TestLoadPublicKey covers the certificate path and the bare-public-key
// fallback operators use when the platform ships only a public key.
func TestLoadPublicKey(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	t.Run("certificate", func(t *testing.T) {
		template := x509.Certificate{
			SerialNumber: big.NewInt(42),
			Subject:      pkix.Name{CommonName: "Tenpay.com platform cert"},
			NotBefore:    time.Now().Add(-time.Hour),
			NotAfter:     time.Now().Add(24 * time.Hour),
		}
		der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
		if err != nil {
			t.Fatalf("failed to mint certificate: %v", err)
		}
		cert, err := loadPublicKey(writePEM(t, "cert.pem", &pem.Block{Type: "CERTIFICATE", Bytes: der}))
		if err != nil {
			t.Fatalf("loadPublicKey rejected a valid certificate: %v", err)
		}
		if cert.PublicKey == nil {
			t.Error("the loaded certificate must carry a public key")
		}
	})

	t.Run("bare public key", func(t *testing.T) {
		der, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
		if err != nil {
			t.Fatal(err)
		}
		cert, err := loadPublicKey(writePEM(t, "pub.pem", &pem.Block{Type: "PUBLIC KEY", Bytes: der}))
		if err != nil {
			t.Fatalf("loadPublicKey rejected a bare public key: %v", err)
		}
		pub, ok := cert.PublicKey.(*rsa.PublicKey)
		if !ok {
			t.Fatal("the fallback certificate must carry the RSA public key")
		}
		if pub.N.Cmp(key.N) != 0 {
			t.Error("loaded public key does not match the source key")
		}
	})

	t.Run("missing file", func(t *testing.T) {
		if _, err := loadPublicKey(filepath.Join(t.TempDir(), "absent.pem")); err == nil {
			t.Error("a missing certificate must fail")
		}
	})

	t.Run("garbage", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "garbage.pem")
		if err := os.WriteFile(path, []byte("-----BEGIN CERTIFICATE-----\nnope\n-----END CERTIFICATE-----\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := loadPublicKey(path); err == nil {
			t.Error("an unparseable certificate must fail")
		}
	})
}

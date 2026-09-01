package wechat

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
)

type Config struct {
	AppID          string
	MchID          string
	APIKey         string
	CertSerialNo   string
	PrivateKeyPath string
	// PublicKeyPath points to the WeChat Pay platform certificate (PEM CERTIFICATE block).
	// Required for verifying APIv3 callback signatures.
	PublicKeyPath string
	// APIv3Key is the 32 character APIv3 secret used to decrypt callback resources.
	APIv3Key  string
	NotifyURL string
}

func (c *Config) Validate() error {
	if c.MchID == "" {
		return fmt.Errorf("wechat: mch_id is required")
	}
	if c.AppID == "" {
		return fmt.Errorf("wechat: app_id is required")
	}
	if c.CertSerialNo == "" {
		return fmt.Errorf("wechat: cert_serial_no is required")
	}
	if c.PrivateKeyPath == "" {
		return fmt.Errorf("wechat: private_key_path is required")
	}
	if c.APIv3Key == "" {
		return fmt.Errorf("wechat: api_v3_key is required")
	}
	if len(c.APIv3Key) != 32 {
		return fmt.Errorf("wechat: api_v3_key must be exactly 32 characters, got %d", len(c.APIv3Key))
	}
	if c.PublicKeyPath == "" {
		return fmt.Errorf("wechat: public_key_path is required to verify APIv3 callbacks")
	}
	return nil
}

func NewClient(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	platformCert, err := loadPublicKey(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load wechat pay platform certificate: %w", err)
	}

	ctx := context.Background()

	client, err := core.NewClient(
		ctx,
		option.WithWechatPayAutoAuthCipher(
			cfg.MchID,
			cfg.CertSerialNo,
			privateKey,
			cfg.APIv3Key,
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat pay client: %w", err)
	}

	return NewClientWithCert(client, cfg, privateKey, platformCert), nil
}

// NewClientWithCert builds a Client around an already created core client. It exists so that
// tests can inject a stubbed transport without touching the filesystem.
func NewClientWithCert(client *core.Client, cfg *Config, privateKey *rsa.PrivateKey, platformCert *x509.Certificate) *Client {
	return &Client{
		client:       client,
		config:       cfg,
		privateKey:   privateKey,
		platformCert: platformCert,
	}
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("not an RSA private key")
		}
		return rsaKey, nil
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// loadPublicKey parses a WeChat Pay platform certificate from PEM. The file contains a
// CERTIFICATE block; we accept a bare PUBLIC KEY block as well for convenience.
func loadPublicKey(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	if cert, err := x509.ParseCertificate(block.Bytes); err == nil {
		return cert, nil
	}

	// Fall back to a bare public key, wrapping it into a synthetic certificate so that
	// callers can always use cert.PublicKey.
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse platform certificate or public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("wechat pay platform key is not an RSA public key")
	}
	return &x509.Certificate{PublicKey: rsaPub, PublicKeyAlgorithm: x509.RSA}, nil
}

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
	APIv3Key       string
	NotifyURL      string
}

func NewClient(cfg *Config) (*Client, error) {
	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
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

	return &Client{
		client:     client,
		config:     cfg,
		privateKey: privateKey,
	}, nil
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

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}

	return rsaKey, nil
}

func loadPublicKey(path string) (*x509.Certificate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert, nil
}

package alipay_test

import (
	"testing"

	"github.com/xuanyiying/smart-park/internal/payment/alipay"
)

func TestNewClient(t *testing.T) {
	cfg := &alipay.Config{
		AppID:           "test_appid",
		PrivateKey:      "test_private_key",
		AlipayPublicKey: "test_public_key",
		NotifyURL:       "https://example.com/notify",
		IsProduction:    false,
	}

	client, err := alipay.NewClient(cfg)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if client == nil {
		t.Error("client should not be nil")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *alipay.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &alipay.Config{
				AppID:           "valid_appid",
				PrivateKey:      "valid_private_key",
				AlipayPublicKey: "valid_public_key",
				NotifyURL:       "https://example.com/notify",
				IsProduction:    false,
			},
			wantErr: false,
		},
		{
			name: "missing appid",
			config: &alipay.Config{
				PrivateKey:      "valid_private_key",
				AlipayPublicKey: "valid_public_key",
				NotifyURL:       "https://example.com/notify",
				IsProduction:    false,
			},
			wantErr: true,
		},
		{
			name: "missing private key",
			config: &alipay.Config{
				AppID:           "valid_appid",
				AlipayPublicKey: "valid_public_key",
				NotifyURL:       "https://example.com/notify",
				IsProduction:    false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := alipay.NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateTradePreCreate(t *testing.T) {
	t.Skip("需要真实的支付宝证书和配置才能运行")
}

func TestVerifyNotification(t *testing.T) {
	t.Skip("需要真实的支付宝证书和配置才能运行")
}

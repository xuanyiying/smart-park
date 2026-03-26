package wechat_test

import (
	"testing"

	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

func TestNewClient(t *testing.T) {
	cfg := &wechat.Config{
		AppID:          "test_appid",
		MchID:          "test_mchid",
		APIKey:         "test_apikey",
		CertSerialNo:   "test_serial",
		PrivateKeyPath: "test_private.pem",
		APIv3Key:       "test_apiv3_key",
		NotifyURL:      "https://example.com/notify",
	}

	client, err := wechat.NewClient(cfg)
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
		config  *wechat.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &wechat.Config{
				AppID:          "valid_appid",
				MchID:          "valid_mchid",
				APIKey:         "valid_apikey",
				CertSerialNo:   "valid_serial",
				PrivateKeyPath: "valid_private.pem",
				APIv3Key:       "valid_apiv3_key",
				NotifyURL:      "https://example.com/notify",
			},
			wantErr: false,
		},
		{
			name: "missing appid",
			config: &wechat.Config{
				MchID:          "valid_mchid",
				APIKey:         "valid_apikey",
				CertSerialNo:   "valid_serial",
				PrivateKeyPath: "valid_private.pem",
				APIv3Key:       "valid_apiv3_key",
				NotifyURL:      "https://example.com/notify",
			},
			wantErr: true,
		},
		{
			name: "missing mchid",
			config: &wechat.Config{
				AppID:          "valid_appid",
				APIKey:         "valid_apikey",
				CertSerialNo:   "valid_serial",
				PrivateKeyPath: "valid_private.pem",
				APIv3Key:       "valid_apiv3_key",
				NotifyURL:      "https://example.com/notify",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := wechat.NewClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateNativePay(t *testing.T) {
	t.Skip("需要真实的微信支付证书和配置才能运行")
}

func TestCreateJSAPIPay(t *testing.T) {
	t.Skip("需要真实的微信支付证书和配置才能运行")
}

func TestNotifyHandler(t *testing.T) {
	t.Skip("需要真实的微信支付证书和配置才能运行")
}

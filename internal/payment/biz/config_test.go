package biz

import (
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/xuanyiying/smart-park/internal/conf"
	"github.com/xuanyiying/smart-park/pkg/config"
)

// TestPaymentStatusCanTransition pins the order state machine: only the
// transitions a payment lifecycle may take are allowed.
func TestPaymentStatusCanTransition(t *testing.T) {
	cases := []struct {
		from, to PaymentStatus
		want     bool
	}{
		{StatusPending, StatusPaid, true},
		{StatusPending, StatusFailed, true},
		{StatusPending, StatusRefunded, false}, // never refund an unpaid order
		{StatusPaid, StatusRefunded, true},
		{StatusPaid, StatusFailed, false},
		{StatusRefunded, StatusPaid, false},
		{StatusFailed, StatusPaid, false},
	}
	for _, tc := range cases {
		if got := tc.from.CanTransition(tc.to); got != tc.want {
			t.Errorf("%s -> %s = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

// TestIsValidPayMethod documents the accepted channels.
func TestIsValidPayMethod(t *testing.T) {
	if !IsValidPayMethod("wechat") || !IsValidPayMethod("alipay") {
		t.Error("wechat and alipay must be accepted")
	}
	for _, bad := range []string{"", "cash", "WECHAT", "unionpay"} {
		if IsValidPayMethod(bad) {
			t.Errorf("method %q must be rejected", bad)
		}
	}
}

// TestDefaultConfig pins the business defaults operators rely on.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.OrderExpiration != 30*time.Minute {
		t.Errorf("OrderExpiration = %s, want 30m", cfg.OrderExpiration)
	}
	if cfg.Currency != "CNY" {
		t.Errorf("Currency = %q, want CNY", cfg.Currency)
	}
	if cfg.DefaultDescription == "" {
		t.Error("DefaultDescription must not be empty")
	}
}

// TestPaymentConfigChannelGating proves WechatPayEnabled/AlipayEnabled require
// complete credentials: a half-configured deployment must not verify callbacks.
func TestPaymentConfigChannelGating(t *testing.T) {
	if (&PaymentConfig{}).WechatPayEnabled() {
		t.Error("an empty config must not enable wechat callbacks")
	}
	if (&PaymentConfig{WechatAPIv3Key: "k"}).WechatPayEnabled() {
		t.Error("a key without the platform certificate must not enable callbacks")
	}
	complete := &PaymentConfig{
		WechatAPIv3Key:         "0123456789abcdef0123456789abcdef",
		WechatPlatformCertPath: "/etc/keys/platform.pem",
		WechatCertSerialNo:     "5157F09EFDC096DE15EBE81A47057A72",
	}
	if !complete.WechatPayEnabled() {
		t.Error("complete credentials must enable wechat callbacks")
	}
	if complete.AlipayEnabled() {
		t.Error("wechat credentials must not enable alipay callbacks")
	}

	if !(&PaymentConfig{AlipayPublicKey: "key"}).AlipayEnabled() {
		t.Error("a configured alipay public key must enable alipay callbacks")
	}
	if (&PaymentConfig{}).AlipayEnabled() {
		t.Error("an empty config must not enable alipay callbacks")
	}
}

// TestNewPaymentConfigFromAppConfig covers the wire provider that maps the
// application configuration onto the payment gateway config.
func TestNewPaymentConfigFromAppConfig(t *testing.T) {
	cfg := &conf.Config{
		Wechat: config.WechatConfig{MchID: "1900000001", APIKey: "v2-key"},
		Alipay: config.AlipayConfig{PublicKey: "alipay-public-key"},
	}
	payment := NewPaymentConfig(cfg)
	if payment.WechatMchID != "1900000001" {
		t.Errorf("WechatMchID = %q, want 1900000001", payment.WechatMchID)
	}
	if payment.WechatKey != "v2-key" {
		t.Errorf("WechatKey = %q, want v2-key", payment.WechatKey)
	}
	if payment.AlipayPublicKey != "alipay-public-key" {
		t.Errorf("AlipayPublicKey = %q, want alipay-public-key", payment.AlipayPublicKey)
	}
}

// TestNewLogger covers the wire provider.
func TestNewLogger(t *testing.T) {
	if NewLogger(log.NewStdLogger(nopWriter{})) == nil {
		t.Error("NewLogger must return a helper")
	}
}

// TestAmountParsingHelpers covers the logging-only parsers and the strict
// yuan-to-cents conversion edge cases not reached elsewhere.
func TestAmountParsingHelpers(t *testing.T) {
	if got := parseAmount("1500"); got != 15 {
		t.Errorf("parseAmount(1500) = %v, want 15", got)
	}
	if got := parseAmount("abc"); got != 0 {
		t.Errorf("parseAmount(abc) = %v, want 0", got)
	}
	if got := parseAmountFloat("15.5"); got != 15.5 {
		t.Errorf("parseAmountFloat(15.5) = %v, want 15.5", got)
	}
	if got := parseAmountFloat("abc"); got != 0 {
		t.Errorf("parseAmountFloat(abc) = %v, want 0", got)
	}

	// Trailing zeros beyond cents are fine; non-zero sub-cents are not.
	if got, err := parseYuanToCents("1.500"); err != nil || got != 150 {
		t.Errorf("parseYuanToCents(1.500) = %d, %v; want 150, nil", got, err)
	}
	if _, err := parseYuanToCents("1.501"); err == nil {
		t.Error("sub-cent precision must be rejected")
	}
	if got, err := parseYuanToCents(".5"); err != nil || got != 50 {
		t.Errorf("parseYuanToCents(.5) = %d, %v; want 50, nil", got, err)
	}
	if _, err := parseYuanToCents("99999999999999999999.00"); err == nil {
		t.Error("an overflowing amount must be rejected")
	}

	if !allZero("000") {
		t.Error("allZero(000) must be true")
	}
	if allZero("001") {
		t.Error("allZero(001) must be false")
	}
	if !currentTime().Before(time.Now().Add(time.Second)) {
		t.Error("currentTime must return approximately now")
	}
}

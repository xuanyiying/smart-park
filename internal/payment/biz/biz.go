// Package biz provides business logic for the payment service.
package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/internal/conf"
)

// ProviderSet is the provider set for biz layer.
var ProviderSet = wire.NewSet(
	NewPaymentUseCase,
	NewPaymentConfig,
	NewLogger,
)

// NewLogger creates a new logger helper.
func NewLogger(logger log.Logger) *log.Helper {
	return log.NewHelper(logger)
}

// NewPaymentConfig creates payment configuration from app config.
func NewPaymentConfig(cfg *conf.Config) *PaymentConfig {
	return &PaymentConfig{
		WechatMchID:     cfg.Wechat.MchID,
		WechatKey:       cfg.Wechat.APIKey,
		AlipayPublicKey: cfg.Alipay.PublicKey,
	}
}

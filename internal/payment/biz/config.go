package biz

import "time"

// PaymentStatus represents the status of a payment order.
type PaymentStatus string

const (
	StatusPending  PaymentStatus = "pending"
	StatusPaid     PaymentStatus = "paid"
	StatusRefunded PaymentStatus = "refunded"
	StatusFailed   PaymentStatus = "failed"
)

// PayMethod represents the payment method.
type PayMethod string

const (
	MethodWechat PayMethod = "wechat"
	MethodAlipay PayMethod = "alipay"
)

// WechatCallbackStatus represents WeChat callback status.
type WechatCallbackStatus string

const (
	WechatStatusSuccess WechatCallbackStatus = "SUCCESS"
	WechatStatusFail    WechatCallbackStatus = "FAIL"
)

// AlipayCallbackStatus represents Alipay trade status.
type AlipayCallbackStatus string

const (
	AlipayStatusSuccess  AlipayCallbackStatus = "TRADE_SUCCESS"
	AlipayStatusFinished AlipayCallbackStatus = "TRADE_FINISHED"
)

// Config holds payment business logic configuration.
type Config struct {
	// Order expiration time
	OrderExpiration time.Duration

	// Default currency (CNY)
	Currency string

	// Default order description
	DefaultDescription string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		OrderExpiration:    30 * time.Minute,
		Currency:           "CNY",
		DefaultDescription: "停车费",
	}
}

// IsValidPayMethod checks if the payment method is valid.
func IsValidPayMethod(method string) bool {
	switch PayMethod(method) {
	case MethodWechat, MethodAlipay:
		return true
	default:
		return false
	}
}

// CanTransition checks if status transition is valid.
func (s PaymentStatus) CanTransition(target PaymentStatus) bool {
	switch s {
	case StatusPending:
		return target == StatusPaid || target == StatusFailed
	case StatusPaid:
		return target == StatusRefunded
	default:
		return false
	}
}

package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Order represents an order entity. All money fields are in cents (分); the
// integer representation is what the WeChat/Alipay SDKs exchange, so keeping it
// end-to-end removes the float rounding hazard from the money path.
type Order struct {
	ID                  uuid.UUID
	RecordID            uuid.UUID
	LotID               uuid.UUID
	VehicleID           *uuid.UUID
	PlateNumber         string
	Amount              int64
	DiscountAmount      int64
	FinalAmount         int64
	Status              string
	PayTime             *time.Time
	PayMethod           string
	TransactionID       string
	PaidAmount          int64
	RefundedAt          *time.Time
	RefundTransactionID string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// OrderRepo defines the repository interface for order operations.
type OrderRepo interface {
	GetOrder(ctx context.Context, orderID uuid.UUID) (*Order, error)
	GetOrderByRecordID(ctx context.Context, recordID uuid.UUID) (*Order, error)
	GetOrderByTransactionID(ctx context.Context, transactionID string) (*Order, error)
	GetOrdersByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*Order, error)
	CreateOrder(ctx context.Context, order *Order) error
	UpdateOrder(ctx context.Context, order *Order) error
	ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*Order, int64, error)
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error

	// MarkOrderPaid atomically transitions an order from pending to paid.
	//
	// It must be implemented as a conditional update (UPDATE ... WHERE status = 'pending')
	// rather than read-modify-write. Returning false means somebody else already settled
	// this order: a duplicate gateway callback, a concurrent replica, or an operator.
	// This is what makes callback processing idempotent across replicas and restarts.
	// paidAmount is in cents (分).
	MarkOrderPaid(ctx context.Context, orderID uuid.UUID, method, transactionID string, paidAmount int64, paidAt time.Time) (bool, error)

	// ListOrdersByStatus returns orders in the given status created before cutoff, up to
	// limit rows. It powers the reconciliation sweeper that closes stale pending orders
	// and confirms them against the payment gateway.
	ListOrdersByStatus(ctx context.Context, status string, cutoff time.Time, limit int) ([]*Order, error)

	// MarkOrderClosed atomically transitions an order from pending to failed. It backs the
	// expiry sweeper; a gateway callback racing the sweeper resolves through the status
	// predicate, so exactly one side wins.
	MarkOrderClosed(ctx context.Context, orderID uuid.UUID, closedAt time.Time) (bool, error)
}

// PaymentConfig holds payment gateway configuration for signature verification.
type PaymentConfig struct {
	WechatMchID string

	// WechatKey is the APIv2 key. It is retained only for legacy APIv2 merchants and is
	// NOT used to verify APIv3 callbacks; see WechatAPIv3Key and WechatPlatformCertPath.
	WechatKey string

	// WechatAPIv3Key is the 32 character APIv3 secret used to AES-256-GCM decrypt
	// callback resources. Required when WeChat payments are enabled.
	WechatAPIv3Key string

	// WechatPlatformCertPath points to the PEM encoded WeChat Pay platform certificate
	// used to verify APIv3 callback signatures. Required when WeChat payments are enabled.
	WechatPlatformCertPath string

	// WechatCertSerialNo is the serial number of the platform certificate above. Callbacks
	// signed by any other certificate are rejected.
	WechatCertSerialNo string

	AlipayPublicKey string

	// AlipaySignType selects the callback signature algorithm: "RSA" (SHA1) or
	// "RSA2" (SHA256). Defaults to RSA2 when empty. It must match the Alipay console.
	AlipaySignType string
}

// WechatPayEnabled reports whether the deployment is configured to accept real WeChat
// APIv3 callbacks. When it is false, callbacks must be rejected instead of trusted.
func (c *PaymentConfig) WechatPayEnabled() bool {
	return c != nil && c.WechatAPIv3Key != "" && c.WechatPlatformCertPath != "" && c.WechatCertSerialNo != ""
}

// AlipayEnabled reports whether the deployment can verify Alipay callbacks.
func (c *PaymentConfig) AlipayEnabled() bool {
	return c != nil && c.AlipayPublicKey != ""
}

package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Order represents an order entity.
type Order struct {
	ID                  uuid.UUID
	RecordID            uuid.UUID
	LotID               uuid.UUID
	VehicleID           *uuid.UUID
	PlateNumber         string
	Amount              float64
	DiscountAmount      float64
	FinalAmount         float64
	Status              string
	PayTime             *time.Time
	PayMethod           string
	TransactionID       string
	PaidAmount          float64
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
}

// PaymentConfig holds payment gateway configuration for signature verification.
type PaymentConfig struct {
	WechatMchID     string
	WechatKey       string
	AlipayPublicKey string
}

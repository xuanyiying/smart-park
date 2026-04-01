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
	AutoPayEnabled      bool
	AutoPayAttempts     int
	AutoPayStatus       string
	UserID              *uuid.UUID
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// OrderRepo defines the repository interface for order operations.
type OrderRepo interface {
	GetOrder(ctx context.Context, orderID uuid.UUID) (*Order, error)
	GetOrderByRecordID(ctx context.Context, recordID uuid.UUID) (*Order, error)
	GetOrderByTransactionID(ctx context.Context, transactionID string) (*Order, error)
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

// Reconciliation represents a reconciliation record entity.
type Reconciliation struct {
	ID              uuid.UUID
	Date            string
	PayMethod       string
	Status          string
	TotalOrders     int
	MatchedOrders   int
	ExceptionOrders int
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ReconciliationException represents a reconciliation exception entity.
type ReconciliationException struct {
	ID                uuid.UUID
	ReconciliationID  uuid.UUID
	OrderID           uuid.UUID
	PlatformOrderID   string
	SystemAmount      float64
	PlatformAmount    float64
	Status            string
	Reason            string
	Action            string
	Remark            string
	HandledAt         *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ReconciliationRepo defines the repository interface for reconciliation operations.
type ReconciliationRepo interface {
	CreateReconciliation(ctx context.Context, reconciliation *Reconciliation) error
	GetReconciliation(ctx context.Context, reconciliationID uuid.UUID) (*Reconciliation, error)
	UpdateReconciliation(ctx context.Context, reconciliation *Reconciliation) error
	ListReconciliations(ctx context.Context, startDate, endDate string, page, pageSize int) ([]*Reconciliation, int64, error)
}

// ReconciliationExceptionRepo defines the repository interface for reconciliation exception operations.
type ReconciliationExceptionRepo interface {
	CreateReconciliationException(ctx context.Context, exception *ReconciliationException) error
	GetReconciliationExceptions(ctx context.Context, reconciliationID uuid.UUID) ([]*ReconciliationException, error)
	UpdateReconciliationException(ctx context.Context, exception *ReconciliationException) error
	GetReconciliationException(ctx context.Context, exceptionID uuid.UUID) (*ReconciliationException, error)
}

// PlatformTransaction represents a transaction record from payment platform.
type PlatformTransaction struct {
	TransactionID string
	OrderID       string
	Amount        float64
	PayTime       time.Time
	Status        string
	PayMethod     string
}

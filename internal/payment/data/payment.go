// Package data provides data access layer for the payment service.
package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/ent"
	"github.com/xuanyiying/smart-park/ent/order"
	"github.com/xuanyiying/smart-park/internal/payment/biz"
)

// Data holds the data layer dependencies.
type Data struct {
	db  *ent.Client
	log *log.Helper
}

// NewData creates a new Data instance.
func NewData(db *ent.Client, logger log.Logger) (*Data, func(), error) {
	d := &Data{
		db:  db,
		log: log.NewHelper(logger),
	}

	cleanup := func() {
		if err := d.db.Close(); err != nil {
			d.log.Errorf("failed to close database: %v", err)
		}
	}

	return d, cleanup, nil
}

// orderRepo implements biz.OrderRepo.
type orderRepo struct {
	data *Data
}

// NewOrderRepo creates a new OrderRepo.
func NewOrderRepo(data *Data) biz.OrderRepo {
	return &orderRepo{data: data}
}

// GetOrder retrieves an order by ID.
func (r *orderRepo) GetOrder(ctx context.Context, orderID uuid.UUID) (*biz.Order, error) {
	o, err := r.data.db.Order.Get(ctx, orderID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toBizOrder(o), nil
}

// GetOrderByRecordID retrieves an order by record ID.
func (r *orderRepo) GetOrderByRecordID(ctx context.Context, recordID uuid.UUID) (*biz.Order, error) {
	o, err := r.data.db.Order.Query().
		Where(order.RecordID(recordID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toBizOrder(o), nil
}

// GetOrderByTransactionID retrieves an order by transaction ID.
func (r *orderRepo) GetOrderByTransactionID(ctx context.Context, transactionID string) (*biz.Order, error) {
	o, err := r.data.db.Order.Query().
		Where(order.TransactionID(transactionID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toBizOrder(o), nil
}

// CreateOrder creates a new order.
func (r *orderRepo) CreateOrder(ctx context.Context, o *biz.Order) error {
	create := r.data.db.Order.Create().
		SetID(o.ID).
		SetRecordID(o.RecordID).
		SetLotID(o.LotID).
		SetPlateNumber(o.PlateNumber).
		SetAmount(o.Amount).
		SetDiscountAmount(o.DiscountAmount).
		SetFinalAmount(o.FinalAmount).
		SetStatus(order.StatusPending)

	if o.VehicleID != nil {
		create.SetVehicleID(*o.VehicleID)
	}

	_, err := create.Save(ctx)
	return err
}

// UpdateOrder updates an order.
func (r *orderRepo) UpdateOrder(ctx context.Context, o *biz.Order) error {
	update := r.data.db.Order.UpdateOneID(o.ID)

	switch o.Status {
	case "pending":
		update.SetStatus(order.StatusPending)
	case "paid":
		update.SetStatus(order.StatusPaid)
	case "refunding":
		update.SetStatus(order.StatusRefunding)
	case "refunded":
		update.SetStatus(order.StatusRefunded)
	case "failed":
		update.SetStatus(order.StatusFailed)
	}

	if o.PayTime != nil {
		update.SetPayTime(*o.PayTime)
	}
	if o.PayMethod != "" {
		switch o.PayMethod {
		case "wechat":
			update.SetPayMethod(order.PayMethodWechat)
		case "alipay":
			update.SetPayMethod(order.PayMethodAlipay)
		case "cash":
			update.SetPayMethod(order.PayMethodCash)
		}
	}
	if o.TransactionID != "" {
		update.SetTransactionID(o.TransactionID)
	}
	if o.PaidAmount > 0 {
		update.SetPaidAmount(o.PaidAmount)
	}
	if o.RefundedAt != nil {
		update.SetRefundedAt(*o.RefundedAt)
	}
	if o.RefundTransactionID != "" {
		update.SetRefundTransactionID(o.RefundTransactionID)
	}

	_, err := update.Save(ctx)
	return err
}

// ListOrders lists orders with pagination.
func (r *orderRepo) ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*biz.Order, int64, error) {
	query := r.data.db.Order.Query().Where(order.LotID(lotID))

	if status != "" {
		switch status {
		case "pending":
			query = query.Where(order.Status(order.StatusPending))
		case "paid":
			query = query.Where(order.Status(order.StatusPaid))
		case "refunding":
			query = query.Where(order.Status(order.StatusRefunding))
		case "refunded":
			query = query.Where(order.Status(order.StatusRefunded))
		case "failed":
			query = query.Where(order.Status(order.StatusFailed))
		}
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	orders, err := query.
		Order(order.Desc(order.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*biz.Order
	for _, o := range orders {
		result = append(result, toBizOrder(o))
	}

	return result, int64(total), nil
}

// Helper function to convert ent Order to biz Order.
func toBizOrder(o *ent.Order) *biz.Order {
	var payMethod string
	if o.PayMethod != nil {
		payMethod = string(*o.PayMethod)
	}

	return &biz.Order{
		ID:                  o.ID,
		RecordID:            o.RecordID,
		LotID:               o.LotID,
		VehicleID:           o.VehicleID,
		PlateNumber:         o.PlateNumber,
		Amount:              o.Amount,
		DiscountAmount:      o.DiscountAmount,
		FinalAmount:         o.FinalAmount,
		Status:              string(o.Status),
		PayTime:             o.PayTime,
		PayMethod:           payMethod,
		TransactionID:       o.TransactionID,
		PaidAmount:          o.PaidAmount,
		RefundedAt:          o.RefundedAt,
		RefundTransactionID: o.RefundTransactionID,
		CreatedAt:           o.CreatedAt,
		UpdatedAt:           o.UpdatedAt,
	}
}

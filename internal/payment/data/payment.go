// Package data provides data access layer for the payment service.
package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/payment/biz"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent/order"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent/predicate"
)

type orderRepo struct {
	data *Data
}

func NewOrderRepo(data *Data) biz.OrderRepo {
	return &orderRepo{data: data}
}

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

func (r *orderRepo) CreateOrder(ctx context.Context, o *biz.Order) error {
	_, err := r.data.db.Order.Create().
		SetID(o.ID).
		SetRecordID(o.RecordID).
		SetLotID(o.LotID).
		SetPlateNumber(o.PlateNumber).
		SetAmount(o.Amount).
		SetDiscountAmount(o.DiscountAmount).
		SetFinalAmount(o.FinalAmount).
		SetStatus(order.StatusPending).
		Save(ctx)
	return err
}

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
		update.SetPayMethod(order.PayMethod(o.PayMethod))
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

func (r *orderRepo) ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*biz.Order, int64, error) {
	predicates := []predicate.Order{}
	if lotID != uuid.Nil {
		predicates = append(predicates, order.LotID(lotID))
	}
	if status != "" {
		var orderStatus order.Status
		switch status {
		case "pending":
			orderStatus = order.StatusPending
		case "paid":
			orderStatus = order.StatusPaid
		case "refunding":
			orderStatus = order.StatusRefunding
		case "refunded":
			orderStatus = order.StatusRefunded
		case "failed":
			orderStatus = order.StatusFailed
		}
		predicates = append(predicates, order.StatusEQ(orderStatus))
	}

	query := r.data.db.Order.Query().Where(predicates...)

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	orders, err := query.
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

func toBizOrder(o *ent.Order) *biz.Order {
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
		PayMethod:           string(o.PayMethod),
		TransactionID:       o.TransactionID,
		PaidAmount:          o.PaidAmount,
		RefundedAt:          o.RefundedAt,
		RefundTransactionID: o.RefundTransactionID,
		CreatedAt:           o.CreatedAt,
		UpdatedAt:           o.UpdatedAt,
	}
}

// Package data provides data access layer for the payment service.
package data

import (
	"context"
	"fmt"
	"time"

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

func (r *orderRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.data.txm.WithTx(ctx, fn)
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

// MarkOrderPaid performs a conditional update so that concurrent duplicate callbacks
// cannot both observe "pending" and then both write "paid".
//
// Ent's Update().Where(...) emits `UPDATE orders SET ... WHERE id = ? AND status = ?`,
// and Exec returns zero affected rows when the status predicate no longer matches.
func (r *orderRepo) MarkOrderPaid(ctx context.Context, orderID uuid.UUID, method, transactionID string, paidAmount int64, paidAt time.Time) (bool, error) {
	n, err := r.data.db.Order.Update().
		Where(
			order.ID(orderID),
			order.StatusEQ(order.StatusPending),
		).
		SetStatus(order.StatusPaid).
		SetPayMethod(order.PayMethod(method)).
		SetTransactionID(transactionID).
		SetPaidAmount(paidAmount).
		SetPayTime(paidAt).
		Save(ctx)
	if err != nil {
		// A unique constraint on transaction_id means the same gateway transaction was
		// already recorded for another order; treat it as "not newly paid".
		if ent.IsConstraintError(err) {
			return false, nil
		}
		return false, err
	}
	return n > 0, nil
}

// MarkOrderClosed performs a conditional update from pending to failed.
//
// Like MarkOrderPaid this must be a predicate update: a gateway callback for an order the
// sweeper just closed must lose the race cleanly instead of overwriting a closed order
// back to paid. (Callbacks settle through MarkOrderPaid, which matches on pending only.)
func (r *orderRepo) MarkOrderClosed(ctx context.Context, orderID uuid.UUID, closedAt time.Time) (bool, error) {
	n, err := r.data.db.Order.Update().
		Where(
			order.ID(orderID),
			order.StatusEQ(order.StatusPending),
		).
		SetStatus(order.StatusFailed).
		SetUpdatedAt(closedAt).
		Save(ctx)
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *orderRepo) ListOrdersByStatus(ctx context.Context, status string, cutoff time.Time, limit int) ([]*biz.Order, error) {
	statusValue, err := parseOrderStatus(status)
	if err != nil {
		return nil, err
	}

	query := r.data.db.Order.Query().
		Where(
			order.StatusEQ(statusValue),
			order.CreatedAtLTE(cutoff),
		).
		Order(ent.Asc(order.FieldCreatedAt))
	if limit > 0 {
		query = query.Limit(limit)
	}

	orders, err := query.All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.Order, 0, len(orders))
	for _, o := range orders {
		result = append(result, toBizOrder(o))
	}
	return result, nil
}

func parseOrderStatus(status string) (order.Status, error) {
	switch status {
	case "pending":
		return order.StatusPending, nil
	case "paid":
		return order.StatusPaid, nil
	case "refunding":
		return order.StatusRefunding, nil
	case "refunded":
		return order.StatusRefunded, nil
	case "failed":
		return order.StatusFailed, nil
	default:
		return "", fmt.Errorf("unknown order status: %s", status)
	}
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

func (r *orderRepo) GetOrdersByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*biz.Order, error) {
	orders, err := r.data.db.Order.Query().
		Where(
			order.PayTimeGTE(startTime),
			order.PayTimeLT(endTime),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*biz.Order
	for _, o := range orders {
		result = append(result, toBizOrder(o))
	}

	return result, nil
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

// Package data provides data access layer for the payment service.
package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/payment/biz"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent/order"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent/predicate"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent/reconciliation"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent/reconciliationexception"
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

type reconciliationRepo struct {
	data *Data
}

func NewReconciliationRepo(data *Data) biz.ReconciliationRepo {
	return &reconciliationRepo{data: data}
}

func (r *reconciliationRepo) CreateReconciliation(ctx context.Context, rec *biz.Reconciliation) error {
	_, err := r.data.db.Reconciliation.Create().
		SetID(rec.ID).
		SetDate(rec.Date).
		SetPayMethod(rec.PayMethod).
		SetStatus(reconciliation.Status(rec.Status)).
		SetTotalOrders(rec.TotalOrders).
		SetMatchedOrders(rec.MatchedOrders).
		SetExceptionOrders(rec.ExceptionOrders).
		Save(ctx)
	return err
}

func (r *reconciliationRepo) GetReconciliation(ctx context.Context, reconciliationID uuid.UUID) (*biz.Reconciliation, error) {
	rec, err := r.data.db.Reconciliation.Get(ctx, reconciliationID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toBizReconciliation(rec), nil
}

func (r *reconciliationRepo) UpdateReconciliation(ctx context.Context, rec *biz.Reconciliation) error {
	update := r.data.db.Reconciliation.UpdateOneID(rec.ID)

	update.SetStatus(reconciliation.Status(rec.Status))
	update.SetTotalOrders(rec.TotalOrders)
	update.SetMatchedOrders(rec.MatchedOrders)
	update.SetExceptionOrders(rec.ExceptionOrders)

	_, err := update.Save(ctx)
	return err
}

func (r *reconciliationRepo) ListReconciliations(ctx context.Context, startDate, endDate string, page, pageSize int) ([]*biz.Reconciliation, int64, error) {
	predicates := []predicate.Reconciliation{}

	query := r.data.db.Reconciliation.Query().Where(predicates...)

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	reconciliations, err := query.
		Offset(offset).
		Limit(pageSize).
		Order(ent.Desc(reconciliation.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*biz.Reconciliation
	for _, rec := range reconciliations {
		result = append(result, toBizReconciliation(rec))
	}

	return result, int64(total), nil
}

type reconciliationExceptionRepo struct {
	data *Data
}

func NewReconciliationExceptionRepo(data *Data) biz.ReconciliationExceptionRepo {
	return &reconciliationExceptionRepo{data: data}
}

func (r *reconciliationExceptionRepo) CreateReconciliationException(ctx context.Context, exception *biz.ReconciliationException) error {
	_, err := r.data.db.ReconciliationException.Create().
		SetID(exception.ID).
		SetReconciliationID(exception.ReconciliationID).
		SetOrderID(exception.OrderID).
		SetPlatformOrderID(exception.PlatformOrderID).
		SetSystemAmount(exception.SystemAmount).
		SetPlatformAmount(exception.PlatformAmount).
		SetStatus(reconciliationexception.Status(exception.Status)).
		SetReason(exception.Reason).
		SetAction(exception.Action).
		SetRemark(exception.Remark).
		Save(ctx)
	return err
}

func (r *reconciliationExceptionRepo) GetReconciliationExceptions(ctx context.Context, reconciliationID uuid.UUID) ([]*biz.ReconciliationException, error) {
	exceptions, err := r.data.db.ReconciliationException.Query().
		Where(reconciliationexception.ReconciliationID(reconciliationID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*biz.ReconciliationException
	for _, exception := range exceptions {
		result = append(result, toBizReconciliationException(exception))
	}

	return result, nil
}

func (r *reconciliationExceptionRepo) UpdateReconciliationException(ctx context.Context, exception *biz.ReconciliationException) error {
	update := r.data.db.ReconciliationException.UpdateOneID(exception.ID)

	update.SetStatus(reconciliationexception.Status(exception.Status))
	update.SetAction(exception.Action)
	update.SetRemark(exception.Remark)
	if exception.HandledAt != nil {
		update.SetHandledAt(*exception.HandledAt)
	}

	_, err := update.Save(ctx)
	return err
}

func (r *reconciliationExceptionRepo) GetReconciliationException(ctx context.Context, exceptionID uuid.UUID) (*biz.ReconciliationException, error) {
	exception, err := r.data.db.ReconciliationException.Get(ctx, exceptionID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toBizReconciliationException(exception), nil
}

func toBizReconciliation(rec *ent.Reconciliation) *biz.Reconciliation {
	return &biz.Reconciliation{
		ID:              rec.ID,
		Date:            rec.Date,
		PayMethod:       rec.PayMethod,
		Status:          string(rec.Status),
		TotalOrders:     rec.TotalOrders,
		MatchedOrders:   rec.MatchedOrders,
		ExceptionOrders: rec.ExceptionOrders,
		CreatedAt:       rec.CreatedAt,
		UpdatedAt:       rec.UpdatedAt,
	}
}

func toBizReconciliationException(exception *ent.ReconciliationException) *biz.ReconciliationException {
	return &biz.ReconciliationException{
		ID:                exception.ID,
		ReconciliationID:  exception.ReconciliationID,
		OrderID:           exception.OrderID,
		PlatformOrderID:   exception.PlatformOrderID,
		SystemAmount:      exception.SystemAmount,
		PlatformAmount:    exception.PlatformAmount,
		Status:            string(exception.Status),
		Reason:            exception.Reason,
		Action:            exception.Action,
		Remark:            exception.Remark,
		HandledAt:         exception.HandledAt,
		CreatedAt:         exception.CreatedAt,
		UpdatedAt:         exception.UpdatedAt,
	}
}

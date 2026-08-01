package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/payment/biz"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent/reconciliation"
)

type reconciliationRepo struct {
	data *Data
}

func NewReconciliationRepo(data *Data) biz.ReconciliationRepo {
	return &reconciliationRepo{data: data}
}

func (r *reconciliationRepo) CreateReconciliation(ctx context.Context, record *biz.ReconciliationRecord) error {
	create := r.data.db.Reconciliation.Create().
		SetID(record.ID).
		SetOrderAmount(record.OrderAmount).
		SetPaidAmount(record.PaidAmount).
		SetReconciliationTime(record.ReconciliationTime).
		SetStatus(reconciliation.Status(record.Status)).
		SetNotes(record.Notes)

	if record.OrderID != uuid.Nil {
		create.SetOrderID(record.OrderID)
	}
	if record.PaymentMethod != "" {
		create.SetPaymentMethod(record.PaymentMethod)
	}
	if record.TransactionID != "" {
		create.SetTransactionID(record.TransactionID)
	}

	_, err := create.Save(ctx)
	return err
}

func (r *reconciliationRepo) GetReconciliationByOrderID(ctx context.Context, orderID uuid.UUID) (*biz.ReconciliationRecord, error) {
	rec, err := r.data.db.Reconciliation.Query().
		Where(reconciliation.OrderID(orderID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toBizReconciliation(rec), nil
}

func (r *reconciliationRepo) GetReconciliationByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*biz.ReconciliationRecord, error) {
	records, err := r.data.db.Reconciliation.Query().
		Where(
			reconciliation.ReconciliationTimeGTE(startTime),
			reconciliation.ReconciliationTimeLT(endTime),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*biz.ReconciliationRecord
	for _, rec := range records {
		result = append(result, toBizReconciliation(rec))
	}
	return result, nil
}

func (r *reconciliationRepo) UpdateReconciliation(ctx context.Context, record *biz.ReconciliationRecord) error {
	update := r.data.db.Reconciliation.UpdateOneID(record.ID).
		SetStatus(reconciliation.Status(record.Status)).
		SetNotes(record.Notes)

	_, err := update.Save(ctx)
	return err
}

func (r *reconciliationRepo) GetMismatchedReconciliations(ctx context.Context, startTime, endTime time.Time) ([]*biz.ReconciliationRecord, error) {
	records, err := r.data.db.Reconciliation.Query().
		Where(
			reconciliation.ReconciliationTimeGTE(startTime),
			reconciliation.ReconciliationTimeLT(endTime),
			reconciliation.StatusEQ(reconciliation.StatusMismatch),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*biz.ReconciliationRecord
	for _, rec := range records {
		result = append(result, toBizReconciliation(rec))
	}
	return result, nil
}

func toBizReconciliation(rec *ent.Reconciliation) *biz.ReconciliationRecord {
	result := &biz.ReconciliationRecord{
		ID:                 rec.ID,
		PaymentMethod:      rec.PaymentMethod,
		OrderAmount:        rec.OrderAmount,
		PaidAmount:         rec.PaidAmount,
		TransactionID:      rec.TransactionID,
		ReconciliationTime: rec.ReconciliationTime,
		Status:             biz.ReconciliationStatus(rec.Status),
		Notes:              rec.Notes,
	}
	if rec.OrderID != nil {
		result.OrderID = *rec.OrderID
	}
	return result
}

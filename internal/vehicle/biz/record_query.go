// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// parseUUID parses a string into uuid.UUID.
func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

// RecordQueryUseCase handles parking record query business logic.
type RecordQueryUseCase struct {
	vehicleRepo VehicleRepo
}

// NewRecordQueryUseCase creates a new RecordQueryUseCase.
func NewRecordQueryUseCase(vehicleRepo VehicleRepo) *RecordQueryUseCase {
	return &RecordQueryUseCase{
		vehicleRepo: vehicleRepo,
	}
}

// ListParkingRecordsByPlates retrieves parking records by plate numbers.
func (uc *RecordQueryUseCase) ListParkingRecordsByPlates(ctx context.Context, plateNumbers []string, page, pageSize int) (*v1.ListParkingRecordsData, error) {
	if len(plateNumbers) == 0 {
		return &v1.ListParkingRecordsData{
			Records: []*v1.ParkingRecordInfo{},
			Total:   0,
		}, nil
	}

	records, total, err := uc.vehicleRepo.ListParkingRecordsByPlates(ctx, plateNumbers, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to list parking records: %w", err)
	}

	// Convert biz entities to proto messages
	recordInfos := make([]*v1.ParkingRecordInfo, len(records))
	for i, record := range records {
		recordInfos[i] = uc.toParkingRecordInfo(record)
	}

	return &v1.ListParkingRecordsData{
		Records: recordInfos,
		Total:   int32(total),
	}, nil
}

// GetParkingRecord retrieves a single parking record by ID.
func (uc *RecordQueryUseCase) GetParkingRecord(ctx context.Context, recordID string) (*v1.ParkingRecordInfo, error) {
	id, err := parseUUID(recordID)
	if err != nil {
		return nil, fmt.Errorf("invalid record ID: %w", err)
	}

	record, err := uc.vehicleRepo.GetParkingRecord(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get parking record: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("parking record not found: %s", recordID)
	}

	return uc.toParkingRecordInfo(record), nil
}

// validExitStatuses lists the exit_status enum values accepted from callers.
var validExitStatuses = map[string]struct{}{
	ExitStatusUnpaid: {},
	ExitStatusPaid:   {},
	"refunded":       {},
	"waived":         {},
}

// finalAmountFromMetadata extracts the server-side fee (in cents) that the exit
// flow stored in the record metadata. JSON round-trips may deliver it as
// float64; Go callers may have written int64.
func finalAmountFromMetadata(metadata map[string]interface{}) int64 {
	if metadata == nil {
		return 0
	}
	switch v := metadata["finalAmount"].(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	default:
		return 0
	}
}

// UpdateRecordStatus updates a parking record's exit payment status.
//
// The payment service calls this after an order settles so the exit barrier can
// release an already-paid driver without recalculating the fee.
func (uc *RecordQueryUseCase) UpdateRecordStatus(ctx context.Context, recordID, status string) error {
	id, err := parseUUID(recordID)
	if err != nil {
		return fmt.Errorf("invalid record ID: %w", err)
	}
	if _, ok := validExitStatuses[status]; !ok {
		return fmt.Errorf("invalid exit status: %q", status)
	}

	record, err := uc.vehicleRepo.GetParkingRecord(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get parking record: %w", err)
	}
	if record == nil {
		return fmt.Errorf("parking record not found: %s", recordID)
	}

	// 已是目标状态时直接返回，保证支付渠道重复通知下的幂等。
	if record.ExitStatus == status {
		return nil
	}

	record.ExitStatus = status
	return uc.vehicleRepo.UpdateParkingRecord(ctx, record)
}

// toParkingRecordInfo converts biz ParkingRecord to proto ParkingRecordInfo.
func (uc *RecordQueryUseCase) toParkingRecordInfo(record *ParkingRecord) *v1.ParkingRecordInfo {
	info := &v1.ParkingRecordInfo{
		RecordId:        record.ID.String(),
		LotId:           record.LotID.String(),
		EntryLaneId:     record.EntryLaneID.String(),
		EntryTime:       record.EntryTime.Format("2006-01-02T15:04:05Z07:00"),
		EntryImageUrl:   record.EntryImageURL,
		RecordStatus:    record.RecordStatus,
		ParkingDuration: int32(record.ParkingDuration),
		ExitStatus:      record.ExitStatus,
		PaymentLock:     int32(record.PaymentLock),
	}

	if record.VehicleID != nil {
		info.VehicleId = record.VehicleID.String()
	}
	if record.PlateNumber != nil {
		info.PlateNumber = *record.PlateNumber
	}
	if record.ExitTime != nil {
		info.ExitTime = record.ExitTime.Format("2006-01-02T15:04:05Z07:00")
	}
	if record.ExitImageURL != "" {
		info.ExitImageUrl = record.ExitImageURL
	}
	if record.ExitLaneID != nil {
		info.ExitLaneId = record.ExitLaneID.String()
	}
	if record.ExitDeviceID != "" {
		info.ExitDeviceId = record.ExitDeviceID
	}
	info.FinalAmount = finalAmountFromMetadata(record.Metadata)

	return info
}

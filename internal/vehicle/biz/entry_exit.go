// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
	"github.com/xuanyiying/smart-park/pkg/lock"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// EntryExitUseCase handles vehicle entry and exit business logic.
type EntryExitUseCase struct {
	vehicleRepo   VehicleRepo
	billingClient billing.Client
	mqttClient    mqtt.Client
	lockRepo      lock.LockRepo
	config        *Config
	log           *log.Helper
}

// NewEntryExitUseCase creates a new EntryExitUseCase.
func NewEntryExitUseCase(vehicleRepo VehicleRepo, billingClient billing.Client, mqttClient mqtt.Client, lockRepo lock.LockRepo, logger log.Logger) *EntryExitUseCase {
	return &EntryExitUseCase{
		vehicleRepo:   vehicleRepo,
		billingClient: billingClient,
		mqttClient:    mqttClient,
		lockRepo:      lockRepo,
		config:        DefaultConfig(),
		log:           log.NewHelper(logger),
	}
}

// Entry handles vehicle entry.
func (uc *EntryExitUseCase) Entry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryData, error) {
	uc.logEntryStart(req.DeviceId, req.PlateNumber, req.Confidence)

	if req.PlateNumber == "" {
		return nil, fmt.Errorf("plate number is required")
	}

	var result *v1.EntryData
	lockKey := lock.GenerateLockKey(LockTypeEntry, req.PlateNumber)

	if err := uc.withDistributedLock(ctx, lockKey, func() error {
		return uc.vehicleRepo.WithTx(ctx, func(ctx context.Context) error {
			var err error
			result, err = uc.processEntryTransaction(ctx, req)
			return err
		})
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// Exit handles vehicle exit.
func (uc *EntryExitUseCase) Exit(ctx context.Context, req *v1.ExitRequest) (*v1.ExitData, error) {
	uc.logExitStart(req.DeviceId, req.PlateNumber, req.Confidence)

	if req.PlateNumber == "" {
		return nil, fmt.Errorf("plate number is required")
	}

	var result *v1.ExitData
	lockKey := lock.GenerateLockKey(LockTypeExit, req.PlateNumber)

	if err := uc.withDistributedLock(ctx, lockKey, func() error {
		return uc.vehicleRepo.WithTx(ctx, func(ctx context.Context) error {
			var err error
			result, err = uc.processExitTransaction(ctx, req)
			return err
		})
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *EntryExitUseCase) processEntryTransaction(ctx context.Context, req *v1.EntryRequest) (*v1.EntryData, error) {
	lane, err := uc.vehicleRepo.GetLaneByDeviceCode(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get lane info: %w", err)
	}
	uc.log.WithContext(ctx).Infof("[ENTRY] Found lane - LaneID: %s, LotID: %s", lane.ID, lane.LotID)

	vehicle, err := uc.vehicleRepo.GetVehicleByPlate(ctx, req.PlateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle info: %w", err)
	}

	existingRecord, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing entry: %w", err)
	}
	if existingRecord != nil {
		uc.log.WithContext(ctx).Warnf("[ENTRY] Duplicate entry - PlateNumber: [REDACTED]")
		return &v1.EntryData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: uc.config.Messages.DuplicateEntry,
		}, nil
	}

	record := uc.createParkingRecord(req, lane, vehicle)
	if err := uc.vehicleRepo.CreateParkingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create parking record: %w", err)
	}

	return uc.buildEntryResponse(record, req.PlateNumber, vehicle), nil
}

func (uc *EntryExitUseCase) processExitTransaction(ctx context.Context, req *v1.ExitRequest) (*v1.ExitData, error) {
	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	lane, err := uc.vehicleRepo.GetLaneByDeviceCode(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get lane info: %w", err)
	}

	record, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry record: %w", err)
	}
	if record == nil {
		return &v1.ExitData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: uc.config.Messages.NoEntryRecord,
		}, nil
	}

	exitTime := time.Now()
	duration := int(exitTime.Sub(record.EntryTime).Seconds())

	// Calculate fee first, before updating the record
	vehicle, vehicleType := uc.getVehicleInfo(ctx, req.PlateNumber)
	amount, discountAmount, finalAmount, err := uc.calculateExitFee(ctx, record, lane, exitTime, vehicle, vehicleType)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate fee: %w", err)
	}

	// Only update the record after fee calculation succeeds
	if err := uc.updateParkingRecordForExit(ctx, record, req, device, lane, exitTime, duration); err != nil {
		return nil, err
	}

	return uc.buildExitResponse(record, req, duration, amount, discountAmount, finalAmount), nil
}

func (uc *EntryExitUseCase) withDistributedLock(ctx context.Context, lockKey string, fn func() error) error {
	owner := lock.GenerateUniqueOwner()
	uc.log.WithContext(ctx).Debugf("[LOCK] Acquiring lock - Key: %s, Owner: %s", lockKey, owner)

	acquired, err := uc.lockRepo.AcquireLock(ctx, lockKey, owner, uc.config.LockTTL)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[LOCK] Failed to acquire lock: %v", err)
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		uc.log.WithContext(ctx).Warnf("[LOCK] Lock held by another process - Key: %s", lockKey)
		return fmt.Errorf("duplicate request in progress")
	}

	defer func() {
		if err := uc.lockRepo.ReleaseLock(ctx, lockKey, owner); err != nil {
			uc.log.WithContext(ctx).Warnf("[LOCK] Failed to release lock: %v", err)
		}
	}()

	return fn()
}

func (uc *EntryExitUseCase) createParkingRecord(req *v1.EntryRequest, lane *Lane, vehicle *Vehicle) *ParkingRecord {
	plateNumber := req.PlateNumber
	record := &ParkingRecord{
		ID:                uuid.New(),
		LotID:             lane.LotID,
		EntryLaneID:       lane.ID,
		EntryTime:         time.Now(),
		EntryImageURL:     req.PlateImageUrl,
		RecordStatus:      RecordStatusEntry,
		ExitStatus:        ExitStatusUnpaid,
		PlateNumber:       &plateNumber,
		PlateNumberSource: "camera",
	}

	if vehicle != nil {
		record.VehicleID = &vehicle.ID
	}

	return record
}

func (uc *EntryExitUseCase) buildEntryResponse(record *ParkingRecord, plateNumber string, vehicle *Vehicle) *v1.EntryData {
	displayMessage := uc.config.Messages.Welcome
	if vehicle != nil {
		switch vehicle.VehicleType {
		case VehicleTypeMonthly:
			displayMessage = uc.config.Messages.MonthlyWelcome
		case VehicleTypeVIP:
			displayMessage = uc.config.Messages.VIPWelcome
		}
	}

	return &v1.EntryData{
		RecordId:       record.ID.String(),
		PlateNumber:    plateNumber,
		Allowed:        true,
		GateOpen:       true,
		DisplayMessage: displayMessage,
	}
}

func (uc *EntryExitUseCase) updateParkingRecordForExit(ctx context.Context, record *ParkingRecord, req *v1.ExitRequest, device *Device, lane *Lane, exitTime time.Time, duration int) error {
	record.ExitTime = &exitTime
	record.ExitImageURL = req.PlateImageUrl
	record.ExitLaneID = &lane.ID
	record.ExitDeviceID = device.DeviceID
	record.RecordStatus = RecordStatusExiting
	record.ParkingDuration = duration

	if err := uc.vehicleRepo.UpdateParkingRecord(ctx, record); err != nil {
		uc.log.WithContext(ctx).Errorf("[EXIT] Failed to update parking record %s: %v", record.ID, err)
		return fmt.Errorf("failed to update parking record: %w", err)
	}
	return nil
}

func (uc *EntryExitUseCase) getVehicleInfo(ctx context.Context, plateNumber string) (*Vehicle, string) {
	vehicleType := VehicleTypeTemporary
	vehicle, err := uc.vehicleRepo.GetVehicleByPlate(ctx, plateNumber)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[EXIT] Failed to get vehicle info: %v, using default type", err)
		return nil, vehicleType
	}
	if vehicle != nil {
		vehicleType = vehicle.VehicleType
	}
	return vehicle, vehicleType
}

func (uc *EntryExitUseCase) calculateExitFee(ctx context.Context, record *ParkingRecord, lane *Lane, exitTime time.Time, vehicle *Vehicle, vehicleType string) (float64, float64, float64, error) {
	feeResult, err := uc.billingClient.CalculateFee(ctx, record.ID.String(), lane.LotID.String(),
		record.EntryTime.Unix(), exitTime.Unix(), vehicleType)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[EXIT] Failed to calculate fee: %v", err)
		return 0, 0, 0, fmt.Errorf("fee calculation failed: %w", err)
	}

	finalAmount := feeResult.FinalAmount

	if vehicle != nil && vehicle.VehicleType == VehicleTypeMonthly {
		if vehicle.MonthlyValidUntil != nil && vehicle.MonthlyValidUntil.After(time.Now()) {
			finalAmount = 0
			uc.log.WithContext(ctx).Infof("[EXIT] Monthly vehicle with valid card - PlateNumber: [REDACTED], ValidUntil: %s",
				vehicle.MonthlyValidUntil.Format(time.RFC3339))
		} else {
			if record.Metadata == nil {
				record.Metadata = make(map[string]interface{})
			}
			record.Metadata["chargeAs"] = VehicleTypeTemporary
			record.Metadata["monthlyExpired"] = true
			if vehicle.MonthlyValidUntil != nil {
				record.Metadata["expiredAt"] = vehicle.MonthlyValidUntil.Format(time.RFC3339)
			}
			uc.log.WithContext(ctx).Warnf("[EXIT] Monthly card expired, charging as temporary - PlateNumber: [REDACTED]")
		}
	}

	return feeResult.BaseAmount, feeResult.DiscountAmount, finalAmount, nil
}

func (uc *EntryExitUseCase) buildExitResponse(record *ParkingRecord, req *v1.ExitRequest, duration int, amount, discountAmount, finalAmount float64) *v1.ExitData {
	allowed := finalAmount == 0
	gateOpen := finalAmount == 0
	displayMessage := uc.config.Messages.PleasePay

	if finalAmount == 0 {
		displayMessage = uc.config.Messages.FreePass
		gateOpen = true
	}

	return &v1.ExitData{
		RecordId:        record.ID.String(),
		PlateNumber:     req.PlateNumber,
		ParkingDuration: int32(duration),
		Amount:          amount,
		DiscountAmount:  discountAmount,
		FinalAmount:     finalAmount,
		Allowed:         allowed,
		GateOpen:        gateOpen,
		DisplayMessage:  displayMessage,
	}
}

func (uc *EntryExitUseCase) logEntryStart(deviceID, plateNumber string, confidence float64) {
	// Plate number redacted for privacy
	uc.log.WithContext(context.Background()).Infof("[ENTRY] Processing entry - DeviceID: %s, PlateNumber: [REDACTED], Confidence: %.2f",
		deviceID, confidence)
}

func (uc *EntryExitUseCase) logExitStart(deviceID, plateNumber string, confidence float64) {
	// Plate number redacted for privacy
	uc.log.WithContext(context.Background()).Infof("[EXIT] Processing exit - DeviceID: %s, PlateNumber: [REDACTED], Confidence: %.2f",
		deviceID, confidence)
}

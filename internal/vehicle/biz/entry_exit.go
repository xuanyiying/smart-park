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
	"github.com/xuanyiying/smart-park/internal/vehicle/device"
	"github.com/xuanyiying/smart-park/pkg/lock"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// Error types for entry/exit control
const (
	ErrTypeValidation   = "VALIDATION_ERROR"
	ErrTypeDatabase     = "DATABASE_ERROR"
	ErrTypeDevice       = "DEVICE_ERROR"
	ErrTypeBilling      = "BILLING_ERROR"
	ErrTypeLock         = "LOCK_ERROR"
	ErrTypeBusiness     = "BUSINESS_ERROR"
	ErrTypeSystem       = "SYSTEM_ERROR"
)

// EntryExitError represents an error in entry/exit processing
type EntryExitError struct {
	Type    string
	Message string
	Err     error
}

func (e *EntryExitError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

func (e *EntryExitError) Unwrap() error {
	return e.Err
}

// EntryExitUseCase handles vehicle entry and exit business logic.
type EntryExitUseCase struct {
	vehicleRepo    VehicleRepo
	billingClient  billing.Client
	mqttClient     mqtt.Client
	lockRepo       lock.LockRepo
	adapterFactory *device.AdapterFactory
	config         *Config
	log            *log.Helper
}

// NewEntryExitUseCase creates a new EntryExitUseCase.
func NewEntryExitUseCase(vehicleRepo VehicleRepo, billingClient billing.Client, mqttClient mqtt.Client, lockRepo lock.LockRepo, adapterFactory *device.AdapterFactory, logger log.Logger) *EntryExitUseCase {
	return &EntryExitUseCase{
		vehicleRepo:    vehicleRepo,
		billingClient:  billingClient,
		mqttClient:     mqttClient,
		lockRepo:       lockRepo,
		adapterFactory: adapterFactory,
		config:         DefaultConfig(),
		log:            log.NewHelper(logger),
	}
}

// Entry handles vehicle entry with enhanced exception handling.
func (uc *EntryExitUseCase) Entry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryData, error) {
	uc.logEntryStart(req.DeviceId, req.PlateNumber, req.Confidence)

	// Validate request
	if err := uc.validateEntryRequest(req); err != nil {
		uc.log.WithContext(ctx).Errorf("[ENTRY] Validation failed: %v", err)
		return uc.handleEntryError(ctx, req, err)
	}

	var result *v1.EntryData
	lockKey := lock.GenerateLockKey(LockTypeEntry, req.PlateNumber)

	if err := uc.withDistributedLock(ctx, lockKey, func() error {
		return uc.vehicleRepo.WithTx(ctx, func(ctx context.Context) error {
			var err error
			result, err = uc.processEntryTransaction(ctx, req)
			if err != nil {
				uc.log.WithContext(ctx).Errorf("[ENTRY] Transaction failed: %v", err)
				return err
			}
			return nil
		})
	}); err != nil {
		uc.log.WithContext(ctx).Errorf("[ENTRY] Processing failed: %v", err)
		return uc.handleEntryError(ctx, req, err)
	}

	// Send device command with retry
	if err := uc.sendDeviceCommandWithRetry(ctx, req.DeviceId, "open_gate", 3); err != nil {
		uc.log.WithContext(ctx).Warnf("[ENTRY] Failed to send gate open command: %v", err)
		// Continue processing even if device command fails
	}

	return result, nil
}

// Exit handles vehicle exit with enhanced exception handling.
func (uc *EntryExitUseCase) Exit(ctx context.Context, req *v1.ExitRequest) (*v1.ExitData, error) {
	uc.logExitStart(req.DeviceId, req.PlateNumber, req.Confidence)

	// Validate request
	if err := uc.validateExitRequest(req); err != nil {
		uc.log.WithContext(ctx).Errorf("[EXIT] Validation failed: %v", err)
		return uc.handleExitError(ctx, req, err)
	}

	var result *v1.ExitData
	lockKey := lock.GenerateLockKey(LockTypeExit, req.PlateNumber)

	if err := uc.withDistributedLock(ctx, lockKey, func() error {
		return uc.vehicleRepo.WithTx(ctx, func(ctx context.Context) error {
			var err error
			result, err = uc.processExitTransaction(ctx, req)
			if err != nil {
				uc.log.WithContext(ctx).Errorf("[EXIT] Transaction failed: %v", err)
				return err
			}
			return nil
		})
	}); err != nil {
		uc.log.WithContext(ctx).Errorf("[EXIT] Processing failed: %v", err)
		return uc.handleExitError(ctx, req, err)
	}

	// Send device command with retry only if gate should open
	if result.GateOpen {
		if err := uc.sendDeviceCommandWithRetry(ctx, req.DeviceId, "open_gate", 3); err != nil {
			uc.log.WithContext(ctx).Warnf("[EXIT] Failed to send gate open command: %v", err)
			// Continue processing even if device command fails
		}
	}

	return result, nil
}

// validateEntryRequest validates entry request
func (uc *EntryExitUseCase) validateEntryRequest(req *v1.EntryRequest) error {
	if req.PlateNumber == "" {
		return &EntryExitError{Type: ErrTypeValidation, Message: "plate number is required"}
	}
	if req.DeviceId == "" {
		return &EntryExitError{Type: ErrTypeValidation, Message: "device ID is required"}
	}
	if req.Confidence < uc.config.MinConfidence {
		return &EntryExitError{Type: ErrTypeValidation, Message: "plate recognition confidence too low"}
	}
	return nil
}

// validateExitRequest validates exit request
func (uc *EntryExitUseCase) validateExitRequest(req *v1.ExitRequest) error {
	if req.PlateNumber == "" {
		return &EntryExitError{Type: ErrTypeValidation, Message: "plate number is required"}
	}
	if req.DeviceId == "" {
		return &EntryExitError{Type: ErrTypeValidation, Message: "device ID is required"}
	}
	if req.Confidence < uc.config.MinConfidence {
		return &EntryExitError{Type: ErrTypeValidation, Message: "plate recognition confidence too low"}
	}
	return nil
}

// handleEntryError handles entry errors with fallback mechanisms
func (uc *EntryExitUseCase) handleEntryError(ctx context.Context, req *v1.EntryRequest, err error) (*v1.EntryData, error) {
	switch e := err.(type) {
	case *EntryExitError:
		switch e.Type {
		case ErrTypeValidation:
			return &v1.EntryData{
				PlateNumber:    req.PlateNumber,
				Allowed:        false,
				GateOpen:       false,
				DisplayMessage: uc.config.Messages.ValidationError,
			}, nil
		case ErrTypeLock:
			return &v1.EntryData{
				PlateNumber:    req.PlateNumber,
				Allowed:        false,
				GateOpen:       false,
				DisplayMessage: uc.config.Messages.DuplicateEntry,
			}, nil
		case ErrTypeDatabase:
			// Database error - use fallback mechanism
			uc.log.WithContext(ctx).Warnf("[ENTRY] Database error, using fallback: %v", err)
			return uc.createFallbackEntryResponse(req), nil
		default:
			return &v1.EntryData{
				PlateNumber:    req.PlateNumber,
				Allowed:        false,
				GateOpen:       false,
				DisplayMessage: uc.config.Messages.SystemError,
			}, nil
		}
	default:
		// Unknown error type
		uc.log.WithContext(ctx).Errorf("[ENTRY] Unknown error: %v", err)
		return &v1.EntryData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: uc.config.Messages.SystemError,
		}, nil
	}
}

// handleExitError handles exit errors with fallback mechanisms
func (uc *EntryExitUseCase) handleExitError(ctx context.Context, req *v1.ExitRequest, err error) (*v1.ExitData, error) {
	switch e := err.(type) {
	case *EntryExitError:
		switch e.Type {
		case ErrTypeValidation:
			return &v1.ExitData{
				PlateNumber:    req.PlateNumber,
				Allowed:        false,
				GateOpen:       false,
				DisplayMessage: uc.config.Messages.ValidationError,
			}, nil
		case ErrTypeLock:
			return &v1.ExitData{
				PlateNumber:    req.PlateNumber,
				Allowed:        false,
				GateOpen:       false,
				DisplayMessage: uc.config.Messages.DuplicateExit,
			}, nil
		case ErrTypeBilling:
			// Billing error - use fallback for fee calculation
			uc.log.WithContext(ctx).Warnf("[EXIT] Billing error, using fallback: %v", err)
			return uc.createFallbackExitResponse(req), nil
		case ErrTypeDatabase:
			// Database error - use fallback mechanism
			uc.log.WithContext(ctx).Warnf("[EXIT] Database error, using fallback: %v", err)
			return uc.createFallbackExitResponse(req), nil
		default:
			return &v1.ExitData{
				PlateNumber:    req.PlateNumber,
				Allowed:        false,
				GateOpen:       false,
				DisplayMessage: uc.config.Messages.SystemError,
			}, nil
		}
	default:
		// Unknown error type
		uc.log.WithContext(ctx).Errorf("[EXIT] Unknown error: %v", err)
		return &v1.ExitData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: uc.config.Messages.SystemError,
		}, nil
	}
}

// sendDeviceCommandWithRetry sends device command with retry mechanism
func (uc *EntryExitUseCase) sendDeviceCommandWithRetry(ctx context.Context, deviceID, command string, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if err := uc.mqttClient.PublishCommand(ctx, deviceID, command); err != nil {
			uc.log.WithContext(ctx).Warnf("[DEVICE] Command attempt %d failed: %v", i+1, err)
			time.Sleep(time.Duration(i+1) * 100 * time.Millisecond)
			continue
		}
		uc.log.WithContext(ctx).Infof("[DEVICE] Command sent successfully: %s to %s", command, deviceID)
		return nil
	}
	return fmt.Errorf("failed to send command after %d attempts", maxRetries)
}

// createFallbackEntryResponse creates fallback response for entry errors
func (uc *EntryExitUseCase) createFallbackEntryResponse(req *v1.EntryRequest) *v1.EntryData {
	// In fallback mode, we allow entry but mark it for manual review
	return &v1.EntryData{
		PlateNumber:    req.PlateNumber,
		Allowed:        true,
		GateOpen:       true,
		DisplayMessage: uc.config.Messages.FallbackMode,
	}
}

// createFallbackExitResponse creates fallback response for exit errors
func (uc *EntryExitUseCase) createFallbackExitResponse(req *v1.ExitRequest) *v1.ExitData {
	// In fallback mode, we allow exit but mark it for manual review
	return &v1.ExitData{
		PlateNumber:    req.PlateNumber,
		Allowed:        true,
		GateOpen:       true,
		DisplayMessage: uc.config.Messages.FallbackMode,
	}
}

func (uc *EntryExitUseCase) processEntryTransaction(ctx context.Context, req *v1.EntryRequest) (*v1.EntryData, error) {
	lane, err := uc.vehicleRepo.GetLaneByDeviceCode(ctx, req.DeviceId)
	if err != nil {
		return nil, &EntryExitError{Type: ErrTypeDevice, Message: "failed to get lane info", Err: err}
	}
	uc.log.WithContext(ctx).Infof("[ENTRY] Found lane - LaneID: %s, LotID: %s", lane.ID, lane.LotID)

	vehicle, err := uc.vehicleRepo.GetVehicleByPlate(ctx, req.PlateNumber)
	if err != nil {
		uc.log.WithContext(ctx).Warnf("[ENTRY] Failed to get vehicle info: %v, proceeding with unknown vehicle", err)
		// Continue processing without vehicle info
	}

	existingRecord, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
	if err != nil {
		return nil, &EntryExitError{Type: ErrTypeDatabase, Message: "failed to check existing entry", Err: err}
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
		return nil, &EntryExitError{Type: ErrTypeDatabase, Message: "failed to create parking record", Err: err}
	}

	return uc.buildEntryResponse(record, req.PlateNumber, vehicle), nil
}

func (uc *EntryExitUseCase) processExitTransaction(ctx context.Context, req *v1.ExitRequest) (*v1.ExitData, error) {
	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, req.DeviceId)
	if err != nil {
		return nil, &EntryExitError{Type: ErrTypeDevice, Message: "failed to get device info", Err: err}
	}

	lane, err := uc.vehicleRepo.GetLaneByDeviceCode(ctx, req.DeviceId)
	if err != nil {
		return nil, &EntryExitError{Type: ErrTypeDevice, Message: "failed to get lane info", Err: err}
	}

	record, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
	if err != nil {
		return nil, &EntryExitError{Type: ErrTypeDatabase, Message: "failed to get entry record", Err: err}
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
		return nil, &EntryExitError{Type: ErrTypeBilling, Message: "failed to calculate fee", Err: err}
	}

	// Only update the record after fee calculation succeeds
	if err := uc.updateParkingRecordForExit(ctx, record, req, device, lane, exitTime, duration); err != nil {
		return nil, &EntryExitError{Type: ErrTypeDatabase, Message: "failed to update parking record", Err: err}
	}

	return uc.buildExitResponse(record, req, duration, amount, discountAmount, finalAmount), nil
}

func (uc *EntryExitUseCase) withDistributedLock(ctx context.Context, lockKey string, fn func() error) error {
	owner := lock.GenerateUniqueOwner()
	uc.log.WithContext(ctx).Debugf("[LOCK] Acquiring lock - Key: %s, Owner: %s", lockKey, owner)

	acquired, err := uc.lockRepo.AcquireLock(ctx, lockKey, owner, uc.config.LockTTL)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[LOCK] Failed to acquire lock: %v", err)
		return &EntryExitError{Type: ErrTypeLock, Message: "failed to acquire distributed lock", Err: err}
	}
	if !acquired {
		uc.log.WithContext(ctx).Warnf("[LOCK] Lock held by another process - Key: %s", lockKey)
		return &EntryExitError{Type: ErrTypeLock, Message: "duplicate request in progress"}
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

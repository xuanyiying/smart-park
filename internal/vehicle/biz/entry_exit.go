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
//
// cfg carries the tunable thresholds (lock TTL, recognition confidence, driver-facing
// messages). Passing nil falls back to the built-in defaults. It is injected rather than
// hard-coded so that operations can change behaviour without a rebuild.
func NewEntryExitUseCase(vehicleRepo VehicleRepo, billingClient billing.Client, mqttClient mqtt.Client, lockRepo lock.LockRepo, adapterFactory *device.AdapterFactory, cfg *Config, logger log.Logger) *EntryExitUseCase {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	return &EntryExitUseCase{
		vehicleRepo:    vehicleRepo,
		billingClient:  billingClient,
		mqttClient:     mqttClient,
		lockRepo:       lockRepo,
		adapterFactory: adapterFactory,
		config:         cfg,
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
		// The parking record is committed, so the system considers this vehicle admitted
		// even though the barrier may never have moved. That inconsistency is an incident
		// staff must be able to find, not a warning to ignore.
		uc.logManualReview(ctx, "entry_gate_command", req.PlateNumber, req.DeviceId, err)
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
			uc.logManualReview(ctx, "exit_gate_command", req.PlateNumber, req.DeviceId, err)
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
			// Database error - admit the vehicle but record the incident for staff.
			return uc.createFallbackEntryResponse(ctx, req, err), nil
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

// handleExitError handles exit errors.
//
// Exit is the revenue checkpoint of the whole system, so unlike entry it never opens the
// gate on a technical failure. Releasing a driver because the billing service hiccupped
// hands out free parking and leaves no trace to reconcile against; keeping the gate
// closed routes the driver to staff, who can charge manually and still leave an auditable
// record.
func (uc *EntryExitUseCase) handleExitError(ctx context.Context, req *v1.ExitRequest, err error) (*v1.ExitData, error) {
	// Every blocked exit is an operational incident: log it loudly and uniformly so that
	// monitoring can alert on it and staff can reconcile the affected sessions later.
	uc.logManualReview(ctx, "exit", req.PlateNumber, req.DeviceId, err)

	switch e := err.(type) {
	case *EntryExitError:
		switch e.Type {
		case ErrTypeValidation:
			return uc.denyExit(req, uc.config.Messages.ValidationError), nil
		case ErrTypeLock:
			return uc.denyExit(req, uc.config.Messages.DuplicateExit), nil
		case ErrTypeBilling:
			return uc.denyExit(req, uc.config.Messages.BillingUnavailable), nil
		case ErrTypeDatabase:
			return uc.denyExit(req, uc.config.Messages.SystemError), nil
		default:
			return uc.denyExit(req, uc.config.Messages.SystemError), nil
		}
	default:
		uc.log.WithContext(ctx).Errorf("[EXIT] Unknown error: %v", err)
		return uc.denyExit(req, uc.config.Messages.SystemError), nil
	}
}

// denyExit builds a refusal response. The gate stays closed and the driver is told to
// contact staff rather than being waved through.
func (uc *EntryExitUseCase) denyExit(req *v1.ExitRequest, message string) *v1.ExitData {
	return &v1.ExitData{
		PlateNumber:    req.PlateNumber,
		Allowed:        false,
		GateOpen:       false,
		DisplayMessage: message,
	}
}

// logManualReview emits a structured, greppable record for sessions that need a human.
//
// The previous code claimed to "mark for manual review" without recording anything, so
// these incidents were invisible. The marker is what lets operations find them.
func (uc *EntryExitUseCase) logManualReview(ctx context.Context, phase, plateNumber, deviceID string, err error) {
	uc.log.WithContext(ctx).Errorf("MANUAL_REVIEW_REQUIRED phase=%s plate=%s device=%s reason=%v",
		phase, plateNumber, deviceID, err)
}

// sendDeviceCommandWithRetry delivers a command to a lane device, retrying with
// exponential backoff.
//
// A failure here is reported to the caller instead of being swallowed: the caller has
// already committed the parking record, so a silently lost "open gate" command leaves the
// session in a state where the system believes the barrier is up and the driver is still
// sitting in front of a closed one.
func (uc *EntryExitUseCase) sendDeviceCommandWithRetry(ctx context.Context, deviceID, command string, maxRetries int) error {
	if uc.mqttClient == nil {
		return fmt.Errorf("device command %s could not be delivered: mqtt client is not configured", command)
	}
	if deviceID == "" {
		return fmt.Errorf("device command %s could not be delivered: no device bound to the lane", command)
	}

	var lastErr error
	for attempt := 0; attempt < maxRetries; attempt++ {
		lastErr = uc.mqttClient.PublishCommand(ctx, &mqtt.Command{DeviceID: deviceID, Command: mqtt.CommandType(command)})
		if lastErr == nil {
			uc.log.WithContext(ctx).Infof("[DEVICE] Command sent successfully: %s to %s", command, deviceID)
			return nil
		}

		uc.log.WithContext(ctx).Warnf("[DEVICE] Command attempt %d/%d failed: %v", attempt+1, maxRetries, lastErr)

		// Exponential backoff bounded by the request context, so a hung broker cannot pin
		// a request thread for longer than the caller is willing to wait.
		backoff := time.Duration(1<<uint(attempt)) * 100 * time.Millisecond
		select {
		case <-ctx.Done():
			return fmt.Errorf("device command %s aborted: %w", command, ctx.Err())
		case <-time.After(backoff):
		}
	}

	return fmt.Errorf("failed to send command %s to device %s after %d attempts: %w", command, deviceID, maxRetries, lastErr)
}

// createFallbackEntryResponse builds the degraded-mode response used when the parking
// record cannot be persisted.
//
// Entry is deliberately still allowed: refusing would strand a car in front of the barrier
// and block the lane. The trade-off is that the session exists without a database record,
// so the incident is recorded through logManualReview (where the old code only pretended
// to) and must be reconciled by staff.
func (uc *EntryExitUseCase) createFallbackEntryResponse(ctx context.Context, req *v1.EntryRequest, err error) *v1.EntryData {
	uc.logManualReview(ctx, "entry", req.PlateNumber, req.DeviceId, err)

	return &v1.EntryData{
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
		// A database failure is not the same as "no registered vehicle": swallowing it
		// silently downgrades monthly/VIP vehicles to temporary rates. Surface it so the
		// entry handler admits the car in degraded mode while the incident is recorded.
		return nil, &EntryExitError{Type: ErrTypeDatabase, Message: "failed to get vehicle info", Err: err}
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

	// 黑名单校验：命中即拦截，道闸保持关闭。查询故障按数据库错误上报，走降级留痕路径。
	blacklisted, err := uc.vehicleRepo.GetBlacklistEntry(ctx, req.PlateNumber)
	if err != nil {
		return nil, &EntryExitError{Type: ErrTypeDatabase, Message: "failed to check blacklist", Err: err}
	}
	if blacklisted != nil && blacklisted.Active {
		uc.log.WithContext(ctx).Warnf("[ENTRY] Blacklisted vehicle denied - PlateNumber: [REDACTED]")
		return &v1.EntryData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: uc.config.Messages.Blacklisted,
		}, nil
	}

	// 车位余量校验：满位时拒绝入场，而不是放行一辆无处可停的车。
	full, err := uc.isLotFull(ctx, lane.LotID)
	if err != nil {
		return nil, err
	}
	if full {
		uc.log.WithContext(ctx).Warnf("[ENTRY] Lot full, entry denied - PlateNumber: [REDACTED]")
		return &v1.EntryData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: uc.config.Messages.LotFull,
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

	// The driver may already have settled through the mini program or by scanning a code.
	// The payment service marks the record as paid; releasing the gate here is what makes
	// pre-payment and frictionless exit work at all.
	if record.ExitStatus == ExitStatusPaid {
		uc.log.WithContext(ctx).Infof("[EXIT] Record already paid, releasing gate - RecordID: %s", record.ID)
		return &v1.ExitData{
			RecordId:        record.ID.String(),
			PlateNumber:     req.PlateNumber,
			ParkingDuration: int32(time.Since(record.EntryTime).Seconds()),
			FinalAmount:     0,
			Allowed:         true,
			GateOpen:        true,
			DisplayMessage:  uc.config.Messages.FreePass,
		}, nil
	}

	exitTime := time.Now()
	duration := int(exitTime.Sub(record.EntryTime).Seconds())

	// Calculate fee first, before updating the record
	vehicle, vehicleType, err := uc.getVehicleInfo(ctx, req.PlateNumber)
	if err != nil {
		// M2: 数据库故障不等于"无车辆信息"，把月卡车按临时车收费是直接的资损与客诉。
		// 这里上报数据库错误，由 handleExitError 保持拦截并留痕。
		return nil, &EntryExitError{Type: ErrTypeDatabase, Message: "failed to get vehicle info", Err: err}
	}
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

// withDistributedLock runs fn while holding a distributed lock, keeping the lock alive
// for as long as fn is still running.
//
// The critical section includes a cross-service billing call, so its duration is not
// bounded by anything we control. Without renewal the lock can expire mid-transaction and
// a second request for the same plate enters, which is exactly what the lock exists to
// prevent; the renewal goroutine extends the lease until fn returns.
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

	stopRenewal := uc.startLockRenewal(ctx, lockKey, owner)
	defer func() {
		stopRenewal()
		if err := uc.lockRepo.ReleaseLock(ctx, lockKey, owner); err != nil {
			uc.log.WithContext(ctx).Warnf("[LOCK] Failed to release lock: %v", err)
		}
	}()

	return fn()
}

// startLockRenewal keeps extending a held lock until the returned stop function is called.
// The lease top-up runs at a third of the TTL so a single lost renewal still leaves time
// to retry before expiry.
func (uc *EntryExitUseCase) startLockRenewal(ctx context.Context, lockKey, owner string) func() {
	interval := uc.config.LockTTL / 3
	if interval <= 0 {
		interval = time.Second
	}

	stop := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				// Detached from the request context: renewal must keep working even if the
				// caller's deadline has passed, otherwise we would drop the lock while the
				// critical section is still executing.
				renewCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), interval)
				err := uc.lockRepo.ExtendLock(renewCtx, lockKey, owner, uc.config.LockTTL)
				cancel()
				if err != nil {
					uc.log.WithContext(ctx).Warnf("[LOCK] Failed to extend lock %s: %v", lockKey, err)
				}
			}
		}
	}()

	return func() {
		close(stop)
		<-stopped
	}
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

func (uc *EntryExitUseCase) getVehicleInfo(ctx context.Context, plateNumber string) (*Vehicle, string, error) {
	vehicleType := VehicleTypeTemporary
	vehicle, err := uc.vehicleRepo.GetVehicleByPlate(ctx, plateNumber)
	if err != nil {
		return nil, vehicleType, err
	}
	if vehicle != nil {
		vehicleType = vehicle.VehicleType
	}
	return vehicle, vehicleType, nil
}

// isLotFull reports whether the parking lot has reached its capacity. A lot with no
// configured or discoverable capacity (<= 0) is treated as unlimited: the check is
// skipped rather than blocking every entry behind a data gap.
func (uc *EntryExitUseCase) isLotFull(ctx context.Context, lotID uuid.UUID) (bool, error) {
	capacity, err := uc.vehicleRepo.GetLotCapacity(ctx, lotID)
	if err != nil {
		// 容量查询失败按"未知容量"处理并放行，避免数据库抖动阻断所有入场；
		// 真实满位风险由每日运营复核兜底。
		uc.log.WithContext(ctx).Warnf("[ENTRY] failed to get lot capacity for %s: %v, capacity check skipped", lotID, err)
		return false, nil
	}
	if capacity <= 0 {
		return false, nil
	}

	occupied, err := uc.vehicleRepo.CountActiveRecordsByLot(ctx, lotID)
	if err != nil {
		return false, &EntryExitError{Type: ErrTypeDatabase, Message: "failed to count occupied spaces", Err: err}
	}
	return occupied >= capacity, nil
}

func (uc *EntryExitUseCase) calculateExitFee(ctx context.Context, record *ParkingRecord, lane *Lane, exitTime time.Time, vehicle *Vehicle, vehicleType string) (int64, int64, int64, error) {
	feeResult, err := uc.billingClient.CalculateFee(ctx, record.ID.String(), lane.LotID.String(),
		record.EntryTime.Unix(), exitTime.Unix(), vehicleType)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[EXIT] Failed to calculate fee: %v", err)
		return 0, 0, 0, fmt.Errorf("fee calculation failed: %w", err)
	}

	// Amounts are cents (分) from the billing service.
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

func (uc *EntryExitUseCase) buildExitResponse(record *ParkingRecord, req *v1.ExitRequest, duration int, amount, discountAmount, finalAmount int64) *v1.ExitData {
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

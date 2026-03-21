// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// Vehicle represents a vehicle entity in business logic.
type Vehicle struct {
	ID                uuid.UUID
	PlateNumber       string
	VehicleType       string
	OwnerName         string
	OwnerPhone        string
	MonthlyValidUntil *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ParkingRecord represents a parking record entity in business logic.
type ParkingRecord struct {
	ID              uuid.UUID
	LotID           uuid.UUID
	EntryLaneID     uuid.UUID
	VehicleID       *uuid.UUID
	PlateNumber     *string
	PlateNumberSource string
	EntryTime       time.Time
	EntryImageURL   string
	RecordStatus    string
	ExitTime        *time.Time
	ExitImageURL    string
	ExitLaneID      *uuid.UUID
	ExitDeviceID    string
	ParkingDuration int
	ExitStatus      string
	PaymentLock     int
	Metadata        map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// Device represents a device entity in business logic.
type Device struct {
	ID            uuid.UUID
	DeviceID      string
	LotID         *uuid.UUID
	LaneID        *uuid.UUID
	DeviceType    string
	DeviceSecret  string
	GateID        string
	Enabled       bool
	Status        string
	LastHeartbeat *time.Time
}

// Lane represents a lane entity in business logic.
type Lane struct {
	ID           uuid.UUID
	LotID        uuid.UUID
	LaneNo       int
	Direction    string
	Status       string
	DeviceConfig map[string]interface{}
}

// VehicleRepo defines the repository interface for vehicle operations.
type VehicleRepo interface {
	GetVehicleByPlate(ctx context.Context, plateNumber string) (*Vehicle, error)
	CreateVehicle(ctx context.Context, vehicle *Vehicle) error
	UpdateVehicle(ctx context.Context, vehicle *Vehicle) error
	GetEntryRecord(ctx context.Context, plateNumber string) (*ParkingRecord, error)
	CreateParkingRecord(ctx context.Context, record *ParkingRecord) error
	UpdateParkingRecord(ctx context.Context, record *ParkingRecord) error
	GetParkingRecord(ctx context.Context, recordID uuid.UUID) (*ParkingRecord, error)
	GetDeviceByCode(ctx context.Context, deviceCode string) (*Device, error)
	UpdateDeviceHeartbeat(ctx context.Context, deviceCode string) error
	GetLaneByDeviceCode(ctx context.Context, deviceCode string) (*Lane, error)
}

// BillingRule represents a billing rule in business logic.
type BillingRule struct {
	ID           uuid.UUID
	LotID        uuid.UUID
	RuleName     string
	RuleType     string
	Conditions   string
	Actions      string
	RuleConfig   map[string]interface{}
	Priority     int
	IsActive     bool
}

// BillingRepo defines the repository interface for billing operations.
type BillingRepo interface {
	GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error)
}

// VehicleUseCase implements vehicle business logic.
type VehicleUseCase struct {
	vehicleRepo VehicleRepo
	billingRepo BillingRepo
	log         *log.Helper
}

// NewVehicleUseCase creates a new VehicleUseCase.
func NewVehicleUseCase(vehicleRepo VehicleRepo, billingRepo BillingRepo, logger log.Logger) *VehicleUseCase {
	return &VehicleUseCase{
		vehicleRepo: vehicleRepo,
		billingRepo: billingRepo,
		log:         log.NewHelper(logger),
	}
}

// Entry handles vehicle entry.
func (uc *VehicleUseCase) Entry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryData, error) {
	// Get device info
	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, req.DeviceId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get device: %v", err)
		return nil, err
	}

	// Get lane info
	lane, err := uc.vehicleRepo.GetLaneByDeviceCode(ctx, req.DeviceId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get lane: %v", err)
		return nil, err
	}

	// Check if there's an existing entry record
	var vehicle *Vehicle
	if req.PlateNumber != "" {
		vehicle, _ = uc.vehicleRepo.GetVehicleByPlate(ctx, req.PlateNumber)

		// Check for duplicate entry
		existingRecord, _ := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
		if existingRecord != nil {
			// Already has an active entry, might be a duplicate
			uc.log.WithContext(ctx).Warnf("duplicate entry detected for plate: %s", req.PlateNumber)
		}
	}

	// Create parking record
	record := &ParkingRecord{
		ID:            uuid.New(),
		LotID:         lane.LotID,
		EntryLaneID:   lane.ID,
		EntryTime:     time.Now(),
		EntryImageURL: req.PlateImageUrl,
		RecordStatus:  "entry",
		ExitStatus:    "unpaid",
	}

	if vehicle != nil {
		record.VehicleID = &vehicle.ID
		plateNumber := req.PlateNumber
		record.PlateNumber = &plateNumber
		record.PlateNumberSource = "camera"
	}

	if err := uc.vehicleRepo.CreateParkingRecord(ctx, record); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create parking record: %v", err)
		return nil, err
	}

	// Determine if gate should open
	allowed := true
	gateOpen := true
	displayMessage := "欢迎光临"

	if vehicle != nil && vehicle.VehicleType == "monthly" {
		displayMessage = "月卡车，欢迎光临"
	} else if vehicle != nil && vehicle.VehicleType == "vip" {
		displayMessage = "VIP车辆，欢迎光临"
	}

	return &v1.EntryData{
		RecordId:       record.ID.String(),
		PlateNumber:    req.PlateNumber,
		Allowed:        allowed,
		GateOpen:       gateOpen,
		DisplayMessage: displayMessage,
	}, nil
}

// Exit handles vehicle exit.
func (uc *VehicleUseCase) Exit(ctx context.Context, req *v1.ExitRequest) (*v1.ExitData, error) {
	// Get device info
	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, req.DeviceId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get device: %v", err)
		return nil, err
	}

	// Get lane info
	lane, err := uc.vehicleRepo.GetLaneByDeviceCode(ctx, req.DeviceId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get lane: %v", err)
		return nil, err
	}

	// Get entry record
	record, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get entry record: %v", err)
		return &v1.ExitData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: "未找到入场记录",
		}, nil
	}

	// Calculate parking duration
	exitTime := time.Now()
	duration := int(exitTime.Sub(record.EntryTime).Seconds())

	// Update parking record
	record.ExitTime = &exitTime
	record.ExitImageURL = req.PlateImageUrl
	record.ExitLaneID = &lane.ID
	record.ExitDeviceID = device.DeviceID
	record.RecordStatus = "exiting"
	record.ParkingDuration = duration

	if err := uc.vehicleRepo.UpdateParkingRecord(ctx, record); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update parking record: %v", err)
		return nil, err
	}

	// Calculate fee (this would call billing service in production)
	amount := float64(duration) * 0.01 // Simple calculation for demo
	var discountAmount float64
	var finalAmount = amount

	// Check vehicle type for discounts
	vehicle, _ := uc.vehicleRepo.GetVehicleByPlate(ctx, req.PlateNumber)
	if vehicle != nil {
		if vehicle.VehicleType == "monthly" && vehicle.MonthlyValidUntil != nil && vehicle.MonthlyValidUntil.After(time.Now()) {
			// Monthly card valid
			discountAmount = amount
			finalAmount = 0
		} else if vehicle.VehicleType == "vip" {
			// VIP discount
			discountAmount = amount * 0.5
			finalAmount = amount * 0.5
		}
	}

	allowed := finalAmount == 0
	gateOpen := finalAmount == 0

	displayMessage := "请缴费"
	if finalAmount == 0 {
		displayMessage = "免费放行"
		gateOpen = true
	}

	return &v1.ExitData{
		RecordId:       record.ID.String(),
		PlateNumber:    req.PlateNumber,
		ParkingDuration: int32(duration),
		Amount:         amount,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
		Allowed:        allowed,
		GateOpen:       gateOpen,
		DisplayMessage: displayMessage,
	}, nil
}

// Heartbeat handles device heartbeat.
func (uc *VehicleUseCase) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) error {
	return uc.vehicleRepo.UpdateDeviceHeartbeat(ctx, req.DeviceId)
}

// GetDeviceStatus retrieves device status.
func (uc *VehicleUseCase) GetDeviceStatus(ctx context.Context, deviceID string) (*v1.DeviceStatus, error) {
	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, deviceID)
	if err != nil {
		return nil, err
	}

	online := true
	if device.LastHeartbeat != nil {
		online = time.Since(*device.LastHeartbeat) < 5*time.Minute
	}

	var lastHeartbeat string
	if device.LastHeartbeat != nil {
		lastHeartbeat = device.LastHeartbeat.Format(time.RFC3339)
	}

	return &v1.DeviceStatus{
		DeviceId:     device.DeviceID,
		Online:       online,
		Status:       device.Status,
		LastHeartbeat: lastHeartbeat,
	}, nil
}

// GetVehicleInfo retrieves vehicle information.
func (uc *VehicleUseCase) GetVehicleInfo(ctx context.Context, plateNumber string) (*v1.VehicleInfo, error) {
	vehicle, err := uc.vehicleRepo.GetVehicleByPlate(ctx, plateNumber)
	if err != nil {
		return nil, err
	}

	var monthlyValidUntil string
	if vehicle.MonthlyValidUntil != nil {
		monthlyValidUntil = vehicle.MonthlyValidUntil.Format(time.RFC3339)
	}

	return &v1.VehicleInfo{
		PlateNumber:       vehicle.PlateNumber,
		VehicleType:       vehicle.VehicleType,
		OwnerName:         vehicle.OwnerName,
		OwnerPhone:        vehicle.OwnerPhone,
		MonthlyValidUntil: monthlyValidUntil,
	}, nil
}

// SendCommand sends a command to a device.
func (uc *VehicleUseCase) SendCommand(ctx context.Context, deviceID string, command string, params map[string]string) (*v1.CommandData, error) {
	// In production, this would send command to device via MQTT/WebSocket
	uc.log.WithContext(ctx).Infof("sending command %s to device %s", command, deviceID)

	return &v1.CommandData{
		CommandId: uuid.New().String(),
		Status:    "sent",
	}, nil
}

// BillingUseCase implements billing business logic.
type BillingUseCase struct {
	repo BillingRepo
	log  *log.Helper
}

// NewBillingUseCase creates a new BillingUseCase.
func NewBillingUseCase(repo BillingRepo, logger log.Logger) *BillingUseCase {
	return &BillingUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// CalculateFee calculates parking fee.
func (uc *BillingUseCase) CalculateFee(ctx context.Context, recordID string, lotID uuid.UUID, entryTime, exitTime time.Time, vehicleType string) (float64, float64, error) {
	// Get billing rules
	rules, err := uc.repo.GetRulesByLotID(ctx, lotID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get billing rules: %v", err)
		return 0, 0, err
	}

	// Calculate base duration
	duration := exitTime.Sub(entryTime)

	// Apply rules (simplified logic)
	var baseAmount float64
	var discountAmount float64

	// Default rate: 2 yuan per hour
	hours := duration.Hours()
	baseAmount = hours * 2

	// Apply discounts based on vehicle type and rules
	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}
		// Apply rule logic here based on rule type
		if rule.RuleType == "vip" && vehicleType == "vip" {
			discountAmount += baseAmount * 0.5
		}
		if rule.RuleType == "monthly" && vehicleType == "monthly" {
			discountAmount = baseAmount
		}
	}

	return baseAmount, discountAmount, nil
}

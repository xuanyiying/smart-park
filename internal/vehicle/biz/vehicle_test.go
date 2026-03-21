package biz

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// MockVehicleRepo is a mock implementation of VehicleRepo for testing.
type MockVehicleRepo struct {
	Vehicles      map[string]*Vehicle
	ParkingRecords map[string]*ParkingRecord
	Devices       map[string]*Device
	Lanes         map[string]*Lane
}

func NewMockVehicleRepo() *MockVehicleRepo {
	return &MockVehicleRepo{
		Vehicles:      make(map[string]*Vehicle),
		ParkingRecords: make(map[string]*ParkingRecord),
		Devices:       make(map[string]*Device),
		Lanes:         make(map[string]*Lane),
	}
}

func (m *MockVehicleRepo) GetVehicleByPlate(ctx context.Context, plateNumber string) (*Vehicle, error) {
	return m.Vehicles[plateNumber], nil
}

func (m *MockVehicleRepo) CreateVehicle(ctx context.Context, vehicle *Vehicle) error {
	m.Vehicles[vehicle.PlateNumber] = vehicle
	return nil
}

func (m *MockVehicleRepo) UpdateVehicle(ctx context.Context, vehicle *Vehicle) error {
	m.Vehicles[vehicle.PlateNumber] = vehicle
	return nil
}

func (m *MockVehicleRepo) GetEntryRecord(ctx context.Context, plateNumber string) (*ParkingRecord, error) {
	return m.ParkingRecords[plateNumber], nil
}

func (m *MockVehicleRepo) CreateParkingRecord(ctx context.Context, record *ParkingRecord) error {
	if record.PlateNumber != nil {
		m.ParkingRecords[*record.PlateNumber] = record
	}
	return nil
}

func (m *MockVehicleRepo) UpdateParkingRecord(ctx context.Context, record *ParkingRecord) error {
	return nil
}

func (m *MockVehicleRepo) GetParkingRecord(ctx context.Context, recordID uuid.UUID) (*ParkingRecord, error) {
	return nil, nil
}

func (m *MockVehicleRepo) GetDeviceByCode(ctx context.Context, deviceCode string) (*Device, error) {
	return m.Devices[deviceCode], nil
}

func (m *MockVehicleRepo) UpdateDeviceHeartbeat(ctx context.Context, deviceCode string) error {
	if d, ok := m.Devices[deviceCode]; ok {
		now := time.Now()
		d.LastHeartbeat = &now
	}
	return nil
}

func (m *MockVehicleRepo) GetLaneByDeviceCode(ctx context.Context, deviceCode string) (*Lane, error) {
	return m.Lanes[deviceCode], nil
}

// MockBillingRepo is a mock implementation of BillingRepo for testing.
type MockBillingRepo struct {
	Rules map[uuid.UUID][]*BillingRule
}

func NewMockBillingRepo() *MockBillingRepo {
	return &MockBillingRepo{
		Rules: make(map[uuid.UUID][]*BillingRule),
	}
}

func (m *MockBillingRepo) GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error) {
	return m.Rules[lotID], nil
}

func TestVehicleUseCase_Entry(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockVehicleRepo()
	mockBillingRepo := NewMockBillingRepo()

	// Setup test data
	lotID := uuid.New()
	deviceID := "test-device-001"
	laneID := uuid.New()

	mockRepo.Devices[deviceID] = &Device{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		DeviceType: "camera",
		Status:     "active",
	}

	mockRepo.Lanes[deviceID] = &Lane{
		ID:        laneID,
		LotID:     lotID,
		LaneNo:    1,
		Direction: "entry",
		Status:    "active",
	}

	uc := NewVehicleUseCase(mockRepo, mockBillingRepo, logger)

	req := &v1.EntryRequest{
		DeviceId:     deviceID,
		PlateNumber:  "京A12345",
		PlateImageUrl: "http://example.com/image.jpg",
		Confidence:   0.95,
	}

	data, err := uc.Entry(context.Background(), req)
	if err != nil {
		t.Fatalf("Entry failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if !data.Allowed {
		t.Error("Expected allowed to be true")
	}

	if !data.GateOpen {
		t.Error("Expected gate_open to be true")
	}
}

func TestVehicleUseCase_Exit(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockVehicleRepo()
	mockBillingRepo := NewMockBillingRepo()

	// Setup test data
	lotID := uuid.New()
	deviceID := "test-device-002"
	laneID := uuid.New()
	entryLaneID := uuid.New()
	plateNumber := "京A12345"

	mockRepo.Devices[deviceID] = &Device{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		DeviceType: "camera",
		Status:     "active",
	}

	mockRepo.Lanes[deviceID] = &Lane{
		ID:        laneID,
		LotID:     lotID,
		LaneNo:    1,
		Direction: "exit",
		Status:    "active",
	}

	entryTime := time.Now().Add(-2 * time.Hour)
	mockRepo.ParkingRecords[plateNumber] = &ParkingRecord{
		ID:            uuid.New(),
		LotID:         lotID,
		EntryLaneID:   entryLaneID,
		PlateNumber:   &plateNumber,
		EntryTime:     entryTime,
		RecordStatus:  "entry",
	}

	uc := NewVehicleUseCase(mockRepo, mockBillingRepo, logger)

	req := &v1.ExitRequest{
		DeviceId:     deviceID,
		PlateNumber:  plateNumber,
		PlateImageUrl: "http://example.com/exit-image.jpg",
		Confidence:   0.95,
	}

	data, err := uc.Exit(context.Background(), req)
	if err != nil {
		t.Fatalf("Exit failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.PlateNumber != plateNumber {
		t.Errorf("Expected plate number %s, got %s", plateNumber, data.PlateNumber)
	}

	if data.ParkingDuration <= 0 {
		t.Error("Expected positive parking duration")
	}
}

func TestVehicleUseCase_Heartbeat(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockVehicleRepo()
	mockBillingRepo := NewMockBillingRepo()

	deviceID := "test-device-003"
	mockRepo.Devices[deviceID] = &Device{
		ID:         uuid.New(),
		DeviceID:   deviceID,
		DeviceType: "camera",
		Status:     "active",
	}

	uc := NewVehicleUseCase(mockRepo, mockBillingRepo, logger)

	req := &v1.HeartbeatRequest{
		DeviceId: deviceID,
		Status:   "active",
	}

	err := uc.Heartbeat(context.Background(), req)
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	if mockRepo.Devices[deviceID].LastHeartbeat == nil {
		t.Error("Expected last heartbeat to be updated")
	}
}

func TestVehicleUseCase_GetDeviceStatus(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockVehicleRepo()
	mockBillingRepo := NewMockBillingRepo()

	deviceID := "test-device-004"
	now := time.Now()
	mockRepo.Devices[deviceID] = &Device{
		ID:            uuid.New(),
		DeviceID:      deviceID,
		DeviceType:    "camera",
		Status:        "active",
		LastHeartbeat: &now,
	}

	uc := NewVehicleUseCase(mockRepo, mockBillingRepo, logger)

	status, err := uc.GetDeviceStatus(context.Background(), deviceID)
	if err != nil {
		t.Fatalf("GetDeviceStatus failed: %v", err)
	}

	if status == nil {
		t.Fatal("Expected non-nil status")
	}

	if !status.Online {
		t.Error("Expected device to be online")
	}
}

func TestVehicleUseCase_GetVehicleInfo(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockVehicleRepo()
	mockBillingRepo := NewMockBillingRepo()

	plateNumber := "京B88888"
	validUntil := time.Now().Add(30 * 24 * time.Hour)
	mockRepo.Vehicles[plateNumber] = &Vehicle{
		ID:                uuid.New(),
		PlateNumber:       plateNumber,
		VehicleType:       "monthly",
		OwnerName:         "张三",
		OwnerPhone:        "13800138000",
		MonthlyValidUntil: &validUntil,
	}

	uc := NewVehicleUseCase(mockRepo, mockBillingRepo, logger)

	info, err := uc.GetVehicleInfo(context.Background(), plateNumber)
	if err != nil {
		t.Fatalf("GetVehicleInfo failed: %v", err)
	}

	if info == nil {
		t.Fatal("Expected non-nil info")
	}

	if info.PlateNumber != plateNumber {
		t.Errorf("Expected plate number %s, got %s", plateNumber, info.PlateNumber)
	}

	if info.VehicleType != "monthly" {
		t.Errorf("Expected vehicle type monthly, got %s", info.VehicleType)
	}
}

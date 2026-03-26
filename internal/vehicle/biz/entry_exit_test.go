package biz

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
)

// MockBillingClient is a mock implementation of billing.Client for testing.
type MockBillingClient struct {
	HourlyRate float64
}

func NewMockBillingClient() *MockBillingClient {
	return &MockBillingClient{
		HourlyRate: 2.0,
	}
}

func (m *MockBillingClient) CalculateFee(ctx context.Context, recordID string, lotID string, entryTime, exitTime int64, vehicleType string) (*billing.FeeResult, error) {
	duration := float64(exitTime-entryTime) / 3600.0
	baseAmount := duration * m.HourlyRate

	return &billing.FeeResult{
		BaseAmount:     baseAmount,
		DiscountAmount: 0,
		FinalAmount:    baseAmount,
	}, nil
}

func TestEntryExitUseCase_Entry(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockVehicleRepo()
	mockBillingClient := NewMockBillingClient()

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

	uc := NewEntryExitUseCase(mockRepo, mockBillingClient, nil, NewMockLockRepo(), logger)

	req := &v1.EntryRequest{
		DeviceId:      deviceID,
		PlateNumber:   "京A12345",
		PlateImageUrl: "http://example.com/image.jpg",
		Confidence:    0.95,
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

func TestEntryExitUseCase_Exit(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockVehicleRepo()
	mockBillingClient := NewMockBillingClient()

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
		ID:           uuid.New(),
		LotID:        lotID,
		EntryLaneID:  entryLaneID,
		PlateNumber:  &plateNumber,
		EntryTime:    entryTime,
		RecordStatus: "entry",
	}

	uc := NewEntryExitUseCase(mockRepo, mockBillingClient, nil, NewMockLockRepo(), logger)

	req := &v1.ExitRequest{
		DeviceId:      deviceID,
		PlateNumber:   plateNumber,
		PlateImageUrl: "http://example.com/exit-image.jpg",
		Confidence:    0.95,
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

func TestEntryExitUseCase_Entry_DuplicateEntry(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockVehicleRepo()
	mockBillingClient := NewMockBillingClient()

	lotID := uuid.New()
	deviceID := "test-device-003"
	laneID := uuid.New()
	plateNumber := "京C12345"

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

	entryTime := time.Now().Add(-1 * time.Hour)
	mockRepo.ParkingRecords[plateNumber] = &ParkingRecord{
		ID:           uuid.New(),
		LotID:        lotID,
		EntryLaneID:  laneID,
		PlateNumber:  &plateNumber,
		EntryTime:    entryTime,
		RecordStatus: "entry",
	}

	uc := NewEntryExitUseCase(mockRepo, mockBillingClient, nil, NewMockLockRepo(), logger)

	req := &v1.EntryRequest{
		DeviceId:      deviceID,
		PlateNumber:   plateNumber,
		PlateImageUrl: "http://example.com/image.jpg",
		Confidence:    0.95,
	}

	data, err := uc.Entry(context.Background(), req)
	if err != nil {
		t.Fatalf("Entry should not return error: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.Allowed {
		t.Error("Expected allowed to be false for duplicate entry")
	}

	if data.GateOpen {
		t.Error("Expected gate_open to be false for duplicate entry")
	}

	if data.DisplayMessage != "车辆已在场内，请勿重复入场" {
		t.Errorf("Expected duplicate entry message, got %s", data.DisplayMessage)
	}
}

func TestEntryExitUseCase_Exit_NoEntryRecord(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockVehicleRepo()
	mockBillingClient := NewMockBillingClient()

	lotID := uuid.New()
	deviceID := "test-device-004"
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
		Direction: "exit",
		Status:    "active",
	}

	uc := NewEntryExitUseCase(mockRepo, mockBillingClient, nil, NewMockLockRepo(), logger)

	req := &v1.ExitRequest{
		DeviceId:      deviceID,
		PlateNumber:   "京D12345",
		PlateImageUrl: "http://example.com/exit-image.jpg",
		Confidence:    0.95,
	}

	data, err := uc.Exit(context.Background(), req)
	if err != nil {
		t.Fatalf("Exit should not return error: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.Allowed {
		t.Error("Expected allowed to be false when no entry record")
	}

	if data.GateOpen {
		t.Error("Expected gate_open to be false when no entry record")
	}

	if data.DisplayMessage != "未找到入场记录" {
		t.Errorf("Expected no entry record message, got %s", data.DisplayMessage)
	}
}

func TestEntryExitUseCase_Exit_MonthlyVehicle(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockVehicleRepo()
	mockBillingClient := NewMockBillingClient()

	lotID := uuid.New()
	deviceID := "test-device-005"
	laneID := uuid.New()
	entryLaneID := uuid.New()
	plateNumber := "京E12345"

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
		ID:           uuid.New(),
		LotID:        lotID,
		EntryLaneID:  entryLaneID,
		PlateNumber:  &plateNumber,
		EntryTime:    entryTime,
		RecordStatus: "entry",
	}

	validUntil := time.Now().Add(30 * 24 * time.Hour)
	mockRepo.Vehicles[plateNumber] = &Vehicle{
		ID:                uuid.New(),
		PlateNumber:       plateNumber,
		VehicleType:       "monthly",
		MonthlyValidUntil: &validUntil,
	}

	uc := NewEntryExitUseCase(mockRepo, mockBillingClient, nil, NewMockLockRepo(), logger)

	req := &v1.ExitRequest{
		DeviceId:      deviceID,
		PlateNumber:   plateNumber,
		PlateImageUrl: "http://example.com/exit-image.jpg",
		Confidence:    0.95,
	}

	data, err := uc.Exit(context.Background(), req)
	if err != nil {
		t.Fatalf("Exit failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.FinalAmount != 0 {
		t.Errorf("Expected final amount to be 0 for monthly vehicle, got %f", data.FinalAmount)
	}

	if !data.Allowed {
		t.Error("Expected allowed to be true for monthly vehicle")
	}

	if !data.GateOpen {
		t.Error("Expected gate_open to be true for monthly vehicle")
	}
}

func TestEntryExitUseCase_Entry_VIPVehicle(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockVehicleRepo()
	mockBillingClient := NewMockBillingClient()

	lotID := uuid.New()
	deviceID := "test-device-006"
	laneID := uuid.New()
	plateNumber := "京G12345"

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

	mockRepo.Vehicles[plateNumber] = &Vehicle{
		ID:          uuid.New(),
		PlateNumber: plateNumber,
		VehicleType: "vip",
	}

	uc := NewEntryExitUseCase(mockRepo, mockBillingClient, nil, NewMockLockRepo(), logger)

	req := &v1.EntryRequest{
		DeviceId:      deviceID,
		PlateNumber:   plateNumber,
		PlateImageUrl: "http://example.com/image.jpg",
		Confidence:    0.95,
	}

	data, err := uc.Entry(context.Background(), req)
	if err != nil {
		t.Fatalf("Entry failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if !data.Allowed {
		t.Error("Expected allowed to be true for VIP vehicle")
	}

	if data.DisplayMessage != "VIP 车辆，欢迎光临" {
		t.Errorf("Expected VIP welcome message, got %s", data.DisplayMessage)
	}
}

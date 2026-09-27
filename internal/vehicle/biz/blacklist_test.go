package biz

import (
	"context"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

func setupEntryTest() (*EntryExitUseCase, *MockVehicleRepo, string, uuid.UUID) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockVehicleRepo()
	mockBillingClient := NewMockBillingClient()

	lotID := uuid.New()
	deviceID := "test-device-blacklist"
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

	uc := NewEntryExitUseCase(mockRepo, mockBillingClient, nil, NewMockLockRepo(), nil, nil, logger)
	return uc, mockRepo, deviceID, lotID
}

// M12: 黑名单车辆必须被拦截，道闸保持关闭。
func TestEntryExitUseCase_Entry_BlacklistedVehicle(t *testing.T) {
	uc, mockRepo, deviceID, _ := setupEntryTest()

	plateNumber := "京B99999"
	if err := mockRepo.CreateBlacklistEntry(context.Background(), &BlacklistEntry{
		ID:          uuid.New(),
		PlateNumber: plateNumber,
		Reason:      "test",
		Active:      true,
	}); err != nil {
		t.Fatalf("seed blacklist failed: %v", err)
	}

	data, err := uc.Entry(context.Background(), &v1.EntryRequest{
		DeviceId:    deviceID,
		PlateNumber: plateNumber,
		Confidence:  0.95,
	})
	if err != nil {
		t.Fatalf("Entry failed: %v", err)
	}
	if data.Allowed {
		t.Error("blacklisted vehicle must not be allowed")
	}
	if data.GateOpen {
		t.Error("gate must stay closed for blacklisted vehicle")
	}
}

// M12: 已移除（active=false）的黑名单记录不再拦截。
func TestEntryExitUseCase_Entry_RemovedBlacklistNotBlocking(t *testing.T) {
	uc, mockRepo, deviceID, _ := setupEntryTest()

	plateNumber := "京B88888"
	if err := mockRepo.CreateBlacklistEntry(context.Background(), &BlacklistEntry{
		ID:          uuid.New(),
		PlateNumber: plateNumber,
		Active:      false,
	}); err != nil {
		t.Fatalf("seed blacklist failed: %v", err)
	}

	data, err := uc.Entry(context.Background(), &v1.EntryRequest{
		DeviceId:    deviceID,
		PlateNumber: plateNumber,
		Confidence:  0.95,
	})
	if err != nil {
		t.Fatalf("Entry failed: %v", err)
	}
	if !data.Allowed || !data.GateOpen {
		t.Error("inactive blacklist entry must not block entry")
	}
}

// M13: 满位时拒绝入场，道闸保持关闭。
func TestEntryExitUseCase_Entry_LotFull(t *testing.T) {
	uc, mockRepo, deviceID, lotID := setupEntryTest()

	capacity := 1
	mockRepo.LotCapacity = &capacity
	mockRepo.ParkingRecords["occupied-1"] = &ParkingRecord{
		ID:           uuid.New(),
		LotID:        lotID,
		RecordStatus: RecordStatusEntry,
	}

	data, err := uc.Entry(context.Background(), &v1.EntryRequest{
		DeviceId:    deviceID,
		PlateNumber: "京B77777",
		Confidence:  0.95,
	})
	if err != nil {
		t.Fatalf("Entry failed: %v", err)
	}
	if data.Allowed {
		t.Error("entry must be denied when the lot is full")
	}
	if data.GateOpen {
		t.Error("gate must stay closed when the lot is full")
	}
}

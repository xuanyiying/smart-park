package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// MockLockRepo is a mock implementation of lock.LockRepo for testing.
type MockLockRepo struct{}

func NewMockLockRepo() *MockLockRepo {
	return &MockLockRepo{}
}

func (m *MockLockRepo) AcquireLock(ctx context.Context, lockKey string, owner string, ttl time.Duration) (bool, error) {
	return true, nil
}

func (m *MockLockRepo) ReleaseLock(ctx context.Context, lockKey string, owner string) error {
	return nil
}

func (m *MockLockRepo) ExtendLock(ctx context.Context, lockKey string, owner string, ttl time.Duration) error {
	return nil
}

func (m *MockLockRepo) GetLockOwner(ctx context.Context, lockKey string) (string, error) {
	return "", nil
}

func (m *MockLockRepo) IsLocked(ctx context.Context, lockKey string) (bool, error) {
	return false, nil
}

func (m *MockLockRepo) TryLockWithRetry(ctx context.Context, lockKey string, owner string, ttl time.Duration,
	maxRetries int, retryInterval time.Duration) (bool, error) {
	return true, nil
}

// MockVehicleRepo is a mock implementation of VehicleRepo for testing.
type MockVehicleRepo struct {
	Vehicles       map[string]*Vehicle
	ParkingRecords map[string]*ParkingRecord
	Devices        map[string]*Device
	Lanes          map[string]*Lane
}

func NewMockVehicleRepo() *MockVehicleRepo {
	return &MockVehicleRepo{
		Vehicles:       make(map[string]*Vehicle),
		ParkingRecords: make(map[string]*ParkingRecord),
		Devices:        make(map[string]*Device),
		Lanes:          make(map[string]*Lane),
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

func (m *MockVehicleRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

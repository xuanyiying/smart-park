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
	Blacklist      map[string]*BlacklistEntry
	LotCapacity    *int
}

func NewMockVehicleRepo() *MockVehicleRepo {
	return &MockVehicleRepo{
		Vehicles:       make(map[string]*Vehicle),
		ParkingRecords: make(map[string]*ParkingRecord),
		Devices:        make(map[string]*Device),
		Lanes:          make(map[string]*Lane),
		Blacklist:      make(map[string]*BlacklistEntry),
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

func (m *MockVehicleRepo) ListDevices(ctx context.Context, page, pageSize int) ([]*Device, int, error) {
	var devices []*Device
	for _, d := range m.Devices {
		devices = append(devices, d)
	}
	return devices, len(devices), nil
}

func (m *MockVehicleRepo) ListParkingRecordsByPlates(ctx context.Context, plateNumbers []string, page, pageSize int) ([]*ParkingRecord, int, error) {
	return nil, 0, nil
}

func (m *MockVehicleRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func (m *MockVehicleRepo) CreateOfflineSyncRecord(ctx context.Context, record *OfflineSyncRecord) error {
	return nil
}

func (m *MockVehicleRepo) GetPendingSyncRecords(ctx context.Context, limit int) ([]*OfflineSyncRecord, error) {
	return nil, nil
}

func (m *MockVehicleRepo) UpdateOfflineSyncRecord(ctx context.Context, record *OfflineSyncRecord) error {
	return nil
}

func (m *MockVehicleRepo) GetDeviceByID(ctx context.Context, deviceID string) (*Device, error) {
	return m.Devices[deviceID], nil
}

func (m *MockVehicleRepo) CreateDevice(ctx context.Context, device *Device) error {
	m.Devices[device.DeviceID] = device
	return nil
}

func (m *MockVehicleRepo) UpdateDevice(ctx context.Context, device *Device) error {
	m.Devices[device.DeviceID] = device
	return nil
}

func (m *MockVehicleRepo) DeleteDevice(ctx context.Context, deviceID string) error {
	delete(m.Devices, deviceID)
	return nil
}

func (m *MockVehicleRepo) SeedData(ctx context.Context, lotID uuid.UUID) error {
	return nil
}

// Stub implementations for extended VehicleRepo interface
func (m *MockVehicleRepo) CreateManufacturer(ctx context.Context, manufacturer *Manufacturer) error       { return nil }
func (m *MockVehicleRepo) GetManufacturer(ctx context.Context, id uuid.UUID) (*Manufacturer, error)      { return nil, nil }
func (m *MockVehicleRepo) UpdateManufacturer(ctx context.Context, manufacturer *Manufacturer) error      { return nil }
func (m *MockVehicleRepo) DeleteManufacturer(ctx context.Context, id uuid.UUID) error                    { return nil }
func (m *MockVehicleRepo) ListManufacturers(ctx context.Context, page, pageSize int) ([]*Manufacturer, int, error) {
	return nil, 0, nil
}
func (m *MockVehicleRepo) CreateFirmware(ctx context.Context, firmware *Firmware) error                  { return nil }
func (m *MockVehicleRepo) GetFirmware(ctx context.Context, id uuid.UUID) (*Firmware, error)             { return nil, nil }
func (m *MockVehicleRepo) GetFirmwareByID(ctx context.Context, firmwareID string) (*Firmware, error)     { return nil, nil }
func (m *MockVehicleRepo) UpdateFirmware(ctx context.Context, firmware *Firmware) error                 { return nil }
func (m *MockVehicleRepo) DeleteFirmware(ctx context.Context, id uuid.UUID) error                       { return nil }
func (m *MockVehicleRepo) ListFirmwares(ctx context.Context, manufacturer, model string, page, pageSize int) ([]*Firmware, int, error) {
	return nil, 0, nil
}
func (m *MockVehicleRepo) GetLatestFirmware(ctx context.Context, manufacturer, model string) (*Firmware, error) { return nil, nil }
func (m *MockVehicleRepo) CreateDevicePerformance(ctx context.Context, performance *DevicePerformance) error { return nil }
func (m *MockVehicleRepo) GetDevicePerformance(ctx context.Context, deviceID string, startTime, endTime time.Time) ([]*DevicePerformance, error) {
	return nil, nil
}
func (m *MockVehicleRepo) GetDevicePerformanceLatest(ctx context.Context, deviceID string) (*DevicePerformance, error) { return nil, nil }
func (m *MockVehicleRepo) CreateDeviceFault(ctx context.Context, fault *DeviceFault) error               { return nil }
func (m *MockVehicleRepo) GetDeviceFault(ctx context.Context, id uuid.UUID) (*DeviceFault, error)        { return nil, nil }
func (m *MockVehicleRepo) UpdateDeviceFault(ctx context.Context, fault *DeviceFault) error               { return nil }
func (m *MockVehicleRepo) ListDeviceFaults(ctx context.Context, deviceID, status string, page, pageSize int) ([]*DeviceFault, int, error) {
	return nil, 0, nil
}
func (m *MockVehicleRepo) ResolveDeviceFault(ctx context.Context, id uuid.UUID) error                   { return nil }
func (m *MockVehicleRepo) GetDeviceUsageStats(ctx context.Context, deviceID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	return nil, nil
}
func (m *MockVehicleRepo) GetDeviceFaultStats(ctx context.Context, deviceID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	return nil, nil
}
func (m *MockVehicleRepo) GetDeviceStatsSummary(ctx context.Context, deviceID string) (map[string]interface{}, error) { return nil, nil }
func (m *MockVehicleRepo) UpdateDeviceStatus(ctx context.Context, deviceID, status string) error         { return nil }
func (m *MockVehicleRepo) UpdateDeviceVersion(ctx context.Context, deviceID, firmwareVersion, hardwareVersion string) error { return nil }
func (m *MockVehicleRepo) UpdateDeviceStats(ctx context.Context, deviceID string, stats map[string]string) error { return nil }
func (m *MockVehicleRepo) CreateDeviceLog(ctx context.Context, log *DeviceLog) error                     { return nil }
func (m *MockVehicleRepo) GetDeviceLogs(ctx context.Context, deviceID string, page, pageSize int) ([]*DeviceLog, int, error) {
	return nil, 0, nil
}
func (m *MockVehicleRepo) CreateDeviceUpgrade(ctx context.Context, deviceID, fromVersion, toVersion, firmwareURL string) (uuid.UUID, error) {
	return uuid.New(), nil
}
func (m *MockVehicleRepo) GetDeviceUpgrade(ctx context.Context, id uuid.UUID) (*DeviceUpgrade, error)    { return nil, nil }
func (m *MockVehicleRepo) UpdateDeviceUpgradeStatus(ctx context.Context, id uuid.UUID, status, errMsg string) error { return nil }

// ---- Blacklist ----

func (m *MockVehicleRepo) CreateBlacklistEntry(ctx context.Context, entry *BlacklistEntry) error {
	m.Blacklist[entry.PlateNumber] = entry
	return nil
}

func (m *MockVehicleRepo) GetBlacklistEntry(ctx context.Context, plateNumber string) (*BlacklistEntry, error) {
	if entry, ok := m.Blacklist[plateNumber]; ok {
		return entry, nil
	}
	return nil, nil
}

func (m *MockVehicleRepo) SetBlacklistEntryActive(ctx context.Context, plateNumber string, active bool) error {
	if entry, ok := m.Blacklist[plateNumber]; ok {
		entry.Active = active
	}
	return nil
}

func (m *MockVehicleRepo) ListBlacklistEntries(ctx context.Context, page, pageSize int) ([]*BlacklistEntry, int, error) {
	var result []*BlacklistEntry
	for _, entry := range m.Blacklist {
		result = append(result, entry)
	}
	return result, len(result), nil
}

// ---- Parking lot capacity ----

func (m *MockVehicleRepo) GetLotCapacity(ctx context.Context, lotID uuid.UUID) (int, error) {
	if m.LotCapacity != nil {
		return *m.LotCapacity, nil
	}
	return 0, nil
}

func (m *MockVehicleRepo) CountActiveRecordsByLot(ctx context.Context, lotID uuid.UUID) (int, error) {
	count := 0
	for _, record := range m.ParkingRecords {
		if record.LotID == lotID {
			count++
		}
	}
	return count, nil
}

// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Constants for vehicle service.
const (
	LockTypeEntry = "entry"
	LockTypeExit  = "exit"

	VehicleTypeTemporary = "temporary"
	VehicleTypeMonthly   = "monthly"
	VehicleTypeVIP       = "vip"

	RecordStatusEntry = "entry"

	SyncStatusPending = "pending"
	SyncStatusSynced  = "synced"
	SyncStatusFailed  = "failed"
)

// OfflineSyncRecord represents an offline sync record for device operations.
type OfflineSyncRecord struct {
	ID         uuid.UUID
	OfflineID  string
	RecordID   uuid.UUID
	LotID      uuid.UUID
	DeviceID   string
	GateID     string
	OpenTime   time.Time
	SyncAmount float64
	SyncStatus string
	SyncError  string
	RetryCount int
	SyncedAt   *time.Time
	CreatedAt  time.Time
}

const (
	RecordStatusExiting = "exiting"
	ExitStatusUnpaid    = "unpaid"
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
	ID                uuid.UUID
	LotID             uuid.UUID
	EntryLaneID       uuid.UUID
	VehicleID         *uuid.UUID
	PlateNumber       *string
	PlateNumberSource string
	EntryTime         time.Time
	EntryImageURL     string
	RecordStatus      string
	ExitTime          *time.Time
	ExitImageURL      string
	ExitLaneID        *uuid.UUID
	ExitDeviceID      string
	ParkingDuration   int
	ExitStatus        string
	PaymentLock       int
	Metadata          map[string]interface{}
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Device represents a device entity in business logic.
type Device struct {
	ID                  uuid.UUID
	DeviceID            string
	LotID               *uuid.UUID
	LaneID              *uuid.UUID
	DeviceType          string
	DeviceSecret        string
	Manufacturer        string
	Model               string
	FirmwareVersion     string
	VendorSpecificConfig map[string]interface{}
	GateID              string
	Enabled             bool
	Status              string
	LastHeartbeat       *time.Time
	LastOnline          *time.Time
	FaultInfo           string
	HeartbeatCount      int
	OfflineCount        int
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

// Manufacturer represents a manufacturer entity in business logic.
type Manufacturer struct {
	ID          uuid.UUID
	Name        string
	Website     string
	ContactInfo string
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Firmware represents a firmware entity in business logic.
type Firmware struct {
	ID          uuid.UUID
	FirmwareID  string
	Manufacturer string
	Model       string
	Version     string
	URL         string
	Size        int64
	MD5         string
	Description string
	Status      string
	ReleaseDate time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DevicePerformance represents device performance metrics.
type DevicePerformance struct {
	ID           uuid.UUID
	DeviceID     string
	CPUUsage     float64
	MemoryUsage  float64
	StorageUsage float64
	NetworkIn    int64
	NetworkOut   int64
	Temperature  float64
	Timestamp    time.Time
	CreatedAt    time.Time
}

// DeviceFault represents a device fault record.
type DeviceFault struct {
	ID          uuid.UUID
	DeviceID    string
	FaultType   string
	FaultCode   string
	Description string
	Severity    string
	Status      string
	Suggestion  string
	DetectedAt  time.Time
	ResolvedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
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
	ListParkingRecordsByPlates(ctx context.Context, plateNumbers []string, page, pageSize int) ([]*ParkingRecord, int, error)
	GetDeviceByCode(ctx context.Context, deviceCode string) (*Device, error)
	GetDeviceByID(ctx context.Context, deviceID string) (*Device, error)
	UpdateDeviceHeartbeat(ctx context.Context, deviceCode string) error
	ListDevices(ctx context.Context, page, pageSize int) ([]*Device, int, error)
	CreateDevice(ctx context.Context, device *Device) error
	UpdateDevice(ctx context.Context, device *Device) error
	DeleteDevice(ctx context.Context, deviceID string) error
	GetLaneByDeviceCode(ctx context.Context, deviceCode string) (*Lane, error)
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
	CreateOfflineSyncRecord(ctx context.Context, record *OfflineSyncRecord) error
	GetPendingSyncRecords(ctx context.Context, limit int) ([]*OfflineSyncRecord, error)
	UpdateOfflineSyncRecord(ctx context.Context, record *OfflineSyncRecord) error
	SeedData(ctx context.Context) error
	// Manufacturer management
	CreateManufacturer(ctx context.Context, manufacturer *Manufacturer) error
	GetManufacturer(ctx context.Context, id uuid.UUID) (*Manufacturer, error)
	UpdateManufacturer(ctx context.Context, manufacturer *Manufacturer) error
	DeleteManufacturer(ctx context.Context, id uuid.UUID) error
	ListManufacturers(ctx context.Context, page, pageSize int) ([]*Manufacturer, int, error)
	// Firmware management
	CreateFirmware(ctx context.Context, firmware *Firmware) error
	GetFirmware(ctx context.Context, id uuid.UUID) (*Firmware, error)
	GetFirmwareByID(ctx context.Context, firmwareID string) (*Firmware, error)
	UpdateFirmware(ctx context.Context, firmware *Firmware) error
	DeleteFirmware(ctx context.Context, id uuid.UUID) error
	ListFirmwares(ctx context.Context, manufacturer, model string, page, pageSize int) ([]*Firmware, int, error)
	GetLatestFirmware(ctx context.Context, manufacturer, model string) (*Firmware, error)
	// Device performance monitoring
	CreateDevicePerformance(ctx context.Context, performance *DevicePerformance) error
	GetDevicePerformance(ctx context.Context, deviceID string, startTime, endTime time.Time) ([]*DevicePerformance, error)
	GetDevicePerformanceLatest(ctx context.Context, deviceID string) (*DevicePerformance, error)
	// Device fault diagnosis
	CreateDeviceFault(ctx context.Context, fault *DeviceFault) error
	GetDeviceFault(ctx context.Context, id uuid.UUID) (*DeviceFault, error)
	UpdateDeviceFault(ctx context.Context, fault *DeviceFault) error
	ListDeviceFaults(ctx context.Context, deviceID string, status string, page, pageSize int) ([]*DeviceFault, int, error)
	ResolveDeviceFault(ctx context.Context, id uuid.UUID) error
	// Device statistics
	GetDeviceUsageStats(ctx context.Context, deviceID string, startTime, endTime time.Time) (map[string]interface{}, error)
	GetDeviceFaultStats(ctx context.Context, deviceID string, startTime, endTime time.Time) (map[string]interface{}, error)
	GetDeviceStatsSummary(ctx context.Context, deviceID string) (map[string]interface{}, error)
}

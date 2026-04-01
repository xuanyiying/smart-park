// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ParkingSpace represents a parking space entity
type ParkingSpace struct {
	ID           uuid.UUID  `json:"id"`
	SpaceID      string     `json:"space_id"`
	LotID        *uuid.UUID `json:"lot_id"`
	DeviceID     string     `json:"device_id"`
	Status       string     `json:"status"` // available, occupied, reserved, maintenance
	LastUpdate   time.Time  `json:"last_update"`
	VehiclePlate *string    `json:"vehicle_plate,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

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
	// Parking space operations
	UpdateParkingSpaceStatus(ctx context.Context, space *ParkingSpace) error
	GetParkingSpaceByID(ctx context.Context, spaceID string) (*ParkingSpace, error)
	ListParkingSpaces(ctx context.Context, lotID *uuid.UUID, page, pageSize int) ([]*ParkingSpace, int, error)
}

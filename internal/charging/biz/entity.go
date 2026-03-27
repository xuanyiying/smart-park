// Package biz provides business logic for the charging service.
package biz

import (
	"time"

	"github.com/google/uuid"
)

// ConnectorType represents the type of connector.
type ConnectorType string

const (
	ConnectorTypeAC     ConnectorType = "ac"
	ConnectorTypeDC     ConnectorType = "dc"
	ConnectorTypeFastDC ConnectorType = "fast_dc"
)

// StationStatus represents the status of a charging station.
type StationStatus string

const (
	StationStatusAvailable   StationStatus = "available"
	StationStatusOffline     StationStatus = "offline"
	StationStatusMaintenance StationStatus = "maintenance"
)

// ConnectorStatus represents the status of a connector.
type ConnectorStatus string

const (
	ConnectorStatusAvailable ConnectorStatus = "available"
	ConnectorStatusCharging  ConnectorStatus = "charging"
	ConnectorStatusFaulted   ConnectorStatus = "faulted"
	ConnectorStatusOffline   ConnectorStatus = "offline"
)

// SessionStatus represents the status of a charging session.
type SessionStatus string

const (
	SessionStatusPending   SessionStatus = "pending"
	SessionStatusCharging  SessionStatus = "charging"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusCancelled SessionStatus = "cancelled"
	SessionStatusExpired   SessionStatus = "expired"
)

// PaymentStatus represents the payment status of a session.
type PaymentStatus string

const (
	PaymentStatusPending  PaymentStatus = "pending"
	PaymentStatusPaid     PaymentStatus = "paid"
	PaymentStatusRefunded PaymentStatus = "refunded"
	PaymentStatusFailed   PaymentStatus = "failed"
)

// Station represents a charging station.
type Station struct {
	ID                  uuid.UUID
	LotID               uuid.UUID
	Name                string
	StationType         ConnectorType
	Status              StationStatus
	ConnectorType       ConnectorType
	MaxPower            float64
	Voltage             float64
	TotalConnectors     int
	AvailableConnectors int
	Location            string
	Floor               string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// HasAvailableConnector checks if the station has available connectors.
func (s *Station) HasAvailableConnector() bool {
	return s.AvailableConnectors > 0 && s.Status == StationStatusAvailable
}

// Connector represents a charging connector.
type Connector struct {
	ID        uuid.UUID
	StationID uuid.UUID
	Number    int
	Type      ConnectorType
	Status    ConnectorStatus
	MaxPower  float64
	Voltage   float64
	Current   float64
	FaultCode string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Session represents a charging session.
type Session struct {
	ID            uuid.UUID
	StationID     uuid.UUID
	ConnectorID   uuid.UUID
	UserID        uuid.UUID
	VehiclePlate  string
	StartTime     time.Time
	EndTime       *time.Time
	StartEnergy   float64
	EndEnergy     float64
	ChargedEnergy float64
	Cost          float64
	ServiceFee    float64
	TotalAmount   float64
	Status        SessionStatus
	PaymentStatus PaymentStatus
	PayTime       *time.Time
	PaymentMethod string
	TransactionID string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Duration returns the duration of the session in hours.
func (s *Session) Duration() float64 {
	endTime := time.Now()
	if s.EndTime != nil {
		endTime = *s.EndTime
	}
	return endTime.Sub(s.StartTime).Hours()
}

// Price represents a price configuration for charging.
type Price struct {
	ID          uuid.UUID
	StationID   uuid.UUID
	Name        string
	StartHour   int
	EndHour     int
	PricePerKWh float64
	ServiceFee  float64
	PeakLoad    float64
	OffPeakLoad float64
	IsPeakHours bool
	EffectiveAt time.Time
	ExpiresAt   time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ChargingSummary represents a summary of a charging session.
type ChargingSummary struct {
	SessionID     uuid.UUID
	StationName   string
	ConnectorNum  int
	VehiclePlate  string
	StartTime     time.Time
	EndTime       time.Time
	Duration      float64
	ChargedEnergy float64
	Cost          float64
	ServiceFee    float64
	TotalAmount   float64
	PaymentStatus string
}

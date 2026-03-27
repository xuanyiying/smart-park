// Package biz provides business logic for the charging service.
package biz

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// Config holds charging service configuration.
type Config struct {
	DefaultServiceFee     float64
	MaxSessionDuration    time.Duration
	DefaultPeakLoadKWh    float64
	DefaultOffPeakLoadKWh float64
}

// DefaultConfig returns default charging configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultServiceFee:     0.8,
		MaxSessionDuration:    8 * time.Hour,
		DefaultPeakLoadKWh:    1.5,
		DefaultOffPeakLoadKWh: 0.8,
	}
}

// ChargingRepo defines the repository interface for charging operations.
type ChargingRepo interface {
	CreateStation(ctx context.Context, station *Station) error
	GetStation(ctx context.Context, stationID uuid.UUID) (*Station, error)
	UpdateStation(ctx context.Context, station *Station) error
	DeleteStation(ctx context.Context, stationID uuid.UUID) error
	ListStations(ctx context.Context, lotID uuid.UUID) ([]*Station, error)

	CreateConnector(ctx context.Context, connector *Connector) error
	GetConnector(ctx context.Context, connectorID uuid.UUID) (*Connector, error)
	UpdateConnector(ctx context.Context, connector *Connector) error
	DeleteConnector(ctx context.Context, connectorID uuid.UUID) error
	ListConnectors(ctx context.Context, stationID uuid.UUID) ([]*Connector, error)
	LockConnector(ctx context.Context, connectorID uuid.UUID) error
	UnlockConnector(ctx context.Context, connectorID uuid.UUID) error

	CreateSession(ctx context.Context, session *Session) error
	GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	GetActiveSession(ctx context.Context, connectorID uuid.UUID) (*Session, error)
	ListUserSessions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*Session, int64, error)
	ListStationSessions(ctx context.Context, stationID uuid.UUID, page, pageSize int) ([]*Session, int64, error)
	ExpireOldSessions(ctx context.Context, threshold time.Duration) (int64, error)

	CreatePrice(ctx context.Context, price *Price) error
	GetPrice(ctx context.Context, priceID uuid.UUID) (*Price, error)
	UpdatePrice(ctx context.Context, price *Price) error
	GetCurrentPrice(ctx context.Context, stationID uuid.UUID) (*Price, error)
	ListPrices(ctx context.Context, stationID uuid.UUID) ([]*Price, error)
}

// ChargingUseCase implements charging business logic.
type ChargingUseCase struct {
	repo   ChargingRepo
	config *Config
	log    *log.Helper
}

// NewChargingUseCase creates a new ChargingUseCase.
func NewChargingUseCase(repo ChargingRepo, logger log.Logger) *ChargingUseCase {
	return &ChargingUseCase{
		repo:   repo,
		config: DefaultConfig(),
		log:    log.NewHelper(logger),
	}
}

// NewChargingUseCaseWithConfig creates a new ChargingUseCase with custom config.
func NewChargingUseCaseWithConfig(repo ChargingRepo, config *Config, logger log.Logger) *ChargingUseCase {
	if config == nil {
		config = DefaultConfig()
	}
	return &ChargingUseCase{
		repo:   repo,
		config: config,
		log:    log.NewHelper(logger),
	}
}

// CreateStation creates a new charging station.
func (uc *ChargingUseCase) CreateStation(ctx context.Context, lotID uuid.UUID, name string, stationType, connectorType ConnectorType, maxPower float64, voltage float64, totalConnectors int, location, floor string) (*Station, error) {
	if lotID == uuid.Nil {
		return nil, fmt.Errorf("lot ID is required")
	}
	if name == "" {
		return nil, fmt.Errorf("station name is required")
	}
	if maxPower <= 0 {
		return nil, ErrInvalidPowerValue
	}
	if totalConnectors <= 0 {
		totalConnectors = 1
	}

	station := &Station{
		ID:                  uuid.New(),
		LotID:               lotID,
		Name:                name,
		StationType:         stationType,
		Status:              StationStatusAvailable,
		ConnectorType:       connectorType,
		MaxPower:            maxPower,
		Voltage:             voltage,
		TotalConnectors:     totalConnectors,
		AvailableConnectors: totalConnectors,
		Location:            location,
		Floor:               floor,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	if err := uc.repo.CreateStation(ctx, station); err != nil {
		return nil, fmt.Errorf("failed to create station: %w", err)
	}

	uc.log.Infow("station created", "station_id", station.ID, "name", name, "connectors", totalConnectors)
	return station, nil
}

// GetStation retrieves a charging station by ID.
func (uc *ChargingUseCase) GetStation(ctx context.Context, stationID uuid.UUID) (*Station, error) {
	station, err := uc.repo.GetStation(ctx, stationID)
	if err != nil {
		return nil, ErrStationNotFound
	}
	return station, nil
}

// ListStations retrieves charging stations by lot ID.
func (uc *ChargingUseCase) ListStations(ctx context.Context, lotID uuid.UUID) ([]*Station, error) {
	stations, err := uc.repo.ListStations(ctx, lotID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stations: %w", err)
	}
	return stations, nil
}

// GetAvailableStations retrieves available charging stations.
func (uc *ChargingUseCase) GetAvailableStations(ctx context.Context, lotID uuid.UUID) ([]*Station, error) {
	stations, err := uc.repo.ListStations(ctx, lotID)
	if err != nil {
		return nil, fmt.Errorf("failed to list stations: %w", err)
	}

	var available []*Station
	for _, s := range stations {
		if s.HasAvailableConnector() {
			available = append(available, s)
		}
	}
	return available, nil
}

// UpdateStationStatus updates a station's status.
func (uc *ChargingUseCase) UpdateStationStatus(ctx context.Context, stationID uuid.UUID, status StationStatus) error {
	station, err := uc.repo.GetStation(ctx, stationID)
	if err != nil {
		return ErrStationNotFound
	}

	station.Status = status
	station.UpdatedAt = time.Now()

	if err := uc.repo.UpdateStation(ctx, station); err != nil {
		return fmt.Errorf("failed to update station status: %w", err)
	}

	uc.log.Infow("station status updated", "station_id", stationID, "status", status)
	return nil
}

// DeleteStation deletes a charging station.
func (uc *ChargingUseCase) DeleteStation(ctx context.Context, stationID uuid.UUID) error {
	if err := uc.repo.DeleteStation(ctx, stationID); err != nil {
		return fmt.Errorf("failed to delete station: %w", err)
	}
	return nil
}

// StartCharging starts a new charging session.
func (uc *ChargingUseCase) StartCharging(ctx context.Context, stationID, connectorID, userID uuid.UUID, vehiclePlate string) (*Session, error) {
	if stationID == uuid.Nil || connectorID == uuid.Nil || userID == uuid.Nil {
		return nil, fmt.Errorf("invalid station, connector or user ID")
	}
	if vehiclePlate == "" {
		return nil, fmt.Errorf("vehicle plate is required")
	}

	station, err := uc.repo.GetStation(ctx, stationID)
	if err != nil {
		return nil, ErrStationNotFound
	}
	if station.Status != StationStatusAvailable {
		return nil, ErrStationNotAvailable
	}

	connector, err := uc.repo.GetConnector(ctx, connectorID)
	if err != nil {
		return nil, ErrConnectorNotFound
	}
	if connector.Status != ConnectorStatusAvailable {
		return nil, ErrConnectorNotAvailable
	}

	activeSession, _ := uc.repo.GetActiveSession(ctx, connectorID)
	if activeSession != nil {
		return nil, ErrConnectorInUse
	}

	if err := uc.repo.LockConnector(ctx, connectorID); err != nil {
		return nil, fmt.Errorf("failed to lock connector: %w", err)
	}

	session := &Session{
		ID:            uuid.New(),
		StationID:     stationID,
		ConnectorID:   connectorID,
		UserID:        userID,
		VehiclePlate:  vehiclePlate,
		StartTime:     time.Now(),
		StartEnergy:   0,
		ChargedEnergy: 0,
		Cost:          0,
		ServiceFee:    0,
		TotalAmount:   0,
		Status:        SessionStatusCharging,
		PaymentStatus: PaymentStatusPending,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := uc.repo.CreateSession(ctx, session); err != nil {
		uc.repo.UnlockConnector(ctx, connectorID)
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	connector.Status = ConnectorStatusCharging
	if err := uc.repo.UpdateConnector(ctx, connector); err != nil {
		uc.log.Warnf("failed to update connector status: %v", err)
	}

	station.AvailableConnectors--
	if err := uc.repo.UpdateStation(ctx, station); err != nil {
		uc.log.Warnf("failed to update station available connectors: %v", err)
	}

	uc.log.Infow("charging session started", "session_id", session.ID, "station_id", stationID, "connector_id", connectorID, "user_id", userID)
	return session, nil
}

// StopCharging stops a charging session.
func (uc *ChargingUseCase) StopCharging(ctx context.Context, sessionID, userID uuid.UUID) (*Session, error) {
	session, err := uc.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	if session.UserID != userID {
		return nil, ErrUnauthorizedAccess
	}
	if session.Status != SessionStatusCharging {
		return nil, ErrSessionNotActive
	}

	now := time.Now()
	session.EndTime = &now
	session.Status = SessionStatusCompleted

	duration := session.Duration()
	if duration > uc.config.MaxSessionDuration.Hours() {
		session.Status = SessionStatusExpired
		uc.log.Warnf("session expired due to max duration", "session_id", sessionID, "duration", duration)
	}

	chargedEnergy := session.ChargedEnergy
	if chargedEnergy <= 0 {
		chargedEnergy = uc.estimateEnergy(ctx, session.StartTime, now, session.StationID)
		session.ChargedEnergy = chargedEnergy
	}

	price, err := uc.repo.GetCurrentPrice(ctx, session.StationID)
	if err != nil {
		uc.log.Warnf("failed to get current price, using default: %v", err)
		price = uc.defaultPrice(session.StationID)
	}

	session.Cost = uc.calculateEnergyCost(chargedEnergy, price)
	session.ServiceFee = uc.config.DefaultServiceFee
	session.TotalAmount = session.Cost + session.ServiceFee

	if session.Status == SessionStatusCompleted {
		session.PaymentStatus = PaymentStatusPending
	} else {
		session.PaymentStatus = PaymentStatusFailed
	}

	if err := uc.repo.UpdateSession(ctx, session); err != nil {
		return nil, fmt.Errorf("failed to update session: %w", err)
	}

	if err := uc.repo.UnlockConnector(ctx, session.ConnectorID); err != nil {
		uc.log.Warnf("failed to unlock connector: %v", err)
	}

	station, err := uc.repo.GetStation(ctx, session.StationID)
	if err == nil && station.AvailableConnectors < station.TotalConnectors {
		station.AvailableConnectors++
		uc.repo.UpdateStation(ctx, station)
	}

	uc.log.Infow("charging session stopped", "session_id", sessionID, "energy", chargedEnergy, "cost", session.Cost)
	return session, nil
}

// UpdateChargingProgress updates charging session progress.
func (uc *ChargingUseCase) UpdateChargingProgress(ctx context.Context, sessionID uuid.UUID, currentEnergy, power, voltage, current float64) error {
	if currentEnergy < 0 {
		return ErrInvalidEnergyValue
	}

	session, err := uc.repo.GetSession(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	if session.Status != SessionStatusCharging {
		return ErrSessionNotActive
	}

	session.ChargedEnergy = currentEnergy
	session.UpdatedAt = time.Now()

	duration := session.Duration()
	if duration > uc.config.MaxSessionDuration.Hours() {
		uc.log.Warnf("session approaching max duration", "session_id", sessionID)
	}

	return uc.repo.UpdateSession(ctx, session)
}

// GetActiveSession retrieves the active session for a connector.
func (uc *ChargingUseCase) GetActiveSession(ctx context.Context, connectorID uuid.UUID) (*Session, error) {
	return uc.repo.GetActiveSession(ctx, connectorID)
}

// GetSession retrieves a charging session by ID.
func (uc *ChargingUseCase) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	session, err := uc.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	return session, nil
}

// GetUserSessions retrieves sessions for a user with pagination.
func (uc *ChargingUseCase) GetUserSessions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*Session, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return uc.repo.ListUserSessions(ctx, userID, page, pageSize)
}

// GetStationSessions retrieves sessions for a station with pagination.
func (uc *ChargingUseCase) GetStationSessions(ctx context.Context, stationID uuid.UUID, page, pageSize int) ([]*Session, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}
	return uc.repo.ListStationSessions(ctx, stationID, page, pageSize)
}

// ConfirmPayment confirms payment for a charging session.
func (uc *ChargingUseCase) ConfirmPayment(ctx context.Context, sessionID uuid.UUID, transactionID, paymentMethod string, paidAmount float64) error {
	session, err := uc.repo.GetSession(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	if session.PaymentStatus != PaymentStatusPending {
		return fmt.Errorf("session payment already processed")
	}

	if paidAmount < session.TotalAmount {
		return fmt.Errorf("insufficient payment amount")
	}

	now := time.Now()
	session.PaymentStatus = PaymentStatusPaid
	session.PayTime = &now
	session.TransactionID = transactionID
	session.PaymentMethod = paymentMethod

	if err := uc.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	uc.log.Infow("payment confirmed", "session_id", sessionID, "transaction_id", transactionID, "amount", paidAmount)
	return nil
}

// RefundPayment refunds payment for a charging session.
func (uc *ChargingUseCase) RefundPayment(ctx context.Context, sessionID uuid.UUID, reason string) error {
	session, err := uc.repo.GetSession(ctx, sessionID)
	if err != nil {
		return ErrSessionNotFound
	}

	if session.PaymentStatus != PaymentStatusPaid {
		return fmt.Errorf("can only refund paid sessions")
	}

	session.PaymentStatus = PaymentStatusRefunded
	session.UpdatedAt = time.Now()

	if err := uc.repo.UpdateSession(ctx, session); err != nil {
		return fmt.Errorf("failed to refund: %w", err)
	}

	uc.log.Infow("payment refunded", "session_id", sessionID, "reason", reason)
	return nil
}

// GetChargingSummary retrieves a charging summary for a session.
func (uc *ChargingUseCase) GetChargingSummary(ctx context.Context, sessionID uuid.UUID) (*ChargingSummary, error) {
	session, err := uc.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	station, err := uc.repo.GetStation(ctx, session.StationID)
	if err != nil {
		return nil, err
	}

	connector, err := uc.repo.GetConnector(ctx, session.ConnectorID)
	if err != nil {
		return nil, err
	}

	endTime := session.EndTime
	if endTime == nil {
		now := time.Now()
		endTime = &now
	}

	return &ChargingSummary{
		SessionID:     session.ID,
		StationName:   station.Name,
		ConnectorNum:  connector.Number,
		VehiclePlate:  session.VehiclePlate,
		StartTime:     session.StartTime,
		EndTime:       *endTime,
		Duration:      session.Duration(),
		ChargedEnergy: session.ChargedEnergy,
		Cost:          session.Cost,
		ServiceFee:    session.ServiceFee,
		TotalAmount:   session.TotalAmount,
		PaymentStatus: string(session.PaymentStatus),
	}, nil
}

// CreatePrice creates a new price configuration.
func (uc *ChargingUseCase) CreatePrice(ctx context.Context, stationID uuid.UUID, name string, startHour, endHour int, pricePerKWh, serviceFee, peakLoad, offPeakLoad float64, effectiveAt, expiresAt time.Time) (*Price, error) {
	if startHour < 0 || startHour > 23 || endHour < 0 || endHour > 23 {
		return nil, fmt.Errorf("invalid hour range")
	}
	if pricePerKWh <= 0 {
		return nil, fmt.Errorf("invalid price per kWh")
	}

	price := &Price{
		ID:          uuid.New(),
		StationID:   stationID,
		Name:        name,
		StartHour:   startHour,
		EndHour:     endHour,
		PricePerKWh: pricePerKWh,
		ServiceFee:  serviceFee,
		PeakLoad:    peakLoad,
		OffPeakLoad: offPeakLoad,
		IsPeakHours: uc.isPeakHours(startHour, endHour),
		EffectiveAt: effectiveAt,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.repo.CreatePrice(ctx, price); err != nil {
		return nil, fmt.Errorf("failed to create price: %w", err)
	}

	return price, nil
}

// GetCurrentPrice retrieves the current effective price for a station.
func (uc *ChargingUseCase) GetCurrentPrice(ctx context.Context, stationID uuid.UUID) (*Price, error) {
	price, err := uc.repo.GetCurrentPrice(ctx, stationID)
	if err != nil {
		return nil, ErrPriceNotConfigured
	}
	return price, nil
}

// calculateEnergyCost calculates the energy cost.
func (uc *ChargingUseCase) calculateEnergyCost(energyKWh float64, price *Price) float64 {
	if price == nil {
		return 0
	}
	baseCost := energyKWh * price.PricePerKWh

	if price.IsPeakHours {
		baseCost += energyKWh * price.PeakLoad
	} else {
		baseCost += energyKWh * price.OffPeakLoad
	}

	return math.Round(baseCost*100) / 100
}

// estimateEnergy estimates energy consumption.
func (uc *ChargingUseCase) estimateEnergy(ctx context.Context, startTime, endTime time.Time, stationID uuid.UUID) float64 {
	duration := endTime.Sub(startTime).Hours()
	price, err := uc.repo.GetCurrentPrice(ctx, stationID)
	if err != nil {
		return duration * 7.0
	}

	power := price.PeakLoad
	if !price.IsPeakHours {
		power = price.OffPeakLoad
	}
	if power <= 0 {
		power = 7.0
	}

	return math.Round(duration*power*100) / 100
}

// defaultPrice returns the default price configuration.
func (uc *ChargingUseCase) defaultPrice(stationID uuid.UUID) *Price {
	return &Price{
		StationID:   stationID,
		PricePerKWh: 1.2,
		ServiceFee:  uc.config.DefaultServiceFee,
		PeakLoad:    uc.config.DefaultPeakLoadKWh,
		OffPeakLoad: uc.config.DefaultOffPeakLoadKWh,
		IsPeakHours: false,
	}
}

// isPeakHours checks if the given hour range is peak hours.
func (uc *ChargingUseCase) isPeakHours(startHour, endHour int) bool {
	peakHours := []struct{ start, end int }{{7, 9}, {17, 21}}
	for _, p := range peakHours {
		if startHour >= p.start && endHour <= p.end && startHour < endHour {
			return true
		}
	}
	return false
}

// ExpireOldSessions expires old charging sessions.
func (uc *ChargingUseCase) ExpireOldSessions(ctx context.Context) (int64, error) {
	return uc.repo.ExpireOldSessions(ctx, uc.config.MaxSessionDuration)
}

// GetConnector retrieves a connector by ID.
func (uc *ChargingUseCase) GetConnector(ctx context.Context, connectorID uuid.UUID) (*Connector, error) {
	return uc.repo.GetConnector(ctx, connectorID)
}

// ListConnectors retrieves connectors by station ID.
func (uc *ChargingUseCase) ListConnectors(ctx context.Context, stationID uuid.UUID) ([]*Connector, error) {
	return uc.repo.ListConnectors(ctx, stationID)
}

// ListPrices retrieves price configurations for a station.
func (uc *ChargingUseCase) ListPrices(ctx context.Context, stationID uuid.UUID) ([]*Price, error) {
	return uc.repo.ListPrices(ctx, stationID)
}

package charging

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type ChargingStation struct {
	ID             uuid.UUID
	LotID          uuid.UUID
	Name           string
	Type           string
	Status         string
	ConnectorType  string
	Power          float64
	CurrentPower   float64
	Voltage        float64
	TotalConnectors int
	AvailableConnectors int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ChargingConnector struct {
	ID         uuid.UUID
	StationID  uuid.UUID
	Type       string
	Status     string
	Power      float64
	Voltage    float64
	Current    float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type ChargingSession struct {
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
	Status        string
	PaymentStatus string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ChargingPrice struct {
	ID            uuid.UUID
	StationID     uuid.UUID
	StartTime     string
	EndTime       string
	PricePerKWh   float64
	ServiceFee    float64
	EffectiveDate time.Time
	CreatedAt     time.Time
}

type ChargingRepo interface {
	CreateStation(ctx context.Context, station *ChargingStation) error
	GetStation(ctx context.Context, stationID uuid.UUID) (*ChargingStation, error)
	UpdateStation(ctx context.Context, station *ChargingStation) error
	DeleteStation(ctx context.Context, stationID uuid.UUID) error
	ListStations(ctx context.Context, lotID uuid.UUID) ([]*ChargingStation, error)

	CreateConnector(ctx context.Context, connector *ChargingConnector) error
	GetConnector(ctx context.Context, connectorID uuid.UUID) (*ChargingConnector, error)
	UpdateConnector(ctx context.Context, connector *ChargingConnector) error
	DeleteConnector(ctx context.Context, connectorID uuid.UUID) error
	ListConnectors(ctx context.Context, stationID uuid.UUID) ([]*ChargingConnector, error)

	CreateSession(ctx context.Context, session *ChargingSession) error
	GetSession(ctx context.Context, sessionID uuid.UUID) (*ChargingSession, error)
	UpdateSession(ctx context.Context, session *ChargingSession) error
	ListSessions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*ChargingSession, int64, error)
	GetActiveSession(ctx context.Context, connectorID uuid.UUID) (*ChargingSession, error)

	CreatePrice(ctx context.Context, price *ChargingPrice) error
	GetCurrentPrice(ctx context.Context, stationID uuid.UUID) (*ChargingPrice, error)
}

type ChargingUseCase struct {
	repo ChargingRepo
	log  *log.Helper
}

func NewChargingUseCase(repo ChargingRepo, logger log.Logger) *ChargingUseCase {
	return &ChargingUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

func (uc *ChargingUseCase) CreateStation(ctx context.Context, lotID uuid.UUID, name, stationType, connectorType string, power float64, totalConnectors int) (*ChargingStation, error) {
	station := &ChargingStation{
		ID:                 uuid.New(),
		LotID:              lotID,
		Name:               name,
		Type:               stationType,
		Status:             "available",
		ConnectorType:      connectorType,
		Power:              power,
		TotalConnectors:    totalConnectors,
		AvailableConnectors: totalConnectors,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := uc.repo.CreateStation(ctx, station); err != nil {
		return nil, err
	}

	return station, nil
}

func (uc *ChargingUseCase) StartCharging(ctx context.Context, stationID, connectorID, userID uuid.UUID, vehiclePlate string) (*ChargingSession, error) {
	connector, err := uc.repo.GetConnector(ctx, connectorID)
	if err != nil {
		return nil, err
	}

	if connector.Status != "available" {
		return nil, ErrConnectorNotAvailable
	}

	activeSession, _ := uc.repo.GetActiveSession(ctx, connectorID)
	if activeSession != nil {
		return nil, ErrConnectorInUse
	}

	session := &ChargingSession{
		ID:           uuid.New(),
		StationID:    stationID,
		ConnectorID:  connectorID,
		UserID:       userID,
		VehiclePlate: vehiclePlate,
		StartTime:    time.Now(),
		StartEnergy:  0,
		Status:       "charging",
		PaymentStatus: "pending",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := uc.repo.CreateSession(ctx, session); err != nil {
		return nil, err
	}

	connector.Status = "charging"
	if err := uc.repo.UpdateConnector(ctx, connector); err != nil {
		return nil, err
	}

	return session, nil
}

func (uc *ChargingUseCase) StopCharging(ctx context.Context, sessionID uuid.UUID) (*ChargingSession, error) {
	session, err := uc.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if session.Status != "charging" {
		return nil, ErrSessionNotActive
	}

	now := time.Now()
	session.EndTime = &now
	session.EndEnergy = session.StartEnergy + session.ChargedEnergy
	session.Status = "completed"

	price, err := uc.repo.GetCurrentPrice(ctx, session.StationID)
	if err != nil {
		uc.log.Warnf("failed to get current price: %v", err)
		price = &ChargingPrice{PricePerKWh: 1.0, ServiceFee: 0.5}
	}

	session.Cost = session.ChargedEnergy*price.PricePerKWh + price.ServiceFee

	if err := uc.repo.UpdateSession(ctx, session); err != nil {
		return nil, err
	}

	connector, err := uc.repo.GetConnector(ctx, session.ConnectorID)
	if err != nil {
		return nil, err
	}

	connector.Status = "available"
	if err := uc.repo.UpdateConnector(ctx, connector); err != nil {
		return nil, err
	}

	return session, nil
}

func (uc *ChargingUseCase) UpdateChargingProgress(ctx context.Context, sessionID uuid.UUID, energy float64, power, voltage, current float64) error {
	session, err := uc.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}

	session.ChargedEnergy = energy
	session.UpdatedAt = time.Now()

	return uc.repo.UpdateSession(ctx, session)
}

func (uc *ChargingUseCase) GetAvailableStations(ctx context.Context, lotID uuid.UUID) ([]*ChargingStation, error) {
	stations, err := uc.repo.ListStations(ctx, lotID)
	if err != nil {
		return nil, err
	}

	var available []*ChargingStation
	for _, station := range stations {
		if station.AvailableConnectors > 0 && station.Status == "available" {
			available = append(available, station)
		}
	}

	return available, nil
}

func (uc *ChargingUseCase) GetUserChargingHistory(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*ChargingSession, int64, error) {
	return uc.repo.ListSessions(ctx, userID, page, pageSize)
}

var (
	ErrConnectorNotAvailable = errors.New("connector not available")
	ErrConnectorInUse        = errors.New("connector already in use")
	ErrSessionNotActive      = errors.New("charging session not active")
)

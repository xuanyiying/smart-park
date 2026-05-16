package biz

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type mockChargingRepo struct {
	mu         sync.Mutex
	stations   map[uuid.UUID]*Station
	connectors map[uuid.UUID]*Connector
	sessions   map[uuid.UUID]*Session
	prices     map[uuid.UUID]*Price
}

func newMockChargingRepo() *mockChargingRepo {
	return &mockChargingRepo{
		stations:   make(map[uuid.UUID]*Station),
		connectors: make(map[uuid.UUID]*Connector),
		sessions:   make(map[uuid.UUID]*Session),
		prices:     make(map[uuid.UUID]*Price),
	}
}

func (m *mockChargingRepo) CreateStation(ctx context.Context, station *Station) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stations[station.ID] = station
	return nil
}

func (m *mockChargingRepo) GetStation(ctx context.Context, stationID uuid.UUID) (*Station, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.stations[stationID]
	if !ok {
		return nil, fmt.Errorf("station not found")
	}
	return s, nil
}

func (m *mockChargingRepo) UpdateStation(ctx context.Context, station *Station) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stations[station.ID] = station
	return nil
}

func (m *mockChargingRepo) DeleteStation(ctx context.Context, stationID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.stations, stationID)
	return nil
}

func (m *mockChargingRepo) ListStations(ctx context.Context, lotID uuid.UUID) ([]*Station, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*Station
	for _, s := range m.stations {
		if s.LotID == lotID {
			result = append(result, s)
		}
	}
	return result, nil
}

func (m *mockChargingRepo) CreateConnector(ctx context.Context, connector *Connector) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectors[connector.ID] = connector
	return nil
}

func (m *mockChargingRepo) GetConnector(ctx context.Context, connectorID uuid.UUID) (*Connector, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c, ok := m.connectors[connectorID]
	if !ok {
		return nil, fmt.Errorf("connector not found")
	}
	return c, nil
}

func (m *mockChargingRepo) UpdateConnector(ctx context.Context, connector *Connector) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connectors[connector.ID] = connector
	return nil
}

func (m *mockChargingRepo) DeleteConnector(ctx context.Context, connectorID uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.connectors, connectorID)
	return nil
}

func (m *mockChargingRepo) ListConnectors(ctx context.Context, stationID uuid.UUID) ([]*Connector, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*Connector
	for _, c := range m.connectors {
		if c.StationID == stationID {
			result = append(result, c)
		}
	}
	return result, nil
}

func (m *mockChargingRepo) LockConnector(ctx context.Context, connectorID uuid.UUID) error {
	return nil
}

func (m *mockChargingRepo) UnlockConnector(ctx context.Context, connectorID uuid.UUID) error {
	return nil
}

func (m *mockChargingRepo) CreateSession(ctx context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
	return nil
}

func (m *mockChargingRepo) GetSession(ctx context.Context, sessionID uuid.UUID) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	s, ok := m.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return s, nil
}

func (m *mockChargingRepo) UpdateSession(ctx context.Context, session *Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = session
	return nil
}

func (m *mockChargingRepo) GetActiveSession(ctx context.Context, connectorID uuid.UUID) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range m.sessions {
		if s.ConnectorID == connectorID && s.Status == SessionStatusCharging {
			return s, nil
		}
	}
	return nil, fmt.Errorf("no active session")
}

func (m *mockChargingRepo) ListUserSessions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*Session, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*Session
	for _, s := range m.sessions {
		if s.UserID == userID {
			result = append(result, s)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockChargingRepo) ListStationSessions(ctx context.Context, stationID uuid.UUID, page, pageSize int) ([]*Session, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*Session
	for _, s := range m.sessions {
		if s.StationID == stationID {
			result = append(result, s)
		}
	}
	return result, int64(len(result)), nil
}

func (m *mockChargingRepo) ListAllSessions(ctx context.Context, page, pageSize int) ([]*Session, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*Session
	for _, s := range m.sessions {
		result = append(result, s)
	}
	return result, int64(len(result)), nil
}

func (m *mockChargingRepo) ExpireOldSessions(ctx context.Context, threshold time.Duration) (int64, error) {
	return 0, nil
}

func (m *mockChargingRepo) CreatePrice(ctx context.Context, price *Price) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prices[price.ID] = price
	return nil
}

func (m *mockChargingRepo) GetPrice(ctx context.Context, priceID uuid.UUID) (*Price, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.prices[priceID]
	if !ok {
		return nil, fmt.Errorf("price not found")
	}
	return p, nil
}

func (m *mockChargingRepo) UpdatePrice(ctx context.Context, price *Price) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.prices[price.ID] = price
	return nil
}

func (m *mockChargingRepo) GetCurrentPrice(ctx context.Context, stationID uuid.UUID) (*Price, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, p := range m.prices {
		if p.StationID == stationID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("price not found")
}

func (m *mockChargingRepo) ListPrices(ctx context.Context, stationID uuid.UUID) ([]*Price, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*Price
	for _, p := range m.prices {
		if p.StationID == stationID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockChargingRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

func setupChargingUseCase() (*ChargingUseCase, *mockChargingRepo) {
	repo := newMockChargingRepo()
	logger := log.NewStdLogger(os.Stdout)
	uc := NewChargingUseCase(repo, logger)
	return uc, repo
}

func TestChargingUseCase_CreateStation(t *testing.T) {
	uc, _ := setupChargingUseCase()
	ctx := context.Background()
	lotID := uuid.New()

	tests := []struct {
		name            string
		lotID           uuid.UUID
		stationName     string
		maxPower        float64
		totalConnectors int
		wantErr         bool
		errContains     string
	}{
		{
			name:        "valid station",
			lotID:       lotID,
			stationName: "Station A",
			maxPower:    120.0,
			totalConnectors: 4,
			wantErr:    false,
		},
		{
			name:        "nil lot ID",
			lotID:       uuid.Nil,
			stationName: "Station B",
			maxPower:    120.0,
			totalConnectors: 2,
			wantErr:    true,
			errContains: "lot ID is required",
		},
		{
			name:        "empty name",
			lotID:       lotID,
			stationName: "",
			maxPower:    120.0,
			totalConnectors: 2,
			wantErr:    true,
			errContains: "station name is required",
		},
		{
			name:        "zero max power",
			lotID:       lotID,
			stationName: "Station C",
			maxPower:    0,
			totalConnectors: 2,
			wantErr:    true,
			errContains: "invalid power value",
		},
		{
			name:        "negative max power",
			lotID:       lotID,
			stationName: "Station D",
			maxPower:    -10.0,
			totalConnectors: 2,
			wantErr:    true,
			errContains: "invalid power value",
		},
		{
			name:        "zero connectors defaults to 1",
			lotID:       lotID,
			stationName: "Station E",
			maxPower:    60.0,
			totalConnectors: 0,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			station, err := uc.CreateStation(ctx, tt.lotID, tt.stationName, ConnectorTypeAC, ConnectorTypeAC, tt.maxPower, 220.0, tt.totalConnectors, "A1", "1F")

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					return
				}
				if tt.errContains != "" && !containsStr(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if station.Name != tt.stationName {
				t.Errorf("station.Name = %s, want %s", station.Name, tt.stationName)
			}
			if station.Status != StationStatusAvailable {
				t.Errorf("station.Status = %s, want available", station.Status)
			}
			if tt.totalConnectors <= 0 {
				if station.TotalConnectors != 1 {
					t.Errorf("station.TotalConnectors = %d, want 1", station.TotalConnectors)
				}
			} else {
				if station.TotalConnectors != tt.totalConnectors {
					t.Errorf("station.TotalConnectors = %d, want %d", station.TotalConnectors, tt.totalConnectors)
				}
			}
		})
	}
}

func TestChargingUseCase_StartCharging(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	stationID := uuid.New()
	connectorID := uuid.New()
	userID := uuid.New()

	station := &Station{
		ID:                  stationID,
		LotID:               uuid.New(),
		Name:                "Test Station",
		Status:              StationStatusAvailable,
		TotalConnectors:     2,
		AvailableConnectors: 2,
	}
	repo.stations[stationID] = station

	connector := &Connector{
		ID:        connectorID,
		StationID: stationID,
		Number:    1,
		Type:      ConnectorTypeAC,
		Status:    ConnectorStatusAvailable,
	}
	repo.connectors[connectorID] = connector

	tests := []struct {
		name        string
		stationID   uuid.UUID
		connectorID uuid.UUID
		userID      uuid.UUID
		plate       string
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid start",
			stationID:   stationID,
			connectorID: connectorID,
			userID:      userID,
			plate:       "京A12345",
			wantErr:     false,
		},
		{
			name:        "nil station ID",
			stationID:   uuid.Nil,
			connectorID: connectorID,
			userID:      userID,
			plate:       "京A12345",
			wantErr:     true,
			errContains: "invalid station",
		},
		{
			name:        "empty plate",
			stationID:   stationID,
			connectorID: connectorID,
			userID:      userID,
			plate:       "",
			wantErr:     true,
			errContains: "vehicle plate is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := uc.StartCharging(ctx, tt.stationID, tt.connectorID, tt.userID, tt.plate)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if session.Status != SessionStatusCharging {
				t.Errorf("session.Status = %s, want charging", session.Status)
			}
			if session.PaymentStatus != PaymentStatusPending {
				t.Errorf("session.PaymentStatus = %s, want pending", session.PaymentStatus)
			}
			if session.VehiclePlate != tt.plate {
				t.Errorf("session.VehiclePlate = %s, want %s", session.VehiclePlate, tt.plate)
			}
		})
	}
}

func TestChargingUseCase_StartCharging_StationNotAvailable(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	stationID := uuid.New()
	connectorID := uuid.New()

	station := &Station{
		ID:                  stationID,
		LotID:               uuid.New(),
		Name:                "Offline Station",
		Status:              StationStatusOffline,
		TotalConnectors:     2,
		AvailableConnectors: 2,
	}
	repo.stations[stationID] = station

	_, err := uc.StartCharging(ctx, stationID, connectorID, uuid.New(), "京A12345")
	if err == nil {
		t.Error("expected error for offline station")
	}
}

func TestChargingUseCase_StartCharging_ConnectorNotAvailable(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	stationID := uuid.New()
	connectorID := uuid.New()

	station := &Station{
		ID:                  stationID,
		LotID:               uuid.New(),
		Name:                "Test Station",
		Status:              StationStatusAvailable,
		TotalConnectors:     2,
		AvailableConnectors: 2,
	}
	repo.stations[stationID] = station

	connector := &Connector{
		ID:        connectorID,
		StationID: stationID,
		Number:    1,
		Type:      ConnectorTypeAC,
		Status:    ConnectorStatusFaulted,
	}
	repo.connectors[connectorID] = connector

	_, err := uc.StartCharging(ctx, stationID, connectorID, uuid.New(), "京A12345")
	if err == nil {
		t.Error("expected error for faulted connector")
	}
}

func TestChargingUseCase_StopCharging(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	stationID := uuid.New()
	connectorID := uuid.New()
	userID := uuid.New()

	station := &Station{
		ID:                  stationID,
		LotID:               uuid.New(),
		Name:                "Test Station",
		Status:              StationStatusAvailable,
		TotalConnectors:     2,
		AvailableConnectors: 1,
	}
	repo.stations[stationID] = station

	connector := &Connector{
		ID:        connectorID,
		StationID: stationID,
		Number:    1,
		Type:      ConnectorTypeAC,
		Status:    ConnectorStatusCharging,
	}
	repo.connectors[connectorID] = connector

	sessionID := uuid.New()
	session := &Session{
		ID:            sessionID,
		StationID:     stationID,
		ConnectorID:   connectorID,
		UserID:        userID,
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-2 * time.Hour),
		ChargedEnergy: 15.0,
		Status:        SessionStatusCharging,
		PaymentStatus: PaymentStatusPending,
	}
	repo.sessions[sessionID] = session

	priceID := uuid.New()
	repo.prices[priceID] = &Price{
		ID:          priceID,
		StationID:   stationID,
		PricePerKWh: 1.2,
		ServiceFee:  0.8,
		PeakLoad:    1.5,
		OffPeakLoad: 0.8,
		IsPeakHours: false,
	}

	updatedSession, err := uc.StopCharging(ctx, sessionID, userID)
	if err != nil {
		t.Fatalf("StopCharging() error: %v", err)
	}

	if updatedSession.Status != SessionStatusCompleted {
		t.Errorf("session.Status = %s, want completed", updatedSession.Status)
	}
	if updatedSession.EndTime == nil {
		t.Error("session.EndTime should not be nil")
	}
	if updatedSession.Cost <= 0 {
		t.Errorf("session.Cost = %f, want > 0", updatedSession.Cost)
	}
	if updatedSession.TotalAmount <= 0 {
		t.Errorf("session.TotalAmount = %f, want > 0", updatedSession.TotalAmount)
	}
}

func TestChargingUseCase_StopCharging_WrongUser(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	userID := uuid.New()
	otherUserID := uuid.New()

	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        userID,
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-1 * time.Hour),
		ChargedEnergy: 10.0,
		Status:        SessionStatusCharging,
		PaymentStatus: PaymentStatusPending,
	}

	_, err := uc.StopCharging(ctx, sessionID, otherUserID)
	if err == nil {
		t.Error("expected error for wrong user")
	}
}

func TestChargingUseCase_StopCharging_NotActive(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	userID := uuid.New()

	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        userID,
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-1 * time.Hour),
		ChargedEnergy: 10.0,
		Status:        SessionStatusCompleted,
		PaymentStatus: PaymentStatusPending,
	}

	_, err := uc.StopCharging(ctx, sessionID, userID)
	if err == nil {
		t.Error("expected error for non-active session")
	}
}

func TestChargingUseCase_UpdateChargingProgress(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        uuid.New(),
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-30 * time.Minute),
		ChargedEnergy: 5.0,
		Status:        SessionStatusCharging,
	}

	err := uc.UpdateChargingProgress(ctx, sessionID, 10.0, 7.0, 220.0, 32.0)
	if err != nil {
		t.Errorf("UpdateChargingProgress() error: %v", err)
	}

	session := repo.sessions[sessionID]
	if session.ChargedEnergy != 10.0 {
		t.Errorf("ChargedEnergy = %f, want 10.0", session.ChargedEnergy)
	}
}

func TestChargingUseCase_UpdateChargingProgress_NegativeEnergy(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        uuid.New(),
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-30 * time.Minute),
		ChargedEnergy: 5.0,
		Status:        SessionStatusCharging,
	}

	err := uc.UpdateChargingProgress(ctx, sessionID, -5.0, 7.0, 220.0, 32.0)
	if err == nil {
		t.Error("expected error for negative energy")
	}
}

func TestChargingUseCase_UpdateChargingProgress_NotActive(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        uuid.New(),
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-1 * time.Hour),
		ChargedEnergy: 5.0,
		Status:        SessionStatusCompleted,
	}

	err := uc.UpdateChargingProgress(ctx, sessionID, 10.0, 7.0, 220.0, 32.0)
	if err == nil {
		t.Error("expected error for non-active session")
	}
}

func TestChargingUseCase_ConfirmPayment(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        uuid.New(),
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-2 * time.Hour),
		ChargedEnergy: 15.0,
		Cost:          18.0,
		ServiceFee:    0.8,
		TotalAmount:   18.8,
		Status:        SessionStatusCompleted,
		PaymentStatus: PaymentStatusPending,
	}

	err := uc.ConfirmPayment(ctx, sessionID, "txn-001", "wechat", 18.8)
	if err != nil {
		t.Errorf("ConfirmPayment() error: %v", err)
	}

	session := repo.sessions[sessionID]
	if session.PaymentStatus != PaymentStatusPaid {
		t.Errorf("PaymentStatus = %s, want paid", session.PaymentStatus)
	}
	if session.TransactionID != "txn-001" {
		t.Errorf("TransactionID = %s, want txn-001", session.TransactionID)
	}
	if session.PaymentMethod != "wechat" {
		t.Errorf("PaymentMethod = %s, want wechat", session.PaymentMethod)
	}
}

func TestChargingUseCase_ConfirmPayment_InsufficientAmount(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        uuid.New(),
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-2 * time.Hour),
		TotalAmount:   18.8,
		Status:        SessionStatusCompleted,
		PaymentStatus: PaymentStatusPending,
	}

	err := uc.ConfirmPayment(ctx, sessionID, "txn-002", "alipay", 10.0)
	if err == nil {
		t.Error("expected error for insufficient payment")
	}
}

func TestChargingUseCase_ConfirmPayment_AlreadyProcessed(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        uuid.New(),
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-2 * time.Hour),
		TotalAmount:   18.8,
		Status:        SessionStatusCompleted,
		PaymentStatus: PaymentStatusPaid,
	}

	err := uc.ConfirmPayment(ctx, sessionID, "txn-003", "wechat", 18.8)
	if err == nil {
		t.Error("expected error for already processed payment")
	}
}

func TestChargingUseCase_RefundPayment(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	now := time.Now()
	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        uuid.New(),
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-2 * time.Hour),
		TotalAmount:   18.8,
		Status:        SessionStatusCompleted,
		PaymentStatus: PaymentStatusPaid,
		PayTime:       &now,
	}

	err := uc.RefundPayment(ctx, sessionID, "test refund")
	if err != nil {
		t.Errorf("RefundPayment() error: %v", err)
	}

	session := repo.sessions[sessionID]
	if session.PaymentStatus != PaymentStatusRefunded {
		t.Errorf("PaymentStatus = %s, want refunded", session.PaymentStatus)
	}
}

func TestChargingUseCase_RefundPayment_NotPaid(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	sessionID := uuid.New()
	repo.sessions[sessionID] = &Session{
		ID:            sessionID,
		StationID:     uuid.New(),
		ConnectorID:   uuid.New(),
		UserID:        uuid.New(),
		VehiclePlate:  "京A12345",
		StartTime:     time.Now().Add(-2 * time.Hour),
		TotalAmount:   18.8,
		Status:        SessionStatusCompleted,
		PaymentStatus: PaymentStatusPending,
	}

	err := uc.RefundPayment(ctx, sessionID, "test refund")
	if err == nil {
		t.Error("expected error for refunding non-paid session")
	}
}

func TestChargingUseCase_CalculateEnergyCost(t *testing.T) {
	uc, _ := setupChargingUseCase()

	tests := []struct {
		name     string
		energy   float64
		price    *Price
		expected float64
	}{
		{
			name:   "nil price",
			energy: 10.0,
			price:  nil,
			expected: 0,
		},
		{
			name:   "off-peak price",
			energy: 10.0,
			price: &Price{
				PricePerKWh: 1.2,
				OffPeakLoad: 0.8,
				IsPeakHours: false,
			},
			expected: 20.0,
		},
		{
			name:   "peak price",
			energy: 10.0,
			price: &Price{
				PricePerKWh: 1.2,
				PeakLoad:    1.5,
				IsPeakHours: true,
			},
			expected: 27.0,
		},
		{
			name:   "zero energy",
			energy: 0,
			price: &Price{
				PricePerKWh: 1.2,
				OffPeakLoad: 0.8,
				IsPeakHours: false,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uc.calculateEnergyCost(tt.energy, tt.price)
			if got != tt.expected {
				t.Errorf("calculateEnergyCost() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestChargingUseCase_IsPeakHours(t *testing.T) {
	uc, _ := setupChargingUseCase()

	tests := []struct {
		name      string
		startHour int
		endHour   int
		expected  bool
	}{
		{
			name:      "morning peak 7-9",
			startHour: 7,
			endHour:   9,
			expected:  true,
		},
		{
			name:      "evening peak 17-21",
			startHour: 17,
			endHour:   21,
			expected:  true,
		},
		{
			name:      "off-peak 10-16",
			startHour: 10,
			endHour:   16,
			expected:  false,
		},
		{
			name:      "night 22-6",
			startHour: 22,
			endHour:   6,
			expected:  false,
		},
		{
			name:      "partial overlap 8-12",
			startHour: 8,
			endHour:   12,
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := uc.isPeakHours(tt.startHour, tt.endHour)
			if got != tt.expected {
				t.Errorf("isPeakHours(%d, %d) = %v, want %v", tt.startHour, tt.endHour, got, tt.expected)
			}
		})
	}
}

func TestStation_HasAvailableConnector(t *testing.T) {
	tests := []struct {
		name     string
		station  *Station
		expected bool
	}{
		{
			name: "available with connectors",
			station: &Station{
				AvailableConnectors: 2,
				Status:              StationStatusAvailable,
			},
			expected: true,
		},
		{
			name: "no available connectors",
			station: &Station{
				AvailableConnectors: 0,
				Status:              StationStatusAvailable,
			},
			expected: false,
		},
		{
			name: "offline station",
			station: &Station{
				AvailableConnectors: 2,
				Status:              StationStatusOffline,
			},
			expected: false,
		},
		{
			name: "maintenance station",
			station: &Station{
				AvailableConnectors: 2,
				Status:              StationStatusMaintenance,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.station.HasAvailableConnector()
			if got != tt.expected {
				t.Errorf("HasAvailableConnector() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSession_Duration(t *testing.T) {
	tests := []struct {
		name     string
		session  *Session
		minHours float64
		maxHours float64
	}{
		{
			name: "with end time",
			session: &Session{
				StartTime: time.Now().Add(-2 * time.Hour),
				EndTime:   ptrTime(time.Now().Add(-1 * time.Hour)),
			},
			minHours: 0.9,
			maxHours: 1.1,
		},
		{
			name: "without end time uses now",
			session: &Session{
				StartTime: time.Now().Add(-30 * time.Minute),
			},
			minHours: 0.4,
			maxHours: 0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dur := tt.session.Duration()
			if dur < tt.minHours || dur > tt.maxHours {
				t.Errorf("Duration() = %v, want between %v and %v", dur, tt.minHours, tt.maxHours)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultServiceFee != 0.8 {
		t.Errorf("DefaultServiceFee = %f, want 0.8", cfg.DefaultServiceFee)
	}
	if cfg.MaxSessionDuration != 8*time.Hour {
		t.Errorf("MaxSessionDuration = %v, want 8h", cfg.MaxSessionDuration)
	}
	if cfg.DefaultPeakLoadKWh != 1.5 {
		t.Errorf("DefaultPeakLoadKWh = %f, want 1.5", cfg.DefaultPeakLoadKWh)
	}
	if cfg.DefaultOffPeakLoadKWh != 0.8 {
		t.Errorf("DefaultOffPeakLoadKWh = %f, want 0.8", cfg.DefaultOffPeakLoadKWh)
	}
}

func TestChargingUseCase_GetUserSessions_Pagination(t *testing.T) {
	uc, repo := setupChargingUseCase()
	ctx := context.Background()

	userID := uuid.New()
	for i := 0; i < 5; i++ {
		sid := uuid.New()
		repo.sessions[sid] = &Session{
			ID:        sid,
			UserID:    userID,
			StationID: uuid.New(),
			Status:    SessionStatusCompleted,
		}
	}

	sessions, total, err := uc.GetUserSessions(ctx, userID, 1, 10)
	if err != nil {
		t.Errorf("GetUserSessions() error: %v", err)
	}
	if total != 5 {
		t.Errorf("total = %d, want 5", total)
	}
	if len(sessions) != 5 {
		t.Errorf("len(sessions) = %d, want 5", len(sessions))
	}
}

func TestChargingUseCase_CreatePrice(t *testing.T) {
	uc, _ := setupChargingUseCase()
	ctx := context.Background()

	stationID := uuid.New()

	tests := []struct {
		name        string
		startHour   int
		endHour     int
		pricePerKWh float64
		wantErr     bool
	}{
		{
			name:        "valid price",
			startHour:   7,
			endHour:     9,
			pricePerKWh: 1.5,
			wantErr:     false,
		},
		{
			name:        "invalid start hour",
			startHour:   -1,
			endHour:     9,
			pricePerKWh: 1.5,
			wantErr:     true,
		},
		{
			name:        "invalid end hour",
			startHour:   7,
			endHour:     25,
			pricePerKWh: 1.5,
			wantErr:     true,
		},
		{
			name:        "zero price",
			startHour:   7,
			endHour:     9,
			pricePerKWh: 0,
			wantErr:     true,
		},
		{
			name:        "negative price",
			startHour:   7,
			endHour:     9,
			pricePerKWh: -1.0,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, err := uc.CreatePrice(ctx, stationID, "test price", tt.startHour, tt.endHour, tt.pricePerKWh, 0.8, 1.5, 0.8, time.Now(), time.Now().Add(24*time.Hour))

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if price.PricePerKWh != tt.pricePerKWh {
				t.Errorf("PricePerKWh = %f, want %f", price.PricePerKWh, tt.pricePerKWh)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

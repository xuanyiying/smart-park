// Package data provides data access layer for the charging service.
package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/charging/biz"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent/connector"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent/price"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent/session"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent/station"
)

// chargingRepo implements biz.ChargingRepo.
type chargingRepo struct {
	data *Data
}

// NewChargingRepo creates a new ChargingRepo.
func NewChargingRepo(data *Data) biz.ChargingRepo {
	return &chargingRepo{data: data}
}

// CreateStation creates a new charging station.
func (r *chargingRepo) CreateStation(ctx context.Context, s *biz.Station) error {
	_, err := r.data.db.Station.Create().
		SetID(s.ID).
		SetLotID(s.LotID).
		SetName(s.Name).
		SetStationType(station.StationType(s.StationType)).
		SetStatus(station.Status(s.Status)).
		SetConnectorType(station.ConnectorType(s.ConnectorType)).
		SetMaxPower(s.MaxPower).
		SetVoltage(s.Voltage).
		SetTotalConnectors(s.TotalConnectors).
		SetAvailableConnectors(s.AvailableConnectors).
		SetLocation(s.Location).
		SetFloor(s.Floor).
		Save(ctx)
	return err
}

// GetStation retrieves a charging station by ID.
func (r *chargingRepo) GetStation(ctx context.Context, stationID uuid.UUID) (*biz.Station, error) {
	s, err := r.data.db.Station.Get(ctx, stationID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrStationNotFound
		}
		return nil, err
	}
	return toBizStation(s), nil
}

// UpdateStation updates a charging station.
func (r *chargingRepo) UpdateStation(ctx context.Context, s *biz.Station) error {
	_, err := r.data.db.Station.UpdateOneID(s.ID).
		SetName(s.Name).
		SetStatus(station.Status(s.Status)).
		SetMaxPower(s.MaxPower).
		SetVoltage(s.Voltage).
		SetTotalConnectors(s.TotalConnectors).
		SetAvailableConnectors(s.AvailableConnectors).
		SetLocation(s.Location).
		SetFloor(s.Floor).
		Save(ctx)
	return err
}

// DeleteStation deletes a charging station.
func (r *chargingRepo) DeleteStation(ctx context.Context, stationID uuid.UUID) error {
	return r.data.db.Station.DeleteOneID(stationID).Exec(ctx)
}

// ListStations retrieves charging stations by lot ID.
func (r *chargingRepo) ListStations(ctx context.Context, lotID uuid.UUID) ([]*biz.Station, error) {
	stations, err := r.data.db.Station.Query().
		Where(station.LotID(lotID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.Station, len(stations))
	for i, s := range stations {
		result[i] = toBizStation(s)
	}
	return result, nil
}

// CreateConnector creates a new connector.
func (r *chargingRepo) CreateConnector(ctx context.Context, c *biz.Connector) error {
	_, err := r.data.db.Connector.Create().
		SetID(c.ID).
		SetStationID(c.StationID).
		SetNumber(c.Number).
		SetType(connector.Type(c.Type)).
		SetStatus(connector.Status(c.Status)).
		SetMaxPower(c.MaxPower).
		SetVoltage(c.Voltage).
		SetCurrent(c.Current).
		SetFaultCode(c.FaultCode).
		Save(ctx)
	return err
}

// GetConnector retrieves a connector by ID.
func (r *chargingRepo) GetConnector(ctx context.Context, connectorID uuid.UUID) (*biz.Connector, error) {
	c, err := r.data.db.Connector.Get(ctx, connectorID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrConnectorNotFound
		}
		return nil, err
	}
	return toBizConnector(c), nil
}

// UpdateConnector updates a connector.
func (r *chargingRepo) UpdateConnector(ctx context.Context, c *biz.Connector) error {
	_, err := r.data.db.Connector.UpdateOneID(c.ID).
		SetStatus(connector.Status(c.Status)).
		SetMaxPower(c.MaxPower).
		SetVoltage(c.Voltage).
		SetCurrent(c.Current).
		SetFaultCode(c.FaultCode).
		Save(ctx)
	return err
}

// DeleteConnector deletes a connector.
func (r *chargingRepo) DeleteConnector(ctx context.Context, connectorID uuid.UUID) error {
	return r.data.db.Connector.DeleteOneID(connectorID).Exec(ctx)
}

// ListConnectors retrieves connectors by station ID.
func (r *chargingRepo) ListConnectors(ctx context.Context, stationID uuid.UUID) ([]*biz.Connector, error) {
	connectors, err := r.data.db.Connector.Query().
		Where(connector.StationID(stationID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.Connector, len(connectors))
	for i, c := range connectors {
		result[i] = toBizConnector(c)
	}
	return result, nil
}

// LockConnector locks a connector for charging.
func (r *chargingRepo) LockConnector(ctx context.Context, connectorID uuid.UUID) error {
	_, err := r.data.db.Connector.UpdateOneID(connectorID).
		SetStatus(connector.StatusCharging).
		Save(ctx)
	return err
}

// UnlockConnector unlocks a connector.
func (r *chargingRepo) UnlockConnector(ctx context.Context, connectorID uuid.UUID) error {
	_, err := r.data.db.Connector.UpdateOneID(connectorID).
		SetStatus(connector.StatusAvailable).
		Save(ctx)
	return err
}

// CreateSession creates a new charging session.
func (r *chargingRepo) CreateSession(ctx context.Context, s *biz.Session) error {
	builder := r.data.db.Session.Create().
		SetID(s.ID).
		SetStationID(s.StationID).
		SetConnectorID(s.ConnectorID).
		SetUserID(s.UserID).
		SetVehiclePlate(s.VehiclePlate).
		SetStartTime(s.StartTime).
		SetStartEnergy(s.StartEnergy).
		SetChargedEnergy(s.ChargedEnergy).
		SetCost(s.Cost).
		SetServiceFee(s.ServiceFee).
		SetTotalAmount(s.TotalAmount).
		SetStatus(session.Status(s.Status)).
		SetPaymentStatus(session.PaymentStatus(s.PaymentStatus))

	if s.EndTime != nil {
		builder.SetEndTime(*s.EndTime)
	}
	if s.PayTime != nil {
		builder.SetPayTime(*s.PayTime)
	}
	if s.PaymentMethod != "" {
		builder.SetPaymentMethod(s.PaymentMethod)
	}
	if s.TransactionID != "" {
		builder.SetTransactionID(s.TransactionID)
	}

	_, err := builder.Save(ctx)
	return err
}

// GetSession retrieves a charging session by ID.
func (r *chargingRepo) GetSession(ctx context.Context, sessionID uuid.UUID) (*biz.Session, error) {
	s, err := r.data.db.Session.Get(ctx, sessionID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrSessionNotFound
		}
		return nil, err
	}
	return toBizSession(s), nil
}

// UpdateSession updates a charging session.
func (r *chargingRepo) UpdateSession(ctx context.Context, s *biz.Session) error {
	builder := r.data.db.Session.UpdateOneID(s.ID).
		SetChargedEnergy(s.ChargedEnergy).
		SetCost(s.Cost).
		SetServiceFee(s.ServiceFee).
		SetTotalAmount(s.TotalAmount).
		SetStatus(session.Status(s.Status)).
		SetPaymentStatus(session.PaymentStatus(s.PaymentStatus))

	if s.EndTime != nil {
		builder.SetEndTime(*s.EndTime)
	}
	if s.PayTime != nil {
		builder.SetPayTime(*s.PayTime)
	}
	if s.PaymentMethod != "" {
		builder.SetPaymentMethod(s.PaymentMethod)
	}
	if s.TransactionID != "" {
		builder.SetTransactionID(s.TransactionID)
	}

	_, err := builder.Save(ctx)
	return err
}

// GetActiveSession retrieves the active session for a connector.
func (r *chargingRepo) GetActiveSession(ctx context.Context, connectorID uuid.UUID) (*biz.Session, error) {
	s, err := r.data.db.Session.Query().
		Where(
			session.ConnectorID(connectorID),
			session.StatusIn(session.StatusPending, session.StatusCharging),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toBizSession(s), nil
}

// ListUserSessions retrieves sessions for a user with pagination.
func (r *chargingRepo) ListUserSessions(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*biz.Session, int64, error) {
	query := r.data.db.Session.Query().
		Where(session.UserID(userID))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	sessions, err := query.
		Order(ent.Desc(session.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*biz.Session, len(sessions))
	for i, s := range sessions {
		result[i] = toBizSession(s)
	}
	return result, int64(total), nil
}

// ListStationSessions retrieves sessions for a station with pagination.
func (r *chargingRepo) ListStationSessions(ctx context.Context, stationID uuid.UUID, page, pageSize int) ([]*biz.Session, int64, error) {
	query := r.data.db.Session.Query().
		Where(session.StationID(stationID))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	sessions, err := query.
		Order(ent.Desc(session.FieldCreatedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*biz.Session, len(sessions))
	for i, s := range sessions {
		result[i] = toBizSession(s)
	}
	return result, int64(total), nil
}

// ExpireOldSessions expires sessions exceeding the threshold duration.
func (r *chargingRepo) ExpireOldSessions(ctx context.Context, threshold time.Duration) (int64, error) {
	cutoff := time.Now().Add(-threshold)
	n, err := r.data.db.Session.Update().
		Where(
			session.StatusEQ(session.StatusCharging),
			session.StartTimeLT(cutoff),
		).
		SetStatus(session.StatusExpired).
		Save(ctx)
	return int64(n), err
}

// CreatePrice creates a new price configuration.
func (r *chargingRepo) CreatePrice(ctx context.Context, p *biz.Price) error {
	builder := r.data.db.Price.Create().
		SetID(p.ID).
		SetStationID(p.StationID).
		SetName(p.Name).
		SetStartHour(p.StartHour).
		SetEndHour(p.EndHour).
		SetPricePerKwh(p.PricePerKWh).
		SetServiceFee(p.ServiceFee).
		SetPeakLoad(p.PeakLoad).
		SetOffPeakLoad(p.OffPeakLoad).
		SetIsPeakHours(p.IsPeakHours).
		SetEffectiveAt(p.EffectiveAt)

	if !p.ExpiresAt.IsZero() {
		builder.SetExpiresAt(p.ExpiresAt)
	}

	_, err := builder.Save(ctx)
	return err
}

// GetPrice retrieves a price configuration by ID.
func (r *chargingRepo) GetPrice(ctx context.Context, priceID uuid.UUID) (*biz.Price, error) {
	p, err := r.data.db.Price.Get(ctx, priceID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPriceNotConfigured
		}
		return nil, err
	}
	return toBizPrice(p), nil
}

// UpdatePrice updates a price configuration.
func (r *chargingRepo) UpdatePrice(ctx context.Context, p *biz.Price) error {
	builder := r.data.db.Price.UpdateOneID(p.ID).
		SetName(p.Name).
		SetStartHour(p.StartHour).
		SetEndHour(p.EndHour).
		SetPricePerKwh(p.PricePerKWh).
		SetServiceFee(p.ServiceFee).
		SetPeakLoad(p.PeakLoad).
		SetOffPeakLoad(p.OffPeakLoad).
		SetIsPeakHours(p.IsPeakHours).
		SetEffectiveAt(p.EffectiveAt)

	if !p.ExpiresAt.IsZero() {
		builder.SetExpiresAt(p.ExpiresAt)
	}

	_, err := builder.Save(ctx)
	return err
}

// GetCurrentPrice retrieves the current effective price for a station.
func (r *chargingRepo) GetCurrentPrice(ctx context.Context, stationID uuid.UUID) (*biz.Price, error) {
	now := time.Now()
	p, err := r.data.db.Price.Query().
		Where(
			price.StationID(stationID),
			price.EffectiveAtLTE(now),
			price.Or(
				price.ExpiresAtIsNil(),
				price.ExpiresAtGT(now),
			),
		).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrPriceNotConfigured
		}
		return nil, err
	}
	return toBizPrice(p), nil
}

// ListPrices retrieves price configurations for a station.
func (r *chargingRepo) ListPrices(ctx context.Context, stationID uuid.UUID) ([]*biz.Price, error) {
	prices, err := r.data.db.Price.Query().
		Where(price.StationID(stationID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.Price, len(prices))
	for i, p := range prices {
		result[i] = toBizPrice(p)
	}
	return result, nil
}

// DeletePrice deletes a price configuration.
func (r *chargingRepo) DeletePrice(ctx context.Context, priceID uuid.UUID) error {
	return r.data.db.Price.DeleteOneID(priceID).Exec(ctx)
}

// Helper functions to convert ent types to biz types
func toBizStation(s *ent.Station) *biz.Station {
	return &biz.Station{
		ID:                  s.ID,
		LotID:               s.LotID,
		Name:                s.Name,
		StationType:         biz.ConnectorType(s.StationType),
		Status:              biz.StationStatus(s.Status),
		ConnectorType:       biz.ConnectorType(s.ConnectorType),
		MaxPower:            s.MaxPower,
		Voltage:             s.Voltage,
		TotalConnectors:     s.TotalConnectors,
		AvailableConnectors: s.AvailableConnectors,
		Location:            s.Location,
		Floor:               s.Floor,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

func toBizConnector(c *ent.Connector) *biz.Connector {
	return &biz.Connector{
		ID:        c.ID,
		StationID: c.StationID,
		Number:    c.Number,
		Type:      biz.ConnectorType(c.Type),
		Status:    biz.ConnectorStatus(c.Status),
		MaxPower:  c.MaxPower,
		Voltage:   c.Voltage,
		Current:   c.Current,
		FaultCode: c.FaultCode,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

func toBizSession(s *ent.Session) *biz.Session {
	session := &biz.Session{
		ID:            s.ID,
		StationID:     s.StationID,
		ConnectorID:   s.ConnectorID,
		UserID:        s.UserID,
		VehiclePlate:  s.VehiclePlate,
		StartTime:     s.StartTime,
		StartEnergy:   s.StartEnergy,
		EndEnergy:     s.EndEnergy,
		ChargedEnergy: s.ChargedEnergy,
		Cost:          s.Cost,
		ServiceFee:    s.ServiceFee,
		TotalAmount:   s.TotalAmount,
		Status:        biz.SessionStatus(s.Status),
		PaymentStatus: biz.PaymentStatus(s.PaymentStatus),
		PaymentMethod: s.PaymentMethod,
		TransactionID: s.TransactionID,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
	if s.EndTime != nil {
		session.EndTime = s.EndTime
	}
	if s.PayTime != nil {
		session.PayTime = s.PayTime
	}
	return session
}

func toBizPrice(p *ent.Price) *biz.Price {
	return &biz.Price{
		ID:          p.ID,
		StationID:   p.StationID,
		Name:        p.Name,
		StartHour:   p.StartHour,
		EndHour:     p.EndHour,
		PricePerKWh: p.PricePerKwh,
		ServiceFee:  p.ServiceFee,
		PeakLoad:    p.PeakLoad,
		OffPeakLoad: p.OffPeakLoad,
		IsPeakHours: p.IsPeakHours,
		EffectiveAt: p.EffectiveAt,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

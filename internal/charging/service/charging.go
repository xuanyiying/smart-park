// Package service provides gRPC service implementation for the charging service.
package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/charging/v1"
	"github.com/xuanyiying/smart-park/internal/charging/biz"
)

// ChargingService implements the ChargingService gRPC service.
type ChargingService struct {
	v1.UnimplementedChargingServiceServer

	uc  *biz.ChargingUseCase
	log *log.Helper
}

// NewChargingService creates a new ChargingService.
func NewChargingService(uc *biz.ChargingUseCase, logger log.Logger) *ChargingService {
	return &ChargingService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

// CreateStation creates a new charging station.
func (s *ChargingService) CreateStation(ctx context.Context, req *v1.CreateStationRequest) (*v1.CreateStationResponse, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return &v1.CreateStationResponse{Code: 400, Message: "无效的停车场ID"}, nil
	}

	station, err := s.uc.CreateStation(ctx, lotID, req.Name, biz.ConnectorType(req.StationType), biz.ConnectorType(req.ConnectorType), req.MaxPower, req.Voltage, int(req.TotalConnectors), req.Location, req.Floor)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateStation failed: %v", err)
		return &v1.CreateStationResponse{Code: 500, Message: "创建充电桩失败"}, nil
	}

	return &v1.CreateStationResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoStation(station),
	}, nil
}

// GetStation retrieves a charging station by ID.
func (s *ChargingService) GetStation(ctx context.Context, req *v1.GetStationRequest) (*v1.GetStationResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.GetStationResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	station, err := s.uc.GetStation(ctx, stationID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetStation failed: %v", err)
		return &v1.GetStationResponse{Code: 404, Message: "充电桩不存在"}, nil
	}

	return &v1.GetStationResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoStation(station),
	}, nil
}

// UpdateStation updates a charging station.
func (s *ChargingService) UpdateStation(ctx context.Context, req *v1.UpdateStationRequest) (*v1.UpdateStationResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.UpdateStationResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	if req.Status != "" {
		if err := s.uc.UpdateStationStatus(ctx, stationID, biz.StationStatus(req.Status)); err != nil {
			s.log.WithContext(ctx).Errorf("UpdateStationStatus failed: %v", err)
			return &v1.UpdateStationResponse{Code: 500, Message: "更新充电桩状态失败"}, nil
		}
	}

	station, err := s.uc.GetStation(ctx, stationID)
	if err != nil {
		return &v1.UpdateStationResponse{Code: 404, Message: "充电桩不存在"}, nil
	}

	return &v1.UpdateStationResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoStation(station),
	}, nil
}

// DeleteStation deletes a charging station.
func (s *ChargingService) DeleteStation(ctx context.Context, req *v1.DeleteStationRequest) (*v1.DeleteStationResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.DeleteStationResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	if err := s.uc.DeleteStation(ctx, stationID); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteStation failed: %v", err)
		return &v1.DeleteStationResponse{Code: 500, Message: "删除充电桩失败"}, nil
	}

	return &v1.DeleteStationResponse{Code: 0, Message: "success"}, nil
}

// ListStations retrieves charging stations by lot ID.
func (s *ChargingService) ListStations(ctx context.Context, req *v1.ListStationsRequest) (*v1.ListStationsResponse, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return &v1.ListStationsResponse{Code: 400, Message: "无效的停车场ID"}, nil
	}

	stations, err := s.uc.ListStations(ctx, lotID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListStations failed: %v", err)
		return &v1.ListStationsResponse{Code: 500, Message: "获取充电桩列表失败"}, nil
	}

	data := make([]*v1.Station, len(stations))
	for i, station := range stations {
		data[i] = toProtoStation(station)
	}

	return &v1.ListStationsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Total:   int64(len(stations)),
	}, nil
}

// GetAvailableStations retrieves available charging stations.
func (s *ChargingService) GetAvailableStations(ctx context.Context, req *v1.GetAvailableStationsRequest) (*v1.GetAvailableStationsResponse, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return &v1.GetAvailableStationsResponse{Code: 400, Message: "无效的停车场ID"}, nil
	}

	stations, err := s.uc.GetAvailableStations(ctx, lotID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetAvailableStations failed: %v", err)
		return &v1.GetAvailableStationsResponse{Code: 500, Message: "获取可用充电桩失败"}, nil
	}

	data := make([]*v1.Station, len(stations))
	for i, station := range stations {
		data[i] = toProtoStation(station)
	}

	return &v1.GetAvailableStationsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// CreateConnector creates a new connector.
func (s *ChargingService) CreateConnector(ctx context.Context, req *v1.CreateConnectorRequest) (*v1.CreateConnectorResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.CreateConnectorResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	connector, err := s.uc.CreateConnector(ctx, stationID, int(req.Number), biz.ConnectorType(req.Type), req.MaxPower, req.Voltage)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateConnector failed: %v", err)
		return &v1.CreateConnectorResponse{Code: 500, Message: "创建连接器失败"}, nil
	}

	return &v1.CreateConnectorResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoConnector(connector),
	}, nil
}

// GetConnector retrieves a connector by ID.
func (s *ChargingService) GetConnector(ctx context.Context, req *v1.GetConnectorRequest) (*v1.GetConnectorResponse, error) {
	connectorID, err := uuid.Parse(req.ConnectorId)
	if err != nil {
		return &v1.GetConnectorResponse{Code: 400, Message: "无效的连接器ID"}, nil
	}

	connector, err := s.uc.GetConnector(ctx, connectorID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetConnector failed: %v", err)
		return &v1.GetConnectorResponse{Code: 404, Message: "连接器不存在"}, nil
	}

	return &v1.GetConnectorResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoConnector(connector),
	}, nil
}

// ListConnectors retrieves connectors by station ID.
func (s *ChargingService) ListConnectors(ctx context.Context, req *v1.ListConnectorsRequest) (*v1.ListConnectorsResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.ListConnectorsResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	connectors, err := s.uc.ListConnectors(ctx, stationID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListConnectors failed: %v", err)
		return &v1.ListConnectorsResponse{Code: 500, Message: "获取连接器列表失败"}, nil
	}

	data := make([]*v1.Connector, len(connectors))
	for i, connector := range connectors {
		data[i] = toProtoConnector(connector)
	}

	return &v1.ListConnectorsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// StartCharging starts a new charging session.
func (s *ChargingService) StartCharging(ctx context.Context, req *v1.StartChargingRequest) (*v1.StartChargingResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.StartChargingResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	connectorID, err := uuid.Parse(req.ConnectorId)
	if err != nil {
		return &v1.StartChargingResponse{Code: 400, Message: "无效的连接器ID"}, nil
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return &v1.StartChargingResponse{Code: 400, Message: "无效的用户ID"}, nil
	}

	session, err := s.uc.StartCharging(ctx, stationID, connectorID, userID, req.VehiclePlate)
	if err != nil {
		s.log.WithContext(ctx).Errorf("StartCharging failed: %v", err)
		return &v1.StartChargingResponse{Code: 500, Message: "开始充电失败"}, nil
	}

	return &v1.StartChargingResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoSession(session),
	}, nil
}

// StopCharging stops a charging session.
func (s *ChargingService) StopCharging(ctx context.Context, req *v1.StopChargingRequest) (*v1.StopChargingResponse, error) {
	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		return &v1.StopChargingResponse{Code: 400, Message: "无效的会话ID"}, nil
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return &v1.StopChargingResponse{Code: 400, Message: "无效的用户ID"}, nil
	}

	session, err := s.uc.StopCharging(ctx, sessionID, userID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("StopCharging failed: %v", err)
		return &v1.StopChargingResponse{Code: 500, Message: "停止充电失败"}, nil
	}

	return &v1.StopChargingResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoSession(session),
	}, nil
}

// GetSession retrieves a charging session by ID.
func (s *ChargingService) GetSession(ctx context.Context, req *v1.GetSessionRequest) (*v1.GetSessionResponse, error) {
	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		return &v1.GetSessionResponse{Code: 400, Message: "无效的会话ID"}, nil
	}

	session, err := s.uc.GetSession(ctx, sessionID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetSession failed: %v", err)
		return &v1.GetSessionResponse{Code: 404, Message: "会话不存在"}, nil
	}

	return &v1.GetSessionResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoSession(session),
	}, nil
}

// ListUserSessions retrieves sessions for a user with pagination.
func (s *ChargingService) ListUserSessions(ctx context.Context, req *v1.ListUserSessionsRequest) (*v1.ListUserSessionsResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return &v1.ListUserSessionsResponse{Code: 400, Message: "无效的用户ID"}, nil
	}

	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 20
	}

	sessions, total, err := s.uc.GetUserSessions(ctx, userID, page, pageSize)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetUserSessions failed: %v", err)
		return &v1.ListUserSessionsResponse{Code: 500, Message: "获取会话列表失败"}, nil
	}

	data := make([]*v1.Session, len(sessions))
	for i, session := range sessions {
		data[i] = toProtoSession(session)
	}

	return &v1.ListUserSessionsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Total:   total,
	}, nil
}

// GetChargingSummary retrieves a charging summary for a session.
func (s *ChargingService) GetChargingSummary(ctx context.Context, req *v1.GetChargingSummaryRequest) (*v1.GetChargingSummaryResponse, error) {
	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		return &v1.GetChargingSummaryResponse{Code: 400, Message: "无效的会话ID"}, nil
	}

	summary, err := s.uc.GetChargingSummary(ctx, sessionID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetChargingSummary failed: %v", err)
		return &v1.GetChargingSummaryResponse{Code: 404, Message: "会话不存在"}, nil
	}

	return &v1.GetChargingSummaryResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoChargingSummary(summary),
	}, nil
}

// CreatePrice creates a new price configuration.
func (s *ChargingService) CreatePrice(ctx context.Context, req *v1.CreatePriceRequest) (*v1.CreatePriceResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.CreatePriceResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	effectiveAt, _ := time.Parse(time.RFC3339, req.EffectiveAt)
	if effectiveAt.IsZero() {
		effectiveAt = time.Now()
	}

	var expiresAt time.Time
	if req.ExpiresAt != "" {
		expiresAt, _ = time.Parse(time.RFC3339, req.ExpiresAt)
	}

	price, err := s.uc.CreatePrice(ctx, stationID, req.Name, int(req.StartHour), int(req.EndHour), req.PricePerKwh, req.ServiceFee, req.PeakLoad, req.OffPeakLoad, effectiveAt, expiresAt)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreatePrice failed: %v", err)
		return &v1.CreatePriceResponse{Code: 500, Message: "创建价格配置失败"}, nil
	}

	return &v1.CreatePriceResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoPrice(price),
	}, nil
}

// GetCurrentPrice retrieves the current effective price for a station.
func (s *ChargingService) GetCurrentPrice(ctx context.Context, req *v1.GetCurrentPriceRequest) (*v1.GetCurrentPriceResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.GetCurrentPriceResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	price, err := s.uc.GetCurrentPrice(ctx, stationID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetCurrentPrice failed: %v", err)
		return &v1.GetCurrentPriceResponse{Code: 404, Message: "价格配置不存在"}, nil
	}

	return &v1.GetCurrentPriceResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoPrice(price),
	}, nil
}

// ListPrices retrieves price configurations for a station.
func (s *ChargingService) ListPrices(ctx context.Context, req *v1.ListPricesRequest) (*v1.ListPricesResponse, error) {
	stationID, err := uuid.Parse(req.StationId)
	if err != nil {
		return &v1.ListPricesResponse{Code: 400, Message: "无效的充电桩ID"}, nil
	}

	prices, err := s.uc.ListPrices(ctx, stationID)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListPrices failed: %v", err)
		return &v1.ListPricesResponse{Code: 500, Message: "获取价格列表失败"}, nil
	}

	data := make([]*v1.Price, len(prices))
	for i, price := range prices {
		data[i] = toProtoPrice(price)
	}

	return &v1.ListPricesResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// ConfirmPayment confirms payment for a charging session.
func (s *ChargingService) ConfirmPayment(ctx context.Context, req *v1.ConfirmPaymentRequest) (*v1.ConfirmPaymentResponse, error) {
	sessionID, err := uuid.Parse(req.SessionId)
	if err != nil {
		return &v1.ConfirmPaymentResponse{Code: 400, Message: "无效的会话ID"}, nil
	}

	if err := s.uc.ConfirmPayment(ctx, sessionID, req.TransactionId, req.PaymentMethod, req.PaidAmount); err != nil {
		s.log.WithContext(ctx).Errorf("ConfirmPayment failed: %v", err)
		return &v1.ConfirmPaymentResponse{Code: 500, Message: "确认支付失败"}, nil
	}

	session, _ := s.uc.GetSession(ctx, sessionID)

	return &v1.ConfirmPaymentResponse{
		Code:    0,
		Message: "success",
		Data:    toProtoSession(session),
	}, nil
}

// Helper functions to convert biz types to proto types
func toProtoStation(s *biz.Station) *v1.Station {
	return &v1.Station{
		Id:                  s.ID.String(),
		LotId:               s.LotID.String(),
		Name:                s.Name,
		StationType:         string(s.StationType),
		Status:              string(s.Status),
		ConnectorType:       string(s.ConnectorType),
		MaxPower:            s.MaxPower,
		Voltage:             s.Voltage,
		TotalConnectors:     int32(s.TotalConnectors),
		AvailableConnectors: int32(s.AvailableConnectors),
		Location:            s.Location,
		Floor:               s.Floor,
		CreatedAt:           s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           s.UpdatedAt.Format(time.RFC3339),
	}
}

func toProtoConnector(c *biz.Connector) *v1.Connector {
	return &v1.Connector{
		Id:        c.ID.String(),
		StationId: c.StationID.String(),
		Number:    int32(c.Number),
		Type:      string(c.Type),
		Status:    string(c.Status),
		MaxPower:  c.MaxPower,
		Voltage:   c.Voltage,
		Current:   c.Current,
		FaultCode: c.FaultCode,
		CreatedAt: c.CreatedAt.Format(time.RFC3339),
		UpdatedAt: c.UpdatedAt.Format(time.RFC3339),
	}
}

func toProtoSession(s *biz.Session) *v1.Session {
	session := &v1.Session{
		Id:            s.ID.String(),
		StationId:     s.StationID.String(),
		ConnectorId:   s.ConnectorID.String(),
		UserId:        s.UserID.String(),
		VehiclePlate:  s.VehiclePlate,
		StartTime:     s.StartTime.Format(time.RFC3339),
		StartEnergy:   s.StartEnergy,
		EndEnergy:     s.EndEnergy,
		ChargedEnergy: s.ChargedEnergy,
		Cost:          s.Cost,
		ServiceFee:    s.ServiceFee,
		TotalAmount:   s.TotalAmount,
		Status:        string(s.Status),
		PaymentStatus: string(s.PaymentStatus),
		PaymentMethod: s.PaymentMethod,
		TransactionId: s.TransactionID,
		CreatedAt:     s.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     s.UpdatedAt.Format(time.RFC3339),
	}
	if s.EndTime != nil {
		session.EndTime = s.EndTime.Format(time.RFC3339)
	}
	if s.PayTime != nil {
		session.PayTime = s.PayTime.Format(time.RFC3339)
	}
	return session
}

func toProtoChargingSummary(s *biz.ChargingSummary) *v1.ChargingSummary {
	return &v1.ChargingSummary{
		SessionId:     s.SessionID.String(),
		StationName:   s.StationName,
		ConnectorNum:  int32(s.ConnectorNum),
		VehiclePlate:  s.VehiclePlate,
		StartTime:     s.StartTime.Format(time.RFC3339),
		EndTime:       s.EndTime.Format(time.RFC3339),
		Duration:      s.Duration,
		ChargedEnergy: s.ChargedEnergy,
		Cost:          s.Cost,
		ServiceFee:    s.ServiceFee,
		TotalAmount:   s.TotalAmount,
		PaymentStatus: s.PaymentStatus,
	}
}

func toProtoPrice(p *biz.Price) *v1.Price {
	price := &v1.Price{
		Id:          p.ID.String(),
		StationId:   p.StationID.String(),
		Name:        p.Name,
		StartHour:   int32(p.StartHour),
		EndHour:     int32(p.EndHour),
		PricePerKwh: p.PricePerKWh,
		ServiceFee:  p.ServiceFee,
		PeakLoad:    p.PeakLoad,
		OffPeakLoad: p.OffPeakLoad,
		IsPeakHours: p.IsPeakHours,
		EffectiveAt: p.EffectiveAt.Format(time.RFC3339),
		CreatedAt:   p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   p.UpdatedAt.Format(time.RFC3339),
	}
	if !p.ExpiresAt.IsZero() {
		price.ExpiresAt = p.ExpiresAt.Format(time.RFC3339)
	}
	return price
}

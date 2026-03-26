package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	biz "github.com/xuanyiying/smart-park/internal/analytics/biz"
	v1 "github.com/xuanyiying/smart-park/api/analytics/v1"
)

// AnalyticsService implements the analytics service interface
type AnalyticsService struct {
	v1.UnimplementedAnalyticsServiceServer

	uc     *biz.AnalyticsUseCase
	logger *log.Helper
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(uc *biz.AnalyticsUseCase, logger log.Logger) *AnalyticsService {
	return &AnalyticsService{
		uc:     uc,
		logger: log.NewHelper(logger),
	}
}

// GetLotAnalytics retrieves analytics data for a specific parking lot
func (s *AnalyticsService) GetLotAnalytics(ctx context.Context, req *v1.GetLotAnalyticsRequest) (*v1.GetLotAnalyticsResponse, error) {
	s.logger.WithContext(ctx).Infof("GetLotAnalytics called for lot: %s", req.LotId)

	data, err := s.uc.GetLotAnalytics(ctx, req)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("GetLotAnalytics failed: %v", err)
		return nil, err
	}

	return &v1.GetLotAnalyticsResponse{
		Data: data,
	}, nil
}

// GetRevenueTrend retrieves revenue trend data
func (s *AnalyticsService) GetRevenueTrend(ctx context.Context, req *v1.GetRevenueTrendRequest) (*v1.GetRevenueTrendResponse, error) {
	s.logger.WithContext(ctx).Infof("GetRevenueTrend called for lot: %s, period: %s", req.LotId, req.Period)

	data, err := s.uc.GetRevenueTrend(ctx, req)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("GetRevenueTrend failed: %v", err)
		return nil, err
	}

	return &v1.GetRevenueTrendResponse{
		Data: data,
	}, nil
}

// PredictPeakHours predicts peak hours based on historical data
func (s *AnalyticsService) PredictPeakHours(ctx context.Context, req *v1.PredictPeakHoursRequest) (*v1.PredictPeakHoursResponse, error) {
	s.logger.WithContext(ctx).Infof("PredictPeakHours called for lot: %s, date: %s", req.LotId, req.Date)

	data, err := s.uc.PredictPeakHours(ctx, req)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("PredictPeakHours failed: %v", err)
		return nil, err
	}

	return &v1.PredictPeakHoursResponse{
		Data: data,
	}, nil
}

// GetOccupancyRate retrieves occupancy rate data
func (s *AnalyticsService) GetOccupancyRate(ctx context.Context, req *v1.GetOccupancyRateRequest) (*v1.GetOccupancyRateResponse, error) {
	s.logger.WithContext(ctx).Infof("GetOccupancyRate called for lot: %s", req.LotId)

	data, err := s.uc.GetOccupancyRate(ctx, req)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("GetOccupancyRate failed: %v", err)
		return nil, err
	}

	return &v1.GetOccupancyRateResponse{
		Data: data,
	}, nil
}

// GetVehicleFlow retrieves vehicle flow data
func (s *AnalyticsService) GetVehicleFlow(ctx context.Context, req *v1.GetVehicleFlowRequest) (*v1.GetVehicleFlowResponse, error) {
	s.logger.WithContext(ctx).Infof("GetVehicleFlow called for lot: %s, date: %s", req.LotId, req.Date)

	data, err := s.uc.GetVehicleFlow(ctx, req)
	if err != nil {
		s.logger.WithContext(ctx).Errorf("GetVehicleFlow failed: %v", err)
		return nil, err
	}

	return &v1.GetVehicleFlowResponse{
		Data: data,
	}, nil
}

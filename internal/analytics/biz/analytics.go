// Package biz provides business logic for the analytics service.
package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/google/wire"

	v1 "github.com/xuanyiying/smart-park/api/analytics/v1"
	"github.com/xuanyiying/smart-park/internal/analytics/ml"
)

// ProviderSet is the provider set for business layer.
var ProviderSet = wire.NewSet(
	NewAnalyticsUseCase,
)

// AnalyticsRepo defines the repository interface for analytics operations.
type AnalyticsRepo interface {
	GetLotStats(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) (*LotStats, error)
	GetRevenueData(ctx context.Context, lotID uuid.UUID, period string, limit int) ([]*RevenuePoint, error)
	GetOccupancyData(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) ([]*OccupancyPoint, error)
	GetVehicleFlowData(ctx context.Context, lotID uuid.UUID, date time.Time) ([]*FlowPoint, error)
	GetHistoricalPeakHours(ctx context.Context, lotID uuid.UUID, days int) (map[int]int, error)
}

// LotStats represents parking lot statistics.
type LotStats struct {
	LotID         uuid.UUID
	LotName       string
	TotalVehicles int
	TotalRevenue  float64
	AvgDuration   float64
	OccupancyRate float64
	PeakHour      int
}

// RevenuePoint represents a revenue data point.
type RevenuePoint struct {
	Date         time.Time
	Revenue      float64
	VehicleCount int
}

// OccupancyPoint represents an occupancy data point.
type OccupancyPoint struct {
	Timestamp      time.Time
	Rate           float64
	OccupiedSpaces int
	TotalSpaces    int
}

// FlowPoint represents a vehicle flow data point.
type FlowPoint struct {
	Timestamp time.Time
	Entries   int
	Exits     int
	NetFlow   int
}

// AnalyticsUseCase implements analytics business logic.
type AnalyticsUseCase struct {
	repo          AnalyticsRepo
	occupancyPredictor ml.OccupancyPredictor
	peakHourPredictor  ml.PeakHourPredictor
	log           *log.Helper
}

// NewAnalyticsUseCase creates a new AnalyticsUseCase.
func NewAnalyticsUseCase(repo AnalyticsRepo, logger log.Logger) *AnalyticsUseCase {
	return &AnalyticsUseCase{
		repo:               repo,
		occupancyPredictor: ml.NewSimplePredictor(),
		peakHourPredictor:  ml.NewSimplePredictor(),
		log:                log.NewHelper(logger),
	}
}

// GetLotAnalytics retrieves analytics data for a specific parking lot.
func (uc *AnalyticsUseCase) GetLotAnalytics(ctx context.Context, req *v1.GetLotAnalyticsRequest) (*v1.LotAnalyticsData, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, err
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, err
	}

	stats, err := uc.repo.GetLotStats(ctx, lotID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	return &v1.LotAnalyticsData{
		LotId:         stats.LotID.String(),
		LotName:       stats.LotName,
		TotalVehicles: int32(stats.TotalVehicles),
		TotalRevenue:  stats.TotalRevenue,
		AvgDuration:   stats.AvgDuration,
		OccupancyRate: stats.OccupancyRate,
		PeakHour:      int32(stats.PeakHour),
	}, nil
}

// GetRevenueTrend retrieves revenue trend data.
func (uc *AnalyticsUseCase) GetRevenueTrend(ctx context.Context, req *v1.GetRevenueTrendRequest) (*v1.RevenueTrendData, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	points, err := uc.repo.GetRevenueData(ctx, lotID, req.Period, int(req.Limit))
	if err != nil {
		return nil, err
	}

	var totalRevenue float64
	var revenuePoints []*v1.RevenuePoint

	for _, p := range points {
		totalRevenue += p.Revenue
		revenuePoints = append(revenuePoints, &v1.RevenuePoint{
			Date:         p.Date.Format("2006-01-02"),
			Revenue:      p.Revenue,
			VehicleCount: int32(p.VehicleCount),
		})
	}

	avgRevenue := totalRevenue / float64(len(points))

	return &v1.RevenueTrendData{
		Points:       revenuePoints,
		TotalRevenue: totalRevenue,
		AvgRevenue:   avgRevenue,
	}, nil
}

// PredictPeakHours predicts peak hours based on historical data using machine learning.
func (uc *AnalyticsUseCase) PredictPeakHours(ctx context.Context, req *v1.PredictPeakHoursRequest) (*v1.PeakHoursPrediction, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}

	// Use machine learning model to predict peak hours
	peakHourPredictions, err := uc.peakHourPredictor.PredictPeakHours(ctx, lotID, date)
	if err != nil {
		// Fallback to historical data if ML prediction fails
		historicalData, err := uc.repo.GetHistoricalPeakHours(ctx, lotID, 30)
		if err != nil {
			return nil, err
		}

		var peakHours []*v1.PeakHour
		for hour, count := range historicalData {
			if count > 100 {
				peakHours = append(peakHours, &v1.PeakHour{
					StartHour:        int32(hour),
					EndHour:          int32(hour + 1),
					ExpectedVehicles: int32(count),
					Probability:      float64(count) / 500.0,
				})
			}
		}

		return &v1.PeakHoursPrediction{
			LotId:      lotID.String(),
			Date:       req.Date,
			PeakHours:  peakHours,
			Confidence: 0.85,
		}, nil
	}

	// Convert ML predictions to API response format
	var peakHours []*v1.PeakHour
	for _, prediction := range peakHourPredictions {
		peakHours = append(peakHours, &v1.PeakHour{
			StartHour:        int32(prediction.StartHour),
			EndHour:          int32(prediction.EndHour),
			ExpectedVehicles: int32(prediction.ExpectedOccupancy * 100), // Convert occupancy rate to vehicle count
			Probability:      prediction.Probability,
		})
	}

	return &v1.PeakHoursPrediction{
		LotId:      lotID.String(),
		Date:       req.Date,
		PeakHours:  peakHours,
		Confidence: 0.9, // Higher confidence with ML model
	}, nil
}

// GetOccupancyRate retrieves occupancy rate data.
func (uc *AnalyticsUseCase) GetOccupancyRate(ctx context.Context, req *v1.GetOccupancyRateRequest) (*v1.OccupancyRateData, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		return nil, err
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		return nil, err
	}

	points, err := uc.repo.GetOccupancyData(ctx, lotID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	var currentRate, avgRate, maxRate, minRate float64
	var occupancyPoints []*v1.OccupancyPoint

	minRate = 100.0
	for i, p := range points {
		if i == 0 {
			currentRate = p.Rate
		}
		avgRate += p.Rate
		if p.Rate > maxRate {
			maxRate = p.Rate
		}
		if p.Rate < minRate {
			minRate = p.Rate
		}

		occupancyPoints = append(occupancyPoints, &v1.OccupancyPoint{
			Timestamp:      p.Timestamp.Format(time.RFC3339),
			Rate:           p.Rate,
			OccupiedSpaces: int32(p.OccupiedSpaces),
			TotalSpaces:    int32(p.TotalSpaces),
		})
	}

	if len(points) > 0 {
		avgRate = avgRate / float64(len(points))
	}

	return &v1.OccupancyRateData{
		LotId:       lotID.String(),
		CurrentRate: currentRate,
		AvgRate:     avgRate,
		MaxRate:     maxRate,
		MinRate:     minRate,
		Points:      occupancyPoints,
	}, nil
}

// GetVehicleFlow retrieves vehicle flow data.
func (uc *AnalyticsUseCase) GetVehicleFlow(ctx context.Context, req *v1.GetVehicleFlowRequest) (*v1.VehicleFlowData, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		return nil, err
	}

	points, err := uc.repo.GetVehicleFlowData(ctx, lotID, date)
	if err != nil {
		return nil, err
	}

	var totalEntries, totalExits, currentVehicles int
	var flowPoints []*v1.FlowPoint

	for _, p := range points {
		totalEntries += p.Entries
		totalExits += p.Exits
		currentVehicles += p.NetFlow

		flowPoints = append(flowPoints, &v1.FlowPoint{
			Timestamp: p.Timestamp.Format(time.RFC3339),
			Entries:   int32(p.Entries),
			Exits:     int32(p.Exits),
			NetFlow:   int32(p.NetFlow),
		})
	}

	return &v1.VehicleFlowData{
		LotId:           lotID.String(),
		Date:            req.Date,
		TotalEntries:    int32(totalEntries),
		TotalExits:      int32(totalExits),
		CurrentVehicles: int32(currentVehicles),
		FlowPoints:      flowPoints,
	}, nil
}

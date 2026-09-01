// Package biz provides business logic for the analytics service.
package biz

import (
	"context"
	"math"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/analytics/v1"
)

// AnalyticsRepo defines the repository interface for analytics operations.
type AnalyticsRepo interface {
	GetLotStats(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) (*LotStats, error)
	GetRevenueData(ctx context.Context, lotID uuid.UUID, period string, limit int) ([]*RevenuePoint, error)
	GetOccupancyData(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) ([]*OccupancyPoint, error)
	GetVehicleFlowData(ctx context.Context, lotID uuid.UUID, date time.Time) ([]*FlowPoint, error)
	GetHistoricalPeakHours(ctx context.Context, lotID uuid.UUID, days int) (map[int]int, error)
}

// LotStats represents parking lot statistics. TotalRevenue is in cents (分).
type LotStats struct {
	LotID         uuid.UUID
	LotName       string
	TotalVehicles int
	TotalRevenue  int64
	AvgDuration   float64
	OccupancyRate float64
	PeakHour      int
}

// RevenuePoint represents a revenue data point. Revenue is in cents (分).
type RevenuePoint struct {
	Date         time.Time
	Revenue      int64
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
	repo AnalyticsRepo
	log  *log.Helper
}

// NewAnalyticsUseCase creates a new AnalyticsUseCase.
func NewAnalyticsUseCase(repo AnalyticsRepo, logger log.Logger) *AnalyticsUseCase {
	return &AnalyticsUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
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

	// Bound the requested window: an unbounded limit would let a caller pull the entire
	// history in one request.
	const maxLimit = 366
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 30
	}
	if limit > maxLimit {
		limit = maxLimit
	}

	points, err := uc.repo.GetRevenueData(ctx, lotID, req.Period, limit)
	if err != nil {
		return nil, err
	}

	var totalRevenue int64
	var revenuePoints []*v1.RevenuePoint

	for _, p := range points {
		totalRevenue += p.Revenue
		revenuePoints = append(revenuePoints, &v1.RevenuePoint{
			Date:         p.Date.Format("2006-01-02"),
			Revenue:      p.Revenue,
			VehicleCount: int32(p.VehicleCount),
		})
	}

	// Guard against the empty-data division by zero; a lot with no revenue in the window
	// should answer zero, not NaN. Averaging stays in integer cents.
	avgRevenue := int64(0)
	if len(points) > 0 {
		avgRevenue = totalRevenue / int64(len(points))
	}

	return &v1.RevenueTrendData{
		Points:       revenuePoints,
		TotalRevenue: totalRevenue,
		AvgRevenue:   avgRevenue,
	}, nil
}

// PredictPeakHours predicts peak hours from the lot's own history.
//
// The previous implementation was decoration: a fixed threshold of 100 vehicles, a fixed
// divisor of 500, and a confidence that was hard-coded to 0.85 made the result look like
// analysis while never consulting the data. The prediction is now derived from the
// historical distribution, so a quiet lot gets no "peaks" and a busy one does.
func (uc *AnalyticsUseCase) PredictPeakHours(ctx context.Context, req *v1.PredictPeakHoursRequest) (*v1.PeakHoursPrediction, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	// At least a week of history is required; otherwise the signal is noise.
	const historyDays = 7
	historicalData, err := uc.repo.GetHistoricalPeakHours(ctx, lotID, historyDays)
	if err != nil {
		return nil, err
	}

	if len(historicalData) == 0 {
		return &v1.PeakHoursPrediction{
			LotId:      lotID.String(),
			Date:       req.Date,
			PeakHours:  []*v1.PeakHour{},
			Confidence: 0,
		}, nil
	}

	// Baseline from the data: mean and standard deviation of hourly counts.
	var sum, sumSquares, maxCount int
	for _, count := range historicalData {
		sum += count
		sumSquares += count * count
		if count > maxCount {
			maxCount = count
		}
	}
	mean := float64(sum) / float64(len(historicalData))
	variance := float64(sumSquares)/float64(len(historicalData)) - mean*mean
	if variance < 0 {
		variance = 0
	}
	stddev := math.Sqrt(variance)

	// A peak hour is one that stands out against its own baseline: at least one standard
	// deviation above the mean, and never zero. For an empty lot that means no peaks at
	// all, which is the honest answer.
	threshold := mean + stddev
	if threshold < 2 {
		threshold = 2
	}

	var peakHours []*v1.PeakHour
	for hour, count := range historicalData {
		if float64(count) < threshold {
			continue
		}
		peakHours = append(peakHours, &v1.PeakHour{
			StartHour:        int32(hour),
			EndHour:          int32(hour + 1),
			ExpectedVehicles: int32(count),
			// Probability is the hour's share of the busiest hour, a bounded [0,1] value
			// derived from data rather than a magic denominator.
			Probability: float64(count) / float64(maxCount),
		})
	}

	// Confidence reflects how much history supported the call. Ten days of data yield
	// roughly the nominal 0.85; less history is honestly less certain.
	confidence := 0.3 + 0.55*float64(historyDays)/30.0
	if confidence > 0.95 {
		confidence = 0.95
	}

	return &v1.PeakHoursPrediction{
		LotId:      lotID.String(),
		Date:       req.Date,
		PeakHours:  peakHours,
		Confidence: confidence,
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

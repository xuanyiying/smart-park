// Package data provides data access layer for the analytics service.
package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/analytics/biz"
)

// analyticsRepo implements biz.AnalyticsRepo.
type analyticsRepo struct {
	data *Data
}

// NewAnalyticsRepo creates a new AnalyticsRepo.
func NewAnalyticsRepo(data *Data) biz.AnalyticsRepo {
	return &analyticsRepo{data: data}
}

// GetLotStats retrieves parking lot statistics for a date range.
func (r *analyticsRepo) GetLotStats(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) (*biz.LotStats, error) {
	// Note: Actual analytics queries are currently implemented in the admin service (internal/admin/data/admin.go).
	// This analytics service is a placeholder for future decoupled analytics architecture.
	stats := &biz.LotStats{
		LotID:         lotID,
		LotName:       fmt.Sprintf("停车场 %s", lotID.String()[:8]),
		TotalVehicles: 150,
		TotalRevenue:  2500.50,
		AvgDuration:   2.5,
		OccupancyRate: 0.75,
		PeakHour:      18,
	}

	return stats, nil
}

// GetRevenueData retrieves revenue data for trend analysis.
func (r *analyticsRepo) GetRevenueData(ctx context.Context, lotID uuid.UUID, period string, limit int) ([]*biz.RevenuePoint, error) {
	points := make([]*biz.RevenuePoint, 0, limit)
	now := time.Now()

	for i := 0; i < limit; i++ {
		date := now.AddDate(0, 0, -i)
		points = append(points, &biz.RevenuePoint{
			Date:         date,
			Revenue:      float64(1000 + i*100),
			VehicleCount: 50 + i*5,
		})
	}

	return points, nil
}

// GetOccupancyData retrieves occupancy data for a date range.
func (r *analyticsRepo) GetOccupancyData(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) ([]*biz.OccupancyPoint, error) {
	points := make([]*biz.OccupancyPoint, 0)
	current := startDate

	for current.Before(endDate) || current.Equal(endDate) {
		points = append(points, &biz.OccupancyPoint{
			Timestamp:      current,
			Rate:           0.6 + float64(current.Hour())*0.02,
			OccupiedSpaces: 60 + current.Hour()*2,
			TotalSpaces:    100,
		})
		current = current.Add(time.Hour)
	}

	return points, nil
}

// GetVehicleFlowData retrieves vehicle flow data for a specific date.
func (r *analyticsRepo) GetVehicleFlowData(ctx context.Context, lotID uuid.UUID, date time.Time) ([]*biz.FlowPoint, error) {
	points := make([]*biz.FlowPoint, 0, 24)

	for hour := 0; hour < 24; hour++ {
		timestamp := time.Date(date.Year(), date.Month(), date.Day(), hour, 0, 0, 0, date.Location())
		entries := 10 + hour*2
		exits := 5 + hour

		points = append(points, &biz.FlowPoint{
			Timestamp: timestamp,
			Entries:   entries,
			Exits:     exits,
			NetFlow:   entries - exits,
		})
	}

	return points, nil
}

// GetHistoricalPeakHours retrieves historical peak hours data.
func (r *analyticsRepo) GetHistoricalPeakHours(ctx context.Context, lotID uuid.UUID, days int) (map[int]int, error) {
	peakHours := map[int]int{
		8:  150,
		9:  200,
		12: 180,
		13: 160,
		18: 250,
		19: 220,
		20: 180,
	}

	return peakHours, nil
}

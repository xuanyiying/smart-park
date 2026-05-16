package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/analytics/biz"
	"github.com/xuanyiying/smart-park/internal/analytics/data/ent/order"
	"github.com/xuanyiying/smart-park/internal/analytics/data/ent/parkingrecord"
)

type analyticsRepo struct {
	data *Data
}

func NewAnalyticsRepo(data *Data) biz.AnalyticsRepo {
	return &analyticsRepo{data: data}
}

func (r *analyticsRepo) GetLotStats(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) (*biz.LotStats, error) {
	startOfDay := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
	endOfDay := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())

	records, err := r.data.db.ParkingRecord.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.EntryTimeGTE(startOfDay),
			parkingrecord.EntryTimeLTE(endOfDay),
		).
		All(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to query parking records: %v", err)
		return nil, err
	}

	var totalVehicles, totalDuration int
	entryCounts := make(map[int]int)
	peakHour := 9
	peakCount := 0

	for _, rec := range records {
		totalVehicles++
		if rec.ParkingDuration > 0 {
			totalDuration += rec.ParkingDuration
		}

		hour := rec.EntryTime.Hour()
		entryCounts[hour]++
		if entryCounts[hour] > peakCount {
			peakCount = entryCounts[hour]
			peakHour = hour
		}
	}

	var totalRevenue float64
	orders, _ := r.data.db.Order.Query().
		Where(
			order.StatusEQ(order.StatusPaid),
			order.PayTimeNotNil(),
			order.PayTimeGTE(startOfDay),
			order.PayTimeLTE(endOfDay),
		).
		All(ctx)

	for _, o := range orders {
		totalRevenue += o.FinalAmount
	}

	avgDuration := 0.0
	if totalVehicles > 0 {
		avgDuration = float64(totalDuration) / float64(totalVehicles) / 3600.0
	}

	stats := &biz.LotStats{
		LotID:         lotID,
		LotName:       "停车场",
		TotalVehicles: totalVehicles,
		TotalRevenue:  totalRevenue,
		AvgDuration:   avgDuration,
		OccupancyRate: 0.0,
		PeakHour:      peakHour,
	}

	return stats, nil
}

func (r *analyticsRepo) GetRevenueData(ctx context.Context, lotID uuid.UUID, period string, limit int) ([]*biz.RevenuePoint, error) {
	var days int
	switch period {
	case "week":
		days = 7
	case "month":
		days = 30
	case "year":
		days = 365
	default:
		days = limit
	}

	points := make([]*biz.RevenuePoint, 0, days)
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
		endOfDay := startOfDay.Add(24 * time.Hour)

		var dailyRevenue float64
		var vehicleCount int

		orders, err := r.data.db.Order.Query().
			Where(
				order.StatusEQ(order.StatusPaid),
				order.PayTimeNotNil(),
				order.PayTimeGTE(startOfDay),
				order.PayTimeLT(endOfDay),
			).
			All(ctx)
		if err == nil {
			for _, o := range orders {
				dailyRevenue += o.FinalAmount
			}
			vehicleCount = len(orders)
		}

		entries, err := r.data.db.ParkingRecord.Query().
			Where(
				parkingrecord.LotID(lotID),
				parkingrecord.EntryTimeGTE(startOfDay),
				parkingrecord.EntryTimeLT(endOfDay),
			).
			Count(ctx)
		if err == nil && entries > vehicleCount {
			vehicleCount = entries
		}

		points = append(points, &biz.RevenuePoint{
			Date:         startOfDay,
			Revenue:      dailyRevenue,
			VehicleCount: vehicleCount,
		})
	}

	return points, nil
}

func (r *analyticsRepo) GetOccupancyData(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) ([]*biz.OccupancyPoint, error) {
	points := make([]*biz.OccupancyPoint, 0)
	current := time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())

	lot, err := r.data.db.ParkingLot.Get(ctx, lotID)
	totalSpaces := 100
	if err == nil && lot.TotalCapacity > 0 {
		totalSpaces = lot.TotalCapacity
	}

	for current.Before(endDate) || current.Equal(endDate) {
		hourStart := current
		hourEnd := hourStart.Add(time.Hour)

		entries, _ := r.data.db.ParkingRecord.Query().
			Where(
				parkingrecord.LotID(lotID),
				parkingrecord.EntryTimeGTE(hourStart),
				parkingrecord.EntryTimeLT(hourEnd),
			).
			Count(ctx)

		occupancyRate := float64(entries) / float64(totalSpaces)
		if occupancyRate > 1.0 {
			occupancyRate = 1.0
		}

		points = append(points, &biz.OccupancyPoint{
			Timestamp:      current,
			Rate:           occupancyRate,
			OccupiedSpaces: entries,
			TotalSpaces:    totalSpaces,
		})

		current = current.Add(time.Hour)
	}

	return points, nil
}

func (r *analyticsRepo) GetVehicleFlowData(ctx context.Context, lotID uuid.UUID, date time.Time) ([]*biz.FlowPoint, error) {
	points := make([]*biz.FlowPoint, 0, 24)
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

	for hour := 0; hour < 24; hour++ {
		hourStart := startOfDay.Add(time.Duration(hour) * time.Hour)
		hourEnd := hourStart.Add(time.Hour)

		entries, _ := r.data.db.ParkingRecord.Query().
			Where(
				parkingrecord.LotID(lotID),
				parkingrecord.EntryTimeGTE(hourStart),
				parkingrecord.EntryTimeLT(hourEnd),
			).
			Count(ctx)

		exits, _ := r.data.db.ParkingRecord.Query().
			Where(
				parkingrecord.LotID(lotID),
				parkingrecord.ExitTimeNotNil(),
				parkingrecord.ExitTimeGTE(hourStart),
				parkingrecord.ExitTimeLT(hourEnd),
			).
			Count(ctx)

		points = append(points, &biz.FlowPoint{
			Timestamp: hourStart,
			Entries:   entries,
			Exits:     exits,
			NetFlow:   entries - exits,
		})
	}

	return points, nil
}

func (r *analyticsRepo) GetHistoricalPeakHours(ctx context.Context, lotID uuid.UUID, days int) (map[int]int, error) {
	peakHours := make(map[int]int)
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

		for hour := 0; hour < 24; hour++ {
			hourStart := startOfDay.Add(time.Duration(hour) * time.Hour)
			hourEnd := hourStart.Add(time.Hour)

			count, err := r.data.db.ParkingRecord.Query().
				Where(
					parkingrecord.LotID(lotID),
					parkingrecord.EntryTimeGTE(hourStart),
					parkingrecord.EntryTimeLT(hourEnd),
				).
				Count(ctx)
			if err == nil {
				peakHours[hour] += count
			}
		}
	}

	return peakHours, nil
}

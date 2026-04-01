// Package ml provides machine learning models for parking analytics.
package ml

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// OccupancyPredictor predicts parking space occupancy.
type OccupancyPredictor interface {
	// PredictOccupancy predicts the occupancy rate for a parking lot at a specific time.
	PredictOccupancy(ctx context.Context, lotID uuid.UUID, timestamp time.Time) (float64, error)

	// PredictOccupancyTrend predicts the occupancy rate trend for a parking lot over a time range.
	PredictOccupancyTrend(ctx context.Context, lotID uuid.UUID, start time.Time, end time.Time, interval time.Duration) ([]OccupancyPrediction, error)
}

// PeakHourPredictor predicts peak hours for parking lots.
type PeakHourPredictor interface {
	// PredictPeakHours predicts peak hours for a parking lot on a specific date.
	PredictPeakHours(ctx context.Context, lotID uuid.UUID, date time.Time) ([]PeakHourPrediction, error)
}

// OccupancyPrediction represents a single occupancy prediction.
type OccupancyPrediction struct {
	Timestamp time.Time
	OccupancyRate float64
	Confidence float64
}

// PeakHourPrediction represents a peak hour prediction.
type PeakHourPrediction struct {
	StartHour int
	EndHour int
	ExpectedOccupancy float64
	Probability float64
}

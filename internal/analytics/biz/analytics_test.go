package biz

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/analytics/v1"
)

// stubAnalyticsRepo satisfies AnalyticsRepo with scriptable history.
type stubAnalyticsRepo struct {
	peakHours map[int]int
}

func (s *stubAnalyticsRepo) GetLotStats(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) (*LotStats, error) {
	return nil, nil
}
func (s *stubAnalyticsRepo) GetRevenueData(ctx context.Context, lotID uuid.UUID, period string, limit int) ([]*RevenuePoint, error) {
	return nil, nil
}
func (s *stubAnalyticsRepo) GetOccupancyData(ctx context.Context, lotID uuid.UUID, startDate, endDate time.Time) ([]*OccupancyPoint, error) {
	return nil, nil
}
func (s *stubAnalyticsRepo) GetVehicleFlowData(ctx context.Context, lotID uuid.UUID, date time.Time) ([]*FlowPoint, error) {
	return nil, nil
}
func (s *stubAnalyticsRepo) GetHistoricalPeakHours(ctx context.Context, lotID uuid.UUID, days int) (map[int]int, error) {
	return s.peakHours, nil
}

func TestPredictPeakHoursEmptyHistory(t *testing.T) {
	uc := NewAnalyticsUseCase(&stubAnalyticsRepo{peakHours: map[int]int{}}, log.NewStdLogger(io.Discard))

	result, err := uc.PredictPeakHours(context.Background(), &v1.PredictPeakHoursRequest{LotId: uuid.New().String(), Date: "2026-08-30"})
	if err != nil {
		t.Fatalf("PredictPeakHours failed: %v", err)
	}

	if len(result.PeakHours) != 0 {
		t.Errorf("expected no peak hours without history, got %d", len(result.PeakHours))
	}
	if result.Confidence != 0 {
		t.Errorf("confidence must be 0 with no history, got %.2f", result.Confidence)
	}
}

func TestPredictPeakHoursDetectsStandoutHours(t *testing.T) {
	// Most hours are quiet; 18:00 clearly stands out and must be reported.
	history := map[int]int{}
	for h := 0; h < 24; h++ {
		history[h] = 5
	}
	history[8] = 120  // morning rush
	history[18] = 150 // evening rush

	uc := NewAnalyticsUseCase(&stubAnalyticsRepo{peakHours: history}, log.NewStdLogger(io.Discard))

	result, err := uc.PredictPeakHours(context.Background(), &v1.PredictPeakHoursRequest{LotId: uuid.New().String(), Date: "2026-08-30"})
	if err != nil {
		t.Fatalf("PredictPeakHours failed: %v", err)
	}

	found := map[int32]bool{}
	for _, p := range result.PeakHours {
		found[p.StartHour] = true
		if p.Probability < 0 || p.Probability > 1 {
			t.Errorf("probability for hour %d out of range: %.2f", p.StartHour, p.Probability)
		}
	}

	if !found[8] || !found[18] {
		t.Errorf("expected peak hours 8 and 18, got %v", found)
	}
	if result.Confidence <= 0 || result.Confidence > 0.95 {
		t.Errorf("confidence out of range: %.2f", result.Confidence)
	}
}

func TestPredictPeakHoursQuietLotHasNoPeaks(t *testing.T) {
	history := map[int]int{}
	for h := 0; h < 24; h++ {
		history[h] = 1
	}

	uc := NewAnalyticsUseCase(&stubAnalyticsRepo{peakHours: history}, log.NewStdLogger(io.Discard))

	result, err := uc.PredictPeakHours(context.Background(), &v1.PredictPeakHoursRequest{LotId: uuid.New().String(), Date: "2026-08-30"})
	if err != nil {
		t.Fatalf("PredictPeakHours failed: %v", err)
	}

	if len(result.PeakHours) != 0 {
		t.Errorf("a uniformly quiet lot must not report peaks, got %d", len(result.PeakHours))
	}
}

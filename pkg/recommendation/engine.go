package recommendation

import (
	"context"
	"sort"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

type RecommendationType string

const (
	RecommendationTypeDiscount    RecommendationType = "discount"
	RecommendationTypeMonthlyCard RecommendationType = "monthly_card"
	RecommendationTypeCoupon      RecommendationType = "coupon"
	RecommendationTypeLoyalty     RecommendationType = "loyalty"
)

type Recommendation struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Type        RecommendationType
	Title       string
	Description string
	Benefit     float64
	ExpiresAt   time.Time
	CreatedAt   time.Time
}

type UserParkingPattern struct {
	UserID           uuid.UUID
	AvgParkingTime   float64
	FrequentLots     []uuid.UUID
	PeakHours        []int
	AvgMonthlySpend  float64
	ParkingFrequency int
}

type RecommendationEngine struct {
	lotRepo  LotRepo
	userRepo UserRepo
	log      *log.Helper
}

type LotRepo interface {
	GetLotByID(ctx context.Context, lotID uuid.UUID) (*LotInfo, error)
	GetLotStats(ctx context.Context, lotID uuid.UUID) (*LotStats, error)
}

type UserRepo interface {
	GetUserParkingHistory(ctx context.Context, userID uuid.UUID, days int) ([]*ParkingRecord, error)
	GetUserSpending(ctx context.Context, userID uuid.UUID, months int) (*SpendingData, error)
}

type LotInfo struct {
	ID          uuid.UUID
	Name        string
	TotalSpaces int
	HourlyRate  float64
	MonthlyRate float64
}

type LotStats struct {
	AvgOccupancyRate float64
	PeakHours        []int
	RevenuePerSpace  float64
}

type ParkingRecord struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	LotID       uuid.UUID
	Duration    int
	Amount      float64
	ParkingTime time.Time
}

type SpendingData struct {
	TotalAmount float64
	AvgPerMonth float64
	Trend       string
}

func NewRecommendationEngine(lotRepo LotRepo, userRepo UserRepo, logger log.Logger) *RecommendationEngine {
	return &RecommendationEngine{
		lotRepo:  lotRepo,
		userRepo: userRepo,
		log:      log.NewHelper(logger),
	}
}

func (e *RecommendationEngine) AnalyzeUserPattern(ctx context.Context, userID uuid.UUID) (*UserParkingPattern, error) {
	history, err := e.userRepo.GetUserParkingHistory(ctx, userID, 90)
	if err != nil {
		return nil, err
	}

	spending, err := e.userRepo.GetUserSpending(ctx, userID, 3)
	if err != nil {
		return nil, err
	}

	pattern := &UserParkingPattern{
		UserID: userID,
	}

	if len(history) > 0 {
		var totalDuration int
		lotCount := make(map[uuid.UUID]int)
		hourCount := make(map[int]int)

		for _, record := range history {
			totalDuration += record.Duration
			lotCount[record.LotID]++
			hourCount[record.ParkingTime.Hour()]++
		}

		pattern.AvgParkingTime = float64(totalDuration) / float64(len(history))
		pattern.ParkingFrequency = len(history)

		for lotID, count := range lotCount {
			if count > len(history)/3 {
				pattern.FrequentLots = append(pattern.FrequentLots, lotID)
			}
		}

		for hour, count := range hourCount {
			if count > len(history)/10 {
				pattern.PeakHours = append(pattern.PeakHours, hour)
			}
		}
	}

	if spending != nil {
		pattern.AvgMonthlySpend = spending.AvgPerMonth
	}

	return pattern, nil
}

func (e *RecommendationEngine) GetRecommendations(ctx context.Context, userID uuid.UUID) ([]*Recommendation, error) {
	pattern, err := e.AnalyzeUserPattern(ctx, userID)
	if err != nil {
		return nil, err
	}

	var recommendations []*Recommendation

	if pattern.AvgMonthlySpend > 300 {
		for _, lotID := range pattern.FrequentLots {
			lot, err := e.lotRepo.GetLotByID(ctx, lotID)
			if err != nil {
				continue
			}

			if lot.MonthlyRate > 0 && lot.MonthlyRate < pattern.AvgMonthlySpend*0.8 {
				recommendations = append(recommendations, &Recommendation{
					ID:          uuid.New(),
					UserID:      userID,
					Type:        RecommendationTypeMonthlyCard,
					Title:       "月卡优惠推荐",
					Description: "根据您的停车频率，购买月卡可节省更多费用",
					Benefit:     pattern.AvgMonthlySpend - lot.MonthlyRate,
					ExpiresAt:   time.Now().Add(7 * 24 * time.Hour),
					CreatedAt:   time.Now(),
				})
			}
		}
	}

	if pattern.ParkingFrequency > 20 {
		recommendations = append(recommendations, &Recommendation{
			ID:          uuid.New(),
			UserID:      userID,
			Type:        RecommendationTypeLoyalty,
			Title:       "会员积分奖励",
			Description: "您是高频用户，享受专属积分奖励",
			Benefit:     float64(pattern.ParkingFrequency) * 0.5,
			ExpiresAt:   time.Now().Add(30 * 24 * time.Hour),
			CreatedAt:   time.Now(),
		})
	}

	if len(pattern.PeakHours) > 0 {
		recommendations = append(recommendations, &Recommendation{
			ID:          uuid.New(),
			UserID:      userID,
			Type:        RecommendationTypeDiscount,
			Title:       "错峰停车优惠",
			Description: "避开高峰时段停车可享受折扣优惠",
			Benefit:     0.2,
			ExpiresAt:   time.Now().Add(14 * 24 * time.Hour),
			CreatedAt:   time.Now(),
		})
	}

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Benefit > recommendations[j].Benefit
	})

	return recommendations, nil
}

func (e *RecommendationEngine) GetOptimalLot(ctx context.Context, userID uuid.UUID, preferredLots []uuid.UUID) (*uuid.UUID, error) {
	pattern, err := e.AnalyzeUserPattern(ctx, userID)
	if err != nil {
		return nil, err
	}

	var bestLot *uuid.UUID
	var bestScore float64

	for _, lotID := range preferredLots {
		lot, err := e.lotRepo.GetLotByID(ctx, lotID)
		if err != nil {
			continue
		}

		stats, err := e.lotRepo.GetLotStats(ctx, lotID)
		if err != nil {
			continue
		}

		score := 100.0

		for _, frequentLot := range pattern.FrequentLots {
			if frequentLot == lotID {
				score += 20
			}
		}

		if stats.AvgOccupancyRate < 0.7 {
			score += 15
		}

		if lot.HourlyRate < pattern.AvgMonthlySpend/100 {
			score += 10
		}

		if bestLot == nil || score > bestScore {
			bestScore = score
			bestLot = &lotID
		}
	}

	return bestLot, nil
}

// Package ml provides machine learning models for parking analytics.
package ml

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/google/uuid"
)

// SimplePredictor is a simple predictor based on historical patterns with improved accuracy.
type SimplePredictor struct {
	// Historical data storage (in a real system, this would be a database)
	historicalData map[uuid.UUID]map[int]float64 // lotID -> hour -> average occupancy rate
	// Day of week adjustment factors
	dayAdjustmentFactors map[int]float64 // dayOfWeek (0=Sunday) -> adjustment factor
	// Seasonal adjustment factors
	seasonAdjustmentFactors map[int]float64 // month (1-12) -> adjustment factor
}

// NewSimplePredictor creates a new SimplePredictor with improved features.
func NewSimplePredictor() *SimplePredictor {
	// Initialize with some default historical data
	historicalData := make(map[uuid.UUID]map[int]float64)
	
	// Add default data for demonstration
	defaultLotID, _ := uuid.Parse("00000000-0000-0000-0000-000000000001")
	historicalData[defaultLotID] = map[int]float64{
		0:  0.1, 1:  0.05, 2:  0.05, 3:  0.05, 4:  0.05, 5:  0.1,
		6:  0.2, 7:  0.4, 8:  0.7, 9:  0.8, 10: 0.7, 11: 0.6,
		12: 0.8, 13: 0.7, 14: 0.6, 15: 0.6, 16: 0.7, 17: 0.8,
		18: 0.9, 19: 0.8, 20: 0.7, 21: 0.6, 22: 0.4, 23: 0.2,
	}
	
	// Initialize day of week adjustment factors
	dayAdjustmentFactors := map[int]float64{
		0: 0.8,  // Sunday - lower occupancy
		1: 1.0,  // Monday
		2: 1.0,  // Tuesday
		3: 1.0,  // Wednesday
		4: 1.0,  // Thursday
		5: 1.1,  // Friday - higher occupancy
		6: 1.2,  // Saturday - higher occupancy
	}
	
	// Initialize seasonal adjustment factors
	seasonAdjustmentFactors := map[int]float64{
		1:  0.9,  // January - lower occupancy
		2:  0.9,  // February - lower occupancy
		3:  1.0,  // March
		4:  1.0,  // April
		5:  1.0,  // May
		6:  1.1,  // June - higher occupancy
		7:  1.1,  // July - higher occupancy
		8:  1.1,  // August - higher occupancy
		9:  1.0,  // September
		10: 1.0,  // October
		11: 1.0,  // November
		12: 1.2,  // December - higher occupancy (holidays)
	}
	
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())
	
	return &SimplePredictor{
		historicalData:          historicalData,
		dayAdjustmentFactors:    dayAdjustmentFactors,
		seasonAdjustmentFactors: seasonAdjustmentFactors,
	}
}

// UpdateHistoricalData updates the historical data for a parking lot.
func (p *SimplePredictor) UpdateHistoricalData(lotID uuid.UUID, hour int, occupancyRate float64) {
	if _, exists := p.historicalData[lotID]; !exists {
		p.historicalData[lotID] = make(map[int]float64)
	}
	
	// Use exponential moving average to update historical data
	if existingRate, exists := p.historicalData[lotID][hour]; exists {
		// Weight: 0.7 for new data, 0.3 for existing data
		p.historicalData[lotID][hour] = 0.7*occupancyRate + 0.3*existingRate
	} else {
		p.historicalData[lotID][hour] = occupancyRate
	}
}

// PredictOccupancy predicts the occupancy rate for a parking lot at a specific time with improved accuracy.
func (p *SimplePredictor) PredictOccupancy(ctx context.Context, lotID uuid.UUID, timestamp time.Time) (float64, error) {
	hour := timestamp.Hour()
	dayOfWeek := int(timestamp.Weekday())
	month := int(timestamp.Month())
	
	// Get historical data for the lot
	lotData, exists := p.historicalData[lotID]
	if !exists {
		// Use default pattern if no data exists
		lotData = map[int]float64{
			0:  0.1, 1:  0.05, 2:  0.05, 3:  0.05, 4:  0.05, 5:  0.1,
			6:  0.2, 7:  0.4, 8:  0.7, 9:  0.8, 10: 0.7, 11: 0.6,
			12: 0.8, 13: 0.7, 14: 0.6, 15: 0.6, 16: 0.7, 17: 0.8,
			18: 0.9, 19: 0.8, 20: 0.7, 21: 0.6, 22: 0.4, 23: 0.2,
		}
	}
	
	// Get base occupancy for the hour
	baseOccupancy, exists := lotData[hour]
	if !exists {
		// Interpolate between neighboring hours
		prevHour := (hour - 1 + 24) % 24
		nextHour := (hour + 1) % 24
		
		prevOccupancy, prevExists := lotData[prevHour]
		nextOccupancy, nextExists := lotData[nextHour]
		
		if prevExists && nextExists {
			baseOccupancy = (prevOccupancy + nextOccupancy) / 2
		} else if prevExists {
			baseOccupancy = prevOccupancy
		} else if nextExists {
			baseOccupancy = nextOccupancy
		} else {
			baseOccupancy = 0.5 // Default to 50% if no data
		}
	}
	
	// Apply day of week adjustment
	dayAdjustment, dayExists := p.dayAdjustmentFactors[dayOfWeek]
	if !dayExists {
		dayAdjustment = 1.0
	}
	
	// Apply seasonal adjustment
	seasonAdjustment, seasonExists := p.seasonAdjustmentFactors[month]
	if !seasonExists {
		seasonAdjustment = 1.0
	}
	
	// Calculate adjusted occupancy
	adjustedOccupancy := baseOccupancy * dayAdjustment * seasonAdjustment
	
	// Add time-of-day factor (minute-based)
	minuteFactor := 1.0 + 0.1*math.Sin(float64(timestamp.Minute())*math.Pi/30.0)
	adjustedOccupancy *= minuteFactor
	
	// Add small random variation for realism (±5%)
	randomVariation := 0.95 + 0.1*rand.Float64()
	adjustedOccupancy *= randomVariation
	
	// Ensure occupancy is between 0 and 1
	adjustedOccupancy = math.Max(0, math.Min(1, adjustedOccupancy))
	
	return adjustedOccupancy, nil
}

// PredictOccupancyTrend predicts the occupancy rate trend for a parking lot over a time range.
func (p *SimplePredictor) PredictOccupancyTrend(ctx context.Context, lotID uuid.UUID, start time.Time, end time.Time, interval time.Duration) ([]OccupancyPrediction, error) {
	var predictions []OccupancyPrediction
	
	current := start
	for current.Before(end) {
		occupancy, err := p.PredictOccupancy(ctx, lotID, current)
		if err != nil {
			continue
		}
		
		// Calculate confidence based on time distance from now
		confidence := 1.0 - math.Min(0.3, time.Since(current).Hours()/72.0)
		
		predictions = append(predictions, OccupancyPrediction{
			Timestamp:    current,
			OccupancyRate: occupancy,
			Confidence:   confidence,
		})
		
		current = current.Add(interval)
	}
	
	return predictions, nil
}

// PredictPeakHours predicts peak hours for a parking lot on a specific date with improved accuracy.
func (p *SimplePredictor) PredictPeakHours(ctx context.Context, lotID uuid.UUID, date time.Time) ([]PeakHourPrediction, error) {
	var peakHours []PeakHourPrediction
	
	// Get historical data for the lot
	lotData, exists := p.historicalData[lotID]
	if !exists {
		// Use default pattern if no data exists
		lotData = map[int]float64{
			0:  0.1, 1:  0.05, 2:  0.05, 3:  0.05, 4:  0.05, 5:  0.1,
			6:  0.2, 7:  0.4, 8:  0.7, 9:  0.8, 10: 0.7, 11: 0.6,
			12: 0.8, 13: 0.7, 14: 0.6, 15: 0.6, 16: 0.7, 17: 0.8,
			18: 0.9, 19: 0.8, 20: 0.7, 21: 0.6, 22: 0.4, 23: 0.2,
		}
	}
	
	// Apply day and season adjustments to get predicted occupancy for each hour
	predictedOccupancy := make(map[int]float64)
	dayOfWeek := int(date.Weekday())
	month := int(date.Month())
	
	// Get adjustment factors
	dayAdjustment, dayExists := p.dayAdjustmentFactors[dayOfWeek]
	if !dayExists {
		dayAdjustment = 1.0
	}
	
	seasonAdjustment, seasonExists := p.seasonAdjustmentFactors[month]
	if !seasonExists {
		seasonAdjustment = 1.0
	}
	
	// Calculate predicted occupancy for each hour
	for hour := 0; hour < 24; hour++ {
		baseOccupancy, exists := lotData[hour]
		if !exists {
			// Interpolate if no data for this hour
			prevHour := (hour - 1 + 24) % 24
			nextHour := (hour + 1) % 24
			
			prevOccupancy, prevExists := lotData[prevHour]
			nextOccupancy, nextExists := lotData[nextHour]
			
			if prevExists && nextExists {
				baseOccupancy = (prevOccupancy + nextOccupancy) / 2
			} else if prevExists {
				baseOccupancy = prevOccupancy
			} else if nextExists {
				baseOccupancy = nextOccupancy
			} else {
				baseOccupancy = 0.5
			}
		}
		
		predictedOccupancy[hour] = baseOccupancy * dayAdjustment * seasonAdjustment
	}
	
	// Identify peak hours (occupancy > 0.7)
	for hour := 0; hour < 23; hour++ {
		occupancy := predictedOccupancy[hour]
		if occupancy > 0.7 {
			// Check if this is part of a continuous peak period
			endHour := hour + 1
			for endHour < 24 {
				if predictedOccupancy[endHour] <= 0.7 {
					break
				}
				endHour++
			}
			
			// Calculate average occupancy for the peak period
			totalOccupancy := 0.0
			for h := hour; h < endHour; h++ {
				totalOccupancy += predictedOccupancy[h]
			}
			averageOccupancy := totalOccupancy / float64(endHour-hour)
			
			// Calculate probability based on historical consistency
			probability := 0.7 + 0.2*averageOccupancy
			probability = math.Min(0.95, probability)
			
			peakHours = append(peakHours, PeakHourPrediction{
				StartHour:        hour,
				EndHour:          endHour,
				ExpectedOccupancy: averageOccupancy,
				Probability:      probability,
			})
			
			// Skip the remaining hours in this peak period
			hour = endHour - 1
		}
	}
	
	return peakHours, nil
}

// EvaluateModel evaluates the model's performance using historical data.
func (p *SimplePredictor) EvaluateModel(ctx context.Context, lotID uuid.UUID, testData []struct {
	Timestamp     time.Time
	ActualOccupancy float64
}) (float64, error) {
	if len(testData) == 0 {
		return 0, nil
	}
	
	var totalError float64
	for _, data := range testData {
		predicted, err := p.PredictOccupancy(ctx, lotID, data.Timestamp)
		if err != nil {
			continue
		}
		error := math.Abs(predicted - data.ActualOccupancy)
		totalError += error
	}
	
	meanAbsoluteError := totalError / float64(len(testData))
	return meanAbsoluteError, nil
}

// OptimizeModel optimizes the model parameters based on training data.
func (p *SimplePredictor) OptimizeModel(trainingData []struct {
	LotID          uuid.UUID
	Timestamp      time.Time
	ActualOccupancy float64
}) {
	// In a real system, this would implement parameter optimization
	// For simplicity, we'll just update the historical data
	for _, data := range trainingData {
		hour := data.Timestamp.Hour()
		p.UpdateHistoricalData(data.LotID, hour, data.ActualOccupancy)
	}
}

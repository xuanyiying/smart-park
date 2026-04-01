// Package data provides data access layer for the analytics service.
package data

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// DataCollector collects real-time data from various sources.
type DataCollector struct {
	db    *Data
	redis *redis.Client
}

// NewDataCollector creates a new DataCollector.
func NewDataCollector(data *Data, redisClient *redis.Client) *DataCollector {
	return &DataCollector{
		db:    data,
		redis: redisClient,
	}
}

// ParkingEvent represents a parking event.
type ParkingEvent struct {
	EventID     string    `json:"event_id"`
	LotID       string    `json:"lot_id"`
	VehicleID   string    `json:"vehicle_id"`
	PlateNumber string    `json:"plate_number"`
	EventType   string    `json:"event_type"` // entry, exit, payment
	Timestamp   time.Time `json:"timestamp"`
	Amount      float64   `json:"amount,omitempty"`
	Duration    int       `json:"duration,omitempty"` // in minutes
}

// CollectParkingEvent collects a parking event.
func (c *DataCollector) CollectParkingEvent(ctx context.Context, event *ParkingEvent) error {
	// Store event in Redis for real-time processing
	eventJSON, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Add to Redis stream for processing
	if c.redis != nil {
		err = c.redis.XAdd(ctx, &redis.XAddArgs{
			Stream: "parking_events",
			Values: map[string]interface{}{
				"event": string(eventJSON),
			},
		}).Err()
		if err != nil {
			log.Printf("Failed to add event to Redis: %v", err)
		}
	}

	// Process event immediately for real-time analytics
	go c.ProcessParkingEvent(ctx, event)

	return nil
}

// ProcessParkingEvent processes a parking event.
func (c *DataCollector) ProcessParkingEvent(ctx context.Context, event *ParkingEvent) error {
	lotID, err := uuid.Parse(event.LotID)
	if err != nil {
		return err
	}

	switch event.EventType {
	case "entry":
		return c.processEntryEvent(ctx, lotID, event)
	case "exit":
		return c.processExitEvent(ctx, lotID, event)
	case "payment":
		return c.processPaymentEvent(ctx, lotID, event)
	default:
		return nil
	}
}

// processEntryEvent processes an entry event.
func (c *DataCollector) processEntryEvent(ctx context.Context, lotID uuid.UUID, event *ParkingEvent) error {
	// Update real-time occupancy
	if c.redis != nil {
		key := fmt.Sprintf("parking:occupancy:%s", lotID)
		c.redis.Incr(ctx, key)
		c.redis.Expire(ctx, key, 24*time.Hour)
	}

	// Update hourly vehicle count
	hourKey := fmt.Sprintf("parking:hourly:entries:%s:%s", lotID, event.Timestamp.Format("2006-01-02-15"))
	if c.redis != nil {
		c.redis.Incr(ctx, hourKey)
		c.redis.Expire(ctx, hourKey, 7*24*time.Hour)
	}

	return nil
}

// processExitEvent processes an exit event.
func (c *DataCollector) processExitEvent(ctx context.Context, lotID uuid.UUID, event *ParkingEvent) error {
	// Update real-time occupancy
	if c.redis != nil {
		key := fmt.Sprintf("parking:occupancy:%s", lotID)
		c.redis.Decr(ctx, key)
		c.redis.Expire(ctx, key, 24*time.Hour)
	}

	// Update hourly vehicle count
	hourKey := fmt.Sprintf("parking:hourly:exits:%s:%s", lotID, event.Timestamp.Format("2006-01-02-15"))
	if c.redis != nil {
		c.redis.Incr(ctx, hourKey)
		c.redis.Expire(ctx, hourKey, 7*24*time.Hour)
	}

	// Update average duration
	if event.Duration > 0 && c.redis != nil {
		durationKey := fmt.Sprintf("parking:avg_duration:%s", lotID)
		c.redis.ZAdd(ctx, durationKey, &redis.Z{
			Score:  float64(event.Duration),
			Member: event.EventID,
		})
		c.redis.Expire(ctx, durationKey, 7*24*time.Hour)
	}

	return nil
}

// processPaymentEvent processes a payment event.
func (c *DataCollector) processPaymentEvent(ctx context.Context, lotID uuid.UUID, event *ParkingEvent) error {
	// Update revenue
	if event.Amount > 0 && c.redis != nil {
		revenueKey := fmt.Sprintf("parking:revenue:%s:%s", lotID, event.Timestamp.Format("2006-01-02"))
		c.redis.IncrByFloat(ctx, revenueKey, event.Amount)
		c.redis.Expire(ctx, revenueKey, 30*24*time.Hour)
	}

	// Update payment count
	paymentKey := fmt.Sprintf("parking:payments:%s:%s", lotID, event.Timestamp.Format("2006-01-02"))
	if c.redis != nil {
		c.redis.Incr(ctx, paymentKey)
		c.redis.Expire(ctx, paymentKey, 30*24*time.Hour)
	}

	return nil
}

// GetRealTimeOccupancy gets real-time occupancy for a parking lot.
func (c *DataCollector) GetRealTimeOccupancy(ctx context.Context, lotID uuid.UUID) (int, error) {
	if c.redis == nil {
		return 0, nil
	}

	key := fmt.Sprintf("parking:occupancy:%s", lotID)
	val, err := c.redis.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return val, nil
}

// GetHourlyStats gets hourly statistics for a parking lot.
func (c *DataCollector) GetHourlyStats(ctx context.Context, lotID uuid.UUID, date time.Time) (map[string]int, error) {
	if c.redis == nil {
		return nil, nil
	}

	stats := make(map[string]int)

	// Get entries for each hour
	for hour := 0; hour < 24; hour++ {
		hourKey := fmt.Sprintf("parking:hourly:entries:%s:%s-%02d", lotID, date.Format("2006-01-02"), hour)
		val, err := c.redis.Get(ctx, hourKey).Int()
		if err == redis.Nil {
			stats[fmt.Sprintf("%02d:00-entries", hour)] = 0
		} else if err != nil {
			stats[fmt.Sprintf("%02d:00-entries", hour)] = 0
		} else {
			stats[fmt.Sprintf("%02d:00-entries", hour)] = val
		}

		// Get exits for each hour
		exitKey := fmt.Sprintf("parking:hourly:exits:%s:%s-%02d", lotID, date.Format("2006-01-02"), hour)
		val, err = c.redis.Get(ctx, exitKey).Int()
		if err == redis.Nil {
			stats[fmt.Sprintf("%02d:00-exits", hour)] = 0
		} else if err != nil {
			stats[fmt.Sprintf("%02d:00-exits", hour)] = 0
		} else {
			stats[fmt.Sprintf("%02d:00-exits", hour)] = val
		}
	}

	return stats, nil
}

// GetDailyRevenue gets daily revenue for a parking lot.
func (c *DataCollector) GetDailyRevenue(ctx context.Context, lotID uuid.UUID, date time.Time) (float64, error) {
	if c.redis == nil {
		return 0, nil
	}

	revenueKey := fmt.Sprintf("parking:revenue:%s:%s", lotID, date.Format("2006-01-02"))
	val, err := c.redis.Get(ctx, revenueKey).Float64()
	if err == redis.Nil {
		return 0, nil
	} else if err != nil {
		return 0, err
	}

	return val, nil
}

// StartProcessing starts processing events from Redis stream.
func (c *DataCollector) StartProcessing(ctx context.Context) error {
	if c.redis == nil {
		return nil
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				streams, err := c.redis.XRead(ctx, &redis.XReadArgs{
					Streams: []string{"parking_events", "0"},
					Block:   5 * time.Second,
				}).Result()

				if err != nil {
					if err == redis.Nil {
						continue
					}
					log.Printf("Failed to read from Redis stream: %v", err)
					continue
				}

				for _, stream := range streams {
					for _, message := range stream.Messages {
						eventJSON := message.Values["event"].(string)
						var event ParkingEvent
						if err := json.Unmarshal([]byte(eventJSON), &event); err != nil {
							log.Printf("Failed to unmarshal event: %v", err)
							continue
						}

						// Process event
						if err := c.ProcessParkingEvent(ctx, &event); err != nil {
							log.Printf("Failed to process event: %v", err)
						}

						// Acknowledge message
						c.redis.XDel(ctx, "parking_events", message.ID)
					}
				}
			}
		}
	}()

	return nil
}
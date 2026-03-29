package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
)

const (
	VehicleKeyPrefix   = "vehicle:"
	VehicleCacheTTL    = 1 * time.Hour
	VehicleLockTimeout = 10 * time.Second
)

type VehicleCache struct {
	cache Cache
}

func NewVehicleCache(cache Cache) *VehicleCache {
	return &VehicleCache{cache: cache}
}

func (c *VehicleCache) GetVehicle(ctx context.Context, plateNumber string) (*biz.Vehicle, error) {
	key := fmt.Sprintf("%s%s", VehicleKeyPrefix, plateNumber)

	data, err := c.cache.Get(ctx, key)
	if err != nil {
		if err == ErrCacheMiss {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	var vehicle biz.Vehicle
	if err := json.Unmarshal([]byte(data), &vehicle); err != nil {
		return nil, err
	}

	return &vehicle, nil
}

func (c *VehicleCache) SetVehicle(ctx context.Context, vehicle *biz.Vehicle) error {
	key := fmt.Sprintf("%s%s", VehicleKeyPrefix, vehicle.PlateNumber)
	data, err := json.Marshal(vehicle)
	if err != nil {
		return err
	}
	return c.cache.Set(ctx, key, string(data), VehicleCacheTTL)
}

func (c *VehicleCache) DeleteVehicle(ctx context.Context, plateNumber string) error {
	key := fmt.Sprintf("%s%s", VehicleKeyPrefix, plateNumber)
	return c.cache.Delete(ctx, key)
}

func (c *VehicleCache) GetOrLoadVehicle(ctx context.Context, plateNumber string, loader func() (*biz.Vehicle, error)) (*biz.Vehicle, error) {
	vehicle, err := c.GetVehicle(ctx, plateNumber)
	if err == nil {
		return vehicle, nil
	}
	if err != ErrCacheMiss {
		return nil, err
	}

	// Load from database
	vehicle, err = loader()
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := c.SetVehicle(ctx, vehicle); err != nil {
		// Log warning but don't fail
		_ = err
	}

	return vehicle, nil
}

func (c *VehicleCache) AcquireVehicleLock(ctx context.Context, plateNumber string) (bool, error) {
	key := fmt.Sprintf("%s%s:lock", VehicleKeyPrefix, plateNumber)
	return c.cache.SetNX(ctx, key, "locked", VehicleLockTimeout)
}

func (c *VehicleCache) ReleaseVehicleLock(ctx context.Context, plateNumber string) error {
	key := fmt.Sprintf("%s%s:lock", VehicleKeyPrefix, plateNumber)
	return c.cache.Delete(ctx, key)
}

func (c *VehicleCache) InvalidateVehicle(ctx context.Context, plateNumber string) error {
	return c.DeleteVehicle(ctx, plateNumber)
}

func (c *VehicleCache) InvalidateAllVehicles(ctx context.Context) error {
	// Note: This requires pattern deletion which is not supported by the Cache interface
	// For now, we just return nil. In production, you might want to use Redis-specific implementation
	return nil
}

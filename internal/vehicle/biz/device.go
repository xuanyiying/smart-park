// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// DeviceUseCase handles device management business logic.
type DeviceUseCase struct {
	vehicleRepo VehicleRepo
	config      *Config
	log         *log.Helper
}

// NewDeviceUseCase creates a new DeviceUseCase.
func NewDeviceUseCase(vehicleRepo VehicleRepo, logger log.Logger) *DeviceUseCase {
	return &DeviceUseCase{
		vehicleRepo: vehicleRepo,
		config:      DefaultConfig(),
		log:         log.NewHelper(logger),
	}
}

// Heartbeat handles device heartbeat.
func (uc *DeviceUseCase) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) error {
	if req.DeviceId == "" {
		return fmt.Errorf("device id is required")
	}
	if err := uc.vehicleRepo.UpdateDeviceHeartbeat(ctx, req.DeviceId); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update heartbeat: %v", err)
		return fmt.Errorf("failed to update device heartbeat: %w", err)
	}
	return nil
}

// GetDeviceStatus retrieves device status.
func (uc *DeviceUseCase) GetDeviceStatus(ctx context.Context, deviceID string) (*v1.DeviceStatus, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	online := true
	if device.LastHeartbeat != nil {
		online = time.Since(*device.LastHeartbeat) < uc.config.DeviceOnlineThreshold
	}

	var lastHeartbeat string
	if device.LastHeartbeat != nil {
		lastHeartbeat = device.LastHeartbeat.Format(time.RFC3339)
	}

	return &v1.DeviceStatus{
		DeviceId:      device.DeviceID,
		Online:        online,
		Status:        device.Status,
		LastHeartbeat: lastHeartbeat,
	}, nil
}

// ListDevices lists all devices with pagination.
func (uc *DeviceUseCase) ListDevices(ctx context.Context, page, pageSize int) ([]*v1.DeviceInfo, int, error) {
	devices, total, err := uc.vehicleRepo.ListDevices(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*v1.DeviceInfo, len(devices))
	for i, d := range devices {
		online := false
		if d.LastHeartbeat != nil {
			online = time.Since(*d.LastHeartbeat) < uc.config.DeviceOnlineThreshold
		}

		var lastHeartbeat string
		if d.LastHeartbeat != nil {
			lastHeartbeat = d.LastHeartbeat.Format(time.RFC3339)
		}

		laneID := ""
		if d.LaneID != nil {
			laneID = d.LaneID.String()
		}
		lotID := ""
		if d.LotID != nil {
			lotID = d.LotID.String()
		}

		result[i] = &v1.DeviceInfo{
			DeviceId:      d.DeviceID,
			Status:        d.Status,
			LastHeartbeat: lastHeartbeat,
			Online:        online,
			LaneId:        laneID,
			LotId:         lotID,
		}
	}

	return result, total, nil
}

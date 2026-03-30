// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
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
		result[i] = uc.convertDeviceToProto(d)
	}

	return result, total, nil
}

// convertDeviceToProto converts biz.Device to v1.DeviceInfo.
func (uc *DeviceUseCase) convertDeviceToProto(d *Device) *v1.DeviceInfo {
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

	return &v1.DeviceInfo{
		DeviceId:      d.DeviceID,
		Status:        d.Status,
		LastHeartbeat: lastHeartbeat,
		Online:        online,
		LaneId:        laneID,
		LotId:         lotID,
		DeviceType:    d.DeviceType,
	}
}

// CreateDevice creates a new device.
func (uc *DeviceUseCase) CreateDevice(ctx context.Context, req *v1.CreateDeviceRequest) (*v1.DeviceInfo, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Check if device already exists
	existing, err := uc.vehicleRepo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing device: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("device already exists: %s", req.DeviceId)
	}

	var lotID, laneID *uuid.UUID
	if req.LotId != "" {
		id, err := uuid.Parse(req.LotId)
		if err != nil {
			return nil, fmt.Errorf("invalid lot id: %w", err)
		}
		lotID = &id
	}
	if req.LaneId != "" {
		id, err := uuid.Parse(req.LaneId)
		if err != nil {
			return nil, fmt.Errorf("invalid lane id: %w", err)
		}
		laneID = &id
	}

	device := &Device{
		DeviceID:   req.DeviceId,
		DeviceType: req.DeviceType,
		LotID:      lotID,
		LaneID:     laneID,
		Status:     req.Status,
	}

	if device.Status == "" {
		device.Status = "active"
	}

	if err := uc.vehicleRepo.CreateDevice(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	// Get the created device
	created, err := uc.vehicleRepo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get created device: %w", err)
	}

	return uc.convertDeviceToProto(created), nil
}

// GetDevice retrieves a device by ID.
func (uc *DeviceUseCase) GetDevice(ctx context.Context, deviceID string) (*v1.DeviceInfo, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	device, err := uc.vehicleRepo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	return uc.convertDeviceToProto(device), nil
}

// UpdateDevice updates an existing device.
func (uc *DeviceUseCase) UpdateDevice(ctx context.Context, req *v1.UpdateDeviceRequest) (*v1.DeviceInfo, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Check if device exists
	existing, err := uc.vehicleRepo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing device: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("device not found: %s", req.DeviceId)
	}

	var lotID, laneID *uuid.UUID
	if req.LotId != "" {
		id, err := uuid.Parse(req.LotId)
		if err != nil {
			return nil, fmt.Errorf("invalid lot id: %w", err)
		}
		lotID = &id
	}
	if req.LaneId != "" {
		id, err := uuid.Parse(req.LaneId)
		if err != nil {
			return nil, fmt.Errorf("invalid lane id: %w", err)
		}
		laneID = &id
	}

	device := &Device{
		DeviceID:   req.DeviceId,
		DeviceType: req.DeviceType,
		LotID:      lotID,
		LaneID:     laneID,
		Status:     req.Status,
	}

	if err := uc.vehicleRepo.UpdateDevice(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	// Get the updated device
	updated, err := uc.vehicleRepo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated device: %w", err)
	}

	return uc.convertDeviceToProto(updated), nil
}

// DeleteDevice deletes a device by ID.
func (uc *DeviceUseCase) DeleteDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}

	// Check if device exists
	existing, err := uc.vehicleRepo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to check existing device: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	if err := uc.vehicleRepo.DeleteDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

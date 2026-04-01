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
	mqttClient  MqttClient
}

// MqttClient MQTT客户端接口
type MqttClient interface {
	// Publish 发布消息
	Publish(ctx context.Context, topic string, payload interface{}) error
	// Subscribe 订阅消息
	Subscribe(ctx context.Context, topic string, handler func(topic string, payload []byte)) error
	// Close 关闭连接
	Close() error
}

// NewDeviceUseCase creates a new DeviceUseCase.
func NewDeviceUseCase(vehicleRepo VehicleRepo, mqttClient MqttClient, logger log.Logger) *DeviceUseCase {
	return &DeviceUseCase{
		vehicleRepo: vehicleRepo,
		config:      DefaultConfig(),
		log:         log.NewHelper(logger),
		mqttClient:  mqttClient,
	}
}

// Heartbeat handles device heartbeat.
func (uc *DeviceUseCase) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) error {
	if req.DeviceId == "" {
		return fmt.Errorf("device id is required")
	}
	
	// 更新设备心跳
	if err := uc.vehicleRepo.UpdateDeviceHeartbeat(ctx, req.DeviceId); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update heartbeat: %v", err)
		return fmt.Errorf("failed to update device heartbeat: %w", err)
	}
	
	// 处理设备状态和故障信息
	if req.Status != "" {
		// 更新设备状态
		if err := uc.vehicleRepo.UpdateDeviceStatus(ctx, req.DeviceId, req.Status); err != nil {
			uc.log.WithContext(ctx).Errorf("failed to update device status: %v", err)
		}
	}
	
	// 处理故障信息
	if req.FaultCode != "" {
		// 记录故障信息
		if err := uc.vehicleRepo.UpdateDeviceFault(ctx, req.DeviceId, req.FaultCode, req.FaultMessage); err != nil {
			uc.log.WithContext(ctx).Errorf("failed to update device fault: %v", err)
		}
		
		// 记录故障日志
		if err := uc.vehicleRepo.CreateDeviceLog(ctx, req.DeviceId, "error", "error", req.FaultMessage, req.FaultCode, nil); err != nil {
			uc.log.WithContext(ctx).Errorf("failed to create device log: %v", err)
		}
	}
	
	// 处理设备统计信息
	if len(req.Stats) > 0 {
		if err := uc.vehicleRepo.UpdateDeviceStats(ctx, req.DeviceId, req.Stats); err != nil {
			uc.log.WithContext(ctx).Errorf("failed to update device stats: %v", err)
		}
	}
	
	// 处理设备版本信息
	if req.FirmwareVersion != "" {
		if err := uc.vehicleRepo.UpdateDeviceVersion(ctx, req.DeviceId, req.FirmwareVersion, req.HardwareVersion); err != nil {
			uc.log.WithContext(ctx).Errorf("failed to update device version: %v", err)
		}
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

// UpgradeDevice handles device remote upgrade.
func (uc *DeviceUseCase) UpgradeDevice(ctx context.Context, req *v1.UpgradeDeviceRequest) (*v1.UpgradeDeviceResponse, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}
	if req.FirmwareVersion == "" {
		return nil, fmt.Errorf("firmware version is required")
	}
	if req.FirmwareUrl == "" {
		return nil, fmt.Errorf("firmware url is required")
	}

	// Check if device exists
	device, err := uc.vehicleRepo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", req.DeviceId)
	}

	// Create upgrade record
	upgradeID, err := uc.vehicleRepo.CreateDeviceUpgrade(ctx, req.DeviceId, device.FirmwareVersion, req.FirmwareVersion, req.FirmwareUrl)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create upgrade record: %v", err)
		return nil, fmt.Errorf("failed to create upgrade record: %w", err)
	}

	// Update device status to upgrading
	if err := uc.vehicleRepo.UpdateDeviceStatus(ctx, req.DeviceId, "upgrading"); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update device status: %v", err)
	}

	// Send upgrade command to device via MQTT
	upgradeCmd := map[string]interface{}{
		"command":         "upgrade",
		"firmware_url":    req.FirmwareUrl,
		"firmware_version": req.FirmwareVersion,
		"upgrade_id":      upgradeID.String(),
		"timestamp":       time.Now().Unix(),
	}

	topic := fmt.Sprintf("devices/%s/commands", req.DeviceId)
	if err := uc.mqttClient.Publish(ctx, topic, upgradeCmd); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to send upgrade command: %v", err)
		// Update upgrade status to failed
		_ = uc.vehicleRepo.UpdateDeviceUpgradeStatus(ctx, upgradeID, "failed", fmt.Sprintf("Failed to send upgrade command: %v", err))
		// Update device status back to active
		_ = uc.vehicleRepo.UpdateDeviceStatus(ctx, req.DeviceId, "active")
		return nil, fmt.Errorf("failed to send upgrade command: %w", err)
	}

	uc.log.WithContext(ctx).Infof("Upgrade command sent to device %s: %s", req.DeviceId, req.FirmwareVersion)

	return &v1.UpgradeDeviceResponse{
		UpgradeId: upgradeID.String(),
		Status:    "in_progress",
		Message:   "Upgrade command sent successfully",
	}, nil
}

// GetDeviceUpgradeStatus retrieves device upgrade status.
func (uc *DeviceUseCase) GetDeviceUpgradeStatus(ctx context.Context, upgradeID string) (*v1.UpgradeStatusResponse, error) {
	if upgradeID == "" {
		return nil, fmt.Errorf("upgrade id is required")
	}

	upgradeUUID, err := uuid.Parse(upgradeID)
	if err != nil {
		return nil, fmt.Errorf("invalid upgrade id: %w", err)
	}

	upgrade, err := uc.vehicleRepo.GetDeviceUpgrade(ctx, upgradeUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get upgrade record: %w", err)
	}
	if upgrade == nil {
		return nil, fmt.Errorf("upgrade record not found: %s", upgradeID)
	}

	return &v1.UpgradeStatusResponse{
		UpgradeId:     upgradeID,
		DeviceId:      upgrade.DeviceID,
		FromVersion:   upgrade.FromVersion,
		ToVersion:     upgrade.ToVersion,
		Status:        upgrade.Status,
		ErrorMessage:  upgrade.ErrorMessage,
		StartTime:     upgrade.StartTime.Format(time.RFC3339),
		EndTime:       func() string {
			if upgrade.EndTime != nil {
				return upgrade.EndTime.Format(time.RFC3339)
			}
			return ""
		}(),
		Duration:      upgrade.Duration,
	}, nil
}

// GetDeviceLogs retrieves device logs.
func (uc *DeviceUseCase) GetDeviceLogs(ctx context.Context, deviceID string, page, pageSize int) ([]*v1.DeviceLog, int, error) {
	if deviceID == "" {
		return nil, 0, fmt.Errorf("device id is required")
	}

	logs, total, err := uc.vehicleRepo.GetDeviceLogs(ctx, deviceID, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get device logs: %w", err)
	}

	result := make([]*v1.DeviceLog, len(logs))
	for i, log := range logs {
		result[i] = &v1.DeviceLog{
			Id:         log.ID.String(),
			DeviceId:   log.DeviceID,
			LogType:    log.LogType,
			LogLevel:   log.LogLevel,
			Message:    log.Message,
			FaultCode:  log.FaultCode,
			Details:    log.Details,
			CreatedAt:  log.CreatedAt.Format(time.RFC3339),
		}
	}

	return result, total, nil
}

// GetDeviceStats retrieves device statistics.
func (uc *DeviceUseCase) GetDeviceStats(ctx context.Context, deviceID string) (*v1.DeviceStatsResponse, error) {
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

	// Calculate uptime
	var uptime int64
	if device.LastHeartbeat != nil {
		uptime = time.Since(*device.LastHeartbeat).Seconds()
	}

	// Get device logs count
	errorCount, err := uc.vehicleRepo.GetDeviceLogCount(ctx, deviceID, "error")
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get error count: %v", err)
		errorCount = 0
	}

	warningCount, err := uc.vehicleRepo.GetDeviceLogCount(ctx, deviceID, "warning")
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get warning count: %v", err)
		warningCount = 0
	}

	return &v1.DeviceStatsResponse{
		DeviceId:        device.DeviceID,
		FirmwareVersion: device.FirmwareVersion,
		HardwareVersion: device.HardwareVersion,
		Status:          device.Status,
		Uptime:          uptime,
		ErrorCount:      errorCount,
		WarningCount:    warningCount,
		Stats:           device.DeviceStats,
		LastHeartbeat:   func() string {
			if device.LastHeartbeat != nil {
				return device.LastHeartbeat.Format(time.RFC3339)
			}
			return ""
		}(),
		LastFaultTime:   func() string {
			if device.LastFaultTime != nil {
				return device.LastFaultTime.Format(time.RFC3339)
			}
			return ""
		}(),
		LastUpgradeTime: func() string {
			if device.LastUpgradeTime != nil {
				return device.LastUpgradeTime.Format(time.RFC3339)
			}
			return ""
		}(),
	}, nil
}

// UpdateDeviceConfig updates device configuration.
func (uc *DeviceUseCase) UpdateDeviceConfig(ctx context.Context, req *v1.UpdateDeviceConfigRequest) (*v1.UpdateDeviceConfigResponse, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Check if device exists
	device, err := uc.vehicleRepo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", req.DeviceId)
	}

	// Update device configuration
	if err := uc.vehicleRepo.UpdateDeviceConfig(ctx, req.DeviceId, req.Config); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update device config: %v", err)
		return nil, fmt.Errorf("failed to update device config: %w", err)
	}

	// Send config update command to device via MQTT
	configCmd := map[string]interface{}{
		"command":   "config",
		"config":    req.Config,
		"timestamp": time.Now().Unix(),
	}

	topic := fmt.Sprintf("devices/%s/commands", req.DeviceId)
	if err := uc.mqttClient.Publish(ctx, topic, configCmd); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to send config command: %v", err)
		// Config update is still successful in database, just log the error
	}

	uc.log.WithContext(ctx).Infof("Config updated for device %s", req.DeviceId)

	return &v1.UpdateDeviceConfigResponse{
		Status:  "success",
		Message: "Device config updated successfully",
	}, nil
}

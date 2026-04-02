// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/xuanyiying/smart-park/internal/vehicle/device"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// DeviceUseCase handles device management business logic.
type DeviceUseCase struct {
	vehicleRepo    VehicleRepo
	adapterFactory *device.AdapterFactory
	config         *Config
	log            *log.Helper
}

// NewDeviceUseCase creates a new DeviceUseCase.
func NewDeviceUseCase(vehicleRepo VehicleRepo, adapterFactory *device.AdapterFactory, logger log.Logger) *DeviceUseCase {
	return &DeviceUseCase{
		vehicleRepo:    vehicleRepo,
		adapterFactory: adapterFactory,
		config:         DefaultConfig(),
		log:            log.NewHelper(logger),
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

	var lastOnline string
	if d.LastOnline != nil {
		lastOnline = d.LastOnline.Format(time.RFC3339)
	}

	laneID := ""
	if d.LaneID != nil {
		laneID = d.LaneID.String()
	}
	lotID := ""
	if d.LotID != nil {
		lotID = d.LotID.String()
	}

	// Convert vendor specific config to proto map
	vendorConfig := make(map[string]string)
	if d.VendorSpecificConfig != nil {
		for k, v := range d.VendorSpecificConfig {
			vendorConfig[k] = fmt.Sprintf("%v", v)
		}
	}

	return &v1.DeviceInfo{
		DeviceId:            d.DeviceID,
		Status:              d.Status,
		LastHeartbeat:       lastHeartbeat,
		LastOnline:          lastOnline,
		Online:              online,
		LaneId:              laneID,
		LotId:               lotID,
		DeviceType:          d.DeviceType,
		Manufacturer:        d.Manufacturer,
		Model:               d.Model,
		FirmwareVersion:     d.FirmwareVersion,
		FaultInfo:           d.FaultInfo,
		HeartbeatCount:      int32(d.HeartbeatCount),
		OfflineCount:        int32(d.OfflineCount),
		VendorSpecificConfig: vendorConfig,
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

	// Parse vendor specific config if provided
	var vendorConfig map[string]interface{}
	if req.VendorSpecificConfig != nil {
		vendorConfig = make(map[string]interface{})
		for k, v := range req.VendorSpecificConfig {
			vendorConfig[k] = v
		}
	}

	device := &Device{
		DeviceID:            req.DeviceId,
		DeviceType:          req.DeviceType,
		Manufacturer:        req.Manufacturer,
		Model:               req.Model,
		FirmwareVersion:     req.FirmwareVersion,
		VendorSpecificConfig: vendorConfig,
		LotID:               lotID,
		LaneID:              laneID,
		Status:              req.Status,
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

	// Parse vendor specific config if provided
	var vendorConfig map[string]interface{}
	if req.VendorSpecificConfig != nil {
		vendorConfig = make(map[string]interface{})
		for k, v := range req.VendorSpecificConfig {
			vendorConfig[k] = v
		}
	}

	device := &Device{
		DeviceID:            req.DeviceId,
		DeviceType:          req.DeviceType,
		Manufacturer:        req.Manufacturer,
		Model:               req.Model,
		FirmwareVersion:     req.FirmwareVersion,
		VendorSpecificConfig: vendorConfig,
		LotID:               lotID,
		LaneID:              laneID,
		Status:              req.Status,
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

// ManufacturerUseCase handles manufacturer management business logic.
type ManufacturerUseCase struct {
	vehicleRepo VehicleRepo
	log         *log.Helper
}

// NewManufacturerUseCase creates a new ManufacturerUseCase.
func NewManufacturerUseCase(vehicleRepo VehicleRepo, logger log.Logger) *ManufacturerUseCase {
	return &ManufacturerUseCase{
		vehicleRepo: vehicleRepo,
		log:         log.NewHelper(logger),
	}
}

// CreateManufacturer creates a new manufacturer.
func (uc *ManufacturerUseCase) CreateManufacturer(ctx context.Context, req *v1.CreateManufacturerRequest) (*v1.Manufacturer, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("manufacturer name is required")
	}

	manufacturer := &Manufacturer{
		ID:          uuid.New(),
		Name:        req.Name,
		Website:     req.Website,
		ContactInfo: req.ContactInfo,
		Description: req.Description,
	}

	if err := uc.vehicleRepo.CreateManufacturer(ctx, manufacturer); err != nil {
		return nil, fmt.Errorf("failed to create manufacturer: %w", err)
	}

	// Get the created manufacturer
	created, err := uc.vehicleRepo.GetManufacturer(ctx, manufacturer.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created manufacturer: %w", err)
	}

	return uc.convertManufacturerToProto(created), nil
}

// GetManufacturer retrieves a manufacturer by ID.
func (uc *ManufacturerUseCase) GetManufacturer(ctx context.Context, id string) (*v1.Manufacturer, error) {
	if id == "" {
		return nil, fmt.Errorf("manufacturer id is required")
	}

	manufacturerID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid manufacturer id: %w", err)
	}

	manufacturer, err := uc.vehicleRepo.GetManufacturer(ctx, manufacturerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get manufacturer: %w", err)
	}
	if manufacturer == nil {
		return nil, fmt.Errorf("manufacturer not found: %s", id)
	}

	return uc.convertManufacturerToProto(manufacturer), nil
}

// ListManufacturers lists all manufacturers with pagination.
func (uc *ManufacturerUseCase) ListManufacturers(ctx context.Context, page, pageSize int) ([]*v1.Manufacturer, int, error) {
	manufacturers, total, err := uc.vehicleRepo.ListManufacturers(ctx, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*v1.Manufacturer, len(manufacturers))
	for i, m := range manufacturers {
		result[i] = uc.convertManufacturerToProto(m)
	}

	return result, total, nil
}

// UpdateManufacturer updates an existing manufacturer.
func (uc *ManufacturerUseCase) UpdateManufacturer(ctx context.Context, req *v1.UpdateManufacturerRequest) (*v1.Manufacturer, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("manufacturer id is required")
	}
	if req.Name == "" {
		return nil, fmt.Errorf("manufacturer name is required")
	}

	manufacturerID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid manufacturer id: %w", err)
	}

	// Check if manufacturer exists
	existing, err := uc.vehicleRepo.GetManufacturer(ctx, manufacturerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing manufacturer: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("manufacturer not found: %s", req.Id)
	}

	manufacturer := &Manufacturer{
		ID:          manufacturerID,
		Name:        req.Name,
		Website:     req.Website,
		ContactInfo: req.ContactInfo,
		Description: req.Description,
	}

	if err := uc.vehicleRepo.UpdateManufacturer(ctx, manufacturer); err != nil {
		return nil, fmt.Errorf("failed to update manufacturer: %w", err)
	}

	// Get the updated manufacturer
	updated, err := uc.vehicleRepo.GetManufacturer(ctx, manufacturerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated manufacturer: %w", err)
	}

	return uc.convertManufacturerToProto(updated), nil
}

// DeleteManufacturer deletes a manufacturer by ID.
func (uc *ManufacturerUseCase) DeleteManufacturer(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("manufacturer id is required")
	}

	manufacturerID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid manufacturer id: %w", err)
	}

	// Check if manufacturer exists
	existing, err := uc.vehicleRepo.GetManufacturer(ctx, manufacturerID)
	if err != nil {
		return fmt.Errorf("failed to check existing manufacturer: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("manufacturer not found: %s", id)
	}

	if err := uc.vehicleRepo.DeleteManufacturer(ctx, manufacturerID); err != nil {
		return fmt.Errorf("failed to delete manufacturer: %w", err)
	}

	return nil
}

// convertManufacturerToProto converts biz.Manufacturer to v1.Manufacturer.
func (uc *ManufacturerUseCase) convertManufacturerToProto(m *Manufacturer) *v1.Manufacturer {
	return &v1.Manufacturer{
		Id:          m.ID.String(),
		Name:        m.Name,
		Website:     m.Website,
		ContactInfo: m.ContactInfo,
		Description: m.Description,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339),
	}
}

// FirmwareUseCase handles firmware management business logic.
type FirmwareUseCase struct {
	vehicleRepo VehicleRepo
	log         *log.Helper
}

// NewFirmwareUseCase creates a new FirmwareUseCase.
func NewFirmwareUseCase(vehicleRepo VehicleRepo, logger log.Logger) *FirmwareUseCase {
	return &FirmwareUseCase{
		vehicleRepo: vehicleRepo,
		log:         log.NewHelper(logger),
	}
}

// CreateFirmware creates a new firmware.
func (uc *FirmwareUseCase) CreateFirmware(ctx context.Context, req *v1.CreateFirmwareRequest) (*v1.Firmware, error) {
	if req.FirmwareId == "" {
		return nil, fmt.Errorf("firmware id is required")
	}
	if req.Manufacturer == "" {
		return nil, fmt.Errorf("manufacturer is required")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("model is required")
	}
	if req.Version == "" {
		return nil, fmt.Errorf("version is required")
	}
	if req.Url == "" {
		return nil, fmt.Errorf("url is required")
	}

	firmware := &Firmware{
		ID:          uuid.New(),
		FirmwareID:  req.FirmwareId,
		Manufacturer: req.Manufacturer,
		Model:       req.Model,
		Version:     req.Version,
		URL:         req.Url,
		Size:        req.Size,
		MD5:         req.Md5,
		Description: req.Description,
		Status:      req.Status,
	}

	if firmware.Status == "" {
		firmware.Status = "draft"
	}

	if err := uc.vehicleRepo.CreateFirmware(ctx, firmware); err != nil {
		return nil, fmt.Errorf("failed to create firmware: %w", err)
	}

	// Get the created firmware
	created, err := uc.vehicleRepo.GetFirmware(ctx, firmware.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created firmware: %w", err)
	}

	return uc.convertFirmwareToProto(created), nil
}

// GetFirmware retrieves a firmware by ID.
func (uc *FirmwareUseCase) GetFirmware(ctx context.Context, id string) (*v1.Firmware, error) {
	if id == "" {
		return nil, fmt.Errorf("firmware id is required")
	}

	firmwareID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid firmware id: %w", err)
	}

	firmware, err := uc.vehicleRepo.GetFirmware(ctx, firmwareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware: %w", err)
	}
	if firmware == nil {
		return nil, fmt.Errorf("firmware not found: %s", id)
	}

	return uc.convertFirmwareToProto(firmware), nil
}

// GetFirmwareByID retrieves a firmware by firmware ID.
func (uc *FirmwareUseCase) GetFirmwareByID(ctx context.Context, firmwareID string) (*v1.Firmware, error) {
	if firmwareID == "" {
		return nil, fmt.Errorf("firmware id is required")
	}

	firmware, err := uc.vehicleRepo.GetFirmwareByID(ctx, firmwareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware: %w", err)
	}
	if firmware == nil {
		return nil, fmt.Errorf("firmware not found: %s", firmwareID)
	}

	return uc.convertFirmwareToProto(firmware), nil
}

// ListFirmwares lists all firmwares with pagination.
func (uc *FirmwareUseCase) ListFirmwares(ctx context.Context, manufacturer, model string, page, pageSize int) ([]*v1.Firmware, int, error) {
	firmwares, total, err := uc.vehicleRepo.ListFirmwares(ctx, manufacturer, model, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*v1.Firmware, len(firmwares))
	for i, f := range firmwares {
		result[i] = uc.convertFirmwareToProto(f)
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

// UpdateFirmware updates an existing firmware.
func (uc *FirmwareUseCase) UpdateFirmware(ctx context.Context, req *v1.UpdateFirmwareRequest) (*v1.Firmware, error) {
	if req.Id == "" {
		return nil, fmt.Errorf("firmware id is required")
	}

	firmwareID, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, fmt.Errorf("invalid firmware id: %w", err)
	}

	// Check if firmware exists
	existing, err := uc.vehicleRepo.GetFirmware(ctx, firmwareID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing firmware: %w", err)
	}
	if existing == nil {
		return nil, fmt.Errorf("firmware not found: %s", req.Id)
	}

	firmware := &Firmware{
		ID:          firmwareID,
		FirmwareID:  req.FirmwareId,
		Manufacturer: req.Manufacturer,
		Model:       req.Model,
		Version:     req.Version,
		URL:         req.Url,
		Size:        req.Size,
		MD5:         req.Md5,
		Description: req.Description,
		Status:      req.Status,
	}

	if err := uc.vehicleRepo.UpdateFirmware(ctx, firmware); err != nil {
		return nil, fmt.Errorf("failed to update firmware: %w", err)
	}

	// Get the updated firmware
	updated, err := uc.vehicleRepo.GetFirmware(ctx, firmwareID)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated firmware: %w", err)
	}

	return uc.convertFirmwareToProto(updated), nil
}

// DeleteFirmware deletes a firmware by ID.
func (uc *FirmwareUseCase) DeleteFirmware(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("firmware id is required")
	}

	firmwareID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid firmware id: %w", err)
	}

	// Check if firmware exists
	existing, err := uc.vehicleRepo.GetFirmware(ctx, firmwareID)
	if err != nil {
		return fmt.Errorf("failed to check existing firmware: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("firmware not found: %s", id)
	}

	if err := uc.vehicleRepo.DeleteFirmware(ctx, firmwareID); err != nil {
		return fmt.Errorf("failed to delete firmware: %w", err)
	}

	return nil
}

// GetLatestFirmware retrieves the latest firmware for a specific manufacturer and model.
func (uc *FirmwareUseCase) GetLatestFirmware(ctx context.Context, manufacturer, model string) (*v1.Firmware, error) {
	if manufacturer == "" {
		return nil, fmt.Errorf("manufacturer is required")
	}
	if model == "" {
		return nil, fmt.Errorf("model is required")
	}

	firmware, err := uc.vehicleRepo.GetLatestFirmware(ctx, manufacturer, model)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest firmware: %w", err)
	}
	if firmware == nil {
		return nil, fmt.Errorf("no firmware found for manufacturer: %s and model: %s", manufacturer, model)
	}

	return uc.convertFirmwareToProto(firmware), nil
}

// convertFirmwareToProto converts biz.Firmware to v1.Firmware.
func (uc *FirmwareUseCase) convertFirmwareToProto(f *Firmware) *v1.Firmware {
	return &v1.Firmware{
		Id:          f.ID.String(),
		FirmwareId:  f.FirmwareID,
		Manufacturer: f.Manufacturer,
		Model:       f.Model,
		Version:     f.Version,
		Url:         f.URL,
		Size:        f.Size,
		Md5:         f.MD5,
		Description: f.Description,
		Status:      f.Status,
		ReleaseDate: f.ReleaseDate.Format(time.RFC3339),
		CreatedAt:   f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   f.UpdatedAt.Format(time.RFC3339),
	}
}

// DevicePerformanceUseCase handles device performance monitoring business logic.
type DevicePerformanceUseCase struct {
	vehicleRepo VehicleRepo
	log         *log.Helper
}

// NewDevicePerformanceUseCase creates a new DevicePerformanceUseCase.
func NewDevicePerformanceUseCase(vehicleRepo VehicleRepo, logger log.Logger) *DevicePerformanceUseCase {
	return &DevicePerformanceUseCase{
		vehicleRepo: vehicleRepo,
		log:         log.NewHelper(logger),
	}
}

// CreateDevicePerformance creates a new device performance record.
func (uc *DevicePerformanceUseCase) CreateDevicePerformance(ctx context.Context, req *v1.CreateDevicePerformanceRequest) error {
	if req.DeviceId == "" {
		return fmt.Errorf("device id is required")
	}

	timestamp, err := time.Parse(time.RFC3339, req.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	performance := &DevicePerformance{
		ID:           uuid.New(),
		DeviceID:     req.DeviceId,
		CPUUsage:     req.CpuUsage,
		MemoryUsage:  req.MemoryUsage,
		StorageUsage: req.StorageUsage,
		NetworkIn:    req.NetworkIn,
		NetworkOut:   req.NetworkOut,
		Temperature:  req.Temperature,
		Timestamp:    timestamp,
		CreatedAt:    time.Now(),
	}

	return uc.vehicleRepo.CreateDevicePerformance(ctx, performance)
}

// GetDevicePerformance retrieves device performance records within a time range.
func (uc *DevicePerformanceUseCase) GetDevicePerformance(ctx context.Context, req *v1.GetDevicePerformanceRequest) ([]*v1.DevicePerformance, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		startTime = time.Now().Add(-24 * time.Hour)
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		endTime = time.Now()
	}

	records, err := uc.vehicleRepo.GetDevicePerformance(ctx, req.DeviceId, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get device performance: %w", err)
	}

	result := make([]*v1.DevicePerformance, len(records))
	for i, record := range records {
		result[i] = uc.convertDevicePerformanceToProto(record)
	}

	return result, nil
}

// GetDevicePerformanceLatest retrieves the latest device performance record.
func (uc *DevicePerformanceUseCase) GetDevicePerformanceLatest(ctx context.Context, deviceID string) (*v1.DevicePerformance, error) {
// GetDeviceStats retrieves device statistics.
func (uc *DeviceUseCase) GetDeviceStats(ctx context.Context, deviceID string) (*v1.DeviceStatsResponse, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	record, err := uc.vehicleRepo.GetDevicePerformanceLatest(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest device performance: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("no performance data found for device: %s", deviceID)
	}

	return uc.convertDevicePerformanceToProto(record), nil
}

// convertDevicePerformanceToProto converts biz.DevicePerformance to v1.DevicePerformance.
func (uc *DevicePerformanceUseCase) convertDevicePerformanceToProto(p *DevicePerformance) *v1.DevicePerformance {
	return &v1.DevicePerformance{
		Id:           p.ID.String(),
		DeviceId:     p.DeviceID,
		CpuUsage:     p.CPUUsage,
		MemoryUsage:  p.MemoryUsage,
		StorageUsage: p.StorageUsage,
		NetworkIn:    p.NetworkIn,
		NetworkOut:   p.NetworkOut,
		Temperature:  p.Temperature,
		Timestamp:    p.Timestamp.Format(time.RFC3339),
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
	}
}

// DeviceFaultUseCase handles device fault diagnosis business logic.
type DeviceFaultUseCase struct {
	vehicleRepo VehicleRepo
	log         *log.Helper
}

// NewDeviceFaultUseCase creates a new DeviceFaultUseCase.
func NewDeviceFaultUseCase(vehicleRepo VehicleRepo, logger log.Logger) *DeviceFaultUseCase {
	return &DeviceFaultUseCase{
		vehicleRepo: vehicleRepo,
		log:         log.NewHelper(logger),
	}
}

// CreateDeviceFault creates a new device fault record.
func (uc *DeviceFaultUseCase) CreateDeviceFault(ctx context.Context, req *v1.CreateDeviceFaultRequest) (*v1.DeviceFault, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}
	if req.FaultType == "" {
		return nil, fmt.Errorf("fault type is required")
	}
	if req.FaultCode == "" {
		return nil, fmt.Errorf("fault code is required")
	}

	detectedAt, err := time.Parse(time.RFC3339, req.DetectedAt)
	if err != nil {
		detectedAt = time.Now()
	}

	fault := &DeviceFault{
		ID:          uuid.New(),
		DeviceID:    req.DeviceId,
		FaultType:   req.FaultType,
		FaultCode:   req.FaultCode,
		Description: req.Description,
		Severity:    req.Severity,
		Status:      req.Status,
		Suggestion:  req.Suggestion,
		DetectedAt:  detectedAt,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if fault.Severity == "" {
		fault.Severity = "error"
	}
	if fault.Status == "" {
		fault.Status = "detected"
	}

	if err := uc.vehicleRepo.CreateDeviceFault(ctx, fault); err != nil {
		return nil, fmt.Errorf("failed to create device fault: %w", err)
	}

	// Get the created fault
	created, err := uc.vehicleRepo.GetDeviceFault(ctx, fault.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get created fault: %w", err)
	}

	return uc.convertDeviceFaultToProto(created), nil
}

// GetDeviceFault retrieves a device fault by ID.
func (uc *DeviceFaultUseCase) GetDeviceFault(ctx context.Context, id string) (*v1.DeviceFault, error) {
	if id == "" {
		return nil, fmt.Errorf("fault id is required")
	}

	faultID, err := uuid.Parse(id)
	if err != nil {
		return nil, fmt.Errorf("invalid fault id: %w", err)
	}

	fault, err := uc.vehicleRepo.GetDeviceFault(ctx, faultID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device fault: %w", err)
	}
	if fault == nil {
		return nil, fmt.Errorf("fault not found: %s", id)
	}

	return uc.convertDeviceFaultToProto(fault), nil
}

// ListDeviceFaults lists device faults with pagination.
func (uc *DeviceFaultUseCase) ListDeviceFaults(ctx context.Context, deviceID, status string, page, pageSize int) ([]*v1.DeviceFault, int, error) {
	faults, total, err := uc.vehicleRepo.ListDeviceFaults(ctx, deviceID, status, page, pageSize)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*v1.DeviceFault, len(faults))
	for i, fault := range faults {
		result[i] = uc.convertDeviceFaultToProto(fault)
	}

	return result, total, nil
}

// ResolveDeviceFault resolves a device fault.
func (uc *DeviceFaultUseCase) ResolveDeviceFault(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("fault id is required")
	}

	faultID, err := uuid.Parse(id)
	if err != nil {
		return fmt.Errorf("invalid fault id: %w", err)
	}

	return uc.vehicleRepo.ResolveDeviceFault(ctx, faultID)
}

// convertDeviceFaultToProto converts biz.DeviceFault to v1.DeviceFault.
func (uc *DeviceFaultUseCase) convertDeviceFaultToProto(f *DeviceFault) *v1.DeviceFault {
	var resolvedAt string
	if f.ResolvedAt != nil {
		resolvedAt = f.ResolvedAt.Format(time.RFC3339)
	}

	return &v1.DeviceFault{
		Id:          f.ID.String(),
		DeviceId:    f.DeviceID,
		FaultType:   f.FaultType,
		FaultCode:   f.FaultCode,
		Description: f.Description,
		Severity:    f.Severity,
		Status:      f.Status,
		Suggestion:  f.Suggestion,
		DetectedAt:  f.DetectedAt.Format(time.RFC3339),
		ResolvedAt:  resolvedAt,
		CreatedAt:   f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   f.UpdatedAt.Format(time.RFC3339),
	}
}

// DeviceStatsUseCase handles device statistics analysis business logic.
type DeviceStatsUseCase struct {
	vehicleRepo VehicleRepo
	log         *log.Helper
}

// NewDeviceStatsUseCase creates a new DeviceStatsUseCase.
func NewDeviceStatsUseCase(vehicleRepo VehicleRepo, logger log.Logger) *DeviceStatsUseCase {
	return &DeviceStatsUseCase{
		vehicleRepo: vehicleRepo,
		log:         log.NewHelper(logger),
	}
}

// GetDeviceUsageStats retrieves device usage statistics.
func (uc *DeviceStatsUseCase) GetDeviceUsageStats(ctx context.Context, req *v1.GetDeviceUsageStatsRequest) (map[string]interface{}, error) {
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

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		endTime = time.Now()
	}

	stats, err := uc.vehicleRepo.GetDeviceUsageStats(ctx, req.DeviceId, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get device usage stats: %w", err)
	}

	return stats, nil
}

// GetDeviceFaultStats retrieves device fault statistics.
func (uc *DeviceStatsUseCase) GetDeviceFaultStats(ctx context.Context, req *v1.GetDeviceFaultStatsRequest) (map[string]interface{}, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	startTime, err := time.Parse(time.RFC3339, req.StartTime)
	if err != nil {
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	}

	endTime, err := time.Parse(time.RFC3339, req.EndTime)
	if err != nil {
		endTime = time.Now()
	}

	stats, err := uc.vehicleRepo.GetDeviceFaultStats(ctx, req.DeviceId, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get device fault stats: %w", err)
	}

	return stats, nil
}

// GetDeviceStatsSummary retrieves device statistics summary.
func (uc *DeviceStatsUseCase) GetDeviceStatsSummary(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	stats, err := uc.vehicleRepo.GetDeviceStatsSummary(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device stats summary: %w", err)
	}

	return stats, nil
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

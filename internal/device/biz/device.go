// Package biz provides business logic for device management
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	v1 "github.com/xuanyiying/smart-park/api/device/v1"
)

// Device represents a device entity
type Device struct {
	DeviceID          string
	DeviceType        string
	Status            string
	Online            bool
	LastHeartbeat     *time.Time
	LotID             *uuid.UUID
	LaneID            *uuid.UUID
	Protocol          string
	Config            map[string]string
	FirmwareVersion   string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// FirmwareVersion represents a firmware version entity
type FirmwareVersion struct {
	ID          string
	DeviceType  string
	Version     string
	URL         string
	Checksum    string
	Description string
	IsActive    bool
	CreatedAt   time.Time
}

// FirmwareUpdate represents a firmware update task
type FirmwareUpdate struct {
	UpdateID         string
	DeviceID         string
	FirmwareVersion  string
	Status           string
	Progress         string
	ErrorMessage     string
	StartedAt        time.Time
	CompletedAt      *time.Time
}

// DeviceAlert represents a device alert entity
type DeviceAlert struct {
	AlertID          string
	DeviceID         string
	Type             string
	Severity         string
	Message          string
	Status           string
	CreatedAt        time.Time
	AcknowledgedAt   *time.Time
	AcknowledgedBy   string
	Notes            string
	Metadata         map[string]string
}

// Command represents a device command
type Command struct {
	CommandID    string
	DeviceID     string
	Command      string
	Params       map[string]string
	Status       string
	CreatedAt    time.Time
	ExecutedAt   *time.Time
	Result       string
}

// DeviceRepo defines the device repository interface
type DeviceRepo interface {
	// Device management
	CreateDevice(ctx context.Context, device *Device) error
	GetDeviceByID(ctx context.Context, deviceID string) (*Device, error)
	GetDeviceByCode(ctx context.Context, deviceCode string) (*Device, error)
	UpdateDevice(ctx context.Context, device *Device) error
	DeleteDevice(ctx context.Context, deviceID string) error
	ListDevices(ctx context.Context, page, pageSize int, deviceType, status string) ([]*Device, int, error)
	UpdateDeviceHeartbeat(ctx context.Context, deviceID string) error
	UpdateDeviceFirmwareVersion(ctx context.Context, deviceID, firmwareVersion string) error

	// Firmware management
	CreateFirmwareVersion(ctx context.Context, firmware *FirmwareVersion) error
	GetFirmwareVersionByID(ctx context.Context, id string) (*FirmwareVersion, error)
	GetFirmwareVersionByVersion(ctx context.Context, deviceType, version string) (*FirmwareVersion, error)
	ListFirmwareVersions(ctx context.Context, page, pageSize int, deviceType string) ([]*FirmwareVersion, int, error)
	UpdateFirmwareVersionStatus(ctx context.Context, id string, isActive bool) error

	// Firmware update management
	CreateFirmwareUpdate(ctx context.Context, update *FirmwareUpdate) error
	GetFirmwareUpdate(ctx context.Context, updateID string) (*FirmwareUpdate, error)
	GetFirmwareUpdateByDevice(ctx context.Context, deviceID string) (*FirmwareUpdate, error)
	UpdateFirmwareUpdateStatus(ctx context.Context, updateID, status, progress, errorMessage string) error

	// Alert management
	CreateAlert(ctx context.Context, alert *DeviceAlert) error
	GetAlertByID(ctx context.Context, alertID string) (*DeviceAlert, error)
	ListAlerts(ctx context.Context, page, pageSize int, deviceID, severity, status string) ([]*DeviceAlert, int, error)
	UpdateAlertStatus(ctx context.Context, alertID, status, acknowledgedBy, notes string) error

	// Command management
	CreateCommand(ctx context.Context, command *Command) error
	GetCommandByID(ctx context.Context, commandID string) (*Command, error)
	UpdateCommandStatus(ctx context.Context, commandID, status, result string) error
}

// DeviceUseCase handles device management business logic
type DeviceUseCase struct {
	repo  DeviceRepo
	log   *log.Helper
}

// NewDeviceUseCase creates a new DeviceUseCase
func NewDeviceUseCase(repo DeviceRepo, logger log.Logger) *DeviceUseCase {
	return &DeviceUseCase{
		repo:  repo,
		log:   log.NewHelper(logger),
	}
}

// ListDevices lists all devices with pagination
func (uc *DeviceUseCase) ListDevices(ctx context.Context, req *v1.ListDevicesRequest) ([]*v1.DeviceInfo, int, error) {
	devices, total, err := uc.repo.ListDevices(ctx, int(req.Page), int(req.PageSize), req.DeviceType, req.Status)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*v1.DeviceInfo, len(devices))
	for i, d := range devices {
		result[i] = uc.convertDeviceToProto(d)
	}

	return result, total, nil
}

// GetDevice retrieves a device by ID
func (uc *DeviceUseCase) GetDevice(ctx context.Context, deviceID string) (*v1.DeviceInfo, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	device, err := uc.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	return uc.convertDeviceToProto(device), nil
}

// CreateDevice creates a new device
func (uc *DeviceUseCase) CreateDevice(ctx context.Context, req *v1.CreateDeviceRequest) (*v1.DeviceInfo, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Check if device already exists
	existing, err := uc.repo.GetDeviceByID(ctx, req.DeviceId)
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
		DeviceID:        req.DeviceId,
		DeviceType:      req.DeviceType,
		Status:          req.Status,
		Protocol:        req.Protocol,
		Config:          req.Config,
		FirmwareVersion: "1.0.0", // Default version
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if device.Status == "" {
		device.Status = "active"
	}

	if err := uc.repo.CreateDevice(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to create device: %w", err)
	}

	// Get the created device
	created, err := uc.repo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get created device: %w", err)
	}

	return uc.convertDeviceToProto(created), nil
}

// UpdateDevice updates an existing device
func (uc *DeviceUseCase) UpdateDevice(ctx context.Context, req *v1.UpdateDeviceRequest) (*v1.DeviceInfo, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Check if device exists
	existing, err := uc.repo.GetDeviceByID(ctx, req.DeviceId)
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
		DeviceID:        req.DeviceId,
		DeviceType:      req.DeviceType,
		Status:          req.Status,
		LotID:           lotID,
		LaneID:          laneID,
		Config:          req.Config,
		UpdatedAt:       time.Now(),
	}

	if err := uc.repo.UpdateDevice(ctx, device); err != nil {
		return nil, fmt.Errorf("failed to update device: %w", err)
	}

	// Get the updated device
	updated, err := uc.repo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated device: %w", err)
	}

	return uc.convertDeviceToProto(updated), nil
}

// DeleteDevice deletes a device by ID
func (uc *DeviceUseCase) DeleteDevice(ctx context.Context, deviceID string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}

	// Check if device exists
	existing, err := uc.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to check existing device: %w", err)
	}
	if existing == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	if err := uc.repo.DeleteDevice(ctx, deviceID); err != nil {
		return fmt.Errorf("failed to delete device: %w", err)
	}

	return nil
}

// GetDeviceStatus retrieves device status
func (uc *DeviceUseCase) GetDeviceStatus(ctx context.Context, deviceID string) (*v1.DeviceStatus, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	device, err := uc.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	online := false
	if device.LastHeartbeat != nil {
		online = time.Since(*device.LastHeartbeat) < 5*time.Minute // 5 minutes threshold
	}

	var lastHeartbeat string
	if device.LastHeartbeat != nil {
		lastHeartbeat = device.LastHeartbeat.Format(time.RFC3339)
	}

	return &v1.DeviceStatus{
		DeviceId:          device.DeviceID,
		Online:            online,
		Status:            device.Status,
		LastHeartbeat:     lastHeartbeat,
		FirmwareVersion:   device.FirmwareVersion,
		Metrics:           map[string]string{}, // TODO: Add actual metrics
	}, nil
}

// Heartbeat handles device heartbeat
func (uc *DeviceUseCase) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) (*v1.HeartbeatResponse, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Update device heartbeat
	if err := uc.repo.UpdateDeviceHeartbeat(ctx, req.DeviceId); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update heartbeat: %v", err)
		return nil, fmt.Errorf("failed to update device heartbeat: %w", err)
	}

	// Check if firmware update is needed
	device, err := uc.repo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get device: %v", err)
		return &v1.HeartbeatResponse{
			Code:    200,
			Message: "Heartbeat received",
		}, nil
	}

	// TODO: Check for available firmware updates
	needsUpdate := false
	newFirmwareVersion := ""

	return &v1.HeartbeatResponse{
		Code:                200,
		Message:             "Heartbeat received",
		NeedsUpdate:         needsUpdate,
		NewFirmwareVersion:  newFirmwareVersion,
	}, nil
}

// ListFirmwareVersions lists all firmware versions
func (uc *DeviceUseCase) ListFirmwareVersions(ctx context.Context, req *v1.ListFirmwareVersionsRequest) ([]*v1.FirmwareVersion, int, error) {
	firmwares, total, err := uc.repo.ListFirmwareVersions(ctx, int(req.Page), int(req.PageSize), req.DeviceType)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*v1.FirmwareVersion, len(firmwares))
	for i, f := range firmwares {
		result[i] = &v1.FirmwareVersion{
			Id:          f.ID,
			DeviceType:  f.DeviceType,
			Version:     f.Version,
			Url:         f.URL,
			Checksum:    f.Checksum,
			Description: f.Description,
			IsActive:    f.IsActive,
			CreatedAt:   f.CreatedAt.Format(time.RFC3339),
		}
	}

	return result, total, nil
}

// CreateFirmwareVersion creates a new firmware version
func (uc *DeviceUseCase) CreateFirmwareVersion(ctx context.Context, req *v1.CreateFirmwareVersionRequest) (*v1.FirmwareVersion, error) {
	firmware := &FirmwareVersion{
		ID:          uuid.New().String(),
		DeviceType:  req.DeviceType,
		Version:     req.Version,
		URL:         req.Url,
		Checksum:    req.Checksum,
		Description: req.Description,
		IsActive:    req.IsActive,
		CreatedAt:   time.Now(),
	}

	if err := uc.repo.CreateFirmwareVersion(ctx, firmware); err != nil {
		return nil, fmt.Errorf("failed to create firmware version: %w", err)
	}

	return &v1.FirmwareVersion{
		Id:          firmware.ID,
		DeviceType:  firmware.DeviceType,
		Version:     firmware.Version,
		Url:         firmware.URL,
		Checksum:    firmware.Checksum,
		Description: firmware.Description,
		IsActive:    firmware.IsActive,
		CreatedAt:   firmware.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateDeviceFirmware updates device firmware
func (uc *DeviceUseCase) UpdateDeviceFirmware(ctx context.Context, req *v1.UpdateDeviceFirmwareRequest) (*v1.FirmwareUpdateInfo, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Check if device exists
	device, err := uc.repo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", req.DeviceId)
	}

	// Check if firmware version exists
	firmware, err := uc.repo.GetFirmwareVersionByVersion(ctx, device.DeviceType, req.FirmwareVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get firmware version: %w", err)
	}
	if firmware == nil {
		return nil, fmt.Errorf("firmware version not found: %s", req.FirmwareVersion)
	}

	// Create firmware update task
	update := &FirmwareUpdate{
		UpdateID:         uuid.New().String(),
		DeviceID:         req.DeviceId,
		FirmwareVersion:  req.FirmwareVersion,
		Status:           "pending",
		Progress:         "0%",
		StartedAt:        time.Now(),
	}

	if err := uc.repo.CreateFirmwareUpdate(ctx, update); err != nil {
		return nil, fmt.Errorf("failed to create firmware update: %w", err)
	}

	// TODO: Send update command to device

	return &v1.FirmwareUpdateInfo{
		UpdateId:        update.UpdateID,
		DeviceId:        update.DeviceID,
		FirmwareVersion: update.FirmwareVersion,
		Status:          update.Status,
		Progress:        update.Progress,
		ErrorMessage:    update.ErrorMessage,
		StartedAt:       update.StartedAt.Format(time.RFC3339),
	}, nil
}

// GetFirmwareUpdateStatus gets firmware update status
func (uc *DeviceUseCase) GetFirmwareUpdateStatus(ctx context.Context, req *v1.GetFirmwareUpdateStatusRequest) (*v1.FirmwareUpdateInfo, error) {
	var update *FirmwareUpdate
	var err error

	if req.UpdateId != "" {
		update, err = uc.repo.GetFirmwareUpdate(ctx, req.UpdateId)
	} else {
		update, err = uc.repo.GetFirmwareUpdateByDevice(ctx, req.DeviceId)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get firmware update: %w", err)
	}
	if update == nil {
		return nil, fmt.Errorf("firmware update not found")
	}

	var completedAt string
	if update.CompletedAt != nil {
		completedAt = update.CompletedAt.Format(time.RFC3339)
	}

	return &v1.FirmwareUpdateInfo{
		UpdateId:        update.UpdateID,
		DeviceId:        update.DeviceID,
		FirmwareVersion: update.FirmwareVersion,
		Status:          update.Status,
		Progress:        update.Progress,
		ErrorMessage:    update.ErrorMessage,
		StartedAt:       update.StartedAt.Format(time.RFC3339),
		CompletedAt:     completedAt,
	}, nil
}

// ListDeviceAlerts lists all device alerts
func (uc *DeviceUseCase) ListDeviceAlerts(ctx context.Context, req *v1.ListDeviceAlertsRequest) ([]*v1.DeviceAlert, int, error) {
	alerts, total, err := uc.repo.ListAlerts(ctx, int(req.Page), int(req.PageSize), req.DeviceId, req.Severity, req.Status)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*v1.DeviceAlert, len(alerts))
	for i, a := range alerts {
		var acknowledgedAt string
		if a.AcknowledgedAt != nil {
			acknowledgedAt = a.AcknowledgedAt.Format(time.RFC3339)
		}

		result[i] = &v1.DeviceAlert{
			AlertId:         a.AlertID,
			DeviceId:        a.DeviceID,
			Type:            a.Type,
			Severity:        a.Severity,
			Message:         a.Message,
			Status:          a.Status,
			CreatedAt:       a.CreatedAt.Format(time.RFC3339),
			AcknowledgedAt:  acknowledgedAt,
			AcknowledgedBy:  a.AcknowledgedBy,
			Notes:           a.Notes,
			Metadata:        a.Metadata,
		}
	}

	return result, total, nil
}

// GetDeviceAlert gets a device alert
func (uc *DeviceUseCase) GetDeviceAlert(ctx context.Context, alertID string) (*v1.DeviceAlert, error) {
	alert, err := uc.repo.GetAlertByID(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}
	if alert == nil {
		return nil, fmt.Errorf("alert not found: %s", alertID)
	}

	var acknowledgedAt string
	if alert.AcknowledgedAt != nil {
		acknowledgedAt = alert.AcknowledgedAt.Format(time.RFC3339)
	}

	return &v1.DeviceAlert{
		AlertId:         alert.AlertID,
		DeviceId:        alert.DeviceID,
		Type:            alert.Type,
		Severity:        alert.Severity,
		Message:         alert.Message,
		Status:          alert.Status,
		CreatedAt:       alert.CreatedAt.Format(time.RFC3339),
		AcknowledgedAt:  acknowledgedAt,
		AcknowledgedBy:  alert.AcknowledgedBy,
		Notes:           alert.Notes,
		Metadata:        alert.Metadata,
	}, nil
}

// AcknowledgeAlert acknowledges a device alert
func (uc *DeviceUseCase) AcknowledgeAlert(ctx context.Context, req *v1.AcknowledgeAlertRequest) error {
	if req.AlertId == "" {
		return fmt.Errorf("alert id is required")
	}

	// Check if alert exists
	alert, err := uc.repo.GetAlertByID(ctx, req.AlertId)
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}
	if alert == nil {
		return fmt.Errorf("alert not found: %s", req.AlertId)
	}

	// Update alert status
	if err := uc.repo.UpdateAlertStatus(ctx, req.AlertId, "acknowledged", "system", req.Notes); err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	return nil
}

// SendCommand sends a command to a device
func (uc *DeviceUseCase) SendCommand(ctx context.Context, req *v1.SendCommandRequest) (*v1.CommandInfo, error) {
	if req.DeviceId == "" {
		return nil, fmt.Errorf("device id is required")
	}
	if req.Command == "" {
		return nil, fmt.Errorf("command is required")
	}

	// Check if device exists
	device, err := uc.repo.GetDeviceByID(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", req.DeviceId)
	}

	// Create command
	command := &Command{
		CommandID:   uuid.New().String(),
		DeviceID:    req.DeviceId,
		Command:     req.Command,
		Params:      req.Params,
		Status:      "pending",
		CreatedAt:   time.Now(),
	}

	if err := uc.repo.CreateCommand(ctx, command); err != nil {
		return nil, fmt.Errorf("failed to create command: %w", err)
	}

	// TODO: Send command to device

	return &v1.CommandInfo{
		CommandId:   command.CommandID,
		Status:      command.Status,
		DeviceId:    command.DeviceID,
		Command:     command.Command,
		CreatedAt:   command.CreatedAt.Format(time.RFC3339),
	}, nil
}

// convertDeviceToProto converts biz.Device to v1.DeviceInfo
func (uc *DeviceUseCase) convertDeviceToProto(d *Device) *v1.DeviceInfo {
	online := false
	if d.LastHeartbeat != nil {
		online = time.Since(*d.LastHeartbeat) < 5*time.Minute // 5 minutes threshold
	}

	var lastHeartbeat string
	if d.LastHeartbeat != nil {
		lastHeartbeat = d.LastHeartbeat.Format(time.RFC3339)
	}

	lotID := ""
	if d.LotID != nil {
		lotID = d.LotID.String()
	}
	laneID := ""
	if d.LaneID != nil {
		laneID = d.LaneID.String()
	}

	return &v1.DeviceInfo{
		DeviceId:          d.DeviceID,
		DeviceType:        d.DeviceType,
		Status:            d.Status,
		Online:            online,
		LastHeartbeat:     lastHeartbeat,
		LotId:             lotID,
		LaneId:            laneID,
		Protocol:          d.Protocol,
		Config:            d.Config,
		FirmwareVersion:   d.FirmwareVersion,
		CreatedAt:         d.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         d.UpdatedAt.Format(time.RFC3339),
	}
}

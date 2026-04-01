// Package biz provides business logic for device management
package biz

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/xuanyiying/smart-park/api/device/v1"
	"github.com/stretchr/testify/assert"
)

// MockDeviceRepo is a mock implementation of DeviceRepo
type MockDeviceRepo struct {
	devices          map[string]*Device
	firmwareVersions map[string]*FirmwareVersion
	firmwareUpdates  map[string]*FirmwareUpdate
	alerts           map[string]*DeviceAlert
	commands         map[string]*Command
}

// NewMockDeviceRepo creates a new MockDeviceRepo
func NewMockDeviceRepo() *MockDeviceRepo {
	return &MockDeviceRepo{
		devices:          make(map[string]*Device),
		firmwareVersions: make(map[string]*FirmwareVersion),
		firmwareUpdates:  make(map[string]*FirmwareUpdate),
		alerts:           make(map[string]*DeviceAlert),
		commands:         make(map[string]*Command),
	}
}

// CreateDevice creates a new device
func (r *MockDeviceRepo) CreateDevice(ctx context.Context, device *Device) error {
	r.devices[device.DeviceID] = device
	return nil
}

// GetDeviceByID retrieves a device by ID
func (r *MockDeviceRepo) GetDeviceByID(ctx context.Context, deviceID string) (*Device, error) {
	device, ok := r.devices[deviceID]
	if !ok {
		return nil, nil
	}
	return device, nil
}

// GetDeviceByCode retrieves a device by code
func (r *MockDeviceRepo) GetDeviceByCode(ctx context.Context, deviceCode string) (*Device, error) {
	device, ok := r.devices[deviceCode]
	if !ok {
		return nil, nil
	}
	return device, nil
}

// UpdateDevice updates an existing device
func (r *MockDeviceRepo) UpdateDevice(ctx context.Context, device *Device) error {
	r.devices[device.DeviceID] = device
	return nil
}

// DeleteDevice deletes a device by ID
func (r *MockDeviceRepo) DeleteDevice(ctx context.Context, deviceID string) error {
	delete(r.devices, deviceID)
	return nil
}

// ListDevices lists all devices with pagination
func (r *MockDeviceRepo) ListDevices(ctx context.Context, page, pageSize int, deviceType, status string) ([]*Device, int, error) {
	var devices []*Device
	for _, device := range r.devices {
		if deviceType != "" && device.DeviceType != deviceType {
			continue
		}
		if status != "" && device.Status != status {
			continue
		}
		devices = append(devices, device)
	}
	return devices, len(devices), nil
}

// UpdateDeviceHeartbeat updates device heartbeat
func (r *MockDeviceRepo) UpdateDeviceHeartbeat(ctx context.Context, deviceID string) error {
	device, ok := r.devices[deviceID]
	if !ok {
		return nil
	}
	now := time.Now()
	device.LastHeartbeat = &now
	device.Online = true
	device.UpdatedAt = now
	return nil
}

// UpdateDeviceFirmwareVersion updates device firmware version
func (r *MockDeviceRepo) UpdateDeviceFirmwareVersion(ctx context.Context, deviceID, firmwareVersion string) error {
	device, ok := r.devices[deviceID]
	if !ok {
		return nil
	}
	device.FirmwareVersion = firmwareVersion
	device.UpdatedAt = time.Now()
	return nil
}

// CreateFirmwareVersion creates a new firmware version
func (r *MockDeviceRepo) CreateFirmwareVersion(ctx context.Context, firmware *FirmwareVersion) error {
	r.firmwareVersions[firmware.ID] = firmware
	return nil
}

// GetFirmwareVersionByID retrieves a firmware version by ID
func (r *MockDeviceRepo) GetFirmwareVersionByID(ctx context.Context, id string) (*FirmwareVersion, error) {
	firmware, ok := r.firmwareVersions[id]
	if !ok {
		return nil, nil
	}
	return firmware, nil
}

// GetFirmwareVersionByVersion retrieves a firmware version by version
func (r *MockDeviceRepo) GetFirmwareVersionByVersion(ctx context.Context, deviceType, version string) (*FirmwareVersion, error) {
	for _, firmware := range r.firmwareVersions {
		if firmware.DeviceType == deviceType && firmware.Version == version {
			return firmware, nil
		}
	}
	return nil, nil
}

// ListFirmwareVersions lists all firmware versions
func (r *MockDeviceRepo) ListFirmwareVersions(ctx context.Context, page, pageSize int, deviceType string) ([]*FirmwareVersion, int, error) {
	var firmwares []*FirmwareVersion
	for _, firmware := range r.firmwareVersions {
		if deviceType != "" && firmware.DeviceType != deviceType {
			continue
		}
		firmwares = append(firmwares, firmware)
	}
	return firmwares, len(firmwares), nil
}

// UpdateFirmwareVersionStatus updates firmware version status
func (r *MockDeviceRepo) UpdateFirmwareVersionStatus(ctx context.Context, id string, isActive bool) error {
	firmware, ok := r.firmwareVersions[id]
	if !ok {
		return nil
	}
	firmware.IsActive = isActive
	return nil
}

// CreateFirmwareUpdate creates a new firmware update
func (r *MockDeviceRepo) CreateFirmwareUpdate(ctx context.Context, update *FirmwareUpdate) error {
	r.firmwareUpdates[update.UpdateID] = update
	return nil
}

// GetFirmwareUpdate retrieves a firmware update by ID
func (r *MockDeviceRepo) GetFirmwareUpdate(ctx context.Context, updateID string) (*FirmwareUpdate, error) {
	update, ok := r.firmwareUpdates[updateID]
	if !ok {
		return nil, nil
	}
	return update, nil
}

// GetFirmwareUpdateByDevice retrieves a firmware update by device
func (r *MockDeviceRepo) GetFirmwareUpdateByDevice(ctx context.Context, deviceID string) (*FirmwareUpdate, error) {
	for _, update := range r.firmwareUpdates {
		if update.DeviceID == deviceID {
			return update, nil
		}
	}
	return nil, nil
}

// UpdateFirmwareUpdateStatus updates firmware update status
func (r *MockDeviceRepo) UpdateFirmwareUpdateStatus(ctx context.Context, updateID, status, progress, errorMessage string) error {
	update, ok := r.firmwareUpdates[updateID]
	if !ok {
		return nil
	}
	update.Status = status
	update.Progress = progress
	update.ErrorMessage = errorMessage
	if status == "completed" {
		now := time.Now()
		update.CompletedAt = &now
	}
	return nil
}

// CreateAlert creates a new device alert
func (r *MockDeviceRepo) CreateAlert(ctx context.Context, alert *DeviceAlert) error {
	r.alerts[alert.AlertID] = alert
	return nil
}

// GetAlertByID retrieves a device alert by ID
func (r *MockDeviceRepo) GetAlertByID(ctx context.Context, alertID string) (*DeviceAlert, error) {
	alert, ok := r.alerts[alertID]
	if !ok {
		return nil, nil
	}
	return alert, nil
}

// ListAlerts lists all device alerts
func (r *MockDeviceRepo) ListAlerts(ctx context.Context, page, pageSize int, deviceID, severity, status string) ([]*DeviceAlert, int, error) {
	var alerts []*DeviceAlert
	for _, alert := range r.alerts {
		if deviceID != "" && alert.DeviceID != deviceID {
			continue
		}
		if severity != "" && alert.Severity != severity {
			continue
		}
		if status != "" && alert.Status != status {
			continue
		}
		alerts = append(alerts, alert)
	}
	return alerts, len(alerts), nil
}

// UpdateAlertStatus updates device alert status
func (r *MockDeviceRepo) UpdateAlertStatus(ctx context.Context, alertID, status, acknowledgedBy, notes string) error {
	alert, ok := r.alerts[alertID]
	if !ok {
		return nil
	}
	alert.Status = status
	alert.AcknowledgedBy = acknowledgedBy
	if acknowledgedBy != "" {
		now := time.Now()
		alert.AcknowledgedAt = &now
	}
	alert.Notes = notes
	return nil
}

// CreateCommand creates a new device command
func (r *MockDeviceRepo) CreateCommand(ctx context.Context, command *Command) error {
	r.commands[command.CommandID] = command
	return nil
}

// GetCommandByID retrieves a device command by ID
func (r *MockDeviceRepo) GetCommandByID(ctx context.Context, commandID string) (*Command, error) {
	command, ok := r.commands[commandID]
	if !ok {
		return nil, nil
	}
	return command, nil
}

// UpdateCommandStatus updates device command status
func (r *MockDeviceRepo) UpdateCommandStatus(ctx context.Context, commandID, status, result string) error {
	command, ok := r.commands[commandID]
	if !ok {
		return nil
	}
	command.Status = status
	command.Result = result
	if status == "completed" {
		now := time.Now()
		command.ExecutedAt = &now
	}
	return nil
}

// TestCreateDevice tests device creation
func TestCreateDevice(t *testing.T) {
	repo := NewMockDeviceRepo()
	uc := NewDeviceUseCase(repo, log.NewStdLogger(t.Writer()))

	req := &v1.CreateDeviceRequest{
		DeviceId:   "test-device-1",
		DeviceType: "gate",
		Status:     "active",
		Protocol:   "http",
	}

	device, err := uc.CreateDevice(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, "test-device-1", device.DeviceId)
	assert.Equal(t, "gate", device.DeviceType)
	assert.Equal(t, "active", device.Status)
	assert.Equal(t, "http", device.Protocol)
}

// TestGetDevice tests device retrieval
func TestGetDevice(t *testing.T) {
	repo := NewMockDeviceRepo()
	uc := NewDeviceUseCase(repo, log.NewStdLogger(t.Writer()))

	// Create a device first
	req := &v1.CreateDeviceRequest{
		DeviceId:   "test-device-1",
		DeviceType: "gate",
		Status:     "active",
	}
	_, err := uc.CreateDevice(context.Background(), req)
	assert.NoError(t, err)

	// Get the device
	device, err := uc.GetDevice(context.Background(), "test-device-1")
	assert.NoError(t, err)
	assert.NotNil(t, device)
	assert.Equal(t, "test-device-1", device.DeviceId)
}

// TestUpdateDevice tests device update
func TestUpdateDevice(t *testing.T) {
	repo := NewMockDeviceRepo()
	uc := NewDeviceUseCase(repo, log.NewStdLogger(t.Writer()))

	// Create a device first
	req := &v1.CreateDeviceRequest{
		DeviceId:   "test-device-1",
		DeviceType: "gate",
		Status:     "active",
	}
	_, err := uc.CreateDevice(context.Background(), req)
	assert.NoError(t, err)

	// Update the device
	updateReq := &v1.UpdateDeviceRequest{
		DeviceId:   "test-device-1",
		DeviceType: "camera",
		Status:     "inactive",
	}
	updatedDevice, err := uc.UpdateDevice(context.Background(), updateReq)
	assert.NoError(t, err)
	assert.NotNil(t, updatedDevice)
	assert.Equal(t, "camera", updatedDevice.DeviceType)
	assert.Equal(t, "inactive", updatedDevice.Status)
}

// TestDeleteDevice tests device deletion
func TestDeleteDevice(t *testing.T) {
	repo := NewMockDeviceRepo()
	uc := NewDeviceUseCase(repo, log.NewStdLogger(t.Writer()))

	// Create a device first
	req := &v1.CreateDeviceRequest{
		DeviceId:   "test-device-1",
		DeviceType: "gate",
		Status:     "active",
	}
	_, err := uc.CreateDevice(context.Background(), req)
	assert.NoError(t, err)

	// Delete the device
	err = uc.DeleteDevice(context.Background(), "test-device-1")
	assert.NoError(t, err)

	// Try to get the device (should not found)
	device, err := uc.GetDevice(context.Background(), "test-device-1")
	assert.Error(t, err)
	assert.Nil(t, device)
}

// TestHeartbeat tests device heartbeat
func TestHeartbeat(t *testing.T) {
	repo := NewMockDeviceRepo()
	uc := NewDeviceUseCase(repo, log.NewStdLogger(t.Writer()))

	// Create a device first
	req := &v1.CreateDeviceRequest{
		DeviceId:   "test-device-1",
		DeviceType: "gate",
		Status:     "active",
	}
	_, err := uc.CreateDevice(context.Background(), req)
	assert.NoError(t, err)

	// Send heartbeat
	heartbeatReq := &v1.HeartbeatRequest{
		DeviceId: "test-device-1",
		Status:   "online",
		FirmwareVersion: "1.0.0",
		Metrics: map[string]string{
			"cpu": "25%",
			"memory": "45%",
		},
	}
	response, err := uc.Heartbeat(context.Background(), heartbeatReq)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, 200, int(response.Code))
	assert.Equal(t, "Heartbeat received", response.Message)
}

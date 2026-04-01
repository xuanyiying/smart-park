// Package data provides data access for device management
package data

import (
	"context"
	"time"

	"github.com/xuanyiying/smart-park/internal/device/biz"
	"github.com/xuanyiying/smart-park/internal/device/data/ent"
	"github.com/xuanyiying/smart-park/internal/device/data/ent/command"
	"github.com/xuanyiying/smart-park/internal/device/data/ent/device"
	"github.com/xuanyiying/smart-park/internal/device/data/ent/devicealert"
	"github.com/xuanyiying/smart-park/internal/device/data/ent/devicemetric"
	"github.com/xuanyiying/smart-park/internal/device/data/ent/devicestatushistory"
	"github.com/xuanyiying/smart-park/internal/device/data/ent/firmwareupdate"
	"github.com/xuanyiying/smart-park/internal/device/data/ent/firmwareversion"
)

// DeviceRepoImpl implements the DeviceRepo interface
type DeviceRepoImpl struct {
	client *ent.Client
}

// NewDeviceRepo creates a new DeviceRepo
func NewDeviceRepo(client *ent.Client) biz.DeviceRepo {
	return &DeviceRepoImpl{
		client: client,
	}
}

// CreateDevice creates a new device
func (r *DeviceRepoImpl) CreateDevice(ctx context.Context, deviceObj *biz.Device) error {
	_, err := r.client.Device.Create().
		SetDeviceID(deviceObj.DeviceID).
		SetDeviceType(deviceObj.DeviceType).
		SetStatus(deviceObj.Status).
		SetOnline(deviceObj.Online).
		SetProtocol(deviceObj.Protocol).
		SetConfig(deviceObj.Config).
		SetFirmwareVersion(deviceObj.FirmwareVersion).
		SetCreatedAt(deviceObj.CreatedAt).
		SetUpdatedAt(deviceObj.UpdatedAt).
		Save(ctx)
	return err
}

// GetDeviceByID retrieves a device by ID
func (r *DeviceRepoImpl) GetDeviceByID(ctx context.Context, deviceID string) (*biz.Device, error) {
	dev, err := r.client.Device.Query().
		Where(device.DeviceID(deviceID)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToDevice(dev), nil
}

// GetDeviceByCode retrieves a device by code
func (r *DeviceRepoImpl) GetDeviceByCode(ctx context.Context, deviceCode string) (*biz.Device, error) {
	dev, err := r.client.Device.Query().
		Where(device.DeviceID(deviceCode)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToDevice(dev), nil
}

// UpdateDevice updates an existing device
func (r *DeviceRepoImpl) UpdateDevice(ctx context.Context, deviceObj *biz.Device) error {
	_, err := r.client.Device.UpdateOneID(deviceObj.DeviceID).
		SetDeviceType(deviceObj.DeviceType).
		SetStatus(deviceObj.Status).
		SetConfig(deviceObj.Config).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

// DeleteDevice deletes a device by ID
func (r *DeviceRepoImpl) DeleteDevice(ctx context.Context, deviceID string) error {
	return r.client.Device.DeleteOneID(deviceID).Exec(ctx)
}

// ListDevices lists all devices with pagination
func (r *DeviceRepoImpl) ListDevices(ctx context.Context, page, pageSize int, deviceType, status string) ([]*biz.Device, int, error) {
	query := r.client.Device.Query()

	if deviceType != "" {
		query = query.Where(device.DeviceType(deviceType))
	}
	if status != "" {
		query = query.Where(device.Status(status))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	devices, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*biz.Device, len(devices))
	for i, dev := range devices {
		result[i] = r.convertToDevice(dev)
	}

	return result, total, nil
}

// UpdateDeviceHeartbeat updates device heartbeat
func (r *DeviceRepoImpl) UpdateDeviceHeartbeat(ctx context.Context, deviceID string) error {
	now := time.Now()
	_, err := r.client.Device.UpdateOneID(deviceID).
		SetLastHeartbeat(now).
		SetOnline(true).
		SetUpdatedAt(now).
		Save(ctx)
	return err
}

// UpdateDeviceFirmwareVersion updates device firmware version
func (r *DeviceRepoImpl) UpdateDeviceFirmwareVersion(ctx context.Context, deviceID, firmwareVersion string) error {
	_, err := r.client.Device.UpdateOneID(deviceID).
		SetFirmwareVersion(firmwareVersion).
		SetUpdatedAt(time.Now()).
		Save(ctx)
	return err
}

// CreateFirmwareVersion creates a new firmware version
func (r *DeviceRepoImpl) CreateFirmwareVersion(ctx context.Context, firmware *biz.FirmwareVersion) error {
	_, err := r.client.FirmwareVersion.Create().
		SetID(firmware.ID).
		SetDeviceType(firmware.DeviceType).
		SetVersion(firmware.Version).
		SetURL(firmware.URL).
		SetChecksum(firmware.Checksum).
		SetDescription(firmware.Description).
		SetIsActive(firmware.IsActive).
		SetCreatedAt(firmware.CreatedAt).
		Save(ctx)
	return err
}

// GetFirmwareVersionByID retrieves a firmware version by ID
func (r *DeviceRepoImpl) GetFirmwareVersionByID(ctx context.Context, id string) (*biz.FirmwareVersion, error) {
	fw, err := r.client.FirmwareVersion.Query().
		Where(firmwareversion.ID(id)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToFirmwareVersion(fw), nil
}

// GetFirmwareVersionByVersion retrieves a firmware version by version
func (r *DeviceRepoImpl) GetFirmwareVersionByVersion(ctx context.Context, deviceType, version string) (*biz.FirmwareVersion, error) {
	fw, err := r.client.FirmwareVersion.Query().
		Where(firmwareversion.DeviceType(deviceType), firmwareversion.Version(version)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToFirmwareVersion(fw), nil
}

// ListFirmwareVersions lists all firmware versions
func (r *DeviceRepoImpl) ListFirmwareVersions(ctx context.Context, page, pageSize int, deviceType string) ([]*biz.FirmwareVersion, int, error) {
	query := r.client.FirmwareVersion.Query()

	if deviceType != "" {
		query = query.Where(firmwareversion.DeviceType(deviceType))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	firmwares, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*biz.FirmwareVersion, len(firmwares))
	for i, fw := range firmwares {
		result[i] = r.convertToFirmwareVersion(fw)
	}

	return result, total, nil
}

// UpdateFirmwareVersionStatus updates firmware version status
func (r *DeviceRepoImpl) UpdateFirmwareVersionStatus(ctx context.Context, id string, isActive bool) error {
	_, err := r.client.FirmwareVersion.UpdateOneID(id).
		SetIsActive(isActive).
		Save(ctx)
	return err
}

// CreateFirmwareUpdate creates a new firmware update
func (r *DeviceRepoImpl) CreateFirmwareUpdate(ctx context.Context, update *biz.FirmwareUpdate) error {
	_, err := r.client.FirmwareUpdate.Create().
		SetUpdateID(update.UpdateID).
		SetDeviceID(update.DeviceID).
		SetFirmwareVersion(update.FirmwareVersion).
		SetStatus(update.Status).
		SetProgress(update.Progress).
		SetErrorMessage(update.ErrorMessage).
		SetStartedAt(update.StartedAt).
		Save(ctx)
	return err
}

// GetFirmwareUpdate retrieves a firmware update by ID
func (r *DeviceRepoImpl) GetFirmwareUpdate(ctx context.Context, updateID string) (*biz.FirmwareUpdate, error) {
	upd, err := r.client.FirmwareUpdate.Query().
		Where(firmwareupdate.UpdateID(updateID)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToFirmwareUpdate(upd), nil
}

// GetFirmwareUpdateByDevice retrieves a firmware update by device
func (r *DeviceRepoImpl) GetFirmwareUpdateByDevice(ctx context.Context, deviceID string) (*biz.FirmwareUpdate, error) {
	upd, err := r.client.FirmwareUpdate.Query().
		Where(firmwareupdate.DeviceID(deviceID)).
		Order(firmwareupdate.ByStartedAtDesc()).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToFirmwareUpdate(upd), nil
}

// UpdateFirmwareUpdateStatus updates firmware update status
func (r *DeviceRepoImpl) UpdateFirmwareUpdateStatus(ctx context.Context, updateID, status, progress, errorMessage string) error {
	update := r.client.FirmwareUpdate.UpdateOneID(updateID).
		SetStatus(status).
		SetProgress(progress)

	if errorMessage != "" {
		update = update.SetErrorMessage(errorMessage)
	}

	if status == "completed" {
		update = update.SetCompletedAt(time.Now())
	}

	_, err := update.Save(ctx)
	return err
}

// CreateAlert creates a new device alert
func (r *DeviceRepoImpl) CreateAlert(ctx context.Context, alert *biz.DeviceAlert) error {
	_, err := r.client.DeviceAlert.Create().
		SetAlertID(alert.AlertID).
		SetDeviceID(alert.DeviceID).
		SetType(alert.Type).
		SetSeverity(alert.Severity).
		SetMessage(alert.Message).
		SetStatus(alert.Status).
		SetCreatedAt(alert.CreatedAt).
		SetMetadata(alert.Metadata).
		Save(ctx)
	return err
}

// GetAlertByID retrieves a device alert by ID
func (r *DeviceRepoImpl) GetAlertByID(ctx context.Context, alertID string) (*biz.DeviceAlert, error) {
	alert, err := r.client.DeviceAlert.Query().
		Where(devicealert.AlertID(alertID)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToDeviceAlert(alert), nil
}

// ListAlerts lists all device alerts
func (r *DeviceRepoImpl) ListAlerts(ctx context.Context, page, pageSize int, deviceID, severity, status string) ([]*biz.DeviceAlert, int, error) {
	query := r.client.DeviceAlert.Query()

	if deviceID != "" {
		query = query.Where(devicealert.DeviceID(deviceID))
	}
	if severity != "" {
		query = query.Where(devicealert.Severity(severity))
	}
	if status != "" {
		query = query.Where(devicealert.Status(status))
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	alerts, err := query.
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Order(devicealert.ByCreatedAtDesc()).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*biz.DeviceAlert, len(alerts))
	for i, alert := range alerts {
		result[i] = r.convertToDeviceAlert(alert)
	}

	return result, total, nil
}

// UpdateAlertStatus updates device alert status
func (r *DeviceRepoImpl) UpdateAlertStatus(ctx context.Context, alertID, status, acknowledgedBy, notes string) error {
	update := r.client.DeviceAlert.UpdateOneID(alertID).
		SetStatus(status)

	if acknowledgedBy != "" {
		update = update.SetAcknowledgedBy(acknowledgedBy).
			SetAcknowledgedAt(time.Now())
	}

	if notes != "" {
		update = update.SetNotes(notes)
	}

	_, err := update.Save(ctx)
	return err
}

// CreateCommand creates a new device command
func (r *DeviceRepoImpl) CreateCommand(ctx context.Context, cmd *biz.Command) error {
	_, err := r.client.Command.Create().
		SetCommandID(cmd.CommandID).
		SetDeviceID(cmd.DeviceID).
		SetCommand(cmd.Command).
		SetParams(cmd.Params).
		SetStatus(cmd.Status).
		SetCreatedAt(cmd.CreatedAt).
		Save(ctx)
	return err
}

// GetCommandByID retrieves a device command by ID
func (r *DeviceRepoImpl) GetCommandByID(ctx context.Context, commandID string) (*biz.Command, error) {
	cmd, err := r.client.Command.Query().
		Where(command.CommandID(commandID)).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToCommand(cmd), nil
}

// UpdateCommandStatus updates device command status
func (r *DeviceRepoImpl) UpdateCommandStatus(ctx context.Context, commandID, status, result string) error {
	update := r.client.Command.UpdateOneID(commandID).
		SetStatus(status)

	if result != "" {
		update = update.SetResult(result)
	}

	if status == "completed" {
		update = update.SetExecutedAt(time.Now())
	}

	_, err := update.Save(ctx)
	return err
}

// MonitoringRepoImpl implements the MonitoringRepo interface
type MonitoringRepoImpl struct {
	client *ent.Client
}

// NewMonitoringRepo creates a new MonitoringRepo
func NewMonitoringRepo(client *ent.Client) biz.MonitoringRepo {
	return &MonitoringRepoImpl{
		client: client,
	}
}

// CreateMetric creates a new device metric
func (r *MonitoringRepoImpl) CreateMetric(ctx context.Context, metric *biz.DeviceMetric) error {
	_, err := r.client.DeviceMetric.Create().
		SetID(metric.ID).
		SetDeviceID(metric.DeviceID).
		SetMetric(metric.Metric).
		SetValue(metric.Value).
		SetUnit(metric.Unit).
		SetTimestamp(metric.Timestamp).
		Save(ctx)
	return err
}

// GetMetrics retrieves device metrics
func (r *MonitoringRepoImpl) GetMetrics(ctx context.Context, deviceID string, metric string, startTime, endTime time.Time, limit int) ([]*biz.DeviceMetric, error) {
	query := r.client.DeviceMetric.Query().
		Where(devicemetric.DeviceID(deviceID))

	if metric != "" {
		query = query.Where(devicemetric.Metric(metric))
	}

	if !startTime.IsZero() {
		query = query.Where(devicemetric.TimestampGTE(startTime))
	}

	if !endTime.IsZero() {
		query = query.Where(devicemetric.TimestampLTE(endTime))
	}

	metrics, err := query.
		Order(devicemetric.ByTimestampDesc()).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.DeviceMetric, len(metrics))
	for i, m := range metrics {
		result[i] = r.convertToDeviceMetric(m)
	}

	return result, nil
}

// GetLatestMetric retrieves the latest device metric
func (r *MonitoringRepoImpl) GetLatestMetric(ctx context.Context, deviceID string, metric string) (*biz.DeviceMetric, error) {
	m, err := r.client.DeviceMetric.Query().
		Where(devicemetric.DeviceID(deviceID), devicemetric.Metric(metric)).
		Order(devicemetric.ByTimestampDesc()).
		First(ctx)
	if err != nil {
		return nil, err
	}

	return r.convertToDeviceMetric(m), nil
}

// CreateStatusHistory creates a new device status history record
func (r *MonitoringRepoImpl) CreateStatusHistory(ctx context.Context, history *biz.DeviceStatusHistory) error {
	_, err := r.client.DeviceStatusHistory.Create().
		SetID(history.ID).
		SetDeviceID(history.DeviceID).
		SetStatus(history.Status).
		SetOnline(history.Online).
		SetFirmwareVersion(history.FirmwareVersion).
		SetTimestamp(history.Timestamp).
		SetMetadata(history.Metadata).
		Save(ctx)
	return err
}

// GetStatusHistory retrieves device status history
func (r *MonitoringRepoImpl) GetStatusHistory(ctx context.Context, deviceID string, startTime, endTime time.Time, limit int) ([]*biz.DeviceStatusHistory, error) {
	query := r.client.DeviceStatusHistory.Query().
		Where(devicestatushistory.DeviceID(deviceID))

	if !startTime.IsZero() {
		query = query.Where(devicestatushistory.TimestampGTE(startTime))
	}

	if !endTime.IsZero() {
		query = query.Where(devicestatushistory.TimestampLTE(endTime))
	}

	history, err := query.
		Order(devicestatushistory.ByTimestampDesc()).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.DeviceStatusHistory, len(history))
	for i, h := range history {
		result[i] = r.convertToDeviceStatusHistory(h)
	}

	return result, nil
}

// convertToDevice converts ent.Device to biz.Device
func (r *DeviceRepoImpl) convertToDevice(dev *ent.Device) *biz.Device {
	return &biz.Device{
		DeviceID:        dev.DeviceID,
		DeviceType:      dev.DeviceType,
		Status:          dev.Status,
		Online:          dev.Online,
		LastHeartbeat:   dev.LastHeartbeat,
		Protocol:        dev.Protocol,
		Config:          dev.Config,
		FirmwareVersion: dev.FirmwareVersion,
		CreatedAt:       dev.CreatedAt,
		UpdatedAt:       dev.UpdatedAt,
	}
}

// convertToFirmwareVersion converts ent.FirmwareVersion to biz.FirmwareVersion
func (r *DeviceRepoImpl) convertToFirmwareVersion(fw *ent.FirmwareVersion) *biz.FirmwareVersion {
	return &biz.FirmwareVersion{
		ID:          fw.ID,
		DeviceType:  fw.DeviceType,
		Version:     fw.Version,
		URL:         fw.URL,
		Checksum:    fw.Checksum,
		Description: fw.Description,
		IsActive:    fw.IsActive,
		CreatedAt:   fw.CreatedAt,
	}
}

// convertToFirmwareUpdate converts ent.FirmwareUpdate to biz.FirmwareUpdate
func (r *DeviceRepoImpl) convertToFirmwareUpdate(upd *ent.FirmwareUpdate) *biz.FirmwareUpdate {
	return &biz.FirmwareUpdate{
		UpdateID:         upd.UpdateID,
		DeviceID:         upd.DeviceID,
		FirmwareVersion:  upd.FirmwareVersion,
		Status:           upd.Status,
		Progress:         upd.Progress,
		ErrorMessage:     upd.ErrorMessage,
		StartedAt:        upd.StartedAt,
		CompletedAt:      upd.CompletedAt,
	}
}

// convertToDeviceAlert converts ent.DeviceAlert to biz.DeviceAlert
func (r *DeviceRepoImpl) convertToDeviceAlert(alert *ent.DeviceAlert) *biz.DeviceAlert {
	return &biz.DeviceAlert{
		AlertID:         alert.AlertID,
		DeviceID:        alert.DeviceID,
		Type:            alert.Type,
		Severity:        alert.Severity,
		Message:         alert.Message,
		Status:          alert.Status,
		CreatedAt:       alert.CreatedAt,
		AcknowledgedAt:  alert.AcknowledgedAt,
		AcknowledgedBy:  alert.AcknowledgedBy,
		Notes:           alert.Notes,
		Metadata:        alert.Metadata,
	}
}

// convertToCommand converts ent.Command to biz.Command
func (r *DeviceRepoImpl) convertToCommand(cmd *ent.Command) *biz.Command {
	return &biz.Command{
		CommandID:   cmd.CommandID,
		DeviceID:    cmd.DeviceID,
		Command:     cmd.Command,
		Params:      cmd.Params,
		Status:      cmd.Status,
		CreatedAt:   cmd.CreatedAt,
		ExecutedAt:  cmd.ExecutedAt,
		Result:      cmd.Result,
	}
}

// convertToDeviceMetric converts ent.DeviceMetric to biz.DeviceMetric
func (r *MonitoringRepoImpl) convertToDeviceMetric(m *ent.DeviceMetric) *biz.DeviceMetric {
	return &biz.DeviceMetric{
		ID:        m.ID,
		DeviceID:  m.DeviceID,
		Metric:    m.Metric,
		Value:     m.Value,
		Unit:      m.Unit,
		Timestamp: m.Timestamp,
	}
}

// convertToDeviceStatusHistory converts ent.DeviceStatusHistory to biz.DeviceStatusHistory
func (r *MonitoringRepoImpl) convertToDeviceStatusHistory(h *ent.DeviceStatusHistory) *biz.DeviceStatusHistory {
	return &biz.DeviceStatusHistory{
		ID:              h.ID,
		DeviceID:        h.DeviceID,
		Status:          h.Status,
		Online:          h.Online,
		FirmwareVersion: h.FirmwareVersion,
		Timestamp:       h.Timestamp,
		Metadata:        h.Metadata,
	}
}

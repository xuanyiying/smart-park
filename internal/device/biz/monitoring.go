// Package biz provides business logic for device monitoring
package biz

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/xuanyiying/smart-park/api/device/v1"
)

// DeviceMetric represents a device metric entity
type DeviceMetric struct {
	ID        string
	DeviceID  string
	Metric    string
	Value     string
	Unit      string
	Timestamp time.Time
}

// DeviceStatusHistory represents a device status history record
type DeviceStatusHistory struct {
	ID            string
	DeviceID      string
	Status        string
	Online        bool
	FirmwareVersion string
	Timestamp     time.Time
	Metadata      map[string]string
}

// MonitoringRepo defines the monitoring repository interface
type MonitoringRepo interface {
	// Metrics management
	CreateMetric(ctx context.Context, metric *DeviceMetric) error
	GetMetrics(ctx context.Context, deviceID string, metric string, startTime, endTime time.Time, limit int) ([]*DeviceMetric, error)
	GetLatestMetric(ctx context.Context, deviceID string, metric string) (*DeviceMetric, error)

	// Status history management
	CreateStatusHistory(ctx context.Context, history *DeviceStatusHistory) error
	GetStatusHistory(ctx context.Context, deviceID string, startTime, endTime time.Time, limit int) ([]*DeviceStatusHistory, error)
}

// MonitoringUseCase handles device monitoring business logic
type MonitoringUseCase struct {
	repo  MonitoringRepo
	deviceRepo DeviceRepo
}

// NewMonitoringUseCase creates a new MonitoringUseCase
func NewMonitoringUseCase(repo MonitoringRepo, deviceRepo DeviceRepo) *MonitoringUseCase {
	return &MonitoringUseCase{
		repo:  repo,
		deviceRepo: deviceRepo,
	}
}

// RecordDeviceMetrics records device metrics
func (uc *MonitoringUseCase) RecordDeviceMetrics(ctx context.Context, deviceID string, metrics map[string]string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}

	// Check if device exists
	device, err := uc.deviceRepo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// Record each metric
	for metric, value := range metrics {
		deviceMetric := &DeviceMetric{
			ID:        fmt.Sprintf("%s-%d", deviceID, time.Now().UnixNano()),
			DeviceID:  deviceID,
			Metric:    metric,
			Value:     value,
			Unit:      "", // TODO: Add unit support
			Timestamp: time.Now(),
		}

		if err := uc.repo.CreateMetric(ctx, deviceMetric); err != nil {
			return fmt.Errorf("failed to record metric: %w", err)
		}
	}

	return nil
}

// GetDeviceMetrics gets device metrics
func (uc *MonitoringUseCase) GetDeviceMetrics(ctx context.Context, deviceID string, metric string, startTime, endTime time.Time, limit int) ([]*v1.DeviceMetric, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	metrics, err := uc.repo.GetMetrics(ctx, deviceID, metric, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	result := make([]*v1.DeviceMetric, len(metrics))
	for i, m := range metrics {
		result[i] = &v1.DeviceMetric{
			Metric:    m.Metric,
			Value:     m.Value,
			Unit:      m.Unit,
			Timestamp: m.Timestamp.Format(time.RFC3339),
		}
	}

	return result, nil
}

// GetDeviceStatusHistory gets device status history
func (uc *MonitoringUseCase) GetDeviceStatusHistory(ctx context.Context, deviceID string, startTime, endTime time.Time, limit int) ([]*v1.DeviceStatusHistory, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	history, err := uc.repo.GetStatusHistory(ctx, deviceID, startTime, endTime, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get status history: %w", err)
	}

	result := make([]*v1.DeviceStatusHistory, len(history))
	for i, h := range history {
		result[i] = &v1.DeviceStatusHistory{
			Status:          h.Status,
			Online:          h.Online,
			FirmwareVersion: h.FirmwareVersion,
			Timestamp:       h.Timestamp.Format(time.RFC3339),
			Metadata:        h.Metadata,
		}
	}

	return result, nil
}

// RecordDeviceStatus records device status change
func (uc *MonitoringUseCase) RecordDeviceStatus(ctx context.Context, deviceID string, status string, online bool, firmwareVersion string, metadata map[string]string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}

	// Check if device exists
	device, err := uc.deviceRepo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// Record status history
	history := &DeviceStatusHistory{
		ID:            fmt.Sprintf("%s-%d", deviceID, time.Now().UnixNano()),
		DeviceID:      deviceID,
		Status:        status,
		Online:        online,
		FirmwareVersion: firmwareVersion,
		Timestamp:     time.Now(),
		Metadata:      metadata,
	}

	if err := uc.repo.CreateStatusHistory(ctx, history); err != nil {
		return fmt.Errorf("failed to record status history: %w", err)
	}

	return nil
}

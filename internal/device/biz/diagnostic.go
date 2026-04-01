// Package biz provides business logic for device diagnostics
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
)

// DiagnosticUseCase handles device diagnostic business logic
type DiagnosticUseCase struct {
	repo  DeviceRepo
	log   *log.Helper
}

// NewDiagnosticUseCase creates a new DiagnosticUseCase
func NewDiagnosticUseCase(repo DeviceRepo, logger log.Logger) *DiagnosticUseCase {
	return &DiagnosticUseCase{
		repo:  repo,
		log:   log.NewHelper(logger),
	}
}

// AnalyzeDeviceHealth analyzes device health based on metrics and status
func (uc *DiagnosticUseCase) AnalyzeDeviceHealth(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Get device information
	device, err := uc.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	// Check device online status
	online := false
	if device.LastHeartbeat != nil {
		online = time.Since(*device.LastHeartbeat) < 5*time.Minute
	}

	// Get recent alerts
	alerts, _, err := uc.repo.ListAlerts(ctx, 1, 10, deviceID, "", "")
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get alerts: %v", err)
	}

	// Analyze health status
	healthStatus := "healthy"
	riskLevel := "low"

	// Check for critical alerts
	criticalAlerts := 0
	for _, alert := range alerts {
		if alert.Severity == "critical" {
			criticalAlerts++
		}
	}

	// Determine health status based on alerts and online status
	if !online {
		healthStatus = "offline"
		riskLevel = "high"
	} else if criticalAlerts > 0 {
		healthStatus = "unhealthy"
		riskLevel = "high"
	} else if len(alerts) > 0 {
		healthStatus = "degraded"
		riskLevel = "medium"
	}

	// Generate health report
	report := map[string]interface{}{
		"deviceId":     deviceID,
		"deviceType":   device.DeviceType,
		"online":       online,
		"healthStatus": healthStatus,
		"riskLevel":    riskLevel,
		"lastHeartbeat": device.LastHeartbeat,
		"firmwareVersion": device.FirmwareVersion,
		"criticalAlerts": criticalAlerts,
		"totalAlerts":   len(alerts),
		"recommendations": uc.generateRecommendations(device, online, criticalAlerts, alerts),
	}

	return report, nil
}

// generateRecommendations generates recommendations based on device status
func (uc *DiagnosticUseCase) generateRecommendations(device *Device, online bool, criticalAlerts int, alerts []*DeviceAlert) []string {
	recommendations := []string{}

	if !online {
		recommendations = append(recommendations, "Check device connectivity and power supply")
		recommendations = append(recommendations, "Verify network connection")
	}

	if criticalAlerts > 0 {
		recommendations = append(recommendations, "Address critical alerts immediately")
	}

	// Check firmware version
	// TODO: Compare with latest firmware version

	// Check for common issues based on alert types
	alertTypes := make(map[string]int)
	for _, alert := range alerts {
		alertTypes[alert.Type]++
	}

	for alertType, count := range alertTypes {
		if count > 3 {
			switch alertType {
			case "connection_error":
				recommendations = append(recommendations, "Check network stability and device connection settings")
			case "power_issue":
				recommendations = append(recommendations, "Inspect power supply and battery status")
			case "sensor_failure":
				recommendations = append(recommendations, "Calibrate or replace faulty sensors")
			case "firmware_error":
				recommendations = append(recommendations, "Update device firmware to latest version")
			}
		}
	}

	return recommendations
}

// PredictDeviceFailures predicts potential device failures based on historical data
func (uc *DiagnosticUseCase) PredictDeviceFailures(ctx context.Context, deviceID string) ([]map[string]interface{}, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// TODO: Implement predictive analytics based on historical metrics and alerts
	// For now, return a placeholder
	predictions := []map[string]interface{}{
		{
			"type":        "battery_degradation",
			"probability": "75%",
			"description": "Battery degradation detected",
			"recommendation": "Replace battery within 30 days",
			"timeframe":   "30 days",
		},
		{
			"type":        "firmware_outdated",
			"probability": "90%",
			"description": "Firmware version is outdated",
			"recommendation": "Update firmware to latest version",
			"timeframe":   "Immediate",
		},
	}

	return predictions, nil
}

// CreateAlert creates a new device alert
func (uc *DiagnosticUseCase) CreateAlert(ctx context.Context, deviceID, alertType, severity, message string, metadata map[string]string) error {
	if deviceID == "" {
		return fmt.Errorf("device id is required")
	}
	if alertType == "" {
		return fmt.Errorf("alert type is required")
	}
	if severity == "" {
		return fmt.Errorf("severity is required")
	}
	if message == "" {
		return fmt.Errorf("message is required")
	}

	// Check if device exists
	device, err := uc.repo.GetDeviceByID(ctx, deviceID)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return fmt.Errorf("device not found: %s", deviceID)
	}

	// Create alert
	alert := &DeviceAlert{
		AlertID:    uuid.New().String(),
		DeviceID:   deviceID,
		Type:       alertType,
		Severity:   severity,
		Message:    message,
		Status:     "active",
		CreatedAt:  time.Now(),
		Metadata:   metadata,
	}

	if err := uc.repo.CreateAlert(ctx, alert); err != nil {
		return fmt.Errorf("failed to create alert: %w", err)
	}

	return nil
}

// ResolveAlert resolves a device alert
func (uc *DiagnosticUseCase) ResolveAlert(ctx context.Context, alertID, notes string) error {
	if alertID == "" {
		return fmt.Errorf("alert id is required")
	}

	// Check if alert exists
	alert, err := uc.repo.GetAlertByID(ctx, alertID)
	if err != nil {
		return fmt.Errorf("failed to get alert: %w", err)
	}
	if alert == nil {
		return fmt.Errorf("alert not found: %s", alertID)
	}

	// Update alert status
	if err := uc.repo.UpdateAlertStatus(ctx, alertID, "resolved", "system", notes); err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	return nil
}

// GetDeviceAlertHistory gets device alert history
func (uc *DiagnosticUseCase) GetDeviceAlertHistory(ctx context.Context, deviceID string, days int) ([]*DeviceAlert, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Calculate time range
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)

	// Get alerts within time range
	// TODO: Implement time-based filtering in repo
	alerts, _, err := uc.repo.ListAlerts(ctx, 1, 100, deviceID, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get alerts: %w", err)
	}

	// Filter alerts by time
	filteredAlerts := []*DeviceAlert{}
	for _, alert := range alerts {
		if alert.CreatedAt.After(startTime) && alert.CreatedAt.Before(endTime) {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	return filteredAlerts, nil
}

// GetAlertStatistics gets alert statistics for a device
func (uc *DiagnosticUseCase) GetAlertStatistics(ctx context.Context, deviceID string, days int) (map[string]interface{}, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	// Get alert history
	alerts, err := uc.GetDeviceAlertHistory(ctx, deviceID, days)
	if err != nil {
		return nil, err
	}

	// Calculate statistics
	stats := map[string]interface{}{
		"totalAlerts":   len(alerts),
		"bySeverity":    make(map[string]int),
		"byType":        make(map[string]int),
		"byStatus":      make(map[string]int),
		"mostFrequent":  "",
		"criticalCount": 0,
	}

	// Count alerts by severity, type, and status
	typeCount := make(map[string]int)
	for _, alert := range alerts {
		// By severity
		if stats["bySeverity"].(map[string]int)[alert.Severity] == 0 {
			stats["bySeverity"].(map[string]int)[alert.Severity] = 0
		}
		stats["bySeverity"].(map[string]int)[alert.Severity]++

		// By type
		typeCount[alert.Type]++
		if stats["byType"].(map[string]int)[alert.Type] == 0 {
			stats["byType"].(map[string]int)[alert.Type] = 0
		}
		stats["byType"].(map[string]int)[alert.Type]++

		// By status
		if stats["byStatus"].(map[string]int)[alert.Status] == 0 {
			stats["byStatus"].(map[string]int)[alert.Status] = 0
		}
		stats["byStatus"].(map[string]int)[alert.Status]++

		// Critical count
		if alert.Severity == "critical" {
			stats["criticalCount"] = stats["criticalCount"].(int) + 1
		}
	}

	// Find most frequent alert type
	maxCount := 0
	mostFrequent := ""
	for alertType, count := range typeCount {
		if count > maxCount {
			maxCount = count
			mostFrequent = alertType
		}
	}
	stats["mostFrequent"] = mostFrequent

	return stats, nil
}

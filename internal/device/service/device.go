// Package service provides device management services
package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	v1 "github.com/xuanyiying/smart-park/api/device/v1"
	"github.com/xuanyiying/smart-park/internal/device/biz"
)

// DeviceService implements the v1.DeviceServiceServer interface
type DeviceService struct {
	v1.UnimplementedDeviceServiceServer

	deviceUC     *biz.DeviceUseCase
	monitoringUC *biz.MonitoringUseCase
	log          *log.Helper
}

// NewDeviceService creates a new DeviceService
func NewDeviceService(deviceUC *biz.DeviceUseCase, monitoringUC *biz.MonitoringUseCase, logger log.Logger) *DeviceService {
	return &DeviceService{
		deviceUC:     deviceUC,
		monitoringUC: monitoringUC,
		log:          log.NewHelper(logger),
	}
}

// ListDevices implements v1.DeviceServiceServer
func (s *DeviceService) ListDevices(ctx context.Context, req *v1.ListDevicesRequest) (*v1.ListDevicesResponse, error) {
	devices, total, err := s.deviceUC.ListDevices(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListDevices error: %v", err)
		return &v1.ListDevicesResponse{
			Code:    500,
			Message: "Failed to list devices",
		}, err
	}

	return &v1.ListDevicesResponse{
		Code:    200,
		Message: "Success",
		Data:    devices,
		Total:   int32(total),
	}, nil
}

// GetDevice implements v1.DeviceServiceServer
func (s *DeviceService) GetDevice(ctx context.Context, req *v1.GetDeviceRequest) (*v1.GetDeviceResponse, error) {
	device, err := s.deviceUC.GetDevice(ctx, req.DeviceId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDevice error: %v", err)
		return &v1.GetDeviceResponse{
			Code:    500,
			Message: "Failed to get device",
		}, err
	}

	return &v1.GetDeviceResponse{
		Code:    200,
		Message: "Success",
		Data:    device,
	}, nil
}

// CreateDevice implements v1.DeviceServiceServer
func (s *DeviceService) CreateDevice(ctx context.Context, req *v1.CreateDeviceRequest) (*v1.CreateDeviceResponse, error) {
	device, err := s.deviceUC.CreateDevice(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateDevice error: %v", err)
		return &v1.CreateDeviceResponse{
			Code:    500,
			Message: "Failed to create device",
		}, err
	}

	return &v1.CreateDeviceResponse{
		Code:    200,
		Message: "Success",
		Data:    device,
	}, nil
}

// UpdateDevice implements v1.DeviceServiceServer
func (s *DeviceService) UpdateDevice(ctx context.Context, req *v1.UpdateDeviceRequest) (*v1.UpdateDeviceResponse, error) {
	device, err := s.deviceUC.UpdateDevice(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpdateDevice error: %v", err)
		return &v1.UpdateDeviceResponse{
			Code:    500,
			Message: "Failed to update device",
		}, err
	}

	return &v1.UpdateDeviceResponse{
		Code:    200,
		Message: "Success",
		Data:    device,
	}, nil
}

// DeleteDevice implements v1.DeviceServiceServer
func (s *DeviceService) DeleteDevice(ctx context.Context, req *v1.DeleteDeviceRequest) (*v1.DeleteDeviceResponse, error) {
	err := s.deviceUC.DeleteDevice(ctx, req.DeviceId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("DeleteDevice error: %v", err)
		return &v1.DeleteDeviceResponse{
			Code:    500,
			Message: "Failed to delete device",
		}, err
	}

	return &v1.DeleteDeviceResponse{
		Code:    200,
		Message: "Success",
	}, nil
}

// GetDeviceStatus implements v1.DeviceServiceServer
func (s *DeviceService) GetDeviceStatus(ctx context.Context, req *v1.GetDeviceStatusRequest) (*v1.GetDeviceStatusResponse, error) {
	status, err := s.deviceUC.GetDeviceStatus(ctx, req.DeviceId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceStatus error: %v", err)
		return &v1.GetDeviceStatusResponse{
			Code:    500,
			Message: "Failed to get device status",
		}, err
	}

	return &v1.GetDeviceStatusResponse{
		Code:    200,
		Message: "Success",
		Data:    status,
	}, nil
}

// Heartbeat implements v1.DeviceServiceServer
func (s *DeviceService) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) (*v1.HeartbeatResponse, error) {
	response, err := s.deviceUC.Heartbeat(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Heartbeat error: %v", err)
		return &v1.HeartbeatResponse{
			Code:    500,
			Message: "Failed to process heartbeat",
		}, err
	}

	// Record device metrics if provided
	if len(req.Metrics) > 0 {
		if err := s.monitoringUC.RecordDeviceMetrics(ctx, req.DeviceId, req.Metrics); err != nil {
			s.log.WithContext(ctx).Errorf("RecordDeviceMetrics error: %v", err)
		}
	}

	// Record device status
	if err := s.monitoringUC.RecordDeviceStatus(ctx, req.DeviceId, req.Status, true, req.FirmwareVersion, req.Metrics); err != nil {
		s.log.WithContext(ctx).Errorf("RecordDeviceStatus error: %v", err)
	}

	return response, nil
}

// ListFirmwareVersions implements v1.DeviceServiceServer
func (s *DeviceService) ListFirmwareVersions(ctx context.Context, req *v1.ListFirmwareVersionsRequest) (*v1.ListFirmwareVersionsResponse, error) {
	firmwares, total, err := s.deviceUC.ListFirmwareVersions(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListFirmwareVersions error: %v", err)
		return &v1.ListFirmwareVersionsResponse{
			Code:    500,
			Message: "Failed to list firmware versions",
		}, err
	}

	return &v1.ListFirmwareVersionsResponse{
		Code:    200,
		Message: "Success",
		Data:    firmwares,
		Total:   int32(total),
	}, nil
}

// CreateFirmwareVersion implements v1.DeviceServiceServer
func (s *DeviceService) CreateFirmwareVersion(ctx context.Context, req *v1.CreateFirmwareVersionRequest) (*v1.CreateFirmwareVersionResponse, error) {
	firmware, err := s.deviceUC.CreateFirmwareVersion(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateFirmwareVersion error: %v", err)
		return &v1.CreateFirmwareVersionResponse{
			Code:    500,
			Message: "Failed to create firmware version",
		}, err
	}

	return &v1.CreateFirmwareVersionResponse{
		Code:    200,
		Message: "Success",
		Data:    firmware,
	}, nil
}

// UpdateDeviceFirmware implements v1.DeviceServiceServer
func (s *DeviceService) UpdateDeviceFirmware(ctx context.Context, req *v1.UpdateDeviceFirmwareRequest) (*v1.UpdateDeviceFirmwareResponse, error) {
	update, err := s.deviceUC.UpdateDeviceFirmware(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpdateDeviceFirmware error: %v", err)
		return &v1.UpdateDeviceFirmwareResponse{
			Code:    500,
			Message: "Failed to update device firmware",
		}, err
	}

	return &v1.UpdateDeviceFirmwareResponse{
		Code:    200,
		Message: "Success",
		Data:    update,
	}, nil
}

// GetFirmwareUpdateStatus implements v1.DeviceServiceServer
func (s *DeviceService) GetFirmwareUpdateStatus(ctx context.Context, req *v1.GetFirmwareUpdateStatusRequest) (*v1.GetFirmwareUpdateStatusResponse, error) {
	status, err := s.deviceUC.GetFirmwareUpdateStatus(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetFirmwareUpdateStatus error: %v", err)
		return &v1.GetFirmwareUpdateStatusResponse{
			Code:    500,
			Message: "Failed to get firmware update status",
		}, err
	}

	return &v1.GetFirmwareUpdateStatusResponse{
		Code:    200,
		Message: "Success",
		Data:    status,
	}, nil
}

// ListDeviceAlerts implements v1.DeviceServiceServer
func (s *DeviceService) ListDeviceAlerts(ctx context.Context, req *v1.ListDeviceAlertsRequest) (*v1.ListDeviceAlertsResponse, error) {
	alerts, total, err := s.deviceUC.ListDeviceAlerts(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListDeviceAlerts error: %v", err)
		return &v1.ListDeviceAlertsResponse{
			Code:    500,
			Message: "Failed to list device alerts",
		}, err
	}

	return &v1.ListDeviceAlertsResponse{
		Code:    200,
		Message: "Success",
		Data:    alerts,
		Total:   int32(total),
	}, nil
}

// GetDeviceAlert implements v1.DeviceServiceServer
func (s *DeviceService) GetDeviceAlert(ctx context.Context, req *v1.GetDeviceAlertRequest) (*v1.GetDeviceAlertResponse, error) {
	alert, err := s.deviceUC.GetDeviceAlert(ctx, req.AlertId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceAlert error: %v", err)
		return &v1.GetDeviceAlertResponse{
			Code:    500,
			Message: "Failed to get device alert",
		}, err
	}

	return &v1.GetDeviceAlertResponse{
		Code:    200,
		Message: "Success",
		Data:    alert,
	}, nil
}

// AcknowledgeAlert implements v1.DeviceServiceServer
func (s *DeviceService) AcknowledgeAlert(ctx context.Context, req *v1.AcknowledgeAlertRequest) (*v1.AcknowledgeAlertResponse, error) {
	err := s.deviceUC.AcknowledgeAlert(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("AcknowledgeAlert error: %v", err)
		return &v1.AcknowledgeAlertResponse{
			Code:    500,
			Message: "Failed to acknowledge alert",
		}, err
	}

	return &v1.AcknowledgeAlertResponse{
		Code:    200,
		Message: "Success",
	}, nil
}

// SendCommand implements v1.DeviceServiceServer
func (s *DeviceService) SendCommand(ctx context.Context, req *v1.SendCommandRequest) (*v1.SendCommandResponse, error) {
	command, err := s.deviceUC.SendCommand(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SendCommand error: %v", err)
		return &v1.SendCommandResponse{
			Code:    500,
			Message: "Failed to send command",
		}, err
	}

	return &v1.SendCommandResponse{
		Code:    200,
		Message: "Success",
		Data:    command,
	}, nil
}

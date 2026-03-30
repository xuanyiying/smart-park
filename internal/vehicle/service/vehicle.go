// Package service provides gRPC service implementation for the vehicle service.
package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
)

// VehicleService implements the VehicleService gRPC service.
type VehicleService struct {
	v1.UnimplementedVehicleServiceServer

	entryExitUseCase *biz.EntryExitUseCase
	deviceUseCase    *biz.DeviceUseCase
	vehicleUseCase   *biz.VehicleQueryUseCase
	commandUseCase   *biz.CommandUseCase
	recordUseCase    *biz.RecordQueryUseCase
	log              *log.Helper
}

// NewVehicleService creates a new VehicleService.
func NewVehicleService(
	entryExitUseCase *biz.EntryExitUseCase,
	deviceUseCase *biz.DeviceUseCase,
	vehicleUseCase *biz.VehicleQueryUseCase,
	commandUseCase *biz.CommandUseCase,
	recordUseCase *biz.RecordQueryUseCase,
	logger log.Logger,
) *VehicleService {
	return &VehicleService{
		entryExitUseCase: entryExitUseCase,
		deviceUseCase:    deviceUseCase,
		vehicleUseCase:   vehicleUseCase,
		commandUseCase:   commandUseCase,
		recordUseCase:    recordUseCase,
		log:              log.NewHelper(logger),
	}
}

// Entry handles vehicle entry request.
func (s *VehicleService) Entry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryResponse, error) {
	data, err := s.entryExitUseCase.Entry(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Entry failed: %v", err)
		return &v1.EntryResponse{
			Code:    500,
			Message: "入场失败",
		}, nil
	}

	return &v1.EntryResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// Exit handles vehicle exit request.
func (s *VehicleService) Exit(ctx context.Context, req *v1.ExitRequest) (*v1.ExitResponse, error) {
	data, err := s.entryExitUseCase.Exit(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Exit failed: %v", err)
		return &v1.ExitResponse{
			Code:    500,
			Message: "出场失败",
		}, nil
	}

	return &v1.ExitResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// Heartbeat handles device heartbeat request.
func (s *VehicleService) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) (*v1.HeartbeatResponse, error) {
	if err := s.deviceUseCase.Heartbeat(ctx, req); err != nil {
		s.log.WithContext(ctx).Errorf("Heartbeat failed: %v", err)
		return &v1.HeartbeatResponse{
			Code:    500,
			Message: "心跳失败",
		}, nil
	}

	return &v1.HeartbeatResponse{
		Code:    0,
		Message: "success",
	}, nil
}

// GetDeviceStatus handles get device status request.
func (s *VehicleService) GetDeviceStatus(ctx context.Context, req *v1.GetDeviceStatusRequest) (*v1.GetDeviceStatusResponse, error) {
	status, err := s.deviceUseCase.GetDeviceStatus(ctx, req.DeviceId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceStatus failed: %v", err)
		return &v1.GetDeviceStatusResponse{
			Code:    500,
			Message: "获取设备状态失败",
		}, nil
	}

	return &v1.GetDeviceStatusResponse{
		Code:    0,
		Message: "success",
		Data:    status,
	}, nil
}

// SendCommand handles send command request.
func (s *VehicleService) SendCommand(ctx context.Context, req *v1.SendCommandRequest) (*v1.SendCommandResponse, error) {
	data, err := s.commandUseCase.SendCommand(ctx, req.DeviceId, req.Command, req.Params)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SendCommand failed: %v", err)
		return &v1.SendCommandResponse{
			Code:    500,
			Message: "发送命令失败",
		}, nil
	}

	return &v1.SendCommandResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// GetVehicleInfo handles get vehicle info request.
func (s *VehicleService) GetVehicleInfo(ctx context.Context, req *v1.GetVehicleInfoRequest) (*v1.GetVehicleInfoResponse, error) {
	info, err := s.vehicleUseCase.GetVehicleInfo(ctx, req.PlateNumber)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetVehicleInfo failed: %v", err)
		return &v1.GetVehicleInfoResponse{
			Code:    500,
			Message: "获取车辆信息失败",
		}, nil
	}

	return &v1.GetVehicleInfoResponse{
		Code:    0,
		Message: "success",
		Data:    info,
	}, nil
}

// ListParkingRecords handles list parking records request.
func (s *VehicleService) ListParkingRecords(ctx context.Context, req *v1.ListParkingRecordsRequest) (*v1.ListParkingRecordsResponse, error) {
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	data, err := s.recordUseCase.ListParkingRecordsByPlates(ctx, req.PlateNumbers, page, pageSize)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListParkingRecords failed: %v", err)
		return &v1.ListParkingRecordsResponse{
			Code:    500,
			Message: "获取停车记录失败",
		}, nil
	}

	return &v1.ListParkingRecordsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// GetParkingRecord handles get parking record request.
func (s *VehicleService) GetParkingRecord(ctx context.Context, req *v1.GetParkingRecordRequest) (*v1.GetParkingRecordResponse, error) {
	info, err := s.recordUseCase.GetParkingRecord(ctx, req.RecordId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetParkingRecord failed: %v", err)
		return &v1.GetParkingRecordResponse{
			Code:    500,
			Message: "获取停车记录失败",
		}, nil
	}

	return &v1.GetParkingRecordResponse{
		Code:    0,
		Message: "success",
		Data:    info,
	}, nil
}

// ListDevices handles list devices request.
func (s *VehicleService) ListDevices(ctx context.Context, req *v1.ListDevicesRequest) (*v1.ListDevicesResponse, error) {
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	devices, total, err := s.deviceUseCase.ListDevices(ctx, page, pageSize)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListDevices failed: %v", err)
		return &v1.ListDevicesResponse{
			Code:    500,
			Message: "获取设备列表失败",
		}, nil
	}

	return &v1.ListDevicesResponse{
		Code:    0,
		Message: "success",
		Data:    devices,
		Total:   int32(total),
	}, nil
}

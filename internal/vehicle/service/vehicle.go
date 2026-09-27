// Package service provides gRPC service implementation for the vehicle service.
package service

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/types/known/structpb"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
)

// VehicleService implements the VehicleService gRPC service.
type VehicleService struct {
	v1.UnimplementedVehicleServiceServer

	entryExitUseCase         *biz.EntryExitUseCase
	deviceUseCase            *biz.DeviceUseCase
	manufacturerUseCase      *biz.ManufacturerUseCase
	firmwareUseCase          *biz.FirmwareUseCase
	devicePerformanceUseCase *biz.DevicePerformanceUseCase
	deviceFaultUseCase       *biz.DeviceFaultUseCase
	deviceStatsUseCase       *biz.DeviceStatsUseCase
	vehicleUseCase           *biz.VehicleQueryUseCase
	commandUseCase           *biz.CommandUseCase
	recordUseCase            *biz.RecordQueryUseCase
	blacklistUseCase         *biz.BlacklistUseCase
	log                      *log.Helper
}

// NewVehicleService creates a new VehicleService.
func NewVehicleService(
	entryExitUseCase *biz.EntryExitUseCase,
	deviceUseCase *biz.DeviceUseCase,
	manufacturerUseCase *biz.ManufacturerUseCase,
	firmwareUseCase *biz.FirmwareUseCase,
	devicePerformanceUseCase *biz.DevicePerformanceUseCase,
	deviceFaultUseCase *biz.DeviceFaultUseCase,
	deviceStatsUseCase *biz.DeviceStatsUseCase,
	vehicleUseCase *biz.VehicleQueryUseCase,
	commandUseCase *biz.CommandUseCase,
	recordUseCase *biz.RecordQueryUseCase,
	blacklistUseCase *biz.BlacklistUseCase,
	logger log.Logger,
) *VehicleService {
	return &VehicleService{
		entryExitUseCase:         entryExitUseCase,
		deviceUseCase:            deviceUseCase,
		manufacturerUseCase:      manufacturerUseCase,
		firmwareUseCase:          firmwareUseCase,
		devicePerformanceUseCase: devicePerformanceUseCase,
		deviceFaultUseCase:       deviceFaultUseCase,
		deviceStatsUseCase:       deviceStatsUseCase,
		vehicleUseCase:           vehicleUseCase,
		commandUseCase:           commandUseCase,
		recordUseCase:            recordUseCase,
		blacklistUseCase:         blacklistUseCase,
		log:                      log.NewHelper(logger),
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

// CreateDevice handles create device request.
func (s *VehicleService) CreateDevice(ctx context.Context, req *v1.CreateDeviceRequest) (*v1.CreateDeviceResponse, error) {
	device, err := s.deviceUseCase.CreateDevice(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateDevice failed: %v", err)
		return &v1.CreateDeviceResponse{
			Code:    500,
			Message: "创建设备失败: " + err.Error(),
		}, nil
	}

	return &v1.CreateDeviceResponse{
		Code:    0,
		Message: "success",
		Data:    device,
	}, nil
}

// GetDevice handles get device request.
func (s *VehicleService) GetDevice(ctx context.Context, req *v1.GetDeviceRequest) (*v1.GetDeviceResponse, error) {
	device, err := s.deviceUseCase.GetDevice(ctx, req.DeviceId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDevice failed: %v", err)
		return &v1.GetDeviceResponse{
			Code:    500,
			Message: "获取设备失败: " + err.Error(),
		}, nil
	}

	return &v1.GetDeviceResponse{
		Code:    0,
		Message: "success",
		Data:    device,
	}, nil
}

// UpdateDevice handles update device request.
func (s *VehicleService) UpdateDevice(ctx context.Context, req *v1.UpdateDeviceRequest) (*v1.UpdateDeviceResponse, error) {
	device, err := s.deviceUseCase.UpdateDevice(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpdateDevice failed: %v", err)
		return &v1.UpdateDeviceResponse{
			Code:    500,
			Message: "更新设备失败: " + err.Error(),
		}, nil
	}

	return &v1.UpdateDeviceResponse{
		Code:    0,
		Message: "success",
		Data:    device,
	}, nil
}

// DeleteDevice handles delete device request.
func (s *VehicleService) DeleteDevice(ctx context.Context, req *v1.DeleteDeviceRequest) (*v1.DeleteDeviceResponse, error) {
	if err := s.deviceUseCase.DeleteDevice(ctx, req.DeviceId); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteDevice failed: %v", err)
		return &v1.DeleteDeviceResponse{
			Code:    500,
			Message: "删除设备失败: " + err.Error(),
		}, nil
	}

	return &v1.DeleteDeviceResponse{
		Code:    0,
		Message: "success",
	}, nil
}

// CreateManufacturer handles create manufacturer request.
func (s *VehicleService) CreateManufacturer(ctx context.Context, req *v1.CreateManufacturerRequest) (*v1.CreateManufacturerResponse, error) {
	manufacturer, err := s.manufacturerUseCase.CreateManufacturer(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateManufacturer failed: %v", err)
		return &v1.CreateManufacturerResponse{
			Code:    500,
			Message: "创建厂商失败: " + err.Error(),
		}, nil
	}

	return &v1.CreateManufacturerResponse{
		Code:    0,
		Message: "success",
		Data:    manufacturer,
	}, nil
}

// GetManufacturer handles get manufacturer request.
func (s *VehicleService) GetManufacturer(ctx context.Context, req *v1.GetManufacturerRequest) (*v1.GetManufacturerResponse, error) {
	manufacturer, err := s.manufacturerUseCase.GetManufacturer(ctx, req.Id)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetManufacturer failed: %v", err)
		return &v1.GetManufacturerResponse{
			Code:    500,
			Message: "获取厂商失败: " + err.Error(),
		}, nil
	}

	return &v1.GetManufacturerResponse{
		Code:    0,
		Message: "success",
		Data:    manufacturer,
	}, nil
}

// ListManufacturers handles list manufacturers request.
func (s *VehicleService) ListManufacturers(ctx context.Context, req *v1.ListManufacturersRequest) (*v1.ListManufacturersResponse, error) {
	page := int(req.Page)
	if page <= 0 {
		page = 1
	}
	pageSize := int(req.PageSize)
	if pageSize <= 0 {
		pageSize = 10
	}

	manufacturers, total, err := s.manufacturerUseCase.ListManufacturers(ctx, page, pageSize)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListManufacturers failed: %v", err)
		return &v1.ListManufacturersResponse{
			Code:    500,
			Message: "获取厂商列表失败",
		}, nil
	}

	return &v1.ListManufacturersResponse{
		Code:    0,
		Message: "success",
		Data:    manufacturers,
		Total:   int32(total),
	}, nil
}

// UpdateManufacturer handles update manufacturer request.
func (s *VehicleService) UpdateManufacturer(ctx context.Context, req *v1.UpdateManufacturerRequest) (*v1.UpdateManufacturerResponse, error) {
	manufacturer, err := s.manufacturerUseCase.UpdateManufacturer(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpdateManufacturer failed: %v", err)
		return &v1.UpdateManufacturerResponse{
			Code:    500,
			Message: "更新厂商失败: " + err.Error(),
		}, nil
	}

	return &v1.UpdateManufacturerResponse{
		Code:    0,
		Message: "success",
		Data:    manufacturer,
	}, nil
}

// DeleteManufacturer handles delete manufacturer request.
func (s *VehicleService) DeleteManufacturer(ctx context.Context, req *v1.DeleteManufacturerRequest) (*v1.DeleteManufacturerResponse, error) {
	if err := s.manufacturerUseCase.DeleteManufacturer(ctx, req.Id); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteManufacturer failed: %v", err)
		return &v1.DeleteManufacturerResponse{
			Code:    500,
			Message: "删除厂商失败: " + err.Error(),
		}, nil
	}

	return &v1.DeleteManufacturerResponse{
		Code:    0,
		Message: "success",
	}, nil
}

// CreateFirmware handles create firmware request.
func (s *VehicleService) CreateFirmware(ctx context.Context, req *v1.CreateFirmwareRequest) (*v1.CreateFirmwareResponse, error) {
	firmware, err := s.firmwareUseCase.CreateFirmware(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateFirmware failed: %v", err)
		return &v1.CreateFirmwareResponse{Code: 500, Message: "创建固件失败: " + err.Error()}, nil
	}
	return &v1.CreateFirmwareResponse{Code: 0, Message: "success", Data: firmware}, nil
}

// GetFirmware handles get firmware request.
func (s *VehicleService) GetFirmware(ctx context.Context, req *v1.GetFirmwareRequest) (*v1.GetFirmwareResponse, error) {
	firmware, err := s.firmwareUseCase.GetFirmware(ctx, req.Id)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetFirmware failed: %v", err)
		return &v1.GetFirmwareResponse{Code: 500, Message: "获取固件失败: " + err.Error()}, nil
	}
	return &v1.GetFirmwareResponse{Code: 0, Message: "success", Data: firmware}, nil
}

// GetFirmwareByID handles get firmware by ID request.
func (s *VehicleService) GetFirmwareByID(ctx context.Context, req *v1.GetFirmwareByIDRequest) (*v1.GetFirmwareByIDResponse, error) {
	firmware, err := s.firmwareUseCase.GetFirmwareByID(ctx, req.FirmwareId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetFirmwareByID failed: %v", err)
		return &v1.GetFirmwareByIDResponse{Code: 500, Message: "获取固件失败: " + err.Error()}, nil
	}
	return &v1.GetFirmwareByIDResponse{Code: 0, Message: "success", Data: firmware}, nil
}

// ListFirmwares handles list firmwares request.
func (s *VehicleService) ListFirmwares(ctx context.Context, req *v1.ListFirmwaresRequest) (*v1.ListFirmwaresResponse, error) {
	firmwares, total, err := s.firmwareUseCase.ListFirmwares(ctx, req.Manufacturer, req.Model, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListFirmwares failed: %v", err)
		return &v1.ListFirmwaresResponse{Code: 500, Message: "获取固件列表失败"}, nil
	}
	return &v1.ListFirmwaresResponse{Code: 0, Message: "success", Data: firmwares, Total: int32(total)}, nil
}

// UpdateFirmware handles update firmware request.
func (s *VehicleService) UpdateFirmware(ctx context.Context, req *v1.UpdateFirmwareRequest) (*v1.UpdateFirmwareResponse, error) {
	firmware, err := s.firmwareUseCase.UpdateFirmware(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpdateFirmware failed: %v", err)
		return &v1.UpdateFirmwareResponse{Code: 500, Message: "更新固件失败: " + err.Error()}, nil
	}
	return &v1.UpdateFirmwareResponse{Code: 0, Message: "success", Data: firmware}, nil
}

// DeleteFirmware handles delete firmware request.
func (s *VehicleService) DeleteFirmware(ctx context.Context, req *v1.DeleteFirmwareRequest) (*v1.DeleteFirmwareResponse, error) {
	if err := s.firmwareUseCase.DeleteFirmware(ctx, req.Id); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteFirmware failed: %v", err)
		return &v1.DeleteFirmwareResponse{Code: 500, Message: "删除固件失败: " + err.Error()}, nil
	}
	return &v1.DeleteFirmwareResponse{Code: 0, Message: "success"}, nil
}

// GetLatestFirmware handles get latest firmware request.
func (s *VehicleService) GetLatestFirmware(ctx context.Context, req *v1.GetLatestFirmwareRequest) (*v1.GetLatestFirmwareResponse, error) {
	firmware, err := s.firmwareUseCase.GetLatestFirmware(ctx, req.Manufacturer, req.Model)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetLatestFirmware failed: %v", err)
		return &v1.GetLatestFirmwareResponse{Code: 500, Message: "获取最新固件失败: " + err.Error()}, nil
	}
	return &v1.GetLatestFirmwareResponse{Code: 0, Message: "success", Data: firmware}, nil
}

// CreateDevicePerformance handles create device performance request.
func (s *VehicleService) CreateDevicePerformance(ctx context.Context, req *v1.CreateDevicePerformanceRequest) (*v1.CreateDevicePerformanceResponse, error) {
	if err := s.devicePerformanceUseCase.CreateDevicePerformance(ctx, req); err != nil {
		s.log.WithContext(ctx).Errorf("CreateDevicePerformance failed: %v", err)
		return &v1.CreateDevicePerformanceResponse{Code: 500, Message: "创建设备性能数据失败: " + err.Error()}, nil
	}
	return &v1.CreateDevicePerformanceResponse{Code: 0, Message: "success"}, nil
}

// GetDevicePerformance handles get device performance request.
func (s *VehicleService) GetDevicePerformance(ctx context.Context, req *v1.GetDevicePerformanceRequest) (*v1.GetDevicePerformanceResponse, error) {
	records, err := s.devicePerformanceUseCase.GetDevicePerformance(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDevicePerformance failed: %v", err)
		return &v1.GetDevicePerformanceResponse{Code: 500, Message: "获取设备性能数据失败"}, nil
	}
	return &v1.GetDevicePerformanceResponse{Code: 0, Message: "success", Data: records}, nil
}

// GetDevicePerformanceLatest handles get latest device performance request.
func (s *VehicleService) GetDevicePerformanceLatest(ctx context.Context, req *v1.GetDevicePerformanceLatestRequest) (*v1.GetDevicePerformanceLatestResponse, error) {
	record, err := s.devicePerformanceUseCase.GetDevicePerformanceLatest(ctx, req.DeviceId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDevicePerformanceLatest failed: %v", err)
		return &v1.GetDevicePerformanceLatestResponse{Code: 500, Message: "获取最新设备性能数据失败"}, nil
	}
	return &v1.GetDevicePerformanceLatestResponse{Code: 0, Message: "success", Data: record}, nil
}

// CreateDeviceFault handles create device fault request.
func (s *VehicleService) CreateDeviceFault(ctx context.Context, req *v1.CreateDeviceFaultRequest) (*v1.CreateDeviceFaultResponse, error) {
	fault, err := s.deviceFaultUseCase.CreateDeviceFault(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateDeviceFault failed: %v", err)
		return &v1.CreateDeviceFaultResponse{Code: 500, Message: "创建设备故障记录失败: " + err.Error()}, nil
	}
	return &v1.CreateDeviceFaultResponse{Code: 0, Message: "success", Data: fault}, nil
}

// GetDeviceFault handles get device fault request.
func (s *VehicleService) GetDeviceFault(ctx context.Context, req *v1.GetDeviceFaultRequest) (*v1.GetDeviceFaultResponse, error) {
	fault, err := s.deviceFaultUseCase.GetDeviceFault(ctx, req.Id)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceFault failed: %v", err)
		return &v1.GetDeviceFaultResponse{Code: 500, Message: "获取设备故障记录失败: " + err.Error()}, nil
	}
	return &v1.GetDeviceFaultResponse{Code: 0, Message: "success", Data: fault}, nil
}

// ListDeviceFaults handles list device faults request.
func (s *VehicleService) ListDeviceFaults(ctx context.Context, req *v1.ListDeviceFaultsRequest) (*v1.ListDeviceFaultsResponse, error) {
	faults, total, err := s.deviceFaultUseCase.ListDeviceFaults(ctx, req.DeviceId, req.Status, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListDeviceFaults failed: %v", err)
		return &v1.ListDeviceFaultsResponse{Code: 500, Message: "获取设备故障列表失败"}, nil
	}
	return &v1.ListDeviceFaultsResponse{Code: 0, Message: "success", Data: faults, Total: int32(total)}, nil
}

// ResolveDeviceFault handles resolve device fault request.
func (s *VehicleService) ResolveDeviceFault(ctx context.Context, req *v1.ResolveDeviceFaultRequest) (*v1.ResolveDeviceFaultResponse, error) {
	if err := s.deviceFaultUseCase.ResolveDeviceFault(ctx, req.Id); err != nil {
		s.log.WithContext(ctx).Errorf("ResolveDeviceFault failed: %v", err)
		return &v1.ResolveDeviceFaultResponse{Code: 500, Message: "解决设备故障失败: " + err.Error()}, nil
	}
	return &v1.ResolveDeviceFaultResponse{Code: 0, Message: "success"}, nil
}

// GetDeviceUsageStats handles get device usage stats request.
func (s *VehicleService) GetDeviceUsageStats(ctx context.Context, req *v1.GetDeviceUsageStatsRequest) (*v1.GetDeviceUsageStatsResponse, error) {
	startTime, _ := time.Parse(time.RFC3339, req.StartTime)
	endTime, _ := time.Parse(time.RFC3339, req.EndTime)
	stats, err := s.deviceStatsUseCase.GetDeviceUsageStats(ctx, req.DeviceId, startTime, endTime)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceUsageStats failed: %v", err)
		return &v1.GetDeviceUsageStatsResponse{Code: 500, Message: "获取设备使用统计失败"}, nil
	}
	data := mapToStructPB(stats)
	return &v1.GetDeviceUsageStatsResponse{Code: 0, Message: "success", Data: data}, nil
}

// GetDeviceFaultStats handles get device fault stats request.
func (s *VehicleService) GetDeviceFaultStats(ctx context.Context, req *v1.GetDeviceFaultStatsRequest) (*v1.GetDeviceFaultStatsResponse, error) {
	startTime, _ := time.Parse(time.RFC3339, req.StartTime)
	endTime, _ := time.Parse(time.RFC3339, req.EndTime)
	stats, err := s.deviceStatsUseCase.GetDeviceFaultStats(ctx, req.DeviceId, startTime, endTime)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceFaultStats failed: %v", err)
		return &v1.GetDeviceFaultStatsResponse{Code: 500, Message: "获取设备故障统计失败"}, nil
	}
	data := mapToStructPB(stats)
	return &v1.GetDeviceFaultStatsResponse{Code: 0, Message: "success", Data: data}, nil
}

// GetDeviceStatsSummary handles get device stats summary request.
func (s *VehicleService) GetDeviceStatsSummary(ctx context.Context, req *v1.GetDeviceStatsSummaryRequest) (*v1.GetDeviceStatsSummaryResponse, error) {
	stats, err := s.deviceStatsUseCase.GetDeviceStatsSummary(ctx, req.DeviceId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceStatsSummary failed: %v", err)
		return &v1.GetDeviceStatsSummaryResponse{Code: 500, Message: "获取设备统计摘要失败"}, nil
	}
	data := mapToStructPB(stats)
	return &v1.GetDeviceStatsSummaryResponse{Code: 0, Message: "success", Data: data}, nil
}

// UpgradeDevice handles upgrade device request.
func (s *VehicleService) UpgradeDevice(ctx context.Context, req *v1.UpgradeDeviceRequest) (*v1.UpgradeDeviceResponse, error) {
	resp, err := s.deviceUseCase.UpgradeDevice(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpgradeDevice failed: %v", err)
		return &v1.UpgradeDeviceResponse{Code: 500, Message: "设备升级失败: " + err.Error()}, nil
	}
	return resp, nil
}

// GetDeviceUpgradeStatus handles get device upgrade status request.
func (s *VehicleService) GetDeviceUpgradeStatus(ctx context.Context, req *v1.GetDeviceUpgradeStatusRequest) (*v1.UpgradeStatusResponse, error) {
	resp, err := s.deviceUseCase.GetDeviceUpgradeStatus(ctx, req.UpgradeId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceUpgradeStatus failed: %v", err)
		return nil, err
	}
	return resp, nil
}

// GetDeviceLogs handles get device logs request.
func (s *VehicleService) GetDeviceLogs(ctx context.Context, req *v1.GetDeviceLogsRequest) (*v1.GetDeviceLogsResponse, error) {
	logs, total, err := s.deviceUseCase.GetDeviceLogs(ctx, req.DeviceId, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceLogs failed: %v", err)
		return &v1.GetDeviceLogsResponse{Code: 500, Message: "获取设备日志失败"}, nil
	}
	return &v1.GetDeviceLogsResponse{Code: 0, Message: "success", Data: logs, Total: int32(total)}, nil
}

// GetDeviceStats handles get device stats request.
func (s *VehicleService) GetDeviceStats(ctx context.Context, req *v1.GetDeviceStatsRequest) (*v1.DeviceStatsResponse, error) {
	// Get device stats summary from repo
	stats, err := s.deviceStatsUseCase.GetDeviceStatsSummary(ctx, req.DeviceId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDeviceStats failed: %v", err)
		return &v1.DeviceStatsResponse{Code: 500, Message: "获取设备统计失败"}, nil
	}
	totalDevices, _ := stats["total_devices"].(int32)
	onlineDevices, _ := stats["online_devices"].(int32)
	faultDevices, _ := stats["fault_devices"].(int32)
	onlineRate, _ := stats["online_rate"].(float64)
	return &v1.DeviceStatsResponse{
		Code:    0,
		Message: "success",
		Data: &v1.DeviceStatsData{
			TotalDevices:  totalDevices,
			OnlineDevices: onlineDevices,
			FaultDevices:  faultDevices,
			OnlineRate:    onlineRate,
		},
	}, nil
}

// UpdateDeviceConfig handles update device config request.
func (s *VehicleService) UpdateDeviceConfig(ctx context.Context, req *v1.UpdateDeviceConfigRequest) (*v1.UpdateDeviceConfigResponse, error) {
	_, err := s.deviceUseCase.UpdateDeviceConfig(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpdateDeviceConfig failed: %v", err)
		return &v1.UpdateDeviceConfigResponse{Code: 500, Message: "更新设备配置失败: " + err.Error()}, nil
	}
	return &v1.UpdateDeviceConfigResponse{Code: 0, Message: "success"}, nil
}

// mapToStructPB converts map[string]interface{} to map[string]*structpb.Value.
func mapToStructPB(m map[string]interface{}) map[string]*structpb.Value {
	result := make(map[string]*structpb.Value, len(m))
	for k, v := range m {
		sv, err := structpb.NewValue(v)
		if err != nil {
			sv, _ = structpb.NewValue(0)
		}
		result[k] = sv
	}
	return result
}

// ---- Blacklist management ----

func toBlacklistEntryInfo(entry *biz.BlacklistEntry) *v1.BlacklistEntryInfo {
	if entry == nil {
		return nil
	}
	return &v1.BlacklistEntryInfo{
		PlateNumber: entry.PlateNumber,
		Reason:      entry.Reason,
		CreatedBy:   entry.CreatedBy,
		Active:      entry.Active,
		CreatedAt:   entry.CreatedAt.Format(time.RFC3339),
	}
}

// AddBlacklistEntry adds (or re-activates) a blacklist record.
func (s *VehicleService) AddBlacklistEntry(ctx context.Context, req *v1.AddBlacklistEntryRequest) (*v1.AddBlacklistEntryResponse, error) {
	entry, err := s.blacklistUseCase.AddBlacklistEntry(ctx, req.PlateNumber, req.Reason, req.CreatedBy)
	if err != nil {
		s.log.WithContext(ctx).Errorf("AddBlacklistEntry failed: %v", err)
		return &v1.AddBlacklistEntryResponse{Code: 500, Message: "加入黑名单失败: " + err.Error()}, nil
	}
	return &v1.AddBlacklistEntryResponse{
		Code:    0,
		Message: "success",
		Data:    toBlacklistEntryInfo(entry),
	}, nil
}

// RemoveBlacklistEntry deactivates a blacklist record.
func (s *VehicleService) RemoveBlacklistEntry(ctx context.Context, req *v1.RemoveBlacklistEntryRequest) (*v1.RemoveBlacklistEntryResponse, error) {
	if err := s.blacklistUseCase.RemoveBlacklistEntry(ctx, req.PlateNumber); err != nil {
		s.log.WithContext(ctx).Errorf("RemoveBlacklistEntry failed: %v", err)
		return &v1.RemoveBlacklistEntryResponse{Code: 500, Message: "移出黑名单失败: " + err.Error()}, nil
	}
	return &v1.RemoveBlacklistEntryResponse{Code: 0, Message: "success"}, nil
}

// ListBlacklistEntries lists blacklist records with pagination.
func (s *VehicleService) ListBlacklistEntries(ctx context.Context, req *v1.ListBlacklistEntriesRequest) (*v1.ListBlacklistEntriesResponse, error) {
	entries, total, err := s.blacklistUseCase.ListBlacklistEntries(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListBlacklistEntries failed: %v", err)
		return &v1.ListBlacklistEntriesResponse{Code: 500, Message: "查询黑名单失败: " + err.Error()}, nil
	}

	items := make([]*v1.BlacklistEntryInfo, 0, len(entries))
	for _, entry := range entries {
		items = append(items, toBlacklistEntryInfo(entry))
	}

	return &v1.ListBlacklistEntriesResponse{
		Code:    0,
		Message: "success",
		Entries: items,
		Total:   int32(total),
	}, nil
}

// CheckBlacklist reports whether a plate is currently blacklisted.
func (s *VehicleService) CheckBlacklist(ctx context.Context, req *v1.CheckBlacklistRequest) (*v1.CheckBlacklistResponse, error) {
	entry, err := s.blacklistUseCase.CheckBlacklist(ctx, req.PlateNumber)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CheckBlacklist failed: %v", err)
		return &v1.CheckBlacklistResponse{Code: 500, Message: "黑名单检查失败: " + err.Error()}, nil
	}
	if entry == nil {
		return &v1.CheckBlacklistResponse{Code: 0, Message: "success", Blacklisted: false}, nil
	}
	return &v1.CheckBlacklistResponse{
		Code:        0,
		Message:     "success",
		Blacklisted: true,
		Entry:       toBlacklistEntryInfo(entry),
	}, nil
}

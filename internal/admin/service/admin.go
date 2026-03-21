// Package service provides gRPC service implementation for the admin service.
package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/xuanyiying/smart-park/api/admin/v1"
	"github.com/xuanyiying/smart-park/internal/admin/biz"
)

// AdminService implements the AdminService gRPC service.
type AdminService struct {
	v1.UnimplementedAdminServiceServer

	uc  *biz.AdminUseCase
	log *log.Helper
}

// NewAdminService creates a new AdminService.
func NewAdminService(uc *biz.AdminUseCase, logger log.Logger) *AdminService {
	return &AdminService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

// CreateParkingLot handles create parking lot request.
func (s *AdminService) CreateParkingLot(ctx context.Context, req *v1.CreateParkingLotRequest) (*v1.CreateParkingLotResponse, error) {
	lot, err := s.uc.CreateParkingLot(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateParkingLot failed: %v", err)
		return &v1.CreateParkingLotResponse{
			Code:    500,
			Message: "创建停车场失败",
		}, nil
	}

	return &v1.CreateParkingLotResponse{
		Code:    0,
		Message: "success",
		Data:    lot,
	}, nil
}

// GetParkingLot handles get parking lot request.
func (s *AdminService) GetParkingLot(ctx context.Context, req *v1.GetParkingLotRequest) (*v1.GetParkingLotResponse, error) {
	lot, err := s.uc.GetParkingLot(ctx, req.Id)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetParkingLot failed: %v", err)
		return &v1.GetParkingLotResponse{
			Code:    500,
			Message: "获取停车场失败",
		}, nil
	}

	return &v1.GetParkingLotResponse{
		Code:    0,
		Message: "success",
		Data:    lot,
	}, nil
}

// UpdateParkingLot handles update parking lot request.
func (s *AdminService) UpdateParkingLot(ctx context.Context, req *v1.UpdateParkingLotRequest) (*v1.UpdateParkingLotResponse, error) {
	if err := s.uc.UpdateParkingLot(ctx, req); err != nil {
		s.log.WithContext(ctx).Errorf("UpdateParkingLot failed: %v", err)
		return &v1.UpdateParkingLotResponse{
			Code:    500,
			Message: "更新停车场失败",
		}, nil
	}

	return &v1.UpdateParkingLotResponse{
		Code:    0,
		Message: "success",
	}, nil
}

// ListParkingLots handles list parking lots request.
func (s *AdminService) ListParkingLots(ctx context.Context, req *v1.ListParkingLotsRequest) (*v1.ListParkingLotsResponse, error) {
	data, err := s.uc.ListParkingLots(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListParkingLots failed: %v", err)
		return &v1.ListParkingLotsResponse{
			Code:    500,
			Message: "获取停车场列表失败",
		}, nil
	}

	return &v1.ListParkingLotsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// ListParkingRecords handles list parking records request.
func (s *AdminService) ListParkingRecords(ctx context.Context, req *v1.ListParkingRecordsRequest) (*v1.ListParkingRecordsResponse, error) {
	data, err := s.uc.ListParkingRecords(ctx, req)
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

// ListOrders handles list orders request.
func (s *AdminService) ListOrders(ctx context.Context, req *v1.ListOrdersRequest) (*v1.ListOrdersResponse, error) {
	data, err := s.uc.ListOrders(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListOrders failed: %v", err)
		return &v1.ListOrdersResponse{
			Code:    500,
			Message: "获取订单列表失败",
		}, nil
	}

	return &v1.ListOrdersResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// GetOrder handles get order request.
func (s *AdminService) GetOrder(ctx context.Context, req *v1.GetOrderRequest) (*v1.GetOrderResponse, error) {
	order, err := s.uc.GetOrder(ctx, req.Id)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetOrder failed: %v", err)
		return &v1.GetOrderResponse{
			Code:    500,
			Message: "获取订单失败",
		}, nil
	}

	return &v1.GetOrderResponse{
		Code:    0,
		Message: "success",
		Data:    order,
	}, nil
}

// CreateVehicle handles create vehicle request.
func (s *AdminService) CreateVehicle(ctx context.Context, req *v1.CreateVehicleRequest) (*v1.CreateVehicleResponse, error) {
	vehicle, err := s.uc.CreateVehicle(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateVehicle failed: %v", err)
		return &v1.CreateVehicleResponse{
			Code:    500,
			Message: "创建车辆失败",
		}, nil
	}

	return &v1.CreateVehicleResponse{
		Code:    0,
		Message: "success",
		Data:    vehicle,
	}, nil
}

// ListVehicles handles list vehicles request.
func (s *AdminService) ListVehicles(ctx context.Context, req *v1.ListVehiclesRequest) (*v1.ListVehiclesResponse, error) {
	data, err := s.uc.ListVehicles(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListVehicles failed: %v", err)
		return &v1.ListVehiclesResponse{
			Code:    500,
			Message: "获取车辆列表失败",
		}, nil
	}

	return &v1.ListVehiclesResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// GetDailyReport handles get daily report request.
func (s *AdminService) GetDailyReport(ctx context.Context, req *v1.GetDailyReportRequest) (*v1.GetDailyReportResponse, error) {
	report, err := s.uc.GetDailyReport(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetDailyReport failed: %v", err)
		return &v1.GetDailyReportResponse{
			Code:    500,
			Message: "获取日报表失败",
		}, nil
	}

	return &v1.GetDailyReportResponse{
		Code:    0,
		Message: "success",
		Data:    report,
	}, nil
}

// GetMonthlyReport handles get monthly report request.
func (s *AdminService) GetMonthlyReport(ctx context.Context, req *v1.GetMonthlyReportRequest) (*v1.GetMonthlyReportResponse, error) {
	report, err := s.uc.GetMonthlyReport(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetMonthlyReport failed: %v", err)
		return &v1.GetMonthlyReportResponse{
			Code:    500,
			Message: "获取月报表失败",
		}, nil
	}

	return &v1.GetMonthlyReportResponse{
		Code:    0,
		Message: "success",
		Data:    report,
	}, nil
}

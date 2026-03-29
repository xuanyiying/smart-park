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

// Login handles admin login request.
func (s *AdminService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	user, token, expiresAt, err := s.uc.Login(ctx, req.Username, req.Password)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Login failed: %v", err)
		return &v1.LoginResponse{
			Code:    401,
			Message: "用户名或密码错误",
		}, nil
	}

	return &v1.LoginResponse{
		Code:    0,
		Message: "success",
		Data: &v1.LoginData{
			Token:     token,
			ExpiresAt: expiresAt,
			User: &v1.User{
				Id:       user.ID.String(),
				Username: user.Username,
				Name:     user.Name,
				Role:     user.Role,
				Avatar:   user.Avatar,
			},
		},
	}, nil
}

// GetCurrentUser handles get current user request.
func (s *AdminService) GetCurrentUser(ctx context.Context, req *v1.GetCurrentUserRequest) (*v1.GetCurrentUserResponse, error) {
	user, err := s.uc.GetCurrentUser(ctx)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetCurrentUser failed: %v", err)
		return &v1.GetCurrentUserResponse{
			Code:    401,
			Message: "未登录或登录已过期",
		}, nil
	}

	return &v1.GetCurrentUserResponse{
		Code:    0,
		Message: "success",
		Data: &v1.User{
			Id:       user.ID.String(),
			Username: user.Username,
			Name:     user.Name,
			Role:     user.Role,
			Avatar:   user.Avatar,
		},
	}, nil
}

// ListUsers handles list users request.
func (s *AdminService) ListUsers(ctx context.Context, req *v1.ListUsersRequest) (*v1.ListUsersResponse, error) {
	users, total, err := s.uc.ListUsers(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListUsers failed: %v", err)
		return &v1.ListUsersResponse{
			Code:    500,
			Message: "获取用户列表失败",
		}, nil
	}

	var list []*v1.User
	for _, u := range users {
		list = append(list, &v1.User{
			Id:       u.ID.String(),
			Username: u.Username,
			Name:     u.Name,
			Role:     u.Role,
			Avatar:   u.Avatar,
		})
	}

	return &v1.ListUsersResponse{
		Code:    0,
		Message: "success",
		Data: &v1.UserListData{
			List:     list,
			Total:    int32(total),
			Page:     req.Page,
			PageSize: req.PageSize,
		},
	}, nil
}

// CreateUser handles create user request.
func (s *AdminService) CreateUser(ctx context.Context, req *v1.CreateUserRequest) (*v1.CreateUserResponse, error) {
	user, err := s.uc.CreateUser(ctx, req.Username, req.Password, req.Name, req.Role, req.Email, req.Status)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreateUser failed: %v", err)
		return &v1.CreateUserResponse{
			Code:    500,
			Message: "创建用户失败",
		}, nil
	}

	return &v1.CreateUserResponse{
		Code:    0,
		Message: "创建成功",
		Data: &v1.User{
			Id:       user.ID.String(),
			Username: user.Username,
			Name:     user.Name,
			Role:     user.Role,
			Avatar:   user.Avatar,
		},
	}, nil
}

// UpdateUser handles update user request.
func (s *AdminService) UpdateUser(ctx context.Context, req *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
	user, err := s.uc.UpdateUser(ctx, req.Id, req.Username, req.Name, req.Role, req.Email, req.Status)
	if err != nil {
		s.log.WithContext(ctx).Errorf("UpdateUser failed: %v", err)
		return &v1.UpdateUserResponse{
			Code:    500,
			Message: "更新用户失败",
		}, nil
	}

	return &v1.UpdateUserResponse{
		Code:    0,
		Message: "更新成功",
		Data: &v1.User{
			Id:       user.ID.String(),
			Username: user.Username,
			Name:     user.Name,
			Role:     user.Role,
			Avatar:   user.Avatar,
		},
	}, nil
}

// DeleteUser handles delete user request.
func (s *AdminService) DeleteUser(ctx context.Context, req *v1.DeleteUserRequest) (*v1.DeleteUserResponse, error) {
	if err := s.uc.DeleteUser(ctx, req.Id); err != nil {
		s.log.WithContext(ctx).Errorf("DeleteUser failed: %v", err)
		return &v1.DeleteUserResponse{
			Code:    500,
			Message: "删除用户失败",
		}, nil
	}

	return &v1.DeleteUserResponse{
		Code:    0,
		Message: "删除成功",
	}, nil
}

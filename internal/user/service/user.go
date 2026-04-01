package service

import (
	"context"

	v1 "github.com/xuanyiying/smart-park/api/user/v1"
	"github.com/xuanyiying/smart-park/internal/user/biz"
	"github.com/xuanyiying/smart-park/internal/user/middleware"
)

type UserService struct {
	v1.UnimplementedUserServiceServer
	uc *biz.UserUseCase
}

func NewUserService(uc *biz.UserUseCase) *UserService {
	return &UserService{uc: uc}
}

func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	data, err := s.uc.Login(ctx, req)
	if err != nil {
		return &v1.LoginResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.LoginResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}, nil
}

func (s *UserService) GetUserInfo(ctx context.Context, req *v1.GetUserInfoRequest) (*v1.GetUserInfoResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)

	userInfo, err := s.uc.GetUserInfo(ctx, userID)
	if err != nil {
		return &v1.GetUserInfoResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.GetUserInfoResponse{
		Code:    200,
		Message: "success",
		Data:    userInfo,
	}, nil
}

func (s *UserService) UpdateAutoPaySettings(ctx context.Context, req *v1.UpdateAutoPaySettingsRequest) (*v1.UpdateAutoPaySettingsResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)

	err := s.uc.UpdateAutoPaySettings(ctx, userID, req)
	if err != nil {
		return &v1.UpdateAutoPaySettingsResponse{
			Code:    500,
			Message: err.Error(),
			Success: false,
		}, nil
	}

	return &v1.UpdateAutoPaySettingsResponse{
		Code:    200,
		Message: "success",
		Success: true,
	}, nil
}

func (s *UserService) GetCreditInfo(ctx context.Context, req *v1.GetCreditInfoRequest) (*v1.GetCreditInfoResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)

	creditInfo, err := s.uc.GetCreditInfo(ctx, userID)
	if err != nil {
		return &v1.GetCreditInfoResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.GetCreditInfoResponse{
		Code:    200,
		Message: "success",
		Data:    creditInfo,
	}, nil
}

func (s *UserService) BindPlate(ctx context.Context, req *v1.BindPlateRequest) (*v1.BindPlateResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)
	err := s.uc.BindPlate(ctx, userID, req)
	if err != nil {
		return &v1.BindPlateResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.BindPlateResponse{
		Code:    200,
		Message: "success",
	}, nil
}

func (s *UserService) UnbindPlate(ctx context.Context, req *v1.UnbindPlateRequest) (*v1.UnbindPlateResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)
	err := s.uc.UnbindPlate(ctx, userID, req.PlateNumber)
	if err != nil {
		return &v1.UnbindPlateResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.UnbindPlateResponse{
		Code:    200,
		Message: "success",
	}, nil
}

func (s *UserService) ListPlates(ctx context.Context, req *v1.ListPlatesRequest) (*v1.ListPlatesResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)
	data, err := s.uc.ListPlates(ctx, userID, int(req.Page), int(req.PageSize))
	if err != nil {
		return &v1.ListPlatesResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.ListPlatesResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}, nil
}

func (s *UserService) ListParkingRecords(ctx context.Context, req *v1.ListParkingRecordsRequest) (*v1.ListParkingRecordsResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)

	page := req.Page
	if page <= 0 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 10
	}

	data, err := s.uc.ListParkingRecords(ctx, userID, page, pageSize)
	if err != nil {
		return &v1.ListParkingRecordsResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.ListParkingRecordsResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}, nil
}

func (s *UserService) GetParkingRecord(ctx context.Context, req *v1.GetParkingRecordRequest) (*v1.GetParkingRecordResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)

	data, err := s.uc.GetParkingRecord(ctx, userID, req.RecordId)
	if err != nil {
		return &v1.GetParkingRecordResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.GetParkingRecordResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}, nil
}

func (s *UserService) ScanPay(ctx context.Context, req *v1.ScanPayRequest) (*v1.ScanPayResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)

	data, err := s.uc.ScanPay(ctx, userID, req)
	if err != nil {
		return &v1.ScanPayResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.ScanPayResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}, nil
}

func (s *UserService) GetMonthlyCard(ctx context.Context, req *v1.GetMonthlyCardRequest) (*v1.GetMonthlyCardResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)
	data, err := s.uc.GetMonthlyCard(ctx, userID, req.PlateNumber)
	if err != nil {
		return &v1.GetMonthlyCardResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.GetMonthlyCardResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}, nil
}

func (s *UserService) PurchaseMonthlyCard(ctx context.Context, req *v1.PurchaseMonthlyCardRequest) (*v1.PurchaseMonthlyCardResponse, error) {
	userID := middleware.GetUserIDFromContext(ctx)
	data, err := s.uc.PurchaseMonthlyCard(ctx, userID, req)
	if err != nil {
		return &v1.PurchaseMonthlyCardResponse{
			Code:    500,
			Message: err.Error(),
		}, nil
	}

	return &v1.PurchaseMonthlyCardResponse{
		Code:    200,
		Message: "success",
		Data:    data,
	}, nil
}

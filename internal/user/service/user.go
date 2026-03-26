package service

import (
	"context"

	v1 "github.com/xuanyiying/smart-park/api/user/v1"
	"github.com/xuanyiying/smart-park/internal/user/biz"
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
	// TODO: 从 context 中获取 userID
	// 暂时返回空实现
	return &v1.GetUserInfoResponse{
		Code:    200,
		Message: "success",
		Data: &v1.UserInfo{
			UserId: "",
			OpenId: "",
		},
	}, nil
}

func (s *UserService) BindPlate(ctx context.Context, req *v1.BindPlateRequest) (*v1.BindPlateResponse, error) {
	// TODO: 从 context 中获取 userID
	userID := "" // 需要从 JWT context 中获取
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
	// TODO: 从 context 中获取 userID
	userID := "" // 需要从 JWT context 中获取
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
	// TODO: 从 context 中获取 userID
	userID := "" // 需要从 JWT context 中获取
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
	// TODO: 在 Task 1.2 中实现跨服务调用
	return &v1.ListParkingRecordsResponse{
		Code:    200,
		Message: "success",
		Data: &v1.ListParkingRecordsData{
			Records: []*v1.ParkingRecordInfo{},
			Total:   0,
		},
	}, nil
}

func (s *UserService) GetParkingRecord(ctx context.Context, req *v1.GetParkingRecordRequest) (*v1.GetParkingRecordResponse, error) {
	// TODO: 在 Task 1.2 中实现
	return &v1.GetParkingRecordResponse{
		Code:    200,
		Message: "success",
		Data:    nil,
	}, nil
}

func (s *UserService) ScanPay(ctx context.Context, req *v1.ScanPayRequest) (*v1.ScanPayResponse, error) {
	// TODO: 在 Task 1.3 中实现扫码支付
	return &v1.ScanPayResponse{
		Code:    200,
		Message: "success",
		Data: &v1.ScanPayData{
			OrderId: "",
			Amount:  0,
			PayUrl:  "",
			QrCode:  "",
		},
	}, nil
}

func (s *UserService) GetMonthlyCard(ctx context.Context, req *v1.GetMonthlyCardRequest) (*v1.GetMonthlyCardResponse, error) {
	// TODO: 后续实现
	return &v1.GetMonthlyCardResponse{
		Code:    200,
		Message: "success",
		Data:    nil,
	}, nil
}

func (s *UserService) PurchaseMonthlyCard(ctx context.Context, req *v1.PurchaseMonthlyCardRequest) (*v1.PurchaseMonthlyCardResponse, error) {
	// TODO: 后续实现
	return &v1.PurchaseMonthlyCardResponse{
		Code:    200,
		Message: "success",
		Data:    nil,
	}, nil
}

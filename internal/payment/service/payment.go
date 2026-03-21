// Package service provides gRPC service implementation for the payment service.
package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	"github.com/xuanyiying/smart-park/internal/payment/biz"
)

// PaymentService implements the PaymentService gRPC service.
type PaymentService struct {
	v1.UnimplementedPaymentServiceServer

	uc  *biz.PaymentUseCase
	log *log.Helper
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(uc *biz.PaymentUseCase, logger log.Logger) *PaymentService {
	return &PaymentService{
		uc:  uc,
		log: log.NewHelper(logger),
	}
}

// CreatePayment handles create payment request.
func (s *PaymentService) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.CreatePaymentResponse, error) {
	data, err := s.uc.CreatePayment(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CreatePayment failed: %v", err)
		return &v1.CreatePaymentResponse{
			Code:    500,
			Message: "创建支付失败",
		}, nil
	}

	return &v1.CreatePaymentResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// GetPaymentStatus handles get payment status request.
func (s *PaymentService) GetPaymentStatus(ctx context.Context, req *v1.GetPaymentStatusRequest) (*v1.GetPaymentStatusResponse, error) {
	data, err := s.uc.GetPaymentStatus(ctx, req.OrderId)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetPaymentStatus failed: %v", err)
		return &v1.GetPaymentStatusResponse{
			Code:    500,
			Message: "获取支付状态失败",
		}, nil
	}

	return &v1.GetPaymentStatusResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// WechatCallback handles WeChat payment callback.
func (s *PaymentService) WechatCallback(ctx context.Context, req *v1.WechatCallbackRequest) (*v1.WechatCallbackResponse, error) {
	return s.uc.HandleWechatCallback(ctx, req)
}

// AlipayCallback handles Alipay payment callback.
func (s *PaymentService) AlipayCallback(ctx context.Context, req *v1.AlipayCallbackRequest) (*v1.AlipayCallbackResponse, error) {
	return s.uc.HandleAlipayCallback(ctx, req)
}

// Refund handles refund request.
func (s *PaymentService) Refund(ctx context.Context, req *v1.RefundRequest) (*v1.RefundResponse, error) {
	data, err := s.uc.Refund(ctx, req.OrderId, req.Reason)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Refund failed: %v", err)
		return &v1.RefundResponse{
			Code:    500,
			Message: "退款失败",
		}, nil
	}

	return &v1.RefundResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

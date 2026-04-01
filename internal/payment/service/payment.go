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

// Reconcile handles reconciliation request.
func (s *PaymentService) Reconcile(ctx context.Context, req *v1.ReconcileRequest) (*v1.ReconcileResponse, error) {
	data, err := s.uc.Reconcile(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Reconcile failed: %v", err)
		return &v1.ReconcileResponse{
			Code:    500,
			Message: "对账失败",
		}, nil
	}

	return &v1.ReconcileResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// GetReconciliationResult handles get reconciliation result request.
func (s *PaymentService) GetReconciliationResult(ctx context.Context, req *v1.GetReconciliationResultRequest) (*v1.GetReconciliationResultResponse, error) {
	data, err := s.uc.GetReconciliationResult(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetReconciliationResult failed: %v", err)
		return &v1.GetReconciliationResultResponse{
			Code:    500,
			Message: "获取对账结果失败",
		}, nil
	}

	return &v1.GetReconciliationResultResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// ListReconciliationResults handles list reconciliation results request.
func (s *PaymentService) ListReconciliationResults(ctx context.Context, req *v1.ListReconciliationResultsRequest) (*v1.ListReconciliationResultsResponse, error) {
	data, err := s.uc.ListReconciliationResults(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ListReconciliationResults failed: %v", err)
		return &v1.ListReconciliationResultsResponse{
			Code:    500,
			Message: "获取对账结果列表失败",
		}, nil
	}

	return &v1.ListReconciliationResultsResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// HandleReconciliationException handles reconciliation exception request.
func (s *PaymentService) HandleReconciliationException(ctx context.Context, req *v1.HandleReconciliationExceptionRequest) (*v1.HandleReconciliationExceptionResponse, error) {
	data, err := s.uc.HandleReconciliationException(ctx, req)
	if err != nil {
		s.log.WithContext(ctx).Errorf("HandleReconciliationException failed: %v", err)
		return &v1.HandleReconciliationExceptionResponse{
			Code:    500,
			Message: "处理对账异常失败",
		}, nil
	}

	return &v1.HandleReconciliationExceptionResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

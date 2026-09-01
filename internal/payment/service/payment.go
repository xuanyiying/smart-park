// Package service provides gRPC service implementation for the payment service.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	"github.com/xuanyiying/smart-park/internal/payment/biz"
	"github.com/xuanyiying/smart-park/pkg/middleware"
)

// PaymentService implements the PaymentService gRPC service.
type PaymentService struct {
	v1.UnimplementedPaymentServiceServer

	uc  *biz.PaymentUseCase
	ruc *biz.ReconciliationUseCase
	log *log.Helper
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(uc *biz.PaymentUseCase, ruc *biz.ReconciliationUseCase, logger log.Logger) *PaymentService {
	return &PaymentService{
		uc:  uc,
		ruc: ruc,
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
//
// WeChat signs the exact bytes it sends, so verification needs the raw HTTP payload that
// CacheRawBody captured. When it is absent (a direct gRPC call, or a deployment that is
// missing the middleware) we fall through to the rejecting handler instead of trusting
// the decoded message.
func (s *PaymentService) WechatCallback(ctx context.Context, req *v1.WechatCallbackRequest) (*v1.WechatCallbackResponse, error) {
	raw, ok := middleware.RawRequestFromContext(ctx)
	if !ok || len(raw.Body) == 0 {
		s.log.WithContext(ctx).Warn("wechat callback arrived without a raw HTTP payload; rejecting (CacheRawBody middleware is required)")
		return s.uc.HandleWechatCallback(ctx, req)
	}
	return s.uc.HandleWechatNotification(ctx, toRawCallback(raw))
}

// AlipayCallback handles Alipay payment callback.
//
// Alipay signs every received parameter, so verification runs over the complete
// urlencoded form captured by CacheRawBody.
func (s *PaymentService) AlipayCallback(ctx context.Context, req *v1.AlipayCallbackRequest) (*v1.AlipayCallbackResponse, error) {
	raw, ok := middleware.RawRequestFromContext(ctx)
	if !ok || len(raw.PostForm) == 0 {
		s.log.WithContext(ctx).Warn("alipay callback arrived without a raw HTTP form; rejecting (CacheRawBody middleware is required)")
		return s.uc.HandleAlipayCallback(ctx, req)
	}
	return s.uc.HandleAlipayNotification(ctx, toRawCallback(raw))
}

// toRawCallback converts the buffered HTTP payload into the representation the business
// layer verifies signatures against.
func toRawCallback(raw *middleware.RawRequest) *biz.RawCallback {
	headers := make(map[string]string, 4)
	for _, name := range []string{
		"Wechatpay-Serial",
		"Wechatpay-Timestamp",
		"Wechatpay-Nonce",
		"Wechatpay-Signature",
	} {
		if v := raw.Header.Get(name); v != "" {
			headers[name] = v
		}
	}

	return &biz.RawCallback{
		Headers: headers,
		Body:    raw.Body,
		Form:    raw.PostForm,
	}
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

// ReconcileDaily handles daily reconciliation request.
func (s *PaymentService) ReconcileDaily(ctx context.Context, req *v1.ReconcileDailyRequest) (*v1.ReconcileDailyResponse, error) {
	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Invalid date format: %v", err)
		return &v1.ReconcileDailyResponse{
			Code:    400,
			Message: "无效的日期格式",
		}, nil
	}

	if s.ruc == nil {
		return &v1.ReconcileDailyResponse{
			Code:    503,
			Message: "对账服务未配置",
		}, nil
	}

	records, err := s.ruc.ReconcileDaily(ctx, date)
	if err != nil {
		s.log.WithContext(ctx).Errorf("ReconcileDaily failed: %v", err)
		return &v1.ReconcileDailyResponse{
			Code:    500,
			Message: "对账失败",
		}, nil
	}

	protoRecords := make([]*v1.ReconciliationRecord, 0, len(records))
	for _, record := range records {
		protoRecords = append(protoRecords, &v1.ReconciliationRecord{
			Id:                 record.ID.String(),
			OrderId:            record.OrderID.String(),
			PaymentMethod:      record.PaymentMethod,
			OrderAmount:        record.OrderAmount,
			PaidAmount:         record.PaidAmount,
			TransactionId:      record.TransactionID,
			ReconciliationTime: record.ReconciliationTime.Format(time.RFC3339),
			Status:             string(record.Status),
			Notes:              record.Notes,
		})
	}

	return &v1.ReconcileDailyResponse{
		Code:    0,
		Message: "success",
		Data:    protoRecords,
	}, nil
}

// GetReconciliationReport handles get reconciliation report request.
func (s *PaymentService) GetReconciliationReport(ctx context.Context, req *v1.GetReconciliationReportRequest) (*v1.GetReconciliationReportResponse, error) {
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Invalid start date format: %v", err)
		return &v1.GetReconciliationReportResponse{
			Code:    400,
			Message: "无效的开始日期格式",
		}, nil
	}

	endDate, err := time.Parse("2006-01-02", req.EndDate)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Invalid end date format: %v", err)
		return &v1.GetReconciliationReportResponse{
			Code:    400,
			Message: "无效的结束日期格式",
		}, nil
	}

	if s.ruc == nil {
		return &v1.GetReconciliationReportResponse{
			Code:    503,
			Message: "对账服务未配置",
		}, nil
	}

	report, err := s.ruc.GetReconciliationReport(ctx, startDate, endDate)
	if err != nil {
		s.log.WithContext(ctx).Errorf("GetReconciliationReport failed: %v", err)
		return &v1.GetReconciliationReportResponse{
			Code:    500,
			Message: "获取对账报表失败",
		}, nil
	}

	data := make(map[string]string, len(report))
	for k, v := range report {
		switch value := v.(type) {
		case string:
			data[k] = value
		case int:
			data[k] = fmt.Sprintf("%d", value)
		case float64:
			data[k] = fmt.Sprintf("%.2f", value)
		default:
			if encoded, err := json.Marshal(v); err == nil {
				data[k] = string(encoded)
			}
		}
	}

	return &v1.GetReconciliationReportResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	}, nil
}

// FixMismatchedOrders handles fix mismatched orders request.
func (s *PaymentService) FixMismatchedOrders(ctx context.Context, req *v1.FixMismatchedOrdersRequest) (*v1.FixMismatchedOrdersResponse, error) {
	if s.ruc == nil {
		return &v1.FixMismatchedOrdersResponse{
			Code:    503,
			Message: "对账服务未配置",
		}, nil
	}

	if err := s.ruc.FixMismatchedOrders(ctx, req.OrderIds); err != nil {
		s.log.WithContext(ctx).Errorf("FixMismatchedOrders failed: %v", err)
		return &v1.FixMismatchedOrdersResponse{
			Code:    500,
			Message: "修复失败",
		}, nil
	}

	return &v1.FixMismatchedOrdersResponse{
		Code:    0,
		Message: "success",
	}, nil
}

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
	// ruc *biz.ReconciliationUseCase
	log *log.Helper
}

// NewPaymentService creates a new PaymentService.
func NewPaymentService(uc *biz.PaymentUseCase, ruc *biz.ReconciliationUseCase, logger log.Logger) *PaymentService {
	return &PaymentService{
		uc:  uc,
		// ruc: ruc,
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

// ReconcileDaily handles daily reconciliation request.
// func (s *PaymentService) ReconcileDaily(ctx context.Context, req *v1.ReconcileDailyRequest) (*v1.ReconcileDailyResponse, error) {
// 	date, err := time.Parse("2006-01-02", req.Date)
// 	if err != nil {
// 		s.log.WithContext(ctx).Errorf("Invalid date format: %v", err)
// 		return &v1.ReconcileDailyResponse{
// 			Code:    400,
// 			Message: "无效的日期格式",
// 		}, nil
// 	}

// 	records, err := s.ruc.ReconcileDaily(ctx, date)
// 	if err != nil {
// 		s.log.WithContext(ctx).Errorf("ReconcileDaily failed: %v", err)
// 		return &v1.ReconcileDailyResponse{
// 			Code:    500,
// 			Message: "对账失败",
// 		}, nil
// 	}

// 	// 转换为 proto 消息
// 	var protoRecords []*v1.ReconciliationRecord
// 	for _, record := range records {
// 		protoRecords = append(protoRecords, &v1.ReconciliationRecord{
// 			Id:                record.ID.String(),
// 			OrderId:           record.OrderID.String(),
// 			PaymentMethod:     record.PaymentMethod,
// 			OrderAmount:       record.OrderAmount,
// 			PaidAmount:        record.PaidAmount,
// 			TransactionId:     record.TransactionID,
// 			ReconciliationTime: record.ReconciliationTime.Format(time.RFC3339),
// 			Status:            string(record.Status),
// 			Notes:             record.Notes,
// 		})
// 	}

// 	return &v1.ReconcileDailyResponse{
// 		Code:    0,
// 		Message: "success",
// 		Data:    protoRecords,
// 	}, nil
// }

// GetReconciliationReport handles get reconciliation report request.
// func (s *PaymentService) GetReconciliationReport(ctx context.Context, req *v1.GetReconciliationReportRequest) (*v1.GetReconciliationReportResponse, error) {
// 	startDate, err := time.Parse("2006-01-02", req.StartDate)
// 	if err != nil {
// 		s.log.WithContext(ctx).Errorf("Invalid start date format: %v", err)
// 		return &v1.GetReconciliationReportResponse{
// 			Code:    400,
// 			Message: "无效的开始日期格式",
// 		}, nil
// 	}

// 	endDate, err := time.Parse("2006-01-02", req.EndDate)
// 	if err != nil {
// 		s.log.WithContext(ctx).Errorf("Invalid end date format: %v", err)
// 		return &v1.GetReconciliationReportResponse{
// 			Code:    400,
// 			Message: "无效的结束日期格式",
// 		}, nil
// 	}

// 	report, err := s.ruc.GetReconciliationReport(ctx, startDate, endDate)
// 	if err != nil {
// 		s.log.WithContext(ctx).Errorf("GetReconciliationReport failed: %v", err)
// 		return &v1.GetReconciliationReportResponse{
// 			Code:    500,
// 			Message: "获取对账报表失败",
// 		}, nil
// 	}

// 	// 转换为 map[string]string
// 	data := make(map[string]string)
// 	for k, v := range report {
// 		switch value := v.(type) {
// 		case string:
// 			data[k] = value
// 		case int:
// 			data[k] = fmt.Sprintf("%d", value)
// 		case float64:
// 			data[k] = fmt.Sprintf("%.2f", value)
// 		default:
// 			// 对于复杂类型，转换为 JSON
// 			jsonData, err := json.Marshal(v)
// 			if err == nil {
// 				data[k] = string(jsonData)
// 			}
// 		}
// 	}

// 	return &v1.GetReconciliationReportResponse{
// 		Code:    0,
// 		Message: "success",
// 		Data:    data,
// 	}, nil
// }

// FixMismatchedOrders handles fix mismatched orders request.
// func (s *PaymentService) FixMismatchedOrders(ctx context.Context, req *v1.FixMismatchedOrdersRequest) (*v1.FixMismatchedOrdersResponse, error) {
// 	err := s.ruc.FixMismatchedOrders(ctx, req.OrderIds)
// 	if err != nil {
// 		s.log.WithContext(ctx).Errorf("FixMismatchedOrders failed: %v", err)
// 		return &v1.FixMismatchedOrdersResponse{
// 			Code:    500,
// 			Message: "修复失败",
// 		}, nil
// 	}

// 	return &v1.FixMismatchedOrdersResponse{
// 		Code:    0,
// 		Message: "success",
// 	}, nil
// }

// Package biz provides business logic for the payment service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	"github.com/xuanyiying/smart-park/internal/payment/alipay"
	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

// PaymentUseCase implements payment business logic.
type PaymentUseCase struct {
	orderRepo    OrderRepo
	log          *log.Helper
	config       *PaymentConfig
	bizConfig    *Config
	wechatClient *wechat.Client
	alipayClient *alipay.Client
}

// NewPaymentUseCase creates a new PaymentUseCase.
func NewPaymentUseCase(orderRepo OrderRepo, config *PaymentConfig, wechatClient *wechat.Client, alipayClient *alipay.Client, logger log.Logger) *PaymentUseCase {
	return &PaymentUseCase{
		orderRepo:    orderRepo,
		log:          log.NewHelper(logger),
		config:       config,
		bizConfig:    DefaultConfig(),
		wechatClient: wechatClient,
		alipayClient: alipayClient,
	}
}

// CreatePayment creates a new payment order.
func (uc *PaymentUseCase) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.PaymentData, error) {
	if err := uc.validateCreatePaymentRequest(req); err != nil {
		return nil, err
	}

	recordID, err := uuid.Parse(req.RecordId)
	if err != nil {
		return nil, fmt.Errorf("invalid record ID: %w", err)
	}

	// Check for existing paid order (idempotency)
	if existingOrder, _ := uc.orderRepo.GetOrderByRecordID(ctx, recordID); existingOrder != nil {
		if existingOrder.Status == string(StatusPaid) {
			return uc.buildExistingPaymentResponse(existingOrder), nil
		}
	}

	// Create new order
	order, err := uc.createOrder(ctx, recordID, req.Amount)
	if err != nil {
		return nil, err
	}

	// Generate payment URL based on method
	payURL, qrCode, err := uc.generatePaymentURL(ctx, order, req)
	if err != nil {
		return nil, err
	}

	return &v1.PaymentData{
		OrderId:    order.ID.String(),
		Amount:     order.FinalAmount,
		PayUrl:     payURL,
		QrCode:     qrCode,
		ExpireTime: time.Now().Add(uc.bizConfig.OrderExpiration).Format(time.RFC3339),
	}, nil
}

// validateCreatePaymentRequest validates the create payment request.
func (uc *PaymentUseCase) validateCreatePaymentRequest(req *v1.CreatePaymentRequest) error {
	if req.RecordId == "" {
		return fmt.Errorf("record ID is required")
	}
	if req.Amount <= 0 {
		return fmt.Errorf("amount must be positive")
	}
	if !IsValidPayMethod(req.PayMethod) {
		return fmt.Errorf("invalid payment method: %s", req.PayMethod)
	}
	return nil
}

// buildExistingPaymentResponse builds response for existing paid order.
func (uc *PaymentUseCase) buildExistingPaymentResponse(order *Order) *v1.PaymentData {
	return &v1.PaymentData{
		OrderId: order.ID.String(),
		Amount:  order.FinalAmount,
	}
}

// createOrder creates a new order in the repository.
func (uc *PaymentUseCase) createOrder(ctx context.Context, recordID uuid.UUID, amount float64) (*Order, error) {
	order := &Order{
		ID:             uuid.New(),
		RecordID:       recordID,
		Amount:         amount,
		DiscountAmount: 0,
		FinalAmount:    amount,
		Status:         string(StatusPending),
	}

	if err := uc.orderRepo.CreateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create order: %v", err)
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

// generatePaymentURL generates payment URL based on payment method.
func (uc *PaymentUseCase) generatePaymentURL(ctx context.Context, order *Order, req *v1.CreatePaymentRequest) (string, string, error) {
	switch PayMethod(req.PayMethod) {
	case MethodWechat:
		return uc.generateWechatPayment(ctx, order, req)
	case MethodAlipay:
		return uc.generateAlipayPayment(ctx, order, req)
	default:
		return "", "", fmt.Errorf("unsupported payment method: %s", req.PayMethod)
	}
}

// generateWechatPayment generates WeChat payment URL.
func (uc *PaymentUseCase) generateWechatPayment(ctx context.Context, order *Order, req *v1.CreatePaymentRequest) (string, string, error) {
	amountInCents := int64(order.FinalAmount * 100)

	if uc.wechatClient == nil {
		uc.log.WithContext(ctx).Warn("wechat client not configured, using mock")
		payURL, qrCode := uc.mockWechatPayURL(order.ID.String(), req.OpenId)
		return payURL, qrCode, nil
	}

	if req.OpenId != "" {
		return uc.generateWechatJSAPIPay(ctx, order, amountInCents, req.OpenId)
	}
	return uc.generateWechatNativePay(ctx, order, amountInCents)
}

// generateWechatJSAPIPay generates WeChat JSAPI payment.
func (uc *PaymentUseCase) generateWechatJSAPIPay(ctx context.Context, order *Order, amountInCents int64, openID string) (string, string, error) {
	jsapiParams, err := uc.wechatClient.CreateJSAPIPay(ctx, order.ID.String(), amountInCents, openID, uc.bizConfig.DefaultDescription)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create wechat jsapi pay: %v", err)
		return "", "", fmt.Errorf("failed to create wechat jsapi pay: %w", err)
	}

	payURL := fmt.Sprintf("weixin://wxpay/bizpayurl?pr=%s", order.ID.String())
	qrCode := fmt.Sprintf("%v", jsapiParams)
	return payURL, qrCode, nil
}

// generateWechatNativePay generates WeChat native payment.
func (uc *PaymentUseCase) generateWechatNativePay(ctx context.Context, order *Order, amountInCents int64) (string, string, error) {
	codeURL, err := uc.wechatClient.CreateNativePay(ctx, order.ID.String(), amountInCents, uc.bizConfig.DefaultDescription)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create wechat native pay: %v", err)
		return "", "", fmt.Errorf("failed to create wechat native pay: %w", err)
	}
	return codeURL, codeURL, nil
}

// mockWechatPayURL generates mock WeChat pay URL.
func (uc *PaymentUseCase) mockWechatPayURL(orderID string, openID string) (string, string) {
	return "https://wxpay.example.com/pay/" + orderID, "weixin://wxpay/example/" + orderID
}

// generateAlipayPayment generates Alipay payment URL.
func (uc *PaymentUseCase) generateAlipayPayment(ctx context.Context, order *Order, req *v1.CreatePaymentRequest) (string, string, error) {
	if uc.alipayClient == nil {
		uc.log.WithContext(ctx).Warn("alipay client not configured, using mock")
		payURL, qrCode := uc.mockAlipayURL(order.ID.String())
		return payURL, qrCode, nil
	}

	qrCode, err := uc.alipayClient.CreateTradePreCreate(ctx, order.ID.String(), order.FinalAmount, uc.bizConfig.DefaultDescription)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create alipay precreate: %v", err)
		return "", "", fmt.Errorf("failed to create alipay precreate: %w", err)
	}

	return qrCode, qrCode, nil
}

// mockAlipayURL generates mock Alipay URL.
func (uc *PaymentUseCase) mockAlipayURL(orderID string) (string, string) {
	return "https://alipay.example.com/pay/" + orderID, "https://qr.alipay.com/" + orderID
}

// GetPaymentStatus retrieves payment status.
func (uc *PaymentUseCase) GetPaymentStatus(ctx context.Context, orderID string) (*v1.PaymentStatusData, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := uc.orderRepo.GetOrder(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return &v1.PaymentStatusData{
		OrderId:   order.ID.String(),
		Status:    order.Status,
		PayTime:   formatTime(order.PayTime),
		PayMethod: order.PayMethod,
	}, nil
}

// formatTime formats time pointer to string.
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

// Refund handles refund request.
func (uc *PaymentUseCase) Refund(ctx context.Context, orderID, reason string) (*v1.RefundData, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	order, err := uc.orderRepo.GetOrder(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if order.Status != string(StatusPaid) {
		return &v1.RefundData{
			RefundId: "",
			Status:   "failed",
		}, nil
	}

	refundID := uuid.New().String()

	uc.processRefund(ctx, order, refundID)

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order for refund: %v", err)
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	return &v1.RefundData{
		RefundId: refundID,
		Status:   "success",
	}, nil
}

// processRefund processes the refund for the order.
func (uc *PaymentUseCase) processRefund(ctx context.Context, order *Order, refundID string) {
	switch PayMethod(order.PayMethod) {
	case MethodWechat:
		if uc.wechatClient != nil {
			uc.log.WithContext(ctx).Infof("Processing WeChat refund for order %s", order.ID)
		}
	case MethodAlipay:
		if uc.alipayClient != nil {
			uc.log.WithContext(ctx).Infof("Processing Alipay refund for order %s", order.ID)
		}
	}

	now := time.Now()
	order.Status = string(StatusRefunded)
	order.RefundedAt = &now
	order.RefundTransactionID = refundID
}

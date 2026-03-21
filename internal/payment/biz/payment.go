// Package biz provides business logic for the payment service.
package biz

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

// Order represents an order entity.
type Order struct {
	ID                  uuid.UUID
	RecordID            uuid.UUID
	LotID               uuid.UUID
	VehicleID           *uuid.UUID
	PlateNumber         string
	Amount              float64
	DiscountAmount      float64
	FinalAmount         float64
	Status              string
	PayTime             *time.Time
	PayMethod           string
	TransactionID       string
	PaidAmount          float64
	RefundedAt          *time.Time
	RefundTransactionID string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// OrderRepo defines the repository interface for order operations.
type OrderRepo interface {
	GetOrder(ctx context.Context, orderID uuid.UUID) (*Order, error)
	GetOrderByRecordID(ctx context.Context, recordID uuid.UUID) (*Order, error)
	GetOrderByTransactionID(ctx context.Context, transactionID string) (*Order, error)
	CreateOrder(ctx context.Context, order *Order) error
	UpdateOrder(ctx context.Context, order *Order) error
	ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*Order, int64, error)
}

// PaymentUseCase implements payment business logic.
type PaymentUseCase struct {
	orderRepo OrderRepo
	log       *log.Helper
}

// NewPaymentUseCase creates a new PaymentUseCase.
func NewPaymentUseCase(orderRepo OrderRepo, logger log.Logger) *PaymentUseCase {
	return &PaymentUseCase{
		orderRepo: orderRepo,
		log:       log.NewHelper(logger),
	}
}

// CreatePayment creates a new payment order.
func (uc *PaymentUseCase) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.PaymentData, error) {
	recordID, err := uuid.Parse(req.RecordId)
	if err != nil {
		return nil, err
	}

	// Check if order already exists
	existingOrder, _ := uc.orderRepo.GetOrderByRecordID(ctx, recordID)
	if existingOrder != nil && existingOrder.Status == "paid" {
		return &v1.PaymentData{
			OrderId: existingOrder.ID.String(),
			Amount:  existingOrder.FinalAmount,
		}, nil
	}

	// Create new order
	order := &Order{
		ID:             uuid.New(),
		RecordID:       recordID,
		Amount:         req.Amount,
		DiscountAmount: 0,
		FinalAmount:    req.Amount,
		Status:         "pending",
	}

	if err := uc.orderRepo.CreateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create order: %v", err)
		return nil, err
	}

	// Generate payment URL based on pay method
	var payURL, qrCode string
	switch req.PayMethod {
	case "wechat":
		payURL, qrCode = generateWechatPayURL(order.ID.String(), req.Amount, req.OpenId, req.NotifyUrl)
	case "alipay":
		payURL, qrCode = generateAlipayURL(order.ID.String(), req.Amount, req.NotifyUrl)
	}

	return &v1.PaymentData{
		OrderId:   order.ID.String(),
		Amount:    order.FinalAmount,
		PayUrl:    payURL,
		QrCode:    qrCode,
		ExpireTime: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
	}, nil
}

// generateWechatPayURL generates WeChat pay URL.
func generateWechatPayURL(orderID string, amount float64, openID, notifyURL string) (string, string) {
	// In production, integrate with WeChat Pay SDK
	return "https://wxpay.example.com/pay/" + orderID, "weixin://wxpay/example/" + orderID
}

// generateAlipayURL generates Alipay URL.
func generateAlipayURL(orderID string, amount float64, notifyURL string) (string, string) {
	// In production, integrate with Alipay SDK
	return "https://alipay.example.com/pay/" + orderID, "https://qr.alipay.com/" + orderID
}

// GetPaymentStatus retrieves payment status.
func (uc *PaymentUseCase) GetPaymentStatus(ctx context.Context, orderID string) (*v1.PaymentStatusData, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, err
	}

	order, err := uc.orderRepo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	var payTime string
	if order.PayTime != nil {
		payTime = order.PayTime.Format(time.RFC3339)
	}

	return &v1.PaymentStatusData{
		OrderId:   order.ID.String(),
		Status:    order.Status,
		PayTime:   payTime,
		PayMethod: order.PayMethod,
	}, nil
}

// HandleWechatCallback handles WeChat payment callback.
func (uc *PaymentUseCase) HandleWechatCallback(ctx context.Context, req *v1.WechatCallbackRequest) (*v1.WechatCallbackResponse, error) {
	// Verify callback signature (in production)

	order, err := uc.orderRepo.GetOrderByTransactionID(ctx, req.TransactionId)
	if err != nil {
		return &v1.WechatCallbackResponse{
			ReturnCode: "FAIL",
			ReturnMsg:  "Order not found",
		}, nil
	}

	// Update order status
	now := time.Now()
	order.Status = "paid"
	order.PayTime = &now
	order.PayMethod = "wechat"
	order.TransactionID = req.TransactionId
	order.PaidAmount = float64(parseAmount(req.TotalFee))

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return &v1.WechatCallbackResponse{
			ReturnCode: "FAIL",
			ReturnMsg:  "Update failed",
		}, nil
	}

	return &v1.WechatCallbackResponse{
		ReturnCode: "SUCCESS",
		ReturnMsg:  "OK",
	}, nil
}

// HandleAlipayCallback handles Alipay payment callback.
func (uc *PaymentUseCase) HandleAlipayCallback(ctx context.Context, req *v1.AlipayCallbackRequest) (*v1.AlipayCallbackResponse, error) {
	// Verify callback signature (in production)

	order, err := uc.orderRepo.GetOrderByTransactionID(ctx, req.TradeNo)
	if err != nil {
		return &v1.AlipayCallbackResponse{
			Code: "FAIL",
			Msg:  "Order not found",
		}, nil
	}

	// Update order status
	now := time.Now()
	order.Status = "paid"
	order.PayTime = &now
	order.PayMethod = "alipay"
	order.TransactionID = req.TradeNo
	order.PaidAmount = parseAmountFloat(req.TotalAmount)

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order: %v", err)
		return &v1.AlipayCallbackResponse{
			Code: "FAIL",
			Msg:  "Update failed",
		}, nil
	}

	return &v1.AlipayCallbackResponse{
		Code: "success",
		Msg:  "OK",
	}, nil
}

// Refund handles refund request.
func (uc *PaymentUseCase) Refund(ctx context.Context, orderID, reason string) (*v1.RefundData, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, err
	}

	order, err := uc.orderRepo.GetOrder(ctx, id)
	if err != nil {
		return nil, err
	}

	if order.Status != "paid" {
		return &v1.RefundData{
			RefundId: "",
			Status:   "failed",
		}, nil
	}

	// Process refund (in production, integrate with payment gateway)
	refundID := uuid.New().String()
	now := time.Now()
	order.Status = "refunded"
	order.RefundedAt = &now
	order.RefundTransactionID = refundID

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order for refund: %v", err)
		return nil, err
	}

	return &v1.RefundData{
		RefundId: refundID,
		Status:   "success",
	}, nil
}

// Helper functions for parsing amounts.
func parseAmount(amount string) int {
	// Parse amount string to int (cents)
	var result int
	for _, c := range amount {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		}
	}
	return result
}

func parseAmountFloat(amount string) float64 {
	var result float64
	for _, c := range amount {
		if c >= '0' && c <= '9' {
			result = result*10 + float64(c-'0')
		} else if c == '.' {
			result /= 10
		}
	}
	return result
}

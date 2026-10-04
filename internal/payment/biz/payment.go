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
	"github.com/xuanyiying/smart-park/pkg/outbox"
)

// PaymentUseCase implements payment business logic.
type PaymentUseCase struct {
	orderRepo    OrderRepo
	recordRepo   RecordRepo
	gateClient   GateControlService
	log          *log.Helper
	config       *PaymentConfig
	bizConfig    *Config
	wechatClient *wechat.Client
	alipayClient *alipay.Client
	// outbox, when attached, receives an order.settled event inside the
	// settlement transaction; a dispatcher then applies the side effects
	// (gate opening) with retries. Nil keeps the legacy inline behaviour.
	outbox outbox.Store

	// chargingClient confirms charging sessions after their orders settle.
	// Nil means charging confirmations are skipped (logged), as in tests and
	// deployments without the charging service.
	chargingClient ChargingPaymentClient

	// queryGateway asks the channel whether an order was paid. Nil means the default
	// implementation is used; tests replace it with a stub so sweep behaviour can be
	// scripted without real merchant credentials.
	queryGateway func(ctx context.Context, order *Order) (paid bool, transactionID string, err error)
}

// NewPaymentUseCase creates a new PaymentUseCase.
func NewPaymentUseCase(orderRepo OrderRepo, recordRepo RecordRepo, gateClient GateControlService, config *PaymentConfig, wechatClient *wechat.Client, alipayClient *alipay.Client, logger log.Logger) *PaymentUseCase {
	return &PaymentUseCase{
		orderRepo:    orderRepo,
		recordRepo:   recordRepo,
		gateClient:   gateClient,
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

	var recordID uuid.UUID
	var err error
	var order *Order
	amount := int64(0)

	if req.OrderType == "charging" && req.SessionId != "" {
		sessionID, parseErr := uuid.Parse(req.SessionId)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid session ID: %w", parseErr)
		}
		recordID = sessionID
	} else {
		recordID, err = uuid.Parse(req.RecordId)
		if err != nil {
			return nil, fmt.Errorf("invalid record ID: %w", err)
		}
	}

	// 同一记录已有订单时幂等处理：
	// paid → 直接返回已有支付信息；pending → 复用订单重建支付链接。
	// 若放任重复创建，record_id 会出现多笔订单，GetOrderByRecordID 的
	// 单行语义随即报错，用户重新扫码反而失败。
	if existingOrder, _ := uc.orderRepo.GetOrderByRecordID(ctx, recordID); existingOrder != nil {
		switch existingOrder.Status {
		case string(StatusPaid):
			return uc.buildExistingPaymentResponse(existingOrder), nil
		case string(StatusPending):
			order = existingOrder
			amount = order.FinalAmount
		}
	}

	if order == nil {
		amount = req.Amount
		if req.OrderType != "charging" {
			// 客户端自报金额不可信：以出场时计费并落库的服务端金额为权威，
			// 防止篡改金额创建低价订单。取不到服务端金额（历史记录/异常路径）
			// 时保持原行为，由对账兜底。
			if serverAmount, ok := uc.serverAmountForRecord(ctx, recordID); ok && serverAmount > 0 {
				if serverAmount != req.Amount {
					uc.logSecurityEvent(ctx, SecurityEventAmountMismatch, recordID.String(),
						float64(serverAmount)/100, float64(req.Amount)/100, "",
						"client-provided amount differs from the server-side exit fee; using the server amount")
				}
				amount = serverAmount
			}
		}

		var err error
		order, err = uc.createOrder(ctx, recordID, req.OrderType, amount)
		if err != nil {
			return nil, err
		}
	}

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

// serverAmountForRecord looks up the server-side exit fee for a parking record.
// ok is false when the record does not exist or carries no authoritative amount.
func (uc *PaymentUseCase) serverAmountForRecord(ctx context.Context, recordID uuid.UUID) (int64, bool) {
	if uc.recordRepo == nil {
		return 0, false
	}
	record, err := uc.recordRepo.GetRecord(ctx, recordID.String())
	if err != nil {
		uc.log.WithContext(ctx).Warnf("failed to load parking record %s for amount check: %v", recordID, err)
		return 0, false
	}
	if record == nil || record.FinalAmount <= 0 {
		return 0, false
	}
	return record.FinalAmount, true
}

// Order types. The settlement side effects branch on this: parking orders open
// the exit gate, charging orders confirm the charging session.
const (
	OrderTypeParking  = "parking"
	OrderTypeCharging = "charging"
)

// createOrder creates a new order in the repository. amount is in cents (分).
func (uc *PaymentUseCase) createOrder(ctx context.Context, recordID uuid.UUID, orderType string, amount int64) (*Order, error) {
	if orderType == "" {
		orderType = OrderTypeParking
	}
	order := &Order{
		ID:             uuid.New(),
		RecordID:       recordID,
		Amount:         amount,
		DiscountAmount: 0,
		FinalAmount:    amount,
		Status:         string(StatusPending),
		OrderType:      orderType,
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
	// order.FinalAmount is already in cents (分); previously it was 元 and the
	// *100 conversion here was the silent rounding point of the money path.
	amountInCents := order.FinalAmount

	if uc.wechatClient == nil {
		uc.log.WithContext(ctx).Error("wechat client not configured, cannot generate payment")
		return "", "", fmt.Errorf("wechat client not configured")
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



// generateAlipayPayment generates Alipay payment URL.
func (uc *PaymentUseCase) generateAlipayPayment(ctx context.Context, order *Order, req *v1.CreatePaymentRequest) (string, string, error) {
	if uc.alipayClient == nil {
		uc.log.WithContext(ctx).Error("alipay client not configured, cannot generate payment")
		return "", "", fmt.Errorf("alipay client not configured")
	}

	qrCode, err := uc.alipayClient.CreateTradePreCreate(ctx, order.ID.String(), order.FinalAmount, uc.bizConfig.DefaultDescription)
	// order.FinalAmount is cents (分); the alipay client converts to the yuan
	// string the gateway expects.
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create alipay precreate: %v", err)
		return "", "", fmt.Errorf("failed to create alipay precreate: %w", err)
	}

	return qrCode, qrCode, nil
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

// Refund handles a full refund request.
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
	// order.FinalAmount is already cents (分).
	totalCents := order.FinalAmount

	transactionID, err := uc.RefundOrder(ctx, order, refundID, totalCents)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to process refund: %v", err)
		return &v1.RefundData{
			RefundId: "",
			Status:   "failed",
		}, nil
	}

	// Persist the terminal state only after the gateway confirmed the refund.
	now := time.Now()
	order.Status = string(StatusRefunded)
	order.RefundedAt = &now
	order.RefundTransactionID = refundID

	if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update order for refund: %v", err)
		return nil, fmt.Errorf("failed to update order: %w", err)
	}

	return &v1.RefundData{
		RefundId: transactionID,
		Status:   "success",
	}, nil
}

// RefundOrder charges the gateway back by amountCents and returns the gateway transaction
// identifier. It performs a real API call and only reports success when the gateway
// confirmed; callers are responsible for persisting the resulting order state.
//
// Fabricating a transaction identifier here would make the ledger claim a refund that
// never happened, which is why this method has no "simulated" path.
func (uc *PaymentUseCase) RefundOrder(ctx context.Context, order *Order, refundID string, amountCents int64) (string, error) {
	if order == nil {
		return "", fmt.Errorf("order is nil")
	}
	if amountCents <= 0 {
		return "", fmt.Errorf("refund amount must be positive, got %d cents", amountCents)
	}

	totalCents := order.FinalAmount
	if amountCents > totalCents {
		return "", fmt.Errorf("refund amount %d cents exceeds order amount %d cents", amountCents, totalCents)
	}

	switch PayMethod(order.PayMethod) {
	case MethodWechat:
		if uc.wechatClient == nil {
			return "", fmt.Errorf("wechat client not configured")
		}
		uc.log.WithContext(ctx).Infof("Processing WeChat refund for order %s, amount: %d cents", order.ID, amountCents)
		if err := uc.wechatClient.Refund(ctx, order.ID.String(), refundID, totalCents, amountCents); err != nil {
			uc.log.WithContext(ctx).Errorf("WeChat refund failed: %v", err)
			return "", fmt.Errorf("wechat refund failed: %w", err)
		}
	case MethodAlipay:
		if uc.alipayClient == nil {
			return "", fmt.Errorf("alipay client not configured")
		}
		// The client converts cents to the yuan string Alipay expects.
		uc.log.WithContext(ctx).Infof("Processing Alipay refund for order %s, amount: %d cents", order.ID, amountCents)
		if err := uc.alipayClient.Refund(ctx, order.ID.String(), refundID, amountCents); err != nil {
			uc.log.WithContext(ctx).Errorf("Alipay refund failed: %v", err)
			return "", fmt.Errorf("alipay refund failed: %w", err)
		}
	default:
		return "", fmt.Errorf("unknown payment method: %s", order.PayMethod)
	}

	return refundID, nil
}

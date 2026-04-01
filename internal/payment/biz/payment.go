// Package biz provides business logic for the payment service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/seata/seata-go/pkg/tm"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	"github.com/xuanyiying/smart-park/internal/payment/alipay"
	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

// PaymentUseCase implements payment business logic.
type PaymentUseCase struct {
	orderRepo                OrderRepo
	recordRepo               RecordRepo
	reconciliationRepo       ReconciliationRepo
	reconciliationExceptionRepo ReconciliationExceptionRepo
	gateClient               GateControlService
	notificationClient       NotificationService
	log                      *log.Helper
	config                   *PaymentConfig
	bizConfig                *Config
	wechatClient             *wechat.Client
	alipayClient             *alipay.Client
}

// NotificationService defines the interface for notification service
type NotificationService interface {
	CreatePaymentNotification(ctx context.Context, userID string, orderID string, amount float64, status string) error
}

// NewPaymentUseCase creates a new PaymentUseCase.
func NewPaymentUseCase(orderRepo OrderRepo, recordRepo RecordRepo, gateClient GateControlService, notificationClient NotificationService, config *PaymentConfig, wechatClient *wechat.Client, alipayClient *alipay.Client, logger log.Logger) *PaymentUseCase {
	return &PaymentUseCase{
		orderRepo:                orderRepo,
		recordRepo:               recordRepo,
		gateClient:               gateClient,
		notificationClient:       notificationClient,
		log:                      log.NewHelper(logger),
		config:                   config,
		bizConfig:                DefaultConfig(),
		wechatClient:             wechatClient,
		alipayClient:             alipayClient,
	}
}

// CreatePayment creates a new payment order.
func (uc *PaymentUseCase) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.PaymentData, error) {
	if err := uc.validateCreatePaymentRequest(req); err != nil {
		return nil, err
	}

	var paymentData *v1.PaymentData
	gc := &tm.GtxConfig{
		Name: "create-payment",
		Timeout: time.Minute,
		Propagation: tm.Required,
	}

	err := tm.WithGlobalTx(ctx, gc, func(ctx context.Context) error {
		recordID, err := uuid.Parse(req.RecordId)
		if err != nil {
			return fmt.Errorf("invalid record ID: %w", err)
		}

		// Check for existing paid order (idempotency)
		if existingOrder, _ := uc.orderRepo.GetOrderByRecordID(ctx, recordID); existingOrder != nil {
			if existingOrder.Status == string(StatusPaid) {
				paymentData = uc.buildExistingPaymentResponse(existingOrder)
				return nil
			}
		}

		// Create new order
		order, err := uc.createOrder(ctx, recordID, req.Amount)
		if err != nil {
			return err
		}

		// Generate payment URL based on method
		payURL, qrCode, err := uc.generatePaymentURL(ctx, order, req)
		if err != nil {
			return err
		}

		paymentData = &v1.PaymentData{
			OrderId:    order.ID.String(),
			Amount:     order.FinalAmount,
			PayUrl:     payURL,
			QrCode:     qrCode,
			ExpireTime: time.Now().Add(uc.bizConfig.OrderExpiration).Format(time.RFC3339),
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return paymentData, nil
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

// Refund handles refund request.
func (uc *PaymentUseCase) Refund(ctx context.Context, orderID, reason string) (*v1.RefundData, error) {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return nil, fmt.Errorf("invalid order ID: %w", err)
	}

	var refundData *v1.RefundData
	gc := &tm.GtxConfig{
		Name: "refund-payment",
		Timeout: time.Minute,
		Propagation: tm.Required,
	}

	err = tm.WithGlobalTx(ctx, gc, func(ctx context.Context) error {
		order, err := uc.orderRepo.GetOrder(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get order: %w", err)
		}

		if order.Status != string(StatusPaid) {
			refundData = &v1.RefundData{
				RefundId: "",
				Status:   "failed",
			}
			return nil
		}

		refundID := uuid.New().String()

		if err := uc.processRefund(ctx, order, refundID); err != nil {
			uc.log.WithContext(ctx).Errorf("failed to process refund: %v", err)
			refundData = &v1.RefundData{
				RefundId: "",
				Status:   "failed",
			}
			return nil
		}

		if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
			uc.log.WithContext(ctx).Errorf("failed to update order for refund: %v", err)
			return fmt.Errorf("failed to update order: %w", err)
		}

		refundData = &v1.RefundData{
			RefundId: refundID,
			Status:   "success",
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return refundData, nil
}

// AutoPay processes automatic payment for an order
func (uc *PaymentUseCase) AutoPay(ctx context.Context, orderID string, userID string) error {
	id, err := uuid.Parse(orderID)
	if err != nil {
		return fmt.Errorf("invalid order ID: %w", err)
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID: %w", err)
	}

	gc := &tm.GtxConfig{
		Name: "auto-pay",
		Timeout: time.Minute,
		Propagation: tm.Required,
	}

	err = tm.WithGlobalTx(ctx, gc, func(ctx context.Context) error {
		order, err := uc.orderRepo.GetOrder(ctx, id)
		if err != nil {
			return fmt.Errorf("failed to get order: %w", err)
		}

		if order.Status == string(StatusPaid) {
			return nil // Order already paid
		}

		if order.AutoPayAttempts >= 3 {
			order.AutoPayStatus = "failed"
			if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
				return fmt.Errorf("failed to update order: %w", err)
			}
			return fmt.Errorf("maximum auto-pay attempts reached")
		}

		// Increment auto-pay attempts
		order.AutoPayAttempts++
		order.AutoPayStatus = "processing"
		order.UserID = &uid

		// Process payment based on user's default payment method
		// For simplicity, we'll use the same payment processing as regular payments
		// In a real system, you would use stored payment information

		// Generate payment (simulate auto-pay)
		// This would typically use a stored payment token
		var transactionID string
		var payMethod string

		// Simulate successful payment
		// In a real system, you would call the payment gateway with stored credentials
		transactionID = fmt.Sprintf("auto_pay_%s", uuid.New().String())
		payMethod = "auto"

		// Update order status
		now := time.Now()
		order.Status = string(StatusPaid)
		order.PayTime = &now
		order.PayMethod = payMethod
		order.TransactionID = transactionID
		order.PaidAmount = order.FinalAmount
		order.AutoPayStatus = "success"

		if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
			return fmt.Errorf("failed to update order: %w", err)
		}

		// Notify gate control to open the gate
		if uc.gateClient != nil {
			if err := uc.gateClient.OpenGate(ctx, order.RecordID.String()); err != nil {
				uc.log.WithContext(ctx).Errorf("failed to open gate: %v", err)
				// Continue even if gate opening fails
			}
		}

		// Send payment notification
		if uc.notificationClient != nil && order.UserID != nil {
			if err := uc.notificationClient.CreatePaymentNotification(ctx, order.UserID.String(), order.ID.String(), order.FinalAmount, "success"); err != nil {
				uc.log.WithContext(ctx).Errorf("failed to send payment notification: %v", err)
				// Continue even if notification fails
			}
		}

		return nil
	})

	return err
}

// processRefund processes the refund for the order.
func (uc *PaymentUseCase) processRefund(ctx context.Context, order *Order, refundID string) error {
	refundAmount := order.FinalAmount

	switch PayMethod(order.PayMethod) {
	case MethodWechat:
		if uc.wechatClient == nil {
			return fmt.Errorf("wechat client not configured")
		}
		uc.log.WithContext(ctx).Infof("Processing WeChat refund for order %s, amount: %.2f", order.ID, refundAmount)
		// Convert to cents
		totalAmount := int64(order.FinalAmount * 100)
		refundAmount := int64(refundAmount * 100)
		if err := uc.wechatClient.Refund(ctx, order.ID.String(), refundID, totalAmount, refundAmount); err != nil {
			uc.log.WithContext(ctx).Errorf("WeChat refund failed: %v", err)
			return fmt.Errorf("wechat refund failed: %w", err)
		}
	case MethodAlipay:
		if uc.alipayClient == nil {
			return fmt.Errorf("alipay client not configured")
		}
		uc.log.WithContext(ctx).Infof("Processing Alipay refund for order %s, amount: %.2f", order.ID, refundAmount)
		if err := uc.alipayClient.Refund(ctx, order.ID.String(), refundID, refundAmount); err != nil {
			uc.log.WithContext(ctx).Errorf("Alipay refund failed: %v", err)
			return fmt.Errorf("alipay refund failed: %w", err)
		}
	default:
		return fmt.Errorf("unknown payment method: %s", order.PayMethod)
	}

	now := time.Now()
	order.Status = string(StatusRefunded)
	order.RefundedAt = &now
	order.RefundTransactionID = refundID
	return nil
}

// Reconcile performs payment reconciliation with payment platforms.
// func (uc *PaymentUseCase) Reconcile(ctx context.Context, req *v1.ReconcileRequest) (*v1.ReconcileData, error) {
// 	date := req.Date
// 	payMethod := req.PayMethod

// 	// Create reconciliation record
// 	reconciliation := &Reconciliation{
// 		ID:        uuid.New(),
// 		Date:      date,
// 		PayMethod: payMethod,
// 		Status:    "pending",
// 	}

// 	if err := uc.reconciliationRepo.CreateReconciliation(ctx, reconciliation); err != nil {
// 		uc.log.WithContext(ctx).Errorf("Failed to create reconciliation record: %v", err)
// 		return nil, fmt.Errorf("failed to create reconciliation record: %w", err)
// 	}

// 	// Get platform transactions
// 	platformTransactions, err := uc.getPlatformTransactions(ctx, date, payMethod)
// 	if err != nil {
// 		uc.log.WithContext(ctx).Errorf("Failed to get platform transactions: %v", err)
// 		reconciliation.Status = "failed"
// 		if err := uc.reconciliationRepo.UpdateReconciliation(ctx, reconciliation); err != nil {
// 			uc.log.WithContext(ctx).Errorf("Failed to update reconciliation status: %v", err)
// 		}
// 		return nil, fmt.Errorf("failed to get platform transactions: %w", err)
// 	}

// 	// Get system orders
// 	systemOrders, err := uc.getSystemOrders(ctx, date, payMethod)
// 	if err != nil {
// 		uc.log.WithContext(ctx).Errorf("Failed to get system orders: %v", err)
// 		reconciliation.Status = "failed"
// 		if err := uc.reconciliationRepo.UpdateReconciliation(ctx, reconciliation); err != nil {
// 			uc.log.WithContext(ctx).Errorf("Failed to update reconciliation status: %v", err)
// 		}
// 		return nil, fmt.Errorf("failed to get system orders: %w", err)
// 	}

// 	// Perform reconciliation
// 	matchedOrders, exceptionOrders := uc.performReconciliation(ctx, reconciliation.ID, systemOrders, platformTransactions)

// 	// Update reconciliation status
// 	reconciliation.TotalOrders = len(systemOrders)
// 	reconciliation.MatchedOrders = matchedOrders
// 	reconciliation.ExceptionOrders = exceptionOrders

// 	if exceptionOrders == 0 {
// 		reconciliation.Status = "success"
// 	} else {
// 		reconciliation.Status = "partial"
// 	}

// 	if err := uc.reconciliationRepo.UpdateReconciliation(ctx, reconciliation); err != nil {
// 		uc.log.WithContext(ctx).Errorf("Failed to update reconciliation record: %v", err)
// 		return nil, fmt.Errorf("failed to update reconciliation record: %w", err)
// 	}

// 	// Create response manually since proto generation is not working
// 	return &v1.ReconcileData{
// 		ReconciliationId: reconciliation.ID.String(),
// 		Status:           reconciliation.Status,
// 		TotalOrders:      int32(reconciliation.TotalOrders),
// 		MatchedOrders:    int32(reconciliation.MatchedOrders),
// 		ExceptionOrders:  int32(reconciliation.ExceptionOrders),
// 	}, nil
// }

// GetReconciliationResult retrieves reconciliation result by ID.
// func (uc *PaymentUseCase) GetReconciliationResult(ctx context.Context, req *v1.GetReconciliationResultRequest) (*v1.ReconciliationResultData, error) {
// 	reconciliationID, err := uuid.Parse(req.ReconciliationId)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid reconciliation ID: %w", err)
// 	}

// 	reconciliation, err := uc.reconciliationRepo.GetReconciliation(ctx, reconciliationID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get reconciliation: %w", err)
// 	}

// 	exceptions, err := uc.reconciliationExceptionRepo.GetReconciliationExceptions(ctx, reconciliationID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get reconciliation exceptions: %w", err)
// 	}

// 	// Create response manually since proto generation is not working
// 	resultData := &v1.ReconciliationResultData{
// 		ReconciliationId: reconciliation.ID.String(),
// 		Date:             reconciliation.Date,
// 		PayMethod:        reconciliation.PayMethod,
// 		Status:           reconciliation.Status,
// 		TotalOrders:      int32(reconciliation.TotalOrders),
// 		MatchedOrders:    int32(reconciliation.MatchedOrders),
// 		ExceptionOrders:  int32(reconciliation.ExceptionOrders),
// 		Exceptions:       []*v1.ReconciliationException{},
// 		CreatedAt:        reconciliation.CreatedAt.Format(time.RFC3339),
// 	}

// 	for _, exception := range exceptions {
// 		exceptionItem := &v1.ReconciliationException{
// 			OrderId:         exception.OrderID.String(),
// 			PlatformOrderId: exception.PlatformOrderID,
// 			SystemAmount:    exception.SystemAmount,
// 			PlatformAmount:  exception.PlatformAmount,
// 			Status:          exception.Status,
// 			Reason:          exception.Reason,
// 		}
// 		resultData.Exceptions = append(resultData.Exceptions, exceptionItem)
// 	}

// 	return resultData, nil
// }

// ListReconciliationResults lists reconciliation results with pagination.
// func (uc *PaymentUseCase) ListReconciliationResults(ctx context.Context, req *v1.ListReconciliationResultsRequest) (*v1.ListReconciliationResultsData, error) {
// 	page := req.Page
// 	if page <= 0 {
// 		page = 1
// 	}

// 	pageSize := req.PageSize
// 	if pageSize <= 0 {
// 		pageSize = 10
// 	}

// 	reconciliations, total, err := uc.reconciliationRepo.ListReconciliations(ctx, req.StartDate, req.EndDate, page, pageSize)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to list reconciliations: %w", err)
// 	}

// 	// Create response manually since proto generation is not working
// 	resultData := &v1.ListReconciliationResultsData{
// 		Total: int32(total),
// 		Items: []*v1.ReconciliationResultSummary{},
// 	}

// 	for _, reconciliation := range reconciliations {
// 		summaryItem := &v1.ReconciliationResultSummary{
// 			ReconciliationId: reconciliation.ID.String(),
// 			Date:             reconciliation.Date,
// 			PayMethod:        reconciliation.PayMethod,
// 			Status:           reconciliation.Status,
// 			TotalOrders:      int32(reconciliation.TotalOrders),
// 			MatchedOrders:    int32(reconciliation.MatchedOrders),
// 			ExceptionOrders:  int32(reconciliation.ExceptionOrders),
// 			CreatedAt:        reconciliation.CreatedAt.Format(time.RFC3339),
// 		}
// 		resultData.Items = append(resultData.Items, summaryItem)
// 	}

// 	return resultData, nil
// }

// HandleReconciliationException handles reconciliation exception.
// func (uc *PaymentUseCase) HandleReconciliationException(ctx context.Context, req *v1.HandleReconciliationExceptionRequest) (*v1.HandleReconciliationExceptionData, error) {
// 	reconciliationID, err := uuid.Parse(req.ReconciliationId)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid reconciliation ID: %w", err)
// 	}

// 	orderID, err := uuid.Parse(req.OrderId)
// 	if err != nil {
// 		return nil, fmt.Errorf("invalid order ID: %w", err)
// 	}

// 	exceptions, err := uc.reconciliationExceptionRepo.GetReconciliationExceptions(ctx, reconciliationID)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to get reconciliation exceptions: %w", err)
// 	}

// 	var targetException *ReconciliationException
// 	for _, exception := range exceptions {
// 		if exception.OrderID == orderID {
// 			targetException = exception
// 			break
// 		}
// 	}

// 	if targetException == nil {
// 		return nil, fmt.Errorf("reconciliation exception not found")
// 	}

// 	// Handle exception based on action
// 	switch req.Action {
// 	case "confirm":
// 		// Confirm the order as correct
// 		targetException.Status = "handled"
// 		targetException.Action = "confirm"
// 		targetException.Remark = req.Remark
// 	case "refund":
// 		// Process refund
// 		order, err := uc.orderRepo.GetOrder(ctx, orderID)
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to get order: %w", err)
// 		}

// 		refundID := uuid.New().String()
// 		if err := uc.processRefund(ctx, order, refundID); err != nil {
// 			return nil, fmt.Errorf("failed to process refund: %w", err)
// 		}

// 		if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
// 			return nil, fmt.Errorf("failed to update order: %w", err)
// 		}

// 		targetException.Status = "handled"
// 		targetException.Action = "refund"
// 		targetException.Remark = req.Remark
// 	case "ignore":
// 		// Ignore the exception
// 		targetException.Status = "ignored"
// 		targetException.Action = "ignore"
// 		targetException.Remark = req.Remark
// 	default:
// 		return nil, fmt.Errorf("invalid action: %s", req.Action)
// 	}

// 	now := time.Now()
// 	targetException.HandledAt = &now

// 	if err := uc.reconciliationExceptionRepo.UpdateReconciliationException(ctx, targetException); err != nil {
// 		return nil, fmt.Errorf("failed to update reconciliation exception: %w", err)
// 	}

// 	// Create response manually since proto generation is not working
// 	return &v1.HandleReconciliationExceptionData{
// 		Status:  "success",
// 		OrderId: req.OrderId,
// 		Action:  req.Action,
// 	}, nil
// }

// getPlatformTransactions retrieves transactions from payment platforms.
func (uc *PaymentUseCase) getPlatformTransactions(ctx context.Context, date, payMethod string) ([]*PlatformTransaction, error) {
	var transactions []*PlatformTransaction

	if payMethod == "wechat" || payMethod == "all" {
		if uc.wechatClient != nil {
			// WeChat transactions are not implemented yet
		}
	}

	if payMethod == "alipay" || payMethod == "all" {
		if uc.alipayClient != nil {
			// Alipay transactions are not implemented yet
		}
	}

	return transactions, nil
}

// getSystemOrders retrieves system orders for the specified date and payment method.
func (uc *PaymentUseCase) getSystemOrders(ctx context.Context, date, payMethod string) ([]*Order, error) {
	// This is a placeholder implementation
	// In real implementation, you would query orders from the database
	// based on pay_time and pay_method
	return []*Order{}, nil
}

// performReconciliation performs the reconciliation between system orders and platform transactions.
func (uc *PaymentUseCase) performReconciliation(ctx context.Context, reconciliationID uuid.UUID, systemOrders []*Order, platformTransactions []*PlatformTransaction) (int, int) {
	matchedCount := 0
	exceptionCount := 0

	// Create a map for quick lookup of platform transactions
	platformTxMap := make(map[string]*PlatformTransaction)
	for _, tx := range platformTransactions {
		platformTxMap[tx.OrderID] = tx
	}

	for _, order := range systemOrders {
		if tx, exists := platformTxMap[order.ID.String()]; exists {
			// Check if amounts match
			if order.FinalAmount == tx.Amount {
				matchedCount++
			} else {
				// Amount mismatch
			exception := &ReconciliationException{
					ID:                uuid.New(),
					ReconciliationID:  reconciliationID,
					OrderID:           order.ID,
					PlatformOrderID:   tx.TransactionID,
					SystemAmount:      order.FinalAmount,
					PlatformAmount:    tx.Amount,
					Status:            "unhandled",
					Reason:            "Amount mismatch",
				}
				if err := uc.reconciliationExceptionRepo.CreateReconciliationException(ctx, exception); err != nil {
					uc.log.WithContext(ctx).Errorf("Failed to create reconciliation exception: %v", err)
				}
				exceptionCount++
			}
		} else {
			// Transaction not found in platform
			exception := &ReconciliationException{
				ID:                uuid.New(),
				ReconciliationID:  reconciliationID,
				OrderID:           order.ID,
				SystemAmount:      order.FinalAmount,
				PlatformAmount:    0,
				Status:            "unhandled",
				Reason:            "Transaction not found in platform",
			}
			if err := uc.reconciliationExceptionRepo.CreateReconciliationException(ctx, exception); err != nil {
				uc.log.WithContext(ctx).Errorf("Failed to create reconciliation exception: %v", err)
			}
			exceptionCount++
		}
	}

	// Check for platform transactions not in system
	for _, tx := range platformTransactions {
		found := false
		for _, order := range systemOrders {
			if order.ID.String() == tx.OrderID {
				found = true
				break
			}
		}
		if !found {
			// Transaction found in platform but not in system
			exception := &ReconciliationException{
				ID:                uuid.New(),
				ReconciliationID:  reconciliationID,
				PlatformOrderID:   tx.TransactionID,
				SystemAmount:      0,
				PlatformAmount:    tx.Amount,
				Status:            "unhandled",
				Reason:            "Transaction not found in system",
			}
			if err := uc.reconciliationExceptionRepo.CreateReconciliationException(ctx, exception); err != nil {
				uc.log.WithContext(ctx).Errorf("Failed to create reconciliation exception: %v", err)
			}
			exceptionCount++
		}
	}

	return matchedCount, exceptionCount
}

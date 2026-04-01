// Package biz provides business logic for the payment service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/payment/alipay"
	"github.com/xuanyiying/smart-park/internal/payment/wechat"
)

// ReconciliationStatus 对账状态
type ReconciliationStatus string

const (
	ReconciliationStatusPending   ReconciliationStatus = "pending"
	ReconciliationStatusMatched   ReconciliationStatus = "matched"
	ReconciliationStatusMismatch ReconciliationStatus = "mismatch"
	ReconciliationStatusMissing   ReconciliationStatus = "missing"
)

// ReconciliationRecord 对账记录
type ReconciliationRecord struct {
	ID                uuid.UUID
	OrderID           uuid.UUID
	PaymentMethod     string
	OrderAmount       float64
	PaidAmount        float64
	TransactionID     string
	ReconciliationTime time.Time
	Status            ReconciliationStatus
	Notes             string
}

// ReconciliationUseCase 对账用例
type ReconciliationUseCase struct {
	orderRepo    OrderRepo
	log          *log.Helper
	wechatClient *wechat.Client
	alipayClient *alipay.Client
}

// NewReconciliationUseCase 创建对账用例
func NewReconciliationUseCase(orderRepo OrderRepo, wechatClient *wechat.Client, alipayClient *alipay.Client, logger log.Logger) *ReconciliationUseCase {
	return &ReconciliationUseCase{
		orderRepo:    orderRepo,
		log:          log.NewHelper(logger),
		wechatClient: wechatClient,
		alipayClient: alipayClient,
	}
}

// ReconcileDaily 每日对账
func (uc *ReconciliationUseCase) ReconcileDaily(ctx context.Context, date time.Time) ([]*ReconciliationRecord, error) {
	uc.log.WithContext(ctx).Infof("开始对账单日: %s", date.Format("2006-01-02"))

	// 获取当天的订单
	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := startTime.Add(24 * time.Hour)

	orders, err := uc.orderRepo.GetOrdersByTimeRange(ctx, startTime, endTime)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取订单失败: %v", err)
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}

	uc.log.WithContext(ctx).Infof("获取到 %d 个订单", len(orders))

	// 执行对账
	reconciliationRecords := make([]*ReconciliationRecord, 0, len(orders))
	for _, order := range orders {
		if order.Status == string(StatusPaid) {
			record, err := uc.reconcileOrder(ctx, order, startTime, endTime)
			if err != nil {
				uc.log.WithContext(ctx).Errorf("对账订单 %s 失败: %v", order.ID, err)
				continue
			}
			reconciliationRecords = append(reconciliationRecords, record)
		}
	}

	// 检查是否有漏单
	missingRecords, err := uc.checkMissingOrders(ctx, startTime, endTime)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("检查漏单失败: %v", err)
	} else {
		reconciliationRecords = append(reconciliationRecords, missingRecords...)
	}

	uc.log.WithContext(ctx).Infof("对账完成，共处理 %d 条记录", len(reconciliationRecords))
	return reconciliationRecords, nil
}

// reconcileOrder 对账单个订单
func (uc *ReconciliationUseCase) reconcileOrder(ctx context.Context, order *Order, startTime, endTime time.Time) (*ReconciliationRecord, error) {
	record := &ReconciliationRecord{
		ID:                uuid.New(),
		OrderID:           order.ID,
		PaymentMethod:     order.PayMethod,
		OrderAmount:       order.FinalAmount,
		PaidAmount:        order.PaidAmount,
		TransactionID:     order.TransactionID,
		ReconciliationTime: time.Now(),
		Status:            ReconciliationStatusPending,
	}

	// 根据支付方式进行对账
	switch PayMethod(order.PayMethod) {
	case MethodWechat:
		if uc.wechatClient != nil {
			err := uc.reconcileWechatOrder(ctx, order, record, startTime, endTime)
			if err != nil {
				record.Status = ReconciliationStatusMismatch
				record.Notes = fmt.Sprintf("微信对账失败: %v", err)
			}
		} else {
			record.Status = ReconciliationStatusPending
			record.Notes = "微信客户端未配置"
		}
	case MethodAlipay:
		if uc.alipayClient != nil {
			err := uc.reconcileAlipayOrder(ctx, order, record, startTime, endTime)
			if err != nil {
				record.Status = ReconciliationStatusMismatch
				record.Notes = fmt.Sprintf("支付宝对账失败: %v", err)
			}
		} else {
			record.Status = ReconciliationStatusPending
			record.Notes = "支付宝客户端未配置"
		}
	default:
		record.Status = ReconciliationStatusPending
		record.Notes = "不支持的支付方式"
	}

	// 检查金额是否匹配
	if record.Status == ReconciliationStatusPending {
		diff := order.FinalAmount - order.PaidAmount
		if diff < -0.01 || diff > 0.01 {
			record.Status = ReconciliationStatusMismatch
			record.Notes = fmt.Sprintf("金额不匹配: 订单金额 %.2f, 实付金额 %.2f", order.FinalAmount, order.PaidAmount)
		} else {
			record.Status = ReconciliationStatusMatched
			record.Notes = "对账成功"
		}
	}

	return record, nil
}

// reconcileWechatOrder 微信订单对账
func (uc *ReconciliationUseCase) reconcileWechatOrder(ctx context.Context, order *Order, record *ReconciliationRecord, startTime, endTime time.Time) error {
	// 这里应该调用微信支付的查询接口，验证交易是否真实存在
	// 由于是模拟环境，我们简化处理
	uc.log.WithContext(ctx).Infof("对账微信订单: %s, 交易号: %s", order.ID, order.TransactionID)
	// 模拟对账成功
	return nil
}

// reconcileAlipayOrder 支付宝订单对账
func (uc *ReconciliationUseCase) reconcileAlipayOrder(ctx context.Context, order *Order, record *ReconciliationRecord, startTime, endTime time.Time) error {
	// 这里应该调用支付宝的查询接口，验证交易是否真实存在
	// 由于是模拟环境，我们简化处理
	uc.log.WithContext(ctx).Infof("对账支付宝订单: %s, 交易号: %s", order.ID, order.TransactionID)
	// 模拟对账成功
	return nil
}

// checkMissingOrders 检查是否有漏单
func (uc *ReconciliationUseCase) checkMissingOrders(ctx context.Context, startTime, endTime time.Time) ([]*ReconciliationRecord, error) {
	// 这里应该调用支付渠道的交易查询接口，获取当天的所有交易
	// 然后与系统中的订单进行比对，找出系统中缺失的订单
	// 由于是模拟环境，我们简化处理
	uc.log.WithContext(ctx).Info("检查漏单")
	return []*ReconciliationRecord{}, nil
}

// GetReconciliationReport 获取对账报表
func (uc *ReconciliationUseCase) GetReconciliationReport(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	// 获取对账期间的订单
	orders, err := uc.orderRepo.GetOrdersByTimeRange(ctx, startDate, endDate)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取订单失败: %v", err)
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}

	// 统计数据
	totalOrders := 0
	totalAmount := 0.0
	totalPaid := 0.0
	matchedOrders := 0
	mismatchedOrders := 0
	pendingOrders := 0

	paymentMethodStats := make(map[string]map[string]interface{})

	for _, order := range orders {
		if order.Status == string(StatusPaid) {
			totalOrders++
			totalAmount += order.FinalAmount
			totalPaid += order.PaidAmount

			// 检查金额是否匹配
			diff := order.FinalAmount - order.PaidAmount
			if diff < -0.01 || diff > 0.01 {
				mismatchedOrders++
			} else {
				matchedOrders++
			}

			// 按支付方式统计
			if _, ok := paymentMethodStats[order.PayMethod]; !ok {
				paymentMethodStats[order.PayMethod] = map[string]interface{}{
					"count":   0,
					"amount":  0.0,
					"paid":    0.0,
				}
			}
			paymentMethodStats[order.PayMethod]["count"] = paymentMethodStats[order.PayMethod]["count"].(int) + 1
			paymentMethodStats[order.PayMethod]["amount"] = paymentMethodStats[order.PayMethod]["amount"].(float64) + order.FinalAmount
			paymentMethodStats[order.PayMethod]["paid"] = paymentMethodStats[order.PayMethod]["paid"].(float64) + order.PaidAmount
		}
	}

	report := map[string]interface{}{
		"start_date":       startDate.Format("2006-01-02"),
		"end_date":         endDate.Format("2006-01-02"),
		"total_orders":     totalOrders,
		"total_amount":     totalAmount,
		"total_paid":       totalPaid,
		"matched_orders":   matchedOrders,
		"mismatched_orders": mismatchedOrders,
		"pending_orders":   pendingOrders,
		"payment_methods":  paymentMethodStats,
	}

	return report, nil
}

// FixMismatchedOrders 修复对账不匹配的订单
func (uc *ReconciliationUseCase) FixMismatchedOrders(ctx context.Context, orderIDs []string) error {
	for _, orderIDStr := range orderIDs {
		orderID, err := uuid.Parse(orderIDStr)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("无效的订单ID: %s", orderIDStr)
			continue
		}

		order, err := uc.orderRepo.GetOrder(ctx, orderID)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("获取订单失败: %v", err)
			continue
		}

		// 这里应该根据实际情况进行修复，例如更新订单金额、状态等
		uc.log.WithContext(ctx).Infof("修复订单: %s, 当前状态: %s, 订单金额: %.2f, 实付金额: %.2f",
			order.ID, order.Status, order.FinalAmount, order.PaidAmount)

		// 模拟修复
		order.Status = string(StatusPaid)
		if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
			uc.log.WithContext(ctx).Errorf("更新订单失败: %v", err)
		}
	}

	return nil
}

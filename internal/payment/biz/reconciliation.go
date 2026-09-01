// Package biz provides business logic for the payment service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	alipaysdk "github.com/smartwalle/alipay/v3"

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
	ID                 uuid.UUID
	OrderID            uuid.UUID
	PaymentMethod      string
	// Amounts in cents (分).
	OrderAmount        int64
	PaidAmount         int64
	TransactionID      string
	ReconciliationTime time.Time
	Status             ReconciliationStatus
	Notes              string
}

// Refunder performs a real refund against a payment gateway. It is satisfied by
// *PaymentUseCase and injected so reconciliation never has to fake a refund.
type Refunder interface {
	// RefundOrder charges the gateway back by amountCents and reports the gateway
	// transaction id. It must only return a nil error when the gateway confirmed.
	RefundOrder(ctx context.Context, order *Order, refundID string, amountCents int64) (string, error)
}

// ReconciliationUseCase 对账用例
type ReconciliationUseCase struct {
	orderRepo          OrderRepo
	reconciliationRepo ReconciliationRepo
	log                *log.Helper
	wechatClient       *wechat.Client
	alipayClient       *alipay.Client
	refunder           Refunder
}

// ReconciliationRepo 对账记录仓库接口
type ReconciliationRepo interface {
	// CreateReconciliation 创建对账记录
	CreateReconciliation(ctx context.Context, record *ReconciliationRecord) error
	// GetReconciliationByOrderID 根据订单ID获取对账记录
	GetReconciliationByOrderID(ctx context.Context, orderID uuid.UUID) (*ReconciliationRecord, error)
	// GetReconciliationByTimeRange 根据时间范围获取对账记录
	GetReconciliationByTimeRange(ctx context.Context, startTime, endTime time.Time) ([]*ReconciliationRecord, error)
	// UpdateReconciliation 更新对账记录
	UpdateReconciliation(ctx context.Context, record *ReconciliationRecord) error
	// GetMismatchedReconciliations 获取对账不匹配的记录
	GetMismatchedReconciliations(ctx context.Context, startTime, endTime time.Time) ([]*ReconciliationRecord, error)
}

// NewReconciliationUseCase 创建对账用例
func NewReconciliationUseCase(orderRepo OrderRepo, reconciliationRepo ReconciliationRepo, wechatClient *wechat.Client, alipayClient *alipay.Client, refunder Refunder, logger log.Logger) *ReconciliationUseCase {
	return &ReconciliationUseCase{
		orderRepo:          orderRepo,
		reconciliationRepo: reconciliationRepo,
		log:                log.NewHelper(logger),
		wechatClient:       wechatClient,
		alipayClient:       alipayClient,
		refunder:           refunder,
	}
}

// ReconcileDaily 每日对账
func (uc *ReconciliationUseCase) ReconcileDaily(ctx context.Context, date time.Time) ([]*ReconciliationRecord, error) {
	uc.log.WithContext(ctx).Infof("开始对账单日: %s", date.Format("2006-01-02"))

	startTime := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endTime := startTime.Add(24 * time.Hour)

	orders, err := uc.orderRepo.GetOrdersByTimeRange(ctx, startTime, endTime)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取订单失败: %v", err)
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}

	uc.log.WithContext(ctx).Infof("获取到 %d 个订单", len(orders))

	reconciliationRecords := make([]*ReconciliationRecord, 0, len(orders))
	for _, order := range orders {
		if order.Status != string(StatusPaid) {
			continue
		}

		record, err := uc.reconcileOrder(ctx, order)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("对账订单 %s 失败: %v", order.ID, err)
			continue
		}
		if err := uc.reconciliationRepo.CreateReconciliation(ctx, record); err != nil {
			uc.log.WithContext(ctx).Errorf("存储对账记录失败: %v", err)
		}
		reconciliationRecords = append(reconciliationRecords, record)
	}

	// 检查是否有漏单：渠道已收款但本地仍停留在 pending。
	missingRecords, err := uc.checkMissingOrders(ctx, startTime, endTime)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("检查漏单失败: %v", err)
	} else {
		for _, record := range missingRecords {
			if err := uc.reconciliationRepo.CreateReconciliation(ctx, record); err != nil {
				uc.log.WithContext(ctx).Errorf("存储漏单记录失败: %v", err)
			}
		}
		reconciliationRecords = append(reconciliationRecords, missingRecords...)
	}

	uc.log.WithContext(ctx).Infof("对账完成，共处理 %d 条记录", len(reconciliationRecords))
	return reconciliationRecords, nil
}

// reconcileOrder 对账单个订单：以渠道返回为准核验本地方状态。
func (uc *ReconciliationUseCase) reconcileOrder(ctx context.Context, order *Order) (*ReconciliationRecord, error) {
	record := &ReconciliationRecord{
		ID:                 uuid.New(),
		OrderID:            order.ID,
		PaymentMethod:      order.PayMethod,
		OrderAmount:        order.FinalAmount,
		PaidAmount:         order.PaidAmount,
		TransactionID:      order.TransactionID,
		ReconciliationTime: time.Now(),
		Status:             ReconciliationStatusPending,
	}

	// order amounts are already cents (分).
	expectedCents := order.FinalAmount

	switch PayMethod(order.PayMethod) {
	case MethodWechat:
		if uc.wechatClient == nil {
			record.Status = ReconciliationStatusPending
			record.Notes = "微信客户端未配置，无法与渠道核验"
			return record, nil
		}
		if err := uc.reconcileWechatOrder(ctx, order, expectedCents, record); err != nil {
			record.Status = ReconciliationStatusMismatch
			record.Notes = fmt.Sprintf("微信对账失败: %v", err)
		}
	case MethodAlipay:
		if uc.alipayClient == nil {
			record.Status = ReconciliationStatusPending
			record.Notes = "支付宝客户端未配置，无法与渠道核验"
			return record, nil
		}
		if err := uc.reconcileAlipayOrder(ctx, order, expectedCents, record); err != nil {
			record.Status = ReconciliationStatusMismatch
			record.Notes = fmt.Sprintf("支付宝对账失败: %v", err)
		}
	default:
		record.Status = ReconciliationStatusPending
		record.Notes = "不支持的支付方式"
		return record, nil
	}

	// 本地侧自洽性检查：订单金额与实付金额必须一致（整数分比较）。
	if record.Status == ReconciliationStatusPending {
		paidCents := order.PaidAmount
		if paidCents != expectedCents {
			record.Status = ReconciliationStatusMismatch
			record.Notes = fmt.Sprintf("金额不匹配: 订单金额 %d, 实付金额 %d", order.FinalAmount, order.PaidAmount)
		} else {
			record.Status = ReconciliationStatusMatched
			record.Notes = "对账成功"
		}
	}

	return record, nil
}

// reconcileWechatOrder 微信订单对账：调用微信查单接口，确认交易真实存在且金额一致。
func (uc *ReconciliationUseCase) reconcileWechatOrder(ctx context.Context, order *Order, expectedCents int64, record *ReconciliationRecord) error {
	result, err := uc.wechatClient.QueryOrder(ctx, order.ID.String())
	if err != nil {
		return fmt.Errorf("渠道查单失败: %w", err)
	}

	state, _ := result["trade_state"].(string)
	if state == "" {
		return fmt.Errorf("渠道未返回 trade_state")
	}
	if state != wechat.TradeStateSuccess {
		return fmt.Errorf("渠道交易状态为 %s，本地却标记为已支付", state)
	}

	// 本地 out_trade_no 即订单号，核验收到的订单号与本地一致，防止串单。
	if got, _ := result["out_trade_no"].(string); got != "" && got != order.ID.String() {
		return fmt.Errorf("渠道返回的 out_trade_no %s 与订单 %s 不一致", got, order.ID)
	}

	// 渠道金额核验：支付成功的订单，渠道实收金额必须等于订单金额。金额不符是必须
	// 报警的对账差异，而不是可以静默通过的本地字段对比。
	if totalFee, ok := result["total_fee"].(int64); ok && totalFee != expectedCents {
		return fmt.Errorf("金额不一致: 渠道 %d 分, 本地 %d 分", totalFee, expectedCents)
	}

	record.Notes = fmt.Sprintf("渠道核验通过: trade_state=%s", state)
	return nil
}

// reconcileAlipayOrder 支付宝订单对账：调用支付宝查单接口核验收款状态与金额。
func (uc *ReconciliationUseCase) reconcileAlipayOrder(ctx context.Context, order *Order, expectedCents int64, record *ReconciliationRecord) error {
	rsp, err := uc.alipayClient.QueryOrder(ctx, order.ID.String())
	if err != nil {
		return fmt.Errorf("渠道查单失败: %w", err)
	}
	if rsp == nil {
		return fmt.Errorf("渠道未返回交易信息")
	}
	if rsp.Code != "10000" {
		return fmt.Errorf("渠道返回错误: %s - %s", rsp.Code, rsp.Msg)
	}

	// TRADE_SUCCESS / TRADE_FINISHED 表示收款成功；WAIT_BUYER_PAY 表示未付款；
	// TRADE_CLOSED 表示已关闭或全额退款。本地标记已支付但渠道未收款，是必须报警的状态。
	if rsp.TradeStatus != alipaysdk.TradeStatusSuccess && rsp.TradeStatus != alipaysdk.TradeStatusFinished {
		return fmt.Errorf("渠道交易状态为 %s，本地却标记为已支付", rsp.TradeStatus)
	}

	gotCents, err := parseYuanToCents(rsp.TotalAmount)
	if err != nil {
		return fmt.Errorf("渠道返回金额非法: %w", err)
	}
	if gotCents != expectedCents {
		return fmt.Errorf("金额不一致: 渠道 %d 分, 本地 %d 分", gotCents, expectedCents)
	}

	record.Notes = fmt.Sprintf("渠道核验通过: trade_status=%s, amount=%d分", rsp.TradeStatus, gotCents)
	return nil
}

// checkMissingOrders 检查是否有漏单：本地仍为 pending，但渠道实际已完成收款。
//
// 这是回调丢失（网络抖动、服务重启、渠道重试耗尽）时唯一的补救发现手段。发现后
// 生成 missing 记录，由运维或 SweepPendingOrders 跟进补单。
func (uc *ReconciliationUseCase) checkMissingOrders(ctx context.Context, startTime, endTime time.Time) ([]*ReconciliationRecord, error) {
	const sweepLimit = 500

	pending, err := uc.orderRepo.ListOrdersByStatus(ctx, string(StatusPending), time.Now(), sweepLimit)
	if err != nil {
		return nil, err
	}

	records := make([]*ReconciliationRecord, 0)
	for _, order := range pending {
		if order.CreatedAt.Before(startTime) || !order.CreatedAt.Before(endTime) {
			continue
		}

		paid, transactionID, err := uc.confirmWithGateway(ctx, order)
		if err != nil {
			uc.log.WithContext(ctx).Warnf("补单检查失败 order=%s: %v", order.ID, err)
			continue
		}
		if !paid {
			continue
		}

		uc.log.WithContext(ctx).Warnf("发现漏单: order=%s 渠道已收款但本地仍为 pending, transaction=%s", order.ID, transactionID)
		records = append(records, &ReconciliationRecord{
			ID:                 uuid.New(),
			OrderID:            order.ID,
			PaymentMethod:      order.PayMethod,
			OrderAmount:        order.FinalAmount,
			PaidAmount:         order.FinalAmount,
			TransactionID:      transactionID,
			ReconciliationTime: time.Now(),
			Status:             ReconciliationStatusMissing,
			Notes:              "渠道已收款但本地订单未结算，需要补单",
		})
	}

	return records, nil
}

// confirmWithGateway asks the gateway whether an order was actually paid.
func (uc *ReconciliationUseCase) confirmWithGateway(ctx context.Context, order *Order) (bool, string, error) {
	switch PayMethod(order.PayMethod) {
	case MethodWechat:
		if uc.wechatClient == nil {
			return false, "", fmt.Errorf("微信支付客户端未配置，无法查单")
		}
		result, err := uc.wechatClient.QueryOrder(ctx, order.ID.String())
		if err != nil {
			return false, "", err
		}
		if state, _ := result["trade_state"].(string); state != wechat.TradeStateSuccess {
			return false, "", nil
		}
		// Prefer the channel's transaction id over the local record: on a lost callback
		// the local field is empty, and the settlement row must carry a real reference.
		transactionID, _ := result["transaction_id"].(string)
		return true, transactionID, nil
	case MethodAlipay:
		if uc.alipayClient == nil {
			return false, "", fmt.Errorf("支付宝客户端未配置，无法查单")
		}
		rsp, err := uc.alipayClient.QueryOrder(ctx, order.ID.String())
		if err != nil || rsp == nil {
			return false, "", err
		}
		if rsp.TradeStatus != alipaysdk.TradeStatusSuccess && rsp.TradeStatus != alipaysdk.TradeStatusFinished {
			return false, "", nil
		}
		return true, rsp.TradeNo, nil
	default:
		return false, "", nil
	}
}

// GetReconciliationReport 获取对账报表
func (uc *ReconciliationUseCase) GetReconciliationReport(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	orders, err := uc.orderRepo.GetOrdersByTimeRange(ctx, startDate, endDate)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("获取订单失败: %v", err)
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}

	totalOrders := 0
	// Amounts are cents (分) end to end; aggregating int64 avoids float drift.
	var totalAmount int64
	var totalPaid int64
	matchedOrders := 0
	mismatchedOrders := 0

	paymentMethodStats := make(map[string]map[string]interface{})

	for _, order := range orders {
		if order.Status != string(StatusPaid) {
			continue
		}
		totalOrders++
		totalAmount += order.FinalAmount
		totalPaid += order.PaidAmount

		if order.FinalAmount != order.PaidAmount {
			mismatchedOrders++
		} else {
			matchedOrders++
		}

		if _, ok := paymentMethodStats[order.PayMethod]; !ok {
			paymentMethodStats[order.PayMethod] = map[string]interface{}{
				"count":  0,
				"amount": int64(0),
				"paid":   int64(0),
			}
		}
		paymentMethodStats[order.PayMethod]["count"] = paymentMethodStats[order.PayMethod]["count"].(int) + 1
		paymentMethodStats[order.PayMethod]["amount"] = paymentMethodStats[order.PayMethod]["amount"].(int64) + order.FinalAmount
		paymentMethodStats[order.PayMethod]["paid"] = paymentMethodStats[order.PayMethod]["paid"].(int64) + order.PaidAmount
	}

	return map[string]interface{}{
		"start_date":        startDate.Format("2006-01-02"),
		"end_date":          endDate.Format("2006-01-02"),
		"total_orders":      totalOrders,
		"total_amount":      totalAmount,
		"total_paid":        totalPaid,
		"matched_orders":    matchedOrders,
		"mismatched_orders": mismatchedOrders,
		"payment_methods":   paymentMethodStats,
	}, nil
}

// FixMismatchedOrders 修复对账不匹配的订单
//
// 差额退款必须通过真实网关退款完成；只有在渠道确认退款后才会改写订单状态。退款失败
// 的订单保持 mismatch，交由人工处理，绝不谎报成功。
func (uc *ReconciliationUseCase) FixMismatchedOrders(ctx context.Context, orderIDs []string) error {
	if uc.refunder == nil {
		return fmt.Errorf("退款服务未配置，拒绝修复对账差异")
	}
	if uc.reconciliationRepo == nil {
		return fmt.Errorf("对账记录仓库未配置，拒绝修复对账差异")
	}

	for _, orderIDStr := range orderIDs {
		orderID, err := uuid.Parse(orderIDStr)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("无效的订单ID: %s", orderIDStr)
			continue
		}

		order, err := uc.orderRepo.GetOrder(ctx, orderID)
		if err != nil || order == nil {
			uc.log.WithContext(ctx).Errorf("获取订单失败: %v", err)
			continue
		}

		reconciliation, err := uc.reconciliationRepo.GetReconciliationByOrderID(ctx, orderID)
		if err != nil {
			uc.log.WithContext(ctx).Errorf("获取对账记录失败: %v", err)
			continue
		}

		expectedCents := order.FinalAmount
		paidCents := order.PaidAmount

		switch {
		case paidCents > expectedCents:
			// 多收了钱：必须把差额真实退给用户。
			//
			// 注意方向与直觉相反：退款金额是"实付 - 应收"，而不是"应收 - 实付"。
			// 原实现把两者弄反了，导致少付的订单被"退款"，多付的订单被静默抹平。
			overchargeCents := paidCents - expectedCents
			uc.log.WithContext(ctx).Infof("订单 %s 多收 %d 分，执行退款", order.ID, overchargeCents)

			refundTxID, err := uc.refunder.RefundOrder(ctx, order, uuid.New().String(), overchargeCents)
			if err != nil {
				// 退款未成功，订单状态与对账记录都必须保持原样。
				uc.log.WithContext(ctx).Errorf("执行退款失败: %v", err)
				uc.markReconciliation(ctx, reconciliation, ReconciliationStatusMismatch,
					fmt.Sprintf("退款失败，需人工处理: %v", err))
				continue
			}

			order.Status = string(StatusRefunded)
			order.RefundTransactionID = refundTxID
			now := time.Now()
			order.RefundedAt = &now
			if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
				uc.log.WithContext(ctx).Errorf("更新订单状态失败: %v", err)
				continue
			}
			uc.markReconciliation(ctx, reconciliation, ReconciliationStatusMatched,
				fmt.Sprintf("已退款 %d 分，渠道交易号 %s", overchargeCents, refundTxID))

		case paidCents < expectedCents:
			// 少收了钱：属于需要人工追收的业务差异。
			//
			// 绝不能用"把订单金额改成实付金额"来抹平差异 —— 那会让账目看起来平了，
			// 而真实的资金缺口从此消失在报表里。
			shortfallCents := expectedCents - paidCents
			uc.log.WithContext(ctx).Warnf("订单 %s 少收 %d 分，需人工追收", order.ID, shortfallCents)
			uc.markReconciliation(ctx, reconciliation, ReconciliationStatusMismatch,
				fmt.Sprintf("少收 %d 分，需人工追收", shortfallCents))

		default:
			uc.markReconciliation(ctx, reconciliation, ReconciliationStatusMatched, "金额匹配，对账成功")
		}
	}

	return nil
}

func (uc *ReconciliationUseCase) markReconciliation(ctx context.Context, record *ReconciliationRecord, status ReconciliationStatus, notes string) {
	if record == nil {
		return
	}
	record.Status = status
	record.Notes = notes
	record.ReconciliationTime = time.Now()
	if err := uc.reconciliationRepo.UpdateReconciliation(ctx, record); err != nil {
		uc.log.WithContext(ctx).Errorf("更新对账记录失败: %v", err)
	}
}

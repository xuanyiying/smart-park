package biz

import (
	"context"
	"fmt"
	"time"
)

// SweepResult summarises one pass of the order sweeper.
type SweepResult struct {
	// Closed is the number of expired orders transitioned to failed.
	Closed int
	// Settled is the number of orders the gateway confirmed as paid even though no
	// callback ever arrived (lost callback, restart, exhausted retries).
	Settled int
	// Errors is the number of orders that could not be processed this pass and will be
	// retried on the next tick.
	Errors int
}

// sweepBatchSize bounds each pass so a large backlog cannot hold the loop forever.
const sweepBatchSize = 500

// SweepExpiredOrders performs one maintenance pass over pending orders.
//
// It closes the two gaps that callbacks alone cannot cover:
//
//  1. Expiry — an order nobody paid must not sit in pending forever. Closing it releases
//     the parking session and stops the exit barrier from waiting on it.
//  2. Lost callbacks — the gateway is the source of truth. If the driver paid but our
//     callback never arrived, the only way to find out is to ask, and the money must be
//     recorded: the driver is entitled to drive out.
//
// Both paths settle through the same conditional updates the callback path uses, so a
// callback racing the sweeper cannot double-settle an order.
func (uc *PaymentUseCase) SweepExpiredOrders(ctx context.Context) (SweepResult, error) {
	result := SweepResult{}

	expired, err := uc.orderRepo.ListOrdersByStatus(ctx, string(StatusPending), time.Now().Add(-uc.bizConfig.OrderExpiration), sweepBatchSize)
	if err != nil {
		return result, fmt.Errorf("查询超时订单失败: %w", err)
	}

	for _, order := range expired {
		paid, transactionID, err := uc.confirmWithGatewayOrder(ctx, order)
		if err != nil {
			// Network and gateway errors are transient; leave the order pending and retry
			// on the next pass rather than closing an order that might actually be paid.
			uc.log.WithContext(ctx).Warnf("订单 %s 查单失败，保留 pending: %v", order.ID, err)
			result.Errors++
			continue
		}

		if paid {
			settled, err := uc.settleFromSweep(ctx, order, transactionID)
			if err != nil {
				uc.log.WithContext(ctx).Errorf("订单 %s 补单失败: %v", order.ID, err)
				result.Errors++
				continue
			}
			if settled {
				result.Settled++
			}
			continue
		}

		closed, err := uc.orderRepo.MarkOrderClosed(ctx, order.ID, time.Now())
		if err != nil {
			uc.log.WithContext(ctx).Errorf("订单 %s 关闭失败: %v", order.ID, err)
			result.Errors++
			continue
		}
		if closed {
			uc.log.WithContext(ctx).Infof("订单 %s 超时未支付，已关闭", order.ID)
			result.Closed++
		}
	}

	return result, nil
}

// confirmWithGatewayOrder asks the payment channel whether the order was actually paid.
// It wraps confirmWithGateway so the sweeper owns the "channel not configured" decision.
func (uc *PaymentUseCase) confirmWithGatewayOrder(ctx context.Context, order *Order) (bool, string, error) {
	if uc.queryGateway != nil {
		return uc.queryGateway(ctx, order)
	}
	recon := &ReconciliationUseCase{
		orderRepo:    uc.orderRepo,
		wechatClient: uc.wechatClient,
		alipayClient: uc.alipayClient,
		log:          uc.log,
	}
	return recon.confirmWithGateway(ctx, order)
}

// settleFromSweep records a gateway-confirmed payment that no callback delivered.
//
// MarkOrderPaid is the same conditional update callbacks use, so if a callback did land
// between the query and this write, it wins and the sweep simply reports nothing settled.
func (uc *PaymentUseCase) settleFromSweep(ctx context.Context, order *Order, transactionID string) (bool, error) {
	paidAmount := order.FinalAmount
	ok, err := uc.orderRepo.MarkOrderPaid(ctx, order.ID, order.PayMethod, transactionID, paidAmount, time.Now())
	if err != nil {
		return false, err
	}
	if !ok {
		// Someone else (a late callback, another replica) already settled it.
		return false, nil
	}

	uc.log.WithContext(ctx).Warnf("补单: order=%s 渠道已收款但未收到回调, transaction=%s, amount=%.2f",
		order.ID, transactionID, paidAmount)

	// Read the authoritative order before opening the gate, mirroring the callback path.
	updated, err := uc.orderRepo.GetOrder(ctx, order.ID)
	if err != nil || updated == nil {
		return true, nil
	}
	if updated.Status == string(StatusPaid) {
		if err := uc.triggerAutoGateOpen(ctx, updated); err != nil {
			// The money is recorded; gate release can be retried by staff, so this is not
			// a settlement failure.
			uc.log.WithContext(ctx).Warnf("补单后自动开闸失败 order=%s: %v", order.ID, err)
		}
	}
	return true, nil
}

// RunOrderSweeper drives SweepExpiredOrders on an interval until stop is closed.
//
// Without this loop none of the above ever executes: closing stale orders and reconciling
// missing money existed as callable code with no caller, which is the same as not
// existing. Interval values below a second fall back to a sane default.
func (uc *PaymentUseCase) RunOrderSweeper(ctx context.Context, interval time.Duration, stop <-chan struct{}) {
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	uc.log.Infof("订单巡检调度器已启动，间隔 %s，订单有效期 %s", interval, uc.bizConfig.OrderExpiration)

	for {
		select {
		case <-stop:
			uc.log.Info("订单巡检调度器已停止")
			return
		case <-ctx.Done():
			uc.log.Info("订单巡检调度器随服务退出")
			return
		case <-ticker.C:
		}

		// Each pass gets its own timeout so a hung gateway query cannot stall the loop.
		sweepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		result, err := uc.SweepExpiredOrders(sweepCtx)
		cancel()
		if err != nil {
			uc.log.Errorf("订单巡检失败: %v", err)
			continue
		}
		if result.Closed > 0 || result.Settled > 0 || result.Errors > 0 {
			uc.log.Infof("订单巡检完成: 关闭 %d, 补单 %d, 待重试 %d", result.Closed, result.Settled, result.Errors)
		}
	}
}

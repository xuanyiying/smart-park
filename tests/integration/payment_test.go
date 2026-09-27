//go:build integration

package integration_test

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	paymentbiz "github.com/xuanyiying/smart-park/internal/payment/biz"
)

// TestPaymentOrderIdempotentSettlement 验证支付回调幂等落库：
// 首次 MarkOrderPaid 成功，重复回调不改变状态，并发回调恰好一笔成交。
func TestPaymentOrderIdempotentSettlement(t *testing.T) {
	repo, ctx := newPaymentStack(t)

	// transaction_id 无唯一约束且测试库跨次运行持久化，交易号必须每次运行唯一
	txnID := fmt.Sprintf("txn-itest-%s", uuid.New().String()[:8])

	order := &paymentbiz.Order{
		ID: uuid.New(), RecordID: uuid.New(), LotID: uuid.New(),
		PlateNumber: "京B67890",
		Amount:      1000, DiscountAmount: 100, FinalAmount: 900,
	}
	require.NoError(t, repo.CreateOrder(ctx, order))

	paidAt := time.Now()
	applied, err := repo.MarkOrderPaid(ctx, order.ID, "wechat", txnID, 900, paidAt)
	require.NoError(t, err)
	require.True(t, applied, "首次回调必须成交")

	// 重复回调：订单已非 pending，条件更新不再命中，返回 false
	applied, err = repo.MarkOrderPaid(ctx, order.ID, "wechat", txnID, 900, paidAt)
	require.NoError(t, err)
	require.False(t, applied, "重复回调必须被幂等拦截")

	got, err := repo.GetOrder(ctx, order.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "paid", got.Status)
	require.Equal(t, "wechat", got.PayMethod)
	require.Equal(t, txnID, got.TransactionID)
	require.Equal(t, int64(900), got.PaidAmount)
	require.NotNil(t, got.PayTime)

	// 不存在的订单：返回 false 而非报错
	applied, err = repo.MarkOrderPaid(ctx, uuid.New(), "alipay", "txn-itest-ghost", 100, time.Now())
	require.NoError(t, err)
	require.False(t, applied)

	// 按交易号可查询
	byTxn, err := repo.GetOrderByTransactionID(ctx, txnID)
	require.NoError(t, err)
	require.NotNil(t, byTxn)
	require.Equal(t, order.ID, byTxn.ID)
}

// TestPaymentOrderConcurrentCallbacks 用 16 个并发回调模拟渠道重复推送，
// 在 -race 下验证恰好只有一笔成交。
func TestPaymentOrderConcurrentCallbacks(t *testing.T) {
	repo, ctx := newPaymentStack(t)

	order := &paymentbiz.Order{
		ID: uuid.New(), RecordID: uuid.New(), LotID: uuid.New(),
		PlateNumber: "京C88888",
		Amount:      500, FinalAmount: 500,
	}
	require.NoError(t, repo.CreateOrder(ctx, order))

	const n = 16
	txnPrefix := uuid.New().String()[:8]
	var wg sync.WaitGroup
	wins := make(chan bool, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ok, err := repo.MarkOrderPaid(ctx, order.ID, "alipay", fmt.Sprintf("txn-conc-%s-%d", txnPrefix, i), order.FinalAmount, time.Now())
			if err == nil {
				wins <- ok
			}
		}(i)
	}
	wg.Wait()
	close(wins)

	successes := 0
	for ok := range wins {
		if ok {
			successes++
		}
	}
	require.Equal(t, 1, successes, "并发重复回调必须恰好成交一笔")

	got, err := repo.GetOrder(ctx, order.ID)
	require.NoError(t, err)
	require.Equal(t, "paid", got.Status)
	require.Equal(t, order.FinalAmount, got.PaidAmount)
}

//go:build integration

package integration_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	chargingbiz "github.com/xuanyiying/smart-park/internal/charging/biz"
)

// TestChargingSessionBillingFlow 走通真实数据库上的完整充电计费链路：
// 建站 → 建桩 → 定价 → 开始充电 → 上报电量 → 结束计费 → 恢复可用 → 确认支付。
// 金额单位为分。
func TestChargingSessionBillingFlow(t *testing.T) {
	uc, ctx := newChargingStack(t)

	station, err := uc.CreateStation(ctx, uuid.New(), "集成测试站", chargingbiz.ConnectorTypeAC, chargingbiz.ConnectorTypeAC, 7, 220, 2, "B1", "F1")
	require.NoError(t, err)

	// 峰时价：8-9 点（isPeakHours 判定 8 点起步、9 点结束在峰时段内）
	price, err := uc.CreatePrice(ctx, station.ID, "峰时电价", 8, 9, 1.0, 0.5, 0.4, 0.2, time.Now().Add(-time.Hour), time.Time{})
	require.NoError(t, err)
	require.True(t, price.IsPeakHours)

	connector, err := uc.CreateConnector(ctx, station.ID, 1, chargingbiz.ConnectorTypeAC, 7, 220)
	require.NoError(t, err)

	userID := uuid.New()
	session, err := uc.StartCharging(ctx, station.ID, connector.ID, userID, "京A12345")
	require.NoError(t, err)
	require.Equal(t, chargingbiz.SessionStatusCharging, session.Status)

	availableAfterStart := func() int {
		st, err := uc.GetStation(ctx, station.ID)
		require.NoError(t, err)
		return st.AvailableConnectors
	}
	require.Equal(t, 1, availableAfterStart())

	// 上报 10 kWh 电量
	require.NoError(t, uc.UpdateChargingProgress(ctx, session.ID, 10.0, 7, 220, 32))

	// 同一枪充电中时拒绝重复开始
	_, err = uc.StartCharging(ctx, station.ID, connector.ID, uuid.New(), "京B00000")
	require.Error(t, err)

	// 非本人结束被拒绝
	_, err = uc.StopCharging(ctx, session.ID, uuid.New())
	require.Error(t, err)

	stopped, err := uc.StopCharging(ctx, session.ID, userID)
	require.NoError(t, err)
	require.Equal(t, chargingbiz.SessionStatusCompleted, stopped.Status)
	require.Equal(t, 10.0, stopped.ChargedEnergy)

	// 电费 = 10kWh × (1.0 元/kWh + 0.4 元峰时附加) = 14 元 = 1400 分
	// 服务费 = 0.5 元 = 50 分；合计 1450 分
	require.Equal(t, int64(1400), stopped.Cost)
	require.Equal(t, int64(50), stopped.ServiceFee)
	require.Equal(t, int64(1450), stopped.TotalAmount)

	// 结束后枪位与站点可用数恢复
	conn, err := uc.GetConnector(ctx, connector.ID)
	require.NoError(t, err)
	require.Equal(t, chargingbiz.ConnectorStatusAvailable, conn.Status)
	st, err := uc.GetStation(ctx, station.ID)
	require.NoError(t, err)
	require.Equal(t, 2, st.AvailableConnectors)

	// 重复结束、金额不足支付被拒绝；足额支付后为已支付
	_, err = uc.StopCharging(ctx, session.ID, userID)
	require.Error(t, err)
	require.Error(t, uc.ConfirmPayment(ctx, session.ID, "txn-charging-1", "wechat", stopped.TotalAmount-1))
	require.NoError(t, uc.ConfirmPayment(ctx, session.ID, "txn-charging-1", "wechat", stopped.TotalAmount))

	final, err := uc.GetSession(ctx, session.ID)
	require.NoError(t, err)
	require.Equal(t, chargingbiz.PaymentStatusPaid, final.PaymentStatus)
	require.Equal(t, "txn-charging-1", final.TransactionID)
	require.NotNil(t, final.PayTime)

	// 已支付后再次确认支付被拒绝
	require.Error(t, uc.ConfirmPayment(ctx, session.ID, "txn-charging-2", "wechat", stopped.TotalAmount))
}

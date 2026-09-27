//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	chargingbiz "github.com/xuanyiying/smart-park/internal/charging/biz"
	"github.com/xuanyiying/smart-park/pkg/tenant"
)

// TestChargingTenantIsolation 验证多租户数据隔离：
// 租户 A 与租户 B 的站点、会话互相不可见，跨租户访问返回未找到。
func TestChargingTenantIsolation(t *testing.T) {
	uc, ctxA := newChargingStack(t) // ctxA 已绑定租户 A

	tenantB := uuid.New()
	ctxB := tenant.WithTenant(context.Background(), &tenant.TenantInfo{
		ID:   tenantB,
		Code: "itest-b-" + tenantB.String()[:8],
		Name: "隔离测试租户B",
	})

	stationA, err := uc.CreateStation(ctxA, uuid.New(), "租户A站点", chargingbiz.ConnectorTypeAC, chargingbiz.ConnectorTypeAC, 7, 220, 1, "B1", "F1")
	require.NoError(t, err)
	stationB, err := uc.CreateStation(ctxB, uuid.New(), "租户B站点", chargingbiz.ConnectorTypeDC, chargingbiz.ConnectorTypeDC, 60, 380, 2, "B2", "F2")
	require.NoError(t, err)

	t.Run("列表只返回本租户站点", func(t *testing.T) {
		listA, err := uc.ListStations(ctxA, uuid.Nil)
		require.NoError(t, err)
		require.Len(t, listA, 1)
		require.Equal(t, stationA.ID, listA[0].ID)

		listB, err := uc.ListStations(ctxB, uuid.Nil)
		require.NoError(t, err)
		require.Len(t, listB, 1)
		require.Equal(t, stationB.ID, listB[0].ID)
	})

	t.Run("跨租户读取返回未找到", func(t *testing.T) {
		_, err := uc.GetStation(ctxA, stationB.ID)
		require.ErrorIs(t, err, chargingbiz.ErrStationNotFound)

		_, err = uc.GetStation(ctxB, stationA.ID)
		require.ErrorIs(t, err, chargingbiz.ErrStationNotFound)
	})

	t.Run("充电会话按租户隔离", func(t *testing.T) {
		connB, err := uc.CreateConnector(ctxB, stationB.ID, 1, chargingbiz.ConnectorTypeDC, 60, 380)
		require.NoError(t, err)

		userB := uuid.New()
		sessB, err := uc.StartCharging(ctxB, stationB.ID, connB.ID, userB, "京C11111")
		require.NoError(t, err)

		// 租户 A 查询用户 B 的会话：不可见
		_, total, err := uc.GetUserSessions(ctxA, userB, 1, 10)
		require.NoError(t, err)
		require.Equal(t, int64(0), total)

		// 租户 B 自己可见
		sessions, total, err := uc.GetUserSessions(ctxB, userB, 1, 10)
		require.NoError(t, err)
		require.Equal(t, int64(1), total)
		require.Equal(t, sessB.ID, sessions[0].ID)
	})
}

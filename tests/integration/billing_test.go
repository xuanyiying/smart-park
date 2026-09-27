//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1 "github.com/xuanyiying/smart-park/api/billing/v1"
	billingbiz "github.com/xuanyiying/smart-park/internal/billing/biz"
)

// seedRules 为一个停车场写入基础费率、长时间优惠、夜间优惠与月卡全免规则。
// 规则 JSON 结构与 internal/billing/data/seed.go 完全一致。
func seedRules(t *testing.T, repo billingbiz.BillingRuleRepo, ctx context.Context, lotID uuid.UUID) {
	t.Helper()
	rules := []*billingbiz.BillingRule{
		{
			ID: uuid.New(), LotID: lotID, RuleName: "小型车基础", RuleType: "base",
			Conditions: `{"type":"vehicle_type","value":"small"}`,
			Actions:    `[{"type":"fixed","amount":5},{"type":"per_hour","amount":2},{"type":"max_daily","amount":50}]`,
			Priority:   10, IsActive: true,
		},
		{
			ID: uuid.New(), LotID: lotID, RuleName: "长时间优惠", RuleType: "discount",
			Conditions: `{"type":"duration_min","operator":"gte","value":120}`,
			Actions:    `[{"type":"fixed","amount":3}]`,
			Priority:   5, IsActive: true,
		},
		{
			ID: uuid.New(), LotID: lotID, RuleName: "夜间优惠", RuleType: "discount",
			Conditions: `{"type":"time_range","value":{"start":22,"end":6}}`,
			Actions:    `[{"type":"fixed","amount":2}]`,
			Priority:   4, IsActive: true,
		},
		{
			ID: uuid.New(), LotID: lotID, RuleName: "月卡全免", RuleType: "monthly",
			Conditions: `{"type":"vehicle_type","value":"monthly"}`,
			Actions:    `[{"type":"fixed","amount":0}]`,
			Priority:   1, IsActive: true,
		},
	}
	for _, r := range rules {
		require.NoError(t, repo.CreateBillingRule(ctx, r))
	}
}

// TestBillingCalculateFeeWithRealRules 在真实数据库中创建规则并计费，
// 验证基础费率、优惠减免、月卡全免与无规则兜底金额。
func TestBillingCalculateFeeWithRealRules(t *testing.T) {
	uc, repo, ctx := newBillingStack(t)
	lotID := uuid.New()
	seedRules(t, repo, ctx, lotID)

	day := func(hour, min int) int64 {
		// 固定周三，避开 weekend/holiday 条件分支
		return time.Date(2026, 9, 23, hour, min, 0, 0, time.Local).Unix()
	}

	t.Run("小型车3小时_基础费加长时间优惠", func(t *testing.T) {
		// base = 5 元起步 + 2 元/小时 × 3h = 11 元；优惠 3 元 → 8 元
		bill, err := uc.CalculateFee(ctx, &v1.CalculateFeeRequest{
			LotId: lotID.String(), RecordId: uuid.New().String(),
			EntryTime: day(10, 0), ExitTime: day(13, 0), VehicleType: "small",
		})
		require.NoError(t, err)
		require.Equal(t, int64(1100), bill.BaseAmount)
		require.Equal(t, int64(300), bill.DiscountAmount)
		require.Equal(t, int64(800), bill.FinalAmount)
		require.NotEmpty(t, bill.AppliedRules)
	})

	t.Run("不足优惠门槛时无减免", func(t *testing.T) {
		// 1 小时：base = 7 元，不满足 120 分钟优惠，也不满足夜间时段
		bill, err := uc.CalculateFee(ctx, &v1.CalculateFeeRequest{
			LotId: lotID.String(), RecordId: uuid.New().String(),
			EntryTime: day(10, 0), ExitTime: day(11, 0), VehicleType: "small",
		})
		require.NoError(t, err)
		require.Equal(t, int64(700), bill.BaseAmount)
		require.Equal(t, int64(0), bill.DiscountAmount)
		require.Equal(t, int64(700), bill.FinalAmount)
	})

	t.Run("夜间出场享受夜间优惠", func(t *testing.T) {
		// 22:00-23:30：base = 5 + 1.5×2 = 8 元，夜间优惠 2 元 → 6 元
		bill, err := uc.CalculateFee(ctx, &v1.CalculateFeeRequest{
			LotId: lotID.String(), RecordId: uuid.New().String(),
			EntryTime: day(22, 0), ExitTime: day(23, 30), VehicleType: "small",
		})
		require.NoError(t, err)
		require.Equal(t, int64(800), bill.BaseAmount)
		require.Equal(t, int64(200), bill.DiscountAmount)
		require.Equal(t, int64(600), bill.FinalAmount)
	})

	t.Run("月卡车辆全额减免", func(t *testing.T) {
		// monthly 条件命中月卡规则：discountAmount 被置为 baseAmount，最终为 0
		bill, err := uc.CalculateFee(ctx, &v1.CalculateFeeRequest{
			LotId: lotID.String(), RecordId: uuid.New().String(),
			EntryTime: day(10, 0), ExitTime: day(13, 0), VehicleType: "monthly",
		})
		require.NoError(t, err)
		require.Equal(t, int64(0), bill.FinalAmount)
	})

	t.Run("无规则的停车场走默认费率", func(t *testing.T) {
		// 3 小时无任何规则：默认 2 元/小时 = 6 元
		bill, err := uc.CalculateFee(ctx, &v1.CalculateFeeRequest{
			LotId: uuid.New().String(), RecordId: uuid.New().String(),
			EntryTime: day(10, 0), ExitTime: day(13, 0), VehicleType: "small",
		})
		require.NoError(t, err)
		require.Equal(t, int64(600), bill.BaseAmount)
		require.Equal(t, int64(600), bill.FinalAmount)
	})
}

// TestBillingRuleTypeRoundTrip 回归测试：biz 层的 base/discount/exemption/
// override 类型此前无法落库（ent 枚举缺失，被静默映射为 time），
// 导致种子数据的优惠规则被当作基础计费规则执行。修复后必须正确往返。
func TestBillingRuleTypeRoundTrip(t *testing.T) {
	uc, repo, ctx := newBillingStack(t)

	original := &billingbiz.BillingRule{
		ID: uuid.New(), LotID: uuid.New(), RuleName: "回归-优惠规则", RuleType: "discount",
		Conditions: `{"type":"duration_min","operator":"gte","value":60}`,
		Actions:    `[{"type":"fixed","amount":2}]`,
		Priority:   5, IsActive: true,
	}
	require.NoError(t, repo.CreateBillingRule(ctx, original))

	got, err := repo.GetBillingRule(ctx, original.ID)
	require.NoError(t, err)
	require.Equal(t, "discount", got.RuleType)

	// 优惠规则在计费时必须作为减免生效，而不是替代基础费率
	day := time.Date(2026, 9, 23, 10, 0, 0, 0, time.Local).Unix()
	bill, err := uc.CalculateFee(ctx, &v1.CalculateFeeRequest{
		LotId: original.LotID.String(), RecordId: uuid.New().String(),
		EntryTime: day, ExitTime: day + 2*3600, VehicleType: "small",
	})
	require.NoError(t, err)
	// 无基础规则：默认费率 2 元/小时 × 2h = 4 元；优惠 2 元 → 2 元
	require.Equal(t, int64(400), bill.BaseAmount)
	require.Equal(t, int64(200), bill.DiscountAmount)
	require.Equal(t, int64(200), bill.FinalAmount)
}

package biz

import (
	"testing"
	"time"
)

// TestEvaluateConditionRemainingBranches covers the condition types the base
// suite does not reach: calendar predicates, entry-time windows and boolean
// combinators. Each of these silently disabling would break real rule configs.
func TestEvaluateConditionRemainingBranches(t *testing.T) {
	// 2026-10-12 is a Monday.
	entry := time.Date(2026, 10, 12, 10, 0, 0, 0, time.Local)
	exit := entry.Add(3 * time.Hour)

	ctx := &BillingContext{
		VehicleType: "small",
		Duration:    3 * time.Hour,
		EntryTime:   entry,
		ExitTime:    exit,
	}

	cases := []struct {
		name string
		cond *Condition
		want bool
	}{
		{"and true", &Condition{Type: "and", Conditions: []*Condition{
			{Type: "vehicle_type", Value: "small"},
			{Type: "duration_min", Operator: "gte", Value: float64(60)},
		}}, true},
		{"and false", &Condition{Type: "and", Conditions: []*Condition{
			{Type: "vehicle_type", Value: "small"},
			{Type: "duration_min", Operator: "gte", Value: float64(600)},
		}}, false},
		{"or true", &Condition{Type: "or", Conditions: []*Condition{
			{Type: "vehicle_type", Value: "large"},
			{Type: "duration_min", Operator: "gte", Value: float64(60)},
		}}, true},
		{"or false", &Condition{Type: "or", Conditions: []*Condition{
			{Type: "vehicle_type", Value: "large"},
			{Type: "duration_min", Operator: "gte", Value: float64(600)},
		}}, false},
		{"not true", &Condition{Type: "not", Conditions: []*Condition{
			{Type: "vehicle_type", Value: "large"},
		}}, true},
		{"not empty", &Condition{Type: "not"}, false},
		{"unknown type", &Condition{Type: "no_such_type"}, false},
		{"vehicle_type_in match", &Condition{Type: "vehicle_type_in", Value: []interface{}{"large", "small"}}, true},
		{"vehicle_type_in no match", &Condition{Type: "vehicle_type_in", Value: []interface{}{"large"}}, false},
		{"vehicle_type_in malformed", &Condition{Type: "vehicle_type_in", Value: "small"}, false},
		{"duration_hour gte", &Condition{Type: "duration_hour", Operator: "gte", Value: float64(2)}, true},
		{"duration_hour lte", &Condition{Type: "duration_hour", Operator: "lte", Value: float64(5)}, true},
		{"duration_hour gt", &Condition{Type: "duration_hour", Operator: "gt", Value: float64(1)}, true},
		{"duration_hour lt", &Condition{Type: "duration_hour", Operator: "lt", Value: float64(5)}, true},
		{"duration_hour eq", &Condition{Type: "duration_hour", Operator: "eq", Value: float64(3)}, true},
		{"duration_hour unknown operator", &Condition{Type: "duration_hour", Operator: "ne", Value: float64(3)}, false},
		{"duration_min unknown operator", &Condition{Type: "duration_min", Operator: "ne", Value: float64(3)}, false},
		{"day_of_week match", &Condition{Type: "day_of_week", Value: []interface{}{float64(1)}}, true}, // Monday
		{"day_of_week no match", &Condition{Type: "day_of_week", Value: []interface{}{float64(0)}}, false},
		{"day_of_week malformed", &Condition{Type: "day_of_week", Value: float64(1)}, false},
		{"day_of_month match", &Condition{Type: "day_of_month", Value: []interface{}{float64(12)}}, true},
		{"day_of_month no match", &Condition{Type: "day_of_month", Value: []interface{}{float64(1)}}, false},
		{"month match", &Condition{Type: "month", Value: []interface{}{float64(10)}}, true},
		{"month no match", &Condition{Type: "month", Value: []interface{}{float64(1)}}, false},
		{"holiday", &Condition{Type: "holiday"}, false},
		{"weekend", &Condition{Type: "weekend"}, false},
		{"workday", &Condition{Type: "workday"}, true},
		{"time_range malformed value", &Condition{Type: "time_range", Value: "nope"}, false},
		{"time_range missing start", &Condition{Type: "time_range", Value: map[string]interface{}{"end": float64(6)}}, false},
		{"entry_time_range match", &Condition{Type: "entry_time_range", Value: map[string]interface{}{"start": float64(8), "end": float64(12)}}, true},
		{"entry_time_range no match", &Condition{Type: "entry_time_range", Value: map[string]interface{}{"start": float64(20), "end": float64(22)}}, false},
		{"entry_time_range malformed", &Condition{Type: "entry_time_range", Value: float64(10)}, false},
		{"nil condition", nil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := EvaluateCondition(tc.cond, ctx); got != tc.want {
				t.Errorf("EvaluateCondition(%s) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}

	// Weekend path: Saturday 2026-10-17.
	satCtx := &BillingContext{
		VehicleType: "small",
		EntryTime:   time.Date(2026, 10, 17, 10, 0, 0, 0, time.Local),
		ExitTime:    time.Date(2026, 10, 17, 12, 0, 0, 0, time.Local),
	}
	if !EvaluateCondition(&Condition{Type: "weekend"}, satCtx) {
		t.Error("Saturday must satisfy the weekend condition")
	}
	if EvaluateCondition(&Condition{Type: "workday"}, satCtx) {
		t.Error("Saturday must not satisfy the workday condition")
	}
	if !EvaluateCondition(&Condition{Type: "holiday"}, &BillingContext{IsHoliday: true}) {
		t.Error("holiday condition must reflect the context flag")
	}
}

// TestApplyActionsRemainingBranches covers the action types not exercised by the
// base suite, including the incremental-combination semantics for tiered and
// time-segment billing.
func TestApplyActionsRemainingBranches(t *testing.T) {
	entry := time.Date(2026, 10, 12, 10, 0, 0, 0, time.Local)

	t.Run("per_minute", func(t *testing.T) {
		got := applyActions([]*Action{{Type: "per_minute", Amount: 0.5}}, entry, entry.Add(60*time.Minute))
		if got != 30 {
			t.Errorf("per_minute 60min at 0.5 = %v, want 30", got)
		}
	})

	t.Run("percentage", func(t *testing.T) {
		got := applyActions([]*Action{
			{Type: "fixed", Amount: 100},
			{Type: "percentage", Percent: 20},
		}, entry, entry.Add(time.Hour))
		if got != 80 {
			t.Errorf("100 with 20%% off = %v, want 80", got)
		}
	})

	t.Run("cap and ceil", func(t *testing.T) {
		got := applyActions([]*Action{
			{Type: "per_hour", Amount: 3},
			{Type: "cap", Cap: 5},
			{Type: "ceil"},
		}, entry, entry.Add(10*time.Hour))
		if got != 5 {
			t.Errorf("capped amount = %v, want 5", got)
		}
	})

	t.Run("min_charge", func(t *testing.T) {
		got := applyActions([]*Action{
			{Type: "per_hour", Amount: 0.1},
			{Type: "min_charge", Amount: 3},
		}, entry, entry.Add(10*time.Minute))
		if got != 3 {
			t.Errorf("min charge = %v, want 3", got)
		}
	})

	t.Run("night_discount applies after 22:00", func(t *testing.T) {
		got := applyActions([]*Action{
			{Type: "fixed", Amount: 100},
			{Type: "night_discount", Amount: 30},
		}, entry, entry.Add(13*time.Hour)) // exit at 23:00
		if got != 70 {
			t.Errorf("night discount = %v, want 70", got)
		}
	})

	t.Run("night_discount skipped during the day", func(t *testing.T) {
		got := applyActions([]*Action{
			{Type: "fixed", Amount: 100},
			{Type: "night_discount", Amount: 30},
		}, entry, entry.Add(2*time.Hour)) // exit at 12:00
		if got != 100 {
			t.Errorf("daytime night_discount must not apply, got %v", got)
		}
	})

	t.Run("first_hour_free within window", func(t *testing.T) {
		got := applyActions([]*Action{
			{Type: "fixed", Amount: 5},
			{Type: "first_hour_free"},
		}, entry, entry.Add(30*time.Minute))
		if got != 0 {
			t.Errorf("stays inside the first hour are free, got %v", got)
		}
	})

	t.Run("tiered accumulates", func(t *testing.T) {
		// 5 小时：首 2 小时 3 元/时，余 3 小时 5 元/时 = 6 + 15 = 21。
		got := applyActions([]*Action{{
			Type: "tiered",
			Value: []interface{}{
				map[string]interface{}{"hours": float64(2), "rate": float64(3)},
				map[string]interface{}{"hours": float64(3), "rate": float64(5)},
			},
		}}, entry, entry.Add(5*time.Hour))
		if got != 21 {
			t.Errorf("tiered 5h = %v, want 21", got)
		}
	})

	t.Run("tiered beyond last tier uses top rate", func(t *testing.T) {
		// 6 小时：2×3 + 3×5 + 1×5 = 26。
		got := applyActions([]*Action{{
			Type: "tiered",
			Value: []interface{}{
				map[string]interface{}{"hours": float64(2), "rate": float64(3)},
				map[string]interface{}{"hours": float64(3), "rate": float64(5)},
			},
		}}, entry, entry.Add(6*time.Hour))
		if got != 26 {
			t.Errorf("tiered 6h = %v, want 26", got)
		}
	})

	t.Run("tiered ignores malformed tiers", func(t *testing.T) {
		// 前两档均畸形被跳过；剩余时长按最后一档可解析的费率兜底（2h × 3 = 6），
		// 即"超出阶梯部分按最高阶梯计费"的既有语义。
		got := applyActions([]*Action{{
			Type: "tiered",
			Value: []interface{}{
				"not-a-map",
				map[string]interface{}{"hours": "bad", "rate": float64(3)},
			},
		}}, entry, entry.Add(2*time.Hour))
		if got != 6 {
			t.Errorf("malformed tiers must be skipped with top-rate fallback, got %v", got)
		}
	})

	t.Run("tiered non-array value", func(t *testing.T) {
		got := applyActions([]*Action{{Type: "tiered", Value: "bad"}}, entry, entry.Add(time.Hour))
		if got != 0 {
			t.Errorf("non-array tiered value must contribute nothing, got %v", got)
		}
	})

	t.Run("time_segment uses real overlap", func(t *testing.T) {
		// 10:00–12:00 停留，08:00–11:00 窗口 3 元/时：重叠 1 小时 = 3 元。
		got := applyActions([]*Action{{
			Type: "time_segment",
			Value: []interface{}{
				map[string]interface{}{"start": float64(8), "end": float64(11), "rate": float64(3)},
			},
		}}, entry, entry.Add(2*time.Hour))
		if got != 3 {
			t.Errorf("time_segment overlap = %v, want 3", got)
		}
	})

	t.Run("time_segment wraps midnight", func(t *testing.T) {
		// 22:00 入场，02:00 出场；窗口 22:00–06:00 全程命中 4 小时 × 2 = 8。
		nightEntry := time.Date(2026, 10, 12, 22, 0, 0, 0, time.Local)
		got := applyActions([]*Action{{
			Type: "time_segment",
			Value: []interface{}{
				map[string]interface{}{"start": float64(22), "end": float64(6), "rate": float64(2)},
			},
		}}, nightEntry, nightEntry.Add(4*time.Hour))
		if got != 8 {
			t.Errorf("overnight time_segment = %v, want 8", got)
		}
	})

	t.Run("time_segment ignores malformed segments", func(t *testing.T) {
		got := applyActions([]*Action{{
			Type: "time_segment",
			Value: []interface{}{
				"bad",
				map[string]interface{}{"start": float64(8), "end": float64(9)}, // missing rate
			},
		}}, entry, entry.Add(time.Hour))
		if got != 0 {
			t.Errorf("malformed segments must contribute nothing, got %v", got)
		}
	})

	t.Run("discount action family", func(t *testing.T) {
		for _, actionType := range []string{"member_discount", "promotion_discount", "seasonal_discount"} {
			got := applyActions([]*Action{
				{Type: "fixed", Amount: 100},
				{Type: actionType, Percent: 10},
			}, entry, entry.Add(time.Hour))
			if got != 90 {
				t.Errorf("%s = %v, want 90", actionType, got)
			}
		}
	})

	t.Run("long_term_discount threshold", func(t *testing.T) {
		actions := []*Action{
			{Type: "fixed", Amount: 100},
			{Type: "long_term_discount", Percent: 20, Value: float64(4)},
		}
		if got := applyActions(actions, entry, entry.Add(5*time.Hour)); got != 80 {
			t.Errorf("long stay = %v, want 80", got)
		}
		if got := applyActions(actions, entry, entry.Add(2*time.Hour)); got != 100 {
			t.Errorf("short stay must not be discounted, got %v", got)
		}
	})

	t.Run("flat_rate is additive", func(t *testing.T) {
		got := applyActions([]*Action{
			{Type: "fixed", Amount: 5},
			{Type: "flat_rate", Amount: 3},
		}, entry, entry.Add(90*time.Minute))
		if got != 8 {
			t.Errorf("flat_rate must add to the running amount, got %v", got)
		}
	})

	t.Run("unknown action is ignored", func(t *testing.T) {
		got := applyActions([]*Action{
			{Type: "fixed", Amount: 7},
			{Type: "no_such_action", Amount: 100},
		}, entry, entry.Add(time.Hour))
		if got != 7 {
			t.Errorf("unknown actions must be ignored, got %v", got)
		}
	})

	t.Run("cross-day max_daily counts natural days", func(t *testing.T) {
		// 跨一天零 2 小时：2 个自然日 × 20 = 40 封顶（而不是 Ceil(26/24)=2 天近似，
		// 也不是 1 天）。
		nightEntry := time.Date(2026, 10, 12, 22, 0, 0, 0, time.Local)
		got := applyActions([]*Action{
			{Type: "per_hour", Amount: 5},
			{Type: "max_daily", Amount: 20},
		}, nightEntry, nightEntry.Add(26*time.Hour))
		if got != 40 {
			t.Errorf("cross-day cap = %v, want 40", got)
		}
	})
}

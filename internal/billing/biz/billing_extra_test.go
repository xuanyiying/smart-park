package biz

import (
	"testing"
	"time"
)

// M4: 配置的节假日日期应命中 holiday 条件。
func TestHolidayCalendar(t *testing.T) {
	cal, err := NewHolidayCalendar([]string{"2026-10-01", "2026-10-02"})
	if err != nil {
		t.Fatalf("NewHolidayCalendar failed: %v", err)
	}

	nationalDay := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if !cal.IsHoliday(nationalDay) {
		t.Error("2026-10-01 should be a holiday")
	}
	ordinaryDay := time.Date(2026, 3, 26, 9, 0, 0, 0, time.UTC)
	if cal.IsHoliday(ordinaryDay) {
		t.Error("2026-03-26 should not be a holiday")
	}

	var nilCal *HolidayCalendar
	if nilCal.IsHoliday(nationalDay) {
		t.Error("nil calendar must never report a holiday")
	}

	if _, err := NewHolidayCalendar([]string{"not-a-date"}); err == nil {
		t.Error("invalid date must produce an error")
	}
}

// M5: max_daily 上限按自然日数量计算，跨零点 25h 停留占 2 个自然日。
func TestApplyActions_MaxDailyNaturalDays(t *testing.T) {
	entry := time.Date(2026, 3, 26, 23, 0, 0, 0, time.UTC)
	exit := entry.Add(25 * time.Hour) // 次日 24:00，覆盖 2 个自然日

	got := applyActions(
		[]*Action{{Type: "per_hour", Amount: 100}, {Type: "max_daily", Amount: 120}},
		entry, exit,
	)
	// 不封顶为 2500；按 2 个自然日封顶为 240。
	if got != 240 {
		t.Errorf("applyActions() = %v, want 240", got)
	}
}

// M5: time_segment 按真实重叠时长计费，跨零点窗口两侧都命中。
func TestApplyActions_TimeSegmentOverlap(t *testing.T) {
	actions := []*Action{{Type: "time_segment", Value: []interface{}{
		map[string]interface{}{"start": 22.0, "end": 6.0, "rate": 4.0},
	}}}

	// 22:30-23:00 与 22-06 窗口重叠 0.5h → 2 元
	got := applyActions(actions,
		time.Date(2026, 3, 26, 22, 30, 0, 0, time.UTC),
		time.Date(2026, 3, 26, 23, 0, 0, 0, time.UTC))
	if got != 2.0 {
		t.Errorf("applyActions() = %v, want 2.0", got)
	}

	// 01:00-02:00 同样落在跨零点窗口 → 4 元
	got = applyActions(actions,
		time.Date(2026, 3, 27, 1, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 27, 2, 0, 0, 0, time.UTC))
	if got != 4.0 {
		t.Errorf("applyActions() = %v, want 4.0", got)
	}
}

// H2: flat_rate 与其他动作组合时为增量叠加；需要整体覆盖语义时使用 override 规则。
func TestApplyActions_FlatRateAdditive(t *testing.T) {
	got := applyActions(
		[]*Action{{Type: "fixed", Amount: 5}, {Type: "flat_rate", Amount: 30}},
		time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 26, 11, 0, 0, 0, time.UTC),
	)
	if got != 35 {
		t.Errorf("applyActions() = %v, want 35", got)
	}
}

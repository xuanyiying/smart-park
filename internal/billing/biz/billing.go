// Package biz provides business logic for the billing service.
package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/billing/v1"
)

// Condition represents a parsed billing condition.
type Condition struct {
	Type       string       `json:"type"`
	Field      string       `json:"field,omitempty"`
	Operator   string       `json:"operator,omitempty"`
	Value      interface{}  `json:"value,omitempty"`
	And        []*Condition `json:"and,omitempty"`
	Or         []*Condition `json:"or,omitempty"`
	Conditions []*Condition `json:"conditions,omitempty"`
}

// Action represents a parsed billing action.
type Action struct {
	Type    string  `json:"type"`
	Amount  float64 `json:"amount,omitempty"`
	Percent float64 `json:"percent,omitempty"`
	Unit    string  `json:"unit,omitempty"`
	Ceil    float64 `json:"ceil,omitempty"`
	Cap     float64 `json:"cap,omitempty"`
	Value   interface{} `json:"value,omitempty"`
}

// ParseConditions parses JSON conditions string into Condition struct.
func ParseConditions(jsonStr string) (*Condition, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var cond Condition
	if err := json.Unmarshal([]byte(jsonStr), &cond); err != nil {
		return nil, fmt.Errorf("failed to parse conditions: %w", err)
	}
	return &cond, nil
}

// ParseActions parses JSON actions string into Action slice.
func ParseActions(jsonStr string) ([]*Action, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var actions []*Action
	if err := json.Unmarshal([]byte(jsonStr), &actions); err != nil {
		return nil, fmt.Errorf("failed to parse actions: %w", err)
	}
	return actions, nil
}

// EvaluateCondition evaluates if a condition is met given the context.
func EvaluateCondition(cond *Condition, ctx *BillingContext) bool {
	if cond == nil {
		return true
	}

	switch cond.Type {
	case "and":
		for _, c := range cond.Conditions {
			if !EvaluateCondition(c, ctx) {
				return false
			}
		}
		return true

	case "or":
		for _, c := range cond.Conditions {
			if EvaluateCondition(c, ctx) {
				return true
			}
		}
		return false

	case "not":
		if len(cond.Conditions) > 0 {
			return !EvaluateCondition(cond.Conditions[0], ctx)
		}
		return false

	case "vehicle_type":
		return ctx.VehicleType == cond.Value

	case "vehicle_type_in":
		types, ok := cond.Value.([]interface{})
		if !ok {
			return false
		}
		for _, v := range types {
			if ctx.VehicleType == v {
				return true
			}
		}
		return false

	case "duration_min":
		minutes := ctx.Duration.Minutes()
		switch cond.Operator {
		case "gte":
			if val, ok := cond.Value.(float64); ok {
				return minutes >= val
			}
		case "lte":
			if val, ok := cond.Value.(float64); ok {
				return minutes <= val
			}
		case "gt":
			if val, ok := cond.Value.(float64); ok {
				return minutes > val
			}
		case "lt":
			if val, ok := cond.Value.(float64); ok {
				return minutes < val
			}
		case "eq":
			if val, ok := cond.Value.(float64); ok {
				return minutes == val
			}
		}
		return false

	case "duration_hour":
		hours := ctx.Duration.Hours()
		switch cond.Operator {
		case "gte":
			if val, ok := cond.Value.(float64); ok {
				return hours >= val
			}
		case "lte":
			if val, ok := cond.Value.(float64); ok {
				return hours <= val
			}
		case "gt":
			if val, ok := cond.Value.(float64); ok {
				return hours > val
			}
		case "lt":
			if val, ok := cond.Value.(float64); ok {
				return hours < val
			}
		case "eq":
			if val, ok := cond.Value.(float64); ok {
				return hours == val
			}
		}
		return false

	case "time_range":
		valueMap, ok := cond.Value.(map[string]interface{})
		if !ok {
			return false
		}
		start, ok1 := valueMap["start"].(float64)
		end, ok2 := valueMap["end"].(float64)
		if !ok1 || !ok2 {
			return false
		}
		hour := float64(ctx.ExitTime.Hour()) + float64(ctx.ExitTime.Minute())/60.0
		return hourInRange(hour, start, end)

	case "entry_time_range":
		valueMap, ok := cond.Value.(map[string]interface{})
		if !ok {
			return false
		}
		start, ok1 := valueMap["start"].(float64)
		end, ok2 := valueMap["end"].(float64)
		if !ok1 || !ok2 {
			return false
		}
		hour := float64(ctx.EntryTime.Hour()) + float64(ctx.EntryTime.Minute())/60.0
		return hourInRange(hour, start, end)

	case "day_of_week":
		days, ok := cond.Value.([]interface{})
		if !ok {
			return false
		}
		weekday := int(ctx.ExitTime.Weekday())
		for _, day := range days {
			if dayNum, ok := day.(float64); ok && int(dayNum) == weekday {
				return true
			}
		}
		return false

	case "day_of_month":
		days, ok := cond.Value.([]interface{})
		if !ok {
			return false
		}
		dayOfMonth := ctx.ExitTime.Day()
		for _, day := range days {
			if dayNum, ok := day.(float64); ok && int(dayNum) == dayOfMonth {
				return true
			}
		}
		return false

	case "month":
		months, ok := cond.Value.([]interface{})
		if !ok {
			return false
		}
		month := int(ctx.ExitTime.Month())
		for _, m := range months {
			if monthNum, ok := m.(float64); ok && int(monthNum) == month {
				return true
			}
		}
		return false

	case "holiday":
		return ctx.IsHoliday

	case "weekend":
		weekday := int(ctx.ExitTime.Weekday())
		return weekday == 0 || weekday == 6

	case "workday":
		weekday := int(ctx.ExitTime.Weekday())
		return weekday >= 1 && weekday <= 5

	default:
		return false
	}
}

// BillingContext contains context for billing rule evaluation.
type BillingContext struct {
	VehicleType string
	Duration    time.Duration
	EntryTime   time.Time
	ExitTime    time.Time
	IsHoliday   bool
}

// BillingRule represents a billing rule entity.
type BillingRule struct {
	ID         uuid.UUID
	LotID      uuid.UUID
	RuleName   string
	RuleType   string
	Conditions string
	Actions    string
	RuleConfig map[string]interface{}
	Priority   int
	IsActive   bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// BillingRuleRepo defines the repository interface for billing rule operations.
type BillingRuleRepo interface {
	GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error)
	GetBillingRule(ctx context.Context, ruleID uuid.UUID) (*BillingRule, error)
	CreateBillingRule(ctx context.Context, rule *BillingRule) error
	UpdateBillingRule(ctx context.Context, rule *BillingRule) error
	DeleteBillingRule(ctx context.Context, ruleID uuid.UUID) error
	ListBillingRules(ctx context.Context, lotID uuid.UUID, page, pageSize int) ([]*BillingRule, int64, error)
	// SeedData provisions starting rules for the given parking lot. Passing uuid.Nil
	// skips seeding, which is the correct production default.
	SeedData(ctx context.Context, lotID uuid.UUID) error
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// BillingUseCase implements billing business logic.
type BillingUseCase struct {
	repo     BillingRuleRepo
	log      *log.Helper
	holidays *HolidayCalendar
	cache    *ruleCache
}

// HolidayCalendar holds the configured statutory holiday dates ("2006-01-02").
//
// The condition engine has always supported a "holiday" condition, but with no
// holiday data source it could never fire. Operators now maintain the calendar
// through billing configuration; an absent or empty calendar simply means no
// day is a holiday.
type HolidayCalendar struct {
	dates map[string]struct{}
}

// NewHolidayCalendar builds a calendar from ISO dates ("2006-01-02").
func NewHolidayCalendar(dates []string) (*HolidayCalendar, error) {
	cal := &HolidayCalendar{dates: make(map[string]struct{}, len(dates))}
	for _, d := range dates {
		t, err := time.Parse("2006-01-02", d)
		if err != nil {
			return nil, fmt.Errorf("invalid holiday date %q: %w", d, err)
		}
		cal.dates[t.Format("2006-01-02")] = struct{}{}
	}
	return cal, nil
}

// IsHoliday reports whether t falls on a configured holiday. A nil calendar
// never reports a holiday.
func (c *HolidayCalendar) IsHoliday(t time.Time) bool {
	if c == nil {
		return false
	}
	_, ok := c.dates[t.Format("2006-01-02")]
	return ok
}

// defaultRuleCacheTTL bounds how long a lot's rules may be served from cache.
// Rule mutations invalidate the cache immediately, so the TTL only controls how
// stale rules can get when someone edits the database out-of-band.
const defaultRuleCacheTTL = 30 * time.Second

// ruleCache caches a parking lot's rules for a short TTL so per-vehicle fee
// calculations stop reading the whole rules table on every call.
type ruleCache struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[uuid.UUID]ruleCacheEntry
}

type ruleCacheEntry struct {
	rules     []*BillingRule
	fetchedAt time.Time
}

func (c *ruleCache) get(lotID uuid.UUID) ([]*BillingRule, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.entries[lotID]
	if !ok || time.Since(e.fetchedAt) >= c.ttl {
		return nil, false
	}
	return e.rules, true
}

func (c *ruleCache) put(lotID uuid.UUID, rules []*BillingRule) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[lotID] = ruleCacheEntry{rules: rules, fetchedAt: time.Now()}
}

func (c *ruleCache) invalidate(lotID uuid.UUID) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.entries, lotID)
}

func (c *ruleCache) invalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries = make(map[uuid.UUID]ruleCacheEntry)
}

// NewBillingUseCase creates a new BillingUseCase.
//
// holidays carries the configured statutory holiday dates ("2006-01-02"); a nil
// or empty slice means the "holiday" condition never matches.
func NewBillingUseCase(repo BillingRuleRepo, logger log.Logger, holidays []string) *BillingUseCase {
	cal, err := NewHolidayCalendar(holidays)
	if err != nil {
		log.NewHelper(logger).Warnf("invalid holiday calendar, holiday conditions disabled: %v", err)
		cal = &HolidayCalendar{dates: map[string]struct{}{}}
	}
	return &BillingUseCase{
		repo:     repo,
		log:      log.NewHelper(logger),
		holidays: cal,
		cache:    &ruleCache{ttl: defaultRuleCacheTTL, entries: make(map[uuid.UUID]ruleCacheEntry)},
	}
}

// getRules returns a parking lot's rules, serving them from the short-TTL cache
// when possible so the rules table is not read on every fee calculation.
func (uc *BillingUseCase) getRules(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error) {
	if rules, ok := uc.cache.get(lotID); ok {
		return rules, nil
	}

	rules, err := uc.repo.GetRulesByLotID(ctx, lotID)
	if err != nil {
		return nil, err
	}

	uc.cache.put(lotID, rules)
	return rules, nil
}

// CalculateFee calculates the parking fee.
func (uc *BillingUseCase) CalculateFee(ctx context.Context, req *v1.CalculateFeeRequest) (*v1.BillData, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	rules, err := uc.getRules(ctx, lotID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get billing rules: %v", err)
		return nil, err
	}

	// 按优先级排序规则（缓存返回的是共享切片，先拷贝再排序）
	sorted := make([]*BillingRule, len(rules))
	copy(sorted, rules)
	sortRulesByPriority(sorted)
	rules = sorted

	entryTime := time.Unix(req.EntryTime, 0)
	exitTime := time.Unix(req.ExitTime, 0)
	duration := exitTime.Sub(entryTime)

	billingCtx := &BillingContext{
		VehicleType: req.VehicleType,
		Duration:    duration,
		EntryTime:   entryTime,
		ExitTime:    exitTime,
		IsHoliday:   uc.holidays.IsHoliday(exitTime),
	}

	var baseAmount float64
	var discountAmount float64
	var appliedRules []*v1.AppliedRule
	var appliedRuleSet = make(map[string]bool)

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		cond, err := ParseConditions(rule.Conditions)
		if err != nil {
			uc.log.WithContext(ctx).Warnf("failed to parse condition for rule %s: %v", rule.RuleName, err)
			continue
		}

		if !EvaluateCondition(cond, billingCtx) {
			continue
		}

		actions, err := ParseActions(rule.Actions)
		if err != nil {
			uc.log.WithContext(ctx).Warnf("failed to parse actions for rule %s: %v", rule.RuleName, err)
			continue
		}

		ruleAmount := applyActions(actions, entryTime, exitTime)
		if ruleAmount != 0 && !appliedRuleSet[rule.ID.String()] {
			appliedRules = append(appliedRules, &v1.AppliedRule{
				RuleId:   rule.ID.String(),
				RuleName: rule.RuleName,
				Amount:   yuanToCents(ruleAmount),
			})
			appliedRuleSet[rule.ID.String()] = true
		}

		switch rule.RuleType {
		case "base", "time":
			if baseAmount == 0 || ruleAmount < baseAmount {
				baseAmount = ruleAmount
			}
		case "discount", "exemption":
			discountAmount += ruleAmount
		case "monthly":
			if req.VehicleType == "monthly" {
				discountAmount = baseAmount
			}
		case "override":
			// 覆盖规则，直接使用该规则的金额
			baseAmount = ruleAmount
			discountAmount = 0
		}
	}

	if baseAmount == 0 {
		hours := duration.Hours()
		baseAmount = calculateDefaultFee(hours)
	}

	finalAmount := baseAmount - discountAmount
	if finalAmount < 0 {
		finalAmount = 0
	}

	return &v1.BillData{
		RecordId:       req.RecordId,
		BaseAmount:     yuanToCents(baseAmount),
		DiscountAmount: yuanToCents(discountAmount),
		FinalAmount:    yuanToCents(finalAmount),
		AppliedRules:   appliedRules,
	}, nil
}

// TestBillingRule tests a billing rule with given context.
func (uc *BillingUseCase) TestBillingRule(ctx context.Context, req *v1.TestBillingRuleRequest) (*v1.TestBillingRuleResponse, error) {
	entryTime := time.Unix(req.EntryTime, 0)
	exitTime := time.Unix(req.ExitTime, 0)
	duration := exitTime.Sub(entryTime)

	billingCtx := &BillingContext{
		VehicleType: req.VehicleType,
		Duration:    duration,
		EntryTime:   entryTime,
		ExitTime:    exitTime,
		IsHoliday:   req.IsHoliday,
	}

	// 解析条件
	cond, err := ParseConditions(req.ConditionsJson)
	if err != nil {
		return nil, fmt.Errorf("failed to parse conditions: %w", err)
	}

	// 评估条件
	conditionMet := EvaluateCondition(cond, billingCtx)

	var ruleAmount float64
	var appliedActions []string

	if conditionMet {
		// 解析动作
		actions, err := ParseActions(req.ActionsJson)
		if err != nil {
			return nil, fmt.Errorf("failed to parse actions: %w", err)
		}

		// 应用动作
		ruleAmount = applyActions(actions, entryTime, exitTime)

		// 记录应用的动作
		for _, action := range actions {
			appliedActions = append(appliedActions, action.Type)
		}
	}

	return &v1.TestBillingRuleResponse{
		ConditionMet:   conditionMet,
		CalculatedFee:  yuanToCents(ruleAmount),
		AppliedActions: appliedActions,
		Duration:       duration.Seconds(),
	}, nil
}

// toFloat coerces a rule parameter to float64.
//
// Values decoded from JSON arrive as float64, but rules constructed in Go code carry int
// literals; accepting both prevents a silent zero that would make thresholds like free
// durations never trigger.
func toFloat(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	default:
		return 0, false
	}
}

// hourInRange reports whether hour falls inside [start, end].
//
// Ranges that wrap midnight (start > end, e.g. 22:00 through 06:00) are supported: the
// comparison wraps around 24h. Without this, a night discount covering 22:00-06:00 could
// never match anything, because no hour is simultaneously >= 22 and <= 6.
func hourInRange(hour, start, end float64) bool {
	if start <= end {
		return hour >= start && hour <= end
	}
	return hour >= start || hour <= end
}

// sortRulesByPriority sorts rules by priority in descending order.
func sortRulesByPriority(rules []*BillingRule) {
	// 规则数量有限但每次计费都会执行，冒泡排序的 O(n²) 在规则增长后会
	// 成为热点；稳定排序同时保证同优先级规则的声明顺序可预期。
	sort.SliceStable(rules, func(i, j int) bool {
		return rules[i].Priority > rules[j].Priority
	})
}

// daySegment is one calendar-day slice of a stay, used for cross-day billing.
type daySegment struct {
	start, end time.Time
}

// splitByNaturalDay slices [entry, exit] into per-calendar-day segments so that
// day-scoped fees (max_daily caps, time-segment windows) are computed against
// natural days instead of the Ceil(hours/24) approximation. A stay crossing
// midnight therefore contributes to two separate days.
func splitByNaturalDay(entry, exit time.Time) []daySegment {
	if !exit.After(entry) {
		return nil
	}

	entryDayStart := time.Date(entry.Year(), entry.Month(), entry.Day(), 0, 0, 0, 0, entry.Location())
	nextDayStart := entryDayStart.AddDate(0, 0, 1)

	if !exit.After(nextDayStart) {
		return []daySegment{{start: entry, end: exit}}
	}

	segs := []daySegment{{start: entry, end: nextDayStart}}
	for cur := nextDayStart; cur.Before(exit); {
		next := cur.AddDate(0, 0, 1)
		end := next
		if exit.Before(end) {
			end = exit
		}
		segs = append(segs, daySegment{start: cur, end: end})
		cur = next
	}
	return segs
}

// windowOverlapHours sums how many hours of the stay fall inside the
// [startHour, endHour] window on each calendar day the stay touches. Windows
// that wrap midnight (start > end) span from the start hour on one day to the
// end hour on the following day, so a 22:00-06:00 night window matches both the
// late-night and early-morning parts of a stay.
func windowOverlapHours(entry, exit time.Time, startHour, endHour float64) float64 {
	secs := func(h float64) time.Duration { return time.Duration(h * float64(time.Hour)) }

	var total float64
	for _, seg := range splitByNaturalDay(entry, exit) {
		day := time.Date(seg.start.Year(), seg.start.Month(), seg.start.Day(), 0, 0, 0, 0, seg.start.Location())

		// 跨零点窗口的凌晨段从"前一天"的窗口开始时间延续而来，因此除当天
		// 窗口外还需检查前一天开始的窗口；非跨零点窗口与前一天不会重叠，
		// 多算一次检查不会产生重复计费。
		for _, d := range []time.Time{day, day.AddDate(0, 0, -1)} {
			var ws, we time.Time
			if startHour <= endHour {
				ws, we = d.Add(secs(startHour)), d.Add(secs(endHour))
			} else {
				ws, we = d.Add(secs(startHour)), d.AddDate(0, 0, 1).Add(secs(endHour))
			}

			os := ws
			if seg.start.After(os) {
				os = seg.start
			}
			oe := we
			if seg.end.Before(oe) {
				oe = seg.end
			}
			if oe.After(os) {
				total += oe.Sub(os).Hours()
			}
		}
	}
	return total
}

// applyActions applies billing actions and returns the calculated amount.
//
// Combination semantics (defined product decision, see audit item H2): within a
// single rule every action is incremental — base-fee actions (fixed, per_hour,
// per_minute, tiered, time_segment, flat_rate) accumulate onto the running
// amount and threshold/discount actions modify it. Rules that must replace the
// whole accumulated fee declare RuleType "override" instead, which CalculateFee
// applies across rules.
func applyActions(actions []*Action, entryTime, exitTime time.Time) float64 {
	var amount float64
	duration := exitTime.Sub(entryTime)
	hours := duration.Hours()
	segs := splitByNaturalDay(entryTime, exitTime)

	for _, a := range actions {
		switch a.Type {
		case "fixed":
			amount += a.Amount
		case "per_hour":
			// 按自然日分段累计：单日费率与总时长线性计算等价，但为跨天
			// 分割计费保留了按日计费的语义入口。
			for _, seg := range segs {
				amount += seg.end.Sub(seg.start).Hours() * a.Amount
			}
		case "per_minute":
			for _, seg := range segs {
				amount += seg.end.Sub(seg.start).Minutes() * a.Amount
			}
		case "percentage":
			amount -= amount * (a.Percent / 100)
		case "cap":
			if amount > a.Cap {
				amount = a.Cap
			}
		case "ceil":
			amount = ceilToDecimal(amount, 2)
		case "max_daily":
			// 上限按自然日数量计：跨天停留每天最多 a.Amount，
			// 而不是把不足 24h 的尾巴也折算成一整天。
			days := len(segs)
			if days < 1 {
				days = 1
			}
			maxAmount := a.Amount * float64(days)
			if amount > maxAmount {
				amount = maxAmount
			}
		case "min_charge":
			if amount < a.Amount {
				amount = a.Amount
			}
		case "free_duration":
			// Value is expressed in seconds. A stay within the free window is free; once it
			// exceeds the window the full computed amount stands. The window therefore acts
			// as a threshold rather than a deduction, and crucially it must not wipe out
			// the amount the preceding actions have already computed.
			freeSeconds, ok := toFloat(a.Value)
			if ok && duration.Seconds() <= freeSeconds {
				amount = 0
			}
		case "night_discount":
			hour := exitTime.Hour()
			if hour >= 22 || hour < 8 {
				amount = amount * (1 - a.Amount/100)
			}
		case "first_hour_free":
			// Threshold semantics, mirroring free_duration: the first hour is free only
			// while the whole stay fits inside it. Beyond that the computed amount stands,
			// so this action must never overwrite what came before it.
			if hours <= 1 {
				amount = 0
			}
		case "tiered":
			// 阶梯计费（增量叠加）
			tiers, ok := a.Value.([]interface{})
			if ok {
				remainingHours := hours
				for _, tier := range tiers {
					tierMap, ok := tier.(map[string]interface{})
					if !ok {
						continue
					}
					tierHours, ok1 := tierMap["hours"].(float64)
					tierRate, ok2 := tierMap["rate"].(float64)
					if !ok1 || !ok2 {
						continue
					}
					if remainingHours <= 0 {
						break
					}
					billableHours := math.Min(remainingHours, tierHours)
					amount += billableHours * tierRate
					remainingHours -= billableHours
				}
				// 超出阶梯部分按照最高阶梯计费
				if remainingHours > 0 {
					if len(tiers) > 0 {
						lastTier, ok := tiers[len(tiers)-1].(map[string]interface{})
						if ok {
							tierRate, ok := lastTier["rate"].(float64)
							if ok {
								amount += remainingHours * tierRate
							}
						}
					}
				}
			}
		case "time_segment":
			// 时间段计费：按停留与窗口的真实重叠时长计费（支持跨零点窗口），
			// 不再要求出场时刻恰落在窗口内、也不再对整段停留计时。
			segments, ok := a.Value.([]interface{})
			if ok {
				for _, segment := range segments {
					segmentMap, ok := segment.(map[string]interface{})
					if !ok {
						continue
					}
					start, ok1 := segmentMap["start"].(float64)
					end, ok2 := segmentMap["end"].(float64)
					rate, ok3 := segmentMap["rate"].(float64)
					if !ok1 || !ok2 || !ok3 {
						continue
					}
					overlap := windowOverlapHours(entryTime, exitTime, start, end)
					if overlap > 0 {
						amount += overlap * rate
						break
					}
				}
			}
		case "member_discount":
			// 会员折扣
			amount = amount * (1 - a.Percent/100)
		case "promotion_discount":
			// 促销折扣
			amount = amount * (1 - a.Percent/100)
		case "seasonal_discount":
			// 季节性折扣
			amount = amount * (1 - a.Percent/100)
		case "long_term_discount":
			// 长期停车折扣
			longTermThreshold, _ := a.Value.(float64)
			if hours >= longTermThreshold {
				amount = amount * (1 - a.Percent/100)
			}
		case "flat_rate":
			// 固定费率（增量叠加语义）。如需"忽略时长整体定价"，请把规则
			// 的 RuleType 设为 override，由 CalculateFee 跨规则整体覆盖。
			amount += a.Amount
		}
	}

	return amount
}

// calculateDefaultFee calculates default fee when no rules match.
func calculateDefaultFee(hours float64) float64 {
	if hours < 1 {
		return 5
	}
	return hours * 2
}

// ceilToDecimal rounds amount up to specified decimal places.
func ceilToDecimal(amount float64, decimals int) float64 {
	m := 1
	for i := 0; i < decimals; i++ {
		m *= 10
	}
	return float64(int(amount*float64(m)+0.999999)) / float64(m)
}

// yuanToCents converts a yuan-denominated amount to integer cents.
//
// The engine computes in yuan because rates and durations are fractional; the
// conversion to cents happens exactly once, at the API boundary, so no money
// decision downstream sees a float.
func yuanToCents(yuan float64) int64 {
	return int64(math.Round(yuan * 100))
}

// CreateBillingRule creates a new billing rule.
func (uc *BillingUseCase) CreateBillingRule(ctx context.Context, req *v1.CreateBillingRuleRequest) (*v1.BillingRule, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	rule := &BillingRule{
		ID:         uuid.New(),
		LotID:      lotID,
		RuleName:   req.RuleName,
		RuleType:   req.RuleType,
		Conditions: req.ConditionsJson,
		Actions:    req.ActionsJson,
		Priority:   int(req.Priority),
		IsActive:   req.IsActive,
	}

	if err := uc.repo.CreateBillingRule(ctx, rule); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create billing rule: %v", err)
		return nil, err
	}
	uc.cache.invalidate(rule.LotID)

	return &v1.BillingRule{
		Id:             rule.ID.String(),
		LotId:          rule.LotID.String(),
		RuleName:       rule.RuleName,
		RuleType:       rule.RuleType,
		ConditionsJson: rule.Conditions,
		ActionsJson:    rule.Actions,
		Priority:       int32(rule.Priority),
		IsActive:       rule.IsActive,
		CreatedAt:      rule.CreatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateBillingRule updates a billing rule.
func (uc *BillingUseCase) UpdateBillingRule(ctx context.Context, req *v1.UpdateBillingRuleRequest) error {
	ruleID, err := uuid.Parse(req.Id)
	if err != nil {
		return err
	}

	rule := &BillingRule{
		ID:         ruleID,
		RuleName:   req.RuleName,
		RuleType:   req.RuleType,
		Conditions: req.ConditionsJson,
		Actions:    req.ActionsJson,
		Priority:   int(req.Priority),
		IsActive:   req.IsActive,
	}

	if err := uc.repo.UpdateBillingRule(ctx, rule); err != nil {
		return err
	}
	// 更新请求不带 lot_id，整体失效缓存，代价可忽略。
	uc.cache.invalidateAll()
	return nil
}

// DeleteBillingRule deletes a billing rule.
func (uc *BillingUseCase) DeleteBillingRule(ctx context.Context, req *v1.DeleteBillingRuleRequest) error {
	ruleID, err := uuid.Parse(req.Id)
	if err != nil {
		return err
	}

	if err := uc.repo.DeleteBillingRule(ctx, ruleID); err != nil {
		return err
	}
	uc.cache.invalidateAll()
	return nil
}

// GetBillingRules retrieves billing rules for a parking lot.
func (uc *BillingUseCase) GetBillingRules(ctx context.Context, req *v1.GetBillingRulesRequest) ([]*v1.BillingRule, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		// 非法 UUID 说明调用方传参有误，静默返回空列表会掩盖真实错误，
		// 改为显式报错（M8）。
		return nil, fmt.Errorf("invalid lot_id %q: %w", req.LotId, err)
	}

	rules, err := uc.repo.GetRulesByLotID(ctx, lotID)
	if err != nil {
		return nil, err
	}

	var result []*v1.BillingRule
	for _, rule := range rules {
		result = append(result, &v1.BillingRule{
			Id:             rule.ID.String(),
			LotId:          rule.LotID.String(),
			RuleName:       rule.RuleName,
			RuleType:       rule.RuleType,
			ConditionsJson: rule.Conditions,
			ActionsJson:    rule.Actions,
			Priority:       int32(rule.Priority),
			IsActive:       rule.IsActive,
			CreatedAt:      rule.CreatedAt.Format(time.RFC3339),
		})
	}

	return result, nil
}

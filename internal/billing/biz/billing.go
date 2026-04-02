// Package biz provides business logic for the billing service.
package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
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
	Value   float64 `json:"value,omitempty"`
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
		return hour >= start && hour <= end

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
		return hour >= start && hour <= end

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
	SeedData(ctx context.Context) error
}

// BillingUseCase implements billing business logic.
type BillingUseCase struct {
	repo BillingRuleRepo
	log  *log.Helper
}

// NewBillingUseCase creates a new BillingUseCase.
func NewBillingUseCase(repo BillingRuleRepo, logger log.Logger) *BillingUseCase {
	return &BillingUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// CalculateFee calculates the parking fee.
func (uc *BillingUseCase) CalculateFee(ctx context.Context, req *v1.CalculateFeeRequest) (*v1.BillData, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	rules, err := uc.repo.GetRulesByLotID(ctx, lotID)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("failed to get billing rules: %v", err)
		return nil, err
	}

	// 按优先级排序规则
	sortRulesByPriority(rules)

	entryTime := time.Unix(req.EntryTime, 0)
	exitTime := time.Unix(req.ExitTime, 0)
	duration := exitTime.Sub(entryTime)

	billingCtx := &BillingContext{
		VehicleType: req.VehicleType,
		Duration:    duration,
		EntryTime:   entryTime,
		ExitTime:    exitTime,
		IsHoliday:   false,
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

		ruleAmount := applyActions(actions, duration, exitTime)
		if ruleAmount != 0 && !appliedRuleSet[rule.ID.String()] {
			appliedRules = append(appliedRules, &v1.AppliedRule{
				RuleId:   rule.ID.String(),
				RuleName: rule.RuleName,
				Amount:   ruleAmount,
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
		BaseAmount:     baseAmount,
		DiscountAmount: discountAmount,
		FinalAmount:    finalAmount,
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
		ruleAmount = applyActions(actions, duration, exitTime)

		// 记录应用的动作
		for _, action := range actions {
			appliedActions = append(appliedActions, action.Type)
		}
	}

	return &v1.TestBillingRuleResponse{
		ConditionMet:  conditionMet,
		CalculatedFee: ruleAmount,
		AppliedActions: appliedActions,
		Duration:      duration.Seconds(),
	}, nil
}

// sortRulesByPriority sorts rules by priority in descending order.
func sortRulesByPriority(rules []*BillingRule) {
	// 冒泡排序，按优先级降序排列
	for i := 0; i < len(rules)-1; i++ {
		for j := 0; j < len(rules)-i-1; j++ {
			if rules[j].Priority < rules[j+1].Priority {
				rules[j], rules[j+1] = rules[j+1], rules[j]
			}
		}
	}
}

// applyActions applies billing actions and returns the calculated amount.
func applyActions(actions []*Action, duration time.Duration, exitTime time.Time) float64 {
	var amount float64
	hours := duration.Hours()
	minutes := duration.Minutes()

	for _, a := range actions {
		switch a.Type {
		case "fixed":
			amount += a.Amount
		case "per_hour":
			amount += hours * a.Amount
		case "per_minute":
			amount += minutes * a.Amount
		case "percentage":
			amount -= amount * (a.Percent / 100)
		case "cap":
			if amount > a.Cap {
				amount = a.Cap
			}
		case "ceil":
			amount = ceilToDecimal(amount, 2)
		case "max_daily":
			days := int(math.Ceil(hours / 24))
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
			freeMinutes := a.Value
			if duration.Minutes() <= freeMinutes {
				amount = 0
			} else {
				// 扣除免费时长后计算费用
				remainingMinutes := duration.Minutes() - freeMinutes
				amount = (remainingMinutes / 60) * a.Amount
			}
		case "night_discount":
			hour := exitTime.Hour()
			if hour >= 22 || hour < 8 {
				amount = amount * (1 - a.Amount/100)
			}
		case "first_hour_free":
			if hours <= 1 {
				amount = 0
			} else {
				// 第一小时免费，超过部分计费
				remainingHours := hours - 1
				amount = remainingHours * a.Amount
			}
		case "tiered":
			// 阶梯计费
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
			// 时间段计费
			segments, ok := a.Value.([]interface{})
			if ok {
				hour := float64(exitTime.Hour()) + float64(exitTime.Minute())/60.0
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
					if hour >= start && hour <= end {
						amount += hours * rate
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
			if hours >= a.Value {
				amount = amount * (1 - a.Percent/100)
			}
		case "flat_rate":
			// 固定费率（不管时长）
			amount = a.Amount
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

	return uc.repo.UpdateBillingRule(ctx, rule)
}

// DeleteBillingRule deletes a billing rule.
func (uc *BillingUseCase) DeleteBillingRule(ctx context.Context, req *v1.DeleteBillingRuleRequest) error {
	ruleID, err := uuid.Parse(req.Id)
	if err != nil {
		return err
	}

	return uc.repo.DeleteBillingRule(ctx, ruleID)
}

// GetBillingRules retrieves billing rules for a parking lot.
func (uc *BillingUseCase) GetBillingRules(ctx context.Context, req *v1.GetBillingRulesRequest) ([]*v1.BillingRule, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		// 如果 lot_id 不是有效的 UUID，返回空列表而不是错误
		return []*v1.BillingRule{}, nil
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

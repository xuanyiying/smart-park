// Package data provides data access layer for the billing service.
package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/billing/biz"
	"github.com/xuanyiying/smart-park/internal/billing/data/ent"
	"github.com/xuanyiying/smart-park/internal/billing/data/ent/billingrule"
)

// billingRuleRepo implements biz.BillingRuleRepo.
type billingRuleRepo struct {
	data *Data
}

// NewBillingRuleRepo creates a new BillingRuleRepo.
func NewBillingRuleRepo(data *Data) biz.BillingRuleRepo {
	return &billingRuleRepo{data: data}
}

// GetRulesByLotID retrieves billing rules by lot ID.
func (r *billingRuleRepo) GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*biz.BillingRule, error) {
	rules, err := r.data.db.BillingRule.Query().
		Where(billingrule.LotID(lotID)).
		Order(ent.Desc(billingrule.FieldPriority)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*biz.BillingRule
	for _, rule := range rules {
		result = append(result, toBizBillingRule(rule))
	}

	return result, nil
}

// GetBillingRule retrieves a billing rule by ID.
func (r *billingRuleRepo) GetBillingRule(ctx context.Context, ruleID uuid.UUID) (*biz.BillingRule, error) {
	rule, err := r.data.db.BillingRule.Get(ctx, ruleID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toBizBillingRule(rule), nil
}

// CreateBillingRule creates a new billing rule.
func (r *billingRuleRepo) CreateBillingRule(ctx context.Context, rule *biz.BillingRule) error {
	ruleType := billingrule.RuleTypeTime
	switch rule.RuleType {
	case "period":
		ruleType = billingrule.RuleTypePeriod
	case "monthly":
		ruleType = billingrule.RuleTypeMonthly
	case "coupon":
		ruleType = billingrule.RuleTypeCoupon
	case "vip":
		ruleType = billingrule.RuleTypeVip
	}

	_, err := r.data.db.BillingRule.Create().
		SetID(rule.ID).
		SetLotID(rule.LotID).
		SetRuleName(rule.RuleName).
		SetRuleType(ruleType).
		SetConditionsJSON(rule.Conditions).
		SetActionsJSON(rule.Actions).
		SetPriority(rule.Priority).
		SetIsActive(rule.IsActive).
		Save(ctx)

	return err
}

// UpdateBillingRule updates a billing rule.
func (r *billingRuleRepo) UpdateBillingRule(ctx context.Context, rule *biz.BillingRule) error {
	ruleType := billingrule.RuleTypeTime
	switch rule.RuleType {
	case "period":
		ruleType = billingrule.RuleTypePeriod
	case "monthly":
		ruleType = billingrule.RuleTypeMonthly
	case "coupon":
		ruleType = billingrule.RuleTypeCoupon
	case "vip":
		ruleType = billingrule.RuleTypeVip
	}

	_, err := r.data.db.BillingRule.UpdateOneID(rule.ID).
		SetRuleName(rule.RuleName).
		SetRuleType(ruleType).
		SetConditionsJSON(rule.Conditions).
		SetActionsJSON(rule.Actions).
		SetPriority(rule.Priority).
		SetIsActive(rule.IsActive).
		Save(ctx)

	return err
}

// DeleteBillingRule deletes a billing rule.
func (r *billingRuleRepo) DeleteBillingRule(ctx context.Context, ruleID uuid.UUID) error {
	return r.data.db.BillingRule.DeleteOneID(ruleID).Exec(ctx)
}

// ListBillingRules lists billing rules with pagination.
func (r *billingRuleRepo) ListBillingRules(ctx context.Context, lotID uuid.UUID, page, pageSize int) ([]*biz.BillingRule, int64, error) {
	query := r.data.db.BillingRule.Query().
		Where(billingrule.LotID(lotID))

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rules, err := query.
		Order(ent.Desc(billingrule.FieldPriority)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*biz.BillingRule
	for _, rule := range rules {
		result = append(result, toBizBillingRule(rule))
	}

	return result, int64(total), nil
}

// Helper function to convert ent BillingRule to biz BillingRule.
func toBizBillingRule(rule *ent.BillingRule) *biz.BillingRule {
	return &biz.BillingRule{
		ID:         rule.ID,
		LotID:      rule.LotID,
		RuleName:   rule.RuleName,
		RuleType:   string(rule.RuleType),
		Conditions: rule.ConditionsJSON,
		Actions:    rule.ActionsJSON,
		RuleConfig: rule.RuleConfig,
		Priority:   rule.Priority,
		IsActive:   rule.IsActive,
		CreatedAt:  rule.CreatedAt,
		UpdatedAt:  rule.UpdatedAt,
	}
}

// AdjustPrice adjusts the price of a billing rule dynamically.
// func (r *billingRuleRepo) AdjustPrice(ctx context.Context, ruleID uuid.UUID, priceAdjustment map[string]interface{}) error {
// 	// Get the rule from database
// 	rule, err := r.data.db.BillingRule.Get(ctx, ruleID)
// 	if err != nil {
// 		return err
// 	}

// 	// Parse existing actions
// 	var actions []map[string]interface{}
// 	if err := json.Unmarshal([]byte(rule.ActionsJSON), &actions); err != nil {
// 		return err
// 	}

// 	// Apply price adjustments
// 	for i, action := range actions {
// 		if actionType, ok := action["type"].(string); ok {
// 			switch actionType {
// 			case "fixed", "per_hour", "per_minute":
// 				if amount, ok := priceAdjustment["amount"].(float64); ok {
// 					actions[i]["amount"] = amount
// 				}
// 			case "percentage":
// 				if percent, ok := priceAdjustment["percent"].(float64); ok {
// 					actions[i]["percent"] = percent
// 				}
// 			case "cap":
// 				if cap, ok := priceAdjustment["cap"].(float64); ok {
// 					actions[i]["cap"] = cap
// 				}
// 			case "max_daily":
// 				if amount, ok := priceAdjustment["amount"].(float64); ok {
// 					actions[i]["amount"] = amount
// 				}
// 			case "min_charge":
// 				if amount, ok := priceAdjustment["amount"].(float64); ok {
// 					actions[i]["amount"] = amount
// 				}
// 			}
// 		}
// 	}

// 	// Serialize updated actions back to JSON
// 	updatedActionsJSON, err := json.Marshal(actions)
// 	if err != nil {
// 		return err
// 	}

// 	// Update the rule in database
// 	_, err = r.data.db.BillingRule.UpdateOneID(ruleID).
// 		SetActionsJSON(string(updatedActionsJSON)).
// 		Save(ctx)

// 	return err
// }

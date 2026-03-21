// Package data provides data access layer for the billing service.
package data

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/ent"
	"github.com/xuanyiying/smart-park/ent/billingrule"
	"github.com/xuanyiying/smart-park/internal/billing/biz"
)

// Data holds the data layer dependencies.
type Data struct {
	db  *ent.Client
	log *log.Helper
}

// NewData creates a new Data instance.
func NewData(db *ent.Client, logger log.Logger) (*Data, func(), error) {
	d := &Data{
		db:  db,
		log: log.NewHelper(logger),
	}

	cleanup := func() {
		if err := d.db.Close(); err != nil {
			d.log.Errorf("failed to close database: %v", err)
		}
	}

	return d, cleanup, nil
}

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
		Order(billingrule.Desc(billingrule.FieldPriority)).
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
		Order(billingrule.Desc(billingrule.FieldPriority)).
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

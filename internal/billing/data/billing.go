package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/billing/biz"
	"github.com/xuanyiying/smart-park/internal/billing/data/ent"
	"github.com/xuanyiying/smart-park/internal/billing/data/ent/billingrule"
	"github.com/xuanyiying/smart-park/pkg/database"
)

type billingRuleRepo struct {
	data *Data
}

func NewBillingRuleRepo(data *Data) biz.BillingRuleRepo {
	return &billingRuleRepo{data: data}
}

func (r *billingRuleRepo) GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*biz.BillingRule, error) {
	rules, err := r.clientFromCtx(ctx).BillingRule.Query().
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

func (r *billingRuleRepo) GetBillingRule(ctx context.Context, ruleID uuid.UUID) (*biz.BillingRule, error) {
	rule, err := r.clientFromCtx(ctx).BillingRule.Get(ctx, ruleID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toBizBillingRule(rule), nil
}

func (r *billingRuleRepo) CreateBillingRule(ctx context.Context, rule *biz.BillingRule) error {
	ruleType := billingrule.RuleTypeTime
	switch rule.RuleType {
	case "base":
		ruleType = billingrule.RuleTypeBase
	case "discount":
		ruleType = billingrule.RuleTypeDiscount
	case "exemption":
		ruleType = billingrule.RuleTypeExemption
	case "override":
		ruleType = billingrule.RuleTypeOverride
	case "period":
		ruleType = billingrule.RuleTypePeriod
	case "monthly":
		ruleType = billingrule.RuleTypeMonthly
	case "coupon":
		ruleType = billingrule.RuleTypeCoupon
	case "vip":
		ruleType = billingrule.RuleTypeVip
	}

	_, err := r.clientFromCtx(ctx).BillingRule.Create().
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

func (r *billingRuleRepo) UpdateBillingRule(ctx context.Context, rule *biz.BillingRule) error {
	ruleType := billingrule.RuleTypeTime
	switch rule.RuleType {
	case "base":
		ruleType = billingrule.RuleTypeBase
	case "discount":
		ruleType = billingrule.RuleTypeDiscount
	case "exemption":
		ruleType = billingrule.RuleTypeExemption
	case "override":
		ruleType = billingrule.RuleTypeOverride
	case "period":
		ruleType = billingrule.RuleTypePeriod
	case "monthly":
		ruleType = billingrule.RuleTypeMonthly
	case "coupon":
		ruleType = billingrule.RuleTypeCoupon
	case "vip":
		ruleType = billingrule.RuleTypeVip
	}

	_, err := r.clientFromCtx(ctx).BillingRule.UpdateOneID(rule.ID).
		SetRuleName(rule.RuleName).
		SetRuleType(ruleType).
		SetConditionsJSON(rule.Conditions).
		SetActionsJSON(rule.Actions).
		SetPriority(rule.Priority).
		SetIsActive(rule.IsActive).
		Save(ctx)

	return err
}

func (r *billingRuleRepo) DeleteBillingRule(ctx context.Context, ruleID uuid.UUID) error {
	return r.clientFromCtx(ctx).BillingRule.DeleteOneID(ruleID).Exec(ctx)
}

func (r *billingRuleRepo) ListBillingRules(ctx context.Context, lotID uuid.UUID, page, pageSize int) ([]*biz.BillingRule, int64, error) {
	query := r.clientFromCtx(ctx).BillingRule.Query().
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

func (r *billingRuleRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.data.txm.WithTx(ctx, fn)
}

func (r *billingRuleRepo) clientFromCtx(ctx context.Context) *ent.Client {
	if tx, ok := database.TxFromCtx(ctx).(*ent.Tx); ok {
		return tx.Client()
	}
	return r.data.db
}

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

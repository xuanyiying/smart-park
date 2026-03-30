// Package data provides data access layer for the billing service.
package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/xuanyiying/smart-park/internal/billing/biz"
)

// SeedData creates initial billing rules for development.
func (r *billingRuleRepo) SeedData(ctx context.Context) error {
	// Check if billing rules already exist
	count, err := r.data.db.BillingRule.Query().Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		// Rules already exist, skip seeding
		return nil
	}

	// Create sample billing rules for parking lot 11111111-1111-1111-1111-111111111111
	lotID := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	rules := []*biz.BillingRule{
		{
			ID:         uuid.New(),
			LotID:      lotID,
			RuleName:   "基础计费-小型车",
			RuleType:   "base",
			Conditions: `{"vehicle_type": "small"}`,
			Actions:    `{"base_fee": 5, "hourly_rate": 2, "max_daily": 50}`,
			Priority:   1,
			IsActive:   true,
		},
		{
			ID:         uuid.New(),
			LotID:      lotID,
			RuleName:   "基础计费-大型车",
			RuleType:   "base",
			Conditions: `{"vehicle_type": "large"}`,
			Actions:    `{"base_fee": 10, "hourly_rate": 5, "max_daily": 100}`,
			Priority:   1,
			IsActive:   true,
		},
		{
			ID:         uuid.New(),
			LotID:      lotID,
			RuleName:   "夜间优惠",
			RuleType:   "discount",
			Conditions: `{"start_hour": 22, "end_hour": 6}`,
			Actions:    `{"discount_percent": 50}`,
			Priority:   2,
			IsActive:   true,
		},
		{
			ID:         uuid.New(),
			LotID:      lotID,
			RuleName:   "长时间停车优惠",
			RuleType:   "discount",
			Conditions: `{"duration_min": 480}`,
			Actions:    `{"discount_percent": 20}`,
			Priority:   3,
			IsActive:   true,
		},
		{
			ID:         uuid.New(),
			LotID:      lotID,
			RuleName:   "电动车减免",
			RuleType:   "exemption",
			Conditions: `{"vehicle_type": "electric"}`,
			Actions:    `{"discount_percent": 30}`,
			Priority:   0,
			IsActive:   true,
		},
	}

	for _, rule := range rules {
		if err := r.CreateBillingRule(ctx, rule); err != nil {
			return err
		}
	}

	return nil
}

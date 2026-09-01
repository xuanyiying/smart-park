// Package data provides data access layer for the billing service.
package data

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/billing/biz"
)

// ErrSeedLotNotConfigured is returned when seeding is requested without a target parking
// lot. Seeding used to attach rules to a hard-coded lot id that exists in no real
// deployment, so operators configuring their own lot got the hard-coded fallback tariff
// instead of the rules they thought were loaded.
var ErrSeedLotNotConfigured = errors.New("seed: no target parking lot configured")

// SeedData creates starting billing rules for the given parking lot.
//
// It is idempotent: rules are only created while the table is empty, so restarting the
// service never duplicates them.
//
// The condition and action payloads below use the exact shape the rule engine parses:
// conditions carry a "type" (and "value"/"operator" where applicable) and actions are a
// JSON *array* of typed steps. Earlier seed data stored bare objects of arbitrary keys,
// which every call to ParseConditions/ParseActions rejected, leaving the engine to fall
// back to its hard-coded tariff.
func (r *billingRuleRepo) SeedData(ctx context.Context, lotID uuid.UUID) error {
	if lotID == uuid.Nil {
		return ErrSeedLotNotConfigured
	}

	count, err := r.data.db.BillingRule.Query().Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		// Rules already exist, skip seeding.
		return nil
	}

	rules := []*biz.BillingRule{
		{
			ID:       uuid.New(),
			LotID:    lotID,
			RuleName: "基础计费-小型车",
			RuleType: "base",
			// Matches EvaluateCondition's vehicle_type branch.
			Conditions: `{"type":"vehicle_type","value":"small"}`,
			// 5 元起步 + 每小时 2 元；单日封顶 50 元。
			Actions:  `[{"type":"fixed","amount":5},{"type":"per_hour","amount":2},{"type":"max_daily","amount":50}]`,
			Priority: 1,
			IsActive: true,
		},
		{
			ID:         uuid.New(),
			LotID:      lotID,
			RuleName:   "基础计费-大型车",
			RuleType:   "base",
			Conditions: `{"type":"vehicle_type","value":"large"}`,
			// 10 元起步 + 每小时 5 元；单日封顶 100 元。
			Actions:  `[{"type":"fixed","amount":10},{"type":"per_hour","amount":5},{"type":"max_daily","amount":100}]`,
			Priority: 1,
			IsActive: true,
		},
		{
			ID:       uuid.New(),
			LotID:    lotID,
			RuleName: "夜间优惠",
			RuleType: "discount",
			// 22:00-06:00 wraps midnight; the engine handles start > end.
			Conditions: `{"type":"time_range","value":{"start":22,"end":6}}`,
			// 减免 2 元。
			Actions:  `[{"type":"fixed","amount":2}]`,
			Priority: 2,
			IsActive: true,
		},
		{
			ID:         uuid.New(),
			LotID:      lotID,
			RuleName:   "长时间停车优惠",
			RuleType:   "discount",
			Conditions: `{"type":"duration_min","operator":"gte","value":480}`,
			// 停满 8 小时减免 5 元。
			Actions:  `[{"type":"fixed","amount":5}]`,
			Priority: 3,
			IsActive: true,
		},
		{
			ID:         uuid.New(),
			LotID:      lotID,
			RuleName:   "电动车减免",
			RuleType:   "exemption",
			Conditions: `{"type":"vehicle_type","value":"electric"}`,
			// 减免 2 元。
			Actions:  `[{"type":"fixed","amount":2}]`,
			Priority: 0,
			IsActive: true,
		},
	}

	for _, rule := range rules {
		if err := r.CreateBillingRule(ctx, rule); err != nil {
			return err
		}
	}

	return nil
}

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/xuanyiying/smart-park/internal/billing/biz"
)

const (
	BillingRuleKeyPrefix = "billing_rule:"
	BillingRuleCacheTTL  = 5 * time.Minute
)

type BillingCache struct {
	cache Cache
}

func NewBillingCache(cache Cache) *BillingCache {
	return &BillingCache{cache: cache}
}

func (c *BillingCache) GetBillingRule(ctx context.Context, lotID string, ruleType string) (*biz.BillingRule, error) {
	key := fmt.Sprintf("%s%s:%s", BillingRuleKeyPrefix, lotID, ruleType)
	
	data, err := c.cache.Get(ctx, key)
	if err != nil {
		if err == ErrCacheMiss {
			return nil, ErrCacheMiss
		}
		return nil, err
	}

	var rule biz.BillingRule
	if err := json.Unmarshal([]byte(data), &rule); err != nil {
		return nil, err
	}

	return &rule, nil
}

func (c *BillingCache) SetBillingRule(ctx context.Context, rule *biz.BillingRule) error {
	key := fmt.Sprintf("%s%s:%s", BillingRuleKeyPrefix, rule.LotID.String(), rule.RuleType)
	data, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	return c.cache.Set(ctx, key, string(data), BillingRuleCacheTTL)
}

func (c *BillingCache) DeleteBillingRule(ctx context.Context, lotID string, ruleType string) error {
	key := fmt.Sprintf("%s%s:%s", BillingRuleKeyPrefix, lotID, ruleType)
	return c.cache.Delete(ctx, key)
}

func (c *BillingCache) GetOrLoadBillingRule(ctx context.Context, lotID string, ruleType string, loader func() (*biz.BillingRule, error)) (*biz.BillingRule, error) {
	rule, err := c.GetBillingRule(ctx, lotID, ruleType)
	if err == nil {
		return rule, nil
	}
	if err != ErrCacheMiss {
		return nil, err
	}

	// Load from database
	rule, err = loader()
	if err != nil {
		return nil, err
	}

	// Cache the result
	if err := c.SetBillingRule(ctx, rule); err != nil {
		// Log warning but don't fail
		_ = err
	}

	return rule, nil
}

func (c *BillingCache) InvalidateBillingRule(ctx context.Context, lotID string, ruleType string) error {
	return c.DeleteBillingRule(ctx, lotID, ruleType)
}

func (c *BillingCache) InvalidateAllBillingRules(ctx context.Context, lotID string) error {
	// Note: This requires pattern deletion which is not supported by the Cache interface
	// For now, we just return nil. In production, you might want to use Redis-specific implementation
	return nil
}

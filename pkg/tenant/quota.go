package tenant

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

type QuotaChecker interface {
	CheckQuota(ctx context.Context, tenantID uuid.UUID, resourceType string, currentCount int) error
}

type ResourceQuota struct {
	Current int
	Limit   int
}

type BizQuotaChecker struct {
	uc *biz.TenantUseCase
}

func NewBizQuotaChecker(uc *biz.TenantUseCase) *BizQuotaChecker {
	return &BizQuotaChecker{uc: uc}
}

func (c *BizQuotaChecker) CheckQuota(ctx context.Context, tenantID uuid.UUID, resourceType string, currentCount int) error {
	return c.uc.CheckQuota(ctx, tenantID, resourceType, currentCount)
}

type StaticQuotaChecker struct {
	limits map[string]int
}

func NewStaticQuotaChecker() *StaticQuotaChecker {
	return &StaticQuotaChecker{
		limits: map[string]int{
			"parking_lots": 1,
			"devices":      10,
			"users":        50,
			"storage":      100,
		},
	}
}

func (c *StaticQuotaChecker) CheckQuota(ctx context.Context, tenantID uuid.UUID, resourceType string, currentCount int) error {
	limit, ok := c.limits[resourceType]
	if !ok {
		return fmt.Errorf("unknown resource type: %s", resourceType)
	}
	if currentCount >= limit {
		return biz.ErrQuotaExceeded
	}
	return nil
}

func (c *StaticQuotaChecker) SetLimit(resourceType string, limit int) {
	c.limits[resourceType] = limit
}

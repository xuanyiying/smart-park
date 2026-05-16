package tenant

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

func TestStaticQuotaChecker_CheckQuota_WithinLimit(t *testing.T) {
	checker := NewStaticQuotaChecker()
	tenantID := uuid.New()

	err := checker.CheckQuota(context.Background(), tenantID, "parking_lots", 0)
	assert.NoError(t, err)

	err = checker.CheckQuota(context.Background(), tenantID, "devices", 5)
	assert.NoError(t, err)
}

func TestStaticQuotaChecker_CheckQuota_Exceeded(t *testing.T) {
	checker := NewStaticQuotaChecker()
	tenantID := uuid.New()

	err := checker.CheckQuota(context.Background(), tenantID, "parking_lots", 1)
	assert.ErrorIs(t, err, biz.ErrQuotaExceeded)

	err = checker.CheckQuota(context.Background(), tenantID, "devices", 10)
	assert.ErrorIs(t, err, biz.ErrQuotaExceeded)
}

func TestStaticQuotaChecker_CheckQuota_UnknownResource(t *testing.T) {
	checker := NewStaticQuotaChecker()
	tenantID := uuid.New()

	err := checker.CheckQuota(context.Background(), tenantID, "unknown_resource", 0)
	assert.Error(t, err)
}

func TestStaticQuotaChecker_SetLimit(t *testing.T) {
	checker := NewStaticQuotaChecker()
	checker.SetLimit("parking_lots", 100)

	tenantID := uuid.New()
	err := checker.CheckQuota(context.Background(), tenantID, "parking_lots", 50)
	assert.NoError(t, err)

	err = checker.CheckQuota(context.Background(), tenantID, "parking_lots", 100)
	assert.ErrorIs(t, err, biz.ErrQuotaExceeded)
}

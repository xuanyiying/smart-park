package multitenancy

import (
	"context"

	"github.com/google/uuid"
)

type TenantContext struct {
	ctx context.Context
}

func NewTenantContext(ctx context.Context) *TenantContext {
	return &TenantContext{ctx: ctx}
}

func (tc *TenantContext) GetTenantID() *uuid.UUID {
	return GetTenantID(tc.ctx)
}

func (tc *TenantContext) WithTenantID(tenantID uuid.UUID) context.Context {
	return ContextWithTenant(tc.ctx, &TenantInfo{ID: tenantID})
}

func MustGetTenantID(ctx context.Context) uuid.UUID {
	tenantID := GetTenantID(ctx)
	if tenantID == nil {
		panic("tenant id not found in context")
	}
	return *tenantID
}

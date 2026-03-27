package multitenancy

import (
	"context"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

type ctxKeyTenant struct{}

var tenantCtxKey = ctxKeyTenant{}

type TenantInfo = biz.TenantInfo

func ContextWithTenant(ctx context.Context, tenant *TenantInfo) context.Context {
	return context.WithValue(ctx, tenantCtxKey, tenant)
}

func TenantFromContext(ctx context.Context) (*TenantInfo, bool) {
	tenant, ok := ctx.Value(tenantCtxKey).(*TenantInfo)
	return tenant, ok
}

func GetTenantID(ctx context.Context) *uuid.UUID {
	tenant, ok := TenantFromContext(ctx)
	if !ok || tenant == nil {
		return nil
	}
	return &tenant.ID
}

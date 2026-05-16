package tenant

import (
	"context"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
)

type tenantKey struct{}

var ctxKeyTenant = tenantKey{}

type TenantInfo = biz.TenantInfo

func WithTenant(ctx context.Context, info *TenantInfo) context.Context {
	return context.WithValue(ctx, ctxKeyTenant, info)
}

func FromTenant(ctx context.Context) (*TenantInfo, bool) {
	info, ok := ctx.Value(ctxKeyTenant).(*TenantInfo)
	return info, ok
}

func MustFromTenant(ctx context.Context) *TenantInfo {
	info, ok := FromTenant(ctx)
	if !ok || info == nil {
		panic("tenant: tenant info not found in context")
	}
	return info
}

func TenantIDFromCtx(ctx context.Context) uuid.UUID {
	info, ok := FromTenant(ctx)
	if !ok || info == nil {
		return uuid.Nil
	}
	return info.ID
}

package tenant

import (
	"context"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

func TenantFilter(ctx context.Context, field string) func(*sql.Selector) {
	tenantID := TenantIDFromCtx(ctx)
	if tenantID == uuid.Nil {
		return nil
	}
	if field == "" {
		field = "tenant_id"
	}
	return sql.FieldEQ(field, tenantID.String())
}

func WithTenantScope(ctx context.Context) bool {
	_, ok := FromTenant(ctx)
	return ok
}

func ApplyTenantFilter(ctx context.Context, s *sql.Selector, tenantField string) {
	tenantID := TenantIDFromCtx(ctx)
	if tenantID == uuid.Nil {
		return
	}
	if tenantField == "" {
		tenantField = "tenant_id"
	}
	s.Where(sql.EQ(tenantField, tenantID.String()))
}

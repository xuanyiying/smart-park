package data

import (
	"context"

	"entgo.io/ent"
	"github.com/google/uuid"

	userent "github.com/xuanyiying/smart-park/internal/user/data/ent"
	"github.com/xuanyiying/smart-park/internal/user/data/ent/user"
	"github.com/xuanyiying/smart-park/internal/user/data/ent/uservehicle"
	"github.com/xuanyiying/smart-park/pkg/tenant"
)

// TenantScopes restricts user queries to the tenant carried by the context.
//
// Accounts and their bound plates are tenant owned; leaking them would expose
// both personal data and the vehicle list used for recognition at the barrier.
func TenantScopes() []tenant.ScopeFunc {
	return []tenant.ScopeFunc{
		func(ctx context.Context, q ent.Query, tenantID uuid.UUID) {
			switch tq := q.(type) {
			case *userent.UserQuery:
				tq.Where(user.TenantID(tenantID))
			case *userent.UserVehicleQuery:
				tq.Where(uservehicle.TenantID(tenantID))
			}
		},
	}
}

// TenantTypes lists the user schemas that own a tenant_id column.
func TenantTypes() []string {
	return []string{
		userent.TypeUser,
		userent.TypeUserVehicle,
	}
}

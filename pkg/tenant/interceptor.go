package tenant

import (
	"context"

	"entgo.io/ent"
	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
)

// TenantIDField is the column name shared by every tenant scoped table.
const TenantIDField = "tenant_id"

// ScopeFunc injects a tenant predicate into a concrete query builder.
//
// Ent generates one query type per schema per service (e.g. *ent.VehicleQuery in
// the vehicle service), so a package under pkg/ cannot address them all by name.
// Each service therefore hands its own typed predicates to TenantQueryInterceptor,
// which keeps the "read the tenant from the context, skip when absent" policy in
// one place while leaving the type-aware part with the owner of the schema.
type ScopeFunc func(ctx context.Context, q ent.Query, tenantID uuid.UUID)

// TenantQueryInterceptor returns an ent query interceptor that automatically
// restricts queries to the tenant carried by the context.
//
// Previously every repository had to remember to call TenantFilter by hand; a
// query that forgot to do so silently returned every tenant's rows. Registering
// this interceptor on the client closes that gap, because Ent applies it to
// every query built through the client, including Query().Count() and the
// queries issued inside eager-loaded edges.
//
// Requests without a tenant in the context are passed through untouched: system
// jobs, seeding and cross-tenant administration still need to read across
// tenants. Only the transport middlewares put a tenant into the context, so
// anything reaching the repositories from a tenant scoped request is filtered.
func TenantQueryInterceptor(scopes ...ScopeFunc) ent.Interceptor {
	return ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(ctx context.Context, q ent.Query) (ent.Value, error) {
			if tenantID := TenantIDFromCtx(ctx); tenantID != uuid.Nil {
				for _, scope := range scopes {
					scope(ctx, q, tenantID)
				}
			}
			return next.Query(ctx, q)
		})
	})
}

// TenantMutationHook returns an ent mutation hook that keeps writes inside the
// tenant boundary.
//
// On create it fills tenant_id from the context for the entity types listed in
// tenantTypes, so a repository cannot accidentally persist a row without an
// owner. On update and delete it appends tenant_id to the statement's WHERE
// clause, which turns a cross-tenant write into a not-found error instead of a
// silent overwrite of another tenant's row.
//
// tenantTypes names the Ent schema types that own a tenant_id column (for
// example "Vehicle"). Types outside the list keep global semantics, which is why
// the caller supplies them rather than the hook guessing from a field lookup.
func TenantMutationHook(tenantTypes ...string) ent.Hook {
	scoped := make(map[string]bool, len(tenantTypes))
	for _, t := range tenantTypes {
		scoped[t] = true
	}

	return func(next ent.Mutator) ent.Mutator {
		return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
			tenantID := TenantIDFromCtx(ctx)
			if tenantID == uuid.Nil || !scoped[m.Type()] {
				return next.Mutate(ctx, m)
			}

			switch m.Op() {
			case ent.OpCreate:
				// Only fill when the caller did not set it explicitly, so an
				// intentional cross-tenant insert stays explicit.
				if _, set := m.Field(TenantIDField); !set {
					// SetField fails for schemas without the column; those types
					// are not in tenantTypes, so the error is unreachable here.
					_ = m.SetField(TenantIDField, tenantID)
				}
			case ent.OpUpdate, ent.OpUpdateOne, ent.OpDelete, ent.OpDeleteOne:
				if wm, ok := m.(interface {
					WhereP(...func(*sql.Selector))
				}); ok {
					wm.WhereP(func(s *sql.Selector) {
						s.Where(sql.EQ(TenantIDField, tenantID.String()))
					})
				}
			}

			return next.Mutate(ctx, m)
		})
	}
}

// ApplyTenantScoping registers tenant isolation on an Ent client.
//
// It is a thin convenience wrapper so each service wires reads and writes the
// same way: interceptors for the read path, a mutation hook for the write path.
// client is any *ent.Client generated per service, hence the generic signature.
func ApplyTenantScoping[C interface {
	Intercept(...ent.Interceptor)
	Use(...ent.Hook)
}](client C, scopes []ScopeFunc, tenantTypes []string) {
	client.Intercept(TenantQueryInterceptor(scopes...))
	client.Use(TenantMutationHook(tenantTypes...))
}

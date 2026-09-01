package data

import (
	"context"

	"entgo.io/ent"
	"github.com/google/uuid"

	billingent "github.com/xuanyiying/smart-park/internal/billing/data/ent"
	"github.com/xuanyiying/smart-park/internal/billing/data/ent/billingrule"
	"github.com/xuanyiying/smart-park/pkg/tenant"
)

// TenantScopes restricts billing queries to the tenant carried by the context.
//
// Billing rules are configured per tenant, so leaking one tenant's tariff into
// another's fee calculation would both expose pricing and mischarge vehicles.
func TenantScopes() []tenant.ScopeFunc {
	return []tenant.ScopeFunc{
		func(ctx context.Context, q ent.Query, tenantID uuid.UUID) {
			if bq, ok := q.(*billingent.BillingRuleQuery); ok {
				bq.Where(billingrule.TenantID(tenantID))
			}
		},
	}
}

// TenantTypes lists the billing schemas that own a tenant_id column.
func TenantTypes() []string {
	return []string{billingent.TypeBillingRule}
}

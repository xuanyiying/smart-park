package data

import (
	"context"

	"entgo.io/ent"
	"github.com/google/uuid"

	chargingt "github.com/xuanyiying/smart-park/internal/charging/data/ent"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent/connector"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent/price"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent/session"
	"github.com/xuanyiying/smart-park/internal/charging/data/ent/station"
	"github.com/xuanyiying/smart-park/pkg/tenant"
)

// TenantScopes restricts charging queries to the tenant carried by the context.
//
// Stations, connectors, prices and charging sessions all belong to a single
// tenant, so an unscoped query would expose another operator's infrastructure
// and revenue.
func TenantScopes() []tenant.ScopeFunc {
	return []tenant.ScopeFunc{
		func(ctx context.Context, q ent.Query, tenantID uuid.UUID) {
			switch tq := q.(type) {
			case *chargingt.ConnectorQuery:
				tq.Where(connector.TenantID(tenantID))
			case *chargingt.PriceQuery:
				tq.Where(price.TenantID(tenantID))
			case *chargingt.SessionQuery:
				tq.Where(session.TenantID(tenantID))
			case *chargingt.StationQuery:
				tq.Where(station.TenantID(tenantID))
			}
		},
	}
}

// TenantTypes lists the charging schemas that own a tenant_id column.
func TenantTypes() []string {
	return []string{
		chargingt.TypeConnector,
		chargingt.TypePrice,
		chargingt.TypeSession,
		chargingt.TypeStation,
	}
}

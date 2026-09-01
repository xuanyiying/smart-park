package data

import (
	"context"

	"entgo.io/ent"
	"github.com/google/uuid"

	adminent "github.com/xuanyiying/smart-park/internal/admin/data/ent"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/order"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/parkinglot"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/parkingrecord"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/user"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/vehicle"
	"github.com/xuanyiying/smart-park/pkg/tenant"
)

// TenantScopes restricts admin queries to the tenant carried by the context.
//
// The admin API lists parking lots, vehicles, parking records, orders and users.
// Each of those tables is tenant owned, so an unscoped query here would hand one
// operator the full customer base of every tenant on the deployment.
func TenantScopes() []tenant.ScopeFunc {
	return []tenant.ScopeFunc{
		func(ctx context.Context, q ent.Query, tenantID uuid.UUID) {
			switch tq := q.(type) {
			case *adminent.OrderQuery:
				tq.Where(order.TenantID(tenantID))
			case *adminent.ParkingLotQuery:
				tq.Where(parkinglot.TenantID(tenantID))
			case *adminent.ParkingRecordQuery:
				tq.Where(parkingrecord.TenantID(tenantID))
			case *adminent.UserQuery:
				tq.Where(user.TenantID(tenantID))
			case *adminent.VehicleQuery:
				tq.Where(vehicle.TenantID(tenantID))
			}
		},
	}
}

// TenantTypes lists the admin schemas that own a tenant_id column.
func TenantTypes() []string {
	return []string{
		adminent.TypeOrder,
		adminent.TypeParkingLot,
		adminent.TypeParkingRecord,
		adminent.TypeUser,
		adminent.TypeVehicle,
	}
}

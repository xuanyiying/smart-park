package data

import (
	"context"

	"entgo.io/ent"
	"github.com/google/uuid"

	vehicleent "github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/blacklistentry"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/billingrule"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/device"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/lane"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/offlinesyncrecord"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/parkingrecord"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/vehicle"
	"github.com/xuanyiying/smart-park/pkg/tenant"
)

// TenantScopes restricts vehicle queries to the tenant carried by the context.
//
// These tables describe a tenant's physical estate: lanes, devices, vehicles and
// the parking records produced at the barrier. Device faults, firmware and
// manufacturers are global reference data and are deliberately left unscoped.
func TenantScopes() []tenant.ScopeFunc {
	return []tenant.ScopeFunc{
		func(ctx context.Context, q ent.Query, tenantID uuid.UUID) {
			switch tq := q.(type) {
			case *vehicleent.BillingRuleQuery:
				tq.Where(billingrule.TenantID(tenantID))
			case *vehicleent.BlacklistEntryQuery:
				tq.Where(blacklistentry.TenantID(tenantID))
			case *vehicleent.DeviceQuery:
				tq.Where(device.TenantID(tenantID))
			case *vehicleent.LaneQuery:
				tq.Where(lane.TenantID(tenantID))
			case *vehicleent.OfflineSyncRecordQuery:
				tq.Where(offlinesyncrecord.TenantID(tenantID))
			case *vehicleent.ParkingRecordQuery:
				tq.Where(parkingrecord.TenantID(tenantID))
			case *vehicleent.VehicleQuery:
				tq.Where(vehicle.TenantID(tenantID))
			}
		},
	}
}

// TenantTypes lists the vehicle schemas that own a tenant_id column.
func TenantTypes() []string {
	return []string{
		vehicleent.TypeBillingRule,
		vehicleent.TypeBlacklistEntry,
		vehicleent.TypeDevice,
		vehicleent.TypeLane,
		vehicleent.TypeOfflineSyncRecord,
		vehicleent.TypeParkingRecord,
		vehicleent.TypeVehicle,
	}
}

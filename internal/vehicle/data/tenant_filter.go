package data

import (
	"context"

	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/device"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/lane"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/offlinesyncrecord"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/parkingrecord"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/vehicle"
	"github.com/xuanyiying/smart-park/pkg/multitenancy"
)

type TenantFilter struct {
	ctx context.Context
}

func NewTenantFilter(ctx context.Context) *TenantFilter {
	return &TenantFilter{ctx: ctx}
}

func (tf *TenantFilter) FilterVehicleQuery(query *ent.VehicleQuery) *ent.VehicleQuery {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return query.Where(vehicle.TenantID(*tenantID))
	}
	return query
}

func (tf *TenantFilter) FilterParkingRecordQuery(query *ent.ParkingRecordQuery) *ent.ParkingRecordQuery {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return query.Where(parkingrecord.TenantID(*tenantID))
	}
	return query
}

func (tf *TenantFilter) FilterDeviceQuery(query *ent.DeviceQuery) *ent.DeviceQuery {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return query.Where(device.TenantID(*tenantID))
	}
	return query
}

func (tf *TenantFilter) FilterLaneQuery(query *ent.LaneQuery) *ent.LaneQuery {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return query.Where(lane.TenantID(*tenantID))
	}
	return query
}

func (tf *TenantFilter) FilterOfflineSyncRecordQuery(query *ent.OfflineSyncRecordQuery) *ent.OfflineSyncRecordQuery {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return query.Where(offlinesyncrecord.TenantID(*tenantID))
	}
	return query
}

func (tf *TenantFilter) ApplyTenantIDToVehicleCreate(create *ent.VehicleCreate) *ent.VehicleCreate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return create.SetTenantID(*tenantID)
	}
	return create
}

func (tf *TenantFilter) ApplyTenantIDToParkingRecordCreate(create *ent.ParkingRecordCreate) *ent.ParkingRecordCreate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return create.SetTenantID(*tenantID)
	}
	return create
}

func (tf *TenantFilter) ApplyTenantIDToDeviceCreate(create *ent.DeviceCreate) *ent.DeviceCreate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return create.SetTenantID(*tenantID)
	}
	return create
}

func (tf *TenantFilter) ApplyTenantIDToLaneCreate(create *ent.LaneCreate) *ent.LaneCreate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return create.SetTenantID(*tenantID)
	}
	return create
}

func (tf *TenantFilter) ApplyTenantIDToOfflineSyncRecordCreate(create *ent.OfflineSyncRecordCreate) *ent.OfflineSyncRecordCreate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return create.SetTenantID(*tenantID)
	}
	return create
}

func (tf *TenantFilter) FilterVehicleUpdate(update *ent.VehicleUpdate) *ent.VehicleUpdate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(vehicle.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterVehicleUpdateOne(update *ent.VehicleUpdateOne) *ent.VehicleUpdateOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(vehicle.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterParkingRecordUpdate(update *ent.ParkingRecordUpdate) *ent.ParkingRecordUpdate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(parkingrecord.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterParkingRecordUpdateOne(update *ent.ParkingRecordUpdateOne) *ent.ParkingRecordUpdateOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(parkingrecord.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterDeviceUpdate(update *ent.DeviceUpdate) *ent.DeviceUpdate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(device.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterDeviceUpdateOne(update *ent.DeviceUpdateOne) *ent.DeviceUpdateOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(device.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterLaneUpdate(update *ent.LaneUpdate) *ent.LaneUpdate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(lane.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterLaneUpdateOne(update *ent.LaneUpdateOne) *ent.LaneUpdateOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(lane.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterOfflineSyncRecordUpdate(update *ent.OfflineSyncRecordUpdate) *ent.OfflineSyncRecordUpdate {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(offlinesyncrecord.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterOfflineSyncRecordUpdateOne(update *ent.OfflineSyncRecordUpdateOne) *ent.OfflineSyncRecordUpdateOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return update.Where(offlinesyncrecord.TenantID(*tenantID))
	}
	return update
}

func (tf *TenantFilter) FilterVehicleDelete(delete *ent.VehicleDelete) *ent.VehicleDelete {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(vehicle.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterVehicleDeleteOne(delete *ent.VehicleDeleteOne) *ent.VehicleDeleteOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(vehicle.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterParkingRecordDelete(delete *ent.ParkingRecordDelete) *ent.ParkingRecordDelete {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(parkingrecord.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterParkingRecordDeleteOne(delete *ent.ParkingRecordDeleteOne) *ent.ParkingRecordDeleteOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(parkingrecord.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterDeviceDelete(delete *ent.DeviceDelete) *ent.DeviceDelete {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(device.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterDeviceDeleteOne(delete *ent.DeviceDeleteOne) *ent.DeviceDeleteOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(device.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterLaneDelete(delete *ent.LaneDelete) *ent.LaneDelete {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(lane.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterLaneDeleteOne(delete *ent.LaneDeleteOne) *ent.LaneDeleteOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(lane.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterOfflineSyncRecordDelete(delete *ent.OfflineSyncRecordDelete) *ent.OfflineSyncRecordDelete {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(offlinesyncrecord.TenantID(*tenantID))
	}
	return delete
}

func (tf *TenantFilter) FilterOfflineSyncRecordDeleteOne(delete *ent.OfflineSyncRecordDeleteOne) *ent.OfflineSyncRecordDeleteOne {
	if tenantID := multitenancy.GetTenantID(tf.ctx); tenantID != nil {
		return delete.Where(offlinesyncrecord.TenantID(*tenantID))
	}
	return delete
}

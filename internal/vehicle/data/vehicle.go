// Package data provides data access layer for the vehicle service.
package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/device"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/offlinesyncrecord"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/parkingrecord"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/vehicle"
	"github.com/xuanyiying/smart-park/pkg/multitenancy"
)

// NewData creates a new Data instance.
func NewData(db *ent.Client, logger log.Logger) (*Data, func(), error) {
	d := &Data{
		db:  db,
		log: log.NewHelper(logger),
	}

	cleanup := func() {
		if err := d.db.Close(); err != nil {
			d.log.Errorf("failed to close database: %v", err)
		}
	}

	return d, cleanup, nil
}

// vehicleRepo implements biz.VehicleRepo.
type vehicleRepo struct {
	data *Data
}

// NewVehicleRepo creates a new VehicleRepo.
func NewVehicleRepo(data *Data) biz.VehicleRepo {
	return &vehicleRepo{data: data}
}

// GetVehicleByPlate retrieves a vehicle by plate number.
func (r *vehicleRepo) GetVehicleByPlate(ctx context.Context, plateNumber string) (*biz.Vehicle, error) {
	query := r.clientFromCtx(ctx).Vehicle.Query().
		Where(vehicle.PlateNumber(plateNumber))
	
	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(vehicle.TenantID(*tenantID))
	}
	
	v, err := query.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Vehicle{
		ID:                v.ID,
		PlateNumber:       v.PlateNumber,
		VehicleType:       string(v.VehicleType),
		OwnerName:         v.OwnerName,
		OwnerPhone:        v.OwnerPhone,
		MonthlyValidUntil: v.MonthlyValidUntil,
		CreatedAt:         v.CreatedAt,
		UpdatedAt:         v.UpdatedAt,
	}, nil
}

// CreateVehicle creates a new vehicle.
func (r *vehicleRepo) CreateVehicle(ctx context.Context, v *biz.Vehicle) error {
	vehicleType := vehicle.VehicleTypeTemporary
	switch v.VehicleType {
	case "monthly":
		vehicleType = vehicle.VehicleTypeMonthly
	case "vip":
		vehicleType = vehicle.VehicleTypeVip
	}

	create := r.data.db.Vehicle.Create().
		SetPlateNumber(v.PlateNumber).
		SetVehicleType(vehicleType).
		SetOwnerName(v.OwnerName).
		SetOwnerPhone(v.OwnerPhone)

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		create.SetTenantID(*tenantID)
	}

	if v.MonthlyValidUntil != nil {
		create.SetMonthlyValidUntil(*v.MonthlyValidUntil)
	}

	_, err := create.Save(ctx)
	return err
}

// UpdateVehicle updates a vehicle.
func (r *vehicleRepo) UpdateVehicle(ctx context.Context, v *biz.Vehicle) error {
	vehicleType := vehicle.VehicleTypeTemporary
	switch v.VehicleType {
	case "monthly":
		vehicleType = vehicle.VehicleTypeMonthly
	case "vip":
		vehicleType = vehicle.VehicleTypeVip
	}

	update := r.data.db.Vehicle.UpdateOneID(v.ID).
		SetVehicleType(vehicleType).
		SetOwnerName(v.OwnerName).
		SetOwnerPhone(v.OwnerPhone)

	if v.MonthlyValidUntil != nil {
		update.SetMonthlyValidUntil(*v.MonthlyValidUntil)
	}

	_, err := update.Save(ctx)
	return err
}

// GetEntryRecord retrieves an active entry record by plate number.
func (r *vehicleRepo) GetEntryRecord(ctx context.Context, plateNumber string) (*biz.ParkingRecord, error) {
	query := r.clientFromCtx(ctx).ParkingRecord.Query().
		Where(
			parkingrecord.PlateNumber(plateNumber),
			parkingrecord.RecordStatusIn(parkingrecord.RecordStatusEntry, parkingrecord.RecordStatusExiting),
		)

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(parkingrecord.TenantID(*tenantID))
	}

	record, err := query.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toBizParkingRecord(record), nil
}

// CreateParkingRecord creates a new parking record.
func (r *vehicleRepo) CreateParkingRecord(ctx context.Context, rec *biz.ParkingRecord) error {
	create := r.clientFromCtx(ctx).ParkingRecord.Create().
		SetID(rec.ID).
		SetLotID(rec.LotID).
		SetEntryLaneID(rec.EntryLaneID).
		SetEntryTime(rec.EntryTime).
		SetEntryImageURL(rec.EntryImageURL).
		SetRecordStatus(parkingrecord.RecordStatusEntry).
		SetExitStatus(parkingrecord.ExitStatusUnpaid)

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		create.SetTenantID(*tenantID)
	}

	if rec.VehicleID != nil {
		create.SetVehicleID(*rec.VehicleID)
	}
	if rec.PlateNumber != nil {
		create.SetPlateNumber(*rec.PlateNumber)
	}
	if rec.PlateNumberSource != "" {
		switch rec.PlateNumberSource {
		case "camera":
			create.SetPlateNumberSource(parkingrecord.PlateNumberSourceCamera)
		case "manual":
			create.SetPlateNumberSource(parkingrecord.PlateNumberSourceManual)
		case "offline":
			create.SetPlateNumberSource(parkingrecord.PlateNumberSourceOffline)
		}
	}

	_, err := create.Save(ctx)
	return err
}

// UpdateParkingRecord updates a parking record.
func (r *vehicleRepo) UpdateParkingRecord(ctx context.Context, rec *biz.ParkingRecord) error {
	update := r.clientFromCtx(ctx).ParkingRecord.UpdateOneID(rec.ID)

	if rec.ExitTime != nil {
		update.SetExitTime(*rec.ExitTime)
	}
	if rec.ExitImageURL != "" {
		update.SetExitImageURL(rec.ExitImageURL)
	}
	if rec.ExitLaneID != nil {
		update.SetExitLaneID(*rec.ExitLaneID)
	}
	if rec.ExitDeviceID != "" {
		update.SetExitDeviceID(rec.ExitDeviceID)
	}
	if rec.ParkingDuration > 0 {
		update.SetParkingDuration(rec.ParkingDuration)
	}
	if rec.RecordStatus != "" {
		switch rec.RecordStatus {
		case "entry":
			update.SetRecordStatus(parkingrecord.RecordStatusEntry)
		case "exiting":
			update.SetRecordStatus(parkingrecord.RecordStatusExiting)
		case "exited":
			update.SetRecordStatus(parkingrecord.RecordStatusExited)
		case "paid":
			update.SetRecordStatus(parkingrecord.RecordStatusPaid)
		}
	}

	_, err := update.Save(ctx)
	return err
}

// GetParkingRecord retrieves a parking record by ID.
func (r *vehicleRepo) GetParkingRecord(ctx context.Context, recordID uuid.UUID) (*biz.ParkingRecord, error) {
	query := r.data.db.ParkingRecord.Query().
		Where(parkingrecord.ID(recordID))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(parkingrecord.TenantID(*tenantID))
	}

	record, err := query.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toBizParkingRecord(record), nil
}

// ListParkingRecordsByPlates retrieves parking records by plate numbers with pagination.
func (r *vehicleRepo) ListParkingRecordsByPlates(ctx context.Context, plateNumbers []string, page, pageSize int) ([]*biz.ParkingRecord, int, error) {
	if len(plateNumbers) == 0 {
		return []*biz.ParkingRecord{}, 0, nil
	}

	// Build query
	query := r.data.db.ParkingRecord.Query().
		Where(parkingrecord.PlateNumberIn(plateNumbers...))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(parkingrecord.TenantID(*tenantID))
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	records, err := query.
		Order(ent.Desc(parkingrecord.FieldEntryTime)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to biz entities
	result := make([]*biz.ParkingRecord, len(records))
	for i, record := range records {
		result[i] = toBizParkingRecord(record)
	}

	return result, total, nil
}

// GetDeviceByCode retrieves a device by device code.
func (r *vehicleRepo) GetDeviceByCode(ctx context.Context, deviceCode string) (*biz.Device, error) {
	query := r.data.db.Device.Query().
		Where(device.DeviceID(deviceCode))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(device.TenantID(*tenantID))
	}

	d, err := query.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Device{
		ID:            d.ID,
		DeviceID:      d.DeviceID,
		LotID:         d.LotID,
		LaneID:        d.LaneID,
		DeviceType:    string(d.DeviceType),
		DeviceSecret:  d.DeviceSecret,
		GateID:        d.GateID,
		Enabled:       d.Enabled,
		Status:        string(d.Status),
		LastHeartbeat: d.LastHeartbeat,
	}, nil
}

// UpdateDeviceHeartbeat updates device heartbeat.
func (r *vehicleRepo) UpdateDeviceHeartbeat(ctx context.Context, deviceCode string) error {
	update := r.data.db.Device.Update().
		Where(device.DeviceID(deviceCode))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		update = update.Where(device.TenantID(*tenantID))
	}

	return update.
		SetLastHeartbeat(time.Now()).
		SetStatus(device.StatusActive).
		Exec(ctx)
}

// ListDevices retrieves all devices with pagination.
func (r *vehicleRepo) ListDevices(ctx context.Context, page, pageSize int) ([]*biz.Device, int, error) {
	query := r.data.db.Device.Query()

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(device.TenantID(*tenantID))
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	devices, err := query.
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to biz entities
	result := make([]*biz.Device, len(devices))
	for i, d := range devices {
		result[i] = &biz.Device{
			ID:            d.ID,
			DeviceID:      d.DeviceID,
			LotID:         d.LotID,
			LaneID:        d.LaneID,
			DeviceType:    string(d.DeviceType),
			DeviceSecret:  d.DeviceSecret,
			GateID:        d.GateID,
			Enabled:       d.Enabled,
			Status:        string(d.Status),
			LastHeartbeat: d.LastHeartbeat,
		}
	}

	return result, total, nil
}

// GetLaneByDeviceCode retrieves a lane by device code.
func (r *vehicleRepo) GetLaneByDeviceCode(ctx context.Context, deviceCode string) (*biz.Lane, error) {
	// First get the device
	query := r.data.db.Device.Query().
		Where(device.DeviceID(deviceCode))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(device.TenantID(*tenantID))
	}

	d, err := query.Only(ctx)
	if err != nil {
		return nil, err
	}

	if d.LaneID == nil {
		return nil, nil
	}

	// Then get the lane
	l, err := r.data.db.Lane.Get(ctx, *d.LaneID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Lane{
		ID:           l.ID,
		LotID:        l.LotID,
		LaneNo:       l.LaneNo,
		Direction:    string(l.Direction),
		Status:       string(l.Status),
		DeviceConfig: l.DeviceConfig,
	}, nil
}

// Helper function to convert ent ParkingRecord to biz ParkingRecord.
func toBizParkingRecord(record *ent.ParkingRecord) *biz.ParkingRecord {
	return &biz.ParkingRecord{
		ID:                record.ID,
		LotID:             record.LotID,
		EntryLaneID:       record.EntryLaneID,
		VehicleID:         record.VehicleID,
		PlateNumber:       record.PlateNumber,
		PlateNumberSource: string(record.PlateNumberSource),
		EntryTime:         record.EntryTime,
		EntryImageURL:     record.EntryImageURL,
		RecordStatus:      string(record.RecordStatus),
		ExitTime:          record.ExitTime,
		ExitImageURL:      record.ExitImageURL,
		ExitLaneID:        record.ExitLaneID,
		ExitDeviceID:      record.ExitDeviceID,
		ParkingDuration:   record.ParkingDuration,
		ExitStatus:        string(record.ExitStatus),
		PaymentLock:       record.PaymentLock,
		Metadata:          record.RecordMetadata,
		CreatedAt:         record.CreatedAt,
		UpdatedAt:         record.UpdatedAt,
	}
}

type txCtxKey struct{}

// WithTx executes a function within a transaction.
// The tx client is injected into context so downstream operations use the transaction.
func (r *vehicleRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()

	// Inject tx client into context so all downstream DB operations use it
	txCtx := context.WithValue(ctx, txCtxKey{}, tx)

	if err := fn(txCtx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return rerr
		}
		return err
	}

	return tx.Commit()
}

// clientFromCtx returns the ent client from context (tx client if in transaction, otherwise original).
func (r *vehicleRepo) clientFromCtx(ctx context.Context) *ent.Client {
	if tx, ok := ctx.Value(txCtxKey{}).(*ent.Tx); ok {
		return tx.Client()
	}
	return r.data.db
}

// getTenantID gets tenant ID from context
func (r *vehicleRepo) getTenantID(ctx context.Context) *uuid.UUID {
	return multitenancy.GetTenantID(ctx)
}

// CreateOfflineSyncRecord creates an offline sync record.
func (r *vehicleRepo) CreateOfflineSyncRecord(ctx context.Context, record *biz.OfflineSyncRecord) error {
	create := r.data.db.OfflineSyncRecord.Create().
		SetOfflineID(record.OfflineID).
		SetRecordID(record.RecordID).
		SetLotID(record.LotID).
		SetDeviceID(record.DeviceID).
		SetGateID(record.GateID).
		SetOpenTime(record.OpenTime).
		SetSyncAmount(record.SyncAmount).
		SetSyncStatus(offlinesyncrecord.SyncStatusPendingSync).
		SetRetryCount(0)

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		create.SetTenantID(*tenantID)
	}

	_, err := create.Save(ctx)

	return err
}

// GetPendingSyncRecords retrieves pending sync records.
func (r *vehicleRepo) GetPendingSyncRecords(ctx context.Context, limit int) ([]*biz.OfflineSyncRecord, error) {
	query := r.data.db.OfflineSyncRecord.Query().
		Where(offlinesyncrecord.SyncStatusEQ(offlinesyncrecord.SyncStatusPendingSync)).
		Where(offlinesyncrecord.RetryCountLT(5))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(offlinesyncrecord.TenantID(*tenantID))
	}

	records, err := query.
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.OfflineSyncRecord, len(records))
	for i, rec := range records {
		result[i] = toBizOfflineSyncRecord(rec)
	}
	return result, nil
}

// UpdateOfflineSyncRecord updates an offline sync record.
func (r *vehicleRepo) UpdateOfflineSyncRecord(ctx context.Context, record *biz.OfflineSyncRecord) error {
	update := r.data.db.OfflineSyncRecord.Update().
		Where(offlinesyncrecord.OfflineID(record.OfflineID))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		update = update.Where(offlinesyncrecord.TenantID(*tenantID))
	}

	update = update.
		SetSyncStatus(offlinesyncrecord.SyncStatus(record.SyncStatus)).
		SetSyncError(record.SyncError).
		SetRetryCount(record.RetryCount)

	if record.SyncedAt != nil {
		update.SetSyncedAt(*record.SyncedAt)
	}

	_, err := update.Save(ctx)
	return err
}

func toBizOfflineSyncRecord(record *ent.OfflineSyncRecord) *biz.OfflineSyncRecord {
	bizRecord := &biz.OfflineSyncRecord{
		ID:         record.ID,
		OfflineID:  record.OfflineID,
		DeviceID:   record.DeviceID,
		GateID:     record.GateID,
		OpenTime:   record.OpenTime,
		SyncAmount: record.SyncAmount,
		SyncStatus: string(record.SyncStatus),
		SyncError:  record.SyncError,
		RetryCount: record.RetryCount,
		SyncedAt:   record.SyncedAt,
		CreatedAt:  record.CreatedAt,
	}
	if record.RecordID != nil {
		bizRecord.RecordID = *record.RecordID
	}
	if record.LotID != nil {
		bizRecord.LotID = *record.LotID
	}
	return bizRecord
}

// GetDeviceByID retrieves a device by device ID.
func (r *vehicleRepo) GetDeviceByID(ctx context.Context, deviceID string) (*biz.Device, error) {
	query := r.data.db.Device.Query().
		Where(device.DeviceID(deviceID))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		query = query.Where(device.TenantID(*tenantID))
	}

	d, err := query.Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Device{
		ID:            d.ID,
		DeviceID:      d.DeviceID,
		LotID:         d.LotID,
		LaneID:        d.LaneID,
		DeviceType:    string(d.DeviceType),
		DeviceSecret:  d.DeviceSecret,
		GateID:        d.GateID,
		Enabled:       d.Enabled,
		Status:        string(d.Status),
		LastHeartbeat: d.LastHeartbeat,
	}, nil
}

// CreateDevice creates a new device.
func (r *vehicleRepo) CreateDevice(ctx context.Context, d *biz.Device) error {
	deviceType := device.DeviceTypeCamera
	switch d.DeviceType {
	case "gate":
		deviceType = device.DeviceTypeGate
	case "display":
		deviceType = device.DeviceTypeDisplay
	case "payment_kiosk":
		deviceType = device.DeviceTypePaymentKiosk
	case "sensor":
		deviceType = device.DeviceTypeSensor
	}

	status := device.StatusActive
	switch d.Status {
	case "offline":
		status = device.StatusOffline
	case "disabled":
		status = device.StatusDisabled
	}

	// Generate device secret if not provided
	deviceSecret := d.DeviceSecret
	if deviceSecret == "" {
		deviceSecret = "secret_" + d.DeviceID
	}

	create := r.data.db.Device.Create().
		SetDeviceID(d.DeviceID).
		SetDeviceSecret(deviceSecret).
		SetDeviceType(deviceType).
		SetStatus(status)

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		create.SetTenantID(*tenantID)
	}

	if d.LotID != nil {
		create.SetLotID(*d.LotID)
	}
	if d.LaneID != nil {
		create.SetLaneID(*d.LaneID)
	}

	_, err := create.Save(ctx)
	return err
}

// UpdateDevice updates an existing device.
func (r *vehicleRepo) UpdateDevice(ctx context.Context, d *biz.Device) error {
	deviceType := device.DeviceTypeCamera
	switch d.DeviceType {
	case "gate":
		deviceType = device.DeviceTypeGate
	case "display":
		deviceType = device.DeviceTypeDisplay
	case "payment_kiosk":
		deviceType = device.DeviceTypePaymentKiosk
	case "sensor":
		deviceType = device.DeviceTypeSensor
	}

	status := device.StatusActive
	switch d.Status {
	case "offline":
		status = device.StatusOffline
	case "disabled":
		status = device.StatusDisabled
	}

	update := r.data.db.Device.Update().
		Where(device.DeviceID(d.DeviceID))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		update = update.Where(device.TenantID(*tenantID))
	}

	update = update.
		SetDeviceType(deviceType).
		SetStatus(status)

	if d.LotID != nil {
		update.SetLotID(*d.LotID)
	} else {
		update.ClearLotID()
	}
	if d.LaneID != nil {
		update.SetLaneID(*d.LaneID)
	} else {
		update.ClearLaneID()
	}

	_, err := update.Save(ctx)
	return err
}

// DeleteDevice deletes a device by device ID.
func (r *vehicleRepo) DeleteDevice(ctx context.Context, deviceID string) error {
	delete := r.data.db.Device.Delete().
		Where(device.DeviceID(deviceID))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		delete = delete.Where(device.TenantID(*tenantID))
	}

	_, err := delete.Exec(ctx)
	return err
}

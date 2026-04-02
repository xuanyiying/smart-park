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
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/devicefault"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/deviceperformance"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent/firmware"
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

	RegisterTenantHooks(db)

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
		ID:                  d.ID,
		DeviceID:            d.DeviceID,
		LotID:               d.LotID,
		LaneID:              d.LaneID,
		DeviceType:          string(d.DeviceType),
		DeviceSecret:        d.DeviceSecret,
		Manufacturer:        d.Manufacturer,
		Model:               d.Model,
		FirmwareVersion:     d.FirmwareVersion,
		VendorSpecificConfig: d.VendorSpecificConfig,
		GateID:              d.GateID,
		Enabled:             d.Enabled,
		Status:              string(d.Status),
		LastHeartbeat:       d.LastHeartbeat,
		LastOnline:          d.LastOnline,
		FaultInfo:           d.FaultInfo,
		HeartbeatCount:      d.HeartbeatCount,
		OfflineCount:        d.OfflineCount,
	}, nil
}

// UpdateDeviceHeartbeat updates device heartbeat.
func (r *vehicleRepo) UpdateDeviceHeartbeat(ctx context.Context, deviceCode string) error {
	now := time.Now()
	return r.data.db.Device.Update().
		Where(device.DeviceID(deviceCode)).
		SetLastHeartbeat(now).
		SetLastOnline(now).
	update := r.data.db.Device.Update().
		Where(device.DeviceID(deviceCode))

	if tenantID := r.getTenantID(ctx); tenantID != nil {
		update = update.Where(device.TenantID(*tenantID))
	}

	return update.
		SetLastHeartbeat(time.Now()).
		SetStatus(device.StatusActive).
		AddHeartbeatCount(1).
		ClearFaultInfo().
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
			ID:                  d.ID,
			DeviceID:            d.DeviceID,
			LotID:               d.LotID,
			LaneID:              d.LaneID,
			DeviceType:          string(d.DeviceType),
			DeviceSecret:        d.DeviceSecret,
			Manufacturer:        d.Manufacturer,
			Model:               d.Model,
			FirmwareVersion:     d.FirmwareVersion,
			VendorSpecificConfig: d.VendorSpecificConfig,
			GateID:              d.GateID,
			Enabled:             d.Enabled,
			Status:              string(d.Status),
			LastHeartbeat:       d.LastHeartbeat,
			LastOnline:          d.LastOnline,
			FaultInfo:           d.FaultInfo,
			HeartbeatCount:      d.HeartbeatCount,
			OfflineCount:        d.OfflineCount,
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
		ID:                  d.ID,
		DeviceID:            d.DeviceID,
		LotID:               d.LotID,
		LaneID:              d.LaneID,
		DeviceType:          string(d.DeviceType),
		DeviceSecret:        d.DeviceSecret,
		Manufacturer:        d.Manufacturer,
		Model:               d.Model,
		FirmwareVersion:     d.FirmwareVersion,
		VendorSpecificConfig: d.VendorSpecificConfig,
		GateID:              d.GateID,
		Enabled:             d.Enabled,
		Status:              string(d.Status),
		LastHeartbeat:       d.LastHeartbeat,
		LastOnline:          d.LastOnline,
		FaultInfo:           d.FaultInfo,
		HeartbeatCount:      d.HeartbeatCount,
		OfflineCount:        d.OfflineCount,
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

	if d.Manufacturer != "" {
		create.SetManufacturer(d.Manufacturer)
	}
	if d.Model != "" {
		create.SetModel(d.Model)
	}
	if d.FirmwareVersion != "" {
		create.SetFirmwareVersion(d.FirmwareVersion)
	}
	if d.VendorSpecificConfig != nil {
		create.SetVendorSpecificConfig(d.VendorSpecificConfig)
	}
	if d.LastOnline != nil {
		create.SetLastOnline(*d.LastOnline)
	}
	if d.FaultInfo != "" {
		create.SetFaultInfo(d.FaultInfo)
	}
	create.SetHeartbeatCount(d.HeartbeatCount)
	create.SetOfflineCount(d.OfflineCount)
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

	if d.Manufacturer != "" {
		update.SetManufacturer(d.Manufacturer)
	} else {
		update.ClearManufacturer()
	}
	if d.Model != "" {
		update.SetModel(d.Model)
	} else {
		update.ClearModel()
	}
	if d.FirmwareVersion != "" {
		update.SetFirmwareVersion(d.FirmwareVersion)
	} else {
		update.ClearFirmwareVersion()
	}
	if d.VendorSpecificConfig != nil {
		update.SetVendorSpecificConfig(d.VendorSpecificConfig)
	} else {
		update.ClearVendorSpecificConfig()
	}
	if d.LastOnline != nil {
		update.SetLastOnline(*d.LastOnline)
	} else {
		update.ClearLastOnline()
	}
	if d.FaultInfo != "" {
		update.SetFaultInfo(d.FaultInfo)
	} else {
		update.ClearFaultInfo()
	}
	update.SetHeartbeatCount(d.HeartbeatCount)
	update.SetOfflineCount(d.OfflineCount)

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

// CreateManufacturer creates a new manufacturer.
func (r *vehicleRepo) CreateManufacturer(ctx context.Context, m *biz.Manufacturer) error {
	_, err := r.data.db.Manufacturer.Create().
		SetName(m.Name).
		SetWebsite(m.Website).
		SetContactInfo(m.ContactInfo).
		SetDescription(m.Description).
		Save(ctx)
	return err
}

// GetManufacturer retrieves a manufacturer by ID.
func (r *vehicleRepo) GetManufacturer(ctx context.Context, id uuid.UUID) (*biz.Manufacturer, error) {
	m, err := r.data.db.Manufacturer.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Manufacturer{
		ID:          m.ID,
		Name:        m.Name,
		Website:     m.Website,
		ContactInfo: m.ContactInfo,
		Description: m.Description,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}, nil
}

// UpdateManufacturer updates an existing manufacturer.
func (r *vehicleRepo) UpdateManufacturer(ctx context.Context, m *biz.Manufacturer) error {
	_, err := r.data.db.Manufacturer.UpdateOneID(m.ID).
		SetName(m.Name).
		SetWebsite(m.Website).
		SetContactInfo(m.ContactInfo).
		SetDescription(m.Description).
		Save(ctx)
	return err
}

// DeleteManufacturer deletes a manufacturer by ID.
func (r *vehicleRepo) DeleteManufacturer(ctx context.Context, id uuid.UUID) error {
	_, err := r.data.db.Manufacturer.DeleteOneID(id).Exec(ctx)
	return err
}

// ListManufacturers retrieves all manufacturers with pagination.
func (r *vehicleRepo) ListManufacturers(ctx context.Context, page, pageSize int) ([]*biz.Manufacturer, int, error) {
	query := r.data.db.Manufacturer.Query()

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	manufacturers, err := query.
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to biz entities
	result := make([]*biz.Manufacturer, len(manufacturers))
	for i, m := range manufacturers {
		result[i] = &biz.Manufacturer{
			ID:          m.ID,
			Name:        m.Name,
			Website:     m.Website,
			ContactInfo: m.ContactInfo,
			Description: m.Description,
			CreatedAt:   m.CreatedAt,
			UpdatedAt:   m.UpdatedAt,
		}
	}

	return result, total, nil
}

// CreateFirmware creates a new firmware.
func (r *vehicleRepo) CreateFirmware(ctx context.Context, f *biz.Firmware) error {
	_, err := r.data.db.Firmware.Create().
		SetFirmwareID(f.FirmwareID).
		SetManufacturer(f.Manufacturer).
		SetModel(f.Model).
		SetVersion(f.Version).
		SetURL(f.URL).
		SetSize(f.Size).
		SetMD5(f.MD5).
		SetDescription(f.Description).
		SetStatus(f.Status).
		SetReleaseDate(f.ReleaseDate).
		Save(ctx)
	return err
}

// GetFirmware retrieves a firmware by ID.
func (r *vehicleRepo) GetFirmware(ctx context.Context, id uuid.UUID) (*biz.Firmware, error) {
	f, err := r.data.db.Firmware.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Firmware{
		ID:          f.ID,
		FirmwareID:  f.FirmwareID,
		Manufacturer: f.Manufacturer,
		Model:       f.Model,
		Version:     f.Version,
		URL:         f.URL,
		Size:        f.Size,
		MD5:         f.MD5,
		Description: f.Description,
		Status:      f.Status,
		ReleaseDate: f.ReleaseDate,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}, nil
}

// GetFirmwareByID retrieves a firmware by firmware ID.
func (r *vehicleRepo) GetFirmwareByID(ctx context.Context, firmwareID string) (*biz.Firmware, error) {
	f, err := r.data.db.Firmware.Query().
		Where(firmware.FirmwareID(firmwareID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Firmware{
		ID:          f.ID,
		FirmwareID:  f.FirmwareID,
		Manufacturer: f.Manufacturer,
		Model:       f.Model,
		Version:     f.Version,
		URL:         f.URL,
		Size:        f.Size,
		MD5:         f.MD5,
		Description: f.Description,
		Status:      f.Status,
		ReleaseDate: f.ReleaseDate,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}, nil
}

// UpdateFirmware updates an existing firmware.
func (r *vehicleRepo) UpdateFirmware(ctx context.Context, f *biz.Firmware) error {
	_, err := r.data.db.Firmware.UpdateOneID(f.ID).
		SetManufacturer(f.Manufacturer).
		SetModel(f.Model).
		SetVersion(f.Version).
		SetURL(f.URL).
		SetSize(f.Size).
		SetMD5(f.MD5).
		SetDescription(f.Description).
		SetStatus(f.Status).
		SetReleaseDate(f.ReleaseDate).
		Save(ctx)
	return err
}

// DeleteFirmware deletes a firmware by ID.
func (r *vehicleRepo) DeleteFirmware(ctx context.Context, id uuid.UUID) error {
	_, err := r.data.db.Firmware.DeleteOneID(id).Exec(ctx)
	return err
}

// ListFirmwares retrieves firmwares with pagination.
func (r *vehicleRepo) ListFirmwares(ctx context.Context, manufacturer, model string, page, pageSize int) ([]*biz.Firmware, int, error) {
	query := r.data.db.Firmware.Query()

	if manufacturer != "" {
		query = query.Where(firmware.Manufacturer(manufacturer))
	}
	if model != "" {
		query = query.Where(firmware.Model(model))
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	firmwares, err := query.
		Order(ent.Desc(firmware.FieldReleaseDate)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to biz entities
	result := make([]*biz.Firmware, len(firmwares))
	for i, f := range firmwares {
		result[i] = &biz.Firmware{
			ID:          f.ID,
			FirmwareID:  f.FirmwareID,
			Manufacturer: f.Manufacturer,
			Model:       f.Model,
			Version:     f.Version,
			URL:         f.URL,
			Size:        f.Size,
			MD5:         f.MD5,
			Description: f.Description,
			Status:      f.Status,
			ReleaseDate: f.ReleaseDate,
			CreatedAt:   f.CreatedAt,
			UpdatedAt:   f.UpdatedAt,
		}
	}

	return result, total, nil
}

// GetLatestFirmware retrieves the latest firmware for a specific manufacturer and model.
func (r *vehicleRepo) GetLatestFirmware(ctx context.Context, manufacturer, model string) (*biz.Firmware, error) {
	f, err := r.data.db.Firmware.Query().
		Where(firmware.Manufacturer(manufacturer), firmware.Model(model), firmware.Status("published")).
		Order(ent.Desc(firmware.FieldReleaseDate)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Firmware{
		ID:          f.ID,
		FirmwareID:  f.FirmwareID,
		Manufacturer: f.Manufacturer,
		Model:       f.Model,
		Version:     f.Version,
		URL:         f.URL,
		Size:        f.Size,
		MD5:         f.MD5,
		Description: f.Description,
		Status:      f.Status,
		ReleaseDate: f.ReleaseDate,
		CreatedAt:   f.CreatedAt,
		UpdatedAt:   f.UpdatedAt,
	}, nil
}

// CreateDevicePerformance creates a new device performance record.
func (r *vehicleRepo) CreateDevicePerformance(ctx context.Context, performance *biz.DevicePerformance) error {
	_, err := r.data.db.DevicePerformance.Create().
		SetDeviceID(performance.DeviceID).
		SetCPUUsage(performance.CPUUsage).
		SetMemoryUsage(performance.MemoryUsage).
		SetStorageUsage(performance.StorageUsage).
		SetNetworkIn(performance.NetworkIn).
		SetNetworkOut(performance.NetworkOut).
		SetTemperature(performance.Temperature).
		SetTimestamp(performance.Timestamp).
		Save(ctx)
	return err
}

// GetDevicePerformance retrieves device performance records within a time range.
func (r *vehicleRepo) GetDevicePerformance(ctx context.Context, deviceID string, startTime, endTime time.Time) ([]*biz.DevicePerformance, error) {
	records, err := r.data.db.DevicePerformance.Query().
		Where(
			deviceperformance.DeviceID(deviceID),
			deviceperformance.TimestampGTE(startTime),
			deviceperformance.TimestampLTE(endTime),
		).
		Order(ent.Asc(deviceperformance.FieldTimestamp)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*biz.DevicePerformance, len(records))
	for i, record := range records {
		result[i] = &biz.DevicePerformance{
			ID:           record.ID,
			DeviceID:     record.DeviceID,
			CPUUsage:     record.CPUUsage,
			MemoryUsage:  record.MemoryUsage,
			StorageUsage: record.StorageUsage,
			NetworkIn:    record.NetworkIn,
			NetworkOut:   record.NetworkOut,
			Temperature:  record.Temperature,
			Timestamp:    record.Timestamp,
			CreatedAt:    record.CreatedAt,
		}
	}

	return result, nil
}

// GetDevicePerformanceLatest retrieves the latest device performance record.
func (r *vehicleRepo) GetDevicePerformanceLatest(ctx context.Context, deviceID string) (*biz.DevicePerformance, error) {
	record, err := r.data.db.DevicePerformance.Query().
		Where(deviceperformance.DeviceID(deviceID)).
		Order(ent.Desc(deviceperformance.FieldTimestamp)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.DevicePerformance{
		ID:           record.ID,
		DeviceID:     record.DeviceID,
		CPUUsage:     record.CPUUsage,
		MemoryUsage:  record.MemoryUsage,
		StorageUsage: record.StorageUsage,
		NetworkIn:    record.NetworkIn,
		NetworkOut:   record.NetworkOut,
		Temperature:  record.Temperature,
		Timestamp:    record.Timestamp,
		CreatedAt:    record.CreatedAt,
	}, nil
}

// CreateDeviceFault creates a new device fault record.
func (r *vehicleRepo) CreateDeviceFault(ctx context.Context, fault *biz.DeviceFault) error {
	_, err := r.data.db.DeviceFault.Create().
		SetDeviceID(fault.DeviceID).
		SetFaultType(fault.FaultType).
		SetFaultCode(fault.FaultCode).
		SetDescription(fault.Description).
		SetSeverity(fault.Severity).
		SetStatus(fault.Status).
		SetSuggestion(fault.Suggestion).
		SetDetectedAt(fault.DetectedAt).
		Save(ctx)
	return err
}

// GetDeviceFault retrieves a device fault by ID.
func (r *vehicleRepo) GetDeviceFault(ctx context.Context, id uuid.UUID) (*biz.DeviceFault, error) {
	fault, err := r.data.db.DeviceFault.Get(ctx, id)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.DeviceFault{
		ID:          fault.ID,
		DeviceID:    fault.DeviceID,
		FaultType:   fault.FaultType,
		FaultCode:   fault.FaultCode,
		Description: fault.Description,
		Severity:    fault.Severity,
		Status:      fault.Status,
		Suggestion:  fault.Suggestion,
		DetectedAt:  fault.DetectedAt,
		ResolvedAt:  fault.ResolvedAt,
		CreatedAt:   fault.CreatedAt,
		UpdatedAt:   fault.UpdatedAt,
	}, nil
}

// UpdateDeviceFault updates an existing device fault.
func (r *vehicleRepo) UpdateDeviceFault(ctx context.Context, fault *biz.DeviceFault) error {
	update := r.data.db.DeviceFault.UpdateOneID(fault.ID).
		SetFaultType(fault.FaultType).
		SetFaultCode(fault.FaultCode).
		SetDescription(fault.Description).
		SetSeverity(fault.Severity).
		SetStatus(fault.Status).
		SetSuggestion(fault.Suggestion)

	if fault.ResolvedAt != nil {
		update.SetResolvedAt(*fault.ResolvedAt)
	}

	_, err := update.Save(ctx)
	return err
}

// ListDeviceFaults retrieves device faults with pagination.
func (r *vehicleRepo) ListDeviceFaults(ctx context.Context, deviceID string, status string, page, pageSize int) ([]*biz.DeviceFault, int, error) {
	query := r.data.db.DeviceFault.Query()

	if deviceID != "" {
		query = query.Where(devicefault.DeviceID(deviceID))
	}
	if status != "" {
		query = query.Where(devicefault.Status(status))
	}

	// Get total count
	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Apply pagination
	offset := (page - 1) * pageSize
	faults, err := query.
		Order(ent.Desc(devicefault.FieldDetectedAt)).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	// Convert to biz entities
	result := make([]*biz.DeviceFault, len(faults))
	for i, fault := range faults {
		result[i] = &biz.DeviceFault{
			ID:          fault.ID,
			DeviceID:    fault.DeviceID,
			FaultType:   fault.FaultType,
			FaultCode:   fault.FaultCode,
			Description: fault.Description,
			Severity:    fault.Severity,
			Status:      fault.Status,
			Suggestion:  fault.Suggestion,
			DetectedAt:  fault.DetectedAt,
			ResolvedAt:  fault.ResolvedAt,
			CreatedAt:   fault.CreatedAt,
			UpdatedAt:   fault.UpdatedAt,
		}
	}

	return result, total, nil
}

// ResolveDeviceFault resolves a device fault.
func (r *vehicleRepo) ResolveDeviceFault(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	_, err := r.data.db.DeviceFault.UpdateOneID(id).
		SetStatus(devicefault.StatusResolved).
		SetResolvedAt(now).
		Save(ctx)
	return err
}

// GetDeviceUsageStats retrieves device usage statistics.
func (r *vehicleRepo) GetDeviceUsageStats(ctx context.Context, deviceID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	// 统计设备的使用情况，包括开闸次数、识别次数等
	// 这里需要根据实际的业务逻辑来实现
	// 暂时返回一个示例结构
	return map[string]interface{}{
		"device_id":    deviceID,
		"start_time":   startTime,
		"end_time":     endTime,
		"open_gate_count": 0,
		"recognition_count": 0,
		"average_response_time": 0.0,
	}, nil
}

// GetDeviceFaultStats retrieves device fault statistics.
func (r *vehicleRepo) GetDeviceFaultStats(ctx context.Context, deviceID string, startTime, endTime time.Time) (map[string]interface{}, error) {
	// 统计设备的故障情况
	faults, err := r.data.db.DeviceFault.Query().
		Where(
			devicefault.DeviceID(deviceID),
			devicefault.DetectedAtGTE(startTime),
			devicefault.DetectedAtLTE(endTime),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	// 按严重程度统计故障
	severityCount := make(map[string]int)
	statusCount := make(map[string]int)

	for _, fault := range faults {
		severityCount[string(fault.Severity)]++
		statusCount[string(fault.Status)]++
	}

	return map[string]interface{}{
		"device_id":    deviceID,
		"start_time":   startTime,
		"end_time":     endTime,
		"total_faults": len(faults),
		"severity_count": severityCount,
		"status_count":   statusCount,
	}, nil
}

// GetDeviceStatsSummary retrieves device statistics summary.
func (r *vehicleRepo) GetDeviceStatsSummary(ctx context.Context, deviceID string) (map[string]interface{}, error) {
	// 获取设备的统计摘要
	// 包括在线时间、使用频率、故障率等
	device, err := r.data.db.Device.Query().
		Where(device.DeviceID(deviceID)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	// 计算在线时间（简单示例）
	onlineTime := time.Since(*device.LastOnline).Hours()

	// 统计故障数量
	faultCount, err := r.data.db.DeviceFault.Query().
		Where(devicefault.DeviceID(deviceID)).
		Count(ctx)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"device_id":      deviceID,
		"manufacturer":   device.Manufacturer,
		"model":          device.Model,
		"firmware_version": device.FirmwareVersion,
		"online_time_hours": onlineTime,
		"heartbeat_count": device.HeartbeatCount,
		"offline_count":   device.OfflineCount,
		"fault_count":     faultCount,
		"status":          device.Status,
	}, nil
}

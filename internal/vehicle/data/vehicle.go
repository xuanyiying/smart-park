// Package data provides data access layer for the vehicle service.
package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/ent"
	"github.com/xuanyiying/smart-park/ent/device"
	"github.com/xuanyiying/smart-park/ent/lane"
	"github.com/xuanyiying/smart-park/ent/parkingrecord"
	"github.com/xuanyiying/smart-park/ent/vehicle"
	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
)

// Data holds the data layer dependencies.
type Data struct {
	db  *ent.Client
	log *log.Helper
}

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
	v, err := r.data.db.Vehicle.Query().
		Where(vehicle.PlateNumber(plateNumber)).
		Only(ctx)
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
	record, err := r.data.db.ParkingRecord.Query().
		Where(
			parkingrecord.PlateNumber(plateNumber),
			parkingrecord.RecordStatusIn(parkingrecord.RecordStatusEntry, parkingrecord.RecordStatusExiting),
		).
		Only(ctx)
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
	create := r.data.db.ParkingRecord.Create().
		SetID(rec.ID).
		SetLotID(rec.LotID).
		SetEntryLaneID(rec.EntryLaneID).
		SetEntryTime(rec.EntryTime).
		SetEntryImageURL(rec.EntryImageURL).
		SetRecordStatus(parkingrecord.RecordStatusEntry).
		SetExitStatus(parkingrecord.ExitStatusUnpaid)

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
	update := r.data.db.ParkingRecord.UpdateOneID(rec.ID)

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
	record, err := r.data.db.ParkingRecord.Get(ctx, recordID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return toBizParkingRecord(record), nil
}

// GetDeviceByCode retrieves a device by device code.
func (r *vehicleRepo) GetDeviceByCode(ctx context.Context, deviceCode string) (*biz.Device, error) {
	d, err := r.data.db.Device.Query().
		Where(device.DeviceID(deviceCode)).
		Only(ctx)
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
	return r.data.db.Device.Update().
		Where(device.DeviceID(deviceCode)).
		SetLastHeartbeat(time.Now()).
		SetStatus(device.StatusActive).
		Exec(ctx)
}

// GetLaneByDeviceCode retrieves a lane by device code.
func (r *vehicleRepo) GetLaneByDeviceCode(ctx context.Context, deviceCode string) (*biz.Lane, error) {
	// First get the device
	d, err := r.data.db.Device.Query().
		Where(device.DeviceID(deviceCode)).
		Only(ctx)
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

// billingRuleRepo implements biz.BillingRepo.
type billingRuleRepo struct {
	data *Data
}

// NewBillingRuleRepo creates a new BillingRuleRepo.
func NewBillingRuleRepo(data *Data) biz.BillingRepo {
	return &billingRuleRepo{data: data}
}

// GetRulesByLotID retrieves billing rules by lot ID.
func (r *billingRuleRepo) GetRulesByLotID(ctx context.Context, lotID uuid.UUID) ([]*biz.BillingRule, error) {
	rules, err := r.data.db.BillingRule.Query().
		Where(
			func(s *sql.Selector) {
				s.Where(sql.EQ("lot_id", lotID))
			},
		).
		All(ctx)
	if err != nil {
		return nil, err
	}

	var result []*biz.BillingRule
	for _, rule := range rules {
		result = append(result, &biz.BillingRule{
			ID:         rule.ID,
			LotID:      rule.LotID,
			RuleName:   rule.RuleName,
			RuleType:   string(rule.RuleType),
			Conditions: rule.ConditionsJSON,
			Actions:    rule.ActionsJSON,
			RuleConfig: rule.RuleConfig,
			Priority:   rule.Priority,
			IsActive:   rule.IsActive,
		})
	}

	return result, nil
}

// Helper function to convert ent ParkingRecord to biz ParkingRecord.
func toBizParkingRecord(record *ent.ParkingRecord) *biz.ParkingRecord {
	return &biz.ParkingRecord{
		ID:              record.ID,
		LotID:           record.LotID,
		EntryLaneID:     record.EntryLaneID,
		VehicleID:       record.VehicleID,
		PlateNumber:     record.PlateNumber,
		PlateNumberSource: string(record.PlateNumberSource),
		EntryTime:       record.EntryTime,
		EntryImageURL:   record.EntryImageURL,
		RecordStatus:    string(record.RecordStatus),
		ExitTime:        record.ExitTime,
		ExitImageURL:    record.ExitImageURL,
		ExitLaneID:      record.ExitLaneID,
		ExitDeviceID:    record.ExitDeviceID,
		ParkingDuration: record.ParkingDuration,
		ExitStatus:      string(record.ExitStatus),
		PaymentLock:     record.PaymentLock,
		Metadata:        record.RecordMetadata,
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
}

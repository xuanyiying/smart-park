// Package ent provides the Ent client and database utilities.
package ent

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/ent/vehicle"
	"github.com/xuanyiying/smart-park/ent/parkingrecord"
	"github.com/xuanyiying/smart-park/ent/billingrule"
	"github.com/xuanyiying/smart-park/ent/order"
	"github.com/xuanyiying/smart-park/ent/device"
	"github.com/xuanyiying/smart-park/ent/lane"
	"github.com/xuanyiying/smart-park/ent/parkinglot"
)

// Open opens a new database connection.
func Open(driverName, dataSourceName string, logger log.Logger) (*Client, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	drv := entsql.OpenDB(dialect.Postgres, db)
	client := NewClient(Driver(drv))

	return client, nil
}

// NewClient creates a new Ent client with custom options.
func NewClient(opts ...Option) *Client {
	cfg := config{logger: log.DefaultLogger}
	for _, o := range opts {
		o(&cfg)
	}
	return newClient(opts...)
}

// Client is the Ent client with custom methods.
type Client struct {
	*client
	logger log.Logger
}

// WithTx executes a function within a transaction.
func (c *Client) WithTx(ctx context.Context, fn func(tx *Tx) error) error {
	tx, err := c.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()
	if err := fn(tx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			c.logger.Log(log.LevelError, "msg", "failed to rollback transaction", "error", rerr)
		}
		return err
	}
	return tx.Commit()
}

// VehicleClient extends the generated VehicleClient with custom queries.
type VehicleClient struct {
	*vehicleClient
}

// GetByPlate retrieves a vehicle by plate number.
func (c *VehicleClient) GetByPlate(ctx context.Context, plateNumber string) (*Vehicle, error) {
	return c.Query().
		Where(vehicle.PlateNumber(plateNumber)).
		Only(ctx)
}

// GetByPlateWithLock retrieves a vehicle by plate number with row lock.
func (c *VehicleClient) GetByPlateWithLock(ctx context.Context, plateNumber string) (*Vehicle, error) {
	tx, err := c.Tx(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	v, err := tx.Vehicle.Query().
		Where(vehicle.PlateNumber(plateNumber)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return v, tx.Commit()
}

// FindMonthlyVehicles retrieves all monthly vehicles.
func (c *VehicleClient) FindMonthlyVehicles(ctx context.Context) ([]*Vehicle, error) {
	return c.Query().
		Where(
			vehicle.VehicleType(vehicle.VehicleTypeMonthly),
			vehicle.MonthlyValidUntilNotNil(),
		).
		All(ctx)
}

// FindActiveMonthlyVehicles retrieves monthly vehicles with valid subscriptions.
func (c *VehicleClient) FindActiveMonthlyVehicles(ctx context.Context) ([]*Vehicle, error) {
	now := time.Now()
	return c.Query().
		Where(
			vehicle.VehicleType(vehicle.VehicleTypeMonthly),
			vehicle.MonthlyValidUntilNotNil(),
			vehicle.MonthlyValidUntilGT(now),
		).
		All(ctx)
}

// ParkingRecordClient extends the generated ParkingRecordClient with custom queries.
type ParkingRecordClient struct {
	*parkingrecordClient
}

// GetEntryRecord retrieves an active entry record by plate number.
func (c *ParkingRecordClient) GetEntryRecord(ctx context.Context, plateNumber string) (*ParkingRecord, error) {
	return c.Query().
		Where(
			parkingrecord.PlateNumber(plateNumber),
			parkingrecord.RecordStatusIn(parkingrecord.RecordStatusEntry, parkingrecord.RecordStatusExiting),
		).
		Only(ctx)
}

// FindActiveByPlate retrieves all active parking records for a plate.
func (c *ParkingRecordClient) FindActiveByPlate(ctx context.Context, plateNumber string) ([]*ParkingRecord, error) {
	return c.Query().
		Where(
			parkingrecord.PlateNumber(plateNumber),
			parkingrecord.RecordStatusNEQ(parkingrecord.RecordStatusExited),
		).
		All(ctx)
}

// FindByLotAndStatus retrieves parking records by lot ID and status.
func (c *ParkingRecordClient) FindByLotAndStatus(ctx context.Context, lotID uuid.UUID, status parkingrecord.RecordStatus) ([]*ParkingRecord, error) {
	return c.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.RecordStatus(status),
		).
		All(ctx)
}

// CountByLot counts parking records in a lot.
func (c *ParkingRecordClient) CountByLot(ctx context.Context, lotID uuid.UUID) (int, error) {
	return c.Query().
		Where(parkingrecord.LotID(lotID)).
		Count(ctx)
}

// CountActiveByLot counts active parking records in a lot.
func (c *ParkingRecordClient) CountActiveByLot(ctx context.Context, lotID uuid.UUID) (int, error) {
	return c.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.RecordStatusNEQ(parkingrecord.RecordStatusExited),
		).
		Count(ctx)
}

// BillingRuleClient extends the generated BillingRuleClient with custom queries.
type BillingRuleClient struct {
	*billingruleClient
}

// GetByLotID retrieves all billing rules for a parking lot.
func (c *BillingRuleClient) GetByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error) {
	return c.Query().
		Where(billingrule.LotID(lotID)).
		Order(billingrule.Desc(billingrule.FieldPriority)).
		All(ctx)
}

// GetActiveByLotID retrieves active billing rules for a parking lot.
func (c *BillingRuleClient) GetActiveByLotID(ctx context.Context, lotID uuid.UUID) ([]*BillingRule, error) {
	return c.Query().
		Where(
			billingrule.LotID(lotID),
			billingrule.IsActive(true),
		).
		Order(billingrule.Desc(billingrule.FieldPriority)).
		All(ctx)
}

// GetByType retrieves billing rules by type.
func (c *BillingRuleClient) GetByType(ctx context.Context, lotID uuid.UUID, ruleType billingrule.RuleType) ([]*BillingRule, error) {
	return c.Query().
		Where(
			billingrule.LotID(lotID),
			billingrule.RuleType(ruleType),
			billingrule.IsActive(true),
		).
		All(ctx)
}

// OrderClient extends the generated OrderClient with custom queries.
type OrderClient struct {
	*orderClient
}

// GetByRecordID retrieves an order by parking record ID.
func (c *OrderClient) GetByRecordID(ctx context.Context, recordID uuid.UUID) (*Order, error) {
	return c.Query().
		Where(order.RecordID(recordID)).
		Only(ctx)
}

// GetByTransactionID retrieves an order by transaction ID.
func (c *OrderClient) GetByTransactionID(ctx context.Context, transactionID string) (*Order, error) {
	return c.Query().
		Where(order.TransactionID(transactionID)).
		Only(ctx)
}

// FindByStatus retrieves orders by status.
func (c *OrderClient) FindByStatus(ctx context.Context, status order.Status) ([]*Order, error) {
	return c.Query().
		Where(order.Status(status)).
		All(ctx)
}

// FindPendingOrders retrieves pending orders.
func (c *OrderClient) FindPendingOrders(ctx context.Context, lotID uuid.UUID, limit int) ([]*Order, error) {
	query := c.Query().
		Where(
			order.LotID(lotID),
			order.Status(order.StatusPending),
		).
		Order(order.Desc(order.FieldCreatedAt))

	if limit > 0 {
		query = query.Limit(limit)
	}

	return query.All(ctx)
}

// DeviceClient extends the generated DeviceClient with custom queries.
type DeviceClient struct {
	*deviceClient
}

// GetByDeviceID retrieves a device by device ID.
func (c *DeviceClient) GetByDeviceID(ctx context.Context, deviceID string) (*Device, error) {
	return c.Query().
		Where(device.DeviceID(deviceID)).
		Only(ctx)
}

// GetByCode retrieves a device by device code (alias for GetByDeviceID).
func (c *DeviceClient) GetByCode(ctx context.Context, deviceCode string) (*Device, error) {
	return c.GetByDeviceID(ctx, deviceCode)
}

// UpdateHeartbeat updates the device heartbeat timestamp.
func (c *DeviceClient) UpdateHeartbeat(ctx context.Context, deviceID string) error {
	return c.Update().
		Where(device.DeviceID(deviceID)).
		SetLastHeartbeat(time.Now()).
		SetStatus(device.StatusActive).
		Exec(ctx)
}

// FindOfflineDevices retrieves devices that haven't sent heartbeat recently.
func (c *DeviceClient) FindOfflineDevices(ctx context.Context, threshold time.Duration) ([]*Device, error) {
	thresholdTime := time.Now().Add(-threshold)
	return c.Query().
		Where(
			device.LastHeartbeatNotNil(),
			device.LastHeartbeatLT(thresholdTime),
		).
		All(ctx)
}

// LaneClient extends the generated LaneClient with custom queries.
type LaneClient struct {
	*laneClient
}

// GetByDeviceCode retrieves a lane by associated device code.
func (c *LaneClient) GetByDeviceCode(ctx context.Context, deviceCode string) (*Lane, error) {
	// First get the device to find its lane
	d, err := c.client.Device.Query().
		Where(device.DeviceID(deviceCode)).
		Only(ctx)
	if err != nil {
		return nil, err
	}

	if d.LaneID == nil {
		return nil, fmt.Errorf("device %s is not associated with any lane", deviceCode)
	}

	return c.Get(ctx, *d.LaneID)
}

// FindByLot retrieves all lanes for a parking lot.
func (c *LaneClient) FindByLot(ctx context.Context, lotID uuid.UUID) ([]*Lane, error) {
	return c.Query().
		Where(lane.LotID(lotID)).
		All(ctx)
}

// FindActiveByLot retrieves active lanes for a parking lot.
func (c *LaneClient) FindActiveByLot(ctx context.Context, lotID uuid.UUID) ([]*Lane, error) {
	return c.Query().
		Where(
			lane.LotID(lotID),
			lane.Status(lane.StatusActive),
		).
		All(ctx)
}

// ParkingLotClient extends the generated ParkingLotClient with custom queries.
type ParkingLotClient struct {
	*parkinglotClient
}

// List retrieves all parking lots.
func (c *ParkingLotClient) List(ctx context.Context) ([]*ParkingLot, error) {
	return c.Query().
		Order(parkinglot.Desc(parkinglot.FieldCreatedAt)).
		All(ctx)
}

// FindActive retrieves all active parking lots.
func (c *ParkingLotClient) FindActive(ctx context.Context) ([]*ParkingLot, error) {
	return c.Query().
		Where(parkinglot.Status(parkinglot.StatusActive)).
		All(ctx)
}

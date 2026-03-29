// Package data provides data access layer for the admin service.
package data

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/admin/biz"
	ent "github.com/xuanyiying/smart-park/internal/admin/data/ent"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/order"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/parkinglot"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/parkingrecord"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/user"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/vehicle"
)

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

// adminRepo implements biz.AdminRepo.
type adminRepo struct {
	data *Data
}

// NewAdminRepo creates a new AdminRepo.
func NewAdminRepo(data *Data) biz.AdminRepo {
	return &adminRepo{data: data}
}

// CreateParkingLot creates a new parking lot.
func (r *adminRepo) CreateParkingLot(ctx context.Context, lot *biz.ParkingLot) error {
	_, err := r.data.db.ParkingLot.Create().
		SetID(lot.ID).
		SetName(lot.Name).
		SetAddress(lot.Address).
		SetLanes(lot.Lanes).
		SetStatus(parkinglot.StatusActive).
		Save(ctx)
	return err
}

// GetParkingLot retrieves a parking lot by ID.
func (r *adminRepo) GetParkingLot(ctx context.Context, lotID uuid.UUID) (*biz.ParkingLot, error) {
	lot, err := r.data.db.ParkingLot.Get(ctx, lotID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.ParkingLot{
		ID:        lot.ID,
		Name:      lot.Name,
		Address:   lot.Address,
		Lanes:     lot.Lanes,
		Status:    string(lot.Status),
		CreatedAt: lot.CreatedAt,
		UpdatedAt: lot.UpdatedAt,
	}, nil
}

// UpdateParkingLot updates a parking lot.
func (r *adminRepo) UpdateParkingLot(ctx context.Context, lot *biz.ParkingLot) error {
	update := r.data.db.ParkingLot.UpdateOneID(lot.ID).
		SetName(lot.Name).
		SetAddress(lot.Address).
		SetLanes(lot.Lanes)

	switch lot.Status {
	case "active":
		update.SetStatus(parkinglot.StatusActive)
	case "inactive":
		update.SetStatus(parkinglot.StatusInactive)
	case "maintenance":
		update.SetStatus(parkinglot.StatusMaintenance)
	}

	_, err := update.Save(ctx)
	return err
}

// ListParkingLots lists parking lots with pagination.
func (r *adminRepo) ListParkingLots(ctx context.Context, page, pageSize int) ([]*biz.ParkingLot, int64, error) {
	query := r.data.db.ParkingLot.Query()

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	lots, err := query.
		Order(ent.Desc("created_at")).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*biz.ParkingLot
	for _, lot := range lots {
		result = append(result, &biz.ParkingLot{
			ID:        lot.ID,
			Name:      lot.Name,
			Address:   lot.Address,
			Lanes:     lot.Lanes,
			Status:    string(lot.Status),
			CreatedAt: lot.CreatedAt,
			UpdatedAt: lot.UpdatedAt,
		})
	}

	return result, int64(total), nil
}

// CreateVehicle creates a new vehicle.
func (r *adminRepo) CreateVehicle(ctx context.Context, v *biz.Vehicle) error {
	vehicleType := vehicle.VehicleTypeTemporary
	switch v.VehicleType {
	case "monthly":
		vehicleType = vehicle.VehicleTypeMonthly
	case "vip":
		vehicleType = vehicle.VehicleTypeVip
	}

	create := r.data.db.Vehicle.Create().
		SetID(v.ID).
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

// ListVehicles lists vehicles with pagination.
func (r *adminRepo) ListVehicles(ctx context.Context, vehicleType string, page, pageSize int) ([]*biz.Vehicle, int64, error) {
	query := r.data.db.Vehicle.Query()

	if vehicleType != "" {
		switch vehicleType {
		case "temporary":
			query = query.Where(vehicle.VehicleTypeEQ(vehicle.VehicleTypeTemporary))
		case "monthly":
			query = query.Where(vehicle.VehicleTypeEQ(vehicle.VehicleTypeMonthly))
		case "vip":
			query = query.Where(vehicle.VehicleTypeEQ(vehicle.VehicleTypeVip))
		}
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	vehicles, err := query.
		Order(ent.Desc("created_at")).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*biz.Vehicle
	for _, v := range vehicles {
		result = append(result, &biz.Vehicle{
			ID:                v.ID,
			PlateNumber:       v.PlateNumber,
			VehicleType:       string(v.VehicleType),
			OwnerName:         v.OwnerName,
			OwnerPhone:        v.OwnerPhone,
			MonthlyValidUntil: v.MonthlyValidUntil,
			CreatedAt:         v.CreatedAt,
		})
	}

	return result, int64(total), nil
}

// ListParkingRecords lists parking records with pagination.
func (r *adminRepo) ListParkingRecords(ctx context.Context, lotID uuid.UUID, plateNumber, startTime, endTime string, page, pageSize int) ([]*biz.ParkingRecord, int64, error) {
	query := r.data.db.ParkingRecord.Query()

	if lotID != uuid.Nil {
		query = query.Where(parkingrecord.LotID(lotID))
	}
	if plateNumber != "" {
		query = query.Where(parkingrecord.PlateNumberContains(plateNumber))
	}
	if startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			query = query.Where(parkingrecord.EntryTimeGTE(t))
		}
	}
	if endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			query = query.Where(parkingrecord.EntryTimeLTE(t))
		}
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	records, err := query.
		Order(ent.Desc("entry_time")).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*biz.ParkingRecord
	for _, rec := range records {
		result = append(result, &biz.ParkingRecord{
			ID:              rec.ID,
			LotID:           rec.LotID,
			PlateNumber:     *rec.PlateNumber,
			EntryTime:       rec.EntryTime,
			ExitTime:        rec.ExitTime,
			ParkingDuration: rec.ParkingDuration,
			Status:          string(rec.RecordStatus),
		})
	}

	return result, int64(total), nil
}

// ListOrders lists orders with pagination.
func (r *adminRepo) ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*biz.Order, int64, error) {
	query := r.data.db.Order.Query()

	if lotID != uuid.Nil {
		query = query.Where(order.LotID(lotID))
	}
	if status != "" {
		switch status {
		case "pending":
			query = query.Where(order.StatusEQ(order.StatusPending))
		case "paid":
			query = query.Where(order.StatusEQ(order.StatusPaid))
		case "refunding":
			query = query.Where(order.StatusEQ(order.StatusRefunding))
		case "refunded":
			query = query.Where(order.StatusEQ(order.StatusRefunded))
		case "failed":
			query = query.Where(order.StatusEQ(order.StatusFailed))
		}
	}

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	orders, err := query.
		Order(ent.Desc("created_at")).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*biz.Order
	for _, o := range orders {
		result = append(result, &biz.Order{
			ID:             o.ID,
			RecordID:       o.RecordID,
			LotID:          o.LotID,
			PlateNumber:    o.PlateNumber,
			Amount:         o.Amount,
			DiscountAmount: o.DiscountAmount,
			FinalAmount:    o.FinalAmount,
			Status:         string(o.Status),
			PayTime:        o.PayTime,
			PayMethod:      string(o.PayMethod),
		})
	}

	return result, int64(total), nil
}

// GetOrder retrieves an order by ID.
func (r *adminRepo) GetOrder(ctx context.Context, orderID uuid.UUID) (*biz.Order, error) {
	o, err := r.data.db.Order.Get(ctx, orderID)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}

	return &biz.Order{
		ID:             o.ID,
		RecordID:       o.RecordID,
		LotID:          o.LotID,
		PlateNumber:    o.PlateNumber,
		Amount:         o.Amount,
		DiscountAmount: o.DiscountAmount,
		FinalAmount:    o.FinalAmount,
		Status:         string(o.Status),
		PayTime:        o.PayTime,
		PayMethod:      string(o.PayMethod),
	}, nil
}

// GetDailyReport retrieves a daily report.
func (r *adminRepo) GetDailyReport(ctx context.Context, lotID uuid.UUID, date string) (*biz.DailyReport, error) {
	// Parse date
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, err
	}

	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	// Count entries
	entries, _ := r.data.db.ParkingRecord.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.EntryTimeGTE(startOfDay),
			parkingrecord.EntryTimeLT(endOfDay),
		).
		Count(ctx)

	// Count exits
	exits, _ := r.data.db.ParkingRecord.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.ExitTimeNotNil(),
			parkingrecord.ExitTimeGTE(startOfDay),
			parkingrecord.ExitTimeLT(endOfDay),
		).
		Count(ctx)

	// Calculate total amount
	orders, _ := r.data.db.Order.Query().
		Where(
			order.LotID(lotID),
			order.StatusEQ(order.StatusPaid),
			order.PayTimeNotNil(),
			order.PayTimeGTE(startOfDay),
			order.PayTimeLT(endOfDay),
		).
		All(ctx)

	var totalAmount, totalDiscount float64
	for _, o := range orders {
		totalAmount += o.FinalAmount
		totalDiscount += o.DiscountAmount
	}

	return &biz.DailyReport{
		LotID:         lotID.String(),
		Date:          date,
		TotalEntries:  entries,
		TotalExits:    exits,
		TotalVehicles: entries,
		TotalAmount:   totalAmount,
		TotalDiscount: totalDiscount,
		NetAmount:     totalAmount,
	}, nil
}

// GetMonthlyReport retrieves a monthly report.
func (r *adminRepo) GetMonthlyReport(ctx context.Context, lotID uuid.UUID, year, month int) (*biz.MonthlyReport, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	// Count entries
	entries, _ := r.data.db.ParkingRecord.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.EntryTimeGTE(startOfMonth),
			parkingrecord.EntryTimeLT(endOfMonth),
		).
		Count(ctx)

	// Count exits
	exits, _ := r.data.db.ParkingRecord.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.ExitTimeNotNil(),
			parkingrecord.ExitTimeGTE(startOfMonth),
			parkingrecord.ExitTimeLT(endOfMonth),
		).
		Count(ctx)

	// Calculate total amount
	orders, _ := r.data.db.Order.Query().
		Where(
			order.LotID(lotID),
			order.StatusEQ(order.StatusPaid),
			order.PayTimeNotNil(),
			order.PayTimeGTE(startOfMonth),
			order.PayTimeLT(endOfMonth),
		).
		All(ctx)

	var totalAmount, totalDiscount float64
	for _, o := range orders {
		totalAmount += o.FinalAmount
		totalDiscount += o.DiscountAmount
	}

	return &biz.MonthlyReport{
		LotID:         lotID.String(),
		Year:          year,
		Month:         month,
		TotalEntries:  entries,
		TotalExits:    exits,
		TotalVehicles: entries,
		TotalAmount:   totalAmount,
		TotalDiscount: totalDiscount,
		NetAmount:     totalAmount,
	}, nil
}

// GetUserByUsername retrieves a user by username.
func (r *adminRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	u, err := r.data.db.User.Query().Where(user.Username(username)).First(ctx)
	if err != nil {
		return nil, err
	}

	return &biz.User{
		ID:        u.ID,
		Username:  u.Username,
		Password:  u.Password,
		Name:      u.Name,
		Role:      u.Role,
		Avatar:    u.Avatar,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}, nil
}

// GetUserByID retrieves a user by ID.
func (r *adminRepo) GetUserByID(ctx context.Context, userID uuid.UUID) (*biz.User, error) {
	u, err := r.data.db.User.Query().Where(user.ID(userID)).First(ctx)
	if err != nil {
		return nil, err
	}

	return &biz.User{
		ID:        u.ID,
		Username:  u.Username,
		Password:  u.Password,
		Name:      u.Name,
		Role:      u.Role,
		Avatar:    u.Avatar,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}, nil
}

// ListUsers lists all users with pagination.
func (r *adminRepo) ListUsers(ctx context.Context, page, pageSize int) ([]*biz.User, int64, error) {
	total, err := r.data.db.User.Query().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	users, err := r.data.db.User.Query().
		Order(ent.Desc("created_at")).
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	var result []*biz.User
	for _, u := range users {
		result = append(result, &biz.User{
			ID:        u.ID,
			Username:  u.Username,
			Password:  u.Password,
			Name:      u.Name,
			Role:      u.Role,
			Avatar:    u.Avatar,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		})
	}

	return result, int64(total), nil
}

// CreateUser creates a new user.
func (r *adminRepo) CreateUser(ctx context.Context, u *biz.User) error {
	_, err := r.data.db.User.Create().
		SetID(u.ID).
		SetUsername(u.Username).
		SetPassword(u.Password).
		SetName(u.Name).
		SetRole(u.Role).
		SetAvatar(u.Avatar).
		SetCreatedAt(u.CreatedAt).
		SetUpdatedAt(u.UpdatedAt).
		Save(ctx)
	return err
}

// UpdateUser updates an existing user.
func (r *adminRepo) UpdateUser(ctx context.Context, u *biz.User) error {
	_, err := r.data.db.User.UpdateOneID(u.ID).
		SetUsername(u.Username).
		SetName(u.Name).
		SetRole(u.Role).
		SetAvatar(u.Avatar).
		SetUpdatedAt(u.UpdatedAt).
		Save(ctx)
	return err
}

// DeleteUser deletes a user by ID.
func (r *adminRepo) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	return r.data.db.User.DeleteOneID(userID).Exec(ctx)
}

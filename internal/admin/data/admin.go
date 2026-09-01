package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/xuanyiying/smart-park/internal/admin/biz"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/order"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/parkingrecord"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/user"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent/vehicle"
	"github.com/xuanyiying/smart-park/pkg/database"
	apperrors "github.com/xuanyiying/smart-park/pkg/errors"
)

type adminRepo struct {
	data *Data
}

func NewAdminRepo(data *Data) biz.AdminRepo {
	return &adminRepo{data: data}
}

func (r *adminRepo) CreateParkingLot(ctx context.Context, lot *biz.ParkingLot) error {
	_, err := r.clientFromCtx(ctx).ParkingLot.Create().
		SetID(lot.ID).
		SetName(lot.Name).
		SetAddress(lot.Address).
		SetLanes(lot.Lanes).
		SetStatus(lot.Status).
		Save(ctx)

	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to create parking lot: %v", err)
		return apperrors.Wrapf(err, apperrors.CodeInternal, "创建停车场失败")
	}

	return nil
}

func (r *adminRepo) GetParkingLot(ctx context.Context, lotID uuid.UUID) (*biz.ParkingLot, error) {
	lot, err := r.clientFromCtx(ctx).ParkingLot.Get(ctx, lotID)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to get parking lot: %v", err)
		return nil, apperrors.NotFoundf("停车场不存在: %s", lotID)
	}

	return &biz.ParkingLot{
		ID:        lot.ID,
		Name:      lot.Name,
		Address:   lot.Address,
		Lanes:     lot.Lanes,
		Status:    lot.Status,
		CreatedAt: lot.CreatedAt,
		UpdatedAt: lot.UpdatedAt,
	}, nil
}

func (r *adminRepo) UpdateParkingLot(ctx context.Context, lot *biz.ParkingLot) error {
	_, err := r.clientFromCtx(ctx).ParkingLot.UpdateOneID(lot.ID).
		SetName(lot.Name).
		SetAddress(lot.Address).
		SetLanes(lot.Lanes).
		SetStatus(lot.Status).
		Save(ctx)

	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to update parking lot: %v", err)
		return apperrors.Wrapf(err, apperrors.CodeInternal, "更新停车场失败")
	}

	return nil
}

func (r *adminRepo) ListParkingLots(ctx context.Context, page, pageSize int) ([]*biz.ParkingLot, int64, error) {
	offset := (page - 1) * pageSize

	lots, err := r.clientFromCtx(ctx).ParkingLot.Query().
		Offset(offset).
		Limit(pageSize).
		All(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to list parking lots: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "查询停车场列表失败")
	}

	total, err := r.clientFromCtx(ctx).ParkingLot.Query().Count(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to count parking lots: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "统计停车场数量失败")
	}

	result := make([]*biz.ParkingLot, 0, len(lots))
	for _, lot := range lots {
		result = append(result, &biz.ParkingLot{
			ID:        lot.ID,
			Name:      lot.Name,
			Address:   lot.Address,
			Lanes:     lot.Lanes,
			Status:    lot.Status,
			CreatedAt: lot.CreatedAt,
			UpdatedAt: lot.UpdatedAt,
		})
	}

	return result, int64(total), nil
}

func (r *adminRepo) CreateVehicle(ctx context.Context, vehicle *biz.Vehicle) error {
	create := r.clientFromCtx(ctx).Vehicle.Create().
		SetID(vehicle.ID).
		SetPlateNumber(vehicle.PlateNumber).
		SetVehicleType(vehicle.VehicleType).
		SetOwnerName(vehicle.OwnerName).
		SetOwnerPhone(vehicle.OwnerPhone)
	if vehicle.MonthlyValidUntil != nil {
		create.SetMonthlyValidUntil(*vehicle.MonthlyValidUntil)
	}
	_, err := create.Save(ctx)

	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to create vehicle: %v", err)
		return apperrors.Wrapf(err, apperrors.CodeInternal, "创建车辆信息失败")
	}

	return nil
}

func (r *adminRepo) ListVehicles(ctx context.Context, vehicleType string, page, pageSize int) ([]*biz.Vehicle, int64, error) {
	offset := (page - 1) * pageSize
	query := r.clientFromCtx(ctx).Vehicle.Query()

	if vehicleType != "" {
		query = query.Where(vehicle.VehicleType(vehicleType))
	}

	vehicles, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to list vehicles: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "查询车辆列表失败")
	}

	total, err := query.Count(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to count vehicles: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "统计车辆数量失败")
	}

	result := make([]*biz.Vehicle, 0, len(vehicles))
	for _, v := range vehicles {
		result = append(result, &biz.Vehicle{
			ID:                v.ID,
			PlateNumber:       v.PlateNumber,
			VehicleType:       v.VehicleType,
			OwnerName:         v.OwnerName,
			OwnerPhone:        v.OwnerPhone,
			MonthlyValidUntil: v.MonthlyValidUntil,
			CreatedAt:         v.CreatedAt,
		})
	}

	return result, int64(total), nil
}

func (r *adminRepo) ListParkingRecords(ctx context.Context, lotID uuid.UUID, plateNumber, startTime, endTime string, page, pageSize int) ([]*biz.ParkingRecord, int64, error) {
	offset := (page - 1) * pageSize
	query := r.clientFromCtx(ctx).ParkingRecord.Query()

	if lotID != uuid.Nil {
		query = query.Where(parkingrecord.LotID(lotID))
	}

	if plateNumber != "" {
		query = query.Where(parkingrecord.PlateNumber(plateNumber))
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

	records, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to list parking records: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "查询停车记录失败")
	}

	total, err := query.Count(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to count parking records: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "统计停车记录数量失败")
	}

	result := make([]*biz.ParkingRecord, 0, len(records))
	for _, rec := range records {
		plateNum := ""
		if rec.PlateNumber != nil {
			plateNum = *rec.PlateNumber
		}
		result = append(result, &biz.ParkingRecord{
			ID:              rec.ID,
			LotID:           rec.LotID,
			PlateNumber:     plateNum,
			EntryTime:       rec.EntryTime,
			ExitTime:        rec.ExitTime,
			ParkingDuration: rec.ParkingDuration,
			Status:          string(rec.RecordStatus),
		})
	}

	return result, int64(total), nil
}

func (r *adminRepo) ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*biz.Order, int64, error) {
	offset := (page - 1) * pageSize
	query := r.clientFromCtx(ctx).Order.Query()

	if lotID != uuid.Nil {
		query = query.Where(order.LotID(lotID))
	}

	if status != "" {
		query = query.Where(order.StatusEQ(order.Status(status)))
	}

	orders, err := query.Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to list orders: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "查询订单列表失败")
	}

	total, err := query.Count(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to count orders: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "统计订单数量失败")
	}

	result := make([]*biz.Order, 0, len(orders))
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

func (r *adminRepo) GetOrder(ctx context.Context, orderID uuid.UUID) (*biz.Order, error) {
	o, err := r.clientFromCtx(ctx).Order.Get(ctx, orderID)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to get order: %v", err)
		return nil, apperrors.NotFoundf("订单不存在: %s", orderID)
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

func (r *adminRepo) GetDailyReport(ctx context.Context, lotID uuid.UUID, date string) (*biz.DailyReport, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, apperrors.InvalidArgument("无效的日期格式")
	}

	startOfDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	entryCount, err := r.clientFromCtx(ctx).ParkingRecord.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.EntryTimeGTE(startOfDay),
			parkingrecord.EntryTimeLT(endOfDay),
		).
		Count(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to count entries: %v", err)
		return nil, apperrors.Wrapf(err, apperrors.CodeInternal, "统计入场记录失败")
	}

	exitCount, err := r.clientFromCtx(ctx).ParkingRecord.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.ExitTimeNotNil(),
			parkingrecord.ExitTimeGTE(startOfDay),
			parkingrecord.ExitTimeLT(endOfDay),
		).
		Count(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to count exits: %v", err)
		return nil, apperrors.Wrapf(err, apperrors.CodeInternal, "统计出场记录失败")
	}

	orders, err := r.clientFromCtx(ctx).Order.Query().
		Where(
			order.LotID(lotID),
			order.StatusEQ(order.StatusPaid),
			order.PayTimeNotNil(),
			order.PayTimeGTE(startOfDay),
			order.PayTimeLT(endOfDay),
		).
		All(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to query orders: %v", err)
		return nil, apperrors.Wrapf(err, apperrors.CodeInternal, "统计订单失败")
	}

	var totalAmount, totalDiscount int64
	for _, o := range orders {
		totalAmount += o.Amount
		totalDiscount += o.DiscountAmount
	}

	return &biz.DailyReport{
		LotID:         lotID.String(),
		Date:          date,
		TotalEntries:  entryCount,
		TotalExits:    exitCount,
		TotalAmount:   totalAmount,
		TotalDiscount: totalDiscount,
		NetAmount:     totalAmount - totalDiscount,
	}, nil
}

func (r *adminRepo) GetMonthlyReport(ctx context.Context, lotID uuid.UUID, year, month int) (*biz.MonthlyReport, error) {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	entryCount, err := r.clientFromCtx(ctx).ParkingRecord.Query().
		Where(
			parkingrecord.LotID(lotID),
			parkingrecord.EntryTimeGTE(startOfMonth),
			parkingrecord.EntryTimeLT(endOfMonth),
		).
		Count(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to count monthly entries: %v", err)
		return nil, apperrors.Wrapf(err, apperrors.CodeInternal, "统计月度入场记录失败")
	}

	orders, err := r.clientFromCtx(ctx).Order.Query().
		Where(
			order.LotID(lotID),
			order.StatusEQ(order.StatusPaid),
			order.PayTimeNotNil(),
			order.PayTimeGTE(startOfMonth),
			order.PayTimeLT(endOfMonth),
		).
		All(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to query monthly orders: %v", err)
		return nil, apperrors.Wrapf(err, apperrors.CodeInternal, "统计月度订单失败")
	}

	var totalAmount, totalDiscount int64
	for _, o := range orders {
		totalAmount += o.Amount
		totalDiscount += o.DiscountAmount
	}

	return &biz.MonthlyReport{
		LotID:         lotID.String(),
		Year:          year,
		Month:         month,
		TotalEntries:  entryCount,
		TotalAmount:   totalAmount,
		TotalDiscount: totalDiscount,
		NetAmount:     totalAmount - totalDiscount,
	}, nil
}

func (r *adminRepo) GetUserByUsername(ctx context.Context, username string) (*biz.User, error) {
	u, err := r.clientFromCtx(ctx).User.Query().
		Where(user.Username(username)).
		Only(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to get user by username: %v", err)
		return nil, apperrors.NotFoundf("用户不存在: %s", username)
	}

	return r.toBizUser(u), nil
}

func (r *adminRepo) GetUserByID(ctx context.Context, userID uuid.UUID) (*biz.User, error) {
	u, err := r.clientFromCtx(ctx).User.Get(ctx, userID)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to get user: %v", err)
		return nil, apperrors.NotFoundf("用户不存在: %s", userID)
	}

	return r.toBizUser(u), nil
}

func (r *adminRepo) ListUsers(ctx context.Context, page, pageSize int) ([]*biz.User, int64, error) {
	offset := (page - 1) * pageSize

	users, err := r.clientFromCtx(ctx).User.Query().Offset(offset).Limit(pageSize).All(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to list users: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "查询用户列表失败")
	}

	total, err := r.clientFromCtx(ctx).User.Query().Count(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to count users: %v", err)
		return nil, 0, apperrors.Wrapf(err, apperrors.CodeInternal, "统计用户数量失败")
	}

	result := make([]*biz.User, 0, len(users))
	for _, u := range users {
		result = append(result, r.toBizUser(u))
	}

	return result, int64(total), nil
}

func (r *adminRepo) CreateUser(ctx context.Context, user *biz.User) error {
	_, err := r.clientFromCtx(ctx).User.Create().
		SetID(user.ID).
		SetUsername(user.Username).
		SetPassword(user.Password).
		SetName(user.Name).
		SetRole(user.Role).
		SetAvatar(user.Avatar).
		Save(ctx)

	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to create user: %v", err)
		return apperrors.Wrapf(err, apperrors.CodeInternal, "创建用户失败")
	}

	return nil
}

func (r *adminRepo) UpdateUser(ctx context.Context, user *biz.User) error {
	_, err := r.clientFromCtx(ctx).User.UpdateOneID(user.ID).
		SetUsername(user.Username).
		SetPassword(user.Password).
		SetName(user.Name).
		SetRole(user.Role).
		SetAvatar(user.Avatar).
		Save(ctx)

	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to update user: %v", err)
		return apperrors.Wrapf(err, apperrors.CodeInternal, "更新用户失败")
	}

	return nil
}

func (r *adminRepo) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	err := r.clientFromCtx(ctx).User.DeleteOneID(userID).Exec(ctx)
	if err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to delete user: %v", err)
		return apperrors.Wrapf(err, apperrors.CodeInternal, "删除用户失败")
	}

	return nil
}

// SeedData provisions a starting parking lot for development.
//
// It is idempotent: the seed only runs while the table is empty. The previous version had
// no such guard, so every service restart inserted another "测试停车场" row into
// production and the parking-lot list silently filled with junk.
func (r *adminRepo) SeedData(ctx context.Context) error {
	count, err := r.clientFromCtx(ctx).ParkingLot.Query().Count(ctx)
	if err != nil {
		return apperrors.Wrapf(err, apperrors.CodeInternal, "检查停车场数据失败")
	}
	if count > 0 {
		// Parking lots already exist, skip seeding.
		return nil
	}

	lotID := uuid.New()
	if _, err := r.clientFromCtx(ctx).ParkingLot.Create().
		SetID(lotID).
		SetName("测试停车场").
		SetAddress("测试地址").
		SetLanes(4).
		SetStatus("active").
		Save(ctx); err != nil {
		r.data.log.WithContext(ctx).Errorf("failed to seed parking lot: %v", err)
		return apperrors.Wrapf(err, apperrors.CodeInternal, "初始化停车场数据失败")
	}

	return nil
}

func (r *adminRepo) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return r.data.txm.WithTx(ctx, fn)
}

func (r *adminRepo) clientFromCtx(ctx context.Context) *ent.Client {
	if tx, ok := database.TxFromCtx(ctx).(*ent.Tx); ok {
		return tx.Client()
	}
	return r.data.db
}

func (r *adminRepo) toBizUser(user *ent.User) *biz.User {
	return &biz.User{
		ID:        user.ID,
		Username:  user.Username,
		Password:  user.Password,
		Name:      user.Name,
		Role:      user.Role,
		Avatar:    user.Avatar,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}

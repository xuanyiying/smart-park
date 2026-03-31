// Package biz provides business logic for the admin service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	v1 "github.com/xuanyiying/smart-park/api/admin/v1"
)

// ParkingLot represents a parking lot entity.
type ParkingLot struct {
	ID        uuid.UUID
	Name      string
	Address   string
	Lanes     int
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Vehicle represents a vehicle entity.
type Vehicle struct {
	ID                uuid.UUID
	PlateNumber       string
	VehicleType       string
	OwnerName         string
	OwnerPhone        string
	MonthlyValidUntil *time.Time
	CreatedAt         time.Time
}

// ParkingRecord represents a parking record entity.
type ParkingRecord struct {
	ID              uuid.UUID
	LotID           uuid.UUID
	PlateNumber     string
	EntryTime       time.Time
	ExitTime        *time.Time
	ParkingDuration int
	Status          string
}

// Order represents an order entity.
type Order struct {
	ID             uuid.UUID
	RecordID       uuid.UUID
	LotID          uuid.UUID
	PlateNumber    string
	Amount         float64
	DiscountAmount float64
	FinalAmount    float64
	Status         string
	PayTime        *time.Time
	PayMethod      string
}

// DailyReport represents a daily report.
type DailyReport struct {
	LotID         string
	Date          string
	TotalEntries  int
	TotalExits    int
	TotalVehicles int
	TotalAmount   float64
	TotalDiscount float64
	NetAmount     float64
}

// MonthlyReport represents a monthly report.
type MonthlyReport struct {
	LotID         string
	Year          int
	Month         int
	TotalEntries  int
	TotalExits    int
	TotalVehicles int
	TotalAmount   float64
	TotalDiscount float64
	NetAmount     float64
	DailyReports  []*DailyReport
}

// User represents an admin user entity.
type User struct {
	ID        uuid.UUID
	Username  string
	Password  string
	Name      string
	Role      string
	Avatar    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// AdminRepo defines the repository interface for admin operations.
type AdminRepo interface {
	CreateParkingLot(ctx context.Context, lot *ParkingLot) error
	GetParkingLot(ctx context.Context, lotID uuid.UUID) (*ParkingLot, error)
	UpdateParkingLot(ctx context.Context, lot *ParkingLot) error
	ListParkingLots(ctx context.Context, page, pageSize int) ([]*ParkingLot, int64, error)
	CreateVehicle(ctx context.Context, vehicle *Vehicle) error
	ListVehicles(ctx context.Context, vehicleType string, page, pageSize int) ([]*Vehicle, int64, error)
	ListParkingRecords(ctx context.Context, lotID uuid.UUID, plateNumber, startTime, endTime string, page, pageSize int) ([]*ParkingRecord, int64, error)
	ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*Order, int64, error)
	GetOrder(ctx context.Context, orderID uuid.UUID) (*Order, error)
	GetDailyReport(ctx context.Context, lotID uuid.UUID, date string) (*DailyReport, error)
	GetMonthlyReport(ctx context.Context, lotID uuid.UUID, year, month int) (*MonthlyReport, error)
	// User related methods
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error)
	ListUsers(ctx context.Context, page, pageSize int) ([]*User, int64, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	// SeedData creates initial seed data for development
	SeedData(ctx context.Context) error
}

// AdminUseCase implements admin business logic.
type AdminUseCase struct {
	repo AdminRepo
	log  *log.Helper
}

// NewAdminUseCase creates a new AdminUseCase.
func NewAdminUseCase(repo AdminRepo, logger log.Logger) *AdminUseCase {
	return &AdminUseCase{
		repo: repo,
		log:  log.NewHelper(logger),
	}
}

// CreateParkingLot creates a new parking lot.
func (uc *AdminUseCase) CreateParkingLot(ctx context.Context, req *v1.CreateParkingLotRequest) (*v1.ParkingLot, error) {
	lot := &ParkingLot{
		ID:      uuid.New(),
		Name:    req.Name,
		Address: req.Address,
		Lanes:   int(req.Lanes),
		Status:  "active",
	}

	if err := uc.repo.CreateParkingLot(ctx, lot); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create parking lot: %v", err)
		return nil, err
	}

	return &v1.ParkingLot{
		Id:        lot.ID.String(),
		Name:      lot.Name,
		Address:   lot.Address,
		Lanes:     int32(lot.Lanes),
		Status:    lot.Status,
		CreatedAt: lot.CreatedAt.Format(time.RFC3339),
		UpdatedAt: lot.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// GetParkingLot retrieves a parking lot.
func (uc *AdminUseCase) GetParkingLot(ctx context.Context, id string) (*v1.ParkingLot, error) {
	lotID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	lot, err := uc.repo.GetParkingLot(ctx, lotID)
	if err != nil {
		return nil, err
	}

	return &v1.ParkingLot{
		Id:        lot.ID.String(),
		Name:      lot.Name,
		Address:   lot.Address,
		Lanes:     int32(lot.Lanes),
		Status:    lot.Status,
		CreatedAt: lot.CreatedAt.Format(time.RFC3339),
		UpdatedAt: lot.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// UpdateParkingLot updates a parking lot.
func (uc *AdminUseCase) UpdateParkingLot(ctx context.Context, req *v1.UpdateParkingLotRequest) error {
	lotID, err := uuid.Parse(req.Id)
	if err != nil {
		return err
	}

	lot := &ParkingLot{
		ID:      lotID,
		Name:    req.Name,
		Address: req.Address,
		Lanes:   int(req.Lanes),
		Status:  req.Status,
	}

	return uc.repo.UpdateParkingLot(ctx, lot)
}

// ListParkingLots lists parking lots.
func (uc *AdminUseCase) ListParkingLots(ctx context.Context, req *v1.ListParkingLotsRequest) (*v1.ParkingLotListData, error) {
	lots, total, err := uc.repo.ListParkingLots(ctx, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	var items []*v1.ParkingLot
	for _, lot := range lots {
		items = append(items, &v1.ParkingLot{
			Id:        lot.ID.String(),
			Name:      lot.Name,
			Address:   lot.Address,
			Lanes:     int32(lot.Lanes),
			Status:    lot.Status,
			CreatedAt: lot.CreatedAt.Format(time.RFC3339),
			UpdatedAt: lot.UpdatedAt.Format(time.RFC3339),
		})
	}

	return &v1.ParkingLotListData{
		List:     items,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// CreateVehicle creates a new vehicle.
func (uc *AdminUseCase) CreateVehicle(ctx context.Context, req *v1.CreateVehicleRequest) (*v1.Vehicle, error) {
	var monthlyValidUntil *time.Time
	if req.MonthlyValidUntil != "" {
		t, err := time.Parse(time.RFC3339, req.MonthlyValidUntil)
		if err == nil {
			monthlyValidUntil = &t
		}
	}

	vehicle := &Vehicle{
		ID:                uuid.New(),
		PlateNumber:       req.PlateNumber,
		VehicleType:       req.VehicleType,
		OwnerName:         req.OwnerName,
		OwnerPhone:        req.OwnerPhone,
		MonthlyValidUntil: monthlyValidUntil,
	}

	if err := uc.repo.CreateVehicle(ctx, vehicle); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to create vehicle: %v", err)
		return nil, err
	}

	var validUntil string
	if vehicle.MonthlyValidUntil != nil {
		validUntil = vehicle.MonthlyValidUntil.Format(time.RFC3339)
	}

	return &v1.Vehicle{
		Id:                vehicle.ID.String(),
		PlateNumber:       vehicle.PlateNumber,
		VehicleType:       vehicle.VehicleType,
		OwnerName:         vehicle.OwnerName,
		OwnerPhone:        vehicle.OwnerPhone,
		MonthlyValidUntil: validUntil,
		CreatedAt:         vehicle.CreatedAt.Format(time.RFC3339),
	}, nil
}

// ListVehicles lists vehicles.
func (uc *AdminUseCase) ListVehicles(ctx context.Context, req *v1.ListVehiclesRequest) (*v1.VehicleListData, error) {
	vehicles, total, err := uc.repo.ListVehicles(ctx, req.VehicleType, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	var items []*v1.Vehicle
	for _, v := range vehicles {
		var validUntil string
		if v.MonthlyValidUntil != nil {
			validUntil = v.MonthlyValidUntil.Format(time.RFC3339)
		}
		items = append(items, &v1.Vehicle{
			Id:                v.ID.String(),
			PlateNumber:       v.PlateNumber,
			VehicleType:       v.VehicleType,
			OwnerName:         v.OwnerName,
			OwnerPhone:        v.OwnerPhone,
			MonthlyValidUntil: validUntil,
			CreatedAt:         v.CreatedAt.Format(time.RFC3339),
		})
	}

	return &v1.VehicleListData{
		List:     items,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ListParkingRecords lists parking records.
func (uc *AdminUseCase) ListParkingRecords(ctx context.Context, req *v1.ListParkingRecordsRequest) (*v1.ParkingRecordListData, error) {
	var lotID uuid.UUID
	if req.LotId != "" {
		var err error
		lotID, err = uuid.Parse(req.LotId)
		if err != nil {
			return nil, err
		}
	}

	records, total, err := uc.repo.ListParkingRecords(ctx, lotID, req.PlateNumber, req.StartTime, req.EndTime, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	var items []*v1.ParkingRecord
	for _, rec := range records {
		items = append(items, &v1.ParkingRecord{
			Id:              rec.ID.String(),
			LotId:           rec.LotID.String(),
			PlateNumber:     rec.PlateNumber,
			EntryTime:       rec.EntryTime.Unix(),
			ParkingDuration: int32(rec.ParkingDuration),
			Status:          rec.Status,
		})
	}

	return &v1.ParkingRecordListData{
		List:     items,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// ListOrders lists orders.
func (uc *AdminUseCase) ListOrders(ctx context.Context, req *v1.ListOrdersRequest) (*v1.OrderListData, error) {
	var lotID uuid.UUID
	if req.LotId != "" {
		var err error
		lotID, err = uuid.Parse(req.LotId)
		if err != nil {
			return nil, err
		}
	}

	orders, total, err := uc.repo.ListOrders(ctx, lotID, req.Status, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, err
	}

	var items []*v1.Order
	for _, o := range orders {
		var payTime string
		if o.PayTime != nil {
			payTime = o.PayTime.Format(time.RFC3339)
		}
		items = append(items, &v1.Order{
			Id:             o.ID.String(),
			RecordId:       o.RecordID.String(),
			LotId:          o.LotID.String(),
			PlateNumber:    o.PlateNumber,
			Amount:         o.Amount,
			DiscountAmount: o.DiscountAmount,
			FinalAmount:    o.FinalAmount,
			Status:         o.Status,
			PayTime:        payTime,
			PayMethod:      o.PayMethod,
		})
	}

	return &v1.OrderListData{
		List:     items,
		Total:    int32(total),
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetOrder retrieves an order.
func (uc *AdminUseCase) GetOrder(ctx context.Context, id string) (*v1.Order, error) {
	orderID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	order, err := uc.repo.GetOrder(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var payTime string
	if order.PayTime != nil {
		payTime = order.PayTime.Format(time.RFC3339)
	}

	return &v1.Order{
		Id:             order.ID.String(),
		RecordId:       order.RecordID.String(),
		LotId:          order.LotID.String(),
		PlateNumber:    order.PlateNumber,
		Amount:         order.Amount,
		DiscountAmount: order.DiscountAmount,
		FinalAmount:    order.FinalAmount,
		Status:         order.Status,
		PayTime:        payTime,
		PayMethod:      order.PayMethod,
	}, nil
}

// GetDailyReport retrieves a daily report.
func (uc *AdminUseCase) GetDailyReport(ctx context.Context, req *v1.GetDailyReportRequest) (*v1.DailyReport, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	report, err := uc.repo.GetDailyReport(ctx, lotID, req.Date)
	if err != nil {
		return nil, err
	}

	return &v1.DailyReport{
		LotId:         report.LotID,
		Date:          report.Date,
		TotalEntries:  int32(report.TotalEntries),
		TotalExits:    int32(report.TotalExits),
		TotalVehicles: int32(report.TotalVehicles),
		TotalAmount:   report.TotalAmount,
		TotalDiscount: report.TotalDiscount,
		NetAmount:     report.NetAmount,
	}, nil
}

// GetMonthlyReport retrieves a monthly report.
func (uc *AdminUseCase) GetMonthlyReport(ctx context.Context, req *v1.GetMonthlyReportRequest) (*v1.MonthlyReport, error) {
	lotID, err := uuid.Parse(req.LotId)
	if err != nil {
		return nil, err
	}

	report, err := uc.repo.GetMonthlyReport(ctx, lotID, int(req.Year), int(req.Month))
	if err != nil {
		return nil, err
	}

	var dailyReports []*v1.DailyReport
	for _, dr := range report.DailyReports {
		dailyReports = append(dailyReports, &v1.DailyReport{
			LotId:         dr.LotID,
			Date:          dr.Date,
			TotalEntries:  int32(dr.TotalEntries),
			TotalExits:    int32(dr.TotalExits),
			TotalVehicles: int32(dr.TotalVehicles),
			TotalAmount:   dr.TotalAmount,
			TotalDiscount: dr.TotalDiscount,
			NetAmount:     dr.NetAmount,
		})
	}

	return &v1.MonthlyReport{
		LotId:         report.LotID,
		Year:          int32(report.Year),
		Month:         int32(report.Month),
		TotalEntries:  int32(report.TotalEntries),
		TotalExits:    int32(report.TotalExits),
		TotalVehicles: int32(report.TotalVehicles),
		TotalAmount:   report.TotalAmount,
		TotalDiscount: report.TotalDiscount,
		NetAmount:     report.NetAmount,
		DailyReports:  dailyReports,
	}, nil
}

// contextKey is the key for context values.
type contextKey string

const userContextKey contextKey = "user"

// Login validates user credentials and returns a token.
func (uc *AdminUseCase) Login(ctx context.Context, username, password string) (*User, string, int64, error) {
	user, err := uc.repo.GetUserByUsername(ctx, username)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("user not found: %v", err)
		return nil, "", 0, err
	}

	// Use bcrypt for password comparison in production
	if err := uc.comparePassword(user.Password, password); err != nil {
		uc.log.WithContext(ctx).Errorf("invalid password for user: %s", username)
		return nil, "", 0, fmt.Errorf("invalid credentials")
	}

	// Generate token (in production, use JWT)
	token := uuid.New().String()
	expiresAt := time.Now().Add(24 * time.Hour).Unix()

	return user, token, expiresAt, nil
}

// comparePassword compares password with hash using bcrypt.
func (uc *AdminUseCase) comparePassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GetCurrentUser retrieves the current user from context.
func (uc *AdminUseCase) GetCurrentUser(ctx context.Context) (*User, error) {
	user, ok := ctx.Value(userContextKey).(*User)
	if !ok {
		return nil, fmt.Errorf("user not found in context")
	}
	return user, nil
}

// SetUserInContext sets the user in the context.
func SetUserInContext(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// ListUsers lists all users with pagination.
func (uc *AdminUseCase) ListUsers(ctx context.Context, page, pageSize int) ([]*User, int64, error) {
	return uc.repo.ListUsers(ctx, page, pageSize)
}

// CreateUser creates a new user.
func (uc *AdminUseCase) CreateUser(ctx context.Context, username, password, name, role, email, status string) (*User, error) {
	user := &User{
		ID:        uuid.New(),
		Username:  username,
		Password:  password,
		Name:      name,
		Role:      role,
		Avatar:    "",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := uc.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// UpdateUser updates an existing user.
func (uc *AdminUseCase) UpdateUser(ctx context.Context, id, username, name, role, email, status string) (*User, error) {
	userID, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}

	user, err := uc.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if username != "" {
		user.Username = username
	}
	if name != "" {
		user.Name = name
	}
	if role != "" {
		user.Role = role
	}
	user.UpdatedAt = time.Now()

	if err := uc.repo.UpdateUser(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

// DeleteUser deletes a user by ID.
func (uc *AdminUseCase) DeleteUser(ctx context.Context, id string) error {
	userID, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	return uc.repo.DeleteUser(ctx, userID)
}

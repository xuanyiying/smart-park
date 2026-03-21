package biz

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/admin/v1"
)

// MockAdminRepo is a mock implementation of AdminRepo for testing.
type MockAdminRepo struct {
	ParkingLots  map[uuid.UUID]*ParkingLot
	Vehicles     map[uuid.UUID]*Vehicle
	Orders       map[uuid.UUID]*Order
	DailyReports map[string]*DailyReport
}

func NewMockAdminRepo() *MockAdminRepo {
	return &MockAdminRepo{
		ParkingLots:  make(map[uuid.UUID]*ParkingLot),
		Vehicles:     make(map[uuid.UUID]*Vehicle),
		Orders:       make(map[uuid.UUID]*Order),
		DailyReports: make(map[string]*DailyReport),
	}
}

func (m *MockAdminRepo) CreateParkingLot(ctx context.Context, lot *ParkingLot) error {
	m.ParkingLots[lot.ID] = lot
	return nil
}

func (m *MockAdminRepo) GetParkingLot(ctx context.Context, lotID uuid.UUID) (*ParkingLot, error) {
	return m.ParkingLots[lotID], nil
}

func (m *MockAdminRepo) UpdateParkingLot(ctx context.Context, lot *ParkingLot) error {
	m.ParkingLots[lot.ID] = lot
	return nil
}

func (m *MockAdminRepo) ListParkingLots(ctx context.Context, page, pageSize int) ([]*ParkingLot, int64, error) {
	var lots []*ParkingLot
	for _, lot := range m.ParkingLots {
		lots = append(lots, lot)
	}
	return lots, int64(len(lots)), nil
}

func (m *MockAdminRepo) CreateVehicle(ctx context.Context, vehicle *Vehicle) error {
	m.Vehicles[vehicle.ID] = vehicle
	return nil
}

func (m *MockAdminRepo) ListVehicles(ctx context.Context, vehicleType string, page, pageSize int) ([]*Vehicle, int64, error) {
	var vehicles []*Vehicle
	for _, v := range m.Vehicles {
		if vehicleType == "" || v.VehicleType == vehicleType {
			vehicles = append(vehicles, v)
		}
	}
	return vehicles, int64(len(vehicles)), nil
}

func (m *MockAdminRepo) ListParkingRecords(ctx context.Context, lotID uuid.UUID, plateNumber, startTime, endTime string, page, pageSize int) ([]*ParkingRecord, int64, error) {
	return nil, 0, nil
}

func (m *MockAdminRepo) ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*Order, int64, error) {
	var orders []*Order
	for _, o := range m.Orders {
		orders = append(orders, o)
	}
	return orders, int64(len(orders)), nil
}

func (m *MockAdminRepo) GetOrder(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	return m.Orders[orderID], nil
}

func (m *MockAdminRepo) GetDailyReport(ctx context.Context, lotID uuid.UUID, date string) (*DailyReport, error) {
	key := lotID.String() + "-" + date
	return m.DailyReports[key], nil
}

func (m *MockAdminRepo) GetMonthlyReport(ctx context.Context, lotID uuid.UUID, year, month int) (*MonthlyReport, error) {
	return &MonthlyReport{
		LotID:        lotID.String(),
		Year:         year,
		Month:        month,
		TotalEntries: 100,
		TotalExits:   95,
	}, nil
}

func TestAdminUseCase_CreateParkingLot(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockAdminRepo()

	uc := NewAdminUseCase(mockRepo, logger)

	req := &v1.CreateParkingLotRequest{
		Name:    "Test Parking Lot",
		Address: "123 Main St",
		Lanes:   2,
	}

	lot, err := uc.CreateParkingLot(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateParkingLot failed: %v", err)
	}

	if lot == nil {
		t.Fatal("Expected non-nil lot")
	}

	if lot.Name != "Test Parking Lot" {
		t.Errorf("Expected name 'Test Parking Lot', got %s", lot.Name)
	}
}

func TestAdminUseCase_GetParkingLot(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockAdminRepo()

	lotID := uuid.New()
	mockRepo.ParkingLots[lotID] = &ParkingLot{
		ID:      lotID,
		Name:    "Existing Lot",
		Address: "456 Oak Ave",
		Lanes:   4,
		Status:  "active",
	}

	uc := NewAdminUseCase(mockRepo, logger)

	lot, err := uc.GetParkingLot(context.Background(), lotID.String())
	if err != nil {
		t.Fatalf("GetParkingLot failed: %v", err)
	}

	if lot == nil {
		t.Fatal("Expected non-nil lot")
	}

	if lot.Name != "Existing Lot" {
		t.Errorf("Expected name 'Existing Lot', got %s", lot.Name)
	}
}

func TestAdminUseCase_UpdateParkingLot(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockAdminRepo()

	lotID := uuid.New()
	mockRepo.ParkingLots[lotID] = &ParkingLot{
		ID:      lotID,
		Name:    "Old Name",
		Address: "Old Address",
		Lanes:   2,
		Status:  "active",
	}

	uc := NewAdminUseCase(mockRepo, logger)

	req := &v1.UpdateParkingLotRequest{
		Id:      lotID.String(),
		Name:    "New Name",
		Address: "New Address",
		Lanes:   4,
		Status:  "active",
	}

	err := uc.UpdateParkingLot(context.Background(), req)
	if err != nil {
		t.Fatalf("UpdateParkingLot failed: %v", err)
	}

	if mockRepo.ParkingLots[lotID].Name != "New Name" {
		t.Errorf("Expected name 'New Name', got %s", mockRepo.ParkingLots[lotID].Name)
	}
}

func TestAdminUseCase_CreateVehicle(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockAdminRepo()

	uc := NewAdminUseCase(mockRepo, logger)

	validUntil := time.Now().Add(30 * 24 * time.Hour).Format(time.RFC3339)

	req := &v1.CreateVehicleRequest{
		PlateNumber:       "京A88888",
		VehicleType:       "monthly",
		OwnerName:         "王五",
		OwnerPhone:        "13900139000",
		MonthlyValidUntil: validUntil,
	}

	vehicle, err := uc.CreateVehicle(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateVehicle failed: %v", err)
	}

	if vehicle == nil {
		t.Fatal("Expected non-nil vehicle")
	}

	if vehicle.PlateNumber != "京A88888" {
		t.Errorf("Expected plate number '京A88888', got %s", vehicle.PlateNumber)
	}
}

func TestAdminUseCase_GetDailyReport(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockAdminRepo()

	lotID := uuid.New()
	date := "2026-03-20"
	key := lotID.String() + "-" + date

	mockRepo.DailyReports[key] = &DailyReport{
		LotID:        lotID.String(),
		Date:         date,
		TotalEntries: 50,
		TotalExits:   45,
		TotalAmount:  500.00,
	}

	uc := NewAdminUseCase(mockRepo, logger)

	req := &v1.GetDailyReportRequest{
		LotId: lotID.String(),
		Date:  date,
	}

	report, err := uc.GetDailyReport(context.Background(), req)
	if err != nil {
		t.Fatalf("GetDailyReport failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	if report.TotalEntries != 50 {
		t.Errorf("Expected total entries 50, got %d", report.TotalEntries)
	}
}

func TestAdminUseCase_GetMonthlyReport(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockAdminRepo()

	uc := NewAdminUseCase(mockRepo, logger)

	lotID := uuid.New()
	req := &v1.GetMonthlyReportRequest{
		LotId: lotID.String(),
		Year:  2026,
		Month: 3,
	}

	report, err := uc.GetMonthlyReport(context.Background(), req)
	if err != nil {
		t.Fatalf("GetMonthlyReport failed: %v", err)
	}

	if report == nil {
		t.Fatal("Expected non-nil report")
	}

	if report.Year != 2026 {
		t.Errorf("Expected year 2026, got %d", report.Year)
	}
}

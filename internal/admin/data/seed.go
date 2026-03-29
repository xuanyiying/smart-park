package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/xuanyiying/smart-park/internal/admin/biz"
)

// SeedData creates initial seed data for development.
func (r *adminRepo) SeedData(ctx context.Context) error {
	// Check if users already exist
	_, err := r.GetUserByUsername(ctx, "admin")
	if err == nil {
		// Users already exist, skip seeding
		return nil
	}

	// Create default users
	users := []*biz.User{
		{
			ID:        uuid.New(),
			Username:  "admin",
			Password:  "admin123",
			Name:      "管理员",
			Role:      "admin",
			Avatar:    "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Username:  "operator",
			Password:  "op123456",
			Name:      "操作员",
			Role:      "operator",
			Avatar:    "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.New(),
			Username:  "manager",
			Password:  "mgr123456",
			Name:      "经理",
			Role:      "manager",
			Avatar:    "",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, u := range users {
		if err := r.CreateUser(ctx, u); err != nil {
			return err
		}
	}

	// Create sample parking lots
	lots := []*biz.ParkingLot{
		{
			ID:        uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:      "中心广场停车场",
			Address:   "北京市朝阳区中心广场1号",
			Lanes:     4,
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Name:      "科技园停车场",
			Address:   "北京市海淀区科技园路88号",
			Lanes:     6,
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        uuid.MustParse("33333333-3333-3333-3333-333333333333"),
			Name:      "商业街停车场",
			Address:   "上海市浦东新区商业街168号",
			Lanes:     3,
			Status:    "active",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, lot := range lots {
		if err := r.CreateParkingLot(ctx, lot); err != nil {
			return err
		}
	}

	// Create monthly vehicles
	now := time.Now()
	validUntil1 := now.Add(30 * 24 * time.Hour)
	validUntil2 := now.Add(60 * 24 * time.Hour)
	validUntil3 := now.Add(15 * 24 * time.Hour)

	vehicles := []*biz.Vehicle{
		{
			ID:                uuid.New(),
			PlateNumber:       "京A88888",
			VehicleType:       "monthly",
			OwnerName:         "张三",
			OwnerPhone:        "13800138000",
			MonthlyValidUntil: &validUntil1,
			CreatedAt:         now,
		},
		{
			ID:                uuid.New(),
			PlateNumber:       "京B12345",
			VehicleType:       "monthly",
			OwnerName:         "李四",
			OwnerPhone:        "13900139000",
			MonthlyValidUntil: &validUntil2,
			CreatedAt:         now,
		},
		{
			ID:                uuid.New(),
			PlateNumber:       "沪C66666",
			VehicleType:       "monthly",
			OwnerName:         "王五",
			OwnerPhone:        "13700137000",
			MonthlyValidUntil: &validUntil3,
			CreatedAt:         now,
		},
	}

	for _, v := range vehicles {
		if err := r.CreateVehicle(ctx, v); err != nil {
			return err
		}
	}

	return nil
}

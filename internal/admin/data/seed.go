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
			Username:  "test",
			Password:  "test123",
			Name:      "测试用户",
			Role:      "admin",
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

	// Create sample parking lot
	lot := &biz.ParkingLot{
		ID:        uuid.New(),
		Name:      "示例停车场",
		Address:   "北京市朝阳区示例路1号",
		Lanes:     4,
		Status:    "active",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := r.CreateParkingLot(ctx, lot); err != nil {
		return err
	}

	return nil
}

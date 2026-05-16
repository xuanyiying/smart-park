package biz

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

func TestVehicleQueryUseCase_GetVehicleInfo(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	validUntil := time.Now().Add(30 * 24 * time.Hour)

	tests := []struct {
		name        string
		plateNumber string
		setupRepo   func() *MockVehicleRepo
		wantErr     bool
		errContains string
		checkResult func(*testing.T, *v1.VehicleInfo)
	}{
		{
			name:        "empty plate number",
			plateNumber: "",
			setupRepo:   func() *MockVehicleRepo { return NewMockVehicleRepo() },
			wantErr:     true,
			errContains: "plate number is required",
		},
		{
			name:        "vehicle not found",
			plateNumber: "京Z99999",
			setupRepo:   func() *MockVehicleRepo { return NewMockVehicleRepo() },
			wantErr:     true,
			errContains: "vehicle not found",
		},
		{
			name:        "vehicle found with monthly pass",
			plateNumber: "京A12345",
			setupRepo: func() *MockVehicleRepo {
				repo := NewMockVehicleRepo()
				repo.Vehicles["京A12345"] = &Vehicle{
					ID:                uuid.New(),
					PlateNumber:       "京A12345",
					VehicleType:       "monthly",
					OwnerName:         "Zhang San",
					OwnerPhone:        "13800138000",
					MonthlyValidUntil: &validUntil,
				}
				return repo
			},
			wantErr: false,
			checkResult: func(t *testing.T, info *v1.VehicleInfo) {
				if info.PlateNumber != "京A12345" {
					t.Errorf("PlateNumber = %s, want 京A12345", info.PlateNumber)
				}
				if info.VehicleType != "monthly" {
					t.Errorf("VehicleType = %s, want monthly", info.VehicleType)
				}
				if info.MonthlyValidUntil == "" {
					t.Error("MonthlyValidUntil should not be empty")
				}
			},
		},
		{
			name:        "vehicle found without monthly pass",
			plateNumber: "京B67890",
			setupRepo: func() *MockVehicleRepo {
				repo := NewMockVehicleRepo()
				repo.Vehicles["京B67890"] = &Vehicle{
					ID:          uuid.New(),
					PlateNumber: "京B67890",
					VehicleType: "temporary",
					OwnerName:   "Li Si",
					OwnerPhone:  "13900139000",
				}
				return repo
			},
			wantErr: false,
			checkResult: func(t *testing.T, info *v1.VehicleInfo) {
				if info.MonthlyValidUntil != "" {
					t.Errorf("MonthlyValidUntil should be empty, got %s", info.MonthlyValidUntil)
				}
			},
		},
		{
			name:        "vip vehicle query",
			plateNumber: "京V88888",
			setupRepo: func() *MockVehicleRepo {
				repo := NewMockVehicleRepo()
				repo.Vehicles["京V88888"] = &Vehicle{
					ID:          uuid.New(),
					PlateNumber: "京V88888",
					VehicleType: "vip",
					OwnerName:   "Wang Wu",
					OwnerPhone:  "13700137000",
				}
				return repo
			},
			wantErr: false,
			checkResult: func(t *testing.T, info *v1.VehicleInfo) {
				if info.VehicleType != "vip" {
					t.Errorf("VehicleType = %s, want vip", info.VehicleType)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := tt.setupRepo()
			uc := NewVehicleQueryUseCase(repo, logger)

			result, err := uc.GetVehicleInfo(context.Background(), tt.plateNumber)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetVehicleInfo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errContains)
				}
			}
			if !tt.wantErr && tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

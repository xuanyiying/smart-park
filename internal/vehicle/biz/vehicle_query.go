// Package biz provides business logic for the vehicle service.
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// VehicleQueryUseCase handles vehicle query business logic.
type VehicleQueryUseCase struct {
	vehicleRepo VehicleRepo
	log         *log.Helper
}

// NewVehicleQueryUseCase creates a new VehicleQueryUseCase.
func NewVehicleQueryUseCase(vehicleRepo VehicleRepo, logger log.Logger) *VehicleQueryUseCase {
	return &VehicleQueryUseCase{
		vehicleRepo: vehicleRepo,
		log:         log.NewHelper(logger),
	}
}

// GetVehicleInfo retrieves vehicle information.
func (uc *VehicleQueryUseCase) GetVehicleInfo(ctx context.Context, plateNumber string) (*v1.VehicleInfo, error) {
	if plateNumber == "" {
		return nil, fmt.Errorf("plate number is required")
	}

	vehicle, err := uc.vehicleRepo.GetVehicleByPlate(ctx, plateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}
	if vehicle == nil {
		return nil, fmt.Errorf("vehicle not found: %s", plateNumber)
	}

	var monthlyValidUntil string
	if vehicle.MonthlyValidUntil != nil {
		monthlyValidUntil = vehicle.MonthlyValidUntil.Format(time.RFC3339)
	}

	return &v1.VehicleInfo{
		PlateNumber:       vehicle.PlateNumber,
		VehicleType:       vehicle.VehicleType,
		OwnerName:         vehicle.OwnerName,
		OwnerPhone:        vehicle.OwnerPhone,
		MonthlyValidUntil: monthlyValidUntil,
	}, nil
}

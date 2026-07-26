// Package biz provides business logic for the vehicle service.
package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/xuanyiying/smart-park/internal/vehicle/device"
)

// ProviderSet is the provider set for biz layer.
var ProviderSet = wire.NewSet(
	NewEntryExitUseCase,
	NewDeviceUseCase,
	NewManufacturerUseCase,
	NewVehicleQueryUseCase,
	NewCommandUseCase,
	NewRecordQueryUseCase,
	NewLogger,
	device.NewAdapterFactory,
)

// NewLogger creates a new logger helper.
func NewLogger(logger log.Logger) *log.Helper {
	return log.NewHelper(logger)
}

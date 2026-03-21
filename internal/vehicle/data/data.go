// Package data provides data access layer for the vehicle service.
package data

import (
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/ent"
)

// ProviderSet is the provider set for data layer.
var ProviderSet = wire.NewSet(
	NewData,
	NewVehicleRepo,
	NewBillingRuleRepo,
)

// Data holds shared data layer dependencies.
type Data struct {
	DB *ent.Client
}

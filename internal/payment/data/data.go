// Package data provides data access layer for the payment service.
package data

import (
	"github.com/google/wire"
)

// ProviderSet is the provider set for data layer.
var ProviderSet = wire.NewSet(
	NewData,
	NewOrderRepo,
)

// Package service provides gRPC service implementation for the billing service.
package service

import (
	"github.com/google/wire"
)

// ProviderSet is the provider set for service layer.
var ProviderSet = wire.NewSet(
	NewBillingService,
)

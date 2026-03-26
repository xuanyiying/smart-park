// Package client provides clients for external services.
package client

import (
	"github.com/google/wire"
	"github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
)

// ProviderSet is the provider set for clients.
var ProviderSet = wire.NewSet(
	billing.NewClient,
)

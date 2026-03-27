// Package data provides data access layer for the analytics service.
package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is the provider set for data layer.
var ProviderSet = wire.NewSet(
	NewData,
	NewAnalyticsRepo,
)

// Data wraps database connection.
type Data struct {
	// db *ent.Client
	log *log.Helper
}

// NewData creates a new Data instance.
func NewData(logger log.Logger) (*Data, func(), error) {
	d := &Data{
		log: log.NewHelper(logger),
	}

	cleanup := func() {
		// Close database connection if needed
	}

	return d, cleanup, nil
}

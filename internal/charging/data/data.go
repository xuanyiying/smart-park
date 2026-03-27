// Package data provides data access layer for the charging service.
package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/internal/charging/data/ent"
)

// ProviderSet is the provider set for data layer.
var ProviderSet = wire.NewSet(
	NewData,
	NewChargingRepo,
)

// Data wraps database connection.
type Data struct {
	db  *ent.Client
	log *log.Helper
}

// NewData creates a new Data instance.
func NewData(db *ent.Client, logger log.Logger) (*Data, func(), error) {
	d := &Data{
		db:  db,
		log: log.NewHelper(logger),
	}

	cleanup := func() {
		if err := d.db.Close(); err != nil {
			d.log.Errorf("failed to close database: %v", err)
		}
	}

	return d, cleanup, nil
}

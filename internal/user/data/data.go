package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/internal/user/data/ent"
)

// ProviderSet is the provider set for data layer.
var ProviderSet = wire.NewSet(
	NewData,
	NewUserRepo,
)

// Data holds the data layer dependencies.
type Data struct {
	db  *ent.Client
	log *log.Helper
}

// NewData creates a new Data.
func NewData(db *ent.Client, logger log.Logger) (*Data, func(), error) {
	logHelper := log.NewHelper(logger)
	cleanup := func() {
		logHelper.Info("closing the data resources")
	}
	return &Data{
		db:  db,
		log: logHelper,
	}, cleanup, nil
}

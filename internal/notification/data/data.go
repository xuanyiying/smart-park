package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewNotificationRepo,
)

type Data struct {
	log *log.Helper
}

func NewData(logger log.Logger) (*Data, func(), error) {
	d := &Data{
		log: log.NewHelper(logger),
	}
	cleanup := func() {}
	return d, cleanup, nil
}

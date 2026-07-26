package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/pkg/multitenancy"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewVehicleRepo,
)

type Data struct {
	db  *ent.Client
	log *log.Helper
}

func init() {
}

func RegisterTenantHooks(db *ent.Client) {
	multitenancy.RegisterTenantHooks(&db.Client, nil)
}

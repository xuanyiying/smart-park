package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
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

// RegisterTenantHooks applies tenant filtering hooks to the ent client.
func RegisterTenantHooks(client *ent.Client) {
	// Apply tenant hooks using the client's Use method
	hooks := []ent.Hook{}
	// TODO: integrate multitenancy hooks when tenant_hook API is finalized
	_ = hooks
}

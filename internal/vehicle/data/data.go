package data

import (
	"database/sql"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewVehicleRepo,
)

type Data struct {
	db *ent.Client
	// raw exposes the same database as a plain *sql.DB for cross-domain reads the
	// vehicle ent schema does not model (e.g. the parking_lots capacity column owned
	// by the admin/analytics services). Read-only by convention.
	raw *sql.DB
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

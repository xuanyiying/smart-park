package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/internal/admin/data/ent"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewAdminRepo,
)

type Data struct {
	db  *ent.Client
	log *log.Helper
}

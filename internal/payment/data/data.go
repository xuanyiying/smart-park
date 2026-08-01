package data

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"

	"github.com/xuanyiying/smart-park/internal/payment/data/ent"
	"github.com/xuanyiying/smart-park/pkg/database"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewOrderRepo,
	NewReconciliationRepo,
)

type Data struct {
	db  *ent.Client
	log *log.Helper
	txm *database.EntTransactionManager
}

func NewData(db *ent.Client, logger log.Logger) (*Data, func(), error) {
	d := &Data{
		db:  db,
		log: log.NewHelper(logger),
	}
	d.txm = database.NewEntTransactionManager(func(ctx context.Context) (database.CommitRollbacker, error) {
		return db.Tx(ctx)
	}, logger)

	cleanup := func() {
		if err := d.db.Close(); err != nil {
			d.log.Errorf("failed to close database: %v", err)
		}
	}

	return d, cleanup, nil
}

// Package data provides data access layer for the analytics service.
package data

import (
	"database/sql"
	"fmt"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	_ "github.com/lib/pq"
)

// ProviderSet is the provider set for data layer.
var ProviderSet = wire.NewSet(
	NewDatabase,
	NewData,
	NewAnalyticsRepo,
)

// Data wraps database connection.
type Data struct {
	db  *sql.DB
	log *log.Helper
}

// NewDatabase creates a new database connection.
func NewDatabase(source string) (*sql.DB, error) {
	db, err := sql.Open("postgres", source)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// NewData creates a new Data instance.
func NewData(db *sql.DB, logger log.Logger) (*Data, error) {
	d := &Data{
		db:  db,
		log: log.NewHelper(logger),
	}

	return d, nil
}

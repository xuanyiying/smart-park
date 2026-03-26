package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Driver       string
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLife  int
}

type DB struct {
	*sql.DB
}

func New(cfg *Config) (*DB, error) {
	if cfg.Driver == "" {
		cfg.Driver = "postgres"
	}
	if cfg.MaxOpenConns <= 0 {
		cfg.MaxOpenConns = 100
	}
	if cfg.MaxIdleConns <= 0 {
		cfg.MaxIdleConns = 10
	}
	if cfg.ConnMaxLife <= 0 {
		cfg.ConnMaxLife = 3600
	}

	db, err := sql.Open(cfg.Driver, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLife) * time.Second)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{DB: db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) Ping(ctx context.Context) error {
	return db.DB.PingContext(ctx)
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRowContext(ctx, query, args...)
}

func (db *DB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.DB.ExecContext(ctx, query, args...)
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*Tx, error) {
	tx, err := db.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &Tx{Tx: tx}, nil
}

func (db *DB) Begin(ctx context.Context) (*Tx, error) {
	return db.BeginTx(ctx, nil)
}

type Tx struct {
	*sql.Tx
}

func (tx *Tx) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return tx.Tx.QueryContext(ctx, query, args...)
}

func (tx *Tx) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return tx.Tx.QueryRowContext(ctx, query, args...)
}

func (tx *Tx) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return tx.Tx.ExecContext(ctx, query, args...)
}

func (tx *Tx) Commit() error {
	return tx.Tx.Commit()
}

func (tx *Tx) Rollback() error {
	return tx.Tx.Rollback()
}

type Transaction func(ctx context.Context, tx *Tx) error

func (db *DB) WithTransaction(ctx context.Context, fn Transaction) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("transaction failed: %v, rollback failed: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

type NullString struct {
	sql.NullString
}

func (ns NullString) MarshalJSON() ([]byte, error) {
	if ns.Valid {
		return []byte(`"` + ns.String + `"`), nil
	}
	return []byte(`null`), nil
}

type NullInt64 struct {
	sql.NullInt64
}

func (ni NullInt64) MarshalJSON() ([]byte, error) {
	if ni.Valid {
		return []byte(fmt.Sprintf(`%d`, ni.Int64)), nil
	}
	return []byte(`null`), nil
}

type NullFloat64 struct {
	sql.NullFloat64
}

func (nf NullFloat64) MarshalJSON() ([]byte, error) {
	if nf.Valid {
		return []byte(fmt.Sprintf(`%f`, nf.Float64)), nil
	}
	return []byte(`null`), nil
}

type NullBool struct {
	sql.NullBool
}

func (nb NullBool) MarshalJSON() ([]byte, error) {
	if nb.Valid {
		return []byte(fmt.Sprintf(`%t`, nb.Bool)), nil
	}
	return []byte(`null`), nil
}

type NullTime struct {
	sql.NullTime
}

func (nt NullTime) MarshalJSON() ([]byte, error) {
	if nt.Valid {
		return []byte(`"` + nt.Time.Format(time.RFC3339) + `"`), nil
	}
	return []byte(`null`), nil
}

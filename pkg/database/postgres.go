package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type Config struct {
	Primary struct {
		Source string
	}
	Replica struct {
		Source string
	}
}

type DBManager struct {
	primary   *sql.DB
	replicas  []*sql.DB
	random    *rand.Rand
	randomMux sync.Mutex
}

func NewDBManager(cfg *Config) (*DBManager, error) {
	// Connect to primary database
	primary, err := sql.Open("postgres", cfg.Primary.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to primary database: %w", err)
	}

	// Test primary connection
	if err := primary.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping primary database: %w", err)
	}

	// Connect to replica database
	var replicas []*sql.DB
	replica, err := sql.Open("postgres", cfg.Replica.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to replica database: %w", err)
	}

	// Test replica connection
	if err := replica.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping replica database: %w", err)
	}

	replicas = append(replicas, replica)

	return &DBManager{
		primary:  primary,
		replicas: replicas,
		random:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}, nil
}

func (m *DBManager) Primary() *sql.DB {
	return m.primary
}

func (m *DBManager) Replica() *sql.DB {
	m.randomMux.Lock()
	defer m.randomMux.Unlock()

	if len(m.replicas) == 0 {
		return m.primary
	}

	index := m.random.Intn(len(m.replicas))
	return m.replicas[index]
}

func (m *DBManager) Close() error {
	if err := m.primary.Close(); err != nil {
		return err
	}

	for _, replica := range m.replicas {
		if err := replica.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (m *DBManager) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return m.primary.BeginTx(ctx, opts)
}

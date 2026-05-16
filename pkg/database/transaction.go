package database

import (
	"context"
	"errors"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/lib/pq"
)

type txCtxKey struct{}

type CommitRollbacker interface {
	Commit() error
	Rollback() error
}

type TransactionManager interface {
	WithTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type BeginTxFunc func(ctx context.Context) (CommitRollbacker, error)

type TxOption func(*EntTransactionManager)

func WithMaxRetries(n int) TxOption {
	return func(tm *EntTransactionManager) {
		tm.maxRetries = n
	}
}

func WithInitialBackoff(d time.Duration) TxOption {
	return func(tm *EntTransactionManager) {
		tm.initialBackoff = d
	}
}

func WithMaxBackoff(d time.Duration) TxOption {
	return func(tm *EntTransactionManager) {
		tm.maxBackoff = d
	}
}

func WithTimeout(d time.Duration) TxOption {
	return func(tm *EntTransactionManager) {
		tm.timeout = d
	}
}

type EntTransactionManager struct {
	beginTx        BeginTxFunc
	logger         *log.Helper
	maxRetries     int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	timeout        time.Duration
}

func NewEntTransactionManager(beginTx BeginTxFunc, logger log.Logger, opts ...TxOption) *EntTransactionManager {
	tm := &EntTransactionManager{
		beginTx:        beginTx,
		logger:         log.NewHelper(logger),
		maxRetries:     3,
		initialBackoff: 100 * time.Millisecond,
		maxBackoff:     5 * time.Second,
		timeout:        30 * time.Second,
	}
	for _, opt := range opts {
		opt(tm)
	}
	return tm
}

func (tm *EntTransactionManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	var lastErr error

	for attempt := 0; attempt <= tm.maxRetries; attempt++ {
		if attempt > 0 {
			backoff := tm.initialBackoff * time.Duration(1<<uint(attempt-1))
			if backoff > tm.maxBackoff {
				backoff = tm.maxBackoff
			}
			tm.logger.WithContext(ctx).Warnf("retrying transaction (attempt %d/%d), backoff: %v, last error: %v",
				attempt, tm.maxRetries, backoff, lastErr)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		err := tm.executeTx(ctx, fn)
		if err == nil {
			return nil
		}

		if !isRetryableError(err) {
			return err
		}

		lastErr = err
	}

	return lastErr
}

func (tm *EntTransactionManager) executeTx(ctx context.Context, fn func(ctx context.Context) error) error {
	txCtx, cancel := context.WithTimeout(ctx, tm.timeout)
	defer cancel()

	tx, err := tm.beginTx(txCtx)
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()

	fnCtx := context.WithValue(txCtx, txCtxKey{}, tx)

	if err := fn(fnCtx); err != nil {
		if rerr := tx.Rollback(); rerr != nil {
			return rerr
		}
		return err
	}

	return tx.Commit()
}

func TxFromCtx(ctx context.Context) interface{} {
	return ctx.Value(txCtxKey{})
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		switch pqErr.Code {
		case "40001", "40P01":
			return true
		}
	}

	return false
}

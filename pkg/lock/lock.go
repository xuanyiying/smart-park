// Package lock provides distributed lock functionality using Redis.
package lock

import (
	"context"
	"time"
)

// LockRepo defines the interface for distributed lock operations.
type LockRepo interface {
	AcquireLock(ctx context.Context, lockKey string, owner string, ttl time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, lockKey string, owner string) error
	ExtendLock(ctx context.Context, lockKey string, owner string, ttl time.Duration) error
	GetLockOwner(ctx context.Context, lockKey string) (string, error)
	IsLocked(ctx context.Context, lockKey string) (bool, error)
	TryLockWithRetry(ctx context.Context, lockKey string, owner string, ttl time.Duration, maxRetries int, retryInterval time.Duration) (bool, error)
}

// GenerateLockKey generates a lock key for a specific operation and resource.
func GenerateLockKey(operation, resource string) string {
	return operation + ":" + resource
}

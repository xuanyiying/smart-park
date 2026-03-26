package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

// RedisLockRepo implements LockRepo using Redis.
type RedisLockRepo struct {
	client *redis.Client
	log    *log.Helper
	prefix string
}

// NewRedisLockRepo creates a new RedisLockRepo.
func NewRedisLockRepo(client *redis.Client, logger log.Logger, prefix string) *RedisLockRepo {
	return &RedisLockRepo{
		client: client,
		log:    log.NewHelper(logger),
		prefix: prefix,
	}
}

// AcquireLock acquires a distributed lock using Redis SETNX with TTL.
func (r *RedisLockRepo) AcquireLock(ctx context.Context, lockKey string, owner string, ttl time.Duration) (bool, error) {
	key := r.formatKey(lockKey)

	// Use SET with NX and PX options for atomic lock acquisition
	success, err := r.client.SetNX(ctx, key, owner, ttl).Result()
	if err != nil {
		r.log.WithContext(ctx).Errorf("failed to acquire lock %s: %v", key, err)
		return false, fmt.Errorf("failed to acquire lock: %w", err)
	}

	if success {
		r.log.WithContext(ctx).Debugf("lock acquired - Key: %s, Owner: %s, TTL: %v", key, owner, ttl)
	} else {
		r.log.WithContext(ctx).Debugf("lock not available - Key: %s, Owner: %s", key, owner)
	}

	return success, nil
}

// ReleaseLock releases a distributed lock.
func (r *RedisLockRepo) ReleaseLock(ctx context.Context, lockKey string, owner string) error {
	key := r.formatKey(lockKey)

	// Verify that we own the lock before releasing
	currentOwner, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			r.log.WithContext(ctx).Warnf("lock already expired or not exists - Key: %s", key)
			return nil
		}
		r.log.WithContext(ctx).Errorf("failed to get lock owner - Key: %s: %v", key, err)
		return fmt.Errorf("failed to verify lock owner: %w", err)
	}

	if currentOwner != owner {
		r.log.WithContext(ctx).Warnf("cannot release lock - not owner - Key: %s, CurrentOwner: %s, RequestedOwner: %s",
			key, currentOwner, owner)
		return fmt.Errorf("cannot release lock: not the owner")
	}

	// Delete the lock
	if err := r.client.Del(ctx, key).Err(); err != nil {
		r.log.WithContext(ctx).Errorf("failed to delete lock - Key: %s: %v", key, err)
		return fmt.Errorf("failed to release lock: %w", err)
	}

	r.log.WithContext(ctx).Debugf("lock released - Key: %s, Owner: %s", key, owner)
	return nil
}

// ExtendLock extends the TTL of an existing lock.
func (r *RedisLockRepo) ExtendLock(ctx context.Context, lockKey string, owner string, ttl time.Duration) error {
	key := r.formatKey(lockKey)

	// Verify ownership before extending
	currentOwner, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("lock does not exist or has expired")
		}
		return fmt.Errorf("failed to get lock owner: %w", err)
	}

	if currentOwner != owner {
		return fmt.Errorf("cannot extend lock: not the owner")
	}

	// Extend the TTL
	if err := r.client.Expire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("failed to extend lock: %w", err)
	}

	r.log.WithContext(ctx).Debugf("lock extended - Key: %s, Owner: %s, NewTTL: %v", key, owner, ttl)
	return nil
}

// GetLockOwner returns the current owner of a lock.
func (r *RedisLockRepo) GetLockOwner(ctx context.Context, lockKey string) (string, error) {
	key := r.formatKey(lockKey)

	owner, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}

	return owner, nil
}

// IsLocked checks if a lock is currently held.
func (r *RedisLockRepo) IsLocked(ctx context.Context, lockKey string) (bool, error) {
	key := r.formatKey(lockKey)

	exists, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return exists > 0, nil
}

// TryLockWithRetry attempts to acquire a lock with retry mechanism.
func (r *RedisLockRepo) TryLockWithRetry(ctx context.Context, lockKey string, owner string, ttl time.Duration,
	maxRetries int, retryInterval time.Duration) (bool, error) {

	var lastErr error

	for i := 0; i < maxRetries; i++ {
		success, err := r.AcquireLock(ctx, lockKey, owner, ttl)
		if err != nil {
			return false, err
		}

		if success {
			return true, nil
		}

		lastErr = fmt.Errorf("lock is held by another process")

		// Wait before retrying
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(retryInterval):
			continue
		}
	}

	return false, lastErr
}

func (r *RedisLockRepo) formatKey(lockKey string) string {
	return fmt.Sprintf("%s:lock:%s", r.prefix, lockKey)
}

// GenerateUniqueOwner generates a unique owner identifier.
func GenerateUniqueOwner() string {
	return uuid.New().String()
}

// DistributedLock provides a distributed lock helper.
type DistributedLock struct {
	repo   LockRepo
	lockID string
	owner  string
	ttl    time.Duration
}

// NewDistributedLock creates a new distributed lock instance.
func NewDistributedLock(repo LockRepo, lockKey string, owner string) *DistributedLock {
	return &DistributedLock{
		repo:  repo,
		lockID: lockKey,
		owner: owner,
		ttl:   10 * time.Second,
	}
}

// WithLock executes a function within a distributed lock.
func (dl *DistributedLock) WithLock(ctx context.Context, fn func() error) error {
	acquired, err := dl.repo.AcquireLock(ctx, dl.lockID, dl.owner, dl.ttl)
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("failed to acquire lock: lock is held by another process")
	}

	defer func() {
		if err := dl.repo.ReleaseLock(ctx, dl.lockID, dl.owner); err != nil {
			dl.repo.(interface{ GetLogHelper() *log.Helper }).GetLogHelper().WithContext(ctx).Warnf("warning: failed to release lock: %v", err)
		}
	}()

	return fn()
}

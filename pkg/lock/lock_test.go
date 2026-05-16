package lock

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

type mockLockRepo struct {
	mu       sync.Mutex
	locks    map[string]string
	acquire  bool
	acqErr   error
	relErr   error
	extErr   error
}

func newMockLockRepo(acquire bool) *mockLockRepo {
	return &mockLockRepo{
		locks:   make(map[string]string),
		acquire: acquire,
	}
}

func (m *mockLockRepo) AcquireLock(ctx context.Context, lockKey string, owner string, ttl time.Duration) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.acqErr != nil {
		return false, m.acqErr
	}
	if existing, ok := m.locks[lockKey]; ok {
		if existing == owner {
			return true, nil
		}
		return false, nil
	}
	if m.acquire {
		m.locks[lockKey] = owner
	}
	return m.acquire, nil
}

func (m *mockLockRepo) ReleaseLock(ctx context.Context, lockKey string, owner string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.relErr != nil {
		return m.relErr
	}
	if existing, ok := m.locks[lockKey]; ok && existing == owner {
		delete(m.locks, lockKey)
		return nil
	}
	return fmt.Errorf("cannot release lock: not the owner")
}

func (m *mockLockRepo) ExtendLock(ctx context.Context, lockKey string, owner string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.extErr != nil {
		return m.extErr
	}
	if existing, ok := m.locks[lockKey]; ok && existing == owner {
		return nil
	}
	return fmt.Errorf("cannot extend lock: not the owner")
}

func (m *mockLockRepo) GetLockOwner(ctx context.Context, lockKey string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.locks[lockKey], nil
}

func (m *mockLockRepo) IsLocked(ctx context.Context, lockKey string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.locks[lockKey]
	return ok, nil
}

func (m *mockLockRepo) TryLockWithRetry(ctx context.Context, lockKey string, owner string, ttl time.Duration, maxRetries int, retryInterval time.Duration) (bool, error) {
	for i := 0; i < maxRetries; i++ {
		success, err := m.AcquireLock(ctx, lockKey, owner, ttl)
		if err != nil {
			return false, err
		}
		if success {
			return true, nil
		}
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-time.After(retryInterval):
			continue
		}
	}
	return false, fmt.Errorf("lock is held by another process")
}

func TestGenerateLockKey(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		resource  string
		expected  string
	}{
		{
			name:      "entry operation",
			operation: "entry",
			resource:  "plate-京A12345",
			expected:  "entry:plate-京A12345",
		},
		{
			name:      "exit operation",
			operation: "exit",
			resource:  "plate-京B67890",
			expected:  "exit:plate-京B67890",
		},
		{
			name:      "payment operation",
			operation: "payment",
			resource:  "record-abc123",
			expected:  "payment:record-abc123",
		},
		{
			name:      "empty operation",
			operation: "",
			resource:  "resource1",
			expected:  ":resource1",
		},
		{
			name:      "empty resource",
			operation: "op1",
			resource:  "",
			expected:  "op1:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateLockKey(tt.operation, tt.resource)
			if got != tt.expected {
				t.Errorf("GenerateLockKey() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDistributedLock_WithLock(t *testing.T) {
	repo := newMockLockRepo(true)
	dl := NewDistributedLock(repo, "test:resource1", "owner1", nil)

	executed := false
	err := dl.WithLock(context.Background(), func() error {
		executed = true
		return nil
	})

	if err != nil {
		t.Errorf("WithLock() unexpected error: %v", err)
	}
	if !executed {
		t.Error("WithLock() function was not executed")
	}
}

func TestDistributedLock_WithLock_AcquireFails(t *testing.T) {
	repo := newMockLockRepo(false)
	dl := NewDistributedLock(repo, "test:resource1", "owner1", nil)

	err := dl.WithLock(context.Background(), func() error {
		return nil
	})

	if err == nil {
		t.Error("WithLock() expected error when lock cannot be acquired")
	}
}

func TestDistributedLock_WithLock_FnError(t *testing.T) {
	repo := newMockLockRepo(true)
	dl := NewDistributedLock(repo, "test:resource1", "owner1", nil)

	err := dl.WithLock(context.Background(), func() error {
		return fmt.Errorf("business error")
	})

	if err == nil {
		t.Error("WithLock() expected error from function")
	}
	if err.Error() != "business error" {
		t.Errorf("WithLock() error = %v, want business error", err)
	}
}

func TestGenerateUniqueOwner(t *testing.T) {
	owner1 := GenerateUniqueOwner()
	owner2 := GenerateUniqueOwner()

	if owner1 == "" {
		t.Error("GenerateUniqueOwner() returned empty string")
	}
	if owner2 == "" {
		t.Error("GenerateUniqueOwner() returned empty string")
	}
	if owner1 == owner2 {
		t.Error("GenerateUniqueOwner() returned duplicate values")
	}
}

func TestMockLockRepo_AcquireAndRelease(t *testing.T) {
	repo := newMockLockRepo(true)
	ctx := context.Background()

	acquired, err := repo.AcquireLock(ctx, "key1", "owner1", 10*time.Second)
	if err != nil || !acquired {
		t.Fatalf("AcquireLock() failed: acquired=%v, err=%v", acquired, err)
	}

	locked, _ := repo.IsLocked(ctx, "key1")
	if !locked {
		t.Error("key1 should be locked after AcquireLock")
	}

	owner, _ := repo.GetLockOwner(ctx, "key1")
	if owner != "owner1" {
		t.Errorf("owner = %s, want owner1", owner)
	}

	if err := repo.ReleaseLock(ctx, "key1", "owner1"); err != nil {
		t.Errorf("ReleaseLock() error: %v", err)
	}

	locked, _ = repo.IsLocked(ctx, "key1")
	if locked {
		t.Error("key1 should not be locked after ReleaseLock")
	}
}

func TestMockLockRepo_AcquireTwice(t *testing.T) {
	repo := newMockLockRepo(true)
	ctx := context.Background()

	acquired1, _ := repo.AcquireLock(ctx, "key1", "owner1", 10*time.Second)
	if !acquired1 {
		t.Error("first acquire should succeed")
	}

	acquired2, _ := repo.AcquireLock(ctx, "key1", "owner2", 10*time.Second)
	if acquired2 {
		t.Error("second acquire with different owner should fail")
	}
}

func TestMockLockRepo_ReleaseWrongOwner(t *testing.T) {
	repo := newMockLockRepo(true)
	ctx := context.Background()

	repo.AcquireLock(ctx, "key1", "owner1", 10*time.Second)

	err := repo.ReleaseLock(ctx, "key1", "owner2")
	if err == nil {
		t.Error("ReleaseLock with wrong owner should return error")
	}
}

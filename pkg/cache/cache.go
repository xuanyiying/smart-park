package cache

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

var (
	ErrCacheMiss    = errors.New("cache miss")
	ErrNotSupported = errors.New("operation not supported")
)

type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	GetMulti(ctx context.Context, keys []string) ([]string, error)
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error)
	Delete(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, keys ...string) (int64, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	TTL(ctx context.Context, key string) (time.Duration, error)
	Incr(ctx context.Context, key string) (int64, error)
	Decr(ctx context.Context, key string) (int64, error)
	IncrBy(ctx context.Context, key string, value int64) (int64, error)
	Close() error
	Ping(ctx context.Context) error
}

type RedisCache struct {
	client *redis.Client
}

func NewRedisCache(addr, password string, db int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	return &RedisCache{client: client}
}

func NewRedisCacheWithPool(addr, password string, db, poolSize, minIdleConn int) *RedisCache {
	client := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     poolSize,
		MinIdleConns: minIdleConn,
	})
	return &RedisCache{client: client}
}

func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return c.client.Set(ctx, key, value, expiration).Err()
}

func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
	result, err := c.client.Get(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", ErrCacheMiss
	}
	return result, err
}

func (c *RedisCache) GetMulti(ctx context.Context, keys []string) ([]string, error) {
	if len(keys) == 0 {
		return nil, nil
	}
	results, err := c.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
	}

	values := make([]string, 0, len(results))
	for _, r := range results {
		if r == nil {
			values = append(values, "")
		} else {
			values = append(values, r.(string))
		}
	}
	return values, nil
}

func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	return c.client.SetNX(ctx, key, value, expiration).Result()
}

func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return c.client.Del(ctx, keys...).Err()
}

func (c *RedisCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return c.client.Exists(ctx, keys...).Result()
}

func (c *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, key, expiration).Err()
}

func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, key).Result()
}

func (c *RedisCache) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, key).Result()
}

func (c *RedisCache) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, key).Result()
}

func (c *RedisCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, key, value).Result()
}

func (c *RedisCache) Close() error {
	return c.client.Close()
}

func (c *RedisCache) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

func (c *RedisCache) Client() *redis.Client {
	return c.client
}

var _ Cache = (*RedisCache)(nil)

type MemCache struct {
	data map[string]cacheItem
	mu   sync.RWMutex
}

type cacheItem struct {
	value      interface{}
	expiration time.Time
}

func NewMemCache() *MemCache {
	return &MemCache{
		data: make(map[string]cacheItem),
	}
}

func (c *MemCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
	return nil
}

func (c *MemCache) Get(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.data[key]
	if !ok {
		return "", ErrCacheMiss
	}
	if time.Now().After(item.expiration) {
		return "", ErrCacheMiss
	}
	return item.value.(string), nil
}

func (c *MemCache) GetMulti(ctx context.Context, keys []string) ([]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := time.Now()
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		item, ok := c.data[key]
		if !ok || now.After(item.expiration) {
			values = append(values, "")
			continue
		}
		values = append(values, item.value.(string))
	}
	return values, nil
}

func (c *MemCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.data[key]
	if ok {
		item := c.data[key]
		if time.Now().Before(item.expiration) {
			return false, nil
		}
	}
	c.data[key] = cacheItem{
		value:      value,
		expiration: time.Now().Add(expiration),
	}
	return true, nil
}

func (c *MemCache) Delete(ctx context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, key := range keys {
		delete(c.data, key)
	}
	return nil
}

func (c *MemCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := time.Now()
	var count int64
	for _, key := range keys {
		item, ok := c.data[key]
		if ok && !now.After(item.expiration) {
			count++
		}
	}
	return count, nil
}

func (c *MemCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	item, ok := c.data[key]
	if !ok {
		return ErrCacheMiss
	}
	item.expiration = time.Now().Add(expiration)
	c.data[key] = item
	return nil
}

func (c *MemCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	item, ok := c.data[key]
	if !ok {
		return -1, ErrCacheMiss
	}
	ttl := time.Until(item.expiration)
	if ttl < 0 {
		return -1, ErrCacheMiss
	}
	return ttl, nil
}

func (c *MemCache) Incr(ctx context.Context, key string) (int64, error) {
	return c.incrBy(ctx, key, 1)
}

func (c *MemCache) Decr(ctx context.Context, key string) (int64, error) {
	return c.incrBy(ctx, key, -1)
}

func (c *MemCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.incrBy(ctx, key, value)
}

func (c *MemCache) incrBy(ctx context.Context, key string, value int64) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.data[key]
	if !ok {
		item = cacheItem{
			value:      int64(0),
			expiration: time.Now().Add(time.Hour * 24 * 365),
		}
	}

	curr, ok := item.value.(int64)
	if !ok {
		curr = 0
	}

	newVal := curr + value
	item.value = newVal
	c.data[key] = item
	return newVal, nil
}

func (c *MemCache) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]cacheItem)
	return nil
}

func (c *MemCache) Ping(ctx context.Context) error {
	return nil
}

func (c *MemCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

var _ Cache = (*MemCache)(nil)

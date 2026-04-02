package cache

import (
	"context"
	"time"
)

// MultiLevelCache 多级缓存实现（内存 + Redis）
type MultiLevelCache struct {
	local  *MemCache
	remote *RedisCache
}

// NewMultiLevelCache 创建多级缓存实例
func NewMultiLevelCache(redisAddr, redisPassword string, redisDB int) *MultiLevelCache {
	return &MultiLevelCache{
		local:  NewMemCache(),
		remote: NewRedisCache(redisAddr, redisPassword, redisDB),
	}
}

// Set 设置缓存（同时设置本地和远程）
func (c *MultiLevelCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	// 先设置远程缓存
	if err := c.remote.Set(ctx, key, value, expiration); err != nil {
		return err
	}
	// 再设置本地缓存，本地缓存过期时间可以比远程短
	localExpiration := expiration
	if localExpiration > 5*time.Minute {
		localExpiration = 5 * time.Minute
	}
	return c.local.Set(ctx, key, value, localExpiration)
}

// Get 获取缓存（先从本地获取，本地没有再从远程获取）
func (c *MultiLevelCache) Get(ctx context.Context, key string) (string, error) {
	// 先从本地缓存获取
	val, err := c.local.Get(ctx, key)
	if err == nil {
		return val, nil
	}

	// 本地缓存未命中，从远程缓存获取
	val, err = c.remote.Get(ctx, key)
	if err != nil {
		return "", err
	}

	// 从远程获取到后，更新本地缓存
	c.local.Set(ctx, key, val, 5*time.Minute)
	return val, nil
}

// GetMulti 批量获取缓存
func (c *MultiLevelCache) GetMulti(ctx context.Context, keys []string) ([]string, error) {
	// 先从本地缓存获取所有键
	localValues, err := c.local.GetMulti(ctx, keys)
	if err != nil {
		return nil, err
	}

	// 收集本地未命中的键
	missedKeys := make([]string, 0)
	missedIndices := make([]int, 0)
	for i, val := range localValues {
		if val == "" {
			missedKeys = append(missedKeys, keys[i])
			missedIndices = append(missedIndices, i)
		}
	}

	// 如果有未命中的键，从远程缓存获取
	if len(missedKeys) > 0 {
		remoteValues, err := c.remote.GetMulti(ctx, missedKeys)
		if err != nil {
			return nil, err
		}

		// 更新本地缓存并填充结果
		for i, idx := range missedIndices {
			if remoteValues[i] != "" {
				localValues[idx] = remoteValues[i]
				c.local.Set(ctx, keys[idx], remoteValues[i], 5*time.Minute)
			}
		}
	}

	return localValues, nil
}

// SetNX 设置缓存（仅当键不存在时）
func (c *MultiLevelCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	// 先在远程设置
	success, err := c.remote.SetNX(ctx, key, value, expiration)
	if err != nil || !success {
		return success, err
	}

	// 远程设置成功后，设置本地缓存
	localExpiration := expiration
	if localExpiration > 5*time.Minute {
		localExpiration = 5 * time.Minute
	}
	_ = c.local.Set(ctx, key, value, localExpiration)
	return true, nil
}

// Delete 删除缓存（同时删除本地和远程）
func (c *MultiLevelCache) Delete(ctx context.Context, keys ...string) error {
	// 先删除本地缓存
	if err := c.local.Delete(ctx, keys...); err != nil {
		return err
	}
	// 再删除远程缓存
	return c.remote.Delete(ctx, keys...)
}

// Exists 检查键是否存在（仅检查远程缓存）
func (c *MultiLevelCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	return c.remote.Exists(ctx, keys...)
}

// Expire 设置键的过期时间
func (c *MultiLevelCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	// 先设置远程缓存的过期时间
	if err := c.remote.Expire(ctx, key, expiration); err != nil {
		return err
	}
	// 再设置本地缓存的过期时间
	return c.local.Expire(ctx, key, expiration)
}

// TTL 获取键的剩余过期时间
func (c *MultiLevelCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.remote.TTL(ctx, key)
}

// Incr 自增
func (c *MultiLevelCache) Incr(ctx context.Context, key string) (int64, error) {
	// 先在远程自增
	val, err := c.remote.Incr(ctx, key)
	if err != nil {
		return 0, err
	}

	// 更新本地缓存
	_ = c.local.Set(ctx, key, val, 5*time.Minute)
	return val, nil
}

// Decr 自减
func (c *MultiLevelCache) Decr(ctx context.Context, key string) (int64, error) {
	// 先在远程自减
	val, err := c.remote.Decr(ctx, key)
	if err != nil {
		return 0, err
	}

	// 更新本地缓存
	_ = c.local.Set(ctx, key, val, 5*time.Minute)
	return val, nil
}

// IncrBy 按指定值自增
func (c *MultiLevelCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	// 先在远程自增
	val, err := c.remote.IncrBy(ctx, key, value)
	if err != nil {
		return 0, err
	}

	// 更新本地缓存
	_ = c.local.Set(ctx, key, val, 5*time.Minute)
	return val, nil
}

// Close 关闭缓存
func (c *MultiLevelCache) Close() error {
	_ = c.local.Close()
	return c.remote.Close()
}

// Ping 检查缓存是否可用
func (c *MultiLevelCache) Ping(ctx context.Context) error {
	return c.remote.Ping(ctx)
}

// LocalCache 获取本地缓存实例
func (c *MultiLevelCache) LocalCache() *MemCache {
	return c.local
}

// RemoteCache 获取远程缓存实例
func (c *MultiLevelCache) RemoteCache() *RedisCache {
	return c.remote
}

var _ Cache = (*MultiLevelCache)(nil)
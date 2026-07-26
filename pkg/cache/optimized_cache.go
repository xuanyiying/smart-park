package cache

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// OptimizedMultiLevelCache 优化的多级缓存实现
type OptimizedMultiLevelCache struct {
	local      *MemCache
	remote     *RedisCache
	hotLoader  *HotDataLoader
	metrics    *CacheMetrics
	mu         sync.RWMutex
}

// NewOptimizedMultiLevelCache 创建优化的多级缓存实例
func NewOptimizedMultiLevelCache(redisAddr, redisPassword string, redisDB int, hotLoadInterval time.Duration) *OptimizedMultiLevelCache {
	local := NewMemCache()
	remote := NewRedisCache(redisAddr, redisPassword, redisDB)
	hotLoader := NewHotDataLoader(remote, hotLoadInterval)
	metrics := NewCacheMetrics()

	cache := &OptimizedMultiLevelCache{
		local:      local,
		remote:     remote,
		hotLoader:  hotLoader,
		metrics:    metrics,
	}

	return cache
}

// Set 设置缓存（同时设置本地和远程）
func (c *OptimizedMultiLevelCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("set", time.Since(start))
	}()

	// 先设置远程缓存
	if err := c.remote.Set(ctx, key, value, expiration); err != nil {
		c.metrics.RecordError("set")
		return err
	}

	// 再设置本地缓存，本地缓存过期时间可以比远程短
	localExpiration := expiration
	if localExpiration > 5*time.Minute {
		localExpiration = 5 * time.Minute
	}

	if err := c.local.Set(ctx, key, value, localExpiration); err != nil {
		// 本地缓存设置失败不影响远程缓存
		log.Printf("local cache set failed: %v", err)
	}

	return nil
}

// Get 获取缓存（先从本地获取，本地没有再从远程获取）
func (c *OptimizedMultiLevelCache) Get(ctx context.Context, key string) (string, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("get", time.Since(start))
	}()

	// 先从本地缓存获取
	val, err := c.local.Get(ctx, key)
	if err == nil {
		c.metrics.RecordHit("local")
		return val, nil
	}

	// 本地缓存未命中，从远程缓存获取
	val, err = c.remote.Get(ctx, key)
	if err != nil {
		c.metrics.RecordMiss()
		c.metrics.RecordError("get")
		return "", err
	}

	c.metrics.RecordHit("remote")

	// 从远程获取到后，更新本地缓存
	c.local.Set(ctx, key, val, 5*time.Minute)

	return val, nil
}

// GetMulti 批量获取缓存
func (c *OptimizedMultiLevelCache) GetMulti(ctx context.Context, keys []string) ([]string, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("get_multi", time.Since(start))
	}()

	// 先从本地缓存获取所有键
	localValues, err := c.local.GetMulti(ctx, keys)
	if err != nil {
		c.metrics.RecordError("get_multi")
		return nil, err
	}

	// 收集本地未命中的键
	missedKeys := make([]string, 0)
	missedIndices := make([]int, 0)
	for i, val := range localValues {
		if val == "" {
			missedKeys = append(missedKeys, keys[i])
			missedIndices = append(missedIndices, i)
		} else {
			c.metrics.RecordHit("local")
		}
	}

	// 如果有未命中的键，从远程缓存获取
	if len(missedKeys) > 0 {
		remoteValues, err := c.remote.GetMulti(ctx, missedKeys)
		if err != nil {
			c.metrics.RecordError("get_multi")
			return nil, err
		}

		// 更新本地缓存并填充结果
		for i, idx := range missedIndices {
			if remoteValues[i] != "" {
				localValues[idx] = remoteValues[i]
				c.local.Set(ctx, keys[idx], remoteValues[i], 5*time.Minute)
				c.metrics.RecordHit("remote")
			} else {
				c.metrics.RecordMiss()
			}
		}
	}

	return localValues, nil
}

// SetNX 设置缓存（仅当键不存在时）
func (c *OptimizedMultiLevelCache) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("setnx", time.Since(start))
	}()

	// 先在远程设置
	success, err := c.remote.SetNX(ctx, key, value, expiration)
	if err != nil || !success {
		c.metrics.RecordError("setnx")
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
func (c *OptimizedMultiLevelCache) Delete(ctx context.Context, keys ...string) error {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("delete", time.Since(start))
	}()

	// 先删除本地缓存
	if err := c.local.Delete(ctx, keys...); err != nil {
		// 本地缓存删除失败不影响远程缓存
		log.Printf("local cache delete failed: %v", err)
	}

	// 再删除远程缓存
	if err := c.remote.Delete(ctx, keys...); err != nil {
		c.metrics.RecordError("delete")
		return err
	}

	return nil
}

// Exists 检查键是否存在（仅检查远程缓存）
func (c *OptimizedMultiLevelCache) Exists(ctx context.Context, keys ...string) (int64, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("exists", time.Since(start))
	}()

	result, err := c.remote.Exists(ctx, keys...)
	if err != nil {
		c.metrics.RecordError("exists")
	}

	return result, err
}

// Expire 设置键的过期时间
func (c *OptimizedMultiLevelCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("expire", time.Since(start))
	}()

	// 先设置远程缓存的过期时间
	if err := c.remote.Expire(ctx, key, expiration); err != nil {
		c.metrics.RecordError("expire")
		return err
	}

	// 再设置本地缓存的过期时间
	if err := c.local.Expire(ctx, key, expiration); err != nil {
		// 本地缓存设置失败不影响远程缓存
		log.Printf("local cache expire failed: %v", err)
	}

	return nil
}

// TTL 获取键的剩余过期时间
func (c *OptimizedMultiLevelCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("ttl", time.Since(start))
	}()

	result, err := c.remote.TTL(ctx, key)
	if err != nil {
		c.metrics.RecordError("ttl")
	}

	return result, err
}

// Incr 自增
func (c *OptimizedMultiLevelCache) Incr(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("incr", time.Since(start))
	}()

	// 先在远程自增
	val, err := c.remote.Incr(ctx, key)
	if err != nil {
		c.metrics.RecordError("incr")
		return 0, err
	}

	// 更新本地缓存
	_ = c.local.Set(ctx, key, val, 5*time.Minute)

	return val, nil
}

// Decr 自减
func (c *OptimizedMultiLevelCache) Decr(ctx context.Context, key string) (int64, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("decr", time.Since(start))
	}()

	// 先在远程自减
	val, err := c.remote.Decr(ctx, key)
	if err != nil {
		c.metrics.RecordError("decr")
		return 0, err
	}

	// 更新本地缓存
	_ = c.local.Set(ctx, key, val, 5*time.Minute)

	return val, nil
}

// IncrBy 按指定值自增
func (c *OptimizedMultiLevelCache) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	start := time.Now()
	defer func() {
		c.metrics.RecordOperation("incrby", time.Since(start))
	}()

	// 先在远程自增
	val, err := c.remote.IncrBy(ctx, key, value)
	if err != nil {
		c.metrics.RecordError("incrby")
		return 0, err
	}

	// 更新本地缓存
	_ = c.local.Set(ctx, key, val, 5*time.Minute)

	return val, nil
}

// Close 关闭缓存
func (c *OptimizedMultiLevelCache) Close() error {
	c.hotLoader.Stop()
	_ = c.local.Close()
	return c.remote.Close()
}

// Ping 检查缓存是否可用
func (c *OptimizedMultiLevelCache) Ping(ctx context.Context) error {
	return c.remote.Ping(ctx)
}

// LocalCache 获取本地缓存实例
func (c *OptimizedMultiLevelCache) LocalCache() *MemCache {
	return c.local
}

// RemoteCache 获取远程缓存实例
func (c *OptimizedMultiLevelCache) RemoteCache() *RedisCache {
	return c.remote
}

// HotLoader 获取热点数据加载器
func (c *OptimizedMultiLevelCache) HotLoader() *HotDataLoader {
	return c.hotLoader
}

// Metrics 获取缓存 metrics
func (c *OptimizedMultiLevelCache) Metrics() *CacheMetrics {
	return c.metrics
}

// StartHotLoader 启动热点数据加载
func (c *OptimizedMultiLevelCache) StartHotLoader(ctx context.Context) {
	c.hotLoader.Start(ctx)
}

// RegisterDataLoader 注册数据加载器
func (c *OptimizedMultiLevelCache) RegisterDataLoader(loader DataLoader) {
	c.hotLoader.RegisterLoader(loader)
}

// CacheMetrics 缓存 metrics
type CacheMetrics struct {
	hits           map[string]int64
	misses         int64
	errors         map[string]int64
	operations     map[string]int64
	totalLatency   map[string]time.Duration
	mu             sync.RWMutex
}

// NewCacheMetrics 创建缓存 metrics 实例
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{
		hits:         make(map[string]int64),
		errors:       make(map[string]int64),
		operations:   make(map[string]int64),
		totalLatency: make(map[string]time.Duration),
	}
}

// RecordHit 记录缓存命中
func (m *CacheMetrics) RecordHit(cacheType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits[cacheType]++
}

// RecordMiss 记录缓存未命中
func (m *CacheMetrics) RecordMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.misses++
}

// RecordError 记录错误
func (m *CacheMetrics) RecordError(operation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errors[operation]++
}

// RecordOperation 记录操作
func (m *CacheMetrics) RecordOperation(operation string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations[operation]++
	m.totalLatency[operation] += latency
}

// GetStats 获取统计信息
func (m *CacheMetrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	totalHits := int64(0)
	for _, hit := range m.hits {
		totalHits += hit
	}

	totalOperations := int64(0)
	for _, op := range m.operations {
		totalOperations += op
	}

	hitRate := 0.0
	if totalOperations > 0 {
		hitRate = float64(totalHits) / float64(totalOperations)
	}

	stats := map[string]interface{}{
		"hits":            m.hits,
		"misses":          m.misses,
		"total_hits":      totalHits,
		"hit_rate":        hitRate,
		"errors":          m.errors,
		"operations":      m.operations,
		"total_operations": totalOperations,
	}

	// 添加平均延迟
	averageLatency := make(map[string]time.Duration)
	for op, total := range m.totalLatency {
		if count := m.operations[op]; count > 0 {
			averageLatency[op] = total / time.Duration(count)
		}
	}
	stats["average_latency"] = averageLatency

	return stats
}

// Reset 重置统计信息
func (m *CacheMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hits = make(map[string]int64)
	m.misses = 0
	m.errors = make(map[string]int64)
	m.operations = make(map[string]int64)
	m.totalLatency = make(map[string]time.Duration)
}

// String 字符串表示
func (m *CacheMetrics) String() string {
	stats := m.GetStats()
	return fmt.Sprintf("CacheMetrics{hits: %v, misses: %d, hit_rate: %.2f, errors: %v, operations: %v}",
		stats["hits"], stats["misses"], stats["hit_rate"], stats["errors"], stats["operations"])
}

var _ Cache = (*OptimizedMultiLevelCache)(nil)

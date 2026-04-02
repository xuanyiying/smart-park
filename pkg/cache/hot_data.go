package cache

import (
	"context"
	"log"
	"time"
)

// HotDataLoader 热点数据加载器
type HotDataLoader struct {
	cache     Cache
	loaders   []DataLoader
	interval  time.Duration
	isRunning bool
}

// DataLoader 数据加载器接口
type DataLoader interface {
	// Load 加载数据，返回键值对和过期时间
	Load(ctx context.Context) (map[string]interface{}, time.Duration, error)
	// Name 加载器名称
	Name() string
}

// NewHotDataLoader 创建热点数据加载器
func NewHotDataLoader(cache Cache, interval time.Duration) *HotDataLoader {
	return &HotDataLoader{
		cache:    cache,
		loaders:  make([]DataLoader, 0),
		interval: interval,
	}
}

// RegisterLoader 注册数据加载器
func (h *HotDataLoader) RegisterLoader(loader DataLoader) {
	h.loaders = append(h.loaders, loader)
}

// Start 启动热点数据加载
func (h *HotDataLoader) Start(ctx context.Context) {
	if h.isRunning {
		return
	}

	h.isRunning = true

	// 立即执行一次加载
	h.load(ctx)

	// 定期执行加载
	ticker := time.NewTicker(h.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				h.load(ctx)
			case <-ctx.Done():
				ticker.Stop()
				h.isRunning = false
				return
			}
		}
	}()
}

// load 执行数据加载
func (h *HotDataLoader) load(ctx context.Context) {
	for _, loader := range h.loaders {
		go func(l DataLoader) {
			start := time.Now()
			
			data, expiration, err := l.Load(ctx)
			if err != nil {
				log.Printf("[HotDataLoader] %s load failed: %v", l.Name(), err)
				return
			}

			// 将数据加载到缓存
			for key, value := range data {
				err := h.cache.Set(ctx, key, value, expiration)
				if err != nil {
					log.Printf("[HotDataLoader] %s set cache failed for key %s: %v", l.Name(), key, err)
				}
			}

			log.Printf("[HotDataLoader] %s loaded %d items in %v", l.Name(), len(data), time.Since(start))
		}(loader)
	}
}

// Stop 停止热点数据加载
func (h *HotDataLoader) Stop() {
	h.isRunning = false
}

// IsRunning 检查是否正在运行
func (h *HotDataLoader) IsRunning() bool {
	return h.isRunning
}

// ExampleDataLoader 示例数据加载器
type ExampleDataLoader struct {
	name string
}

// NewExampleDataLoader 创建示例数据加载器
func NewExampleDataLoader(name string) *ExampleDataLoader {
	return &ExampleDataLoader{name: name}
}

// Load 加载数据
func (l *ExampleDataLoader) Load(ctx context.Context) (map[string]interface{}, time.Duration, error) {
	// 这里实现具体的数据加载逻辑
	// 例如从数据库加载热点数据
	data := make(map[string]interface{})
	// 示例数据
	data["example:key1"] = "value1"
	data["example:key2"] = "value2"
	
	return data, 10 * time.Minute, nil
}

// Name 加载器名称
func (l *ExampleDataLoader) Name() string {
	return l.name
}
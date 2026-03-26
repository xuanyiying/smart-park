package biz

import (
	"context"
	"strings"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

// RouteConfig 路由配置
type RouteConfig struct {
	Path   string
	Target string
}

// ServiceInstance 服务实例
type ServiceInstance struct {
	ID       string
	Name     string
	Address  string
	Port     int
	Metadata map[string]string
}

// ServiceDiscovery 服务发现接口
type ServiceDiscovery interface {
	// Discover 发现服务实例
	Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error)
	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (<-chan []*ServiceInstance, error)
	// Close 关闭服务发现
	Close() error
}

// StaticDiscovery 静态服务发现实现（从配置加载）
type StaticDiscovery struct {
	instances map[string][]*ServiceInstance
	mu        sync.RWMutex
}

// NewStaticDiscovery 创建静态服务发现
func NewStaticDiscovery(routes []*RouteConfig) *StaticDiscovery {
	instances := make(map[string][]*ServiceInstance)
	for _, route := range routes {
		// 从 target 解析服务名和地址
		// 格式: "vehicle-svc:8001"
		parts := strings.Split(route.Target, ":")
		if len(parts) != 2 {
			continue
		}
		serviceName := parts[0]
		instances[serviceName] = []*ServiceInstance{
			{
				ID:      serviceName + "-1",
				Name:    serviceName,
				Address: "localhost", // TODO: 从配置或服务发现获取
				Port:    mustParseInt(parts[1]),
			},
		}
	}
	return &StaticDiscovery{instances: instances}
}

// Discover 发现服务实例
func (d *StaticDiscovery) Discover(ctx context.Context, serviceName string) ([]*ServiceInstance, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	instances, ok := d.instances[serviceName]
	if !ok {
		return nil, nil
	}
	return instances, nil
}

// Watch 监听服务变化
func (d *StaticDiscovery) Watch(ctx context.Context, serviceName string) (<-chan []*ServiceInstance, error) {
	ch := make(chan []*ServiceInstance, 1)
	// 静态配置不需要监听，直接返回当前实例
	go func() {
		instances, _ := d.Discover(ctx, serviceName)
		ch <- instances
	}()
	return ch, nil
}

// Close 关闭服务发现
func (d *StaticDiscovery) Close() error {
	return nil
}

// RouterUseCase 路由用例
type RouterUseCase struct {
	discovery ServiceDiscovery
	etcdReg  *EtcdRegistry
	routes   []*RouteConfig
	log      *log.Helper
	useEtcd  bool
}

// NewRouterUseCase 创建路由用例
func NewRouterUseCase(discovery ServiceDiscovery, etcdReg *EtcdRegistry, routes []*RouteConfig, useEtcd bool, logger log.Logger) *RouterUseCase {
	return &RouterUseCase{
		discovery: discovery,
		etcdReg:  etcdReg,
		routes:   routes,
		useEtcd:  useEtcd,
		log:      log.NewHelper(logger),
	}
}

// MatchRoute 匹配路由
func (uc *RouterUseCase) MatchRoute(path string) *RouteConfig {
	for _, route := range uc.routes {
		if strings.HasPrefix(path, route.Path) {
			return route
		}
	}
	return nil
}

// GetServiceTarget 获取服务目标地址
func (uc *RouterUseCase) GetServiceTarget(ctx context.Context, path string) (string, error) {
	route := uc.MatchRoute(path)
	if route == nil {
		return "", ErrRouteNotFound
	}

	if uc.useEtcd && uc.etcdReg != nil {
		serviceName := strings.Split(route.Target, ":")[0]
		instances, err := uc.etcdReg.GetService(ctx, serviceName)
		if err == nil && len(instances) > 0 {
			return instances[0].Endpoints[0], nil
		}
	}

	return route.Target, nil
}

// GetAllRoutes 获取所有路由
func (uc *RouterUseCase) GetAllRoutes() []*RouteConfig {
	return uc.routes
}

// Route errors
var (
	ErrRouteNotFound = &RouteError{Code: 404, Message: "route not found"}
)

// RouteError 路由错误
type RouteError struct {
	Code    int
	Message string
}

func (e *RouteError) Error() string {
	return e.Message
}

func mustParseInt(s string) int {
	var n int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

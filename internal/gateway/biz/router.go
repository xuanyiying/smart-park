package biz

import (
	"context"
	"strings"
	"sync"
	"time"

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
				Address: serviceName, // 使用 serviceName 作为默认域名
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

// targetState tracks per-target circuit-breaker state and the round-robin counter
// for the service it belongs to.
type targetState struct {
	failures  int
	openUntil time.Time
}

// circuitBreakerThreshold/thresholdReset define when a target is pulled from the
// rotation after consecutive failures, and for how long.
const (
	circuitBreakerThreshold = 3
	circuitBreakerCooldown  = 10 * time.Second
)

// RouterUseCase 路由用例
type RouterUseCase struct {
	discovery ServiceDiscovery
	etcdReg   *EtcdRegistry
	routes    []*RouteConfig
	log       *log.Helper
	useEtcd   bool

	mu           sync.Mutex
	states       map[string]*targetState
	rrCounters   map[string]uint64
}

// NewRouterUseCase 创建路由用例
func NewRouterUseCase(discovery ServiceDiscovery, etcdReg *EtcdRegistry, routes []*RouteConfig, useEtcd bool, logger log.Logger) *RouterUseCase {
	return &RouterUseCase{
		discovery:  discovery,
		etcdReg:    etcdReg,
		routes:     routes,
		useEtcd:    useEtcd,
		log:        log.NewHelper(logger),
		states:     make(map[string]*targetState),
		rrCounters: make(map[string]uint64),
	}
}

// MarkFailed records a delivery failure against a target. Consecutive failures
// open the circuit so the target is skipped while it is known to be down.
func (uc *RouterUseCase) MarkFailed(target string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	st, ok := uc.states[target]
	if !ok {
		st = &targetState{}
		uc.states[target] = st
	}
	st.failures++
	if st.failures >= circuitBreakerThreshold {
		st.openUntil = time.Now().Add(circuitBreakerCooldown)
	}
}

// MarkSucceeded resets the failure streak of a target after any successful response.
func (uc *RouterUseCase) MarkSucceeded(target string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()

	if st, ok := uc.states[target]; ok {
		st.failures = 0
		st.openUntil = time.Time{}
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
//
// 从 etcd 取回全部实例后做健康感知的轮询负载均衡：连续失败的目标会被熔断
// 冷却一段时间，期间流量自动切到其余实例；静态配置退化为单实例轮询。
func (uc *RouterUseCase) GetServiceTarget(ctx context.Context, path string) (string, error) {
	route := uc.MatchRoute(path)
	if route == nil {
		return "", ErrRouteNotFound
	}

	var candidates []string
	if uc.useEtcd && uc.etcdReg != nil {
		serviceName := strings.Split(route.Target, ":")[0]
		if instances, err := uc.etcdReg.GetService(ctx, serviceName); err == nil && len(instances) > 0 {
			for _, inst := range instances {
				candidates = append(candidates, inst.Endpoints...)
			}
		}
	}
	if len(candidates) == 0 {
		candidates = []string{route.Target}
	}

	eligible := uc.eligibleTargets(candidates)
	if len(eligible) == 0 {
		// 所有目标都在熔断冷却期：半开放行一次尝试，否则后端恢复后网关
		// 会永久 502。选轮询序列中的下一个即可。
		eligible = candidates
	}

	uc.mu.Lock()
	defer uc.mu.Unlock()

	if uc.rrCounters == nil {
		uc.rrCounters = make(map[string]uint64)
	}
	if uc.states == nil {
		uc.states = make(map[string]*targetState)
	}

	key := route.Target
	n := uint64(len(eligible))
	idx := uc.rrCounters[key] % n
	uc.rrCounters[key]++

	return eligible[idx], nil
}

// eligibleTargets filters out targets whose circuit is currently open.
func (uc *RouterUseCase) eligibleTargets(candidates []string) []string {
	now := time.Now()
	var out []string
	for _, c := range candidates {
		st, ok := uc.states[c]
		if ok && st.openUntil.After(now) {
			continue
		}
		out = append(out, c)
	}
	return out
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

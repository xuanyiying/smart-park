package service

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"github.com/xuanyiying/smart-park/internal/gateway/biz"
	"github.com/xuanyiying/smart-park/pkg/middleware"
	"github.com/xuanyiying/smart-park/pkg/ws"
)

type GatewayService struct {
	router *biz.RouterUseCase
	hub    *ws.Hub
	logger log.Logger
	log    *log.Helper
}

func NewGatewayService(router *biz.RouterUseCase, hub *ws.Hub, logger log.Logger) *GatewayService {
	return &GatewayService{
		router: router,
		hub:    hub,
		logger: logger,
		log:    log.NewHelper(logger),
	}
}

// ServeHTTP 实现 http.Handler 接口
func (s *GatewayService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 匹配路由
	target, err := s.router.GetServiceTarget(ctx, r.URL.Path)
	if err != nil {
		s.log.Errorf("route not found: %s", r.URL.Path)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// 创建反向代理
	proxy, err := s.createProxy(target)
	if err != nil {
		s.log.Errorf("failed to create proxy for target %s: %v", target, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	// 记录请求开始
	startTime := time.Now()
	s.log.Infof("proxying request: %s %s -> %s", r.Method, r.URL.Path, target)

	// 设置请求头
	r.Header.Set("X-Forwarded-For", getClientIP(r))
	r.Header.Set("X-Forwarded-Proto", "http")
	r.Header.Set("X-Real-IP", getClientIP(r))

	// 代理请求
	proxy.ServeHTTP(w, r)

	// 记录请求完成
	duration := time.Since(startTime)
	s.log.Infof("request completed: %s %s, duration: %v", r.Method, r.URL.Path, duration)
}

// proxyTransport 是所有反向代理共享的传输层：带拨号/响应头超时与受限的连接池，
// 防止后端挂起或慢响应无限占用网关资源（默认 Transport 没有任何超时）。
var proxyTransport = &http.Transport{
	Proxy:                 http.ProxyFromEnvironment,
	DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
	MaxIdleConns:          200,
	MaxIdleConnsPerHost:   50,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   5 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	ResponseHeaderTimeout: 30 * time.Second,
}

// createProxy 创建反向代理
func (s *GatewayService) createProxy(target string) (*httputil.ReverseProxy, error) {
	// 解析目标地址
	targetURL, err := url.Parse(fmt.Sprintf("http://%s", target))
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Transport = proxyTransport

	// 自定义错误处理：转发失败即向熔断统计上报，触发实例剔除
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.log.Errorf("proxy error: %v, path: %s", err, r.URL.Path)
		s.router.MarkFailed(target)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// 收到任何响应（即使是 4xx/5xx）都证明后端存活，重置熔断计数
	proxy.ModifyResponse = func(resp *http.Response) error {
		s.router.MarkSucceeded(target)
		return nil
	}

	// 修改请求
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		// 保留原始路径
		req.Host = targetURL.Host
	}

	return proxy, nil
}

// getClientIP 获取客户端真实 IP
func getClientIP(r *http.Request) string {
	// X-Forwarded-For
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}

// HealthCheck 健康检查端点
func (s *GatewayService) HealthCheck(ctx context.Context) (map[string]bool, error) {
	routes := s.router.GetAllRoutes()
	health := make(map[string]bool)

	client := &http.Client{Timeout: 2 * time.Second}
	for _, route := range routes {
		target, err := s.router.GetServiceTarget(ctx, route.Path)
		if err != nil || target == "" {
			health[route.Path] = false
			continue
		}

		// 判活标准：拿到任何 HTTP 响应（含 404/401 等）都证明服务进程存活；
		// 传输层错误（连接被拒、DNS 失败、超时）一律判不健康。此前靠错误文本里
		// 是否包含 "connection refused" 来区分，极度脆弱。
		resp, err := client.Get(fmt.Sprintf("http://%s/", target))
		if err != nil {
			health[route.Path] = false
			continue
		}
		health[route.Path] = true
		resp.Body.Close()
	}

	return health, nil
}

// GetRoutes 获取所有路由
func (s *GatewayService) GetRoutes(ctx context.Context) ([]*biz.RouteConfig, error) {
	return s.router.GetAllRoutes(), nil
}

func (s *GatewayService) StreamProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	target, err := s.router.GetServiceTarget(ctx, r.URL.Path)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	proxy, err := s.createProxy(target)
	if err != nil {
		s.log.Errorf("failed to create websocket proxy for %s: %v", target, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}

	s.log.Infof("stream proxy upgrading: %s -> %s", r.URL.Path, target)
	proxy.ServeHTTP(w, r)
}

// HandleWebSocket upgrades a connection for an authenticated client.
//
// Identity is taken from the JWT claims established by the auth middleware. It is
// deliberately not read from the query string: a URL parameter is attacker controlled, so
// the previous implementation let anyone subscribe to any user's or tenant's events simply
// by typing a different id.
func (s *GatewayService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims, err := middleware.GetClaimsFromContext(r.Context())
	if err != nil || claims == nil {
		s.log.Warnf("websocket upgrade rejected: %v", err)
		http.Error(w, "Unauthorized: valid token required", http.StatusUnauthorized)
		return
	}

	opts := ws.UpgradeOptions{
		UserID:   claims.UserID,
		TenantID: claims.TenantID,
	}

	if err := ws.UpgradeHTTP(w, r, s.hub, opts, s.logger); err != nil {
		s.log.Errorf("websocket upgrade failed: %v", err)
		return
	}

	s.log.Infof("websocket connection established: user=%s, tenant=%s", claims.UserID, claims.TenantID)
}

func (s *GatewayService) Hub() *ws.Hub {
	return s.hub
}

// ReadinessProbe 就绪探针
func (s *GatewayService) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	health, err := s.HealthCheck(r.Context())
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	healthy := 0
	for path, ok := range health {
		if ok {
			healthy++
		} else {
			s.log.Errorf("service unhealthy: %s", path)
		}
	}

	// 至少有一个后端可达即视为就绪。此前要求全部后端健康，任一服务下线就会把
	// 网关整体摘出负载均衡，把单个服务的故障放大成全站不可用（雪崩）。
	if healthy == 0 {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

// LivenessProbe 存活探针
func (s *GatewayService) LivenessProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

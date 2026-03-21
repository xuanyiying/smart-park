package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"

	"smart-park/internal/gateway/biz"
)

// GatewayService 网关服务
type GatewayService struct {
	router *biz.RouterUseCase
	log    *log.Helper
}

// NewGatewayService 创建网关服务
func NewGatewayService(router *biz.RouterUseCase, logger log.Logger) *GatewayService {
	return &GatewayService{
		router: router,
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

// createProxy 创建反向代理
func (s *GatewayService) createProxy(target string) (*httputil.ReverseProxy, error) {
	// 解析目标地址
	targetURL, err := url.Parse(fmt.Sprintf("http://%s", target))
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		s.log.Errorf("proxy error: %v, path: %s", err, r.URL.Path)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
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

	for _, route := range routes {
		// TODO: 实现实际的健康检查
		health[route.Path] = true
	}

	return health, nil
}

// GetRoutes 获取所有路由
func (s *GatewayService) GetRoutes(ctx context.Context) ([]*biz.RouteConfig, error) {
	return s.router.GetAllRoutes(), nil
}

// StreamProxy WebSocket 代理支持
func (s *GatewayService) StreamProxy(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 匹配路由
	target, err := s.router.GetServiceTarget(ctx, r.URL.Path)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// 创建 WebSocket 代理
	// TODO: 实现 WebSocket 代理逻辑
	s.log.Infof("stream proxy: %s -> %s", r.URL.Path, target)
	http.Error(w, "WebSocket proxy not implemented", http.StatusNotImplemented)
}

// ReadinessProbe 就绪探针
func (s *GatewayService) ReadinessProbe(w http.ResponseWriter, r *http.Request) {
	// 检查所有后端服务是否可用
	health, err := s.HealthCheck(r.Context())
	if err != nil {
		http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
		return
	}

	for path, ok := range health {
		if !ok {
			s.log.Errorf("service unhealthy: %s", path)
			http.Error(w, "Service Unavailable", http.StatusServiceUnavailable)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

// LivenessProbe 存活探针
func (s *GatewayService) LivenessProbe(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, "OK")
}

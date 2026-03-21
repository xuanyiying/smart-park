package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/http"

	"smart-park/internal/conf"
	"smart-park/internal/gateway/biz"
	"smart-park/internal/gateway/service"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/gateway.yaml", "config path")
}

func newApp(logger log.Logger, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name("gateway"),
		kratos.Logger(logger),
		kratos.Server(hs),
	)
}

func main() {
	flag.Parse()

	logger := log.NewStdLogger(os.Stdout)
	logHelper := log.NewHelper(logger)

	// Load configuration
	cfg, err := conf.LoadConfig(flagconf)
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		os.Exit(1)
	}

	// Parse routes from config
	routes := parseRoutes(cfg)
	logHelper.Infof("loaded %d routes", len(routes))

	// Create service discovery (static for now)
	discovery := biz.NewStaticDiscovery(routes)

	// Create router use case
	routerUseCase := biz.NewRouterUseCase(discovery, routes, logger)

	// Create gateway service
	gatewaySvc := service.NewGatewayService(routerUseCase, logger)

	// Create HTTP server with gateway handler
	hs := http.NewServer(
		http.Addr(fmt.Sprintf(":%d", cfg.Server.Port)),
	)

	// Register gateway handler
	hs.HandlePrefix("/", gatewaySvc)

	// Register health check endpoints
	hs.HandleFunc("/health", gatewaySvc.LivenessProbe)
	hs.HandleFunc("/ready", gatewaySvc.ReadinessProbe)
	hs.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		routes, _ := gatewaySvc.GetRoutes(r.Context())
		for _, route := range routes {
			fmt.Fprintf(w, "%s -> %s\n", route.Path, route.Target)
		}
	})

	// Start application
	app := newApp(logger, hs)
	logHelper.Infof("gateway service starting on port %d", cfg.Server.Port)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

// parseRoutes 从配置解析路由
func parseRoutes(cfg *conf.Config) []*biz.RouteConfig {
	var routes []*biz.RouteConfig

	// 从配置中读取路由规则
	if routeMap, ok := cfg.Raw["routes"].([]interface{}); ok {
		for _, r := range routeMap {
			if route, ok := r.(map[interface{}]interface{}); ok {
				path, _ := route["path"].(string)
				target, _ := route["target"].(string)
				if path != "" && target != "" {
					routes = append(routes, &biz.RouteConfig{
						Path:   path,
						Target: target,
					})
				}
			}
		}
	}

	// 默认路由
	if len(routes) == 0 {
		routes = []*biz.RouteConfig{
			{Path: "/api/v1/device", Target: "vehicle-svc:8001"},
			{Path: "/api/v1/billing", Target: "billing-svc:8002"},
			{Path: "/api/v1/pay", Target: "payment-svc:8003"},
			{Path: "/api/v1/admin", Target: "admin-svc:8004"},
		}
	}

	return routes
}

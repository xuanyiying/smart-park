package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	khttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/xuanyiying/smart-park/internal/conf"
	"github.com/xuanyiying/smart-park/internal/gateway/biz"
	"github.com/xuanyiying/smart-park/internal/gateway/service"
	"github.com/xuanyiying/smart-park/pkg/auth"
	"github.com/xuanyiying/smart-park/pkg/metrics"
	"github.com/xuanyiying/smart-park/pkg/middleware"
	"github.com/xuanyiying/smart-park/pkg/trace"
	"github.com/xuanyiying/smart-park/pkg/ws"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/gateway.yaml", "config path")
}

func newApp(logger log.Logger, hs *khttp.Server) *kratos.App {
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

	cfg, err := conf.LoadConfig(flagconf)
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		os.Exit(1)
	}

	// Initialize tracing
	traceCfg := &trace.Config{
		Enabled:     true,
		ServiceName: "gateway-svc",
		Endpoint:    cfg.Otel.Endpoint,
		SampleRate:  1.0,
	}
	tracerProvider, err := trace.NewTracerProvider(traceCfg)
	if err != nil {
		logHelper.Errorf("failed to initialize tracer: %v", err)
		// Don't exit, just log the error
	} else {
		logHelper.Info("tracing initialized successfully")
		defer tracerProvider.Shutdown(context.Background())
	}

	routes := parseRoutes(cfg)
	logHelper.Infof("loaded %d routes", len(routes))

	discovery := biz.NewStaticDiscovery(routes)
	var etcdReg *biz.EtcdRegistry
	useEtcd := false
	routerUseCase := biz.NewRouterUseCase(discovery, etcdReg, routes, useEtcd, logger)

	hub := ws.NewHub(logger)
	go hub.Run()

	gatewaySvc := service.NewGatewayService(routerUseCase, hub, logger)

	// Authentication is mandatory for the gateway.
	//
	// The gateway is the single entry point to every backend service, so running it
	// without verifying tokens exposes the whole system. Previously the JWT section in
	// gateway.yaml was documentation only: it was parsed into config but never applied,
	// which made the deployment look protected while every route was open.
	if cfg.JWT.PublicKeyPath == "" {
		logHelper.Error("jwt.public_key_path is required: refusing to start an unauthenticated gateway")
		os.Exit(1)
	}

	jwtManager, err := auth.NewJWTManager(&auth.JWTConfig{
		PublicKeyPath: cfg.JWT.PublicKeyPath,
		TokenDuration: cfg.JWT.TokenDuration,
	})
	if err != nil {
		logHelper.Errorf("failed to create JWT manager: %v", err)
		os.Exit(1)
	}

	// Probe and metrics endpoints must stay reachable by the orchestrator and scraper.
	skipPaths := append([]string{"/health", "/ready", "/metrics"}, cfg.JWT.SkipPaths...)
	requireAuth := middleware.NewJWTMiddleware(jwtManager, logger, skipPaths).Handler

	hs := khttp.NewServer(
		khttp.Address(fmt.Sprintf(":%d", cfg.Server.Port)),
	)

	hs.HandlePrefix("/", requireAuth(gatewaySvc))
	hs.HandleFunc("/health", gatewaySvc.LivenessProbe)
	hs.HandleFunc("/ready", gatewaySvc.ReadinessProbe)
	hs.Handle("/ws", requireAuth(http.HandlerFunc(gatewaySvc.HandleWebSocket)))
	hs.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		routes, _ := gatewaySvc.GetRoutes(r.Context())
		for _, route := range routes {
			fmt.Fprintf(w, "%s -> %s\n", route.Path, route.Target)
		}
	})
	hs.HandlePrefix("/metrics", metrics.NewHandler())

	app := newApp(logger, hs)
	logHelper.Infof("gateway service starting on port %d", cfg.Server.Port)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

func parseRoutes(cfg *conf.Config) []*biz.RouteConfig {
	routes := make([]*biz.RouteConfig, 0, len(cfg.Routes))
	for _, r := range cfg.Routes {
		routes = append(routes, &biz.RouteConfig{
			Path:   r.Path,
			Target: r.Target,
		})
	}
	return routes
}

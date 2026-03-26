package main

import (
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

	routes := parseRoutes(cfg)
	logHelper.Infof("loaded %d routes", len(routes))

	discovery := biz.NewStaticDiscovery(routes)
	var etcdReg *biz.EtcdRegistry
	useEtcd := false
	routerUseCase := biz.NewRouterUseCase(discovery, etcdReg, routes, useEtcd, logger)
	gatewaySvc := service.NewGatewayService(routerUseCase, logger)

	hs := khttp.NewServer(
		khttp.Address(fmt.Sprintf(":%d", cfg.Server.Port)),
	)

	hs.HandlePrefix("/", gatewaySvc)
	hs.HandleFunc("/health", gatewaySvc.LivenessProbe)
	hs.HandleFunc("/ready", gatewaySvc.ReadinessProbe)
	hs.HandleFunc("/routes", func(w http.ResponseWriter, r *http.Request) {
		routes, _ := gatewaySvc.GetRoutes(r.Context())
		for _, route := range routes {
			fmt.Fprintf(w, "%s -> %s\n", route.Path, route.Target)
		}
	})

	app := newApp(logger, hs)
	logHelper.Infof("gateway service starting on port %d", cfg.Server.Port)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

func parseRoutes(cfg *conf.Config) []*biz.RouteConfig {
	routes := []*biz.RouteConfig{
		{Path: "/api/v1/device", Target: "vehicle-svc:8001"},
		{Path: "/api/v1/billing", Target: "billing-svc:8002"},
		{Path: "/api/v1/pay", Target: "payment-svc:8003"},
		{Path: "/api/v1/admin", Target: "admin-svc:8004"},
	}
	return routes
}

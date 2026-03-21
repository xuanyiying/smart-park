//go:build !wireinject
// +build !wireinject

package main

import (
	"context"
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "github.com/xuanyiying/smart-park/api/billing/v1"
	"github.com/xuanyiying/smart-park/ent"
	"github.com/xuanyiying/smart-park/internal/billing/biz"
	"github.com/xuanyiying/smart-park/internal/billing/data"
	"github.com/xuanyiying/smart-park/internal/billing/service"
	"github.com/xuanyiying/smart-park/internal/conf"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/billing.yaml", "config path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name("billing"),
		kratos.Logger(logger),
		kratos.Server(gs, hs),
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

	// Connect to database
	dbClient, err := ent.Open("postgres", cfg.Database.Source, logger)
	if err != nil {
		logHelper.Errorf("failed to connect database: %v", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	// Run migrations
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		logHelper.Errorf("failed to migrate database: %v", err)
		os.Exit(1)
	}

	// Initialize data layer
	dataLayer, cleanup, err := data.NewData(dbClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	// Initialize repositories
	billingRepo := data.NewBillingRuleRepo(dataLayer)

	// Initialize business logic
	billingUseCase := biz.NewBillingUseCase(billingRepo, logger)

	// Initialize gRPC service
	billingSvc := service.NewBillingService(billingUseCase, logger)

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Addr(":9002"),
	)

	// Create HTTP server
	hs := http.NewServer(
		http.Addr(":8002"),
	)

	// Register services
	v1.RegisterBillingServiceServer(gs, billingSvc)
	v1.RegisterBillingServiceHTTPServer(hs, billingSvc)

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

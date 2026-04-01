package main

import (
	"context"
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/xuanyiying/smart-park/pkg/database"

	v1 "github.com/xuanyiying/smart-park/api/billing/v1"
	"github.com/xuanyiying/smart-park/internal/billing/biz"
	"github.com/xuanyiying/smart-park/internal/billing/data/ent"
	"github.com/xuanyiying/smart-park/internal/billing/service"
	"github.com/xuanyiying/smart-park/pkg/config"
	"github.com/xuanyiying/smart-park/pkg/metrics"
	"github.com/xuanyiying/smart-park/pkg/trace"
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
	cfg, err := config.Load(flagconf)
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		os.Exit(1)
	}

	// Initialize tracing
	traceCfg := &trace.Config{
		Enabled:     cfg.Telemetry.Enabled,
		ServiceName: cfg.Telemetry.ServiceName,
		Endpoint:    cfg.Telemetry.Endpoint,
		SampleRate:  cfg.Telemetry.SampleRate,
	}
	tracerProvider, err := trace.NewTracerProvider(traceCfg)
	if err != nil {
		logHelper.Errorf("failed to initialize tracer: %v", err)
		// Don't exit, just log the error
	} else {
		logHelper.Info("tracing initialized successfully")
		defer tracerProvider.Shutdown(context.Background())
	}

	// Connect to database with read-write separation
	dbCfg := &database.Config{
		Primary: struct {
			Source string
		}{
			Source: cfg.Database.Primary.Source,
		},
		Replica: struct {
			Source string
		}{
			Source: cfg.Database.Replica.Source,
		},
	}
	dbManager, err := database.NewDBManager(dbCfg)
	if err != nil {
		logHelper.Errorf("failed to connect database: %v", err)
		os.Exit(1)
	}
	defer dbManager.Close()

	// Connect to database using ent
	dbClient, err := ent.Open("postgres", dbManager.Primary())
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

	// Seed billing rules data
	if err := billingRepo.SeedData(context.Background()); err != nil {
		logHelper.Errorf("failed to seed billing rules: %v", err)
		// Don't exit, just log the error
	}

	// Initialize business logic
	billingUseCase := biz.NewBillingUseCase(billingRepo, logger)

	// Initialize gRPC service
	billingSvc := service.NewBillingService(billingUseCase, logger)

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9002"),
	)

	// Create HTTP server
	hs := http.NewServer(
		http.Address(":8002"),
	)

	// Register services
	v1.RegisterBillingServiceServer(gs, billingSvc)
	v1.RegisterBillingServiceHTTPServer(hs, billingSvc)

	// Register Prometheus metrics endpoint
	hs.HandlePrefix("/metrics", metrics.NewHandler())

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

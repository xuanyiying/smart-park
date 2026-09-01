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
	"github.com/xuanyiying/smart-park/pkg/trace"

	v1 "github.com/xuanyiying/smart-park/api/admin/v1"
	"github.com/xuanyiying/smart-park/internal/admin/biz"
	"github.com/xuanyiying/smart-park/internal/admin/data"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent"
	"github.com/xuanyiying/smart-park/internal/admin/service"
	multentdata "github.com/xuanyiying/smart-park/internal/multitenancy/data"
	multent "github.com/xuanyiying/smart-park/internal/multitenancy/data/ent"
	"github.com/xuanyiying/smart-park/pkg/config"
	"github.com/xuanyiying/smart-park/pkg/metrics"
	tenantpkg "github.com/xuanyiying/smart-park/pkg/tenant"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/admin.yaml", "config path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name("admin"),
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
	dbCfg := &database.RWConfig{
		Primary: struct {
			Source string
		}{
			Source: cfg.Database.Source,
		},
		Replica: struct {
			Source string
		}{
			Source: cfg.Database.Source,
		},
	}
	dbManager, err := database.NewDBManager(dbCfg)
	if err != nil {
		logHelper.Errorf("failed to connect database: %v", err)
		os.Exit(1)
	}
	defer dbManager.Close()

	// Connect to database using ent
	dbClient, err := ent.Open("postgres", cfg.Database.Source)
	if err != nil {
		logHelper.Errorf("failed to connect database: %v", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	// Scope every query and mutation to the tenant carried by the request
	// context, so an operator only ever sees their own estate. The multitenancy
	// client below is intentionally left unscoped: it manages tenants themselves.
	tenantpkg.ApplyTenantScoping(dbClient, data.TenantScopes(), data.TenantTypes())

	// Run migrations
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		logHelper.Errorf("failed to migrate database: %v", err)
		os.Exit(1)
	}
	logHelper.Info("database migrated successfully")

	// Initialize data layer
	dataLayer, cleanup, err := data.NewData(dbClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	// Initialize repositories
	adminRepo := data.NewAdminRepo(dataLayer)

	// Seed initial data
	if err := adminRepo.SeedData(context.Background()); err != nil {
		logHelper.Errorf("failed to seed data: %v", err)
		// Don't exit, just log the error
	} else {
		logHelper.Info("seed data created successfully")
	}

	// Initialize business logic
	adminUseCase := biz.NewAdminUseCase(adminRepo, logger)

	multentClient, err := multent.Open("postgres", cfg.Database.Source)
	if err != nil {
		logHelper.Errorf("failed to connect multitenancy database: %v", err)
		os.Exit(1)
	}
	defer multentClient.Close()

	multentDataLayer, multentCleanup, err := multentdata.NewData(multentClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize multitenancy data layer: %v", err)
		os.Exit(1)
	}
	defer multentCleanup()

	tenantRepo := multentdata.NewTenantRepo(multentDataLayer)
	tenantExtractor := tenantpkg.NewHeaderExtractor("X-Tenant-ID")
	tenantMW := tenantpkg.TenantMiddleware(tenantRepo, tenantExtractor, logger)

	// Initialize gRPC service
	adminSvc := service.NewAdminService(adminUseCase, logger)

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9004"),
		grpc.Middleware(tenantMW),
	)

	// Create HTTP server
	hs := http.NewServer(
		http.Address(":8004"),
		http.Middleware(tenantMW),
	)

	// Register services
	v1.RegisterAdminServiceServer(gs, adminSvc)
	v1.RegisterAdminServiceHTTPServer(hs, adminSvc)

	// Register Prometheus metrics endpoint
	hs.HandlePrefix("/metrics", metrics.NewHandler())

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

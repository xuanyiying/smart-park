package main

import (
	"context"
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	_ "github.com/lib/pq"

	v1 "github.com/xuanyiying/smart-park/api/admin/v1"
	"github.com/xuanyiying/smart-park/internal/admin/biz"
	"github.com/xuanyiying/smart-park/internal/admin/data"
	"github.com/xuanyiying/smart-park/internal/admin/data/ent"
	"github.com/xuanyiying/smart-park/internal/admin/service"
	"github.com/xuanyiying/smart-park/pkg/config"
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

	// Connect to database
	dbClient, err := ent.Open("postgres", cfg.Database.Source)
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

	// Initialize gRPC service
	adminSvc := service.NewAdminService(adminUseCase, logger)

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9004"),
	)

	// Create HTTP server
	hs := http.NewServer(
		http.Address(":8004"),
	)

	// Register services
	v1.RegisterAdminServiceServer(gs, adminSvc)
	v1.RegisterAdminServiceHTTPServer(hs, adminSvc)

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

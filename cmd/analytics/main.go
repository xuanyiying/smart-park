package main

import (
	"context"
	"database/sql"
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"

	v1 "github.com/xuanyiying/smart-park/api/analytics/v1"
	"github.com/xuanyiying/smart-park/internal/analytics/biz"
	"github.com/xuanyiying/smart-park/internal/analytics/data"
	"github.com/xuanyiying/smart-park/internal/analytics/service"
	"github.com/xuanyiying/smart-park/pkg/config"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/analytics.yaml", "config path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name("analytics"),
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
	var dbClient *sql.DB
	var dataLayer *data.Data
	
	dbClient, err = data.NewDatabase(cfg.Database.Source)
	if err != nil {
		logHelper.Warnf("failed to connect database: %v, using mock data", err)
		// 继续执行，使用模拟数据
		dataLayer, _ = data.NewData(nil, logger)
	} else {
		defer dbClient.Close()
		dataLayer, err = data.NewData(dbClient, logger)
		if err != nil {
			logHelper.Warnf("failed to initialize data layer: %v, using mock data", err)
			// 继续执行，使用模拟数据
			dataLayer, _ = data.NewData(nil, logger)
		}
	}

	// Connect to Redis
	var redisClient *redis.Client
	if cfg.Redis.Addr != "" {
		redisClient = redis.NewClient(&redis.Options{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		
		// Test Redis connection
		if err := redisClient.Ping(context.Background()).Err(); err != nil {
			logHelper.Warnf("failed to connect to Redis: %v, running without real-time data", err)
			redisClient = nil
		} else {
			logHelper.Info("connected to Redis successfully")
		}
	}

	// Initialize data collector
	collector := data.NewDataCollector(dataLayer, redisClient)
	
	// Start processing events
	if err := collector.StartProcessing(context.Background()); err != nil {
		logHelper.Warnf("failed to start event processing: %v", err)
	}

	// Initialize repositories
	analyticsRepo := data.NewAnalyticsRepo(dataLayer)

	// Initialize business logic layer
	analyticsUseCase := biz.NewAnalyticsUseCase(analyticsRepo, logger)

	// Initialize gRPC service
	analyticsSvc := service.NewAnalyticsService(analyticsUseCase, logger)

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9006"),
	)

	// Create HTTP server
	hs := http.NewServer(
		http.Address(":8006"),
	)

	// Register services
	v1.RegisterAnalyticsServiceServer(gs, analyticsSvc)
	v1.RegisterAnalyticsServiceHTTPServer(hs, analyticsSvc)

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

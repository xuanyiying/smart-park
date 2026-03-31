package main

import (
	"context"
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"

	billingv1 "github.com/xuanyiying/smart-park/api/billing/v1"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
	"github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
	"github.com/xuanyiying/smart-park/internal/vehicle/data"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
	"github.com/xuanyiying/smart-park/internal/vehicle/service"
	"github.com/xuanyiying/smart-park/pkg/config"
	"github.com/xuanyiying/smart-park/pkg/lock"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/vehicle.yaml", "config path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name("vehicle"),
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

	// Initialize data layer
	dataLayer, cleanup, err := data.NewData(dbClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	// Initialize repositories
	vehicleRepo := data.NewVehicleRepo(dataLayer)

	// Seed device data
	if err := vehicleRepo.SeedData(context.Background()); err != nil {
		logHelper.Errorf("failed to seed device data: %v", err)
		// Don't exit, just log the error
	}

	// Initialize MQTT client
	var mqttClient mqtt.Client
	if cfg.MQTT.Broker != "" {
		mqttCfg := &mqtt.Config{
			Broker:   cfg.MQTT.Broker,
			Port:     cfg.MQTT.Port,
			ClientID: cfg.MQTT.ClientID,
			Username: cfg.MQTT.Username,
			Password: cfg.MQTT.Password,
		}
		mqttClient = mqtt.NewMQTTClient(mqttCfg)
		if err := mqttClient.Connect(); err != nil {
			logHelper.Errorf("failed to connect MQTT client: %v", err)
			os.Exit(1)
		}
		logHelper.Info("mqtt client connected successfully")
	} else {
		logHelper.Warn("mqtt config not provided, using mock client")
		mqttClient = mqtt.NewMockMQTTClient()
		if err := mqttClient.Connect(); err != nil {
			logHelper.Errorf("failed to connect mock MQTT client: %v", err)
			os.Exit(1)
		}
	}
	defer mqttClient.Disconnect()

	// Initialize Redis client for distributed lock
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// Test Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logHelper.Warnf("failed to connect Redis: %v, distributed lock will not be available", err)
	} else {
		logHelper.Info("redis client connected successfully")
	}
	defer redisClient.Close()

	// Initialize distributed lock repository
	lockRepo := lock.NewRedisLockRepo(redisClient, logger, "smart-park:vehicle")

	// Initialize billing service client
	var billingClient billing.Client
	if cfg.Billing == nil || cfg.Billing.Endpoint == "" {
		logHelper.Error("billing service endpoint is required")
		os.Exit(1)
	}
	conn, err := grpc.DialInsecure(
		context.Background(),
		grpc.WithEndpoint(cfg.Billing.Endpoint),
	)
	if err != nil {
		logHelper.Errorf("failed to connect billing service: %v", err)
		os.Exit(1)
	}
	billingGrpcClient := billingv1.NewBillingServiceClient(conn)
	billingClient = billing.NewClient(billingGrpcClient, logger)
	logHelper.Infof("billing service client connected to %s", cfg.Billing.Endpoint)

	// Initialize business logic layer
	entryExitUseCase := biz.NewEntryExitUseCase(vehicleRepo, billingClient, mqttClient, lockRepo, logger)
	deviceUseCase := biz.NewDeviceUseCase(vehicleRepo, logger)
	vehicleQueryUseCase := biz.NewVehicleQueryUseCase(vehicleRepo, logger)
	commandUseCase := biz.NewCommandUseCase(vehicleRepo, mqttClient, logger)
	recordQueryUseCase := biz.NewRecordQueryUseCase(vehicleRepo)

	// Initialize gRPC service
	vehicleSvc := service.NewVehicleService(entryExitUseCase, deviceUseCase, vehicleQueryUseCase, commandUseCase, recordQueryUseCase, logger)

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9001"),
	)

	// Create HTTP server
	hs := http.NewServer(
		http.Address(":8001"),
	)

	// Register services
	v1.RegisterVehicleServiceServer(gs, vehicleSvc)
	v1.RegisterVehicleServiceHTTPServer(hs, vehicleSvc)

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"

	adminv1 "github.com/xuanyiying/smart-park/api/admin/v1"
	billingv1 "github.com/xuanyiying/smart-park/api/billing/v1"
	paymentv1 "github.com/xuanyiying/smart-park/api/payment/v1"
	vehiclev1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	adminBiz "github.com/xuanyiying/smart-park/internal/admin/biz"
	adminData "github.com/xuanyiying/smart-park/internal/admin/data"
	adminService "github.com/xuanyiying/smart-park/internal/admin/service"
	billingBiz "github.com/xuanyiying/smart-park/internal/billing/biz"
	billingData "github.com/xuanyiying/smart-park/internal/billing/data"
	billingService "github.com/xuanyiying/smart-park/internal/billing/service"
	paymentBiz "github.com/xuanyiying/smart-park/internal/payment/biz"
	paymentData "github.com/xuanyiying/smart-park/internal/payment/data"
	paymentService "github.com/xuanyiying/smart-park/internal/payment/service"
	vehicleBiz "github.com/xuanyiying/smart-park/internal/vehicle/biz"
	vehicleClient "github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
	vehicleData "github.com/xuanyiying/smart-park/internal/vehicle/data"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
	vehicleService "github.com/xuanyiying/smart-park/internal/vehicle/service"
	"github.com/xuanyiying/smart-park/pkg/config"
	"github.com/xuanyiying/smart-park/pkg/lock"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "./configs/dev.yaml", "config path")
}

func main() {
	flag.Parse()

	logger := log.NewStdLogger(os.Stdout)
	logHelper := log.NewHelper(logger)

	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                                                           ║")
	fmt.Println("║   Smart Park - All-in-One Development Server              ║")
	fmt.Println("║                                                           ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Load configuration
	cfg, err := config.Load(flagconf)
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		os.Exit(1)
	}
	logHelper.Info("configuration loaded successfully")

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

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		logHelper.Warnf("failed to connect Redis: %v", err)
	} else {
		logHelper.Info("redis client connected successfully")
	}
	defer redisClient.Close()

	// Initialize MQTT client (using mock for development)
	var mqttClient mqtt.Client
	mqttClient = mqtt.NewMockMQTTClient()
	if err := mqttClient.Connect(); err != nil {
		logHelper.Errorf("failed to connect MQTT client: %v", err)
		os.Exit(1)
	}
	logHelper.Info("mqtt client connected (mock mode)")
	defer mqttClient.Disconnect()

	// Initialize distributed lock repository
	lockRepo := lock.NewRedisLockRepo(redisClient, logger, "smart-park:all-in-one")

	// Initialize Vehicle Service
	logHelper.Info("initializing vehicle service...")
	vehicleDataLayer, cleanup, err := vehicleData.NewData(dbClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize vehicle data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	vehicleRepo := vehicleData.NewVehicleRepo(vehicleDataLayer)
	mockBillingClient := &mockBillingClient{}

	entryExitUseCase := vehicleBiz.NewEntryExitUseCase(vehicleRepo, mockBillingClient, mqttClient, lockRepo, logger)
	deviceUseCase := vehicleBiz.NewDeviceUseCase(vehicleRepo, logger)
	vehicleQueryUseCase := vehicleBiz.NewVehicleQueryUseCase(vehicleRepo, logger)
	commandUseCase := vehicleBiz.NewCommandUseCase(vehicleRepo, mqttClient, logger)
	recordQueryUseCase := vehicleBiz.NewRecordQueryUseCase(vehicleRepo)

	vehicleSvc := vehicleService.NewVehicleService(
		entryExitUseCase,
		deviceUseCase,
		vehicleQueryUseCase,
		commandUseCase,
		recordQueryUseCase,
		logger,
	)
	logHelper.Info("vehicle service initialized")

	// Initialize Billing Service
	logHelper.Info("initializing billing service...")
	billingDataLayer, cleanup2, err := billingData.NewData(dbClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize billing data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup2()

	billingRepo := billingData.NewBillingRepo(billingDataLayer)
	billingUseCase := billingBiz.NewBillingUseCase(billingRepo, logger)
	billingSvc := billingService.NewBillingService(billingUseCase, logger)
	logHelper.Info("billing service initialized")

	// Initialize Payment Service
	logHelper.Info("initializing payment service...")
	paymentDataLayer, cleanup3, err := paymentData.NewData(dbClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize payment data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup3()

	paymentRepo := paymentData.NewPaymentRepo(paymentDataLayer)
	paymentUseCase := paymentBiz.NewPaymentUseCase(paymentRepo, logger)
	paymentSvc := paymentService.NewPaymentService(paymentUseCase, logger)
	logHelper.Info("payment service initialized")

	// Initialize Admin Service
	logHelper.Info("initializing admin service...")
	adminDataLayer, cleanup4, err := adminData.NewData(dbClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize admin data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup4()

	adminRepo := adminData.NewAdminRepo(adminDataLayer)
	adminUseCase := adminBiz.NewAdminUseCase(adminRepo, logger)
	adminSvc := adminService.NewAdminService(adminUseCase, logger)
	logHelper.Info("admin service initialized")

	// Create servers
	gs := grpc.NewServer(
		grpc.Address(":9000"),
	)

	hs := http.NewServer(
		http.Address(":8000"),
	)

	// Register all services
	vehiclev1.RegisterVehicleServiceServer(gs, vehicleSvc)
	vehiclev1.RegisterVehicleServiceHTTPServer(hs, vehicleSvc)

	billingv1.RegisterBillingServiceServer(gs, billingSvc)
	billingv1.RegisterBillingServiceHTTPServer(hs, billingSvc)

	paymentv1.RegisterPaymentServiceServer(gs, paymentSvc)
	paymentv1.RegisterPaymentServiceHTTPServer(hs, paymentSvc)

	adminv1.RegisterAdminServiceServer(gs, adminSvc)
	adminv1.RegisterAdminServiceHTTPServer(hs, adminSvc)

	logHelper.Info("all services registered")

	// Create application
	app := kratos.New(
		kratos.Name("smart-park-all-in-one"),
		kratos.Version("v1.0.0"),
		kratos.Logger(logger),
		kratos.Server(gs, hs),
	)

	// Start application in goroutine
	go func() {
		if err := app.Run(); err != nil {
			logHelper.Errorf("app run failed: %v", err)
			os.Exit(1)
		}
	}()

	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  All Services Started Successfully!")
	fmt.Println("========================================")
	fmt.Println()
	fmt.Println("Services:")
	fmt.Println("  - Vehicle Service:  http://localhost:8000/api/v1/device")
	fmt.Println("  - Billing Service:  http://localhost:8000/api/v1/billing")
	fmt.Println("  - Payment Service:  http://localhost:8000/api/v1/pay")
	fmt.Println("  - Admin Service:    http://localhost:8000/api/v1/admin")
	fmt.Println()
	fmt.Println("gRPC Server: :9000")
	fmt.Println("HTTP Server: :8000")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop...")
	fmt.Println()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	fmt.Println()
	fmt.Println("Shutting down Smart Park All-in-One Service...")
}

// mockBillingClient is a mock implementation of billing.Client for development.
type mockBillingClient struct{}

func (m *mockBillingClient) CalculateFee(ctx context.Context, recordID string, lotID string, entryTime, exitTime int64, vehicleType string) (*vehicleClient.FeeResult, error) {
	duration := float64(exitTime-entryTime) / 3600.0
	baseAmount := duration * 2.0

	return &vehicleClient.FeeResult{
		BaseAmount:     baseAmount,
		DiscountAmount: 0,
		FinalAmount:    baseAmount,
	}, nil
}

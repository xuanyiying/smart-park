package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis/v8"
	"github.com/xuanyiying/smart-park/pkg/database"

	billingv1 "github.com/xuanyiying/smart-park/api/billing/v1"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	multentdata "github.com/xuanyiying/smart-park/internal/multitenancy/data"
	multent "github.com/xuanyiying/smart-park/internal/multitenancy/data/ent"
	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
	"github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
	"github.com/xuanyiying/smart-park/internal/vehicle/data"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
	"github.com/xuanyiying/smart-park/internal/vehicle/device"
	"github.com/xuanyiying/smart-park/internal/vehicle/service"
	"github.com/xuanyiying/smart-park/pkg/config"
	"github.com/xuanyiying/smart-park/pkg/lock"
	"github.com/xuanyiying/smart-park/pkg/metrics"
	tenantpkg "github.com/xuanyiying/smart-park/pkg/tenant"
	"github.com/xuanyiying/smart-park/pkg/trace"
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

// vehicleRepoStateReader adapts the vehicle repository to the contract the device package
// needs, so adapters can resolve heartbeat-derived device status without depending on the
// business layer (which would create an import cycle).
type vehicleRepoStateReader struct {
	repo biz.VehicleRepo
}

func (r vehicleRepoStateReader) DeviceState(ctx context.Context, deviceID string) (string, *time.Time, error) {
	device, err := r.repo.GetDeviceByCode(ctx, deviceID)
	if err != nil {
		return "", nil, err
	}
	if device == nil {
		return "", nil, fmt.Errorf("device not found: %s", deviceID)
	}
	return device.Status, device.LastHeartbeat, nil
}

// messageOverrideKeys maps config message keys to the built-in message fields they
// override. Keys not listed here are ignored so a typo cannot silently break the
// default copy.
var messageOverrideKeys = map[string]func(cfg *biz.Config, text string){
	"welcome":             func(cfg *biz.Config, text string) { cfg.Messages.Welcome = text },
	"monthly_welcome":     func(cfg *biz.Config, text string) { cfg.Messages.MonthlyWelcome = text },
	"vip_welcome":         func(cfg *biz.Config, text string) { cfg.Messages.VIPWelcome = text },
	"duplicate_entry":     func(cfg *biz.Config, text string) { cfg.Messages.DuplicateEntry = text },
	"duplicate_exit":      func(cfg *biz.Config, text string) { cfg.Messages.DuplicateExit = text },
	"no_entry_record":     func(cfg *biz.Config, text string) { cfg.Messages.NoEntryRecord = text },
	"please_pay":          func(cfg *biz.Config, text string) { cfg.Messages.PleasePay = text },
	"free_pass":           func(cfg *biz.Config, text string) { cfg.Messages.FreePass = text },
	"validation_error":    func(cfg *biz.Config, text string) { cfg.Messages.ValidationError = text },
	"system_error":        func(cfg *biz.Config, text string) { cfg.Messages.SystemError = text },
	"fallback_mode":       func(cfg *biz.Config, text string) { cfg.Messages.FallbackMode = text },
	"billing_unavailable": func(cfg *biz.Config, text string) { cfg.Messages.BillingUnavailable = text },
	"lot_full":            func(cfg *biz.Config, text string) { cfg.Messages.LotFull = text },
	"blacklisted":         func(cfg *biz.Config, text string) { cfg.Messages.Blacklisted = text },
}

// applyMessageOverrides overlays operator-configured display messages onto the
// defaults, so on-site copy can be tuned without a rebuild.
func applyMessageOverrides(cfg *biz.Config, overrides map[string]string) {
	for key, apply := range messageOverrideKeys {
		if text, ok := overrides[key]; ok && text != "" {
			apply(cfg, text)
		}
	}
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
	// context. Device faults, firmware and manufacturers stay global by design;
	// everything describing a tenant's own estate is filtered automatically.
	tenantpkg.ApplyTenantScoping(dbClient, data.TenantScopes(), data.TenantTypes())

	// Run migrations
	if err := dbClient.Schema.Create(context.Background()); err != nil {
		logHelper.Errorf("failed to migrate database: %v", err)
		os.Exit(1)
	}

	// Initialize data layer
	dataLayer, cleanup, err := data.NewData(dbClient, cfg.Database.Source, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	// Initialize repositories
	vehicleRepo := data.NewVehicleRepo(dataLayer)

	// Seed device data for the configured parking lot, if any.
	//
	// Seeding used to attach fixed placeholder devices to a hard-coded lot id on every boot. Now an
	// operator opts in by setting entry_exit.seed_lot_id to the target lot; production
	// deployments register devices through the device management API instead.
	seedLotID := uuid.Nil
	if cfg.EntryExit.SeedLotID != "" {
		parsed, err := uuid.Parse(cfg.EntryExit.SeedLotID)
		if err != nil {
			logHelper.Errorf("invalid entry_exit.seed_lot_id %q: %v", cfg.EntryExit.SeedLotID, err)
			os.Exit(1)
		}
		seedLotID = parsed
	}
	if err := vehicleRepo.SeedData(context.Background(), seedLotID); err != nil {
		if errors.Is(err, data.ErrSeedLotNotConfigured) {
			logHelper.Info("device seeding skipped: no entry_exit.seed_lot_id configured")
		} else {
			logHelper.Errorf("failed to seed device data: %v", err)
			// Don't exit, just log the error
		}
	}

	// Initialize MQTT client.
	//
	// Gate commands are delivered over MQTT, so an unconfigured broker used to silently
	// install a mock client that reported every command as "delivered" without sending
	// anything. The vehicle service is the component that physically opens barriers; if it
	// cannot talk to devices it must not start, rather than pretending to operate a car
	// park that nobody can enter or leave.
	if cfg.MQTT.Broker == "" {
		logHelper.Error("mqtt broker is not configured: gate commands would never reach devices")
		os.Exit(1)
	}

	mqttCfg := &mqtt.Config{
		Broker:        cfg.MQTT.Broker,
		Port:          cfg.MQTT.Port,
		ClientID:      cfg.MQTT.ClientID,
		Username:      cfg.MQTT.Username,
		Password:      cfg.MQTT.Password,
		TLS:           cfg.MQTT.TLS,
		TLSSkipVerify: cfg.MQTT.TLSSkipVerify,
	}

	var mqttClient mqtt.Client = mqtt.NewMQTTClient(mqttCfg)
	if err := mqttClient.Connect(); err != nil {
		logHelper.Errorf("failed to connect MQTT broker at %s: %v", cfg.MQTT.Broker, err)
		os.Exit(1)
	}
	logHelper.Info("mqtt client connected successfully")
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

	// Entry/exit tunables (lock TTL, plate confidence threshold, device online
	// threshold) come from configuration so operations can adjust them without a rebuild.
	entryExitConfig := biz.DefaultConfig()
	if d, err := time.ParseDuration(cfg.EntryExit.LockTTL); err == nil && d > 0 {
		entryExitConfig.LockTTL = d
	}
	if d, err := time.ParseDuration(cfg.EntryExit.DeviceOnlineThreshold); err == nil && d > 0 {
		entryExitConfig.DeviceOnlineThreshold = d
	}
	if cfg.EntryExit.MinConfidence > 0 {
		entryExitConfig.MinConfidence = cfg.EntryExit.MinConfidence
	}
	applyMessageOverrides(entryExitConfig, cfg.EntryExit.Messages)

	// Initialize device adapter factory.
	//
	// Adapters publish real commands over MQTT and derive device state from heartbeats,
	// so both dependencies are injected here. Without a transport the adapters refuse to
	// act instead of reporting success for commands that were never sent.
	adapterFactory := device.NewAdapterFactory(
		device.NewMQTTCommandTransport(mqttClient),
		device.NewRegistryStatusProvider(vehicleRepoStateReader{repo: vehicleRepo}, entryExitConfig.DeviceOnlineThreshold, time.Now),
	)

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
	entryExitUseCase := biz.NewEntryExitUseCase(vehicleRepo, billingClient, mqttClient, lockRepo, adapterFactory, entryExitConfig, logger)
	deviceUseCase := biz.NewDeviceUseCase(vehicleRepo, adapterFactory, mqttClient, entryExitConfig, logger)
	manufacturerUseCase := biz.NewManufacturerUseCase(vehicleRepo, logger)
	firmwareUseCase := biz.NewFirmwareUseCase(vehicleRepo, logger)
	devicePerformanceUseCase := biz.NewDevicePerformanceUseCase(vehicleRepo, logger)
	deviceFaultUseCase := biz.NewDeviceFaultUseCase(vehicleRepo, logger)
	deviceStatsUseCase := biz.NewDeviceStatsUseCase(vehicleRepo, logger)
	vehicleQueryUseCase := biz.NewVehicleQueryUseCase(vehicleRepo, logger)
	commandUseCase := biz.NewCommandUseCase(vehicleRepo, mqttClient, logger)
	recordQueryUseCase := biz.NewRecordQueryUseCase(vehicleRepo)
	blacklistUseCase := biz.NewBlacklistUseCase(vehicleRepo, logger)

	// Initialize gRPC service
	vehicleSvc := service.NewVehicleService(entryExitUseCase, deviceUseCase, manufacturerUseCase, firmwareUseCase, devicePerformanceUseCase, deviceFaultUseCase, deviceStatsUseCase, vehicleQueryUseCase, commandUseCase, recordQueryUseCase, blacklistUseCase, logger)

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

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9001"),
		grpc.Middleware(tenantMW),
	)

	// Create HTTP server
	hs := http.NewServer(
		http.Address(":8001"),
		http.Middleware(tenantMW),
	)

	// Register services
	v1.RegisterVehicleServiceServer(gs, vehicleSvc)
	v1.RegisterVehicleServiceHTTPServer(hs, vehicleSvc)

	// Register Prometheus metrics endpoint
	hs.HandlePrefix("/metrics", metrics.NewHandler())

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

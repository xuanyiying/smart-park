package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/xuanyiying/smart-park/pkg/database"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	vehiclev1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/payment/alipay"
	"github.com/xuanyiying/smart-park/internal/payment/biz"
	"github.com/xuanyiying/smart-park/internal/payment/data"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent"
	"github.com/xuanyiying/smart-park/internal/payment/service"
	"github.com/xuanyiying/smart-park/internal/payment/wechat"
	"github.com/xuanyiying/smart-park/pkg/config"
	"github.com/xuanyiying/smart-park/pkg/metrics"
	"github.com/xuanyiying/smart-park/pkg/middleware"
	"github.com/xuanyiying/smart-park/pkg/outbox"
	"github.com/xuanyiying/smart-park/pkg/trace"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/payment.yaml", "config path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name("payment"),
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
		Enabled:     true,
		ServiceName: cfg.Otel.ServiceName,
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
	orderRepo := data.NewOrderRepo(dataLayer)
	reconciliationRepo := data.NewReconciliationRepo(dataLayer)

	// Initialize payment clients.
	//
	// A misconfigured gateway used to be downgraded to a silent "mock" mode, which meant
	// the service happily accepted payments that were never actually collected. Payment is
	// a money-moving dependency, so a broken configuration now fails the boot instead.
	var wechatClient *wechat.Client
	var alipayClient *alipay.Client

	if cfg.Wechat.AppID != "" || cfg.Wechat.MchID != "" {
		wechatCfg := &wechat.Config{
			AppID:          cfg.Wechat.AppID,
			MchID:          cfg.Wechat.MchID,
			APIKey:         cfg.Wechat.APIKey,
			CertSerialNo:   cfg.Wechat.CertSerialNo,
			PrivateKeyPath: cfg.Wechat.PrivateKeyPath,
			PublicKeyPath:  cfg.Wechat.PublicKeyPath,
			APIv3Key:       cfg.Wechat.APIv3Key,
			NotifyURL:      cfg.Wechat.NotifyURL,
		}
		var err error
		wechatClient, err = wechat.NewClient(wechatCfg)
		if err != nil {
			logHelper.Errorf("wechat payment is configured but unusable: %v", err)
			os.Exit(1)
		}
		logHelper.Info("wechat payment client initialized successfully")
	} else {
		logHelper.Warn("wechat payment not configured; WeChat callbacks will be rejected")
	}

	if cfg.Alipay.AppID != "" || cfg.Alipay.PrivateKey != "" {
		alipayCfg := &alipay.Config{
			AppID:           cfg.Alipay.AppID,
			PrivateKey:      cfg.Alipay.PrivateKey,
			AlipayPublicKey: cfg.Alipay.PublicKey,
			NotifyURL:       cfg.Alipay.NotifyURL,
			IsProduction:    cfg.Alipay.IsProduction,
			SignType:        cfg.Alipay.SignType,
		}
		var err error
		alipayClient, err = alipay.NewClient(alipayCfg)
		if err != nil {
			logHelper.Errorf("alipay payment is configured but unusable: %v", err)
			os.Exit(1)
		}
		logHelper.Info("alipay payment client initialized successfully")
	} else {
		logHelper.Warn("alipay payment not configured; Alipay callbacks will be rejected")
	}

	if wechatClient == nil && alipayClient == nil {
		logHelper.Error("no payment gateway is configured; refusing to start a payment service that cannot settle orders")
		os.Exit(1)
	}

	// Initialize payment config
	paymentConfig := &biz.PaymentConfig{
		WechatMchID:            cfg.Wechat.MchID,
		WechatKey:              cfg.Wechat.APIKey,
		WechatAPIv3Key:         cfg.Wechat.APIv3Key,
		WechatPlatformCertPath: cfg.Wechat.PublicKeyPath,
		WechatCertSerialNo:     cfg.Wechat.CertSerialNo,
		AlipayPublicKey:        cfg.Alipay.PublicKey,
		AlipaySignType:         cfg.Alipay.SignType,
	}

	// Initialize vehicle service client for gate control
	vehicleConn, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint("vehicle-svc:9001"))
	if err != nil {
		logHelper.Warnf("failed to connect vehicle service: %v, gate control disabled", err)
	}

	var recordRepo biz.RecordRepo
	var gateClient biz.GateControlService
	if vehicleConn != nil {
		vehicleClient := vehiclev1.NewVehicleServiceClient(vehicleConn)
		recordRepo = biz.NewVehicleRecordRepoAdapter(vehicleClient)
		gateClient = biz.NewGateControlAdapter(vehicleClient)
	} else {
		logHelper.Warn("vehicle service unavailable; paid orders will not open the exit gate automatically")
	}

	// Initialize business logic
	paymentUseCase := biz.NewPaymentUseCase(orderRepo, recordRepo, gateClient, paymentConfig, wechatClient, alipayClient, logger)
	// The reconciliation use case refunds through the real gateway client, never a stub.
	reconciliationUseCase := biz.NewReconciliationUseCase(orderRepo, reconciliationRepo, wechatClient, alipayClient, paymentUseCase, logger)

	// The transactional outbox turns "mark the order paid" and "open the exit
	// gate" into one atomic unit: the order.settled event is written in the same
	// transaction as the order update, and the dispatcher below applies the gate
	// opening with retries. A crash after the commit no longer strands a paid
	// order in front of a closed barrier.
	outboxStore := outbox.NewPostgresStore(dbManager.Primary())
	if err := outboxStore.EnsureSchema(context.Background()); err != nil {
		logHelper.Errorf("failed to prepare outbox table: %v", err)
		os.Exit(1)
	}
	paymentUseCase.SetOutbox(outboxStore)
	go outbox.NewDispatcher(outboxStore, paymentUseCase.HandleOutboxMessage,
		outbox.DispatcherConfig{
			PollInterval:   5 * time.Second,
			BatchSize:      32,
			MaxAttempts:    10,
			InitialBackoff: time.Second,
			MaxBackoff:     time.Minute,
		}, func(msg *outbox.Message, err error) {
			// Claim and handler failures are expected from time to time; log and
			// keep the dispatcher loop alive. The message itself is retried or
			// dead-lettered by the dispatcher.
			if msg != nil {
				logHelper.WithContext(context.Background()).Warnf("outbox delivery failed for %s (type=%s): %v", msg.ID, msg.Type, err)
			} else {
				logHelper.WithContext(context.Background()).Errorf("outbox sweep error: %v", err)
			}
		}).Run(context.Background())

	// Initialize gRPC service
	paymentSvc := service.NewPaymentService(paymentUseCase, reconciliationUseCase, logger)

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9003"),
	)

	// Create HTTP server.
	//
	// CacheRawBody must run before Kratos binds the request body onto the protobuf
	// message: gateway signatures cover the transmitted bytes, and once consumed the body
	// cannot be read again.
	hs := http.NewServer(
		http.Address(":8003"),
		http.Middleware(
			middleware.CacheRawBody(),
		),
	)

	// Register services
	v1.RegisterPaymentServiceServer(gs, paymentSvc)
	v1.RegisterPaymentServiceHTTPServer(hs, paymentSvc)

	// Register Prometheus metrics endpoint
	hs.HandlePrefix("/metrics", metrics.NewHandler())

	// Start the order sweeper.
	//
	// Callbacks alone cannot cover expiry or a lost callback: without this loop, unpaid
	// orders stayed pending forever and gateway-confirmed payments that no callback
	// reported were never recorded. The loop shares the process lifetime and the same
	// conditional-update settlement path as callbacks, so the two cannot double-settle.
	sweepInterval := cfg.Payment.SweepInterval
	if sweepInterval <= 0 {
		sweepInterval = time.Minute
	}
	sweeperStop := make(chan struct{})
	go paymentUseCase.RunOrderSweeper(context.Background(), sweepInterval, sweeperStop)
	defer close(sweeperStop)

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

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

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	vehiclev1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/payment/alipay"
	"github.com/xuanyiying/smart-park/internal/payment/biz"
	"github.com/xuanyiying/smart-park/internal/payment/data"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent"
	"github.com/xuanyiying/smart-park/internal/payment/service"
	"github.com/xuanyiying/smart-park/internal/payment/wechat"
	"github.com/xuanyiying/smart-park/pkg/config"
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
	orderRepo := data.NewOrderRepo(dataLayer)

	// Initialize payment clients
	var wechatClient *wechat.Client
	var alipayClient *alipay.Client

	if cfg.Wechat.AppID != "" && cfg.Wechat.MchID != "" {
		wechatCfg := &wechat.Config{
			AppID:          cfg.Wechat.AppID,
			MchID:          cfg.Wechat.MchID,
			APIKey:         cfg.Wechat.APIKey,
			CertSerialNo:   cfg.Wechat.CertSerialNo,
			PrivateKeyPath: cfg.Wechat.PrivateKeyPath,
			NotifyURL:      cfg.Wechat.NotifyURL,
		}
		var err error
		wechatClient, err = wechat.NewClient(wechatCfg)
		if err != nil {
			logHelper.Errorf("failed to create wechat client: %v", err)
		} else {
			logHelper.Info("wechat payment client initialized successfully")
		}
	} else {
		logHelper.Warn("wechat payment config not provided, using mock")
	}

	if cfg.Alipay.AppID != "" && cfg.Alipay.PrivateKey != "" {
		alipayCfg := &alipay.Config{
			AppID:           cfg.Alipay.AppID,
			PrivateKey:      cfg.Alipay.PrivateKey,
			AlipayPublicKey: cfg.Alipay.PublicKey,
			IsProduction:    cfg.Alipay.IsProduction,
		}
		var err error
		alipayClient, err = alipay.NewClient(alipayCfg)
		if err != nil {
			logHelper.Errorf("failed to create alipay client: %v", err)
		} else {
			logHelper.Info("alipay payment client initialized successfully")
		}
	} else {
		logHelper.Warn("alipay payment config not provided, using mock")
	}

	// Initialize payment config
	paymentConfig := &biz.PaymentConfig{
		WechatMchID:     cfg.Wechat.MchID,
		WechatKey:       cfg.Wechat.APIKey,
		AlipayPublicKey: cfg.Alipay.PublicKey,
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
	}

	// Initialize business logic
	paymentUseCase := biz.NewPaymentUseCase(orderRepo, recordRepo, gateClient, paymentConfig, wechatClient, alipayClient, logger)

	// Initialize gRPC service
	paymentSvc := service.NewPaymentService(paymentUseCase, logger)

	// Create gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9003"),
	)

	// Create HTTP server
	hs := http.NewServer(
		http.Address(":8003"),
	)

	// Register services
	v1.RegisterPaymentServiceServer(gs, paymentSvc)
	v1.RegisterPaymentServiceHTTPServer(hs, paymentSvc)

	// Start application
	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

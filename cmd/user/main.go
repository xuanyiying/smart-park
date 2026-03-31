package main

import (
	"context"
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	paymentv1 "github.com/xuanyiying/smart-park/api/payment/v1"
	v1 "github.com/xuanyiying/smart-park/api/user/v1"
	vehiclev1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/user/biz"
	"github.com/xuanyiying/smart-park/internal/user/client/payment"
	"github.com/xuanyiying/smart-park/internal/user/client/vehicle"
	"github.com/xuanyiying/smart-park/internal/user/data"
	"github.com/xuanyiying/smart-park/internal/user/data/ent"
	"github.com/xuanyiying/smart-park/internal/user/service"
	userwechat "github.com/xuanyiying/smart-park/internal/user/wechat"
	"github.com/xuanyiying/smart-park/pkg/auth"
	"github.com/xuanyiying/smart-park/pkg/config"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/user.yaml", "config path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name("user"),
		kratos.Logger(logger),
		kratos.Server(gs, hs),
	)
}

func main() {
	flag.Parse()

	logger := log.NewStdLogger(os.Stdout)
	logHelper := log.NewHelper(logger)

	cfg, err := config.Load(flagconf)
	if err != nil {
		logHelper.Errorf("failed to load config: %v", err)
		os.Exit(1)
	}

	dbClient, err := ent.Open("postgres", cfg.Database.Source)
	if err != nil {
		logHelper.Errorf("failed to connect database: %v", err)
		os.Exit(1)
	}
	defer dbClient.Close()

	if err := dbClient.Schema.Create(context.Background()); err != nil {
		logHelper.Errorf("failed to migrate database: %v", err)
		os.Exit(1)
	}

	dataLayer, cleanup, err := data.NewData(dbClient, logger)
	if err != nil {
		logHelper.Errorf("failed to initialize data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	userRepo := data.NewUserRepo(dataLayer)

	var wechatClient *userwechat.Client
	if cfg.Wechat.AppID != "" {
		wechatCfg := &userwechat.Config{
			AppID:     cfg.Wechat.AppID,
			AppSecret: cfg.Wechat.APIKey,
		}
		wechatClient = userwechat.NewClient(wechatCfg)
		logHelper.Info("wechat client initialized successfully")
	} else {
		logHelper.Warn("wechat config not provided, using mock openid")
	}

	// Initialize vehicle service client (required)
	var vehicleClient vehicle.Client
	if cfg.Vehicle == nil || cfg.Vehicle.Endpoint == "" {
		logHelper.Error("vehicle service endpoint is required")
		os.Exit(1)
	}
	conn, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(cfg.Vehicle.Endpoint))
	if err != nil {
		logHelper.Errorf("failed to connect to vehicle service: %v", err)
		os.Exit(1)
	}
	vehicleClient = vehicle.NewClient(vehiclev1.NewVehicleServiceClient(conn))
	logHelper.Info("vehicle client initialized successfully")

	// Initialize payment service client (required)
	var paymentClient payment.Client
	if cfg.Payment == nil || cfg.Payment.Endpoint == "" {
		logHelper.Error("payment service endpoint is required")
		os.Exit(1)
	}
	conn, err = grpc.DialInsecure(context.Background(), grpc.WithEndpoint(cfg.Payment.Endpoint))
	if err != nil {
		logHelper.Errorf("failed to connect to payment service: %v", err)
		os.Exit(1)
	}
	paymentClient = payment.NewClient(paymentv1.NewPaymentServiceClient(conn))
	logHelper.Info("payment client initialized successfully")

	jwtConfig := &auth.JWTConfig{
		PublicKeyPath:  cfg.JWT.PublicKeyPath,
		PrivateKeyPath: cfg.JWT.PrivateKeyPath,
		TokenDuration:  cfg.JWT.TokenDuration,
	}
	jwtManager, err := auth.NewJWTManager(jwtConfig)
	if err != nil {
		logHelper.Errorf("failed to create JWT manager: %v", err)
		os.Exit(1)
	}

	userUseCase := biz.NewUserUseCase(userRepo, vehicleClient, paymentClient, jwtManager, wechatClient, logger)

	userSvc := service.NewUserService(userUseCase)

	gs := grpc.NewServer(
		grpc.Address(":9005"),
	)

	hs := http.NewServer(
		http.Address(":8005"),
	)

	v1.RegisterUserServiceServer(gs, userSvc)
	v1.RegisterUserServiceHTTPServer(hs, userSvc)

	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

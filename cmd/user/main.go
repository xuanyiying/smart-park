package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

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

// mockVehicleClient is a mock implementation of vehicle.Client for testing.
type mockVehicleClient struct{}

func (m *mockVehicleClient) ListParkingRecords(ctx context.Context, plateNumbers []string, page, pageSize int32) (*vehiclev1.ListParkingRecordsData, error) {
	return &vehiclev1.ListParkingRecordsData{
		Records: []*vehiclev1.ParkingRecordInfo{},
		Total:   0,
	}, nil
}

func (m *mockVehicleClient) GetParkingRecord(ctx context.Context, recordID string) (*vehiclev1.ParkingRecordInfo, error) {
	return nil, fmt.Errorf("not implemented")
}

// mockPaymentClient is a mock implementation of payment.Client for testing.
type mockPaymentClient struct{}

func (m *mockPaymentClient) CreatePayment(ctx context.Context, recordID string, amount float64, payMethod string, openID string) (*paymentv1.PaymentData, error) {
	return &paymentv1.PaymentData{
		OrderId: "mock_order_" + recordID,
		Amount:  10.0,
		PayUrl:  "https://payment.example.com/mock",
		QrCode:  "",
	}, nil
}

func (m *mockPaymentClient) GetPaymentStatus(ctx context.Context, orderID string) (*paymentv1.PaymentStatusData, error) {
	return &paymentv1.PaymentStatusData{
		OrderId:   orderID,
		Status:    "pending",
		PayTime:   "",
		PayMethod: "",
	}, nil
}

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

	// Initialize vehicle service client
	var vehicleClient vehicle.Client
	if cfg.Vehicle != nil && cfg.Vehicle.Endpoint != "" {
		conn, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(cfg.Vehicle.Endpoint))
		if err != nil {
			logHelper.Errorf("failed to connect to vehicle service: %v", err)
		} else {
			vehicleClient = vehicle.NewClient(vehiclev1.NewVehicleServiceClient(conn))
			logHelper.Info("vehicle client initialized successfully")
		}
	} else {
		logHelper.Warn("vehicle endpoint not configured")
		vehicleClient = &mockVehicleClient{}
	}

	// Initialize payment service client
	var paymentClient payment.Client
	if cfg.Payment != nil && cfg.Payment.Endpoint != "" {
		conn, err := grpc.DialInsecure(context.Background(), grpc.WithEndpoint(cfg.Payment.Endpoint))
		if err != nil {
			logHelper.Errorf("failed to connect to payment service: %v", err)
		} else {
			paymentClient = payment.NewClient(paymentv1.NewPaymentServiceClient(conn))
			logHelper.Info("payment client initialized successfully")
		}
	} else {
		logHelper.Warn("payment endpoint not configured")
		paymentClient = &mockPaymentClient{}
	}

	jwtConfig := &auth.JWTConfig{
		SecretKey:     cfg.JWT.Secret,
		TokenDuration: time.Duration(cfg.JWT.Expiry) * time.Hour,
	}
	jwtManager := auth.NewJWTManager(jwtConfig)

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

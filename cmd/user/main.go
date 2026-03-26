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

	v1 "github.com/xuanyiying/smart-park/api/user/v1"
	"github.com/xuanyiying/smart-park/internal/user/biz"
	"github.com/xuanyiying/smart-park/internal/user/data"
	"github.com/xuanyiying/smart-park/internal/user/data/ent"
	"github.com/xuanyiying/smart-park/internal/user/service"
	"github.com/xuanyiying/smart-park/internal/user/wechat"
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

	var wechatClient *wechat.Client
	if cfg.Wechat.AppID != "" {
		wechatCfg := &wechat.Config{
			AppID:     cfg.Wechat.AppID,
			AppSecret: cfg.Wechat.APIKey,
		}
		wechatClient = wechat.NewClient(wechatCfg)
		logHelper.Info("wechat client initialized successfully")
	} else {
		logHelper.Warn("wechat config not provided, using mock openid")
	}

	jwtConfig := &auth.JWTConfig{
		SecretKey:     cfg.JWT.Secret,
		TokenDuration: time.Duration(cfg.JWT.Expiry) * time.Hour,
	}
	jwtManager := auth.NewJWTManager(jwtConfig)

	userUseCase := biz.NewUserUseCase(userRepo, jwtManager, wechatClient, logger)

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

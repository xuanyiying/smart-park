package main

import (
	"flag"
	"os"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	"github.com/xuanyiying/smart-park/internal/notification/biz"
	"github.com/xuanyiying/smart-park/internal/notification/data"
	"github.com/xuanyiying/smart-park/internal/notification/service"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs/notification.yaml", "config path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name("notification"),
		kratos.Logger(logger),
		kratos.Server(gs, hs),
	)
}

func main() {
	flag.Parse()

	logger := log.NewStdLogger(os.Stdout)
	logHelper := log.NewHelper(logger)

	dataLayer, cleanup, err := data.NewData(logger)
	if err != nil {
		logHelper.Errorf("failed to initialize data layer: %v", err)
		os.Exit(1)
	}
	defer cleanup()

	notificationRepo := data.NewNotificationRepo()

	inAppNotifier := biz.NewInAppNotifier(logger)
	emailNotifier := biz.NewEmailNotifier(logger)
	smsNotifier := biz.NewSMSNotifier(logger)
	wechatNotifier := biz.NewWechatNotifier(logger)

	compositeNotifier := biz.NewCompositeNotifier([]biz.ChannelRoute{
		{
			Types:    []biz.NotificationType{biz.NotificationTypePaymentSuccess, biz.NotificationTypeVehicleEntry, biz.NotificationTypeVehicleExit, biz.NotificationTypeMonthlyExpiry, biz.NotificationTypeSystemAlert},
			Notifier: inAppNotifier,
		},
		{
			Types:    []biz.NotificationType{biz.NotificationTypePaymentSuccess},
			Notifier: emailNotifier,
		},
		{
			Types:    []biz.NotificationType{biz.NotificationTypeSystemAlert},
			Notifier: smsNotifier,
		},
		{
			Types:    []biz.NotificationType{biz.NotificationTypePaymentSuccess, biz.NotificationTypeMonthlyExpiry},
			Notifier: wechatNotifier,
		},
	}, logger)

	notificationUseCase := biz.NewNotificationUseCase(notificationRepo, compositeNotifier, logger)

	_ = service.NewNotificationService(notificationUseCase, logger)

	_ = dataLayer

	gs := grpc.NewServer(
		grpc.Address(":9008"),
	)

	hs := http.NewServer(
		http.Address(":8008"),
	)

	app := newApp(logger, gs, hs)
	if err := app.Run(); err != nil {
		logHelper.Error(err)
	}
}

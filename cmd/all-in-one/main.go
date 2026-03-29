package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/xuanyiying/smart-park/internal/admin/service"
	billingService "github.com/xuanyiying/smart-park/internal/billing/service"
	paymentService "github.com/xuanyiying/smart-park/internal/payment/service"
	vehicleService "github.com/xuanyiying/smart-park/internal/vehicle/service"
)

var (
	flagconf string
)

func init() {
	flag.StringVar(&flagconf, "conf", "./configs", "config path, eg: -conf config.yaml")
}

func main() {
	flag.Parse()

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service", "smart-park-all-in-one",
	)

	app := kratos.New(
		kratos.Name("smart-park"),
		kratos.Version("v1.0.0"),
		kratos.Logger(logger),
		kratos.Server(
			// Vehicle Service
			grpc.NewServer(),
			http.NewServer(),
		),
	)

	// Register all services
	// Vehicle Service
	vehicleSvc := vehicleService.NewVehicleService()
	// Billing Service
	billingSvc := billingService.NewBillingService()
	// Payment Service
	paymentSvc := paymentService.NewPaymentService()
	// Admin Service
	adminSvc := service.NewAdminService()

	// Register services to servers
	// Note: This is a simplified version. You'll need to properly register
	// all services to the gRPC and HTTP servers

	fmt.Println("Starting Smart Park All-in-One Service...")
	fmt.Println("Services included:")
	fmt.Println("  - Vehicle Service")
	fmt.Println("  - Billing Service")
	fmt.Println("  - Payment Service")
	fmt.Println("  - Admin Service")

	if err := app.Run(); err != nil {
		log.NewHelper(logger).Fatalf("app run failed: %v", err)
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c

	fmt.Println("Shutting down Smart Park All-in-One Service...")
}

// Package main provides the device service main entry point
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/registry"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	consul "github.com/go-kratos/kratos/contrib/registry/consul/v2"
	v1 "github.com/xuanyiying/smart-park/api/device/v1"
	"github.com/xuanyiying/smart-park/internal/device/service"
)

var (
	// Name is the service name
	Name = "device"
	// Version is the service version
	Version = "1.0.0"
	// flagConf is the config file path
	flagConf string
)

func init() {
	flag.StringVar(&flagConf, "conf", "configs/device.yaml", "config file path")
}

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.Parse()

	// Load config
	c := config.New(
		config.WithSource(
			file.NewSource(flagConf),
		),
	)
	if err := c.Load(); err != nil {
		panic(err)
	}

	// Setup logger
	logger := log.NewStdLogger(os.Stdout)

	// Setup registry
	r := consul.New(func(options *consul.Options) {
		options.Address = "localhost:8500"
		options.Schema = "http"
	})

	// Setup gRPC server
	gs := grpc.NewServer(
		grpc.Address(":9000"),
		grpc.Middleware(
			recovery.Recovery(),
		),
	)

	// Setup HTTP server
	hs := http.NewServer(
		http.Address(":8000"),
		http.Middleware(
			recovery.Recovery(),
		),
	)

	// Setup wire
	svc := initService(logger)

	// Register gRPC service
	v1.RegisterDeviceServiceServer(gs, svc)
	// Register HTTP service
	v1.RegisterDeviceServiceHTTPServer(hs, svc)

	// Register service to registry
	app := newApp(logger, gs, hs)

	// Start service
	if err := app.Run(); err != nil {
		panic(err)
	}
}

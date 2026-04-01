//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/wire"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
	"github.com/xuanyiying/smart-park/internal/conf"
	"github.com/xuanyiying/smart-park/internal/payment/biz"
	"github.com/xuanyiying/smart-park/internal/payment/data"
	"github.com/xuanyiying/smart-park/internal/payment/data/ent"
	"github.com/xuanyiying/smart-park/internal/payment/service"
)

// initApp initializes the application with wire.
func initApp(entClient *ent.Client, conf *conf.Config, logger log.Logger) (*kratos.App, func(), error) {
	wire.Build(
		// Data layer
		data.ProviderSet,

		// Business layer
		biz.ProviderSet,

		// Service layer
		service.ProviderSet,

		// gRPC and HTTP servers
		grpc.NewServer,
		http.NewServer,

		// Service registration
		wire.Bind(new(v1.PaymentServiceServer), new(*service.PaymentService)),

		// App
		newApp,
	)
	return nil, nil, nil
}

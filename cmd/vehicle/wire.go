//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/wire"

	billingv1 "github.com/xuanyiying/smart-park/api/billing/v1"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
	"github.com/xuanyiying/smart-park/internal/vehicle/biz"
	"github.com/xuanyiying/smart-park/internal/vehicle/client"
	"github.com/xuanyiying/smart-park/internal/vehicle/data"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/ent"
	"github.com/xuanyiying/smart-park/internal/vehicle/service"
)

// initApp initializes the application with wire.
func initApp(entClient *ent.Client, billingClient billingv1.BillingServiceClient, logger log.Logger) (*app, func(), error) {
	wire.Build(
		// Client layer
		client.ProviderSet,

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
		wire.Bind(new(v1.VehicleServiceServer), new(*service.VehicleService)),

		// App
		newApp,
	)
	return nil, nil, nil
}

type app = struct {
	*grpc.Server
	*http.Server
}

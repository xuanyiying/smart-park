//go:build wireinject
// +build wireinject

package main

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/google/wire"

	v1 "github.com/xuanyiying/smart-park/api/multitenancy/v1"
	"github.com/xuanyiying/smart-park/internal/multitenancy/biz"
	"github.com/xuanyiying/smart-park/internal/multitenancy/data"
	"github.com/xuanyiying/smart-park/internal/multitenancy/data/ent"
	"github.com/xuanyiying/smart-park/internal/multitenancy/service"
)

// initApp initializes the application with wire.
func initApp(entClient *ent.Client, logger log.Logger) (*app, func(), error) {
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
		wire.Bind(new(v1.TenantServiceServer), new(*service.TenantService)),

		// App
		newApp,
	)
	return nil, nil, nil
}

type app = struct {
	*grpc.Server
	*http.Server
}

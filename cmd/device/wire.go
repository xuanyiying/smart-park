//go:build wireinject
// +build wireinject

package main

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/xuanyiying/smart-park/internal/device/biz"
	"github.com/xuanyiying/smart-park/internal/device/data"
	"github.com/xuanyiying/smart-park/internal/device/data/ent"
	"github.com/xuanyiying/smart-park/internal/device/service"
	"github.com/google/wire"
)

// initService initializes the device service
func initService(logger log.Logger) *service.DeviceService {
	wire.Build(
		provideDB,
		data.NewDeviceRepo,
		data.NewMonitoringRepo,
		biz.NewDeviceUseCase,
		biz.NewMonitoringUseCase,
		service.NewDeviceService,
	)
	return &service.DeviceService{}
}

// provideDB provides the database connection
func provideDB() *ent.Client {
	// TODO: Use a real database in production
	client, err := ent.Open("sqlite3", "file::memory:?cache=shared")
	if err != nil {
		panic(err)
	}

	// Run migrations
	if err := client.Schema.Create(context.Background()); err != nil {
		panic(err)
	}

	return client
}

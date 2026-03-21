// Package biz provides business logic for the admin service.
package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is the provider set for biz layer.
var ProviderSet = wire.NewSet(
	NewAdminUseCase,
	NewLogger,
)

// NewLogger creates a new logger helper.
func NewLogger(logger log.Logger) *log.Helper {
	return log.NewHelper(logger)
}

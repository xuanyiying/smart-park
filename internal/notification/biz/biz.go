package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	NewNotificationUseCase,
	NewInAppNotifier,
	NewCompositeNotifier,
	NewLogger,
)

func NewLogger(logger log.Logger) *log.Helper {
	return log.NewHelper(logger)
}

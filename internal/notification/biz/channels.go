package biz

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

type InAppNotifier struct {
	log *log.Helper
}

func NewInAppNotifier(logger log.Logger) *InAppNotifier {
	return &InAppNotifier{
		log: log.NewHelper(logger),
	}
}

func (n *InAppNotifier) Send(ctx context.Context, notification *Notification) error {
	n.log.WithContext(ctx).Infow("msg", "in-app notification sent",
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", notification.Recipient,
	)
	return nil
}

type EmailNotifier struct {
	log *log.Helper
}

func NewEmailNotifier(logger log.Logger) *EmailNotifier {
	return &EmailNotifier{
		log: log.NewHelper(logger),
	}
}

func (n *EmailNotifier) Send(ctx context.Context, notification *Notification) error {
	n.log.WithContext(ctx).Infow("msg", "email notification queued",
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", notification.Recipient,
	)
	return nil
}

type SMSNotifier struct {
	log *log.Helper
}

func NewSMSNotifier(logger log.Logger) *SMSNotifier {
	return &SMSNotifier{
		log: log.NewHelper(logger),
	}
}

func (n *SMSNotifier) Send(ctx context.Context, notification *Notification) error {
	n.log.WithContext(ctx).Infow("msg", "sms notification queued",
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", notification.Recipient,
	)
	return nil
}

type WechatNotifier struct {
	log *log.Helper
}

func NewWechatNotifier(logger log.Logger) *WechatNotifier {
	return &WechatNotifier{
		log: log.NewHelper(logger),
	}
}

func (n *WechatNotifier) Send(ctx context.Context, notification *Notification) error {
	n.log.WithContext(ctx).Infow("msg", "wechat notification queued",
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", notification.Recipient,
	)
	return nil
}

type ChannelRoute struct {
	Types    []NotificationType
	Notifier Notifier
}

type CompositeNotifier struct {
	routes []ChannelRoute
	mu     sync.RWMutex
	log    *log.Helper
}

func NewCompositeNotifier(routes []ChannelRoute, logger log.Logger) *CompositeNotifier {
	return &CompositeNotifier{
		routes: routes,
		log:    log.NewHelper(logger),
	}
}

func (cn *CompositeNotifier) Send(ctx context.Context, notification *Notification) error {
	cn.mu.RLock()
	defer cn.mu.RUnlock()

	var lastErr error
	sent := false

	for _, route := range cn.routes {
		matched := false
		for _, t := range route.Types {
			if t == notification.Type {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		if err := route.Notifier.Send(ctx, notification); err != nil {
			cn.log.WithContext(ctx).Warnf("failed to send notification via channel: %v", err)
			lastErr = err
		} else {
			sent = true
		}
	}

	if !sent && lastErr != nil {
		return fmt.Errorf("all notification channels failed: %w", lastErr)
	}

	return nil
}

func (cn *CompositeNotifier) AddRoute(route ChannelRoute) {
	cn.mu.Lock()
	defer cn.mu.Unlock()
	cn.routes = append(cn.routes, route)
}

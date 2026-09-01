package biz

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-kratos/kratos/v2/log"
)

// InAppNotifier is a no-op delivery step: in-app notifications are persisted by
// the use case (NotificationUseCase.SendNotification calls the repo), so the
// "channel" has nothing more to deliver.
type InAppNotifier struct {
	log *log.Helper
}

func NewInAppNotifier(logger log.Logger) *InAppNotifier {
	return &InAppNotifier{
		log: log.NewHelper(logger),
	}
}

func (n *InAppNotifier) Send(ctx context.Context, notification *Notification) error {
	n.log.WithContext(ctx).Infow("msg", "in-app notification persisted",
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", notification.Recipient,
	)
	return nil
}

// EmailNotifier delivers notifications over SMTP.
//
// Without credentials it returns ErrChannelNotConfigured instead of logging
// "queued" and pretending the mail went out: the caller can then record a real
// failure, and a misconfigured deployment is visible on the very first send.
type EmailNotifier struct {
	config  *EmailConfig
	resolve RecipientResolver
	log     *log.Helper
}

func NewEmailNotifier(config *EmailConfig, resolve RecipientResolver, logger log.Logger) *EmailNotifier {
	return &EmailNotifier{
		config:  config,
		resolve: resolve,
		log:     log.NewHelper(logger),
	}
}

func (n *EmailNotifier) Send(ctx context.Context, notification *Notification) error {
	if !n.config.Enabled() {
		return ErrChannelNotConfigured
	}

	to, err := n.resolveRecipient(ctx, notification.Recipient)
	if err != nil {
		return fmt.Errorf("email: resolve recipient %q: %w", notification.Recipient, err)
	}

	message := buildMIMEMessage(n.config.From, to, notification.Title, notification.Content)
	if err := deliverEmail(n.config, to, message); err != nil {
		return fmt.Errorf("email: deliver to %s: %w", to, err)
	}

	n.log.WithContext(ctx).Infow("msg", "email notification sent",
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", to,
	)
	return nil
}

func (n *EmailNotifier) resolveRecipient(ctx context.Context, recipient string) (string, error) {
	if n.resolve == nil {
		return recipient, nil
	}
	return n.resolve(ctx, recipient)
}

// SMSNotifier delivers notifications through a signed SMS gateway request.
type SMSNotifier struct {
	config  *SMSConfig
	resolve RecipientResolver
	log     *log.Helper
}

func NewSMSNotifier(config *SMSConfig, resolve RecipientResolver, logger log.Logger) *SMSNotifier {
	return &SMSNotifier{
		config:  config,
		resolve: resolve,
		log:     log.NewHelper(logger),
	}
}

func (n *SMSNotifier) Send(ctx context.Context, notification *Notification) error {
	if !n.config.Enabled() {
		return ErrChannelNotConfigured
	}

	phone, err := n.resolveRecipient(ctx, notification.Recipient)
	if err != nil {
		return fmt.Errorf("sms: resolve recipient %q: %w", notification.Recipient, err)
	}

	// Template variables must match the template registered with the provider;
	// title/content is the lowest common denominator across templates.
	params := map[string]string{
		"title":   notification.Title,
		"content": notification.Content,
	}
	if err := sendSMS(ctx, n.config, phone, params); err != nil {
		return fmt.Errorf("sms: deliver to %s: %w", phone, err)
	}

	n.log.WithContext(ctx).Infow("msg", "sms notification sent",
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", phone,
	)
	return nil
}

func (n *SMSNotifier) resolveRecipient(ctx context.Context, recipient string) (string, error) {
	if n.resolve == nil {
		return recipient, nil
	}
	return n.resolve(ctx, recipient)
}

// WechatNotifier delivers template messages through the Official Account API.
type WechatNotifier struct {
	config  *WechatConfig
	resolve RecipientResolver
	tokens  *wechatTokenCache
	log     *log.Helper
}

func NewWechatNotifier(config *WechatConfig, resolve RecipientResolver, logger log.Logger) *WechatNotifier {
	return &WechatNotifier{
		config:  config,
		resolve: resolve,
		tokens:  &wechatTokenCache{},
		log:     log.NewHelper(logger),
	}
}

func (n *WechatNotifier) Send(ctx context.Context, notification *Notification) error {
	if !n.config.Enabled() {
		return ErrChannelNotConfigured
	}

	openID, err := n.resolveRecipient(ctx, notification.Recipient)
	if err != nil {
		return fmt.Errorf("wechat: resolve recipient %q: %w", notification.Recipient, err)
	}

	if err := sendWechatTemplate(ctx, n.config, n.tokens, openID, notification.Title, notification.Content); err != nil {
		return fmt.Errorf("wechat: deliver to %s: %w", openID, err)
	}

	n.log.WithContext(ctx).Infow("msg", "wechat notification sent",
		"notification_id", notification.ID,
		"type", notification.Type,
		"recipient", openID,
	)
	return nil
}

func (n *WechatNotifier) resolveRecipient(ctx context.Context, recipient string) (string, error) {
	if n.resolve == nil {
		return recipient, nil
	}
	return n.resolve(ctx, recipient)
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

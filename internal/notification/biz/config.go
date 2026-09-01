package biz

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ErrChannelNotConfigured is returned when a channel is asked to deliver a
// message but has no provider credentials behind it.
//
// The previous implementation logged "queued" and returned nil, so every email,
// SMS and WeChat notification reported success while nothing was ever sent.
// Failing loudly lets the caller record the failure instead of assuming delivery.
var ErrChannelNotConfigured = errors.New("notification channel is not configured")

// EmailConfig carries SMTP credentials for the email channel.
type EmailConfig struct {
	// Host and Port point at the SMTP server. Port 465 implies implicit TLS,
	// port 587 STARTTLS; both are handled, see EmailNotifier.Send.
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	// From is the envelope sender. Some providers reject a sender that does not
	// match the authenticated account, so it is required alongside credentials.
	From string `mapstructure:"from"`
	// UseTLS selects implicit TLS (typically port 465). When false the notifier
	// still upgrades with STARTTLS if the server advertises it.
	UseTLS bool `mapstructure:"use_tls"`
	// InsecureSkipVerify relaxes certificate verification. Intended only for
	// staging relays with self-signed certificates, never for production.
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify"`
}

// Enabled reports whether the email channel has enough configuration to deliver.
func (c *EmailConfig) Enabled() bool {
	return c != nil && c.Host != "" && c.From != "" && c.Username != "" && c.Password != ""
}

// SMSConfig carries the SMS gateway credentials.
//
// The endpoint defaults to the Aliyun SMS gateway; any compatible gateway that
// accepts the same signed request can be used by overriding Endpoint.
type SMSConfig struct {
	Endpoint    string `mapstructure:"endpoint"`
	AccessKeyID string `mapstructure:"access_key_id"`
	AccessKey   string `mapstructure:"access_key"`
	SignName    string `mapstructure:"sign_name"`
	// TemplateCode identifies the approved template. Notifications that need a
	// different template can override it per send via SMSConfig.TemplateCode.
	TemplateCode string `mapstructure:"template_code"`
	Timeout      time.Duration
}

// Enabled reports whether the SMS channel has enough configuration to deliver.
func (c *SMSConfig) Enabled() bool {
	return c != nil && c.AccessKeyID != "" && c.AccessKey != "" && c.SignName != "" && c.TemplateCode != ""
}

// EndpointURL returns the gateway endpoint, defaulting to Aliyun SMS.
func (c *SMSConfig) EndpointURL() string {
	if c.Endpoint != "" {
		return strings.TrimSuffix(c.Endpoint, "/")
	}
	return "https://dysmsapi.aliyuncs.com"
}

// WechatConfig carries the Official Account credentials used for template
// messages.
type WechatConfig struct {
	AppID  string `mapstructure:"app_id"`
	Secret string `mapstructure:"secret"`
	// TemplateID is the approved template message id.
	TemplateID string `mapstructure:"template_id"`
	// API base, overridable for the sandbox environment.
	BaseURL string `mapstructure:"base_url"`
	// TokenTTL caches the access token between sends. WeChat issues tokens valid
	// for two hours; refreshing on every notification would exhaust the quota.
	TokenTTL time.Duration
}

// Enabled reports whether the WeChat channel has enough configuration to deliver.
func (c *WechatConfig) Enabled() bool {
	return c != nil && c.AppID != "" && c.Secret != "" && c.TemplateID != ""
}

// Base returns the WeChat API base URL, defaulting to the production endpoint.
func (c *WechatConfig) Base() string {
	if c.BaseURL != "" {
		return strings.TrimSuffix(c.BaseURL, "/")
	}
	return "https://api.weixin.qq.com"
}

// RecipientResolver maps a business identifier to a channel address.
//
// Notifications are created with whatever identifies the subject: a plate number
// for a monthly pass reminder, an order id for a payment. Those are not
// deliverable addresses, so each channel needs a resolver that turns them into a
// phone number or an OpenID. When no resolver is configured the recipient is
// used verbatim, which is what tests and single-tenant deployments rely on.
type RecipientResolver func(ctx context.Context, recipient string) (string, error)

// ChannelConfig bundles the per-channel provider settings read from the
// notification service's YAML file. Every member is optional: an absent section
// simply disables that channel, and sends through it fail with
// ErrChannelNotConfigured rather than silently succeeding.
type ChannelConfig struct {
	Email  *EmailConfig  `mapstructure:"email"`
	SMS    *SMSConfig    `mapstructure:"sms"`
	Wechat *WechatConfig `mapstructure:"wechat"`
}

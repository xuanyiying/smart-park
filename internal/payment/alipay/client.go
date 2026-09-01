package alipay

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/smartwalle/alipay/v3"
)

type Config struct {
	AppID           string
	PrivateKey      string
	AlipayPublicKey string
	NotifyURL       string
	IsProduction    bool
	// SignType selects the signature algorithm: "RSA" (SHA1) or "RSA2" (SHA256).
	// Defaults to RSA2 when empty. It must match the key configured in the Alipay console.
	SignType string
}

// Validate reports configuration problems before the client is used.
func (c *Config) Validate() error {
	if c.AppID == "" {
		return errors.New("alipay: app_id is required")
	}
	if c.PrivateKey == "" {
		return errors.New("alipay: private_key is required")
	}
	if c.AlipayPublicKey == "" {
		return errors.New("alipay: public_key is required to verify callbacks")
	}
	return nil
}

type Client struct {
	client *alipay.Client
	config *Config
}

func NewClient(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client, err := alipay.New(cfg.AppID, cfg.PrivateKey, cfg.IsProduction)
	if err != nil {
		return nil, fmt.Errorf("failed to create alipay client: %w", err)
	}

	if err := client.LoadAliPayPublicKey(cfg.AlipayPublicKey); err != nil {
		return nil, fmt.Errorf("failed to load alipay public key: %w", err)
	}

	return &Client{
		client: client,
		config: cfg,
	}, nil
}

// NewTestClient builds a Client that can verify callbacks but cannot reach the gateway.
//
// It exists because verifying a callback only needs the Alipay public key, while creating
// a payment requires real merchant key material and a live SDK handshake. Using it in
// production would mean the service cannot issue payments, so every gateway call routed
// through it fails loudly instead of panicking on a nil SDK client.
func NewTestClient(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	// Parse the public key eagerly so that a malformed key surfaces at startup rather
	// than on the first inbound callback.
	if _, err := parsePublicKey(cfg.AlipayPublicKey); err != nil {
		return nil, err
	}
	return &Client{config: cfg}, nil
}

// gateway returns the underlying SDK client, or an error when the client was built
// without merchant credentials.
func (c *Client) gateway() (*alipay.Client, error) {
	if c.client == nil {
		return nil, fmt.Errorf("alipay gateway client is not initialised: merchant private key is missing or invalid")
	}
	return c.client, nil
}

// CreateTradePagePay creates an Alipay page pay. amount is in cents (分).
func (c *Client) CreateTradePagePay(ctx context.Context, orderID string, amount int64, subject string) (string, error) {
	if _, err := c.gateway(); err != nil {
		return "", err
	}

	p := alipay.TradePagePay{}
	p.NotifyURL = c.config.NotifyURL
	p.ReturnURL = ""
	p.Subject = subject
	p.OutTradeNo = orderID
	p.TotalAmount = centsToYuanString(amount)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	url, err := c.client.TradePagePay(p)
	if err != nil {
		return "", fmt.Errorf("failed to create trade page pay: %w", err)
	}

	return url.String(), nil
}

// CreateTradeWapPay creates an Alipay WAP pay. amount is in cents (分).
func (c *Client) CreateTradeWapPay(ctx context.Context, orderID string, amount int64, subject string) (string, error) {
	if _, err := c.gateway(); err != nil {
		return "", err
	}

	p := alipay.TradeWapPay{}
	p.NotifyURL = c.config.NotifyURL
	p.ReturnURL = ""
	p.Subject = subject
	p.OutTradeNo = orderID
	p.TotalAmount = centsToYuanString(amount)
	p.ProductCode = "QUICK_WAP_WAY"

	url, err := c.client.TradeWapPay(p)
	if err != nil {
		return "", fmt.Errorf("failed to create trade wap pay: %w", err)
	}

	return url.String(), nil
}

// CreateTradePreCreate creates an Alipay precreate order. amount is in cents
// (分); the gateway expects a yuan string with two decimals.
func (c *Client) CreateTradePreCreate(ctx context.Context, orderID string, amount int64, subject string) (string, error) {
	if _, err := c.gateway(); err != nil {
		return "", err
	}

	p := alipay.TradePreCreate{}
	p.NotifyURL = c.config.NotifyURL
	p.Subject = subject
	p.OutTradeNo = orderID
	p.TotalAmount = centsToYuanString(amount)

	rsp, err := c.client.TradePreCreate(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to create trade precreate: %w", err)
	}

	if rsp.Code != "10000" {
		return "", fmt.Errorf("alipay error: %s - %s", rsp.Code, rsp.Msg)
	}

	return rsp.QRCode, nil
}

// DecodeNotification verifies the callback signature with the parameter-complete signing
// string (see BuildSignContent) and then decodes it into a typed notification.
func (c *Client) DecodeNotification(ctx context.Context, params url.Values) (*alipay.Notification, error) {
	if _, err := c.gateway(); err != nil {
		return nil, err
	}

	if err := c.VerifyNotification(params); err != nil {
		return nil, err
	}
	notification, err := c.client.DecodeNotification(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to decode notification: %w", err)
	}
	return notification, nil
}

func (c *Client) QueryOrder(ctx context.Context, orderID string) (*alipay.TradeQueryRsp, error) {
	if _, err := c.gateway(); err != nil {
		return nil, err
	}

	p := alipay.TradeQuery{}
	p.OutTradeNo = orderID

	rsp, err := c.client.TradeQuery(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("failed to query order: %w", err)
	}

	return rsp, nil
}

func (c *Client) CloseOrder(ctx context.Context, orderID string) error {
	if _, err := c.gateway(); err != nil {
		return err
	}

	p := alipay.TradeClose{}
	p.OutTradeNo = orderID

	_, err := c.client.TradeClose(ctx, p)
	if err != nil {
		return fmt.Errorf("failed to close order: %w", err)
	}

	return nil
}

// Refund requests a refund for a paid order. amount is in cents (分).
func (c *Client) Refund(ctx context.Context, orderID, refundID string, amount int64) error {
	if _, err := c.gateway(); err != nil {
		return err
	}

	p := alipay.TradeRefund{}
	p.OutTradeNo = orderID
	p.OutRequestNo = refundID
	p.RefundAmount = centsToYuanString(amount)
	p.RefundReason = "用户申请退款"

	rsp, err := c.client.TradeRefund(ctx, p)
	if err != nil {
		return fmt.Errorf("failed to refund: %w", err)
	}

	if rsp.Code != "10000" {
		return fmt.Errorf("alipay refund error: %s - %s", rsp.Code, rsp.Msg)
	}

	return nil
}

// QueryRefund queries the refund status.
func (c *Client) QueryRefund(ctx context.Context, orderID, refundID string) (string, error) {
	if _, err := c.gateway(); err != nil {
		return "", err
	}

	p := alipay.TradeFastPayRefundQuery{}
	p.OutTradeNo = orderID
	p.OutRequestNo = refundID

	rsp, err := c.client.TradeFastPayRefundQuery(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to query refund: %w", err)
	}

	if rsp.Code != "10000" {
		return "", fmt.Errorf("alipay query refund error: %s - %s", rsp.Code, rsp.Msg)
	}

	return rsp.RefundStatus, nil
}

package alipay

import (
	"context"
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
}

type Client struct {
	client *alipay.Client
	config *Config
}

func NewClient(cfg *Config) (*Client, error) {
	var client *alipay.Client
	var err error

	if cfg.IsProduction {
		client, err = alipay.New(cfg.AppID, cfg.PrivateKey, true)
	} else {
		client, err = alipay.New(cfg.AppID, cfg.PrivateKey, false)
	}

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

func (c *Client) CreateTradePagePay(ctx context.Context, orderID string, amount float64, subject string) (string, error) {
	p := alipay.TradePagePay{}
	p.NotifyURL = c.config.NotifyURL
	p.ReturnURL = ""
	p.Subject = subject
	p.OutTradeNo = orderID
	p.TotalAmount = fmt.Sprintf("%.2f", amount)
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	url, err := c.client.TradePagePay(p)
	if err != nil {
		return "", fmt.Errorf("failed to create trade page pay: %w", err)
	}

	return url.String(), nil
}

func (c *Client) CreateTradeWapPay(ctx context.Context, orderID string, amount float64, subject string) (string, error) {
	p := alipay.TradeWapPay{}
	p.NotifyURL = c.config.NotifyURL
	p.ReturnURL = ""
	p.Subject = subject
	p.OutTradeNo = orderID
	p.TotalAmount = fmt.Sprintf("%.2f", amount)
	p.ProductCode = "QUICK_WAP_WAY"

	url, err := c.client.TradeWapPay(p)
	if err != nil {
		return "", fmt.Errorf("failed to create trade wap pay: %w", err)
	}

	return url.String(), nil
}

func (c *Client) CreateTradePreCreate(ctx context.Context, orderID string, amount float64, subject string) (string, error) {
	p := alipay.TradePreCreate{}
	p.NotifyURL = c.config.NotifyURL
	p.Subject = subject
	p.OutTradeNo = orderID
	p.TotalAmount = fmt.Sprintf("%.2f", amount)

	rsp, err := c.client.TradePreCreate(ctx, p)
	if err != nil {
		return "", fmt.Errorf("failed to create trade precreate: %w", err)
	}

	if rsp.Code != "10000" {
		return "", fmt.Errorf("alipay error: %s - %s", rsp.Code, rsp.Msg)
	}

	return rsp.QRCode, nil
}

func (c *Client) VerifyNotification(ctx context.Context, params url.Values) (*alipay.Notification, error) {
	notification, err := c.client.DecodeNotification(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to verify notification: %w", err)
	}
	return notification, nil
}

func (c *Client) QueryOrder(ctx context.Context, orderID string) (*alipay.TradeQueryRsp, error) {
	p := alipay.TradeQuery{}
	p.OutTradeNo = orderID

	rsp, err := c.client.TradeQuery(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("failed to query order: %w", err)
	}

	return rsp, nil
}

func (c *Client) CloseOrder(ctx context.Context, orderID string) error {
	p := alipay.TradeClose{}
	p.OutTradeNo = orderID

	_, err := c.client.TradeClose(ctx, p)
	if err != nil {
		return fmt.Errorf("failed to close order: %w", err)
	}

	return nil
}

// Refund requests a refund for a paid order.
func (c *Client) Refund(ctx context.Context, orderID, refundID string, amount float64) error {
	p := alipay.TradeRefund{}
	p.OutTradeNo = orderID
	p.OutRequestNo = refundID
	p.RefundAmount = fmt.Sprintf("%.2f", amount)
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

package wechat

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
	"github.com/wechatpay-apiv3/wechatpay-go/services/refunddomestic"
)

// PlatformTransaction represents a transaction record from payment platform.
type PlatformTransaction struct {
	TransactionID string
	OrderID       string
	Amount        float64
	PayTime       time.Time
	Status        string
	PayMethod     string
}

type Client struct {
	client     *core.Client
	config     *Config
	privateKey *rsa.PrivateKey
}

func (c *Client) CreateNativePay(ctx context.Context, orderID string, amount int64, description string) (string, error) {
	svc := native.NativeApiService{Client: c.client}

	resp, _, err := svc.Prepay(ctx, native.PrepayRequest{
		Appid:       core.String(c.config.AppID),
		Mchid:       core.String(c.config.MchID),
		Description: core.String(description),
		OutTradeNo:  core.String(orderID),
		NotifyUrl:   core.String(c.config.NotifyURL),
		Amount: &native.Amount{
			Total: core.Int64(amount),
		},
	})

	if err != nil {
		return "", fmt.Errorf("failed to create native pay order: %w", err)
	}

	return *resp.CodeUrl, nil
}

func (c *Client) CreateJSAPIPay(ctx context.Context, orderID string, amount int64, openID string, description string) (map[string]interface{}, error) {
	svc := jsapi.JsapiApiService{Client: c.client}

	resp, _, err := svc.Prepay(ctx, jsapi.PrepayRequest{
		Appid:       core.String(c.config.AppID),
		Mchid:       core.String(c.config.MchID),
		Description: core.String(description),
		OutTradeNo:  core.String(orderID),
		NotifyUrl:   core.String(c.config.NotifyURL),
		Amount: &jsapi.Amount{
			Total: core.Int64(amount),
		},
		Payer: &jsapi.Payer{
			Openid: core.String(openID),
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create jsapi pay order: %w", err)
	}

	timeStamp := fmt.Sprintf("%d", time.Now().Unix())
	nonceStr := generateNonceStr()
	packageStr := "prepay_id=" + *resp.PrepayId

	sign, err := c.signJSAPI(timeStamp, nonceStr, packageStr)
	if err != nil {
		return nil, fmt.Errorf("failed to sign: %w", err)
	}

	return map[string]interface{}{
		"appId":     c.config.AppID,
		"timeStamp": timeStamp,
		"nonceStr":  nonceStr,
		"package":   packageStr,
		"signType":  "RSA",
		"paySign":   sign,
	}, nil
}

func (c *Client) QueryOrder(ctx context.Context, orderID string) (map[string]interface{}, error) {
	svc := native.NativeApiService{Client: c.client}

	order, _, err := svc.QueryOrderByOutTradeNo(ctx, native.QueryOrderByOutTradeNoRequest{
		OutTradeNo: core.String(orderID),
		Mchid:      core.String(c.config.MchID),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to query order: %w", err)
	}

	return map[string]interface{}{
		"out_trade_no": *order.OutTradeNo,
		"trade_state":  *order.TradeState,
	}, nil
}

func (c *Client) CloseOrder(ctx context.Context, orderID string) error {
	svc := native.NativeApiService{Client: c.client}

	_, err := svc.CloseOrder(ctx, native.CloseOrderRequest{
		OutTradeNo: core.String(orderID),
		Mchid:      core.String(c.config.MchID),
	})

	if err != nil {
		return fmt.Errorf("failed to close order: %w", err)
	}

	return nil
}

func generateNonceStr() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (c *Client) signJSAPI(timeStamp, nonceStr, packageStr string) (string, error) {
	message := fmt.Sprintf("%s\n%s\n%s\n%s\n", c.config.AppID, timeStamp, nonceStr, packageStr)

	hashed := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPKCS1v15(rand.Reader, c.privateKey, crypto.SHA256, hashed[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// Refund requests a refund for a paid order.
func (c *Client) Refund(ctx context.Context, orderID, refundID string, totalAmount, refundAmount int64) error {
	svc := refunddomestic.RefundsApiService{Client: c.client}

	_, _, err := svc.Create(ctx, refunddomestic.CreateRequest{
		OutTradeNo:  core.String(orderID),
		OutRefundNo: core.String(refundID),
		Reason:      core.String("用户申请退款"),
		NotifyUrl:   core.String(c.config.NotifyURL + "/refund"),
		Amount: &refunddomestic.AmountReq{
			Total:    core.Int64(totalAmount),
			Refund:   core.Int64(refundAmount),
			Currency: core.String("CNY"),
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	return nil
}

// QueryRefund queries the refund status.
func (c *Client) QueryRefund(ctx context.Context, orderID, refundID string) (string, error) {
	svc := refunddomestic.RefundsApiService{Client: c.client}

	// Use QueryByOutRefundNo - the API only needs out_refund_no
	resp, _, err := svc.QueryByOutRefundNo(ctx, refunddomestic.QueryByOutRefundNoRequest{
		OutRefundNo: core.String(refundID),
	})

	if err != nil {
		return "", fmt.Errorf("failed to query refund: %w", err)
	}

	if resp.Status != nil {
		return string(*resp.Status), nil
	}

	return "", nil
}

// GetTransactions retrieves transaction records for the specified date.
func (c *Client) GetTransactions(ctx context.Context, date string) ([]*PlatformTransaction, error) {
	// In a real implementation, you would call WeChat's bill download API
	// to get the transaction records for the specified date
	// For now, we'll return an empty list as a placeholder
	return []*PlatformTransaction{}, nil
}

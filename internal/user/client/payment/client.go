package payment

import (
	"context"
	"fmt"

	paymentv1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

type Client interface {
	CreatePayment(ctx context.Context, recordID string, amount float64, payMethod string, openID string) (*paymentv1.PaymentData, error)
	GetPaymentStatus(ctx context.Context, orderID string) (*paymentv1.PaymentStatusData, error)
	CreateMonthlyCardPayment(ctx context.Context, plateNumber string, months int32, amount float64, payMethod string, openID string) (*paymentv1.PaymentData, error)
}

type client struct {
	paymentClient paymentv1.PaymentServiceClient
}

func NewClient(paymentClient paymentv1.PaymentServiceClient) Client {
	return &client{paymentClient: paymentClient}
}

func (c *client) CreatePayment(ctx context.Context, recordID string, amount float64, payMethod string, openID string) (*paymentv1.PaymentData, error) {
	resp, err := c.paymentClient.CreatePayment(ctx, &paymentv1.CreatePaymentRequest{
		RecordId:  recordID,
		Amount:    amount,
		PayMethod: payMethod,
		OpenId:    openID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create payment: %w", err)
	}
	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("create payment failed: %s", resp.Message)
	}
	return resp.Data, nil
}

func (c *client) GetPaymentStatus(ctx context.Context, orderID string) (*paymentv1.PaymentStatusData, error) {
	resp, err := c.paymentClient.GetPaymentStatus(ctx, &paymentv1.GetPaymentStatusRequest{
		OrderId: orderID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get payment status: %w", err)
	}
	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("get payment status failed: %s", resp.Message)
	}
	return resp.Data, nil
}

func (c *client) CreateMonthlyCardPayment(ctx context.Context, plateNumber string, months int32, amount float64, payMethod string, openID string) (*paymentv1.PaymentData, error) {
	resp, err := c.paymentClient.CreatePayment(ctx, &paymentv1.CreatePaymentRequest{
		RecordId:  fmt.Sprintf("monthly_%s_%d", plateNumber, months),
		Amount:    amount,
		PayMethod: payMethod,
		OpenId:    openID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create monthly card payment: %w", err)
	}
	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("create monthly card payment failed: %s", resp.Message)
	}
	return resp.Data, nil
}

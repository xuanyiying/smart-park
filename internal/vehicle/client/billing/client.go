// Package billing provides client for billing service.
package billing

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	billingv1 "github.com/xuanyiying/smart-park/api/billing/v1"
)

// Client defines the interface for billing service client.
type Client interface {
	CalculateFee(ctx context.Context, recordID string, lotID string, entryTime, exitTime int64, vehicleType string) (*FeeResult, error)
}

// FeeResult represents the fee calculation result.
type FeeResult struct {
	BaseAmount     float64
	DiscountAmount float64
	FinalAmount    float64
}

// billingClient implements Client interface.
type billingClient struct {
	client billingv1.BillingServiceClient
	log    *log.Helper
}

// NewClient creates a new billing client.
func NewClient(client billingv1.BillingServiceClient, logger log.Logger) Client {
	return &billingClient{
		client: client,
		log:    log.NewHelper(logger),
	}
}

// CalculateFee calculates parking fee via billing service.
func (c *billingClient) CalculateFee(ctx context.Context, recordID string, lotID string, entryTime, exitTime int64, vehicleType string) (*FeeResult, error) {
	resp, err := c.client.CalculateFee(ctx, &billingv1.CalculateFeeRequest{
		RecordId:    recordID,
		LotId:       lotID,
		EntryTime:   entryTime,
		ExitTime:    exitTime,
		VehicleType: vehicleType,
	})
	if err != nil {
		c.log.WithContext(ctx).Errorf("failed to calculate fee: %v", err)
		return nil, err
	}

	return &FeeResult{
		BaseAmount:     resp.Data.BaseAmount,
		DiscountAmount: resp.Data.DiscountAmount,
		FinalAmount:    resp.Data.FinalAmount,
	}, nil
}

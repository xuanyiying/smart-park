package biz

import (
	"context"
	"fmt"

	chargingv1 "github.com/xuanyiying/smart-park/api/charging/v1"
	vehiclev1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

type VehicleRecordClient interface {
	GetParkingRecord(ctx context.Context, recordID string) (*vehiclev1.ParkingRecordInfo, error)
	UpdateRecordStatus(ctx context.Context, recordID, status string) error
}

type vehicleClientAdapter struct {
	client vehiclev1.VehicleServiceClient
}

func (v *vehicleClientAdapter) GetParkingRecord(ctx context.Context, recordID string) (*vehiclev1.ParkingRecordInfo, error) {
	resp, err := v.client.GetParkingRecord(ctx, &vehiclev1.GetParkingRecordRequest{
		RecordId: recordID,
	})
	if err != nil {
		return nil, err
	}
	if resp.Code != 0 {
		return nil, fmt.Errorf("get parking record failed: %s", resp.Message)
	}
	return resp.Data, nil
}

func (v *vehicleClientAdapter) UpdateRecordStatus(ctx context.Context, recordID, status string) error {
	resp, err := v.client.UpdateRecordStatus(ctx, &vehiclev1.UpdateRecordStatusRequest{
		RecordId: recordID,
		Status:   status,
	})
	if err != nil {
		return fmt.Errorf("update record status failed: %w", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("update record status failed: %s", resp.Message)
	}
	return nil
}

type recordRepoAdapter struct {
	client VehicleRecordClient
}

func NewRecordRepoAdapter(client VehicleRecordClient) RecordRepo {
	return &recordRepoAdapter{client: client}
}

func NewVehicleRecordRepoAdapter(vehicleClient vehiclev1.VehicleServiceClient) RecordRepo {
	return &recordRepoAdapter{client: &vehicleClientAdapter{client: vehicleClient}}
}

func (r *recordRepoAdapter) GetRecord(ctx context.Context, recordID string) (*ParkingRecordInfo, error) {
	record, err := r.client.GetParkingRecord(ctx, recordID)
	if err != nil {
		return nil, err
	}

	info := &ParkingRecordInfo{
		ID:           record.RecordId,
		PlateNumber:  record.PlateNumber,
		ExitDeviceID: record.ExitDeviceId,
		LotID:        record.LotId,
		FinalAmount:  record.FinalAmount,
	}

	return info, nil
}

func (r *recordRepoAdapter) UpdateRecordStatus(ctx context.Context, recordID string, status string) error {
	return r.client.UpdateRecordStatus(ctx, recordID, status)
}

type GateControlAdapter struct {
	vehicleClient vehiclev1.VehicleServiceClient
}

func NewGateControlAdapter(vehicleClient vehiclev1.VehicleServiceClient) GateControlService {
	return &GateControlAdapter{vehicleClient: vehicleClient}
}

func (g *GateControlAdapter) OpenGate(ctx context.Context, deviceID string, recordID string) error {
	_, err := g.vehicleClient.SendCommand(ctx, &vehiclev1.SendCommandRequest{
		DeviceId: deviceID,
		Command:  "open_gate",
		Params:   map[string]string{"record_id": recordID},
	})
	if err != nil {
		return fmt.Errorf("failed to open gate: %w", err)
	}
	return nil
}

// ChargingPaymentClient confirms a charging session payment after the order settles.
type ChargingPaymentClient interface {
	ConfirmChargingPayment(ctx context.Context, sessionID, transactionID, paymentMethod string, paidAmount int64) error
}

type chargingClientAdapter struct {
	client chargingv1.ChargingServiceClient
}

// NewChargingPaymentAdapter wraps the charging gRPC client behind the biz interface.
func NewChargingPaymentAdapter(client chargingv1.ChargingServiceClient) ChargingPaymentClient {
	return &chargingClientAdapter{client: client}
}

func (c *chargingClientAdapter) ConfirmChargingPayment(ctx context.Context, sessionID, transactionID, paymentMethod string, paidAmount int64) error {
	resp, err := c.client.ConfirmPayment(ctx, &chargingv1.ConfirmPaymentRequest{
		SessionId:     sessionID,
		TransactionId: transactionID,
		PaymentMethod: paymentMethod,
		PaidAmount:    paidAmount,
	})
	if err != nil {
		return fmt.Errorf("confirm charging payment failed: %w", err)
	}
	if resp.Code != 0 {
		return fmt.Errorf("confirm charging payment failed: %s", resp.Message)
	}
	return nil
}

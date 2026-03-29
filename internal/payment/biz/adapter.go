package biz

import (
	"context"
	"fmt"

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
	if resp.Code != 200 {
		return nil, fmt.Errorf("get parking record failed: %s", resp.Message)
	}
	return resp.Data, nil
}

func (v *vehicleClientAdapter) UpdateRecordStatus(ctx context.Context, recordID, status string) error {
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

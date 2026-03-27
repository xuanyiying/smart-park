package vehicle

import (
	"context"
	"fmt"

	vehiclev1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// Client defines the interface for vehicle service client.
type Client interface {
	ListParkingRecords(ctx context.Context, plateNumbers []string, page, pageSize int32) (*vehiclev1.ListParkingRecordsData, error)
	GetParkingRecord(ctx context.Context, recordID string) (*vehiclev1.ParkingRecordInfo, error)
	GetVehicleInfo(ctx context.Context, plateNumber string) (*vehiclev1.VehicleInfo, error)
}

// client implements Client interface.
type client struct {
	vehicleClient vehiclev1.VehicleServiceClient
}

// NewClient creates a new vehicle client.
func NewClient(vehicleClient vehiclev1.VehicleServiceClient) Client {
	return &client{
		vehicleClient: vehicleClient,
	}
}

// ListParkingRecords retrieves parking records by plate numbers.
func (c *client) ListParkingRecords(ctx context.Context, plateNumbers []string, page, pageSize int32) (*vehiclev1.ListParkingRecordsData, error) {
	resp, err := c.vehicleClient.ListParkingRecords(ctx, &vehiclev1.ListParkingRecordsRequest{
		PlateNumbers: plateNumbers,
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list parking records: %w", err)
	}

	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("list parking records failed: %s", resp.Message)
	}

	return resp.Data, nil
}

// GetParkingRecord retrieves a single parking record by ID.
func (c *client) GetParkingRecord(ctx context.Context, recordID string) (*vehiclev1.ParkingRecordInfo, error) {
	resp, err := c.vehicleClient.GetParkingRecord(ctx, &vehiclev1.GetParkingRecordRequest{
		RecordId: recordID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get parking record: %w", err)
	}

	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("get parking record failed: %s", resp.Message)
	}

	return resp.Data, nil
}

// GetVehicleInfo retrieves vehicle information by plate number.
func (c *client) GetVehicleInfo(ctx context.Context, plateNumber string) (*vehiclev1.VehicleInfo, error) {
	resp, err := c.vehicleClient.GetVehicleInfo(ctx, &vehiclev1.GetVehicleInfoRequest{
		PlateNumber: plateNumber,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle info: %w", err)
	}

	if resp.Code != 0 && resp.Code != 200 {
		return nil, fmt.Errorf("get vehicle info failed: %s", resp.Message)
	}

	return resp.Data, nil
}

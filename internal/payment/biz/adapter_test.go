package biz

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc"

	chargingv1 "github.com/xuanyiying/smart-park/api/charging/v1"
	vehiclev1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// fakeVehicleClient implements the generated client interface, embedding it so
// only the methods under test need behaviour.
type fakeVehicleClient struct {
	vehiclev1.VehicleServiceClient

	record        *vehiclev1.ParkingRecordInfo
	recordErr     error
	recordCode    int32
	recordMessage string

	statusErr  error
	statusCode int32

	commandErr error

	lastStatusRecordID string
	lastStatus         string
	lastCommand        *vehiclev1.SendCommandRequest
}

func (f *fakeVehicleClient) GetParkingRecord(ctx context.Context, in *vehiclev1.GetParkingRecordRequest, opts ...grpc.CallOption) (*vehiclev1.GetParkingRecordResponse, error) {
	if f.recordErr != nil {
		return nil, f.recordErr
	}
	return &vehiclev1.GetParkingRecordResponse{
		Code: f.recordCode, Message: f.recordMessage, Data: f.record,
	}, nil
}

func (f *fakeVehicleClient) UpdateRecordStatus(ctx context.Context, in *vehiclev1.UpdateRecordStatusRequest, opts ...grpc.CallOption) (*vehiclev1.UpdateRecordStatusResponse, error) {
	f.lastStatusRecordID = in.RecordId
	f.lastStatus = in.Status
	if f.statusErr != nil {
		return nil, f.statusErr
	}
	return &vehiclev1.UpdateRecordStatusResponse{Code: f.statusCode}, nil
}

func (f *fakeVehicleClient) SendCommand(ctx context.Context, in *vehiclev1.SendCommandRequest, opts ...grpc.CallOption) (*vehiclev1.SendCommandResponse, error) {
	f.lastCommand = in
	if f.commandErr != nil {
		return nil, f.commandErr
	}
	return &vehiclev1.SendCommandResponse{Code: 0}, nil
}

// TestVehicleRecordRepoAdapter covers the gRPC-to-biz translation, including
// the transport error and business-code failure paths.
func TestVehicleRecordRepoAdapter(t *testing.T) {
	record := &vehiclev1.ParkingRecordInfo{
		RecordId: "rec-1", LotId: "lot-1", PlateNumber: "京A12345",
		ExitDeviceId: "CAM003", FinalAmount: 1500,
	}

	t.Run("get record maps fields", func(t *testing.T) {
		fake := &fakeVehicleClient{record: record}
		repo := NewVehicleRecordRepoAdapter(fake)

		info, err := repo.GetRecord(context.Background(), "rec-1")
		if err != nil {
			t.Fatalf("GetRecord failed: %v", err)
		}
		if info.ID != "rec-1" || info.ExitDeviceID != "CAM003" || info.FinalAmount != 1500 || info.PlateNumber != "京A12345" {
			t.Errorf("mapped record = %+v", info)
		}
	})

	t.Run("transport error surfaces", func(t *testing.T) {
		fake := &fakeVehicleClient{recordErr: errors.New("connection refused")}
		repo := NewVehicleRecordRepoAdapter(fake)

		if _, err := repo.GetRecord(context.Background(), "rec-1"); err == nil {
			t.Error("a transport failure must surface")
		}
	})

	t.Run("business error code surfaces", func(t *testing.T) {
		fake := &fakeVehicleClient{recordCode: 404, recordMessage: "record not found"}
		repo := NewVehicleRecordRepoAdapter(fake)

		if _, err := repo.GetRecord(context.Background(), "rec-1"); err == nil {
			t.Error("a non-zero response code must surface as an error")
		}
	})

	t.Run("update status passes through", func(t *testing.T) {
		fake := &fakeVehicleClient{}
		repo := NewVehicleRecordRepoAdapter(fake)

		if err := repo.UpdateRecordStatus(context.Background(), "rec-1", "paid"); err != nil {
			t.Fatalf("UpdateRecordStatus failed: %v", err)
		}
		if fake.lastStatusRecordID != "rec-1" || fake.lastStatus != "paid" {
			t.Errorf("forwarded status update = %s/%s", fake.lastStatusRecordID, fake.lastStatus)
		}
	})

	t.Run("update status failures surface", func(t *testing.T) {
		transport := &fakeVehicleClient{statusErr: errors.New("unavailable")}
		if err := NewVehicleRecordRepoAdapter(transport).UpdateRecordStatus(context.Background(), "rec-1", "paid"); err == nil {
			t.Error("a transport failure must surface")
		}
		business := &fakeVehicleClient{statusCode: 500}
		if err := NewVehicleRecordRepoAdapter(business).UpdateRecordStatus(context.Background(), "rec-1", "paid"); err == nil {
			t.Error("a business error code must surface")
		}
	})

	t.Run("NewRecordRepoAdapter accepts a custom client", func(t *testing.T) {
		fake := &fakeVehicleRecordClient{record: record}
		repo := NewRecordRepoAdapter(fake)
		info, err := repo.GetRecord(context.Background(), "rec-1")
		if err != nil {
			t.Fatalf("GetRecord failed: %v", err)
		}
		if info.ID != "rec-1" {
			t.Errorf("record id = %q, want rec-1", info.ID)
		}
	})
}

// fakeVehicleRecordClient implements the biz-level VehicleRecordClient.
type fakeVehicleRecordClient struct {
	record    *vehiclev1.ParkingRecordInfo
	recordErr error

	lastRecordID string
	lastStatus   string
	statusErr    error
}

func (f *fakeVehicleRecordClient) GetParkingRecord(ctx context.Context, recordID string) (*vehiclev1.ParkingRecordInfo, error) {
	f.lastRecordID = recordID
	if f.recordErr != nil {
		return nil, f.recordErr
	}
	return f.record, nil
}

func (f *fakeVehicleRecordClient) UpdateRecordStatus(ctx context.Context, recordID, status string) error {
	f.lastRecordID = recordID
	f.lastStatus = status
	return f.statusErr
}

// TestGateControlAdapter covers the barrier release path over gRPC.
func TestGateControlAdapter(t *testing.T) {
	t.Run("open gate sends the command", func(t *testing.T) {
		fake := &fakeVehicleClient{}
		gate := NewGateControlAdapter(fake)

		if err := gate.OpenGate(context.Background(), "CAM003", "rec-1"); err != nil {
			t.Fatalf("OpenGate failed: %v", err)
		}
		if fake.lastCommand.DeviceId != "CAM003" || fake.lastCommand.Command != "open_gate" {
			t.Errorf("command = %+v", fake.lastCommand)
		}
		if fake.lastCommand.Params["record_id"] != "rec-1" {
			t.Errorf("command params = %v", fake.lastCommand.Params)
		}
	})

	t.Run("open gate failure surfaces", func(t *testing.T) {
		fake := &fakeVehicleClient{commandErr: errors.New("device offline")}
		gate := NewGateControlAdapter(fake)

		if err := gate.OpenGate(context.Background(), "CAM003", "rec-1"); err == nil {
			t.Error("a failed gate command must surface")
		}
	})
}

// fakeChargingClient implements the generated charging client interface.
type fakeChargingClient struct {
	chargingv1.ChargingServiceClient

	confirmErr  error
	confirmCode int32
	lastRequest *chargingv1.ConfirmPaymentRequest
}

func (f *fakeChargingClient) ConfirmPayment(ctx context.Context, in *chargingv1.ConfirmPaymentRequest, opts ...grpc.CallOption) (*chargingv1.ConfirmPaymentResponse, error) {
	f.lastRequest = in
	if f.confirmErr != nil {
		return nil, f.confirmErr
	}
	return &chargingv1.ConfirmPaymentResponse{Code: f.confirmCode}, nil
}

// TestChargingPaymentAdapter covers the charging confirmation path.
func TestChargingPaymentAdapter(t *testing.T) {
	t.Run("confirm passes the settlement details", func(t *testing.T) {
		fake := &fakeChargingClient{}
		client := NewChargingPaymentAdapter(fake)

		if err := client.ConfirmChargingPayment(context.Background(), "sess-1", "tx-1", "wechat", 500); err != nil {
			t.Fatalf("ConfirmChargingPayment failed: %v", err)
		}
		if fake.lastRequest.SessionId != "sess-1" || fake.lastRequest.TransactionId != "tx-1" {
			t.Errorf("request = %+v", fake.lastRequest)
		}
		if fake.lastRequest.PaymentMethod != "wechat" || fake.lastRequest.PaidAmount != 500 {
			t.Errorf("payment details = %+v", fake.lastRequest)
		}
	})

	t.Run("transport error surfaces", func(t *testing.T) {
		fake := &fakeChargingClient{confirmErr: errors.New("unavailable")}
		client := NewChargingPaymentAdapter(fake)

		if err := client.ConfirmChargingPayment(context.Background(), "sess-1", "tx-1", "wechat", 500); err == nil {
			t.Error("a transport failure must surface")
		}
	})

	t.Run("business error code surfaces", func(t *testing.T) {
		fake := &fakeChargingClient{confirmCode: 400}
		client := NewChargingPaymentAdapter(fake)

		if err := client.ConfirmChargingPayment(context.Background(), "sess-1", "tx-1", "wechat", 500); err == nil {
			t.Error("a business error code must surface")
		}
	})
}

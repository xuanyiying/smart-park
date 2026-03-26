package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuanyiying/smart-park/api/admin/v1/adminsvc"
	"github.com/xuanyiying/smart-park/api/billing/v1/billingsvc"
	"github.com/xuanyiying/smart-park/api/payment/v1/paymentsvc"
	"github.com/xuanyiying/smart-park/api/vehicle/v1/vehiclesvc"
)

func TestParkingFlow(t *testing.T) {
	ctx := context.Background()

	t.Run("CompleteParkingFlow", func(t *testing.T) {
		plateNumber := "京A12345"
		deviceID := "lane_entry_001"

		t.Log("Step 1: Vehicle Entry")
		entryResp, err := vehiclesvc.NewVehicleServiceClient(connVehicle).Entry(ctx, &vehiclesvc.EntryRequest{
			DeviceId:    deviceID,
			PlateNumber: plateNumber,
			Confidence:  0.95,
		})
		require.NoError(t, err)
		require.NotNil(t, entryResp)
		assert.True(t, entryResp.Data.GateOpen, "Gate should open on entry")
		assert.NotEmpty(t, entryResp.Data.RecordId, "Record ID should be generated")
		recordID := entryResp.Data.RecordId

		t.Log("Step 2: Simulate parking duration")
		time.Sleep(2 * time.Second)

		t.Log("Step 3: Vehicle Exit")
		exitResp, err := vehiclesvc.NewVehicleServiceClient(connVehicle).Exit(ctx, &vehiclesvc.ExitRequest{
			DeviceId:    "lane_exit_001",
			PlateNumber: plateNumber,
			Confidence:  0.93,
		})
		require.NoError(t, err)
		require.NotNil(t, exitResp)
		assert.Equal(t, "unpaid", exitResp.Data.ExitStatus, "Exit status should be unpaid")
		assert.Greater(t, exitResp.Data.FinalAmount, 0.0, "Final amount should be greater than 0")
		finalAmount := exitResp.Data.FinalAmount

		t.Log("Step 4: Create Payment")
		payResp, err := paymentsvc.NewPaymentServiceClient(connPayment).CreatePayment(ctx, &paymentsvc.CreatePaymentRequest{
			RecordId:  recordID,
			Amount:   finalAmount,
			PayMethod: "wechat",
		})
		require.NoError(t, err)
		require.NotNil(t, payResp)
		assert.NotEmpty(t, payResp.Data.OrderId, "Order ID should be generated")
		assert.NotEmpty(t, payResp.Data.PayUrl, "Pay URL should be generated")
		orderID := payResp.Data.OrderId

		t.Log("Step 5: Verify Payment Status (Pending)")
		statusResp, err := paymentsvc.NewPaymentServiceClient(connPayment).GetPaymentStatus(ctx, &paymentsvc.GetPaymentStatusRequest{
			OrderId: orderID,
		})
		require.NoError(t, err)
		require.NotNil(t, statusResp)
		assert.Equal(t, "pending", statusResp.Data.Status, "Payment status should be pending")

		t.Log("Step 6: Simulate Payment Callback (Mock)")
		callbackResp, err := paymentsvc.NewPaymentServiceClient(connPayment).HandleWechatCallback(ctx, &paymentsvc.WechatCallbackRequest{
			TransactionId: "mock_tx_" + orderID,
			OrderId:       orderID,
			TradeState:    "SUCCESS",
		})
		require.NoError(t, err)
		require.NotNil(t, callbackResp)
		assert.True(t, callbackResp.Data.Success, "Callback should be processed successfully")

		t.Log("Step 7: Verify Payment Status (Paid)")
		statusResp, err = paymentsvc.NewPaymentServiceClient(connPayment).GetPaymentStatus(ctx, &paymentsvc.GetPaymentStatusRequest{
			OrderId: orderID,
		})
		require.NoError(t, err)
		require.NotNil(t, statusResp)
		assert.Equal(t, "paid", statusResp.Data.Status, "Payment status should be paid")

		t.Log("Step 8: Verify Admin Can Query Records")
		recordsResp, err := adminsvc.NewAdminServiceClient(connAdmin).GetParkingRecords(ctx, &adminsvc.GetParkingRecordsRequest{
			LotId: 1,
			Page:  1,
		})
		require.NoError(t, err)
		require.NotNil(t, recordsResp)
		assert.GreaterOrEqual(t, recordsResp.Data.Total, 1, "Should have at least 1 record")

		t.Log("Parking flow completed successfully!")
	})
}

func TestBillingCalculation(t *testing.T) {
	ctx := context.Background()

	t.Run("CalculateFee", func(t *testing.T) {
		entryTime := time.Now().Add(-2 * time.Hour).Unix()
		exitTime := time.Now().Unix()

		resp, err := billingsvc.NewBillingServiceClient(connBilling).CalculateFee(ctx, &billingsvc.CalculateFeeRequest{
			RecordId:    "test_record_001",
			LotId:       "test_lot_001",
			VehicleType: "temporary",
			EntryTime:   entryTime,
			ExitTime:    exitTime,
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Greater(t, resp.Data.FinalAmount, 0.0, "Fee should be calculated")
		t.Logf("Calculated fee: %.2f", resp.Data.FinalAmount)
	})
}

func TestDeviceHeartbeat(t *testing.T) {
	ctx := context.Background()

	t.Run("DeviceHeartbeat", func(t *testing.T) {
		deviceID := "device_001"

		resp, err := vehiclesvc.NewVehicleServiceClient(connVehicle).Heartbeat(ctx, &vehiclesvc.HeartbeatRequest{
			DeviceId: deviceID,
			Status:   "online",
		})
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.Equal(t, "active", resp.Data.Status, "Device status should be active")
	})
}

var (
	connVehicle *vehiclesvc.VehicleServiceClient
	connBilling *billingsvc.BillingServiceClient
	connPayment *paymentsvc.PaymentServiceClient
	connAdmin   *adminsvc.AdminServiceClient
)

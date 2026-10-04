package biz

import (
	"context"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type recordingChargingClient struct {
	sessionID      string
	transactionID  string
	paymentMethod  string
	paidAmount     int64
	calls          int
}

func (m *recordingChargingClient) ConfirmChargingPayment(ctx context.Context, sessionID, transactionID, paymentMethod string, paidAmount int64) error {
	m.calls++
	m.sessionID = sessionID
	m.transactionID = transactionID
	m.paymentMethod = paymentMethod
	m.paidAmount = paidAmount
	return nil
}

func newSideEffectsUseCase() (*PaymentUseCase, *MockOrderRepo, *recordingChargingClient) {
	logger := log.NewStdLogger(&discardWriter{})
	mockRepo := NewMockOrderRepo()
	uc := NewPaymentUseCase(mockRepo, NewMockRecordRepo(), NewMockGateControlService(), &PaymentConfig{}, nil, nil, logger)
	charging := &recordingChargingClient{}
	uc.SetChargingClient(charging)
	return uc, mockRepo, charging
}

// TestSideEffectsChargingOrderConfirmsSession 回归断点 C：充电订单结算后
// 必须调用 charging 服务的 ConfirmPayment，而不是走开闸逻辑。
func TestSideEffectsChargingOrderConfirmsSession(t *testing.T) {
	uc, _, charging := newSideEffectsUseCase()

	sessionID := uuid.New()
	order := &Order{
		ID:            uuid.New(),
		RecordID:      sessionID, // 充电单：RecordID 即会话ID
		OrderType:     OrderTypeCharging,
		PayMethod:     "wechat",
		TransactionID: "tx-charging-1",
		PaidAmount:    500,
	}

	require.NoError(t, uc.applyOrderSettledSideEffects(context.Background(), order))
	require.Equal(t, 1, charging.calls)
	require.Equal(t, sessionID.String(), charging.sessionID)
	require.Equal(t, "tx-charging-1", charging.transactionID)
	require.Equal(t, "wechat", charging.paymentMethod)
	require.Equal(t, int64(500), charging.paidAmount)
}

// TestSideEffectsParkingOrderDoesNotTouchCharging 停车单不应触发充电确认。
func TestSideEffectsParkingOrderDoesNotTouchCharging(t *testing.T) {
	uc, _, charging := newSideEffectsUseCase()

	order := &Order{
		ID:        uuid.New(),
		RecordID:  uuid.New(),
		OrderType: OrderTypeParking,
	}

	// 未配置 recordRepo/gateClient 时开闸路径静默跳过，不报错也不调充电
	require.NoError(t, uc.applyOrderSettledSideEffects(context.Background(), order))
	require.Equal(t, 0, charging.calls)
}

type discardWriter struct{}

func (d *discardWriter) Write(p []byte) (int, error) { return len(p), nil }

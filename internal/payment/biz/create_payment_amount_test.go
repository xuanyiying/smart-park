package biz

import (
	"context"
	"os"
	"testing"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

// staticRecordRepo serves a fixed parking record for amount checks.
type staticRecordRepo struct {
	record *ParkingRecordInfo
}

func (s *staticRecordRepo) GetRecord(ctx context.Context, recordID string) (*ParkingRecordInfo, error) {
	return s.record, nil
}

func (s *staticRecordRepo) UpdateRecordStatus(ctx context.Context, recordID, status string) error {
	return nil
}

func findOrderByRecordID(t *testing.T, repo *MockOrderRepo, recordID uuid.UUID) *Order {
	t.Helper()
	for _, order := range repo.Orders {
		if order.RecordID == recordID {
			return order
		}
	}
	return nil
}

// TestCreatePaymentUsesServerAmount 回归断点 B：客户端自报 1 分钱也必须按
// 服务端计费金额（10 元）创建订单。
func TestCreatePaymentUsesServerAmount(t *testing.T) {
	logger := log.NewStdLogger(os.Stderr)
	mockRepo := NewMockOrderRepo()

	recordID := uuid.New()
	recordRepo := &staticRecordRepo{record: &ParkingRecordInfo{
		ID:          recordID.String(),
		FinalAmount: 1000,
	}}
	uc := NewPaymentUseCase(mockRepo, recordRepo, NewMockGateControlService(), &PaymentConfig{}, nil, nil, logger)

	_, _ = uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
		RecordId:  recordID.String(),
		Amount:    1, // 篡改的低价
		PayMethod: "wechat",
	})

	order := findOrderByRecordID(t, mockRepo, recordID)
	require.NotNil(t, order, "order should be created before payment URL generation")
	require.Equal(t, int64(1000), order.FinalAmount, "server-side fee must win over the client amount")
}

// TestCreatePaymentFallsBackWithoutServerAmount 服务端无金额（历史记录等）
// 时保持原行为，按客户端金额创建。
func TestCreatePaymentFallsBackWithoutServerAmount(t *testing.T) {
	logger := log.NewStdLogger(os.Stderr)
	mockRepo := NewMockOrderRepo()

	recordID := uuid.New()
	recordRepo := &staticRecordRepo{record: &ParkingRecordInfo{ID: recordID.String(), FinalAmount: 0}}
	uc := NewPaymentUseCase(mockRepo, recordRepo, NewMockGateControlService(), &PaymentConfig{}, nil, nil, logger)

	_, _ = uc.CreatePayment(context.Background(), &v1.CreatePaymentRequest{
		RecordId:  recordID.String(),
		Amount:    500,
		PayMethod: "wechat",
	})

	order := findOrderByRecordID(t, mockRepo, recordID)
	require.NotNil(t, order)
	require.Equal(t, int64(500), order.FinalAmount)
}

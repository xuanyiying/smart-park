package biz

import (
	"context"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

// MockOrderRepo is a mock implementation of OrderRepo for testing.
type MockOrderRepo struct {
	Orders map[uuid.UUID]*Order
}

func NewMockOrderRepo() *MockOrderRepo {
	return &MockOrderRepo{
		Orders: make(map[uuid.UUID]*Order),
	}
}

func (m *MockOrderRepo) GetOrder(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	return m.Orders[orderID], nil
}

func (m *MockOrderRepo) GetOrderByRecordID(ctx context.Context, recordID uuid.UUID) (*Order, error) {
	for _, order := range m.Orders {
		if order.RecordID == recordID {
			return order, nil
		}
	}
	return nil, nil
}

func (m *MockOrderRepo) GetOrderByTransactionID(ctx context.Context, transactionID string) (*Order, error) {
	for _, order := range m.Orders {
		if order.TransactionID == transactionID {
			return order, nil
		}
	}
	return nil, nil
}

func (m *MockOrderRepo) CreateOrder(ctx context.Context, order *Order) error {
	m.Orders[order.ID] = order
	return nil
}

func (m *MockOrderRepo) UpdateOrder(ctx context.Context, order *Order) error {
	m.Orders[order.ID] = order
	return nil
}

func (m *MockOrderRepo) ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*Order, int64, error) {
	var result []*Order
	for _, order := range m.Orders {
		if order.LotID == lotID {
			result = append(result, order)
		}
	}
	return result, int64(len(result)), nil
}

func TestPaymentUseCase_CreatePayment(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockOrderRepo()

	uc := NewPaymentUseCase(mockRepo, logger)

	req := &v1.CreatePaymentRequest{
		RecordId:  uuid.New().String(),
		Amount:    10.50,
		PayMethod: "wechat",
		OpenId:    "test-open-id",
		NotifyUrl: "http://example.com/notify",
	}

	data, err := uc.CreatePayment(context.Background(), req)
	if err != nil {
		t.Fatalf("CreatePayment failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.Amount != 10.50 {
		t.Errorf("Expected amount 10.50, got %f", data.Amount)
	}

	if data.OrderId == "" {
		t.Error("Expected order ID to be set")
	}
}

func TestPaymentUseCase_GetPaymentStatus(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockOrderRepo()

	orderID := uuid.New()
	payTime := time.Now()
	mockRepo.Orders[orderID] = &Order{
		ID:        orderID,
		RecordID:  uuid.New(),
		LotID:     uuid.New(),
		Status:    "paid",
		PayTime:   &payTime,
		PayMethod: "wechat",
	}

	uc := NewPaymentUseCase(mockRepo, logger)

	data, err := uc.GetPaymentStatus(context.Background(), orderID.String())
	if err != nil {
		t.Fatalf("GetPaymentStatus failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.Status != "paid" {
		t.Errorf("Expected status 'paid', got %s", data.Status)
	}
}

func TestPaymentUseCase_Refund(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockOrderRepo()

	orderID := uuid.New()
	mockRepo.Orders[orderID] = &Order{
		ID:          orderID,
		RecordID:    uuid.New(),
		LotID:       uuid.New(),
		Status:      "paid",
		FinalAmount: 10.00,
	}

	uc := NewPaymentUseCase(mockRepo, logger)

	data, err := uc.Refund(context.Background(), orderID.String(), "Test refund")
	if err != nil {
		t.Fatalf("Refund failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.Status != "success" {
		t.Errorf("Expected status 'success', got %s", data.Status)
	}

	// Verify order status was updated
	if mockRepo.Orders[orderID].Status != "refunded" {
		t.Error("Expected order status to be 'refunded'")
	}
}

func TestPaymentUseCase_HandleWechatCallback(t *testing.T) {
	logger := log.NewStdLogger(nil)
	mockRepo := NewMockOrderRepo()

	transactionID := "wx-transaction-123"
	orderID := uuid.New()
	mockRepo.Orders[orderID] = &Order{
		ID:            orderID,
		RecordID:      uuid.New(),
		LotID:         uuid.New(),
		Status:        "pending",
		TransactionID: transactionID,
	}

	uc := NewPaymentUseCase(mockRepo, logger)

	req := &v1.WechatCallbackRequest{
		ReturnCode:    "SUCCESS",
		ResultCode:    "SUCCESS",
		TransactionId: transactionID,
		OutTradeNo:    orderID.String(),
		TotalFee:      "1000",
	}

	resp, err := uc.HandleWechatCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleWechatCallback failed: %v", err)
	}

	if resp.ReturnCode != "SUCCESS" {
		t.Errorf("Expected return code 'SUCCESS', got %s", resp.ReturnCode)
	}

	// Verify order status was updated
	if mockRepo.Orders[orderID].Status != "paid" {
		t.Error("Expected order status to be 'paid'")
	}
}

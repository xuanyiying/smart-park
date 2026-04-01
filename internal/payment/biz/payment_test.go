package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// MockOrderRepo is a mock implementation of OrderRepo for testing
type MockOrderRepo struct {
	orders map[string]*Order
}

func NewMockOrderRepo() *MockOrderRepo {
	return &MockOrderRepo{
		orders: make(map[string]*Order),
	}
}

func (m *MockOrderRepo) GetOrder(ctx context.Context, orderID uuid.UUID) (*Order, error) {
	order, exists := m.orders[orderID.String()]
	if !exists {
		// Create a default order if not found
		order = &Order{
			ID:            orderID,
			Amount:        10.0,
			FinalAmount:   10.0,
			Status:        string(StatusPending),
			AutoPayAttempts: 0,
		}
		m.orders[orderID.String()] = order
	}
	return order, nil
}

func (m *MockOrderRepo) GetOrderByRecordID(ctx context.Context, recordID uuid.UUID) (*Order, error) {
	return nil, nil
}

func (m *MockOrderRepo) GetOrderByTransactionID(ctx context.Context, transactionID string) (*Order, error) {
	return nil, nil
}

func (m *MockOrderRepo) CreateOrder(ctx context.Context, order *Order) error {
	m.orders[order.ID.String()] = order
	return nil
}

func (m *MockOrderRepo) UpdateOrder(ctx context.Context, order *Order) error {
	m.orders[order.ID.String()] = order
	return nil
}

func (m *MockOrderRepo) ListOrders(ctx context.Context, lotID uuid.UUID, status string, page, pageSize int) ([]*Order, int64, error) {
	return nil, 0, nil
}

// MockRecordRepo is a mock implementation of RecordRepo for testing
type MockRecordRepo struct {}

func (m *MockRecordRepo) GetRecord(ctx context.Context, recordID uuid.UUID) (*ParkingRecord, error) {
	return nil, nil
}

func (m *MockRecordRepo) CreateRecord(ctx context.Context, record *ParkingRecord) error {
	return nil
}

func (m *MockRecordRepo) UpdateRecord(ctx context.Context, record *ParkingRecord) error {
	return nil
}

// MockGateControlService is a mock implementation of GateControlService for testing
type MockGateControlService struct {
	openGateCalled bool
}

func (m *MockGateControlService) OpenGate(ctx context.Context, recordID string) error {
	m.openGateCalled = true
	return nil
}

// MockNotificationService is a mock implementation of NotificationService for testing
type MockNotificationService struct {
	notificationSent bool
}

func (m *MockNotificationService) CreatePaymentNotification(ctx context.Context, userID string, orderID string, amount float64, status string) error {
	m.notificationSent = true
	return nil
}

func TestPaymentUseCase_AutoPay(t *testing.T) {
	mockOrderRepo := NewMockOrderRepo()
	mockRecordRepo := &MockRecordRepo{}
	mockGateControl := &MockGateControlService{}
	mockNotification := &MockNotificationService{}

	paymentUseCase := NewPaymentUseCase(
		mockOrderRepo,
		mockRecordRepo,
		mockGateControl,
		mockNotification,
		nil,
		nil,
		nil,
		nil,
	)

	// Test case 1: Successful auto-pay
	orderID := uuid.New().String()
	userID := uuid.New().String()

	err := paymentUseCase.AutoPay(context.Background(), orderID, userID)
	assert.NoError(t, err)

	// Verify order status was updated
	order, _ := mockOrderRepo.GetOrder(context.Background(), uuid.MustParse(orderID))
	assert.Equal(t, string(StatusPaid), order.Status)
	assert.Equal(t, "auto", order.PayMethod)
	assert.NotEmpty(t, order.TransactionID)
	assert.True(t, mockGateControl.openGateCalled)
	assert.True(t, mockNotification.notificationSent)

	// Test case 2: Order already paid
	err = paymentUseCase.AutoPay(context.Background(), orderID, userID)
	assert.NoError(t, err)

	// Test case 3: Maximum auto-pay attempts reached
	orderID3 := uuid.New().String()
	order3 := &Order{
		ID:            uuid.MustParse(orderID3),
		Amount:        10.0,
		FinalAmount:   10.0,
		Status:        string(StatusPending),
		AutoPayAttempts: 3,
	}
	mockOrderRepo.orders[orderID3] = order3

	err = paymentUseCase.AutoPay(context.Background(), orderID3, userID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "maximum auto-pay attempts reached")
}

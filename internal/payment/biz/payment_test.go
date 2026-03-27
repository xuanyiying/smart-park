package biz

import (
	"context"
	"os"
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

type MockRecordRepo struct{}

func NewMockRecordRepo() *MockRecordRepo {
	return &MockRecordRepo{}
}

func (m *MockRecordRepo) GetRecord(ctx context.Context, recordID string) (*ParkingRecordInfo, error) {
	return nil, nil
}

func (m *MockRecordRepo) UpdateRecordStatus(ctx context.Context, recordID string, status string) error {
	return nil
}

type MockGateControlService struct{}

func NewMockGateControlService() *MockGateControlService {
	return &MockGateControlService{}
}

func (m *MockGateControlService) OpenGate(ctx context.Context, deviceID string, recordID string) error {
	return nil
}

func TestPaymentUseCase_CreatePayment(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockOrderRepo()
	mockRecordRepo := NewMockRecordRepo()
	mockGateSvc := NewMockGateControlService()
	config := &PaymentConfig{}

	uc := NewPaymentUseCase(mockRepo, mockRecordRepo, mockGateSvc, config, nil, nil, logger)

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

func TestPaymentUseCase_CreatePayment_InvalidAmount(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockOrderRepo()
	mockRecordRepo := NewMockRecordRepo()
	mockGateSvc := NewMockGateControlService()
	config := &PaymentConfig{}

	uc := NewPaymentUseCase(mockRepo, mockRecordRepo, mockGateSvc, config, nil, nil, logger)

	req := &v1.CreatePaymentRequest{
		RecordId:  uuid.New().String(),
		Amount:    -10.50,
		PayMethod: "wechat",
	}

	_, err := uc.CreatePayment(context.Background(), req)
	if err == nil {
		t.Error("Expected error for negative amount")
	}
}

func TestPaymentUseCase_CreatePayment_InvalidMethod(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockOrderRepo()
	mockRecordRepo := NewMockRecordRepo()
	mockGateSvc := NewMockGateControlService()
	config := &PaymentConfig{}

	uc := NewPaymentUseCase(mockRepo, mockRecordRepo, mockGateSvc, config, nil, nil, logger)

	req := &v1.CreatePaymentRequest{
		RecordId:  uuid.New().String(),
		Amount:    10.50,
		PayMethod: "invalid",
	}

	_, err := uc.CreatePayment(context.Background(), req)
	if err == nil {
		t.Error("Expected error for invalid payment method")
	}
}

func TestPaymentUseCase_GetPaymentStatus(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockOrderRepo()
	mockRecordRepo := NewMockRecordRepo()
	mockGateSvc := NewMockGateControlService()
	config := &PaymentConfig{}

	orderID := uuid.New()
	payTime := time.Now()
	mockRepo.Orders[orderID] = &Order{
		ID:        orderID,
		RecordID:  uuid.New(),
		LotID:     uuid.New(),
		Status:    string(StatusPaid),
		PayTime:   &payTime,
		PayMethod: string(MethodWechat),
	}

	uc := NewPaymentUseCase(mockRepo, mockRecordRepo, mockGateSvc, config, nil, nil, logger)

	data, err := uc.GetPaymentStatus(context.Background(), orderID.String())
	if err != nil {
		t.Fatalf("GetPaymentStatus failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil response")
	}

	if data.Status != string(StatusPaid) {
		t.Errorf("Expected status 'paid', got %s", data.Status)
	}
}

func TestPaymentUseCase_Refund(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockOrderRepo()
	mockRecordRepo := NewMockRecordRepo()
	mockGateSvc := NewMockGateControlService()
	config := &PaymentConfig{}

	orderID := uuid.New()
	mockRepo.Orders[orderID] = &Order{
		ID:          orderID,
		RecordID:    uuid.New(),
		LotID:       uuid.New(),
		Status:      string(StatusPaid),
		FinalAmount: 10.00,
		PayMethod:   string(MethodWechat),
	}

	uc := NewPaymentUseCase(mockRepo, mockRecordRepo, mockGateSvc, config, nil, nil, logger)

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
	if mockRepo.Orders[orderID].Status != string(StatusRefunded) {
		t.Errorf("Expected order status to be 'refunded', got %s", mockRepo.Orders[orderID].Status)
	}
}

func TestPaymentUseCase_Refund_NotPaid(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockOrderRepo()
	mockRecordRepo := NewMockRecordRepo()
	mockGateSvc := NewMockGateControlService()
	config := &PaymentConfig{}

	orderID := uuid.New()
	mockRepo.Orders[orderID] = &Order{
		ID:          orderID,
		RecordID:    uuid.New(),
		LotID:       uuid.New(),
		Status:      string(StatusPending),
		FinalAmount: 10.00,
	}

	uc := NewPaymentUseCase(mockRepo, mockRecordRepo, mockGateSvc, config, nil, nil, logger)

	data, err := uc.Refund(context.Background(), orderID.String(), "Test refund")
	if err != nil {
		t.Fatalf("Refund failed: %v", err)
	}

	if data.Status != "failed" {
		t.Errorf("Expected status 'failed', got %s", data.Status)
	}
}

func TestPaymentUseCase_HandleWechatCallback(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockOrderRepo()
	mockRecordRepo := NewMockRecordRepo()
	mockGateSvc := NewMockGateControlService()
	config := &PaymentConfig{}

	transactionID := "wx-transaction-123"
	orderID := uuid.New()
	mockRepo.Orders[orderID] = &Order{
		ID:            orderID,
		RecordID:      uuid.New(),
		LotID:         uuid.New(),
		Status:        string(StatusPending),
		TransactionID: transactionID,
	}

	uc := NewPaymentUseCase(mockRepo, mockRecordRepo, mockGateSvc, config, nil, nil, logger)

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
	if mockRepo.Orders[orderID].Status != string(StatusPaid) {
		t.Errorf("Expected order status to be 'paid', got %s", mockRepo.Orders[orderID].Status)
	}
}

func TestPaymentUseCase_HandleAlipayCallback(t *testing.T) {
	logger := log.NewStdLogger(os.Stdout)
	mockRepo := NewMockOrderRepo()
	mockRecordRepo := NewMockRecordRepo()
	mockGateSvc := NewMockGateControlService()
	config := &PaymentConfig{}

	transactionID := "alipay-transaction-123"
	orderID := uuid.New()
	mockRepo.Orders[orderID] = &Order{
		ID:            orderID,
		RecordID:      uuid.New(),
		LotID:         uuid.New(),
		Status:        string(StatusPending),
		TransactionID: transactionID,
	}

	uc := NewPaymentUseCase(mockRepo, mockRecordRepo, mockGateSvc, config, nil, nil, logger)

	req := &v1.AlipayCallbackRequest{
		TradeStatus: "TRADE_SUCCESS",
		TradeNo:     transactionID,
		OutTradeNo:  orderID.String(),
		TotalAmount: "10.00",
	}

	resp, err := uc.HandleAlipayCallback(context.Background(), req)
	if err != nil {
		t.Fatalf("HandleAlipayCallback failed: %v", err)
	}

	if resp.Code != "success" {
		t.Errorf("Expected code 'success', got %s", resp.Code)
	}

	// Verify order status was updated
	if mockRepo.Orders[orderID].Status != string(StatusPaid) {
		t.Errorf("Expected order status to be 'paid', got %s", mockRepo.Orders[orderID].Status)
	}
}

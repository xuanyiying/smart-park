package biz

import (
	"strings"
	"testing"
	"time"

	v1 "github.com/xuanyiying/smart-park/api/payment/v1"
)

func TestValidateAmount(t *testing.T) {
	tests := []struct {
		name        string
		orderAmount float64
		paidAmount  float64
		expectError bool
	}{
		{
			name:        "exact match",
			orderAmount: 10.00,
			paidAmount:  10.00,
			expectError: false,
		},
		{
			name:        "within tolerance",
			orderAmount: 10.00,
			paidAmount:  10.005,
			expectError: false,
		},
		{
			name:        "below tolerance",
			orderAmount: 10.00,
			paidAmount:  10.02,
			expectError: true,
		},
		{
			name:        "negative amount",
			orderAmount: 10.00,
			paidAmount:  -5.00,
			expectError: true,
		},
		{
			name:        "zero amount",
			orderAmount: 10.00,
			paidAmount:  0.00,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{FinalAmount: tt.orderAmount}
			err := (&PaymentUseCase{}).validateAmount(order, tt.paidAmount)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateWechatCallbackTime(t *testing.T) {
	uc := &PaymentUseCase{}

	validTime := "20260516003000"
	emptyTime := ""
	invalidTime := "invalid"
	oldTime := "20260514003000"
	futureTime := "20991231235959"

	tests := []struct {
		name        string
		timeEnd     string
		expectError bool
	}{
		{
			name:        "valid recent time",
			timeEnd:     validTime,
			expectError: false,
		},
		{
			name:        "empty time",
			timeEnd:     emptyTime,
			expectError: true,
		},
		{
			name:        "invalid format",
			timeEnd:     invalidTime,
			expectError: true,
		},
		{
			name:        "time too old",
			timeEnd:     oldTime,
			expectError: true,
		},
		{
			name:        "time in future (exceeds tolerance)",
			timeEnd:     futureTime,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.validateWechatCallbackTime(tt.timeEnd)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestValidateAlipayCallbackTime(t *testing.T) {
	uc := &PaymentUseCase{}

	validTime := "2026-05-16 00:30:00"
	emptyTime := ""
	invalidTime := "invalid"
	oldTime := "2026-05-14 00:30:00"
	futureTime := "2099-12-31 23:59:59"

	tests := []struct {
		name        string
		gmtPayment  string
		expectError bool
	}{
		{
			name:        "valid recent time",
			gmtPayment:  validTime,
			expectError: false,
		},
		{
			name:        "empty time",
			gmtPayment:  emptyTime,
			expectError: true,
		},
		{
			name:        "invalid format",
			gmtPayment:  invalidTime,
			expectError: true,
		},
		{
			name:        "time too old",
			gmtPayment:  oldTime,
			expectError: true,
		},
		{
			name:        "time in future (exceeds tolerance)",
			gmtPayment:  futureTime,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := uc.validateAlipayCallbackTime(tt.gmtPayment)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestUpdateOrderAsPaid(t *testing.T) {
	uc := &PaymentUseCase{}

	tests := []struct {
		name        string
		orderStatus string
		amount      float64
		expectError bool
	}{
		{
			name:        "valid update",
			orderStatus: string(StatusPending),
			amount:      10.00,
			expectError: false,
		},
		{
			name:        "invalid status",
			orderStatus: string(StatusPaid),
			amount:      10.00,
			expectError: true,
		},
		{
			name:        "zero amount",
			orderStatus: string(StatusPending),
			amount:      0.00,
			expectError: true,
		},
		{
			name:        "negative amount",
			orderStatus: string(StatusPending),
			amount:      -10.00,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{
				ID:     [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
				Status: tt.orderStatus,
			}

			err := uc.updateOrderAsPaid(order, MethodWechat, "transaction-123", tt.amount)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if !tt.expectError {
				if order.Status != string(StatusPaid) {
					t.Errorf("Expected status 'paid', got %s", order.Status)
				}
				if order.PayTime == nil {
					t.Error("Expected PayTime to be set")
				}
				if order.TransactionID != "transaction-123" {
					t.Errorf("Expected TransactionID 'transaction-123', got %s", order.TransactionID)
				}
				if order.PaidAmount != tt.amount {
					t.Errorf("Expected PaidAmount %.2f, got %.2f", tt.amount, order.PaidAmount)
				}
			}
		})
	}
}

func TestCallbackDeduplication(t *testing.T) {
	d := &callbackDeduplication{
		callbacks: make(map[string]time.Time),
	}

	callbackID := "wechat_test-order_123"

	if d.isProcessed(callbackID) {
		t.Error("New callback should not be marked as processed")
	}

	d.markProcessed(callbackID)

	if !d.isProcessed(callbackID) {
		t.Error("Marked callback should be processed")
	}

	if d.isProcessed("different-callback") {
		t.Error("Different callback should not be marked as processed")
	}
}

func TestCallbackDeduplication_Cleanup(t *testing.T) {
	d := &callbackDeduplication{
		callbacks: make(map[string]time.Time),
	}

	callbackID := "wechat_test-order_456"
	d.callbacks[callbackID] = time.Now().Add(-25 * time.Hour)

	d.cleanup()

	if _, exists := d.callbacks[callbackID]; exists {
		t.Error("Old callback should have been cleaned up")
	}
}

func TestBuildWechatSignString(t *testing.T) {
	req := &v1.WechatCallbackRequest{
		ReturnCode:    "SUCCESS",
		ReturnMsg:     "OK",
		ResultCode:    "SUCCESS",
		TransactionId: "wx123456",
		OutTradeNo:    "order123",
		TotalFee:      "1000",
		TimeEnd:       "20260326120000",
		Sign:          "abc123",
	}

	signString := buildWechatSignString(req)

	if signString == "" {
		t.Error("Sign string should not be empty")
	}

	expectedFields := []string{
		"result_code=SUCCESS",
		"return_code=SUCCESS",
		"return_msg=OK",
		"time_end=20260326120000",
		"total_fee=1000",
		"transaction_id=wx123456",
		"out_trade_no=order123",
	}

	for _, field := range expectedFields {
		if !strings.Contains(signString, field) {
			t.Errorf("Sign string missing expected field: %s", field)
		}
	}

	if !strings.HasPrefix(signString, "out_trade_no=") {
		t.Error("Sign string should start with out_trade_no field")
	}
}

func TestBuildAlipaySignString(t *testing.T) {
	req := &v1.AlipayCallbackRequest{
		TradeStatus: "TRADE_SUCCESS",
		TradeNo:     "ali123456",
		OutTradeNo:  "order123",
		TotalAmount: "10.00",
		GmtPayment:  "2026-03-26 12:00:00",
		Sign:        "abc123",
	}

	signString := buildAlipaySignString(req)

	if signString == "" {
		t.Error("Sign string should not be empty")
	}
}

func TestIsAlipaySuccessStatus(t *testing.T) {
	uc := &PaymentUseCase{}

	tests := []struct {
		status      string
		expectMatch bool
	}{
		{"TRADE_SUCCESS", true},
		{"TRADE_FINISHED", true},
		{"TRADE_CLOSED", false},
		{"WAIT_BUYER_PAY", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := uc.isAlipaySuccessStatus(tt.status)
			if result != tt.expectMatch {
				t.Errorf("isAlipaySuccessStatus(%s) = %v, want %v", tt.status, result, tt.expectMatch)
			}
		})
	}
}

func TestParseAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"1000", 10.00},
		{"0", 0.00},
		{"100", 1.00},
		{"12345", 123.45},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAmount(tt.input)
			if result != tt.expected {
				t.Errorf("parseAmount(%s) = %.2f, want %.2f", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseAmountFloat(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"10.00", 10.00},
		{"0.00", 0.00},
		{"123.45", 123.45},
		{"999.99", 999.99},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseAmountFloat(tt.input)
			if result != tt.expected {
				t.Errorf("parseAmountFloat(%s) = %.2f, want %.2f", tt.input, result, tt.expected)
			}
		})
	}
}

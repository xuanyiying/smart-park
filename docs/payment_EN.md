# Payment Service

## Module Overview

The Payment Service is the financial core of the Smart Park system, responsible for processing all parking fee payment flows. The service integrates WeChat Pay and Alipay, the two mainstream payment channels, supporting QR code payment, JSAPI payment, and other payment methods, providing complete order management, refund processing, and reconciliation functions.

## Core Functions

### 1. Payment Order Management

#### Order Lifecycle

```
Create Order → Pending Payment → Payment Success → Paid → Refund Request → Refund Success
                    ↓
               Payment Failed → Closed
```

#### Order Status Definitions

```go
type OrderStatus string

const (
    StatusPending   OrderStatus = "pending"    // Pending payment
    StatusPaid      OrderStatus = "paid"       // Paid
    StatusFailed    OrderStatus = "failed"     // Payment failed
    StatusRefunding OrderStatus = "refunding"  // Refunding
    StatusRefunded  OrderStatus = "refunded"   // Refunded
    StatusClosed    OrderStatus = "closed"     // Closed
)

type Order struct {
    ID                  uuid.UUID
    RecordID            uuid.UUID      // Associated parking record
    LotID               uuid.UUID      // Parking lot ID
    PlateNumber         string         // License plate
    Amount              float64        // Amount due
    DiscountAmount      float64        // Discount amount
    FinalAmount         float64        // Actual payment amount
    Status              string         // Order status
    PayMethod           string         // Payment method
    PayTime             *time.Time     // Payment time
    TransactionID       string         // Third-party payment transaction ID
    PaidAmount          float64        // Actual paid amount (written by callback)
    RefundedAt          *time.Time     // Refund time
    RefundTransactionID string         // Refund transaction ID
    ExpireTime          time.Time      // Order expiration time
}
```

### 2. Multi-Channel Payment Integration

#### WeChat Pay

**Supported Payment Methods**:
- Native Payment: Generate QR code for user scanning
- JSAPI Payment: Invoke payment within WeChat

**Core Implementation**:
```go
// WeChat Pay client
type WechatClient struct {
    client *core.Client
    config *WechatConfig
}

// Native Payment (QR code)
func (c *WechatClient) CreateNativePay(ctx context.Context, orderID string, amount int64, description string) (string, error) {
    resp, _, err := c.client.Prepay(ctx, 
        native.PrepayRequest{
            Appid:       c.config.AppID,
            Mchid:       c.config.MchID,
            Description: description,
            OutTradeNo:  orderID,
            Amount: &native.Amount{
                Total: amount,  // Unit: cents
            },
            NotifyUrl: c.config.NotifyURL,
        },
    )
    if err != nil {
        return "", err
    }
    return resp.CodeUrl, nil  // Return QR code URL
}

// JSAPI Payment (Mini Program/Official Account)
func (c *WechatClient) CreateJSAPIPay(ctx context.Context, orderID string, amount int64, openID, description string) (map[string]string, error) {
    resp, _, err := c.client.Prepay(ctx,
        jsapi.PrepayRequest{
            Appid:       c.config.AppID,
            Mchid:       c.config.MchID,
            Description: description,
            OutTradeNo:  orderID,
            Openid:      openID,
            Amount: &jsapi.Amount{
                Total: amount,
            },
            NotifyUrl: c.config.NotifyURL,
        },
    )
    if err != nil {
        return nil, err
    }
    
    // Generate frontend payment parameters
    params := c.buildJSAPIParams(resp.PrepayId)
    return params, nil
}
```

#### Alipay

**Supported Payment Methods**:
- Face-to-Face Payment: Generate QR code for user scanning

**Core Implementation**:
```go
// Alipay client
type AlipayClient struct {
    client *alipay.Client
    config *AlipayConfig
}

// Create face-to-face payment order
func (c *AlipayClient) CreateTradePreCreate(ctx context.Context, orderID string, amount float64, subject string) (string, error) {
    var req model.AlipayTradePrecreateRequest
    req.OutTradeNo = orderID
    req.TotalAmount = fmt.Sprintf("%.2f", amount)
    req.Subject = subject
    req.NotifyUrl = c.config.NotifyURL
    
    resp, err := c.client.TradePrecreate(ctx, req)
    if err != nil {
        return "", err
    }
    
    return resp.QrCode, nil  // Return QR code URL
}
```

### 3. Payment Callback Processing

#### Callback Security Mechanism

```go
// WeChat Pay callback processing
func (uc *PaymentUseCase) HandleWechatCallback(ctx context.Context, req *v1.PayCallbackRequest) error {
    // 1. Verify signature
    if err := uc.verifyWechatSignature(req); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }
    
    // 2. Idempotency check
    if uc.isCallbackProcessed(req.TransactionId) {
        return nil  // Already processed, return directly
    }
    
    // 3. Amount verification
    order, err := uc.orderRepo.GetOrderByID(ctx, req.OrderId)
    if err != nil {
        return err
    }
    
    paidAmount := float64(req.TotalFee) / 100  // Cents to yuan
    if math.Abs(paidAmount-order.FinalAmount) > 0.01 {
        uc.logSecurityEvent("amount_mismatch", order.ID, paidAmount, order.FinalAmount)
        return fmt.Errorf("amount mismatch")
    }
    
    // 4. Update order status
    return uc.processPaymentSuccess(ctx, order, req.TransactionId, paidAmount)
}
```

#### Payment Success Processing Flow

```go
func (uc *PaymentUseCase) processPaymentSuccess(ctx context.Context, order *Order, transactionID string, paidAmount float64) error {
    // 1. Update order status
    now := time.Now()
    order.Status = string(StatusPaid)
    order.PayTime = &now
    order.TransactionID = transactionID
    order.PaidAmount = paidAmount
    
    if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
        return err
    }
    
    // 2. Update parking record status
    if err := uc.recordRepo.UpdateExitStatus(ctx, order.RecordID, "paid"); err != nil {
        uc.log.WithContext(ctx).Errorf("failed to update record status: %v", err)
    }
    
    // 3. Auto-trigger barrier open
    record, _ := uc.recordRepo.GetRecordByID(ctx, order.RecordID)
    if record != nil && record.ExitDeviceID != nil {
        if err := uc.gateClient.OpenGate(ctx, *record.ExitDeviceID, order.RecordID.String()); err != nil {
            uc.log.WithContext(ctx).Errorf("failed to open gate: %v", err)
            // Gate open failure doesn't affect payment result, just log exception
        }
    }
    
    // 4. Send payment success notification
    uc.notificationService.SendPaymentSuccess(ctx, order)
    
    return nil
}
```

### 4. Refund Processing

#### Refund Flow

```go
// Refund request
func (uc *PaymentUseCase) Refund(ctx context.Context, orderID, reason string) (*v1.RefundData, error) {
    order, err := uc.orderRepo.GetOrderByID(ctx, orderID)
    if err != nil {
        return nil, err
    }
    
    // Verify order status
    if order.Status != string(StatusPaid) {
        return nil, fmt.Errorf("order status is not paid")
    }
    
    // Check refund time window (e.g., auto-refund within 30 minutes)
    if time.Since(*order.PayTime) > uc.config.RefundWindow {
        return nil, fmt.Errorf("refund window expired")
    }
    
    // Generate refund ID
    refundID := uuid.New().String()
    
    // Call corresponding refund interface based on payment method
    switch order.PayMethod {
    case "wechat":
        err = uc.wechatClient.Refund(ctx, order.TransactionID, refundID, order.FinalAmount)
    case "alipay":
        err = uc.alipayClient.Refund(ctx, order.TransactionID, refundID, order.FinalAmount)
    }
    
    if err != nil {
        return nil, err
    }
    
    // Update order status
    order.Status = string(StatusRefunded)
    order.RefundTransactionID = refundID
    now := time.Now()
    order.RefundedAt = &now
    
    if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
        return nil, err
    }
    
    return &v1.RefundData{
        RefundId: refundID,
        Status:   "success",
    }, nil
}
```

#### Refund Approval Flow (Admin)

```go
// Create refund approval request
func (uc *PaymentUseCase) CreateRefundApproval(ctx context.Context, orderID string, amount float64, reason, applicant string) (*RefundApproval, error) {
    order, err := uc.orderRepo.GetOrderByID(ctx, orderID)
    if err != nil {
        return nil, err
    }
    
    if amount > order.FinalAmount {
        return nil, fmt.Errorf("refund amount exceeds paid amount")
    }
    
    approval := &RefundApproval{
        ID:           uuid.New(),
        OrderID:      order.ID,
        Applicant:    applicant,
        Amount:       amount,
        Reason:       reason,
        RefundMethod: "original",  // Original path refund
        Status:       "pending",
    }
    
    if err := uc.refundApprovalRepo.Create(ctx, approval); err != nil {
        return nil, err
    }
    
    return approval, nil
}

// Approve and execute refund
func (uc *PaymentUseCase) ApproveRefund(ctx context.Context, approvalID, approver string) error {
    approval, err := uc.refundApprovalRepo.GetByID(ctx, approvalID)
    if err != nil {
        return err
    }
    
    if approval.Status != "pending" {
        return fmt.Errorf("approval already processed")
    }
    
    // Execute refund
    order, _ := uc.orderRepo.GetOrderByID(ctx, approval.OrderID.String())
    if err := uc.executeRefund(ctx, order, approval.Amount); err != nil {
        return err
    }
    
    // Update approval status
    approval.Status = "approved"
    approval.Approver = approver
    now := time.Now()
    approval.ApprovedAt = &now
    
    return uc.refundApprovalRepo.Update(ctx, approval)
}
```

## Application Scenarios

### Scenario 1: Exit QR Code Payment

**Business Flow**:
1. Vehicle arrives at exit, license plate recognized
2. System calculates parking fee
3. Exit screen displays QR code
4. Driver scans QR code to pay
5. Payment successful, barrier opens automatically

**Technical Implementation**:
```go
func (uc *PaymentUseCase) CreateExitPayment(ctx context.Context, recordID string, amount float64, payMethod string) (*v1.PaymentData, error) {
    // Create payment order
    order, err := uc.createOrder(ctx, recordID, amount)
    if err != nil {
        return nil, err
    }
    
    // Generate payment QR code
    var payURL, qrCode string
    switch payMethod {
    case "wechat":
        qrCode, err = uc.wechatClient.CreateNativePay(ctx, order.ID.String(), int64(amount*100), "Parking Fee")
    case "alipay":
        qrCode, err = uc.alipayClient.CreateTradePreCreate(ctx, order.ID.String(), amount, "Parking Fee")
    }
    
    if err != nil {
        return nil, err
    }
    
    return &v1.PaymentData{
        OrderId:    order.ID.String(),
        Amount:     amount,
        PayUrl:     payURL,
        QrCode:     qrCode,
        ExpireTime: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
    }, nil
}
```

### Scenario 2: Mini Program Advance Payment

**Business Flow**:
1. Driver queries parking records in mini program
2. Selects record and clicks pay
3. Invoke WeChat Pay (JSAPI)
4. Payment successful, generates payment voucher
5. Exit automatically recognizes paid status and releases

**Technical Implementation**:
```go
func (uc *PaymentUseCase) CreateJSAPIPayment(ctx context.Context, recordID, openID string, amount float64) (*v1.PaymentData, error) {
    // Create order
    order, err := uc.createOrder(ctx, recordID, amount)
    if err != nil {
        return nil, err
    }
    
    // Generate JSAPI payment parameters
    params, err := uc.wechatClient.CreateJSAPIPay(ctx, order.ID.String(), int64(amount*100), openID, "Parking Fee")
    if err != nil {
        return nil, err
    }
    
    return &v1.PaymentData{
        OrderId:    order.ID.String(),
        Amount:     amount,
        JsapiParams: params,  // Frontend payment parameters
        ExpireTime: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
    }, nil
}
```

## Technical Challenges and Solutions

### Challenge 1: Payment Security

**Problem Description**:
Payment involves fund security, requiring prevention of forged callbacks, amount tampering, duplicate payments, and other risks.

**Solution**:

1. **Signature Verification**
   ```go
   // WeChat Pay signature verification
   func (uc *PaymentUseCase) verifyWechatSignature(req *v1.PayCallbackRequest) error {
       // Verify signature using WeChat Pay platform certificate
       certificate, err := uc.wechatClient.GetCertificate(ctx, req.SerialNo)
       if err != nil {
           return err
       }
       
       return utils.VerifySignature(certificate, req.Signature, req.Body)
   }
   
   // Alipay signature verification
   func (uc *PaymentUseCase) verifyAlipaySignature(req *v1.PayCallbackRequest) error {
       return uc.alipayClient.VerifySign(req.Params)
   }
   ```

2. **Amount Verification**
   ```go
   // Compare callback amount with order amount
   paidAmount := float64(req.TotalFee) / 100
   if math.Abs(paidAmount-order.FinalAmount) > 0.01 {
       // Amount mismatch, log security event
       uc.logSecurityEvent("amount_mismatch", order.ID, paidAmount, order.FinalAmount)
       return fmt.Errorf("amount mismatch: expected %.2f, got %.2f", order.FinalAmount, paidAmount)
   }
   ```

3. **Idempotency Guarantee**
   ```go
   // Use transaction_id for deduplication
   func (uc *PaymentUseCase) isCallbackProcessed(transactionID string) bool {
       key := fmt.Sprintf("parking:v1:callback:%s", transactionID)
       exists, _ := uc.cache.Exists(ctx, key)
       if exists {
           return true
       }
       // Set 24-hour expiration
       uc.cache.Set(ctx, key, "1", 24*time.Hour)
       return false
   }
   ```

### Challenge 2: Payment Timeout Handling

**Problem Description**:
Users may not pay or pay after timeout after scanning QR code, requiring proper handling of order status.

**Solution**:

1. **Order Expiration Mechanism**
   ```go
   // Set expiration time when creating order
   order.ExpireTime = time.Now().Add(30 * time.Minute)
   
   // Scheduled task to clean up expired orders
   func (uc *PaymentUseCase) CleanupExpiredOrders(ctx context.Context) {
       expiredOrders, _ := uc.orderRepo.GetExpiredOrders(ctx)
       for _, order := range expiredOrders {
           order.Status = string(StatusClosed)
           uc.orderRepo.UpdateOrder(ctx, order)
       }
   }
   ```

2. **Order Close Handling**
   ```go
   // Actively close unpaid orders
   func (uc *PaymentUseCase) CloseOrder(ctx context.Context, orderID string) error {
       order, err := uc.orderRepo.GetOrderByID(ctx, orderID)
       if err != nil {
           return err
       }
       
       if order.Status != string(StatusPending) {
           return fmt.Errorf("order cannot be closed")
       }
       
       // Call third-party payment close interface
       switch order.PayMethod {
       case "wechat":
           uc.wechatClient.CloseOrder(ctx, order.ID.String())
       case "alipay":
           uc.alipayClient.CloseTrade(ctx, order.ID.String())
       }
       
       order.Status = string(StatusClosed)
       return uc.orderRepo.UpdateOrder(ctx, order)
   }
   ```

### Challenge 3: Payment Channel Failure

**Problem Description**:
Payment channels may experience failures, requiring system availability guarantees.

**Solution**:

1. **Multi-Channel Redundancy**
   ```go
   // Switch to backup channel when primary channel fails
   func (uc *PaymentUseCase) CreatePaymentWithFallback(ctx context.Context, order *Order) (string, error) {
       // Try primary channel
       qrCode, err := uc.wechatClient.CreateNativePay(ctx, order)
       if err == nil {
           return qrCode, nil
       }
       
       uc.log.WithContext(ctx).Warnf("primary payment channel failed: %v", err)
       
       // Switch to backup channel
       qrCode, err = uc.alipayClient.CreateTradePreCreate(ctx, order)
       if err != nil {
           return "", fmt.Errorf("all payment channels failed")
       }
       
       order.PayMethod = "alipay"  // Update payment method
       return qrCode, nil
   }
   ```

2. **Degradation Strategy**
   ```go
   // Allow offline release when payment service unavailable
   func (uc *PaymentUseCase) HandlePaymentFailure(ctx context.Context, recordID string) error {
       // Record unpaid
       uc.recordRepo.UpdateExitStatus(ctx, recordID, "unpaid")
       
       // Add to collection list
       uc.debtRepo.Create(ctx, &Debt{
           RecordID: recordID,
           Status:   "pending",
       })
       
       return nil
   }
   ```

### Challenge 4: Reconciliation and Discrepancy Handling

**Problem Description**:
Need to ensure system orders are consistent with third-party payment channel bills, handling one-sided accounts and other issues.

**Solution**:

1. **Automatic Reconciliation**
   ```go
   // Daily reconciliation task
   func (uc *PaymentUseCase) DailyReconciliation(ctx context.Context, date string) error {
       // Get system orders
       systemOrders, _ := uc.orderRepo.GetPaidOrdersByDate(ctx, date)
       
       // Get WeChat Pay bills
       wechatBills, _ := uc.wechatClient.DownloadBill(ctx, date)
       
       // Get Alipay bills
       alipayBills, _ := uc.alipayClient.DownloadBill(ctx, date)
       
       // Compare discrepancies
       discrepancies := uc.compareOrders(systemOrders, wechatBills, alipayBills)
       
       // Log discrepancies and alert
       for _, d := range discrepancies {
           uc.logDiscrepancy(ctx, d)
       }
       
       return nil
   }
   ```

2. **Discrepancy Handling**
   ```go
   // Handle one-sided accounts
   func (uc *PaymentUseCase) HandleDiscrepancy(ctx context.Context, d *Discrepancy) error {
       switch d.Type {
       case "system_missing":
           // Channel has, system doesn't: supplement order
           return uc.createOrderFromBill(ctx, d.Bill)
       case "channel_missing":
           // System has, channel doesn't: query channel status, refund if necessary
           return uc.verifyAndRefund(ctx, d.Order)
       case "amount_mismatch":
           // Amount mismatch: log exception, manual processing
           return uc.escalateToManual(ctx, d)
       }
       return nil
   }
   ```

## API Documentation

### Create Payment Order

```http
POST /api/v1/pay/create
Content-Type: application/json

{
  "recordId": "rec_xxx",
  "amount": 15.00,
  "payMethod": "wechat",
  "openId": "oxxxxx"  // Required for JSAPI payment
}
```

**Response**:
```json
{
  "code": 0,
  "data": {
    "orderId": "ord_xxx",
    "amount": 15.00,
    "payUrl": "weixin://wxpay/bizpayurl?pr=xxx",
    "qrCode": "https://api.example.com/qr/xxx",
    "expireTime": "2026-03-20T12:30:00Z"
  }
}
```

### Query Payment Status

```http
GET /api/v1/pay/{orderId}/status
```

**Response**:
```json
{
  "code": 0,
  "data": {
    "orderId": "ord_xxx",
    "status": "paid",
    "payTime": "2026-03-20T12:15:00Z",
    "payMethod": "wechat"
  }
}
```

### Request Refund

```http
POST /api/v1/pay/{orderId}/refund
Content-Type: application/json

{
  "reason": "Billing error"
}
```

**Response**:
```json
{
  "code": 0,
  "data": {
    "refundId": "ref_xxx",
    "status": "success"
  }
}
```

## Configuration

```yaml
# configs/payment.yaml
server:
  http:
    addr: 0.0.0.0:8003
  grpc:
    addr: 0.0.0.0:9003

data:
  database:
    driver: postgres
    source: postgres://postgres:postgres@localhost:5432/parking?sslmode=disable

payment:
  orderExpiration: 30m  # Order expiration time
  refundWindow: 30m     # Refund time window
  
wechat:
  appId: "wx_app_id"
  mchId: "mch_id"
  apiV3Key: "api_v3_key"
  privateKeyPath: "/secrets/wechat_private_key.pem"
  serialNo: "certificate_serial_no"
  notifyUrl: "https://api.example.com/api/v1/pay/callback/wechat"
  
alipay:
  appId: "alipay_app_id"
  privateKeyPath: "/secrets/alipay_private_key.pem"
  publicKeyPath: "/secrets/alipay_public_key.pem"
  notifyUrl: "https://api.example.com/api/v1/pay/callback/alipay"
```

## Monitoring Metrics

| Metric | Type | Description |
|--------|------|-------------|
| payment_order_total | Counter | Total orders created |
| payment_success_total | Counter | Total successful payments |
| payment_failure_total | Counter | Total failed payments |
| payment_callback_total | Counter | Total callback processing |
| payment_refund_total | Counter | Total refunds |
| payment_duration | Histogram | Payment processing duration |
| payment_amount_histogram | Histogram | Payment amount distribution |

## Related Documentation

- [Vehicle Service Documentation](vehicle_EN.md)
- [Billing Service Documentation](billing_EN.md)
- [Deployment Documentation](deployment_EN.md)

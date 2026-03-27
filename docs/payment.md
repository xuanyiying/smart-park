# Payment Service 支付服务

## 模块概述

支付服务是 Smart Park 系统的财务核心，负责处理所有停车费用的支付流程。服务集成了微信支付和支付宝两大主流支付渠道，支持扫码支付、JSAPI支付等多种支付方式，并提供完整的订单管理、退款处理、对账功能。

## 核心功能

### 1. 支付订单管理

#### 订单生命周期

```
创建订单 → 待支付 → 支付成功 → 已支付 → 退款申请 → 退款成功
                    ↓
               支付失败 → 已关闭
```

#### 订单状态定义

```go
type OrderStatus string

const (
    StatusPending   OrderStatus = "pending"    // 待支付
    StatusPaid      OrderStatus = "paid"       // 已支付
    StatusFailed    OrderStatus = "failed"     // 支付失败
    StatusRefunding OrderStatus = "refunding"  // 退款中
    StatusRefunded  OrderStatus = "refunded"   // 已退款
    StatusClosed    OrderStatus = "closed"     // 已关闭
)

type Order struct {
    ID                  uuid.UUID
    RecordID            uuid.UUID      // 关联停车记录
    LotID               uuid.UUID      // 停车场ID
    PlateNumber         string         // 车牌号
    Amount              float64        // 应付金额
    DiscountAmount      float64        // 优惠金额
    FinalAmount         float64        // 实付金额
    Status              string         // 订单状态
    PayMethod           string         // 支付方式
    PayTime             *time.Time     // 支付时间
    TransactionID       string         // 第三方支付流水号
    PaidAmount          float64        // 实际支付金额（回调写入）
    RefundedAt          *time.Time     // 退款时间
    RefundTransactionID string         // 退款流水号
    ExpireTime          time.Time      // 订单过期时间
}
```

### 2. 多渠道支付集成

#### 微信支付

**支持的支付方式**：
- Native 支付：生成二维码，用户扫码支付
- JSAPI 支付：微信内网页调起支付

**核心实现**：
```go
// 微信支付客户端
type WechatClient struct {
    client *core.Client
    config *WechatConfig
}

// Native 支付（扫码支付）
func (c *WechatClient) CreateNativePay(ctx context.Context, orderID string, amount int64, description string) (string, error) {
    resp, _, err := c.client.Prepay(ctx, 
        native.PrepayRequest{
            Appid:       c.config.AppID,
            Mchid:       c.config.MchID,
            Description: description,
            OutTradeNo:  orderID,
            Amount: &native.Amount{
                Total: amount,  // 单位：分
            },
            NotifyUrl: c.config.NotifyURL,
        },
    )
    if err != nil {
        return "", err
    }
    return resp.CodeUrl, nil  // 返回二维码链接
}

// JSAPI 支付（小程序/公众号）
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
    
    // 生成前端调起支付所需的参数
    params := c.buildJSAPIParams(resp.PrepayId)
    return params, nil
}
```

#### 支付宝

**支持的支付方式**：
- 当面付：生成二维码，用户扫码支付

**核心实现**：
```go
// 支付宝客户端
type AlipayClient struct {
    client *alipay.Client
    config *AlipayConfig
}

// 创建当面付订单
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
    
    return resp.QrCode, nil  // 返回二维码链接
}
```

### 3. 支付回调处理

#### 回调安全机制

```go
// 微信支付回调处理
func (uc *PaymentUseCase) HandleWechatCallback(ctx context.Context, req *v1.PayCallbackRequest) error {
    // 1. 验证签名
    if err := uc.verifyWechatSignature(req); err != nil {
        return fmt.Errorf("signature verification failed: %w", err)
    }
    
    // 2. 幂等性检查
    if uc.isCallbackProcessed(req.TransactionId) {
        return nil  // 已处理过，直接返回
    }
    
    // 3. 金额校验
    order, err := uc.orderRepo.GetOrderByID(ctx, req.OrderId)
    if err != nil {
        return err
    }
    
    paidAmount := float64(req.TotalFee) / 100  // 分转元
    if math.Abs(paidAmount-order.FinalAmount) > 0.01 {
        uc.logSecurityEvent("amount_mismatch", order.ID, paidAmount, order.FinalAmount)
        return fmt.Errorf("amount mismatch")
    }
    
    // 4. 更新订单状态
    return uc.processPaymentSuccess(ctx, order, req.TransactionId, paidAmount)
}
```

#### 支付成功处理流程

```go
func (uc *PaymentUseCase) processPaymentSuccess(ctx context.Context, order *Order, transactionID string, paidAmount float64) error {
    // 1. 更新订单状态
    now := time.Now()
    order.Status = string(StatusPaid)
    order.PayTime = &now
    order.TransactionID = transactionID
    order.PaidAmount = paidAmount
    
    if err := uc.orderRepo.UpdateOrder(ctx, order); err != nil {
        return err
    }
    
    // 2. 更新停车记录状态
    if err := uc.recordRepo.UpdateExitStatus(ctx, order.RecordID, "paid"); err != nil {
        uc.log.WithContext(ctx).Errorf("failed to update record status: %v", err)
    }
    
    // 3. 自动触发开闸
    record, _ := uc.recordRepo.GetRecordByID(ctx, order.RecordID)
    if record != nil && record.ExitDeviceID != nil {
        if err := uc.gateClient.OpenGate(ctx, *record.ExitDeviceID, order.RecordID.String()); err != nil {
            uc.log.WithContext(ctx).Errorf("failed to open gate: %v", err)
            // 开闸失败不影响支付结果，记录异常即可
        }
    }
    
    // 4. 发送支付成功通知
    uc.notificationService.SendPaymentSuccess(ctx, order)
    
    return nil
}
```

### 4. 退款处理

#### 退款流程

```go
// 退款申请
func (uc *PaymentUseCase) Refund(ctx context.Context, orderID, reason string) (*v1.RefundData, error) {
    order, err := uc.orderRepo.GetOrderByID(ctx, orderID)
    if err != nil {
        return nil, err
    }
    
    // 校验订单状态
    if order.Status != string(StatusPaid) {
        return nil, fmt.Errorf("order status is not paid")
    }
    
    // 检查退款时效（如 30 分钟内可自动退款）
    if time.Since(*order.PayTime) > uc.config.RefundWindow {
        return nil, fmt.Errorf("refund window expired")
    }
    
    // 生成退款ID
    refundID := uuid.New().String()
    
    // 根据支付方式调用对应退款接口
    switch order.PayMethod {
    case "wechat":
        err = uc.wechatClient.Refund(ctx, order.TransactionID, refundID, order.FinalAmount)
    case "alipay":
        err = uc.alipayClient.Refund(ctx, order.TransactionID, refundID, order.FinalAmount)
    }
    
    if err != nil {
        return nil, err
    }
    
    // 更新订单状态
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

#### 退款审批流程（管理端）

```go
// 创建退款审批申请
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
        RefundMethod: "original",  // 原路返回
        Status:       "pending",
    }
    
    if err := uc.refundApprovalRepo.Create(ctx, approval); err != nil {
        return nil, err
    }
    
    return approval, nil
}

// 审批通过并执行退款
func (uc *PaymentUseCase) ApproveRefund(ctx context.Context, approvalID, approver string) error {
    approval, err := uc.refundApprovalRepo.GetByID(ctx, approvalID)
    if err != nil {
        return err
    }
    
    if approval.Status != "pending" {
        return fmt.Errorf("approval already processed")
    }
    
    // 执行退款
    order, _ := uc.orderRepo.GetOrderByID(ctx, approval.OrderID.String())
    if err := uc.executeRefund(ctx, order, approval.Amount); err != nil {
        return err
    }
    
    // 更新审批状态
    approval.Status = "approved"
    approval.Approver = approver
    now := time.Now()
    approval.ApprovedAt = &now
    
    return uc.refundApprovalRepo.Update(ctx, approval)
}
```

## 应用场景区景

### 场景一：出口扫码支付

**业务流程**：
1. 车辆到达出口，车牌识别
2. 系统计算停车费用
3. 出口屏幕显示二维码
4. 车主扫码支付
5. 支付成功，自动开闸

**技术实现**：
```go
func (uc *PaymentUseCase) CreateExitPayment(ctx context.Context, recordID string, amount float64, payMethod string) (*v1.PaymentData, error) {
    // 创建支付订单
    order, err := uc.createOrder(ctx, recordID, amount)
    if err != nil {
        return nil, err
    }
    
    // 生成支付二维码
    var payURL, qrCode string
    switch payMethod {
    case "wechat":
        qrCode, err = uc.wechatClient.CreateNativePay(ctx, order.ID.String(), int64(amount*100), "停车费")
    case "alipay":
        qrCode, err = uc.alipayClient.CreateTradePreCreate(ctx, order.ID.String(), amount, "停车费")
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

### 场景二：小程序提前支付

**业务流程**：
1. 车主在小程序查询停车记录
2. 选择记录，点击支付
3. 调起微信支付（JSAPI）
4. 支付成功，生成支付凭证
5. 出口自动识别已支付，直接放行

**技术实现**：
```go
func (uc *PaymentUseCase) CreateJSAPIPayment(ctx context.Context, recordID, openID string, amount float64) (*v1.PaymentData, error) {
    // 创建订单
    order, err := uc.createOrder(ctx, recordID, amount)
    if err != nil {
        return nil, err
    }
    
    // 生成 JSAPI 支付参数
    params, err := uc.wechatClient.CreateJSAPIPay(ctx, order.ID.String(), int64(amount*100), openID, "停车费")
    if err != nil {
        return nil, err
    }
    
    return &v1.PaymentData{
        OrderId:    order.ID.String(),
        Amount:     amount,
        JsapiParams: params,  // 前端调起支付所需参数
        ExpireTime: time.Now().Add(30 * time.Minute).Format(time.RFC3339),
    }, nil
}
```

### 场景三：无感支付（签约代扣）

**业务流程**：
1. 车主在小程序签约无感支付
2. 车辆出场时自动识别车牌
3. 系统自动扣款
4. 扣款成功，直接放行

**技术实现**：
```go
func (uc *PaymentUseCase) ProcessAutoPayment(ctx context.Context, recordID, plateNumber string, amount float64) error {
    // 查询用户签约信息
    contract, err := uc.contractRepo.GetByPlateNumber(ctx, plateNumber)
    if err != nil {
        return fmt.Errorf("no active contract found")
    }
    
    // 创建订单
    order, err := uc.createOrder(ctx, recordID, amount)
    if err != nil {
        return err
    }
    
    // 调用委托扣款接口
    err = uc.wechatClient.Pappay(ctx, contract.ContractID, order.ID.String(), int64(amount*100))
    if err != nil {
        return err
    }
    
    // 异步等待扣款结果
    return nil
}
```

## 技术挑战与解决方案

### 挑战一：支付安全

**问题描述**：
支付涉及资金安全，需要防范伪造回调、金额篡改、重复支付等风险。

**解决方案**：

1. **签名验证**
   ```go
   // 微信支付签名验证
   func (uc *PaymentUseCase) verifyWechatSignature(req *v1.PayCallbackRequest) error {
       // 使用微信支付平台证书验证签名
       certificate, err := uc.wechatClient.GetCertificate(ctx, req.SerialNo)
       if err != nil {
           return err
       }
       
       return utils.VerifySignature(certificate, req.Signature, req.Body)
   }
   
   // 支付宝签名验证
   func (uc *PaymentUseCase) verifyAlipaySignature(req *v1.PayCallbackRequest) error {
       return uc.alipayClient.VerifySign(req.Params)
   }
   ```

2. **金额校验**
   ```go
   // 回调金额与订单金额比对
   paidAmount := float64(req.TotalFee) / 100
   if math.Abs(paidAmount-order.FinalAmount) > 0.01 {
       // 金额不符，记录安全事件
       uc.logSecurityEvent("amount_mismatch", order.ID, paidAmount, order.FinalAmount)
       return fmt.Errorf("amount mismatch: expected %.2f, got %.2f", order.FinalAmount, paidAmount)
   }
   ```

3. **幂等性保证**
   ```go
   // 使用 transaction_id 去重
   func (uc *PaymentUseCase) isCallbackProcessed(transactionID string) bool {
       key := fmt.Sprintf("parking:v1:callback:%s", transactionID)
       exists, _ := uc.cache.Exists(ctx, key)
       if exists {
           return true
       }
       // 设置 24 小时过期
       uc.cache.Set(ctx, key, "1", 24*time.Hour)
       return false
   }
   ```

### 挑战二：支付超时处理

**问题描述**：
用户扫码后可能不支付或支付超时，需要正确处理订单状态。

**解决方案**：

1. **订单过期机制**
   ```go
   // 创建订单时设置过期时间
   order.ExpireTime = time.Now().Add(30 * time.Minute)
   
   // 定时任务清理过期订单
   func (uc *PaymentUseCase) CleanupExpiredOrders(ctx context.Context) {
       expiredOrders, _ := uc.orderRepo.GetExpiredOrders(ctx)
       for _, order := range expiredOrders {
           order.Status = string(StatusClosed)
           uc.orderRepo.UpdateOrder(ctx, order)
       }
   }
   ```

2. **关单处理**
   ```go
   // 主动关闭未支付订单
   func (uc *PaymentUseCase) CloseOrder(ctx context.Context, orderID string) error {
       order, err := uc.orderRepo.GetOrderByID(ctx, orderID)
       if err != nil {
           return err
       }
       
       if order.Status != string(StatusPending) {
           return fmt.Errorf("order cannot be closed")
       }
       
       // 调用第三方支付关单接口
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

### 挑战三：支付渠道故障

**问题描述**：
支付渠道可能出现故障，需要保证系统可用性。

**解决方案**：

1. **多通道冗余**
   ```go
   // 优先通道支付失败时切换备用通道
   func (uc *PaymentUseCase) CreatePaymentWithFallback(ctx context.Context, order *Order) (string, error) {
       // 尝试主通道
       qrCode, err := uc.wechatClient.CreateNativePay(ctx, order)
       if err == nil {
           return qrCode, nil
       }
       
       uc.log.WithContext(ctx).Warnf("primary payment channel failed: %v", err)
       
       // 切换到备用通道
       qrCode, err = uc.alipayClient.CreateTradePreCreate(ctx, order)
       if err != nil {
           return "", fmt.Errorf("all payment channels failed")
       }
       
       order.PayMethod = "alipay"  // 更新支付方式
       return qrCode, nil
   }
   ```

2. **降级策略**
   ```go
   // 支付服务不可用时，允许离线放行
   func (uc *PaymentUseCase) HandlePaymentFailure(ctx context.Context, recordID string) error {
       // 记录欠费
       uc.recordRepo.UpdateExitStatus(ctx, recordID, "unpaid")
       
       // 加入追缴名单
       uc.debtRepo.Create(ctx, &Debt{
           RecordID: recordID,
           Status:   "pending",
       })
       
       return nil
   }
   ```

### 挑战四：对账与差错处理

**问题描述**：
需要确保系统订单与第三方支付渠道账单一致，处理单边账等问题。

**解决方案**：

1. **自动对账**
   ```go
   // 每日对账任务
   func (uc *PaymentUseCase) DailyReconciliation(ctx context.Context, date string) error {
       // 获取系统订单
       systemOrders, _ := uc.orderRepo.GetPaidOrdersByDate(ctx, date)
       
       // 获取微信支付账单
       wechatBills, _ := uc.wechatClient.DownloadBill(ctx, date)
       
       // 获取支付宝账单
       alipayBills, _ := uc.alipayClient.DownloadBill(ctx, date)
       
       // 比对差异
       discrepancies := uc.compareOrders(systemOrders, wechatBills, alipayBills)
       
       // 记录差异并告警
       for _, d := range discrepancies {
           uc.logDiscrepancy(ctx, d)
       }
       
       return nil
   }
   ```

2. **差错处理**
   ```go
   // 处理单边账
   func (uc *PaymentUseCase) HandleDiscrepancy(ctx context.Context, d *Discrepancy) error {
       switch d.Type {
       case "system_missing":
           // 渠道有，系统无：补录订单
           return uc.createOrderFromBill(ctx, d.Bill)
       case "channel_missing":
           // 系统有，渠道无：查询渠道状态，必要时退款
           return uc.verifyAndRefund(ctx, d.Order)
       case "amount_mismatch":
           // 金额不符：记录异常，人工处理
           return uc.escalateToManual(ctx, d)
       }
       return nil
   }
   ```

## API 接口文档

### 创建支付订单

```http
POST /api/v1/pay/create
Content-Type: application/json

{
  "recordId": "rec_xxx",
  "amount": 15.00,
  "payMethod": "wechat",
  "openId": "oxxxxx"  // JSAPI支付时必填
}
```

**响应**：
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

### 查询支付状态

```http
GET /api/v1/pay/{orderId}/status
```

**响应**：
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

### 申请退款

```http
POST /api/v1/pay/{orderId}/refund
Content-Type: application/json

{
  "reason": "收费错误"
}
```

**响应**：
```json
{
  "code": 0,
  "data": {
    "refundId": "ref_xxx",
    "status": "success"
  }
}
```

## 配置说明

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
  orderExpiration: 30m  # 订单过期时间
  refundWindow: 30m     # 退款时效窗口
  
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

## 监控指标

| 指标 | 类型 | 说明 |
|------|------|------|
| payment_order_total | Counter | 订单创建总数 |
| payment_success_total | Counter | 支付成功总数 |
| payment_failure_total | Counter | 支付失败总数 |
| payment_callback_total | Counter | 回调处理总数 |
| payment_refund_total | Counter | 退款总数 |
| payment_duration | Histogram | 支付处理耗时 |
| payment_amount_histogram | Histogram | 支付金额分布 |

## 相关文档

- [车辆服务文档](vehicle.md)
- [计费服务文档](billing.md)
- [部署文档](deployment.md)

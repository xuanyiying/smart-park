# Smart Park 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成 Smart Park 停车场系统的核心功能实现，包括车主端 API、支付 SDK 集成、设备控制服务，使系统达到可演示状态。

**Architecture:** 基于 Kratos 微服务框架，采用 Service/Biz/Data 三层架构，使用 Ent ORM 进行数据访问，通过 gRPC + HTTP 双协议对外提供服务。

**Tech Stack:** Go 1.21+, Kratos, Ent, PostgreSQL, Redis, MQTT, Docker

---

## 优先级划分

### 🔴 P0 - MVP 必需（第一阶段）
1. 车主端 API 完整实现
2. 支付 SDK 集成与验签
3. 设备控制 MQTT 集成

### 🟡 P1 - 完整功能（第二阶段）
4. 计费引擎完善
5. 退款审批流
6. 通知服务

### 🟢 P2 - 增强功能（第三阶段）
7. Redis 缓存层
8. 消息队列
9. 监控告警

---

## Phase 1: 车主端 API 实现

### Task 1.1: 完善 User Service 结构

**Files:**
- Create: `internal/user/service/user.go`
- Modify: `internal/user/biz/user.go`
- Modify: `api/user/v1/user.proto`

- [ ] **Step 1: 检查现有 user.proto 定义**

```bash
cat api/user/v1/user.proto
```

确认已有接口：
- Login (微信登录)
- BindPlate / UnbindPlate / ListPlates

需要补充：
- GetParkingRecords (停车记录查询)
- GetParkingRecord (单条记录详情)
- CreatePayment (扫码支付)
- ListOrders / GetOrder (订单查询)
- GetMonthlyCard / PurchaseMonthlyCard (月卡管理)

- [ ] **Step 2: 补充 user.proto 定义**

```protobuf
// 在 api/user/v1/user.proto 中添加

rpc GetParkingRecords (GetParkingRecordsRequest) returns (GetParkingRecordsResponse) {
  option (google.api.http) = {
    get: "/api/v1/user/records"
  };
}

rpc GetParkingRecord (GetParkingRecordRequest) returns (GetParkingRecordResponse) {
  option (google.api.http) = {
    get: "/api/v1/user/records/{id}"
  };
}

rpc CreatePayment (CreatePaymentRequest) returns (CreatePaymentResponse) {
  option (google.api.http) = {
    post: "/api/v1/user/pay/create"
    body: "*"
  };
}

rpc ListOrders (ListOrdersRequest) returns (ListOrdersResponse) {
  option (google.api.http) = {
    get: "/api/v1/user/orders"
  };
}

rpc GetOrder (GetOrderRequest) returns (GetOrderResponse) {
  option (google.api.http) = {
    get: "/api/v1/user/orders/{id}"
  };
}

rpc GetMonthlyCard (GetMonthlyCardRequest) returns (GetMonthlyCardResponse) {
  option (google.api.http) = {
    get: "/api/v1/user/monthly-card"
  };
}

rpc PurchaseMonthlyCard (PurchaseMonthlyCardRequest) returns (PurchaseMonthlyCardResponse) {
  option (google.api.http) = {
    post: "/api/v1/user/monthly-card/purchase"
    body: "*"
  };
}
```

- [ ] **Step 3: 生成 proto 代码**

```bash
cd /Users/yiying/dev-app/smart-park
make proto
# 或
buf generate
```

- [ ] **Step 4: 实现 UserService 方法**

在 `internal/user/service/user.go` 中实现：

```go
func (s *UserService) GetParkingRecords(ctx context.Context, req *v1.GetParkingRecordsRequest) (*v1.GetParkingRecordsResponse, error) {
    // 从 JWT 中获取 userID
    userID := s.getUserIDFromContext(ctx)
    data, err := s.uc.GetParkingRecords(ctx, userID, req)
    // ...
}

func (s *UserService) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.CreatePaymentResponse, error) {
    // 调用 payment service 创建支付
    // 需要集成 payment client
}
```

- [ ] **Step 5: 添加 Gateway 路由**

修改 `configs/gateway.yaml`：

```yaml
routes:
  - path: "/api/v1/user"
    target: "user-svc:8005"
```

- [ ] **Step 6: 测试 API**

```bash
# 启动 user service
go run ./cmd/user -conf ./configs

# 测试登录
curl -X POST http://localhost:8000/api/v1/user/login \
  -H "Content-Type: application/json" \
  -d '{"code": "test_code"}'
```

- [ ] **Step 7: Commit**

```bash
git add api/user/v1/user.proto internal/user/service/user.go configs/gateway.yaml
git commit -m "feat(user): add user service API definitions and implementations"
```

---

### Task 1.2: 实现停车记录查询

**Files:**
- Modify: `internal/user/biz/user.go`
- Modify: `internal/user/data/user.go`
- Create: `internal/user/data/ent/schema/parkingrecord.go`

- [ ] **Step 1: 在 user service 中访问 vehicle 数据**

由于停车记录在 vehicle service 中，需要：
1. 通过 gRPC client 调用 vehicle service
2. 或创建只读视图

选择方案：创建 VehicleClient

```go
// internal/user/client/vehicle/client.go
type VehicleClient interface {
    GetParkingRecordsByPlate(ctx context.Context, plateNumbers []string) ([]*ParkingRecord, error)
}
```

- [ ] **Step 2: 实现 Biz 层逻辑**

```go
// internal/user/biz/user.go
func (uc *UserUseCase) GetParkingRecords(ctx context.Context, userID string, req *v1.GetParkingRecordsRequest) (*v1.ParkingRecordsData, error) {
    // 1. 获取用户绑定的车牌
    vehicles, _, err := uc.userRepo.ListUserVehicles(ctx, uuid.MustParse(userID), 1, 100)
    if err != nil {
        return nil, err
    }
    
    // 2. 通过车牌查询停车记录
    var plateNumbers []string
    for _, v := range vehicles {
        plateNumbers = append(plateNumbers, v.PlateNumber)
    }
    
    records, err := uc.vehicleClient.GetParkingRecordsByPlates(ctx, plateNumbers)
    // ...
}
```

- [ ] **Step 3: 在 vehicle service 添加查询接口**

```go
// internal/vehicle/service/vehicle.go
func (s *VehicleService) GetParkingRecordsByPlates(ctx context.Context, req *v1.GetParkingRecordsByPlatesRequest) (*v1.GetParkingRecordsByPlatesResponse, error) {
    records, err := s.uc.GetRecordsByPlates(ctx, req.PlateNumbers)
    // ...
}
```

- [ ] **Step 4: Commit**

```bash
git add internal/user/client/ internal/user/biz/user.go internal/vehicle/
git commit -m "feat(user): implement parking records query"
```

---

### Task 1.3: 实现扫码支付

**Files:**
- Modify: `internal/user/biz/user.go`
- Modify: `internal/user/service/user.go`
- Create: `internal/user/client/payment/client.go`

- [ ] **Step 1: 创建 Payment Client**

```go
// internal/user/client/payment/client.go
type Client struct {
    paymentClient v1.PaymentServiceClient
}

func (c *Client) CreatePayment(ctx context.Context, recordID string, amount float64, openID string) (*v1.PaymentData, error) {
    resp, err := c.paymentClient.CreatePayment(ctx, &v1.CreatePaymentRequest{
        RecordId: recordID,
        Amount: amount,
        PayMethod: "wechat",
        OpenId: openID,
    })
    // ...
}
```

- [ ] **Step 2: 实现扫码支付逻辑**

```go
// internal/user/biz/user.go
func (uc *UserUseCase) CreatePayment(ctx context.Context, userID string, req *v1.CreatePaymentRequest) (*v1.PaymentData, error) {
    // 1. 验证记录存在且未支付
    // 2. 获取用户 openID
    user, _ := uc.userRepo.GetUserByID(ctx, uuid.MustParse(userID))
    
    // 3. 调用 payment service
    return uc.paymentClient.CreatePayment(ctx, req.RecordId, req.Amount, user.OpenID)
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/user/client/payment/ internal/user/biz/user.go
git commit -m "feat(user): implement scan-to-pay functionality"
```

---

## Phase 2: 支付 SDK 集成

### Task 2.1: 微信支付 SDK 集成

**Files:**
- Modify: `internal/payment/wechat/client.go`
- Modify: `internal/payment/biz/callback.go`
- Modify: `configs/payment.yaml`

- [ ] **Step 1: 添加微信支付 SDK 依赖**

```bash
go get github.com/wechatpay-apiv3/wechatpay-go
```

- [ ] **Step 2: 完善 WeChat Client**

```go
// internal/payment/wechat/client.go
import (
    "github.com/wechatpay-apiv3/wechatpay-go/core"
    "github.com/wechatpay-apiv3/wechatpay-go/core/option"
)

type Client struct {
    client *core.Client
    config *Config
}

func NewClient(config *Config) (*Client, error) {
    // 加载商户私钥
    privateKey, err := utils.LoadPrivateKey(config.PrivateKey)
    if err != nil {
        return nil, err
    }
    
    ctx := context.Background()
    opts := []core.ClientOption{
        option.WithWechatPayAutoAuthCipher(config.MchID, config.SerialNo, privateKey, config.APIKey),
    }
    
    client, err := core.NewClient(ctx, opts...)
    if err != nil {
        return nil, err
    }
    
    return &Client{client: client, config: config}, nil
}
```

- [ ] **Step 3: 实现 JSAPI 支付**

```go
func (c *Client) CreateJSAPIPay(ctx context.Context, orderID string, amount int64, openID, description string) (map[string]string, error) {
    svc := jsapi.JsapiApiService{Client: c.client}
    resp, _, err := svc.Prepay(ctx, jsapi.PrepayRequest{
        Appid:       core.String(c.config.AppID),
        Mchid:       core.String(c.config.MchID),
        Description: core.String(description),
        OutTradeNo:  core.String(orderID),
        Amount: &jsapi.Amount{
            Total: core.Int64(amount),
        },
        Payer: &jsapi.Payer{
            Openid: core.String(openID),
        },
        NotifyUrl: core.String(c.config.NotifyURL),
    })
    // 生成前端调起支付参数
    return c.buildPayParams(resp.PrepayId)
}
```

- [ ] **Step 4: 实现回调验签**

```go
// internal/payment/biz/callback.go
import "github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"

func (uc *PaymentUseCase) HandleWechatCallback(ctx context.Context, req *v1.WechatCallbackRequest) (*v1.WechatCallbackResponse, error) {
    // 1. 验证签名
    validator := verifiers.NewSHA256WithRSAVerifier(
        &wechatCertGetter{uc.wechatClient},
    )
    
    // 2. 解密回调数据
    // 3. 幂等性检查
    // 4. 更新订单状态
    // 5. 触发开闸
}
```

- [ ] **Step 5: 配置支付参数**

修改 `configs/payment.yaml`：

```yaml
wechat:
  app_id: "wx_app_id"
  mch_id: "mch_id"
  api_key: "api_key"
  serial_no: "serial_no"
  private_key_path: "/secrets/wechat_private_key.pem"
  notify_url: "https://api.example.com/api/v1/pay/callback/wechat"
  sandbox: true
```

- [ ] **Step 6: Commit**

```bash
git add internal/payment/wechat/ internal/payment/biz/callback.go configs/payment.yaml go.mod go.sum
git commit -m "feat(payment): integrate WeChat Pay SDK with signature verification"
```

---

### Task 2.2: 支付宝 SDK 集成

**Files:**
- Modify: `internal/payment/alipay/client.go`
- Modify: `internal/payment/biz/callback.go`

- [ ] **Step 1: 添加支付宝 SDK 依赖**

```bash
go get github.com/smartwalle/alipay/v3
```

- [ ] **Step 2: 完善 Alipay Client**

```go
// internal/payment/alipay/client.go
import "github.com/smartwalle/alipay/v3"

type Client struct {
    client *alipay.Client
    config *Config
}

func NewClient(config *Config) (*Client, error) {
    client, err := alipay.New(config.AppID, config.PrivateKey, config.IsProduction)
    if err != nil {
        return nil, err
    }
    
    // 加载支付宝公钥
    err = client.LoadAliPayPublicKey(config.AliPublicKey)
    if err != nil {
        return nil, err
    }
    
    return &Client{client: client, config: config}, nil
}
```

- [ ] **Step 3: 实现扫码支付**

```go
func (c *Client) CreateTradePreCreate(ctx context.Context, orderID string, amount float64, subject string) (string, error) {
    var p = alipay.TradePreCreate{
        Subject:     subject,
        OutTradeNo:  orderID,
        TotalAmount: fmt.Sprintf("%.2f", amount),
    }
    
    rsp, err := c.client.TradePreCreate(ctx, p)
    if err != nil {
        return "", err
    }
    
    return rsp.QrCode, nil
}
```

- [ ] **Step 4: 实现回调验签**

```go
func (uc *PaymentUseCase) HandleAlipayCallback(ctx context.Context, req *v1.AlipayCallbackRequest) (*v1.AlipayCallbackResponse, error) {
    // 1. 验证签名
    ok, err := uc.alipayClient.VerifySign(req)
    if !ok || err != nil {
        return nil, fmt.Errorf("signature verification failed")
    }
    
    // 2. 幂等性检查
    // 3. 更新订单状态
    // 4. 触发开闸
}
```

- [ ] **Step 5: Commit**

```bash
git add internal/payment/alipay/ internal/payment/biz/callback.go
git commit -m "feat(payment): integrate Alipay SDK with signature verification"
```

---

## Phase 3: 设备控制服务

### Task 3.1: MQTT 客户端集成

**Files:**
- Modify: `internal/vehicle/data/mqtt/client.go`
- Modify: `internal/vehicle/device/command.go`
- Modify: `configs/vehicle.yaml`

- [ ] **Step 1: 添加 MQTT 依赖**

```bash
go get github.com/eclipse/paho.mqtt.golang
```

- [ ] **Step 2: 完善 MQTT Client**

```go
// internal/vehicle/data/mqtt/client.go
import mqtt "github.com/eclipse/paho.mqtt.golang"

type Client struct {
    client mqtt.Client
    config *Config
}

func NewClient(config *Config) (*Client, error) {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(config.Broker)
    opts.SetClientID(config.ClientID)
    opts.SetUsername(config.Username)
    opts.SetPassword(config.Password)
    opts.SetAutoReconnect(true)
    
    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        return nil, token.Error()
    }
    
    return &Client{client: client, config: config}, nil
}
```

- [ ] **Step 3: 实现指令发布**

```go
func (c *Client) PublishCommand(deviceID string, cmd Command) error {
    topic := fmt.Sprintf("parking/device/%s/command", deviceID)
    payload, _ := json.Marshal(cmd)
    
    token := c.client.Publish(topic, 1, false, payload)
    token.Wait()
    return token.Error()
}

func (c *Client) SubscribeStatus(deviceID string, handler StatusHandler) error {
    topic := fmt.Sprintf("parking/device/%s/status", deviceID)
    token := c.client.Subscribe(topic, 1, func(client mqtt.Client, msg mqtt.Message) {
        handler(msg.Payload())
    })
    token.Wait()
    return token.Error()
}
```

- [ ] **Step 4: 实现设备控制逻辑**

```go
// internal/vehicle/device/command.go
func (s *CommandService) OpenGate(ctx context.Context, deviceID, recordID string) error {
    // 1. 检查设备状态
    device, err := s.repo.GetDeviceByCode(ctx, deviceID)
    if err != nil {
        return err
    }
    if !device.Enabled {
        return fmt.Errorf("device is disabled")
    }
    
    // 2. 发送开闸指令
    cmd := Command{
        Type:     "open_gate",
        RecordID: recordID,
        Timestamp: time.Now().Unix(),
    }
    
    if err := s.mqttClient.PublishCommand(device.GateID, cmd); err != nil {
        // 降级到离线模式
        return s.openGateOffline(ctx, device, recordID)
    }
    
    // 3. 记录操作日志
    return s.repo.RecordCommand(ctx, deviceID, cmd)
}
```

- [ ] **Step 5: 配置 MQTT**

修改 `configs/vehicle.yaml`：

```yaml
mqtt:
  broker: "tcp://mqtt.example.com:1883"
  client_id: "vehicle-service"
  username: "vehicle"
  password: "${MQTT_PASSWORD}"
  topics:
    command: "parking/device/{device_id}/command"
    status: "parking/device/{device_id}/status"
    heartbeat: "parking/device/{device_id}/heartbeat"
```

- [ ] **Step 6: Commit**

```bash
git add internal/vehicle/data/mqtt/ internal/vehicle/device/command.go configs/vehicle.yaml
git commit -m "feat(vehicle): implement MQTT device control"
```

---

### Task 3.2: 离线模式支持

**Files:**
- Modify: `internal/vehicle/device/command.go`
- Modify: `internal/vehicle/data/vehicle.go`

- [ ] **Step 1: 实现离线开闸**

```go
func (s *CommandService) openGateOffline(ctx context.Context, device *Device, recordID string) error {
    // 1. 生成本地流水号
    offlineID := fmt.Sprintf("%s_%d", device.DeviceID, time.Now().Unix())
    
    // 2. 记录离线放行
    record := &OfflineSyncRecord{
        OfflineID:  offlineID,
        RecordID:   recordID,
        DeviceID:   device.DeviceID,
        GateID:     device.GateID,
        OpenTime:   time.Now(),
        SyncStatus: "pending_sync",
    }
    
    return s.repo.CreateOfflineRecord(ctx, record)
}
```

- [ ] **Step 2: 实现同步机制**

```go
func (s *CommandService) SyncOfflineRecords(ctx context.Context) error {
    records, err := s.repo.GetPendingSyncRecords(ctx)
    if err != nil {
        return err
    }
    
    for _, record := range records {
        // 1. 查询订单状态
        order, err := s.paymentClient.GetOrderStatus(ctx, record.RecordID)
        
        // 2. 更新同步状态
        if order.Status == "paid" {
            record.SyncStatus = "synced"
            record.SyncedAt = time.Now()
        } else {
            record.SyncStatus = "sync_failed"
            record.RetryCount++
        }
        
        s.repo.UpdateOfflineRecord(ctx, record)
    }
    
    return nil
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/vehicle/device/command.go internal/vehicle/data/vehicle.go
git commit -m "feat(vehicle): add offline mode and sync mechanism"
```

---

## Phase 4: 计费引擎完善

### Task 4.1: 完善计费规则引擎

**Files:**
- Modify: `internal/billing/biz/billing.go`

- [ ] **Step 1: 完善条件解析**

```go
// 添加更多条件类型支持
func EvaluateCondition(cond *Condition, ctx *BillingContext) bool {
    switch cond.Type {
    // 已有类型...
    
    case "period":
        // 时段优惠 (夜间)
        hour := ctx.ExitTime.Hour()
        start := cond.Value.(map[string]interface{})["start"].(int)
        end := cond.Value.(map[string]interface{})["end"].(int)
        return hour >= start && hour <= end
        
    case "coupon":
        // 优惠券验证
        return ctx.CouponCode == cond.Value.(string)
    }
}
```

- [ ] **Step 2: 完善动作执行**

```go
func applyActions(actions []*Action, duration time.Duration) float64 {
    var amount float64
    
    for _, a := range actions {
        switch a.Type {
        // 已有类型...
        
        case "max_daily":
            // 按天封顶
            days := int(duration.Hours()/24) + 1
            cap := a.Amount * float64(days)
            if amount > cap {
                amount = cap
            }
            
        case "free_duration":
            // 免费时长 (月卡)
            freeHours := a.Value / 3600
            if duration.Hours() <= freeHours {
                amount = 0
            }
        }
    }
    
    return amount
}
```

- [ ] **Step 3: Commit**

```bash
git add internal/billing/biz/billing.go
git commit -m "feat(billing): enhance billing rule engine with period and cap support"
```

---

## Phase 5: 集成测试

### Task 5.1: 编写端到端测试

**Files:**
- Create: `tests/e2e/parking_flow_test.go`

- [ ] **Step 1: 实现完整停车流程测试**

```go
func TestParkingFlow(t *testing.T) {
    ctx := context.Background()
    
    // 1. 车辆入场
    entryResp, err := vehicleClient.Entry(ctx, &v1.EntryRequest{
        DeviceId:    "lane_001",
        PlateNumber: "京A12345",
        Confidence:  0.95,
    })
    require.NoError(t, err)
    assert.True(t, entryResp.Data.GateOpen)
    
    recordID := entryResp.Data.RecordId
    
    // 2. 车辆出场
    exitResp, err := vehicleClient.Exit(ctx, &v1.ExitRequest{
        DeviceId:    "lane_002",
        PlateNumber: "京A12345",
        Confidence:  0.93,
    })
    require.NoError(t, err)
    assert.False(t, exitResp.Data.GateOpen) // 等待支付
    assert.Greater(t, exitResp.Data.FinalAmount, 0.0)
    
    // 3. 创建支付
    payResp, err := paymentClient.CreatePayment(ctx, &paymentv1.CreatePaymentRequest{
        RecordId: recordID,
        Amount:   exitResp.Data.FinalAmount,
        PayMethod: "wechat",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, payResp.Data.PayUrl)
    
    // 4. 模拟支付回调
    // ...
}
```

- [ ] **Step 2: 运行测试**

```bash
# 启动基础设施
docker-compose -f deploy/docker-compose.yml up -d postgres redis etcd

# 运行测试
go test -v ./tests/e2e/... -run TestParkingFlow
```

- [ ] **Step 3: Commit**

```bash
git add tests/e2e/parking_flow_test.go
git commit -m "test(e2e): add parking flow end-to-end test"
```

---

## 执行策略

### 推荐执行顺序

```
Phase 1 (车主端 API)
├── Task 1.1 (User Service 结构)
├── Task 1.2 (停车记录查询)
└── Task 1.3 (扫码支付)

Phase 2 (支付 SDK)
├── Task 2.1 (微信支付)
└── Task 2.2 (支付宝)

Phase 3 (设备控制)
├── Task 3.1 (MQTT 集成)
└── Task 3.2 (离线模式)

Phase 4 (计费引擎)
└── Task 4.1 (规则引擎)

Phase 5 (测试)
└── Task 5.1 (E2E 测试)
```

### 并行执行建议

- Task 1.1 和 Task 2.1 可以并行（不同服务）
- Task 3.1 和 Task 4.1 可以并行（不同服务）

### 依赖关系

- Task 1.3 依赖 Task 2.1（扫码支付需要微信支付）
- Task 5.1 依赖所有其他 Task（集成测试）

---

## 验收标准

### 功能验收

- [ ] 车主可以通过微信登录
- [ ] 车主可以绑定/解绑车牌
- [ ] 车主可以查询停车记录
- [ ] 车主可以扫码支付
- [ ] 支付成功后自动开闸
- [ ] 设备离线时可以本地放行
- [ ] 网络恢复后自动同步

### 技术验收

- [ ] 所有 P0 任务完成
- [ ] 单元测试覆盖率 > 60%
- [ ] E2E 测试通过
- [ ] 代码通过 lint 检查
- [ ] 可以 Docker Compose 一键启动

---

## 风险与应对

| 风险 | 可能性 | 影响 | 应对措施 |
|------|--------|------|----------|
| 微信支付 SDK 集成复杂 | 中 | 高 | 先用沙箱环境测试，文档参考官方示例 |
| MQTT 设备协议不统一 | 中 | 中 | 定义标准协议格式，适配层处理差异 |
| 性能瓶颈 | 低 | 中 | 提前设计缓存策略，压测验证 |

---

## 附录：常用命令

```bash
# 启动基础设施
docker-compose -f deploy/docker-compose.yml up -d postgres redis etcd jaeger

# 生成 proto
make proto

# 运行服务
go run ./cmd/gateway -conf ./configs
go run ./cmd/vehicle -conf ./configs
go run ./cmd/billing -conf ./configs
go run ./cmd/payment -conf ./configs
go run ./cmd/admin -conf ./configs
go run ./cmd/user -conf ./configs

# 运行测试
go test -v ./...
go test -v ./tests/e2e/...

# 构建镜像
docker-compose -f deploy/docker-compose.yml build
```

# Smart Park 开发计划 - P0 阶段（必须完成）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 完成支付安全、车主端 API、设备控制、配置安全等核心功能，使系统具备基本上线条件

**Architecture:** 
- 支付服务：集成微信支付和支付宝官方 SDK，实现完整的签名验证和回调处理
- 车主端服务：新增 user service，实现 JWT 认证、车牌管理、支付功能
- 设备控制：集成 MQTT，实现真实的设备指令下发
- 配置管理：使用环境变量和 Vault 管理敏感信息

**Tech Stack:** 
- Go 1.26 + Kratos v2.9.2
- 微信支付 SDK: github.com/wechatpay-apiv3/wechatpay-go
- 支付宝 SDK: github.com/smartwalle/alipay/v3
- MQTT: github.com/eclipse/paho.mqtt.golang (已依赖)
- JWT: github.com/golang-jwt/jwt/v5

---

## Phase 1: 支付安全（预计 3-4 天）

### Task 1.1: 集成微信支付 SDK

**Files:**
- Create: `internal/payment/wechat/client.go`
- Create: `internal/payment/wechat/client_test.go`
- Modify: `internal/payment/biz/payment.go`
- Modify: `configs/payment.yaml`

**Context:** 当前支付回调验签是占位实现，存在安全隐患。需要集成官方 SDK 实现完整的签名验证。

- [ ] **Step 1: 添加微信支付 SDK 依赖**

Run: `go get github.com/wechatpay-apiv3/wechatpay-go@latest`

Expected: 依赖添加成功

- [ ] **Step 2: 创建微信支付客户端配置结构**

Create: `internal/payment/wechat/config.go`

```go
package wechat

import (
    "github.com/wechatpay-apiv3/wechatpay-go/core"
    "github.com/wechatpay-apiv3/wechatpay-go/core/option"
)

type Config struct {
    AppID       string
    MchID       string
    APIKey      string
    CertSerialNo string
    PrivateKey  string
    PublicKey   string
    NotifyURL   string
}

func NewClient(cfg *Config) (*core.APIV3Client, error) {
    // 使用官方 SDK 创建客户端
    // 实现证书加载和签名验证
}
```

- [ ] **Step 3: 实现微信支付下单接口**

Create: `internal/payment/wechat/client.go`

```go
package wechat

import (
    "context"
    "github.com/wechatpay-apiv3/wechatpay-go/services/payments/native"
    "github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
)

type Client struct {
    client *core.APIV3Client
    config *Config
}

func (c *Client) CreateNativePay(ctx context.Context, orderID string, amount int64, description string) (string, error) {
    // 实现扫码支付下单
    // 返回二维码链接
}

func (c *Client) CreateJSAPIPay(ctx context.Context, orderID string, amount int64, openID string) (map[string]interface{}, error) {
    // 实现小程序/公众号支付
    // 返回前端调起支付的参数
}
```

- [ ] **Step 4: 实现微信支付回调验签**

Create: `internal/payment/wechat/notify.go`

```go
package wechat

import (
    "context"
    "net/http"
    "github.com/wechatpay-apiv3/wechatpay-go/core/notify"
)

type NotifyHandler struct {
    handler *notify.Handler
}

func NewNotifyHandler(cfg *Config) (*NotifyHandler, error) {
    // 创建验签处理器
}

func (h *NotifyHandler) ParseNotifyRequest(req *http.Request) (*notify.Transaction, error) {
    // 解析并验证回调请求
    // 自动验证签名
}
```

- [ ] **Step 5: 编写微信支付客户端测试**

Create: `internal/payment/wechat/client_test.go`

```go
package wechat_test

import (
    "testing"
    "github.com/xuanyiying/smart-park/internal/payment/wechat"
)

func TestNewClient(t *testing.T) {
    cfg := &wechat.Config{
        AppID: "test_appid",
        MchID: "test_mchid",
        // 测试配置
    }
    
    client, err := wechat.NewClient(cfg)
    if err != nil {
        t.Fatalf("failed to create client: %v", err)
    }
    
    if client == nil {
        t.Error("client should not be nil")
    }
}

func TestCreateNativePay(t *testing.T) {
    // 测试扫码支付下单
}

func TestNotifyHandler(t *testing.T) {
    // 测试回调验签
}
```

- [ ] **Step 6: 运行测试验证**

Run: `go test ./internal/payment/wechat/... -v`

Expected: 所有测试通过

- [ ] **Step 7: 更新 payment.yaml 配置**

Modify: `configs/payment.yaml`

```yaml
wechat:
  app_id: ${WECHAT_APP_ID}
  mch_id: ${WECHAT_MCH_ID}
  api_key: ${WECHAT_API_KEY}
  cert_serial_no: ${WECHAT_CERT_SERIAL_NO}
  private_key_path: ${WECHAT_PRIVATE_KEY_PATH}
  public_key_path: ${WECHAT_PUBLIC_KEY_PATH}
  notify_url: ${WECHAT_NOTIFY_URL}
```

- [ ] **Step 8: 重构 PaymentUseCase 使用真实 SDK**

Modify: `internal/payment/biz/payment.go`

```go
// 替换占位实现
func (uc *PaymentUseCase) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.PaymentData, error) {
    // 使用 wechat.Client 创建真实订单
    if req.PayMethod == "wechat" {
        payURL, qrCode, err := uc.wechatClient.CreateNativePay(ctx, orderID, amount, description)
        // 错误处理
    }
}

func (uc *PaymentUseCase) HandleWechatCallback(ctx context.Context, req *v1.WechatCallbackRequest) (*v1.WechatCallbackResponse, error) {
    // 使用 notify handler 验证签名
    transaction, err := uc.notifyHandler.ParseNotifyRequest(httpRequest)
    if err != nil {
        // 验签失败，返回错误
    }
    // 更新订单状态
}
```

- [ ] **Step 9: 提交代码**

```bash
git add internal/payment/wechat/ configs/payment.yaml internal/payment/biz/payment.go
git commit -m "feat(payment): integrate WeChat Pay SDK with signature verification"
```

### Task 1.2: 集成支付宝 SDK

**Files:**
- Create: `internal/payment/alipay/client.go`
- Create: `internal/payment/alipay/client_test.go`
- Modify: `internal/payment/biz/payment.go`
- Modify: `configs/payment.yaml`

- [ ] **Step 1: 添加支付宝 SDK 依赖**

Run: `go get github.com/smartwalle/alipay/v3@latest`

- [ ] **Step 2: 创建支付宝客户端**

Create: `internal/payment/alipay/client.go`

```go
package alipay

import (
    "context"
    "github.com/smartwalle/alipay/v3"
)

type Config struct {
    AppID        string
    PrivateKey   string
    PublicKey    string
    NotifyURL    string
    IsProduction bool
}

type Client struct {
    client *alipay.Client
    config *Config
}

func NewClient(cfg *Config) (*Client, error) {
    // 创建支付宝客户端
    client, err := alipay.New(cfg.AppID, cfg.PrivateKey, cfg.IsProduction)
    // 加载公钥
    // 返回客户端
}

func (c *Client) CreateTradePagePay(ctx context.Context, orderID string, amount float64, subject string) (string, error) {
    // PC 网页支付
}

func (c *Client) CreateTradeWapPay(ctx context.Context, orderID string, amount float64, subject string) (string, error) {
    // 移动端网页支付
}

func (c *Client) CreateTradePreCreate(ctx context.Context, orderID string, amount float64, subject string) (string, error) {
    // 扫码支付，返回二维码
}

func (c *Client) VerifyNotification(params url.Values) (*alipay.Notification, error) {
    // 验证回调签名
}
```

- [ ] **Step 3: 编写支付宝客户端测试**

Create: `internal/payment/alipay/client_test.go`

```go
package alipay_test

import (
    "testing"
    "github.com/xuanyiying/smart-park/internal/payment/alipay"
)

func TestNewClient(t *testing.T) {
    // 测试客户端创建
}

func TestCreateTradePreCreate(t *testing.T) {
    // 测试扫码支付
}

func TestVerifyNotification(t *testing.T) {
    // 测试回调验签
}
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/payment/alipay/... -v`

- [ ] **Step 5: 更新配置文件**

Modify: `configs/payment.yaml`

```yaml
alipay:
  app_id: ${ALIPAY_APP_ID}
  private_key: ${ALIPAY_PRIVATE_KEY}
  public_key: ${ALIPAY_PUBLIC_KEY}
  notify_url: ${ALIPAY_NOTIFY_URL}
  is_production: ${ALIPAY_IS_PRODUCTION:false}
```

- [ ] **Step 6: 集成到 PaymentUseCase**

Modify: `internal/payment/biz/payment.go`

```go
// 添加支付宝客户端
type PaymentUseCase struct {
    orderRepo    OrderRepo
    wechatClient *wechat.Client
    alipayClient *alipay.Client  // 新增
    log          *log.Helper
}

// 实现支付宝支付
func (uc *PaymentUseCase) CreatePayment(ctx context.Context, req *v1.CreatePaymentRequest) (*v1.PaymentData, error) {
    if req.PayMethod == "alipay" {
        qrCode, err := uc.alipayClient.CreateTradePreCreate(ctx, orderID, amount, subject)
        // 错误处理
    }
}

// 实现支付宝回调
func (uc *PaymentUseCase) HandleAlipayCallback(ctx context.Context, params url.Values) error {
    notification, err := uc.alipayClient.VerifyNotification(params)
    if err != nil {
        // 验签失败
    }
    // 更新订单状态
}
```

- [ ] **Step 7: 提交代码**

```bash
git add internal/payment/alipay/ configs/payment.yaml internal/payment/biz/payment.go
git commit -m "feat(payment): integrate Alipay SDK with signature verification"
```

### Task 1.3: 添加支付安全测试

**Files:**
- Create: `internal/payment/biz/payment_security_test.go`

- [ ] **Step 1: 编写签名验证测试**

Create: `internal/payment/biz/payment_security_test.go`

```go
package biz_test

import (
    "testing"
    "net/http"
    "net/http/httptest"
)

func TestWechatCallbackSignatureVerification(t *testing.T) {
    // 测试微信回调签名验证
    // 1. 构造合法签名请求
    // 2. 构造非法签名请求
    // 3. 验证处理结果
}

func TestAlipayCallbackSignatureVerification(t *testing.T) {
    // 测试支付宝回调签名验证
}

func TestPaymentIdempotency(t *testing.T) {
    // 测试支付幂等性
    // 防止重复支付
}

func TestPaymentAmountValidation(t *testing.T) {
    // 测试金额校验
    // 防止金额篡改
}
```

- [ ] **Step 2: 运行安全测试**

Run: `go test ./internal/payment/biz/... -v -run Security`

- [ ] **Step 3: 提交测试代码**

```bash
git add internal/payment/biz/payment_security_test.go
git commit -m "test(payment): add security tests for payment callback verification"
```

---

## Phase 2: 车主端 API（预计 5-7 天）

### Task 2.1: 创建用户服务 Proto 定义

**Files:**
- Create: `api/user/v1/user.proto`
- Create: `api/user/v1/user.pb.go` (生成)
- Create: `api/user/v1/user_grpc.pb.go` (生成)

- [ ] **Step 1: 编写 user.proto 定义**

Create: `api/user/v1/user.proto`

```protobuf
syntax = "proto3";

package api.user.v1;

option go_package = "github.com/xuanyiying/smart-park/api/user/v1;v1";

import "google/api/annotations.proto";
import "google/protobuf/empty.proto";

service UserService {
  // 认证相关
  rpc Login (LoginRequest) returns (LoginResponse) {
    option (google.api.http) = {
      post: "/api/v1/user/login"
      body: "*"
    };
  }
  
  rpc GetUserInfo (GetUserInfoRequest) returns (GetUserInfoResponse) {
    option (google.api.http) = {
      get: "/api/v1/user/info"
    };
  }
  
  // 车牌管理
  rpc BindPlate (BindPlateRequest) returns (BindPlateResponse) {
    option (google.api.http) = {
      post: "/api/v1/user/plates"
      body: "*"
    };
  }
  
  rpc UnbindPlate (UnbindPlateRequest) returns (UnbindPlateResponse) {
    option (google.api.http) = {
      delete: "/api/v1/user/plates/{plate_number}"
    };
  }
  
  rpc ListPlates (ListPlatesRequest) returns (ListPlatesResponse) {
    option (google.api.http) = {
      get: "/api/v1/user/plates"
    };
  }
  
  // 停车记录
  rpc ListParkingRecords (ListParkingRecordsRequest) returns (ListParkingRecordsResponse) {
    option (google.api.http) = {
      get: "/api/v1/user/parking-records"
    };
  }
  
  rpc GetParkingRecord (GetParkingRecordRequest) returns (GetParkingRecordResponse) {
    option (google.api.http) = {
      get: "/api/v1/user/parking-records/{record_id}"
    };
  }
  
  // 支付相关
  rpc ScanPay (ScanPayRequest) returns (ScanPayResponse) {
    option (google.api.http) = {
      post: "/api/v1/user/scan-pay"
      body: "*"
    };
  }
  
  // 月卡相关
  rpc GetMonthlyCard (GetMonthlyCardRequest) returns (GetMonthlyCardResponse) {
    option (google.api.http) = {
      get: "/api/v1/user/monthly-card/{plate_number}"
    };
  }
  
  rpc PurchaseMonthlyCard (PurchaseMonthlyCardRequest) returns (PurchaseMonthlyCardResponse) {
    option (google.api.http) = {
      post: "/api/v1/user/monthly-card"
      body: "*"
    };
  }
}

message LoginRequest {
  string code = 1;  // 微信小程序 code
}

message LoginResponse {
  int32 code = 1;
  string message = 2;
  LoginData data = 3;
}

message LoginData {
  string token = 1;
  string open_id = 2;
  int64 expires_at = 3;
}

message GetUserInfoRequest {}

message GetUserInfoResponse {
  int32 code = 1;
  string message = 2;
  UserInfo data = 3;
}

message UserInfo {
  string user_id = 1;
  string open_id = 2;
  string nickname = 3;
  string avatar = 4;
  string phone = 5;
  int64 created_at = 6;
}

message BindPlateRequest {
  string plate_number = 1;
  string owner_name = 2;
  string owner_phone = 3;
}

message BindPlateResponse {
  int32 code = 1;
  string message = 2;
}

message UnbindPlateRequest {
  string plate_number = 1;
}

message UnbindPlateResponse {
  int32 code = 1;
  string message = 2;
}

message ListPlatesRequest {
  int32 page = 1;
  int32 page_size = 2;
}

message ListPlatesResponse {
  int32 code = 1;
  string message = 2;
  ListPlatesData data = 3;
}

message ListPlatesData {
  repeated PlateInfo plates = 1;
  int32 total = 2;
}

message PlateInfo {
  string plate_number = 1;
  string vehicle_type = 2;
  string owner_name = 3;
  string owner_phone = 4;
  string monthly_valid_until = 5;
}

message ListParkingRecordsRequest {
  string plate_number = 1;
  int32 page = 2;
  int32 page_size = 3;
}

message ListParkingRecordsResponse {
  int32 code = 1;
  string message = 2;
  ListParkingRecordsData data = 3;
}

message ListParkingRecordsData {
  repeated ParkingRecordInfo records = 1;
  int32 total = 2;
}

message ParkingRecordInfo {
  string record_id = 1;
  string plate_number = 2;
  string lot_name = 3;
  string entry_time = 4;
  string exit_time = 5;
  int32 duration = 6;
  double amount = 7;
  string status = 8;
}

message GetParkingRecordRequest {
  string record_id = 1;
}

message GetParkingRecordResponse {
  int32 code = 1;
  string message = 2;
  ParkingRecordInfo data = 3;
}

message ScanPayRequest {
  string record_id = 1;
  string pay_method = 2;  // wechat/alipay
  string open_id = 3;     // 微信支付需要
}

message ScanPayResponse {
  int32 code = 1;
  string message = 2;
  ScanPayData data = 3;
}

message ScanPayData {
  string order_id = 1;
  double amount = 2;
  string pay_url = 3;
  string qr_code = 4;
  string expire_time = 5;
}

message GetMonthlyCardRequest {
  string plate_number = 1;
}

message GetMonthlyCardResponse {
  int32 code = 1;
  string message = 2;
  MonthlyCardInfo data = 3;
}

message MonthlyCardInfo {
  string plate_number = 1;
  string valid_until = 2;
  int32 days_remaining = 3;
  bool is_valid = 4;
}

message PurchaseMonthlyCardRequest {
  string plate_number = 1;
  int32 months = 2;
  string pay_method = 3;
  string open_id = 4;
}

message PurchaseMonthlyCardResponse {
  int32 code = 1;
  string message = 2;
  PurchaseMonthlyCardData data = 3;
}

message PurchaseMonthlyCardData {
  string order_id = 1;
  double amount = 2;
  string pay_url = 3;
  string qr_code = 4;
}
```

- [ ] **Step 2: 生成 Proto 代码**

Run: `./scripts/generate_proto.sh`

Expected: 生成 user.pb.go 和 user_grpc.pb.go

- [ ] **Step 3: 验证生成文件**

Run: `ls -la api/user/v1/`

Expected: 看到 user.pb.go, user_grpc.pb.go, user_http.pb.go

- [ ] **Step 4: 提交 Proto 定义**

```bash
git add api/user/v1/
git commit -m "feat(user): add user service proto definition for car owner APIs"
```

### Task 2.2: 实现 JWT 认证中间件

**Files:**
- Create: `pkg/auth/jwt.go`
- Create: `pkg/auth/jwt_test.go`
- Create: `internal/user/middleware/auth.go`

- [ ] **Step 1: 添加 JWT 依赖**

Run: `go get github.com/golang-jwt/jwt/v5@latest`

- [ ] **Step 2: 实现 JWT 工具类**

Create: `pkg/auth/jwt.go`

```go
package auth

import (
    "errors"
    "time"
    
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID string `json:"user_id"`
    OpenID string `json:"open_id"`
    jwt.RegisteredClaims
}

type JWTConfig struct {
    SecretKey     string
    TokenDuration time.Duration
}

type JWTManager struct {
    config *JWTConfig
}

func NewJWTManager(config *JWTConfig) *JWTManager {
    return &JWTManager{config: config}
}

func (m *JWTManager) GenerateToken(userID, openID string) (string, error) {
    claims := &Claims{
        UserID: userID,
        OpenID: openID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.TokenDuration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Issuer:    "smart-park",
        },
    }
    
    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString([]byte(m.config.SecretKey))
}

func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        return []byte(m.config.SecretKey), nil
    })
    
    if err != nil {
        return nil, err
    }
    
    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }
    
    return nil, errors.New("invalid token")
}
```

- [ ] **Step 3: 编写 JWT 测试**

Create: `pkg/auth/jwt_test.go`

```go
package auth_test

import (
    "testing"
    "time"
    "github.com/xuanyiying/smart-park/pkg/auth"
)

func TestGenerateAndParseToken(t *testing.T) {
    config := &auth.JWTConfig{
        SecretKey:     "test-secret-key",
        TokenDuration: time.Hour,
    }
    
    manager := auth.NewJWTManager(config)
    
    token, err := manager.GenerateToken("user123", "open123")
    if err != nil {
        t.Fatalf("failed to generate token: %v", err)
    }
    
    claims, err := manager.ParseToken(token)
    if err != nil {
        t.Fatalf("failed to parse token: %v", err)
    }
    
    if claims.UserID != "user123" {
        t.Errorf("expected user_id user123, got %s", claims.UserID)
    }
    
    if claims.OpenID != "open123" {
        t.Errorf("expected open_id open123, got %s", claims.OpenID)
    }
}

func TestExpiredToken(t *testing.T) {
    // 测试过期 token
}

func TestInvalidToken(t *testing.T) {
    // 测试无效 token
}
```

- [ ] **Step 4: 运行测试**

Run: `go test ./pkg/auth/... -v`

- [ ] **Step 5: 实现认证中间件**

Create: `internal/user/middleware/auth.go`

```go
package middleware

import (
    "context"
    "strings"
    
    "github.com/go-kratos/kratos/v2/middleware"
    "github.com/go-kratos/kratos/v2/transport"
    
    "github.com/xuanyiying/smart-park/pkg/auth"
)

func Auth(jwtManager *auth.JWTManager) middleware.Middleware {
    return func(handler middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req interface{}) (interface{}, error) {
            // 从 header 获取 token
            if tr, ok := transport.FromServerContext(ctx); ok {
                authHeader := tr.RequestHeader().Get("Authorization")
                if authHeader == "" {
                    return nil, errors.New("missing authorization header")
                }
                
                tokenString := strings.TrimPrefix(authHeader, "Bearer ")
                claims, err := jwtManager.ParseToken(tokenString)
                if err != nil {
                    return nil, errors.New("invalid token")
                }
                
                // 将用户信息存入 context
                ctx = context.WithValue(ctx, "user_id", claims.UserID)
                ctx = context.WithValue(ctx, "open_id", claims.OpenID)
            }
            
            return handler(ctx, req)
        }
    }
}
```

- [ ] **Step 6: 提交代码**

```bash
git add pkg/auth/ internal/user/middleware/
git commit -m "feat(auth): implement JWT authentication middleware"
```

### Task 2.3: 实现用户服务业务逻辑

**Files:**
- Create: `internal/user/biz/user.go`
- Create: `internal/user/biz/user_test.go`
- Create: `internal/user/data/user.go`
- Create: `internal/user/service/user.go`
- Create: `cmd/user/main.go`
- Create: `configs/user.yaml`

**注意**: 这是一个较大的任务，需要创建完整的 user service

- [ ] **Step 1: 创建 user service 目录结构**

Run: `mkdir -p internal/user/{biz,data,service} cmd/user`

- [ ] **Step 2: 实现 UserUseCase**

Create: `internal/user/biz/user.go`

```go
package biz

import (
    "context"
    "time"
    
    "github.com/go-kratos/kratos/v2/log"
    "github.com/google/uuid"
    
    v1 "github.com/xuanyiying/smart-park/api/user/v1"
    "github.com/xuanyiying/smart-park/pkg/auth"
)

type User struct {
    ID        uuid.UUID
    OpenID    string
    Nickname  string
    Avatar    string
    Phone     string
    CreatedAt time.Time
    UpdatedAt time.Time
}

type UserVehicle struct {
    ID          uuid.UUID
    UserID      uuid.UUID
    PlateNumber string
    OwnerName   string
    OwnerPhone  string
    CreatedAt   time.Time
}

type UserRepo interface {
    GetUserByOpenID(ctx context.Context, openID string) (*User, error)
    CreateUser(ctx context.Context, user *User) error
    GetUserByID(ctx context.Context, userID uuid.UUID) (*User, error)
    
    BindVehicle(ctx context.Context, userVehicle *UserVehicle) error
    UnbindVehicle(ctx context.Context, userID uuid.UUID, plateNumber string) error
    ListUserVehicles(ctx context.Context, userID uuid.UUID, page, pageSize int) ([]*UserVehicle, int64, error)
}

type UserUseCase struct {
    userRepo   UserRepo
    jwtManager *auth.JWTManager
    log        *log.Helper
}

func NewUserUseCase(userRepo UserRepo, jwtManager *auth.JWTManager, logger log.Logger) *UserUseCase {
    return &UserUseCase{
        userRepo:   userRepo,
        jwtManager: jwtManager,
        log:        log.NewHelper(logger),
    }
}

func (uc *UserUseCase) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginData, error) {
    // 1. 使用 code 换取 openID（调用微信 API）
    openID, err := uc.getOpenIDFromWechat(ctx, req.Code)
    if err != nil {
        return nil, err
    }
    
    // 2. 查询或创建用户
    user, err := uc.userRepo.GetUserByOpenID(ctx, openID)
    if err != nil {
        // 用户不存在，创建新用户
        user = &User{
            ID:     uuid.New(),
            OpenID: openID,
        }
        if err := uc.userRepo.CreateUser(ctx, user); err != nil {
            return nil, err
        }
    }
    
    // 3. 生成 JWT token
    token, err := uc.jwtManager.GenerateToken(user.ID.String(), user.OpenID)
    if err != nil {
        return nil, err
    }
    
    return &v1.LoginData{
        Token:     token,
        OpenId:    openID,
        ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
    }, nil
}

func (uc *UserUseCase) BindPlate(ctx context.Context, userID string, req *v1.BindPlateRequest) error {
    uid, err := uuid.Parse(userID)
    if err != nil {
        return err
    }
    
    userVehicle := &UserVehicle{
        ID:          uuid.New(),
        UserID:      uid,
        PlateNumber: req.PlateNumber,
        OwnerName:   req.OwnerName,
        OwnerPhone:  req.OwnerPhone,
    }
    
    return uc.userRepo.BindVehicle(ctx, userVehicle)
}

// 其他方法实现...
```

- [ ] **Step 3: 实现 UserRepository**

Create: `internal/user/data/user.go`

```go
package data

import (
    "context"
    
    "github.com/xuanyiying/smart-park/internal/user/biz"
    "github.com/xuanyiying/smart-park/internal/user/data/ent"
    "github.com/xuanyiying/smart-park/internal/user/data/ent/user"
    "github.com/xuanyiying/smart-park/internal/user/data/ent/uservehicle"
)

type userRepo struct {
    data *Data
}

func NewUserRepo(data *Data) biz.UserRepo {
    return &userRepo{data: data}
}

func (r *userRepo) GetUserByOpenID(ctx context.Context, openID string) (*biz.User, error) {
    u, err := r.data.db.User.Query().
        Where(user.OpenID(openID)).
        Only(ctx)
    
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, nil
        }
        return nil, err
    }
    
    return &biz.User{
        ID:        u.ID,
        OpenID:    u.OpenID,
        Nickname:  u.Nickname,
        Avatar:    u.Avatar,
        Phone:     u.Phone,
        CreatedAt: u.CreatedAt,
    }, nil
}

// 其他方法实现...
```

- [ ] **Step 4: 实现 UserService**

Create: `internal/user/service/user.go`

```go
package service

import (
    "context"
    
    v1 "github.com/xuanyiying/smart-park/api/user/v1"
    "github.com/xuanyiying/smart-park/internal/user/biz"
)

type UserService struct {
    v1.UnimplementedUserServiceServer
  
    uc *biz.UserUseCase
}

func NewUserService(uc *biz.UserUseCase) *UserService {
    return &UserService{uc: uc}
}

func (s *UserService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
    data, err := s.uc.Login(ctx, req)
    if err != nil {
        return &v1.LoginResponse{
            Code:    500,
            Message: err.Error(),
        }, nil
    }
    
    return &v1.LoginResponse{
        Code:    200,
        Message: "success",
        Data:    data,
    }, nil
}

// 其他方法实现...
```

- [ ] **Step 5: 创建服务启动文件**

Create: `cmd/user/main.go`

```go
package main

import (
    "flag"
    
    "github.com/go-kratos/kratos/v2"
    "github.com/go-kratos/kratos/v2/log"
    
    "github.com/xuanyiying/smart-park/internal/user/service"
)

var (
    flagconf string
)

func init() {
    flag.StringVar(&flagconf, "conf", "../../configs", "config path")
}

func main() {
    flag.Parse()
  
    // 初始化配置、数据库、依赖注入等
    // 参考 cmd/vehicle/main.go
  
    app := kratos.New(
        kratos.Name("user-svc"),
        // 其他配置
    )
  
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

- [ ] **Step 6: 创建配置文件**

Create: `configs/user.yaml`

```yaml
server:
  port: 8005
  timeout: 60

database:
  driver: postgres
  source: "host=${DB_HOST} user=${DB_USER} password=${DB_PASSWORD} dbname=parking port=${DB_PORT} sslmode=disable"

redis:
  addr: "${REDIS_ADDR}"
  password: "${REDIS_PASSWORD}"
  db: 0

jwt:
  secret_key: ${JWT_SECRET_KEY}
  token_duration: 24h

wechat:
  app_id: ${WECHAT_MINI_APP_ID}
  app_secret: ${WECHAT_MINI_APP_SECRET}

log:
  level: info
  format: json

otel:
  endpoint: "${OTEL_ENDPOINT}"
  serviceName: "user-svc"
```

- [ ] **Step 7: 编写测试**

Create: `internal/user/biz/user_test.go`

```go
package biz_test

import (
    "context"
    "testing"
    
    "github.com/xuanyiying/smart-park/internal/user/biz"
)

func TestUserLogin(t *testing.T) {
    // 测试用户登录
}

func TestBindPlate(t *testing.T) {
    // 测试车牌绑定
}

// 其他测试...
```

- [ ] **Step 8: 运行测试**

Run: `go test ./internal/user/... -v`

- [ ] **Step 9: 提交代码**

```bash
git add internal/user/ cmd/user/ configs/user.yaml
git commit -m "feat(user): implement user service with JWT authentication and plate management"
```

### Task 2.4: 更新 Gateway 路由

**Files:**
- Modify: `internal/gateway/biz/router.go`
- Modify: `configs/gateway.yaml`

- [ ] **Step 1: 添加 user service 路由**

Modify: `internal/gateway/biz/router.go`

```go
// 添加 user service 路由
routes := []*RouteConfig{
    {Path: "/api/v1/device", Target: "vehicle-svc:8001"},
    {Path: "/api/v1/billing", Target: "billing-svc:8002"},
    {Path: "/api/v1/pay", Target: "payment-svc:8003"},
    {Path: "/api/v1/admin", Target: "admin-svc:8004"},
    {Path: "/api/v1/user", Target: "user-svc:8005"},  // 新增
}
```

- [ ] **Step 2: 更新 gateway 配置**

Modify: `configs/gateway.yaml`

```yaml
routes:
  - path: /api/v1/device
    target: vehicle-svc:8001
  - path: /api/v1/billing
    target: billing-svc:8002
  - path: /api/v1/pay
    target: payment-svc:8003
  - path: /api/v1/admin
    target: admin-svc:8004
  - path: /api/v1/user
    target: user-svc:8005
```

- [ ] **Step 3: 提交代码**

```bash
git add internal/gateway/ configs/gateway.yaml
git commit -m "feat(gateway): add user service routing"
```

---

## Phase 3: 设备控制（预计 3-4 天）

### Task 3.1: 实现 MQTT 客户端

**Files:**
- Create: `pkg/mqtt/client.go`
- Create: `pkg/mqtt/client_test.go`
- Modify: `internal/vehicle/data/mqtt/client.go`

- [ ] **Step 1: 创建通用 MQTT 客户端**

Create: `pkg/mqtt/client.go`

```go
package mqtt

import (
    "context"
    "fmt"
    "sync"
    "time"
    
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/go-kratos/kratos/v2/log"
)

type Config struct {
    Broker   string
    ClientID string
    Username string
    Password string
    QoS      byte
}

type Client interface {
    Connect(ctx context.Context) error
    Disconnect()
    Publish(ctx context.Context, topic string, payload []byte) error
    Subscribe(ctx context.Context, topic string, handler mqtt.MessageHandler) error
    Unsubscribe(ctx context.Context, topic string) error
}

type MQTTClient struct {
    client mqtt.Client
    config *Config
    log    *log.Helper
    mu     sync.RWMutex
}

func NewClient(config *Config, logger log.Logger) *MQTTClient {
    opts := mqtt.NewClientOptions()
    opts.AddBroker(config.Broker)
    opts.SetClientID(config.ClientID)
    opts.SetUsername(config.Username)
    opts.SetPassword(config.Password)
    opts.SetAutoReconnect(true)
    opts.SetCleanSession(true)
    
    client := mqtt.NewClient(opts)
    
    return &MQTTClient{
        client: client,
        config: config,
        log:    log.NewHelper(logger),
    }
}

func (c *MQTTClient) Connect(ctx context.Context) error {
    if token := c.client.Connect(); token.Wait() && token.Error() != nil {
        return token.Error()
    }
    c.log.Info("MQTT client connected")
    return nil
}

func (c *MQTTClient) Disconnect() {
    c.client.Disconnect(250)
    c.log.Info("MQTT client disconnected")
}

func (c *MQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
    token := c.client.Publish(topic, c.config.QoS, false, payload)
    if token.Wait() && token.Error() != nil {
        return token.Error()
    }
    c.log.Infof("Published to topic %s", topic)
    return nil
}

func (c *MQTTClient) Subscribe(ctx context.Context, topic string, handler mqtt.MessageHandler) error {
    token := c.client.Subscribe(topic, c.config.QoS, handler)
    if token.Wait() && token.Error() != nil {
        return token.Error()
    }
    c.log.Infof("Subscribed to topic %s", topic)
    return nil
}

func (c *MQTTClient) Unsubscribe(ctx context.Context, topic string) error {
    token := c.client.Unsubscribe(topic)
    if token.Wait() && token.Error() != nil {
        return token.Error()
    }
    c.log.Infof("Unsubscribed from topic %s", topic)
    return nil
}
```

- [ ] **Step 2: 编写 MQTT 客户端测试**

Create: `pkg/mqtt/client_test.go`

```go
package mqtt_test

import (
    "context"
    "testing"
    "time"
    
    "github.com/xuanyiying/smart-park/pkg/mqtt"
)

func TestMQTTClientConnect(t *testing.T) {
    config := &mqtt.Config{
        Broker:   "tcp://localhost:1883",
        ClientID: "test-client",
    }
    
    client := mqtt.NewClient(config, nil)
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := client.Connect(ctx)
    if err != nil {
        t.Fatalf("failed to connect: %v", err)
    }
    
    defer client.Disconnect()
}

func TestMQTTPublishSubscribe(t *testing.T) {
    // 测试发布订阅
}
```

- [ ] **Step 3: 运行测试**

Run: `go test ./pkg/mqtt/... -v`

- [ ] **Step 4: 重构 vehicle mqtt client**

Modify: `internal/vehicle/data/mqtt/client.go`

```go
// 使用 pkg/mqtt.Client 接口
// 实现设备特定的 MQTT 逻辑
```

- [ ] **Step 5: 提交代码**

```bash
git add pkg/mqtt/ internal/vehicle/data/mqtt/
git commit -m "feat(mqtt): implement generic MQTT client with reconnection support"
```

### Task 3.2: 实现设备指令下发

**Files:**
- Create: `internal/vehicle/device/command.go`
- Create: `internal/vehicle/device/command_test.go`
- Modify: `internal/vehicle/biz/vehicle.go`

- [ ] **Step 1: 定义设备指令协议**

Create: `internal/vehicle/device/command.go`

```go
package device

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/google/uuid"
    "github.com/xuanyiying/smart-park/pkg/mqtt"
)

type CommandType string

const (
    CommandOpenGate  CommandType = "open_gate"
    CommandCloseGate CommandType = "close_gate"
    CommandDisplay   CommandType = "display"
    CommandVoice     CommandType = "voice"
)

type Command struct {
    ID        string                 `json:"id"`
    Type      CommandType            `json:"type"`
    DeviceID  string                 `json:"device_id"`
    Timestamp int64                  `json:"timestamp"`
    Params    map[string]interface{} `json:"params"`
}

type CommandResponse struct {
    ID        string `json:"id"`
    Success   bool   `json:"success"`
    Message   string `json:"message"`
    Timestamp int64  `json:"timestamp"`
}

type CommandManager struct {
    mqttClient mqtt.Client
    pending    map[string]chan *CommandResponse
    timeout    time.Duration
}

func NewCommandManager(mqttClient mqtt.Client) *CommandManager {
    return &CommandManager{
        mqttClient: mqttClient,
        pending:    make(map[string]chan *CommandResponse),
        timeout:    10 * time.Second,
    }
}

func (m *CommandManager) SendCommand(ctx context.Context, deviceID string, cmdType CommandType, params map[string]interface{}) (*CommandResponse, error) {
    cmd := &Command{
        ID:        uuid.New().String(),
        Type:      cmdType,
        DeviceID:  deviceID,
        Timestamp: time.Now().Unix(),
        Params:    params,
    }
    
    // 构造 topic: device/{device_id}/command
    topic := fmt.Sprintf("device/%s/command", deviceID)
    
    payload, err := json.Marshal(cmd)
    if err != nil {
        return nil, err
    }
    
    // 创建响应通道
    respChan := make(chan *CommandResponse, 1)
    m.pending[cmd.ID] = respChan
    defer delete(m.pending, cmd.ID)
    
    // 发送指令
    if err := m.mqttClient.Publish(ctx, topic, payload); err != nil {
        return nil, err
    }
    
    // 等待响应
    select {
    case resp := <-respChan:
        return resp, nil
    case <-time.After(m.timeout):
        return nil, fmt.Errorf("command timeout")
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}

func (m *CommandManager) HandleResponse(resp *CommandResponse) {
    if ch, ok := m.pending[resp.ID]; ok {
        ch <- resp
    }
}
```

- [ ] **Step 2: 集成到 VehicleUseCase**

Modify: `internal/vehicle/biz/vehicle.go`

```go
type VehicleUseCase struct {
    vehicleRepo    VehicleRepo
    billingRepo    BillingRepo
    mqttClient     mqtt.Client
    commandManager *device.CommandManager  // 新增
    log            *log.Helper
}

func (uc *VehicleUseCase) SendCommand(ctx context.Context, req *v1.SendCommandRequest) (*v1.CommandData, error) {
    // 解析指令类型
    cmdType := device.CommandType(req.Command)
    
    // 发送指令
    resp, err := uc.commandManager.SendCommand(ctx, req.DeviceId, cmdType, req.Params)
    if err != nil {
        uc.log.WithContext(ctx).Errorf("failed to send command: %v", err)
        return nil, err
    }
    
    return &v1.CommandData{
        CommandId: resp.ID,
        Status:    "success",
    }, nil
}

func (uc *VehicleUseCase) Entry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryData, error) {
    // ... 现有逻辑 ...
    
    // 发送开闸指令
    if allowed {
        _, err := uc.commandManager.SendCommand(ctx, deviceID, device.CommandOpenGate, nil)
        if err != nil {
            uc.log.Errorf("failed to open gate: %v", err)
        }
    }
    
    // 发送显示指令
    displayParams := map[string]interface{}{
        "message": displayMessage,
    }
    uc.commandManager.SendCommand(ctx, deviceID, device.CommandDisplay, displayParams)
    
    // ... 返回结果 ...
}
```

- [ ] **Step 3: 编写测试**

Create: `internal/vehicle/device/command_test.go`

```go
package device_test

import (
    "context"
    "testing"
    
    "github.com/xuanyiying/smart-park/internal/vehicle/device"
)

func TestSendCommand(t *testing.T) {
    // 测试指令发送
}

func TestCommandTimeout(t *testing.T) {
    // 测试指令超时
}
```

- [ ] **Step 4: 运行测试**

Run: `go test ./internal/vehicle/device/... -v`

- [ ] **Step 5: 提交代码**

```bash
git add internal/vehicle/device/ internal/vehicle/biz/vehicle.go
git commit -m "feat(vehicle): implement device command sending via MQTT"
```

---

## Phase 4: 配置安全（预计 1-2 天）

### Task 4.1: 环境变量管理

**Files:**
- Create: `.env.example`
- Modify: `configs/*.yaml`
- Create: `scripts/load_env.sh`

- [ ] **Step 1: 创建环境变量模板**

Create: `.env.example`

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_secure_password_here
DB_NAME=parking

# Redis
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=

# JWT
JWT_SECRET_KEY=your_jwt_secret_key_here

# WeChat Mini Program
WECHAT_MINI_APP_ID=your_app_id
WECHAT_MINI_APP_SECRET=your_app_secret

# WeChat Pay
WECHAT_APP_ID=your_wechat_pay_app_id
WECHAT_MCH_ID=your_mch_id
WECHAT_API_KEY=your_api_key
WECHAT_CERT_SERIAL_NO=your_cert_serial_no
WECHAT_PRIVATE_KEY_PATH=/path/to/private_key.pem
WECHAT_PUBLIC_KEY_PATH=/path/to/public_key.pem
WECHAT_NOTIFY_URL=https://your-domain.com/api/v1/pay/wechat/notify

# Alipay
ALIPAY_APP_ID=your_alipay_app_id
ALIPAY_PRIVATE_KEY=your_private_key
ALIPAY_PUBLIC_KEY=alipay_public_key
ALIPAY_NOTIFY_URL=https://your-domain.com/api/v1/pay/alipay/notify
ALIPAY_IS_PRODUCTION=false

# MQTT
MQTT_BROKER=tcp://localhost:1883
MQTT_CLIENT_ID=smart-park
MQTT_USERNAME=
MQTT_PASSWORD=

# Observability
OTEL_ENDPOINT=localhost:4317
JAEGER_AGENT_HOST=localhost

# Environment
ENV=development
```

- [ ] **Step 2: 创建环境变量加载脚本**

Create: `scripts/load_env.sh`

```bash
#!/bin/bash

if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
    echo "Environment variables loaded from .env"
else
    echo "Warning: .env file not found"
fi
```

- [ ] **Step 3: 更新配置文件使用环境变量**

Modify: `configs/vehicle.yaml`

```yaml
server:
  port: 8001
  timeout: 60

database:
  driver: postgres
  source: "host=${DB_HOST} user=${DB_USER} password=${DB_PASSWORD} dbname=${DB_NAME} port=${DB_PORT} sslmode=disable"

redis:
  addr: "${REDIS_ADDR}"
  password: "${REDIS_PASSWORD}"
  db: 0

mqtt:
  broker: "${MQTT_BROKER}"
  client_id: "${MQTT_CLIENT_ID}"
  username: "${MQTT_USERNAME}"
  password: "${MQTT_PASSWORD}"

log:
  level: info
  format: json

otel:
  endpoint: "${OTEL_ENDPOINT}"
  serviceName: "vehicle-svc"
```

- [ ] **Step 4: 更新其他配置文件**

类似地更新 `billing.yaml`, `payment.yaml`, `admin.yaml`, `gateway.yaml`, `user.yaml`

- [ ] **Step 5: 添加 .env 到 .gitignore**

Modify: `.gitignore`

```
# Environment variables
.env
.env.local
.env.*.local

# Secrets
*.pem
*.key
secrets/
```

- [ ] **Step 6: 提交代码**

```bash
git add .env.example scripts/load_env.sh configs/ .gitignore
git commit -m "feat(config): use environment variables for sensitive configuration"
```

### Task 4.2: 配置验证

**Files:**
- Create: `pkg/config/validator.go`
- Create: `pkg/config/validator_test.go`

- [ ] **Step 1: 实现配置验证器**

Create: `pkg/config/validator.go`

```go
package config

import (
    "errors"
    "fmt"
    "os"
    "strings"
)

type Validator struct {
    requiredVars []string
}

func NewValidator() *Validator {
    return &Validator{
        requiredVars: []string{
            "DB_HOST",
            "DB_PASSWORD",
            "JWT_SECRET_KEY",
            // 添加其他必需的环境变量
        },
    }
}

func (v *Validator) Validate() error {
    var missing []string
  
    for _, varName := range v.requiredVars {
        value := os.Getenv(varName)
        if value == "" || strings.Contains(value, "your_") {
            missing = append(missing, varName)
        }
    }
  
    if len(missing) > 0 {
        return fmt.Errorf("missing or invalid environment variables: %s", strings.Join(missing, ", "))
    }
  
    return nil
}

func (v *Validator) ValidateDatabaseConfig() error {
    password := os.Getenv("DB_PASSWORD")
    if password == "postgres" || password == "password" {
        return errors.New("database password is too weak")
    }
    return nil
}

func (v *Validator) ValidateJWTSecret() error {
    secret := os.Getenv("JWT_SECRET_KEY")
    if len(secret) < 32 {
        return errors.New("JWT secret key must be at least 32 characters")
    }
    return nil
}
```

- [ ] **Step 2: 编写测试**

Create: `pkg/config/validator_test.go`

```go
package config_test

import (
    "os"
    "testing"
    
    "github.com/xuanyiying/smart-park/pkg/config"
)

func TestValidateMissingVars(t *testing.T) {
    os.Clearenv()
  
    validator := config.NewValidator()
    err := validator.Validate()
  
    if err == nil {
        t.Error("expected error for missing environment variables")
    }
}

func TestValidateWeakPassword(t *testing.T) {
    os.Setenv("DB_PASSWORD", "password")
    defer os.Unsetenv("DB_PASSWORD")
  
    validator := config.NewValidator()
    err := validator.ValidateDatabaseConfig()
  
    if err == nil {
        t.Error("expected error for weak password")
    }
}
```

- [ ] **Step 3: 运行测试**

Run: `go test ./pkg/config/... -v`

- [ ] **Step 4: 在服务启动时添加验证**

Modify: `cmd/vehicle/main.go` (以及其他服务)

```go
func main() {
    // 验证配置
    validator := config.NewValidator()
    if err := validator.Validate(); err != nil {
        log.Fatalf("Configuration validation failed: %v", err)
    }
  
    // ... 其他初始化代码 ...
}
```

- [ ] **Step 5: 提交代码**

```bash
git add pkg/config/ cmd/*/main.go
git commit -m "feat(config): add configuration validation for security"
```

---

## 验收标准

### Phase 1 完成标准：
- [ ] 微信支付 SDK 集成完成，签名验证通过
- [ ] 支付宝 SDK 集成完成，签名验证通过
- [ ] 支付安全测试全部通过
- [ ] 能够成功创建支付订单并处理回调

### Phase 2 完成标准：
- [ ] User service Proto 定义完成并生成代码
- [ ] JWT 认证中间件实现并测试通过
- [ ] 用户登录、车牌绑定、停车记录查询功能可用
- [ ] Gateway 路由配置正确，user service 可访问

### Phase 3 完成标准：
- [ ] MQTT 客户端实现并支持重连
- [ ] 设备指令下发功能实现
- [ ] 车辆入场/出场时能正确控制道闸
- [ ] 指令超时处理正确

### Phase 4 完成标准：
- [ ] 所有敏感配置使用环境变量
- [ ] 配置验证机制生效
- [ ] .env.example 文档完整
- [ ] 弱密码检测生效

---

## 后续计划（P1/P2）

完成 P0 阶段后，可以继续推进：

**P1 - 重要功能**：
1. 缓存层实现（Redis 缓存车辆信息、计费规则）
2. 通知服务实现（短信、微信通知）
3. 测试覆盖率提升（单元测试、集成测试）
4. 监控告警（Prometheus + Grafana）

**P2 - 增强功能**：
1. 数据分析服务
2. 智能推荐系统
3. 多租户支持
4. 充电桩集成

---

**计划完成时间估算**：
- Phase 1: 3-4 天
- Phase 2: 5-7 天
- Phase 3: 3-4 天
- Phase 4: 1-2 天
- **总计**: 12-17 个工作日

**风险提示**：
1. 微信/支付宝 SDK 集成可能需要申请测试账号
2. MQTT 需要搭建测试环境（EMQX/Mosquitto）
3. 用户认证需要微信小程序 AppID 和 AppSecret
4. 测试覆盖率目标建议 70% 以上

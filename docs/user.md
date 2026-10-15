# User Service 用户服务

## 模块概述

用户服务负责 C 端用户相关功能，包括用户认证（JWT）、车牌绑定、扫码支付等。服务基于 Kratos 框架构建，通过 gRPC 与 HTTP 双协议对外提供服务，并与车辆服务、支付服务协作完成扫码缴费闭环。

- **服务端口**：HTTP `8007` / gRPC `9007`
- **网关路由**：`/api/v1/user/*` → `user-svc:8007`

> 注：用户服务端口原为 `8005`（与充电服务冲突），后调整至 `8007`（避开 multitenancy 的 8006）。

## 核心功能

| 功能 | 说明 |
|------|------|
| 用户注册/登录 | 微信小程序 code 换 openID 登录，首次登录自动创建用户，签发 JWT；无独立注册接口 |
| 用户信息管理 | 查询与更新用户资料 |
| 车牌绑定 | 一个用户可绑定多个车牌 |
| 扫码支付 | 扫码识别入场记录，调用支付服务完成缴费 |
| 支付记录 | 查询用户历史停车记录（简化实现：金额硬编码为 0，未真实查询订单，待完善） |

## 网关免认证路径

以下路径不经过 JWT 校验（见 `configs/gateway.yaml`）：

- `/api/v1/user/login`
- `/api/v1/user/register`（网关配置中保留了该免认证路径，但 user 服务当前无对应实现）

## 架构分层

```
cmd/user/main.go              # 服务入口
internal/user/
├── biz/                      # 业务逻辑层
│   └── user.go               # UserUseCase
├── data/                     # 数据访问层
└── service/                  # gRPC/HTTP 服务层
    └── user.go
```

## 依赖

- **车辆服务 (vehicle-svc)**：查询入场记录、出场放行
- **支付服务 (payment-svc)**：创建支付订单、查询支付状态
- **JWT**：签发与校验用户令牌
- **微信小程序**：扫码支付场景下的用户身份与支付能力

## 配置说明

`configs/user.yaml`：

```yaml
server:
  port: 8007
  timeout: 60
database:
  driver: postgres
  source: "host=localhost user=postgres password=postgres dbname=parking port=5432 sslmode=disable"
redis:
  addr: "localhost:6379"
  password: ""
  db: 0
# 下游依赖 gRPC 端点（必填，缺失则服务启动退出）
vehicle:
  endpoint: "localhost:9001"
payment:
  endpoint: "localhost:9003"
jwt:
  # RSA 公私钥路径；留空时 JWTManager 不加载密钥（开发态可用）
  public_key_path: ""
  private_key_path: ""
  token_duration: 24h
wechat:
  app_id: ""
  api_key: ""   # 微信小程序 AppSecret（可通过 SP_WECHAT_API_KEY 环境变量覆盖）
log:
  level: info
  format: json
otel:
  endpoint: "localhost:4317"
  serviceName: "user-svc"
```

## 运行

```bash
# 本地运行（需先启动 postgres / redis）
go run ./cmd/user -conf ./configs/user.yaml

# 构建
go build -o bin/user-svc ./cmd/user
```

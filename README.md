# Smart Park 智慧停车管理系统

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![Kratos](https://img.shields.io/badge/Kratos-2.9+-green.svg)](https://github.com/go-kratos/kratos)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## 项目概述

Smart Park 是一个基于 Go 语言和 Kratos 微服务框架构建的现代化智慧停车管理系统。系统采用微服务架构设计，支持车牌识别、智能计费、多渠道支付、设备控制等核心功能，适用于商业综合体、住宅小区、写字楼等多种停车场景。

### 核心特性

- **微服务架构**：基于 Kratos 框架，服务间通过 gRPC 通信，支持独立部署和扩展
- **双协议支持**：同时支持 gRPC 和 HTTP/REST API，便于不同客户端接入
- **智能计费引擎**：支持多种计费规则（按时、按时段、月卡、VIP等），支持规则叠加
- **多渠道支付**：集成微信支付、支付宝，支持扫码支付、JSAPI支付
- **设备管理**：支持车牌识别摄像头、道闸、地感等设备的接入和控制
- **分布式锁**：基于 Redis 的分布式锁，确保并发场景数据一致性
- **链路追踪**：集成 OpenTelemetry 和 Jaeger，实现全链路追踪
- **多租户支持**：支持多停车场管理，数据隔离

## 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         接入层                                   │
├───────────────┬───────────────┬───────────────┬──────────────────┤
│   车主小程序   │   管理后台     │   设备网关    │   第三方回调      │
└───────┬───────┴───────┬───────┴───────┬───────┴────────┬─────────┘
        │               │               │                │
        └───────────────┴───────┬───────┴────────────────┘
                                │
                    ┌───────────▼───────────┐
                    │    API Gateway        │
                    │    (Port: 8000)       │
                    └───────────┬───────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
┌───────▼───────┐    ┌───────▼───────┐    ┌───────▼───────┐
│  Vehicle      │    │   Billing     │    │   Payment     │
│  Service      │    │   Service     │    │   Service     │
│  (Port: 8001) │    │  (Port: 8002) │    │  (Port: 8003) │
└───────┬───────┘    └───────┬───────┘    └───────┬───────┘
        │                    │                    │
┌───────▼───────┐    ┌───────▼───────┐    ┌───────▼───────┐
│   Admin       │    │   Charging    │    │ MultiTenancy  │
│   Service     │    │   Service     │    │   Service     │
│  (Port: 8004) │    │  (Port: 8005) │    │  (Port: 8006) │
└───────────────┘    └───────────────┘    └───────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
┌───────▼───────┐    ┌───────▼───────┐    ┌───────▼───────┐
│  PostgreSQL   │    │    Redis      │    │    Etcd       │
│   (主数据)     │    │  (缓存/锁)    │    │ (服务发现)    │
└───────────────┘    └───────────────┘    └───────────────┘
```

## 技术栈

| 层级 | 技术选型 |
|------|----------|
| 后端框架 | [Kratos](https://github.com/go-kratos/kratos) v2.9 |
| 编程语言 | Go 1.26+ |
| API 协议 | gRPC + HTTP/REST (grpc-gateway) |
| 数据库 | PostgreSQL 15 |
| ORM | Ent |
| 缓存 | Redis 7 |
| 服务注册 | Etcd v3.5 |
| 链路追踪 | OpenTelemetry + Jaeger |
| 消息队列 | Redis Streams / NATS / RocketMQ |
| 支付 SDK | wechatpay-go, alipay |
| 容器化 | Docker, Docker Compose |

## 快速开始

### 环境要求

- Go 1.26+
- Docker & Docker Compose
- Make (可选)

### 启动基础设施

```bash
# 启动 PostgreSQL, Redis, Etcd, Jaeger
docker-compose -f deploy/docker-compose.yml up -d postgres redis etcd jaeger
```

### 本地运行服务

```bash
# 运行网关服务
go run ./cmd/gateway -conf ./configs

# 运行车辆服务
go run ./cmd/vehicle -conf ./configs

# 运行计费服务
go run ./cmd/billing -conf ./configs

# 运行支付服务
go run ./cmd/payment -conf ./configs

# 运行管理服务
go run ./cmd/admin -conf ./configs
```

### 构建 Docker 镜像

```bash
# 构建所有服务
docker-compose -f deploy/docker-compose.yml build

# 启动完整服务栈
docker-compose -f deploy/docker-compose.yml up -d
```

## 服务说明

| 服务 | 端口 | 职责 | 文档 |
|------|------|------|------|
| Gateway | 8000 | API 网关，路由转发 | [docs/gateway.md](docs/gateway.md) |
| Vehicle | 8001 | 车辆入场/出场，设备管理 | [docs/vehicle.md](docs/vehicle.md) |
| Billing | 8002 | 费用计算，计费规则 | [docs/billing.md](docs/billing.md) |
| Payment | 8003 | 支付处理，订单管理 | [docs/payment.md](docs/payment.md) |
| Admin | 8004 | 停车场管理，报表统计 | [docs/admin.md](docs/admin.md) |
| Charging | 8005 | 充电桩管理 | [docs/charging.md](docs/charging.md) |
| MultiTenancy | 8006 | 多租户管理 | [docs/multitenancy.md](docs/multitenancy.md) |

## API 接口

### 设备端接口

- `POST /api/v1/device/entry` - 车辆入场
- `POST /api/v1/device/exit` - 车辆出场
- `POST /api/v1/device/heartbeat` - 设备心跳
- `GET /api/v1/device/{id}/status` - 设备状态查询
- `POST /api/v1/device/{id}/command` - 设备控制指令

### 计费接口

- `POST /api/v1/billing/calculate` - 费用计算
- `GET /api/v1/admin/billing/rules` - 计费规则列表
- `POST /api/v1/admin/billing/rules` - 创建计费规则
- `PUT /api/v1/admin/billing/rules/{id}` - 更新计费规则
- `DELETE /api/v1/admin/billing/rules/{id}` - 删除计费规则

### 支付接口

- `POST /api/v1/pay/create` - 创建支付订单
- `GET /api/v1/pay/{id}/status` - 查询支付状态
- `POST /api/v1/pay/callback/wechat` - 微信支付回调
- `POST /api/v1/pay/callback/alipay` - 支付宝支付回调
- `POST /api/v1/pay/{id}/refund` - 退款申请

### 管理接口

- `GET /api/v1/admin/lots` - 停车场列表
- `POST /api/v1/admin/lots` - 创建停车场
- `GET /api/v1/admin/records` - 入出场记录
- `GET /api/v1/admin/orders` - 订单列表
- `GET /api/v1/admin/reports/daily` - 日报表
- `GET /api/v1/admin/reports/monthly` - 月报表

## 项目结构

```
smart-park/
├── api/                    # Protocol Buffers 定义
│   ├── admin/v1/
│   ├── billing/v1/
│   ├── payment/v1/
│   ├── vehicle/v1/
│   └── ...
├── cmd/                    # 服务入口
│   ├── gateway/
│   ├── vehicle/
│   ├── billing/
│   ├── payment/
│   └── admin/
├── configs/                # 配置文件
├── deploy/                 # 部署配置
│   ├── docker/
│   └── docker-compose.yml
├── docs/                   # 文档
├── internal/               # 内部实现
│   ├── admin/
│   ├── billing/
│   ├── payment/
│   ├── vehicle/
│   └── pkg/               # 公共包
├── pkg/                    # 可复用公共库
│   ├── auth/              # JWT 认证
│   ├── cache/             # 缓存封装
│   ├── lock/              # 分布式锁
│   ├── middleware/        # 中间件
│   └── ...
├── web/                    # 前端管理后台 (Next.js)
└── scripts/                # 脚本工具
```

## 开发指南

### 生成 Protocol Buffers

```bash
./scripts/generate_proto.sh
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行指定服务测试
go test ./internal/vehicle/...

# 运行集成测试
go test ./tests/... -tags=integration
```

### 代码规范

项目使用 golangci-lint 进行代码检查：

```bash
golangci-lint run
```

## 生产部署

详细部署文档请参考 [docs/deployment.md](docs/deployment.md)

### 部署架构

- **小型部署**（1-20 个停车场）：单机 Docker Compose 部署
- **中型部署**（20-100 个停车场）：主备双活架构
- **大型部署**（100+ 个停车场）：Kubernetes 多活集群

## 贡献指南

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 联系方式

- 项目主页：https://github.com/xuanyiying/smart-park
- 问题反馈：https://github.com/xuanyiying/smart-park/issues

---

**Smart Park** - 让停车更智能，让出行更便捷

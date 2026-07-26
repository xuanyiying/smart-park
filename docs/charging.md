# Charging Service 充电服务

## 模块概述

充电服务是 Smart Park 系统的扩展服务之一，负责电动汽车充电桩的全生命周期管理，包括充电站（桩）管理、连接器管理、充电会话管理、分时电价配置以及充电费用结算。服务基于 Kratos 框架构建，使用 Ent ORM 进行数据持久化，通过 gRPC 与 HTTP 双协议对外提供服务。

- **服务端口**：HTTP `8005` / gRPC `9005`
- **网关路由**：`/api/v1/charging/*` → `charging-svc:8005`
- **数据存储**：PostgreSQL（`stations`、`connectors`、`sessions`、`prices` 四张表）

## 核心功能

### 1. 充电站管理 (Station)

充电站是物理充电桩的逻辑抽象，归属于某个停车场（lot）。

| 接口 | 方法 | 路径 |
|------|------|------|
| 创建充电站 | POST | `/api/v1/charging/stations` |
| 查询充电站 | GET | `/api/v1/charging/stations/{stationId}` |
| 更新充电站 | PUT | `/api/v1/charging/stations/{stationId}` |
| 删除充电站 | DELETE | `/api/v1/charging/stations/{stationId}` |
| 充电站列表 | GET | `/api/v1/charging/stations` |
| 可用充电站 | GET | `/api/v1/charging/stations/available` |

**充电站状态**：`available`（可用）、`offline`（离线）、`maintenance`（维护中）

**充电站类型**：`ac`（交流）、`dc`（直流）、`fast_dc`（直流快充）

> `GetAvailableStations` 仅返回状态为 `available` 且 `AvailableConnectors > 0` 的充电站，供用户端筛选可直接使用的桩。

### 2. 连接器管理 (Connector)

连接器是充电站上的物理充电枪接口，一个充电站可包含多个连接器。

| 接口 | 方法 | 路径 |
|------|------|------|
| 创建连接器 | POST | `/api/v1/charging/connectors` |
| 查询连接器 | GET | `/api/v1/charging/connectors/{connectorId}` |
| 连接器列表 | GET | `/api/v1/charging/stations/{stationId}/connectors` |

**连接器状态**：`available`（可用）、`charging`（充电中）、`faulted`（故障）、`offline`（离线）

**容量约束**：创建连接器时会校验当前数量不超过充电站的 `TotalConnectors` 上限。

### 3. 充电会话管理 (Session)

充电会话记录一次完整的充电过程，是计费与支付的核心载体。

#### 充电流程

```
用户扫码 → 选择连接器 → StartCharging → 锁定连接器 → 充电中
    → StopCharging → 计算电量与费用 → 解锁连接器 → 待支付
    → ConfirmPayment → 完成
```

| 接口 | 方法 | 路径 |
|------|------|------|
| 开始充电 | POST | `/api/v1/charging/sessions/start` |
| 停止充电 | POST | `/api/v1/charging/sessions/{sessionId}/stop` |
| 查询会话 | GET | `/api/v1/charging/sessions/{sessionId}` |
| 用户会话列表 | GET | `/api/v1/charging/sessions` |
| 充电汇总 | GET | `/api/v1/charging/sessions/{sessionId}/summary` |

**会话状态**：`pending`、`charging`、`completed`、`cancelled`、`expired`

**支付状态**：`pending`、`paid`、`refunded`、`failed`

#### 关键技术实现

**防并发占用**：`StartCharging` 通过 `LockConnector` 将连接器状态置为 `charging`，并查询 `GetActiveSession` 防止同一连接器被重复开启会话。

**费用计算**：`StopCharging` 时按当前生效电价计算：

```go
baseCost := energyKWh * price.PricePerKWh
if price.IsPeakHours {
    baseCost += energyKWh * price.PeakLoad      // 峰时服务费
} else {
    baseCost += energyKWh * price.OffPeakLoad    // 谷时服务费
}
totalAmount = cost + serviceFee
```

**电量估算**：若设备未上报实际电量，则根据充电时长与连接器功率估算：

```go
energy = duration(hours) × power(kW)
```

**超时过期**：会话时长超过 `MaxSessionDuration`（默认 8 小时）会被标记为 `expired`，`ExpireOldSessions` 可批量清理。

### 4. 分时电价管理 (Price)

支持按充电站配置分时电价，区分峰时/谷时负荷。

| 接口 | 方法 | 路径 |
|------|------|------|
| 创建电价 | POST | `/api/v1/charging/prices` |
| 当前电价 | GET | `/api/v1/charging/stations/{stationId}/price` |
| 电价列表 | GET | `/api/v1/charging/stations/{stationId}/prices` |

**峰时判定**：默认峰段为 7-9 点、17-21 点，创建电价时自动计算 `IsPeakHours`。

**生效规则**：`GetCurrentPrice` 查询 `effectiveAt <= now` 且（`expiresAt` 为空或 `expiresAt > now`）的电价。

### 5. 支付确认 (Payment)

充电服务不直接对接第三方支付，而是由支付服务完成扣款后回调确认：

| 接口 | 方法 | 路径 |
|------|------|------|
| 确认支付 | POST | `/api/v1/charging/sessions/{sessionId}/payment/confirm` |

`ConfirmPayment` 校验支付金额 ≥ 应付金额后，将支付状态更新为 `paid`，并记录交易号与支付方式。`RefundPayment` 可对已支付会话发起退款（状态置为 `refunded`）。

## 架构分层

```
cmd/charging/main.go              # 服务入口，手动装配依赖
internal/charging/
├── biz/                          # 业务逻辑层
│   ├── charging.go               # ChargingUseCase + ChargingRepo 接口
│   ├── entity.go                 # 领域实体与枚举
│   ├── errors.go                 # 业务错误
│   └── charging_test.go          # 单元测试
├── data/                         # 数据访问层
│   ├── data.go                   # Data 结构与事务管理器
│   ├── charging.go               # ChargingRepo 实现（Ent）
│   └── ent/                      # Ent 生成代码与 schema
└── service/                      # gRPC/HTTP 服务层
    └── charging.go               # proto ↔ biz 转换
```

## 配置说明

`configs/charging.yaml`：

```yaml
server:
  port: 8005
  timeout: 60
database:
  driver: postgres
  source: "host=localhost user=postgres password=postgres dbname=parking port=5432 sslmode=disable"
redis:
  addr: "localhost:6379"
```

**默认业务参数**（`biz.DefaultConfig`）：

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `DefaultServiceFee` | 0.8 | 默认服务费（元） |
| `MaxSessionDuration` | 8h | 单次充电最长时长 |
| `DefaultPeakLoadKWh` | 1.5 | 默认峰时功率（kW） |
| `DefaultOffPeakLoadKWh` | 0.8 | 默认谷时功率（kW） |

## 部署

### Docker Compose

```bash
docker-compose -f deploy/docker-compose.yml up -d charging
```

### Kubernetes

```bash
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
kubectl apply -f deploy/k8s/configmap.yaml
```

## 运行

```bash
# 本地运行（需先启动 postgres / redis）
go run ./cmd/charging -conf ./configs/charging.yaml

# 构建
go build -o bin/charging-svc ./cmd/charging
```

服务启动时会通过 `dbClient.Schema.Create` 自动执行数据库迁移，无需手动建表。

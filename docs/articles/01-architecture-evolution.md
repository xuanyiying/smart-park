# Smart Park 架构设计：从单体到微服务的演进之路

## 引言

智慧停车系统作为城市交通基础设施的重要组成部分，面临着业务复杂度高、实时性要求强、系统可靠性要求严苛等多重挑战。一个现代化的停车管理系统不仅要处理车辆进出场的实时识别和计费，还需要对接多种支付渠道、管理复杂的计费规则、支持设备控制、提供数据分析等多元化功能。在项目初期，我们面临着一个关键的技术决策：是选择快速开发的单体架构，还是选择更具扩展性的微服务架构？

本文将以 Smart Park 智慧停车系统为例，详细阐述从 v0.1 单体架构到 v0.3 微服务架构的演进历程。我们将深入探讨架构演进背后的技术决策逻辑，分析服务拆分的边界和原则，分享 Kratos 框架在实际项目中的应用经验，以及在架构演进过程中遇到的挑战和解决方案。本文的目标读者是正在面临类似架构决策的架构师和技术负责人，希望通过我们的实践经验，为您的技术选型和架构设计提供参考。

## 一、为什么选择微服务架构

### 1.1 业务复杂度分析

智慧停车系统的业务复杂度远超一般人的想象。从业务域来看，系统至少包含以下几个核心领域：

**车辆管理域**：包括车辆信息管理、车牌识别、车辆分类（临时车、月卡车、VIP车）、车辆入场出场记录等。这个领域需要处理实时数据，对响应时间要求极高（通常要求在 1 秒内完成识别和决策）。

**计费管理域**：涉及复杂的计费规则引擎，需要支持按时长计费、分时段计费、封顶计费、会员折扣、优惠券叠加等多种计费策略。不同停车场可能有完全不同的计费规则，甚至同一停车场在不同时段也有不同的费率。

**支付管理域**：需要对接微信支付、支付宝、银联等多种支付渠道，处理支付回调、退款、对账等复杂流程。支付安全性和幂等性是核心要求。

**设备管理域**：包括车牌识别摄像头、道闸、地感、显示屏等硬件设备的控制和管理。设备通信协议多样（HTTP、MQTT、TCP等），需要处理离线场景和设备故障。

**运营管理域**：提供停车场管理、车辆管理、订单管理、数据报表、退款审批等后台功能，涉及复杂的权限控制和业务流程。

如果将这些功能都放在一个单体应用中，代码耦合度会非常高，一个模块的修改可能影响其他模块的稳定性。更重要的是，不同业务域的技术特性差异很大：车辆识别需要高性能计算，计费引擎需要灵活的规则配置，支付服务需要高安全性，设备控制需要实时通信。单体架构难以针对不同特性进行优化。

### 1.2 团队协作需求

Smart Park 项目采用敏捷开发模式，团队规模从最初的 2 人发展到 6 人。在单体架构阶段，我们遇到了以下协作问题：

**代码冲突频繁**：多人同时修改同一个代码仓库，经常出现代码冲突，尤其是在业务高峰期需要紧急修复 bug 时。

**部署风险高**：单体应用任何模块的修改都需要重新部署整个应用，一次部署可能影响所有功能，回滚成本高。

**技术栈受限**：单体架构通常要求统一技术栈，难以根据业务特性选择最合适的技术方案。例如，计费引擎可能更适合用规则引擎实现，而设备控制可能更适合用 Go 语言实现。

**责任边界不清**：在单体架构中，模块之间的边界模糊，容易出现"公共代码"无人维护的情况，代码质量难以保证。

微服务架构天然适合团队协作。每个服务由独立的团队或个人负责，服务之间通过明确定义的 API 进行通信，技术栈可以根据业务特性灵活选择。这种模式大大提高了开发效率和代码质量。

### 1.3 系统可扩展性要求

智慧停车系统的扩展性需求主要体现在以下几个方面：

**水平扩展**：随着停车场数量增加，系统需要支持水平扩展。不同停车场的业务量差异很大，商业综合体的日车流量可能达到数千辆，而小区停车场可能只有几百辆。单体架构难以针对不同停车场进行差异化扩展。

**功能扩展**：系统需要支持新功能的快速迭代，例如车位预约、充电桩对接、会员体系、数据分析等。这些功能与核心业务相对独立，适合作为独立的服务进行开发和部署。

**性能扩展**：不同业务模块的性能要求不同。车牌识别服务需要高并发处理能力，计费服务需要快速响应，支付服务需要高可靠性。微服务架构可以针对不同服务的性能要求进行独立优化和扩展。

**地理扩展**：随着业务向不同城市扩展，系统需要支持多地部署和就近服务。微服务架构可以更容易实现服务的地理分布和负载均衡。

基于以上分析，我们决定在 v0.2 版本进行微服务架构改造。这个决策虽然增加了初期的开发成本，但为系统的长期发展奠定了坚实基础。

## 二、服务拆分的边界和原则

### 2.1 DDD 领域驱动设计思想

服务拆分是微服务架构设计的核心挑战。拆分过细会导致服务数量爆炸，增加运维复杂度；拆分过粗则无法发挥微服务的优势。我们采用领域驱动设计（Domain-Driven Design，DDD）的思想来指导服务拆分。

DDD 的核心概念是"限界上下文"（Bounded Context），即一个明确的边界，边界内的模型具有一致性，边界外的模型可能具有不同的含义。在 Smart Park 项目中，我们识别出以下限界上下文：

```
┌─────────────────────────────────────────────────────────┐
│                  Smart Park 领域模型                      │
├─────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ 车辆管理上下文 │  │ 计费管理上下文 │  │ 支付管理上下文 │ │
│  │              │  │              │  │              │ │
│  │ - 车辆实体    │  │ - 计费规则    │  │ - 订单实体    │ │
│  │ - 停车记录    │  │ - 计费引擎    │  │ - 支付渠道    │ │
│  │ - 设备管理    │  │ - 优惠策略    │  │ - 退款流程    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐ │
│  │ 运营管理上下文 │  │ 用户管理上下文 │  │ 通知管理上下文 │ │
│  │              │  │              │  │              │ │
│  │ - 停车场管理  │  │ - 用户认证    │  │ - 短信通知    │ │
│  │ - 数据报表    │  │ - 车牌绑定    │  │ - 推送通知    │ │
│  │ - 权限控制    │  │ - 月卡管理    │  │ - 消息队列    │ │
│  └──────────────┘  └──────────────┘  └──────────────┘ │
│                                                          │
└─────────────────────────────────────────────────────────┘
```

每个限界上下文对应一个微服务，服务内部采用分层架构：

- **接口层（API Layer）**：定义 gRPC 和 HTTP 接口，处理请求响应
- **应用层（Application Layer）**：编排业务流程，协调领域对象
- **领域层（Domain Layer）**：核心业务逻辑，包含实体、值对象、领域服务
- **基础设施层（Infrastructure Layer）**：数据库访问、外部服务调用等

### 2.2 业务边界划分

基于 DDD 的限界上下文，我们将 Smart Park 拆分为以下微服务：

**Vehicle Service（车辆服务）**：负责车辆进出场的核心业务，包括车牌识别、入场记录、出场记录、设备管理等。这是系统的核心服务，对实时性要求最高。

**Billing Service（计费服务）**：负责计费规则管理和费用计算。计费引擎是该服务的核心，需要支持灵活的规则配置和复杂的计费逻辑。

**Payment Service（支付服务）**：负责支付订单管理、支付渠道对接、退款处理等。该服务需要保证支付的安全性和幂等性。

**Admin Service（管理服务）**：负责停车场管理、车辆管理、订单查询、数据报表等后台功能。该服务面向运营人员，对数据一致性和查询性能要求较高。

**User Service（用户服务）**：负责用户认证、车牌绑定、月卡管理等车主端功能。该服务需要支持微信登录和 JWT 认证。

**Gateway Service（网关服务）**：作为系统的统一入口，负责请求路由、认证鉴权、限流熔断等。该服务是系统的门面，对性能和稳定性要求极高。

**Notification Service（通知服务）**：负责短信通知、推送通知等消息发送功能。该服务采用异步处理模式，通过消息队列解耦。

### 2.3 数据边界划分

微服务架构的一个重要原则是"每个服务拥有独立的数据库"。这样可以避免服务之间的数据耦合，实现真正的服务自治。在 Smart Park 项目中，我们采用以下数据边界划分策略：

**按服务拆分数据库**：每个微服务拥有独立的数据库 Schema，服务之间通过 API 通信，不直接访问对方的数据库。

**数据冗余策略**：对于需要跨服务共享的数据（如车辆信息），采用数据冗余策略。例如，Vehicle Service 存储完整的车辆信息，而 Payment Service 只存储订单相关的车辆基本信息（车牌号、车辆类型）。

**数据同步机制**：对于需要实时同步的数据，采用事件驱动架构。例如，当车辆出场时，Vehicle Service 发布"车辆出场事件"，Billing Service 和 Payment Service 订阅该事件并执行相应逻辑。

**分布式事务处理**：对于跨服务的事务操作，采用 Saga 模式或 TCC 模式。例如，支付流程涉及 Payment Service 和 Vehicle Service，我们采用 Saga 模式保证最终一致性。

以下是数据库 Schema 的划分：

```sql
-- Vehicle Service 数据库
CREATE TABLE parking_records (
  id UUID PRIMARY KEY,
  lot_id UUID,
  plate_number VARCHAR(20),
  entry_time TIMESTAMP,
  exit_time TIMESTAMP,
  record_status VARCHAR(20),
  ...
);

CREATE TABLE devices (
  id UUID PRIMARY KEY,
  device_id VARCHAR(64) UNIQUE,
  device_secret VARCHAR(128),
  device_type VARCHAR(20),
  ...
);

-- Billing Service 数据库
CREATE TABLE billing_rules (
  id UUID PRIMARY KEY,
  lot_id UUID,
  rule_name VARCHAR(100),
  rule_config JSONB,
  priority INTEGER,
  ...
);

-- Payment Service 数据库
CREATE TABLE orders (
  id UUID PRIMARY KEY,
  record_id UUID,
  plate_number VARCHAR(20),
  amount DECIMAL(10,2),
  status VARCHAR(20),
  ...
);

CREATE TABLE refund_approvals (
  id UUID PRIMARY KEY,
  order_id UUID,
  amount DECIMAL(10,2),
  status VARCHAR(20),
  ...
);
```

## 三、Kratos 框架在项目中的应用

### 3.1 Kratos 框架特点

Kratos 是哔哩哔哩开源的 Go 微服务框架，我们在 Smart Park 项目中选择 Kratos 主要基于以下考虑：

**完整的微服务支持**：Kratos 提供了服务发现、负载均衡、熔断降级、限流、链路追踪等微服务必需的能力，开箱即用。

**gRPC 优先**：Kratos 采用 gRPC 作为服务间通信协议，同时支持 gRPC-Gateway 自动生成 HTTP 接口，满足内部高性能通信和外部 HTTP 接入的双重需求。

**清晰的分层架构**：Kratos 推荐的分层架构（API、Service、Biz、Data）与 DDD 的分层架构高度契合，有助于保持代码的清晰性和可维护性。

**强大的工具链**：Kratos 提供了 kratos-cli 工具，可以快速生成项目模板、proto 文件、CRUD 代码等，大大提高开发效率。

**丰富的生态**：Kratos 与 Ent（ORM）、Protobuf、Wire（依赖注入）等主流工具深度集成，生态完善。

### 3.2 项目结构设计

Smart Park 项目采用 Kratos 推荐的项目结构，每个微服务都是一个独立的 Go Module：

```
smart-park/
├── api/                    # Proto 定义
│   ├── vehicle/v1/
│   ├── billing/v1/
│   ├── payment/v1/
│   └── admin/v1/
├── cmd/                    # 服务入口
│   ├── vehicle/main.go
│   ├── billing/main.go
│   ├── payment/main.go
│   └── admin/main.go
├── internal/               # 服务内部实现
│   ├── vehicle/
│   │   ├── service/       # 服务层
│   │   ├── biz/           # 业务逻辑层
│   │   └── data/          # 数据访问层
│   ├── billing/
│   ├── payment/
│   └── admin/
├── configs/               # 配置文件
└── pkg/                   # 公共库
    ├── auth/             # JWT 认证
    ├── cache/            # 缓存
    ├── lock/             # 分布式锁
    └── mq/               # 消息队列
```

每个服务的内部结构遵循 DDD 分层：

```go
// internal/vehicle/service/vehicle.go
type VehicleService struct {
    v1.UnimplementedVehicleServiceServer
    
    uc *biz.VehicleUseCase
}

func (s *VehicleService) HandleEntry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryResult, error) {
    return s.uc.HandleEntry(ctx, req)
}

// internal/vehicle/biz/vehicle.go
type VehicleUseCase struct {
    repo       VehicleRepository
    deviceRepo DeviceRepository
    log        *log.Helper
}

func (uc *VehicleUseCase) HandleEntry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryResult, error) {
    existing, err := uc.repo.FindActiveByPlate(ctx, req.LotId, req.PlateNumber)
    if err != nil {
        return nil, err
    }
    if existing != nil {
        return &v1.EntryResult{
            RecordId:       existing.ID.String(),
            PlateNumber:    req.PlateNumber,
            Allowed:        true,
            GateOpen:       true,
            DisplayMessage: "欢迎光临",
            IsDuplicate:    true,
        }, nil
    }
    
    record, err := uc.repo.Create(ctx, &ParkingRecord{
        LotID:       req.LotId,
        PlateNumber: &req.PlateNumber,
        EntryTime:   time.Now(),
        RecordStatus: "entry",
    })
    if err != nil {
        return nil, err
    }
    
    return &v1.EntryResult{
        RecordId:       record.ID.String(),
        PlateNumber:    req.PlateNumber,
        Allowed:        true,
        GateOpen:       true,
        DisplayMessage: "欢迎光临",
    }, nil
}

// internal/vehicle/data/vehicle.go
type vehicleRepo struct {
    data *Data
    log  *log.Helper
}

func (r *vehicleRepo) FindActiveByPlate(ctx context.Context, lotID, plateNumber string) (*ParkingRecord, error) {
    record, err := r.data.db.ParkingRecord.Query().
        Where(
            parkingrecord.LotID(lotID),
            parkingrecord.PlateNumber(plateNumber),
            parkingrecord.RecordStatusIn("entry", "exiting"),
        ).
        First(ctx)
    if err != nil {
        if ent.IsNotFound(err) {
            return nil, nil
        }
        return nil, err
    }
    return record, nil
}
```

### 3.3 服务间通信

Kratos 支持多种服务间通信方式，Smart Park 项目主要使用以下方式：

**gRPC 同步调用**：对于需要实时响应的场景，使用 gRPC 进行服务间调用。例如，Vehicle Service 在车辆出场时调用 Billing Service 计算费用。

```go
// internal/vehicle/client/billing/client.go
type BillingClient struct {
    conn *grpc.ClientConn
    client v1.BillingServiceClient
}

func (c *BillingClient) CalculateFee(ctx context.Context, recordID string) (*v1.CalculateFeeReply, error) {
    return c.client.CalculateFee(ctx, &v1.CalculateFeeRequest{
        RecordId: recordID,
    })
}
```

**消息队列异步通信**：对于不需要实时响应的场景，使用消息队列进行异步通信。Smart Park 项目使用 Redis Streams 作为消息队列（v0.3 版本计划迁移到 Kafka）。

```go
// 发布事件
func (uc *VehicleUseCase) HandleExit(ctx context.Context, req *v1.ExitRequest) (*v1.ExitResult, error) {
    // ... 处理出场逻辑
    
    // 发布车辆出场事件
    event := &VehicleExitEvent{
        RecordID:    record.ID.String(),
        PlateNumber: req.PlateNumber,
        ExitTime:    time.Now(),
        Amount:      bill.FinalAmount,
    }
    uc.eventPublisher.Publish("vehicle.exit", event)
    
    return &v1.ExitResult{...}, nil
}

// 订阅事件
func (s *PaymentService) SubscribeVehicleExit() {
    s.subscriber.Subscribe("vehicle.exit", func(event *VehicleExitEvent) error {
        // 创建待支付订单
        _, err := s.uc.CreateOrder(context.Background(), &CreateOrderRequest{
            RecordID:    event.RecordID,
            PlateNumber: event.PlateNumber,
            Amount:      event.Amount,
        })
        return err
    })
}
```

**服务发现与负载均衡**：Kratos 内置了服务发现机制，支持 Consul、Etcd、Nacos 等注册中心。Smart Park 项目使用 Etcd 作为服务注册中心：

```yaml
# configs/vehicle.yaml
server:
  grpc:
    addr: 0.0.0.0:8001
    timeout: 1s

client:
  billing:
    endpoint: discovery:///billing-svc
  payment:
    endpoint: discovery:///payment-svc

registry:
  etcd:
    endpoints:
      - localhost:2379
```

## 四、从 v0.1 到 v0.3 的架构演进

### 4.1 v0.1 单体架构设计

Smart Park 项目始于 2026 年 1 月，v0.1 版本采用单体架构，主要功能包括：

- 基础的停车场管理
- 简单的计费计算
- 手动支付记录

单体架构的优势在于开发速度快、部署简单、调试方便。在项目初期，团队只有 2 人，业务需求不明确，单体架构是合理的选择。

v0.1 的架构如下：

```
┌─────────────────────────────────────┐
│         Smart Park 单体应用          │
│  ┌───────────────────────────────┐  │
│  │  HTTP API (Gin Framework)     │  │
│  └───────────────┬───────────────┘  │
│                  │                   │
│  ┌───────────────▼───────────────┐  │
│  │      Business Logic Layer     │  │
│  │  - Vehicle Management         │  │
│  │  - Billing Calculation        │  │
│  │  - Payment Recording          │  │
│  └───────────────┬───────────────┘  │
│                  │                   │
│  ┌───────────────▼───────────────┐  │
│  │      Data Access Layer        │  │
│  │  - GORM                       │  │
│  │  - PostgreSQL                 │  │
│  └───────────────────────────────┘  │
└─────────────────────────────────────┘
```

然而，随着业务发展，单体架构的问题逐渐暴露：

**代码耦合严重**：车辆管理、计费、支付等逻辑混在一起，修改一个功能可能影响其他功能。

**扩展性差**：无法针对不同模块进行独立扩展，整个应用必须一起扩展。

**部署风险高**：每次部署都需要重新部署整个应用，一次部署失败可能影响所有功能。

### 4.2 v0.2 微服务拆分

2026 年 2 月，我们启动了 v0.2 版本的微服务改造。这次改造的核心目标是将单体应用拆分为独立的微服务，同时保持业务功能的完整性。

v0.2 的架构如下：

```
┌─────────────────────────────────────────────────────┐
│                  API Gateway (8000)                  │
│  - 请求路由                                          │
│  - 认证鉴权                                          │
│  - 限流熔断                                          │
└────────────────────┬────────────────────────────────┘
                     │
      ┌──────────────┼──────────────┬──────────────┐
      │              │              │              │
      ▼              ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐
│ Vehicle  │  │ Billing  │  │ Payment  │  │  Admin   │
│ Service  │  │ Service  │  │ Service  │  │ Service  │
│  (8001)  │  │  (8002)  │  │  (8003)  │  │  (8004)  │
└────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘
     │              │              │              │
     └──────────────┴──────────────┴──────────────┘
                           │
                  ┌────────▼────────┐
                  │   PostgreSQL    │
                  │   Redis Cache   │
                  │   Etcd Registry │
                  └─────────────────┘
```

微服务拆分的关键步骤：

**第一步：定义服务边界**。基于 DDD 的限界上下文，明确每个服务的职责和边界。

**第二步：拆分数据库**。将单体数据库按服务拆分，每个服务拥有独立的数据库 Schema。

**第三步：迁移业务逻辑**。将单体应用中的业务逻辑迁移到对应的微服务中，保持业务功能不变。

**第四步：实现服务间通信**。使用 gRPC 实现服务间的同步调用，使用消息队列实现异步通信。

**第五步：实现 API Gateway**。Gateway 作为统一入口，负责请求路由、认证鉴权等。

v0.2 版本的主要挑战：

**数据一致性**：跨服务的事务处理是最大的挑战。我们采用 Saga 模式保证最终一致性，例如支付流程：

```go
// Saga: 支付流程
func (s *PaymentSaga) Execute(recordID string, amount float64) error {
    // Step 1: 创建订单
    order, err := s.paymentService.CreateOrder(recordID, amount)
    if err != nil {
        return err
    }
    
    // Step 2: 调用支付渠道
    payResult, err := s.paymentService.CallPaymentChannel(order.ID)
    if err != nil {
        // Compensating action: 取消订单
        s.paymentService.CancelOrder(order.ID)
        return err
    }
    
    // Step 3: 更新车辆记录状态
    err = s.vehicleService.UpdateRecordStatus(recordID, "paid")
    if err != nil {
        // Compensating action: 退款
        s.paymentService.Refund(order.ID)
        return err
    }
    
    return nil
}
```

**服务发现**：微服务之间需要相互发现，我们使用 Etcd 作为服务注册中心，Kratos 内置的服务发现机制简化了配置。

**监控和调试**：单体应用的监控和调试相对简单，微服务架构需要分布式追踪。我们集成了 OpenTelemetry 和 Jaeger，实现了全链路追踪。

### 4.3 v0.3 完善和优化

2026 年 3 月，我们发布了 v0.3 版本，主要完善了以下方面：

**用户服务**：新增 User Service，支持微信登录、JWT 认证、车牌绑定、月卡管理等功能。

```go
// internal/user/service/user.go
func (s *UserService) WechatLogin(ctx context.Context, req *v1.WechatLoginRequest) (*v1.WechatLoginReply, error) {
    // 1. 调用微信 API 获取 openId
    openId, err := s.wechatClient.GetOpenId(req.Code)
    if err != nil {
        return nil, err
    }
    
    // 2. 查询或创建用户
    user, err := s.uc.FindOrCreateByOpenId(ctx, openId)
    if err != nil {
        return nil, err
    }
    
    // 3. 生成 JWT token
    token, err := s.jwtService.GenerateToken(user.ID)
    if err != nil {
        return nil, err
    }
    
    return &v1.WechatLoginReply{
        Token:    token,
        UserId:   user.ID.String(),
        Nickname: user.Nickname,
    }, nil
}
```

**分布式锁**：实现了基于 Redis 的分布式锁，解决并发问题。例如，车辆出场计费时需要加锁，防止重复计费：

```go
// pkg/lock/redis_lock.go
type RedisLock struct {
    client *redis.Client
    key    string
    value  string
    ttl    time.Duration
}

func (l *RedisLock) Acquire(ctx context.Context) (bool, error) {
    result, err := l.client.SetNX(ctx, l.key, l.value, l.ttl).Result()
    if err != nil {
        return false, err
    }
    return result, nil
}

func (l *RedisLock) Release(ctx context.Context) error {
    script := `
        if redis.call("get", KEYS[1]) == ARGV[1] then
            return redis.call("del", KEYS[1])
        else
            return 0
        end
    `
    _, err := l.client.Eval(ctx, script, []string{l.key}, l.value).Result()
    return err
}

// 使用示例
func (uc *VehicleUseCase) CalculateFeeWithLock(ctx context.Context, recordID string) (*Bill, error) {
    lock := lock.NewRedisLock("parking:lock:exit:"+recordID, 30*time.Second)
    acquired, err := lock.Acquire(ctx)
    if err != nil {
        return nil, err
    }
    if !acquired {
        return nil, errors.New("failed to acquire lock")
    }
    defer lock.Release(ctx)
    
    return uc.calculateFee(ctx, recordID)
}
```

**MQTT 设备通信**：实现了基于 MQTT 的设备控制，支持远程开闸、关闸等操作：

```go
// internal/vehicle/data/mqtt/client.go
type MQTTClient struct {
    client mqtt.Client
}

func (c *MQTTClient) OpenGate(deviceID, recordID string) error {
    topic := fmt.Sprintf("device/%s/command", deviceID)
    payload := Command{
        Type:     "open_gate",
        RecordID: recordID,
        Time:     time.Now().Unix(),
    }
    data, _ := json.Marshal(payload)
    
    token := c.client.Publish(topic, 1, false, data)
    token.Wait()
    return token.Error()
}
```

**监控和追踪**：集成了 Prometheus 和 OpenTelemetry，实现了完整的监控体系：

```go
// pkg/metrics/prometheus.go
var (
    EntryCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "vehicle_entry_total",
            Help: "Total number of vehicle entries",
        },
        []string{"lot_id", "vehicle_type"},
    )
    
    EntryDuration = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "vehicle_entry_duration_seconds",
            Help:    "Duration of vehicle entry processing",
            Buckets: []float64{.1, .5, 1, 2, 5},
        },
        []string{"lot_id"},
    )
)

// 使用示例
func (uc *VehicleUseCase) HandleEntry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryResult, error) {
    start := time.Now()
    defer func() {
        duration := time.Since(start).Seconds()
        EntryDuration.WithLabelValues(req.LotId).Observe(duration)
    }()
    
    // ... 处理入场逻辑
    
    EntryCounter.WithLabelValues(req.LotId, "temporary").Inc()
    return result, nil
}
```

**容器化部署**：提供了 Docker Compose 和 Kubernetes 部署方案：

```yaml
# deploy/docker-compose.yml
version: '3.8'
services:
  vehicle-svc:
    build:
      context: .
      dockerfile: deploy/docker/Dockerfile.vehicle
    ports:
      - "8001:8001"
    environment:
      - DATABASE_URL=postgres://postgres:postgres@postgres:5432/parking
      - REDIS_URL=redis://redis:6379
      - ETCD_ENDPOINTS=etcd:2379
    depends_on:
      - postgres
      - redis
      - etcd
```

v0.3 版本的架构更加完善，支持水平扩展、高可用部署、完整监控等生产级特性。

## 五、最佳实践

### 5.1 架构演进的经验教训

**不要过早微服务化**：微服务架构虽然有很多优势，但也带来了分布式系统的复杂性。在项目初期，业务需求不明确，团队规模小，单体架构是更好的选择。只有当单体架构成为瓶颈时，才考虑微服务化。

**服务拆分要适度**：服务拆分过细会导致服务数量爆炸，增加运维复杂度；拆分过粗则无法发挥微服务的优势。我们建议基于 DDD 的限界上下文进行拆分，每个服务应该是一个独立的业务单元。

**数据一致性是最大挑战**：微服务架构中，跨服务的事务处理是最复杂的。我们采用 Saga 模式和事件驱动架构保证最终一致性，但这也增加了系统的复杂度。在设计阶段就要充分考虑数据一致性问题。

**监控和调试必不可少**：微服务架构的监控和调试比单体架构复杂得多。分布式追踪、日志聚合、指标监控是必需的基础设施。我们建议在微服务化之前就搭建好监控体系。

**团队协作模式要调整**：微服务架构要求团队有更高的自治能力和协作能力。每个服务团队需要对服务的全生命周期负责，包括开发、测试、部署、运维。这需要团队文化和流程的调整。

### 5.2 常见问题和解决方案

**问题 1：服务间调用失败如何处理？**

解决方案：实现重试机制和熔断降级。Kratos 框架内置了熔断器，可以配置重试策略：

```go
// 使用 Kratos 的中间件
client := grpc.NewClient(
    "discovery:///billing-svc",
    grpc.WithMiddleware(
        recovery.Recovery(),
        tracing.Client(),
        circuitbreaker.Client(),
        retry.Client(retry.WithMaxAttempts(3)),
    ),
)
```

**问题 2：如何保证消息队列的可靠性？**

解决方案：实现消息确认机制和死信队列。消息消费失败后，重试 3 次，如果仍然失败，进入死信队列，人工处理：

```go
func (s *PaymentService) ConsumePaymentCallback() {
    for {
        streams, err := s.redisClient.XRead(ctx, &redis.XReadArgs{
            Streams: []string{"payment:callback", "$"},
            Count:   10,
            Block:   5 * time.Second,
        }).Result()
        
        if err != nil {
            continue
        }
        
        for _, stream := range streams {
            for _, message := range stream.Messages {
                err := s.handlePaymentCallback(message.Values)
                if err != nil {
                    // 重试 3 次后进入死信队列
                    s.moveToDeadLetterQueue(message)
                } else {
                    // 确认消息
                    s.redisClient.XDel(ctx, "payment:callback", message.ID)
                }
            }
        }
    }
}
```

**问题 3：如何处理分布式事务？**

解决方案：采用 Saga 模式，每个步骤都有对应的补偿操作。如果某个步骤失败，执行前面步骤的补偿操作，保证最终一致性。

**问题 4：如何实现灰度发布？**

解决方案：使用 Kubernetes 的 Deployment 和 Service，通过标签选择器实现灰度发布。例如，先部署 10% 的新版本流量，观察监控指标，如果正常，逐步增加流量。

### 5.3 性能优化建议

**数据库优化**：
- 为高频查询添加索引，例如 `(plate_number, entry_time)` 联合索引
- 使用连接池，避免频繁创建连接
- 读写分离，将查询操作路由到只读副本

**缓存优化**：
- 使用 Redis 缓存热点数据，例如车辆信息、计费规则
- 实现多级缓存，本地缓存 + Redis 缓存
- 缓存预热，在系统启动时加载热点数据

**服务优化**：
- 使用 gRPC 进行服务间通信，性能优于 HTTP
- 实现异步处理，将非核心逻辑异步化
- 使用连接池和对象池，减少资源创建开销

**网络优化**：
- 启用 HTTP/2，提高并发性能
- 使用 CDN 加速静态资源
- 启用 Gzip 压缩，减少网络传输

## 六、总结

Smart Park 项目从 v0.1 的单体架构演进到 v0.3 的微服务架构，是一次成功的架构升级。这次演进不仅解决了业务发展带来的技术挑战，也为系统的长期发展奠定了坚实基础。

回顾整个演进过程，我们总结出以下核心要点：

**架构演进要循序渐进**：不要为了微服务而微服务，要根据业务需求和团队规模选择合适的架构。单体架构在项目初期是合理的选择，只有当单体架构成为瓶颈时，才考虑微服务化。

**服务拆分要基于业务边界**：采用 DDD 的限界上下文进行服务拆分，每个服务应该是一个独立的业务单元。服务之间通过明确定义的 API 通信，避免紧耦合。

**数据一致性是核心挑战**：微服务架构中，分布式事务处理是最复杂的。采用 Saga 模式和事件驱动架构保证最终一致性，但也要接受最终一致性的代价。

**基础设施要先行**：微服务架构需要完善的基础设施支持，包括服务发现、配置中心、监控告警、日志聚合、分布式追踪等。在微服务化之前，要先搭建好这些基础设施。

**团队协作模式要调整**：微服务架构要求团队有更高的自治能力和协作能力。每个服务团队需要对服务的全生命周期负责，这需要团队文化和流程的调整。

展望未来，Smart Park 项目将继续演进，计划在 v0.4 版本引入以下特性：

- **多租户支持**：支持 SaaS 部署模式，实现租户隔离和独立计费
- **充电桩对接**：支持电动汽车充电桩的对接和管理
- **数据分析服务**：提供更丰富的数据分析和商业智能功能
- **AI 智能定价**：基于历史数据和实时流量，实现智能定价

微服务架构为这些新功能的快速迭代提供了坚实基础。我们相信，随着技术的不断演进和业务的持续发展，Smart Park 将成为智慧停车领域的标杆项目。

## 参考资料

1. Kratos 框架官方文档：https://go-kratos.dev/
2. 领域驱动设计：Eric Evans 著
3. 微服务架构设计模式：Chris Richardson 著
4. Smart Park 项目文档：docs/parking-system-arch.md
5. Smart Park 变更日志：CHANGELOG.md

---

**作者**：Smart Park Team  
**日期**：2026-03-31  
**版本**：v1.0

# Smart Park Code Wiki
## Table of Contents
1. [Project Overview](#project-overview)
2. [Architecture](#architecture)
3. [Microservices](#microservices)
4. [Key Packages](#key-packages)
5. [API Definitions](#api-definitions)
6. [Deployment](#deployment)
7. [Frontend](#frontend)
8. [Running the Project](#running-the-project)


## Project Overview
**Smart Park** is a modern, microservices-based smart parking management system built with Go (Kratos framework) and TypeScript/React (Next.js).

### Key Features
- Vehicle entry/exit management with license plate recognition
- Intelligent billing with flexible rule engine
- Payment integration (WeChat Pay, Alipay)
- Device management (cameras, gates)
- Multi-tenant support
- Monitoring, alerting, and distributed tracing


## Architecture
Smart Park uses a **microservices architecture** with:
- **API Gateway**: Routes requests to appropriate services
- **Microservices**: Vehicle, Billing, Payment, Admin, Charging, Multitenancy
- **Databases**: PostgreSQL (primary), Redis (caching, distributed locks)
- **Infrastructure**: Docker, Kubernetes, Prometheus, Grafana, Jaeger

```
┌─────────────────────────────────────────────────────────────────┐
│                         Access Layer                             │
│  (车主小程序, 管理后台, 设备网关, 第三方回调)                  │
└───────────────────────┬─────────────────────────────────────────┘
                        │
                    ┌───▼──────────────┐
                    │   API Gateway     │
                    │   (Port: 8000)   │
                    └───┬──────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
┌───────▼───────┐ ┌───▼───────┐ ┌───▼───────┐
│  Vehicle Svc  │ │ Billing   │ │ Payment   │
│  (Port:8001)  │ │ (Port:8002)│ │ (Port:8003)│
└───────┬───────┘ └───┬───────┘ └───┬───────┘
        │               │               │
┌───────▼───────┐ ┌───▼───────┐ ┌───▼───────┐
│  Admin Svc    │ │ Charging  │ │ Multi-    │
│  (Port:8004)  │ │ (Port:8005)│ │ Tenancy   │
└───────────────┘ └───────────┘ └───────────┘
```


## Microservices

### 1. Gateway Service (`cmd/gateway/`)
- **Port**: 8000
- **Responsibilities**:
  - API routing and forwarding
  - Health checks (`/health`, `/ready`)
  - Prometheus metrics endpoint (`/metrics`)
- **Key Files**: [main.go](file:///workspace/cmd/gateway/main.go)

### 2. Vehicle Service (`cmd/vehicle/`)
- **Port**: 8001 (HTTP), 9001 (gRPC)
- **Responsibilities**:
  - Vehicle entry/exit processing
  - Device management (cameras, gates)
  - Parking record management
  - MQTT integration for device communication
- **Key Files**:
  - [main.go](file:///workspace/cmd/vehicle/main.go) - Entry point
  - [internal/vehicle/biz/entry_exit.go](file:///workspace/internal/vehicle/biz/entry_exit.go) - Entry/exit business logic
  - [internal/vehicle/device/](file:///workspace/internal/vehicle/device/) - Device adapters (Dahua, Hikvision, Jieshun, etc.)

#### Key Classes/Functions
- `EntryExitUseCase` - Handles vehicle entry and exit with distributed locks and transactions
- `Entry()` - Processes vehicle entry
- `Exit()` - Processes vehicle exit with fee calculation
- `DeviceAdapterFactory` - Factory for creating device-specific adapters

### 3. Billing Service (`cmd/billing/`)
- **Port**: 8002 (HTTP), 9002 (gRPC)
- **Responsibilities**:
  - Flexible billing rule engine
  - Fee calculation based on time, vehicle type, etc.
- **Key Files**:
  - [main.go](file:///workspace/cmd/billing/main.go)
  - [internal/billing/biz/billing.go](file:///workspace/internal/billing/biz/billing.go) - Billing rule engine

#### Key Classes/Functions
- `BillingUseCase` - Implements billing business logic
- `CalculateFee()` - Calculates parking fees using rules
- `EvaluateCondition()` - Evaluates billing rule conditions
- `applyActions()` - Applies billing actions (fixed, per-hour, discounts, etc.)
- `ParseConditions()` / `ParseActions()` - Parse JSON rule definitions

### 4. Payment Service (`cmd/payment/`)
- **Port**: 8003 (HTTP), 9003 (gRPC)
- **Responsibilities**:
  - Payment order management
  - WeChat Pay and Alipay integration
  - Refund processing
  - Reconciliation
- **Key Files**:
  - [main.go](file:///workspace/cmd/payment/main.go)
  - [internal/payment/biz/payment.go](file:///workspace/internal/payment/biz/payment.go) - Payment business logic
  - [internal/payment/wechat/](file:///workspace/internal/payment/wechat/) - WeChat Pay client
  - [internal/payment/alipay/](file:///workspace/internal/payment/alipay/) - Alipay client

#### Key Classes/Functions
- `PaymentUseCase` - Payment business logic
- `CreatePayment()` - Creates payment order and returns payment URL/QR code
- `GetPaymentStatus()` - Retrieves payment status
- `Refund()` - Processes refunds

### 5. Admin Service (`cmd/admin/`)
- **Port**: 8004
- **Responsibilities**:
  - Parking lot management
  - User management
  - Reporting and analytics
- **Key Files**: [internal/admin/](file:///workspace/internal/admin/)

### 6. Charging Service (`cmd/charging/`)
- **Port**: 8005
- **Responsibilities**:
  - EV charging station management
  - Charging session management
- **Key Files**: [internal/charging/](file:///workspace/internal/charging/)

### 7. Multitenancy Service (`cmd/multitenancy/`)
- **Port**: 8006
- **Responsibilities**:
  - Tenant management
  - Data isolation
- **Key Files**: [internal/multitenancy/](file:///workspace/internal/multitenancy/)


## Key Packages

### `pkg/` - Shared Packages

#### `pkg/auth/`
- JWT authentication
- [jwt.go](file:///workspace/pkg/auth/jwt.go)

#### `pkg/cache/`
- Multi-level caching
- Hot data caching
- [cache.go](file:///workspace/pkg/cache/cache.go)

#### `pkg/database/`
- PostgreSQL database management with read-write separation
- [database.go](file:///workspace/pkg/database/database.go)

#### `pkg/lock/`
- Distributed lock implementation using Redis
- [lock.go](file:///workspace/pkg/lock/lock.go)

#### `pkg/logger/`
- Logging with Zap and Lumberjack for log rotation
- [logger.go](file:///workspace/pkg/logger/logger.go)

#### `pkg/metrics/`
- Prometheus metrics
- [prometheus.go](file:///workspace/pkg/metrics/prometheus.go)

#### `pkg/mq/`
- Message queue abstraction (supports NATS, Redis, RocketMQ)
- [factory.go](file:///workspace/pkg/mq/factory.go)

#### `pkg/mqtt/`
- MQTT client for device communication
- [client.go](file:///workspace/pkg/mqtt/client.go)

#### `pkg/multitenancy/`
- Multi-tenant support utilities
- [context.go](file:///workspace/pkg/multitenancy/context.go)

#### `pkg/trace/`
- OpenTelemetry + Jaeger distributed tracing
- [trace.go](file:///workspace/pkg/trace/trace.go)


## API Definitions
APIs are defined using Protocol Buffers in the [api/](file:///workspace/api/) directory.

### Available APIs
- [api/vehicle/v1/vehicle.proto](file:///workspace/api/vehicle/v1/vehicle.proto) - Vehicle and device management
- [api/billing/v1/billing.proto](file:///workspace/api/billing/v1/billing.proto) - Billing and fee calculation
- [api/payment/v1/payment.proto](file:///workspace/api/payment/v1/payment.proto) - Payment processing
- [api/admin/v1/admin.proto](file:///workspace/api/admin/v1/admin.proto) - Admin operations
- [api/multitenancy/v1/multitenancy.proto](file:///workspace/api/multitenancy/v1/multitenancy.proto) - Tenant management

### Key API Endpoints (Vehicle Service)
- `POST /api/v1/device/entry` - Vehicle entry
- `POST /api/v1/device/exit` - Vehicle exit
- `POST /api/v1/device/heartbeat` - Device heartbeat
- `GET /api/v1/devices` - List devices
- `POST /api/v1/devices` - Create device


## Deployment
Smart Park supports multiple deployment options:

### Docker Compose (for development/small-scale)
- [docker-compose.yml](file:///workspace/deploy/docker-compose.yml)
- [docker-compose.infra.yml](file:///workspace/deploy/docker-compose.infra.yml) - Infrastructure only (PostgreSQL, Redis, etcd, Jaeger)

### Kubernetes (for production/large-scale)
- Kubernetes manifests in [deploy/k8s/](file:///workspace/deploy/k8s/)
  - [namespace.yaml](file:///workspace/deploy/k8s/namespace.yaml)
  - [configmap.yaml](file:///workspace/deploy/k8s/configmap.yaml)
  - [deployment.yaml](file:///workspace/deploy/k8s/deployment.yaml)
  - [service.yaml](file:///workspace/deploy/k8s/service.yaml)
  - [hpa.yaml](file:///workspace/deploy/k8s/hpa.yaml) - Horizontal Pod Autoscaler
  - [istio-setup.yaml](file:///workspace/deploy/k8s/istio-setup.yaml) - Istio service mesh

### Monitoring & Observability
- **Prometheus**: Metrics collection - [prometheus.yml](file:///workspace/deploy/prometheus/prometheus.yml)
- **Grafana**: Visualization - [grafana/](file:///workspace/deploy/grafana/)
- **Loki**: Log aggregation - [loki.yml](file:///workspace/deploy/loki/loki.yml)
- **Jaeger**: Distributed tracing


## Frontend
The frontend is a Next.js 16 application with React 19 and TypeScript.

### Directory Structure
```
site/
├── src/
│   ├── app/              # Next.js App Router pages
│   │   ├── layout.tsx
│   │   └── page.tsx
│   ├── components/
│   │   ├── layout/       # Layout components (Header, Footer)
│   │   ├── preview/      # Dashboard preview components
│   │   ├── sections/     # Landing page sections
│   │   └── ui/           # UI components (Button, Card, etc.)
│   ├── hooks/            # Custom React hooks
│   ├── lib/              # Utilities and constants
│   └── types/            # TypeScript type definitions
├── package.json
├── tsconfig.json
└── next.config.ts
```

### Key Technologies
- Next.js 16 (App Router)
- React 19
- TypeScript 5
- Tailwind CSS 4
- Lucide React icons


## Running the Project

### Prerequisites
- Go 1.26+
- Docker 20.10+
- Docker Compose 2.20+
- Node.js 18+ (for frontend)

### Quick Start with Docker Compose
```bash
# Clone the repository
git clone https://github.com/xuanyiying/smart-park.git
cd smart-park

# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# Access the system
# - Management UI: http://localhost:3000
# - API Gateway: http://localhost:8000
```

### Local Development
```bash
# 1. Start infrastructure only
docker-compose -f deploy/docker-compose.infra.yml up -d

# 2. Install Go dependencies
go mod download

# 3. Start backend services (in separate terminals)
go run ./cmd/gateway -conf ./configs
go run ./cmd/vehicle -conf ./configs
go run ./cmd/billing -conf ./configs
go run ./cmd/payment -conf ./configs
go run ./cmd/admin -conf ./configs

# 4. Start frontend
cd site
npm install
npm run dev
```

### Scripts
Helper scripts are available in [scripts/](file:///workspace/scripts/):
- [start-all.sh](file:///workspace/scripts/start-all.sh) - Start all services
- [stop-all.sh](file:///workspace/scripts/stop-all.sh) - Stop all services
- [start-dev.sh](file:///workspace/scripts/start-dev.sh) - Start development environment
- [generate_proto.sh](file:///workspace/scripts/generate_proto.sh) - Generate Go code from proto files


## Configuration
Service configurations are stored in [configs/](file:///workspace/configs/):
- [gateway.yaml](file:///workspace/configs/gateway.yaml)
- [vehicle.yaml](file:///workspace/configs/vehicle.yaml)
- [billing.yaml](file:///workspace/configs/billing.yaml)
- [payment.yaml](file:///workspace/configs/payment.yaml)
- [admin.yaml](file:///workspace/configs/admin.yaml)
- [seata.yaml](file:///workspace/configs/seata.yaml) - Distributed transaction configuration


## Key Dependencies
### Backend
- [Kratos](https://github.com/go-kratos/kratos) - Microservices framework
- [gRPC](https://grpc.io/) - RPC communication
- [Ent](https://entgo.io/) - ORM framework
- [PostgreSQL](https://www.postgresql.org/) - Primary database
- [Redis](https://redis.io/) - Caching and distributed locks
- [etcd](https://etcd.io/) - Service discovery
- [OpenTelemetry](https://opentelemetry.io/) - Observability
- [Jaeger](https://www.jaegertracing.io/) - Distributed tracing

### Frontend
- [Next.js](https://nextjs.org/) - React framework
- [React](https://react.dev/) - UI library
- [TypeScript](https://www.typescriptlang.org/) - Type safety
- [Tailwind CSS](https://tailwindcss.com/) - Utility-first CSS framework

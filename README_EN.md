# Smart Park - Intelligent Parking Management System

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![Kratos](https://img.shields.io/badge/Kratos-2.9+-green.svg)](https://github.com/go-kratos/kratos)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## Project Overview

Smart Park is a modern intelligent parking management system built with Go language and the Kratos microservices framework. The system adopts a microservices architecture design, supporting core functions such as license plate recognition, intelligent billing, multi-channel payment, and device control, suitable for various parking scenarios including commercial complexes, residential communities, and office buildings.

### Core Features

- **Microservices Architecture**: Based on the Kratos framework, services communicate via gRPC, supporting independent deployment and scaling
- **Dual Protocol Support**: Simultaneously supports gRPC and HTTP/REST APIs for easy integration with different clients
- **Intelligent Billing Engine**: Supports multiple billing rules (hourly, time-based, monthly pass, VIP, etc.) with rule stacking capability
- **Multi-Channel Payment**: Integrated WeChat Pay and Alipay, supporting QR code payment and JSAPI payment
- **Device Management**: Supports access and control of license plate recognition cameras, barriers, ground sensors, and other devices
- **Distributed Locking**: Redis-based distributed locks ensure data consistency in concurrent scenarios
- **Distributed Tracing**: Integrated OpenTelemetry and Jaeger for full-link tracing
- **Multi-Tenancy Support**: Supports multi-parking lot management with data isolation

## System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Access Layer                             │
├───────────────┬───────────────┬───────────────┬──────────────────┤
│  Driver Mini  │   Admin Panel │  Device GW    │  3rd Party CB    │
│     App       │   (Web)       │               │                  │
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
│   (Primary)   │    │  (Cache/Lock) │    │ (Discovery)   │
└───────────────┘    └───────────────┘    └───────────────┘
```

## Technology Stack

| Layer | Technology |
|-------|------------|
| Backend Framework | [Kratos](https://github.com/go-kratos/kratos) v2.9 |
| Language | Go 1.26+ |
| API Protocol | gRPC + HTTP/REST (grpc-gateway) |
| Database | PostgreSQL 15 |
| ORM | Ent |
| Cache | Redis 7 |
| Service Discovery | Etcd v3.5 |
| Distributed Tracing | OpenTelemetry + Jaeger |
| Message Queue | Redis Streams / NATS / RocketMQ |
| Payment SDK | wechatpay-go, alipay |
| Containerization | Docker, Docker Compose |

## Quick Start

### Prerequisites

- Go 1.26+
- Docker & Docker Compose
- Make (optional)

### Start Infrastructure

```bash
# Start PostgreSQL, Redis, Etcd, Jaeger
docker-compose -f deploy/docker-compose.yml up -d postgres redis etcd jaeger
```

### Run Services Locally

```bash
# Run Gateway Service
go run ./cmd/gateway -conf ./configs

# Run Vehicle Service
go run ./cmd/vehicle -conf ./configs

# Run Billing Service
go run ./cmd/billing -conf ./configs

# Run Payment Service
go run ./cmd/payment -conf ./configs

# Run Admin Service
go run ./cmd/admin -conf ./configs
```

### Build Docker Images

```bash
# Build all services
docker-compose -f deploy/docker-compose.yml build

# Start full stack
docker-compose -f deploy/docker-compose.yml up -d
```

## Service Documentation

| Service | Port | Description | Documentation |
|---------|------|-------------|---------------|
| Gateway | 8000 | API Gateway for request routing | [docs/gateway.md](docs/gateway.md) |
| Vehicle | 8001 | Vehicle entry/exit, device management | [docs/vehicle_EN.md](docs/vehicle_EN.md) |
| Billing | 8002 | Fee calculation and billing rules | [docs/billing_EN.md](docs/billing_EN.md) |
| Payment | 8003 | Payment processing and order management | [docs/payment_EN.md](docs/payment_EN.md) |
| Admin | 8004 | Parking lot management and reports | [docs/admin_EN.md](docs/admin_EN.md) |
| Charging | 8005 | Charging pile management | [docs/charging.md](docs/charging.md) |

## API Endpoints

### Device APIs

- `POST /api/v1/device/entry` - Vehicle entry
- `POST /api/v1/device/exit` - Vehicle exit
- `POST /api/v1/device/heartbeat` - Device heartbeat
- `GET /api/v1/device/{id}/status` - Device status query
- `POST /api/v1/device/{id}/command` - Device control command

### Billing APIs

- `POST /api/v1/billing/calculate` - Fee calculation
- `GET /api/v1/admin/billing/rules` - Billing rules list
- `POST /api/v1/admin/billing/rules` - Create billing rule
- `PUT /api/v1/admin/billing/rules/{id}` - Update billing rule
- `DELETE /api/v1/admin/billing/rules/{id}` - Delete billing rule

### Payment APIs

- `POST /api/v1/pay/create` - Create payment order
- `GET /api/v1/pay/{id}/status` - Query payment status
- `POST /api/v1/pay/callback/wechat` - WeChat Pay callback
- `POST /api/v1/pay/callback/alipay` - Alipay callback
- `POST /api/v1/pay/{id}/refund` - Refund request

### Admin APIs

- `GET /api/v1/admin/lots` - Parking lot list
- `POST /api/v1/admin/lots` - Create parking lot
- `GET /api/v1/admin/records` - Entry/exit records
- `GET /api/v1/admin/orders` - Order list
- `GET /api/v1/admin/reports/daily` - Daily report
- `GET /api/v1/admin/reports/monthly` - Monthly report

## Project Structure

```
smart-park/
├── api/                    # Protocol Buffers definitions
│   ├── admin/v1/
│   ├── billing/v1/
│   ├── payment/v1/
│   ├── vehicle/v1/
│   └── ...
├── cmd/                    # Service entry points
│   ├── gateway/
│   ├── vehicle/
│   ├── billing/
│   ├── payment/
│   └── admin/
├── configs/                # Configuration files per service
├── deploy/                 # Deployment configurations
│   ├── docker/
│   ├── docker-compose.yml
│   └── k8s/                # Kubernetes manifests
├── docs/                   # Documentation
├── internal/               # Internal implementations
│   ├── admin/
│   ├── billing/
│   ├── payment/
│   ├── vehicle/
│   └── pkg/               # Common packages
├── pkg/                    # Reusable public libraries
│   ├── auth/              # JWT authentication
│   ├── cache/             # Cache wrapper
│   ├── lock/              # Distributed lock
│   ├── middleware/        # Middleware
│   └── ...
├── web/                    # Admin web frontend (Next.js)
└── scripts/                # Utility scripts
```

## Development Guide

### Generate Protocol Buffers

```bash
./scripts/generate_proto.sh
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run specific service tests
go test ./internal/vehicle/...

# Run integration tests
go test ./tests/... -tags=integration
```

### Code Standards

The project uses golangci-lint for code checking:

```bash
golangci-lint run
```

## Production Deployment

For detailed deployment documentation, please refer to [docs/deployment_EN.md](docs/deployment_EN.md)

### Deployment Architectures

- **Small Scale** (1-20 parking lots): Single-node Docker Compose deployment
- **Medium Scale** (20-100 parking lots): Active-standby dual-active architecture
- **Large Scale** (100+ parking lots): Kubernetes multi-active cluster

## Contributing

1. Fork this repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details

## Contact

- Project Homepage: https://github.com/xuanyiying/smart-park
- Issue Tracker: https://github.com/xuanyiying/smart-park/issues

---

**Smart Park** - Making Parking Smarter, Making Travel Easier

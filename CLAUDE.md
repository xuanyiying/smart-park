# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Smart Park is a parking lot management and billing system built with Go microservices using the Kratos framework. The system handles vehicle entry/exit, billing calculation, payment processing, and administrative management.

## Build and Run Commands

```bash
# Start infrastructure (PostgreSQL, Redis, Etcd, Jaeger)
docker-compose -f deploy/docker-compose.yml up -d postgres redis etcd jaeger

# Build a specific service
go build -o bin/vehicle-svc ./cmd/vehicle
go build -o bin/billing-svc ./cmd/billing
go build -o bin/payment-svc ./cmd/payment
go build -o bin/admin-svc ./cmd/admin
go build -o bin/gateway-svc ./cmd/gateway

# Run a service locally (requires running infrastructure)
go run ./cmd/vehicle -conf ./configs
go run ./cmd/billing -conf ./configs
go run ./cmd/payment -conf ./configs
go run ./cmd/admin -conf ./configs
go run ./cmd/gateway -conf ./configs

# Build all services with Docker
docker-compose -f deploy/docker-compose.yml build

# Run all services
docker-compose -f deploy/docker-compose.yml up -d
```

## Architecture

### Microservices

The system follows a microservices architecture with these services:

| Service | Port | Purpose |
|---------|------|---------|
| Gateway | 8000 | API gateway routing requests to backend services |
| Vehicle | 8001 | Vehicle entry/exit, device management, plate recognition |
| Billing | 8002 | Fee calculation and billing rules management |
| Payment | 8003 | Payment processing (WeChat Pay, Alipay) |
| Admin | 8004 | Parking lot management, reports, vehicle administration |
| Notification | - | Notification service (placeholder) |

### Technology Stack

- **Framework**: [Kratos](https://github.com/go-kratos/kratos) - Go microservices framework
- **ORM**: Ent - Entity framework for Go
- **Database**: PostgreSQL 15
- **Cache/Message Queue**: Redis 7
- **Service Discovery**: Etcd v3.5
- **Tracing**: Jaeger (OpenTelemetry)
- **API Definition**: Protocol Buffers with gRPC-Gateway annotations

### Directory Structure

```
smart-park/
├── api/                    # Proto definitions for each service
│   ├── admin/v1/
│   ├── billing/v1/
│   ├── payment/v1/
│   └── vehicle/v1/
├── cmd/                    # Service entry points
│   ├── admin/main.go
│   ├── billing/main.go
│   ├── gateway/main.go
│   ├── notification/main.go
│   ├── payment/main.go
│   └── vehicle/main.go
├── configs/                # YAML configuration files per service
├── deploy/
│   ├── docker/             # Dockerfiles for each service
│   ├── docker-compose.yml  # Full stack deployment
│   └── k8s/                # Kubernetes manifests
└── parking-system-arch.md  # Detailed architecture documentation (Chinese)
```

### Service Communication

- Gateway routes HTTP requests to backend services based on path prefix
- Services communicate via gRPC internally
- Redis streams used for async messaging between services
- OpenTelemetry for distributed tracing

### Database Connection

Default connection string (from configs):
```
postgres://postgres:postgres@localhost:5432/parking?sslmode=disable
```

### API Endpoints

Defined in proto files under `api/*/v1/*.proto`:
- `/api/v1/device/*` → Vehicle service
- `/api/v1/billing/*` → Billing service
- `/api/v1/pay/*` → Payment service
- `/api/v1/admin/*` → Admin service

## Development Notes

- Each service uses the Kratos framework's standard structure with dependency injection
- Services run database migrations on startup via `ent.Migrate()`
- Configuration is loaded from YAML files specified via `-conf` flag
- Proto files define both gRPC and HTTP endpoints using `google.api.http` annotations

## Implementation Status

See [TODO.md](./TODO.md) for detailed implementation status.

### Current State (v0.2)

| Component | Status | Notes |
|-----------|--------|-------|
| Proto Definitions | ✅ Complete | All API endpoints defined |
| Business Logic | ✅ Complete | UseCase layer implemented |
| Service Layer | ✅ Complete | gRPC services implemented |
| Repository Layer | 🔲 Todo | Data layer needs implementation |
| Gateway | ✅ Complete | Basic routing working |
| Payment Integration | 🔲 Partial | SDK integration pending |
| User APIs | 🔲 Todo | Not implemented yet |
| Notification | 🔲 Todo | Placeholder only |

### Gateway Routes

```
/api/v1/device/*  → vehicle-svc:8001
/api/v1/billing/* → billing-svc:8002
/api/v1/pay/*     → payment-svc:8003
/api/v1/admin/*    → admin-svc:8004
```

### Next Steps

1. **Implement Repository Layer** - Complete data access layer for all services
2. **Integrate Payment SDKs** - WeChat Pay and Alipay
3. **Implement User APIs** - User authentication, plate binding, QR payment
4. **Device Control** - MQTT/WebSocket for gate control

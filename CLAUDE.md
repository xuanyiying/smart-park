# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Smart Park is a parking lot management and billing system built with Go microservices using the Kratos framework. The system handles vehicle entry/exit, billing calculation, payment processing, EV charging, multi-tenancy, and administrative management.

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
go build -o bin/charging-svc ./cmd/charging
go build -o bin/user-svc ./cmd/user

# Run a service locally (requires running infrastructure)
go run ./cmd/vehicle -conf ./configs
go run ./cmd/billing -conf ./configs
go run ./cmd/payment -conf ./configs
go run ./cmd/admin -conf ./configs
go run ./cmd/gateway -conf ./configs
go run ./cmd/charging -conf ./configs
go run ./cmd/user -conf ./configs

# Build all services with Docker
docker-compose -f deploy/docker-compose.yml build

# Run all services
docker-compose -f deploy/docker-compose.yml up -d

# Tests (vet + lint + test + coverage gate), integration tests optional
./scripts/test.sh
./scripts/test.sh --integration
```

## Architecture

### Microservices

The system follows a microservices architecture with these services:

| Service | HTTP Port | gRPC Port (hardcoded in `cmd/*/main.go`) | Purpose |
|---------|-----------|------|---------|
| Gateway | 8000 | - | API gateway: routing, JWT auth, WebSocket hub (no gRPC server) |
| Vehicle | 8001 | 9001 | Vehicle entry/exit, device management, MQTT device control |
| Billing | 8002 | 9002 | Fee calculation and billing rules engine |
| Payment | 8003 | 9003 | Payment processing (WeChat Pay, Alipay), reconciliation, outbox |
| Admin | 8004 | 9004 | Parking lot management, reports, vehicle administration |
| Charging | 8005 | 9005 | EV charging stations, sessions, pricing |
| Multitenancy | 8006 | 9006 | Multi-tenant management (gRPC only, no HTTP routes) |
| User | 8007 | 9007 | WeChat login, plate binding, scan-to-pay |
| Notification | 8008 | 9008 | Notification service (placeholder: assembled but not registered) |

### Technology Stack

- **Framework**: [Kratos](https://github.com/go-kratos/kratos) v2.9 - Go microservices framework
- **ORM**: Ent v0.14 - Entity framework for Go
- **Database**: PostgreSQL 15
- **Cache/Message Queue**: Redis 7 (Redis Streams for async messaging)
- **Service Discovery**: Etcd v3.5
- **Tracing**: Jaeger (OpenTelemetry)
- **API Definition**: Protocol Buffers with gRPC-Gateway annotations
- **Frontend**: `web/` admin dashboard (Next.js 16 + React 19 + Tailwind 4), `site/` marketing site

### Directory Structure

```
smart-park/
├── api/                    # Proto definitions for each service
│   ├── admin/v1/  ├── analytics/v1/  ├── billing/v1/  ├── charging/v1/
│   ├── multitenancy/v1/  ├── notification/v1/  ├── payment/v1/
│   ├── user/v1/   └── vehicle/v1/
├── cmd/                    # Service entry points (9 services, manual DI wiring)
├── configs/                # YAML configuration files per service (flat structure)
├── internal/               # Per-service biz/data/service layers
├── pkg/                    # Shared libraries (auth, cache, lock, outbox, tenant, ...)
├── deploy/                 # Dockerfiles, docker-compose, k8s manifests
├── scripts/                # Dev/test/proto scripts
├── web/                    # Admin web frontend (Next.js)
├── site/                   # Marketing site (Next.js)
└── openapi.yaml            # Generated OpenAPI spec (VehicleService)
```

### Service Communication

- Gateway routes HTTP requests to backend services based on path prefix (see `configs/gateway.yaml`)
- Services communicate via gRPC internally (ports 9001-9008, hardcoded in each `cmd/*/main.go`)
- Redis Streams used for async messaging between services
- OpenTelemetry for distributed tracing; transactional outbox in `pkg/outbox` for payment events

### Gateway Routes

From `configs/gateway.yaml`:

```
/api/v1/device       → vehicle:8001
/api/v1/billing      → billing:8002
/api/v1/pay          → payment:8003
/api/v1/admin/billing → billing:8002
/api/v1/admin        → admin:8004
/api/v1/charging     → charging:8005
/api/v1/user         → user:8007
```

Gateway self endpoints: `/health`, `/ready`, `/routes`, `/ws` (JWT required), `/metrics`.
JWT is enforced on all `/api/v1/*` paths except skip paths: `/health`, `/ready`, `/api/v1/user/login`, `/api/v1/user/register` (note: user service has no register implementation yet).

### Configuration

- Config files use a **flat structure** (`server.port`, `database.primary.source`, ...), loaded via the `-conf` flag
- There is no `server.http.addr` / `grpc.addr` nesting; gRPC ports are hardcoded in each `cmd/*/main.go`
- Sensitive values are overridden via `SP_{SECTION}_{KEY}` environment variables (e.g. `SP_DATABASE_SOURCE`, `SP_JWT_SECRET`); `ENC(...)` values need `SP_CONFIG_ENCRYPTION_KEY`
- Money amounts are **int64 cents (分)** across all protos and business logic; the billing engine keeps float64 internally and converts once at the API boundary

### Database Connection

Default connection string (from configs):
```
host=localhost user=postgres password=postgres dbname=parking port=5432 sslmode=disable
```

## Development Notes

- Each service uses the Kratos framework structure with **manual dependency wiring** in `cmd/*/main.go` (wire is a dependency but no generated wire code)
- Services run database migrations on startup via Ent `Schema.Create` (no separate migrate command)
- Configuration is loaded from YAML files specified via `-conf` flag
- Proto files define both gRPC and HTTP endpoints using `google.api.http` annotations; regenerate with `./scripts/generate_proto.sh`
- Strict startup validation: vehicle requires MQTT broker, payment requires WeChat/Alipay credentials, gateway requires `jwt.public_key_path` — unconfigured services refuse to start instead of degrading silently

## Implementation Status

All core services are implemented with real integrations (WeChat APIv3 signature verification, Alipay signature verification, MQTT device commands, reconciliation, outbox, tenant interceptors). Detailed fix history: [docs/code-audit-report.md](docs/code-audit-report.md).

| Component | Status | Notes |
|-----------|--------|-------|
| Proto Definitions | ✅ Complete | All API endpoints defined |
| Business Logic | ✅ Complete | UseCase layer implemented |
| Service/Data Layers | ✅ Complete | gRPC services and Ent repositories implemented |
| Gateway | ✅ Complete | Routing + JWT auth + WebSocket |
| Payment Integration | ✅ Complete | Real WeChat/Alipay SDK calls with idempotent callbacks |
| User APIs | ✅ Complete | WeChat login, plates, scan-to-pay (payment history is simplified) |
| Notification | 🚧 Placeholder | Service assembled in main.go but never registered |
| Admin Refund Approvals / RBAC | 🚧 Planned | Designed in docs, not implemented in code |
| Admin Web Frontend | ✅ Complete | `web/` Next.js dashboard |

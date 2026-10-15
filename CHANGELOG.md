# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Removed

- Stray compiled binaries (`charging`, `gateway`) accidentally committed to the repository root

### Changed

- Moved one-off working documents (`code_audit_report.md`, `GITHUB_ABOUT.md`) into `docs/`
- Updated `.gitignore` to ignore root-level service build artifacts
- Documentation overhaul: aligned `docs/` service docs, README, README_EN and CLAUDE.md with the actual code —
  money amounts documented as int64 cents, flat `server.port` config format, real gateway route table (7 routes),
  correct service inventory (9 services, ports 8000-8008 / gRPC 9001-9008), removed or marked as planned the
  undocumented APIs (HMAC device signatures, refund approvals, `reports/revenue`, `refundWindow`,
  `docker-compose.prod.yml`, `cmd/migrate`), fixed broken doc links and outdated frontend stack info
  (web/ is a Next.js 16 admin dashboard, no longer "not implemented")

### Added

- Charging service for electric vehicle charging stations
- Multi-tenancy support for SaaS deployment
- Analytics service for business intelligence

## [0.3.0] - 2026-03-29

### Added

- User service with WeChat login integration
- JWT authentication middleware
- Vehicle entry/exit flow with automatic gate control
- Payment callback handling with signature verification
- Distributed lock implementation using Redis
- MQTT client for device communication
- Prometheus metrics for monitoring
- OpenTelemetry tracing integration
- Docker Compose deployment configuration
- Kubernetes deployment manifests with HPA

### Changed

- Improved billing rule engine with flexible conditions
- Enhanced payment security with amount validation
- Optimized database connection pool settings

### Fixed

- Race condition in device command manager (#42)
- Memory leak in MQTT client reconnection logic
- Incorrect billing calculation for overnight parking

### Security

- Added signature verification for payment callbacks
- Implemented rate limiting for API endpoints
- Added input validation for all user inputs

## [0.2.0] - 2026-02-15

### Added

- Billing service with rule engine
- Payment service (WeChat Pay, Alipay)
- Admin service for parking lot management
- Gateway service for API routing
- Vehicle service for entry/exit management

### Changed

- Migrated from monolith to microservices architecture
- Upgraded to Kratos v2.9

### Fixed

- Duplicate entry prevention using distributed locks
- Correct fee calculation for monthly vehicles

## [0.1.0] - 2026-01-15

### Added

- Initial release
- Basic parking lot management
- Simple billing calculation
- Manual payment recording

[Unreleased]: https://github.com/xuanyiying/smart-park/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/xuanyiying/smart-park/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/xuanyiying/smart-park/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/xuanyiying/smart-park/releases/tag/v0.1.0

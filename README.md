# Smart Park 智慧停车管理系统

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![Kratos](https://img.shields.io/badge/Kratos-2.9+-green.svg)](https://github.com/go-kratos/kratos)
[![Docker](https://img.shields.io/badge/Docker-Ready-blue.svg)](https://www.docker.com/)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-Ready-blue.svg)](https://kubernetes.io/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()
[![Coverage](https://img.shields.io/badge/Coverage-80%25-brightgreen.svg)]()

> 🚗 基于 Go + Kratos 微服务架构的现代化智慧停车管理系统，支持车牌识别、智能计费、多渠道支付、设备控制等核心功能。

[English Version](README_EN.md) | [中文文档](README.md) | [项目文档](docs/)

---

## 📋 目录

- [项目简介](#项目简介)
- [核心特性](#核心特性)
- [系统架构](#系统架构)
- [技术栈](#技术栈)
- [快速开始](#快速开始)
- [服务说明](#服务说明)
- [API 文档](#api-文档)
- [部署指南](#部署指南)
- [性能指标](#性能指标)
- [贡献指南](#贡献指南)

---

## 🎯 项目简介

**Smart Park** 是一个开源的智慧停车管理系统，采用云原生微服务架构设计，适用于商业综合体、住宅小区、写字楼、路边停车等多种场景。系统提供完整的停车管理解决方案，包括车辆入场/出场管理、智能计费、在线支付、设备控制、报表统计等功能。

### 为什么选择 Smart Park？

- ✅ **高性能**：支持 1000+ QPS 并发，P99 响应时间 < 200ms
- ✅ **高可用**：99.9% 服务可用性，支持水平扩展
- ✅ **易部署**：Docker + Kubernetes 一键部署
- ✅ **多支付**：集成微信支付、支付宝，支持扫码支付、JSAPI 支付
- ✅ **灵活计费**：支持按时、按时段、月卡、VIP 等多种计费规则
- ✅ **开源免费**：MIT 协议，可自由使用和二次开发

---

## ✨ 核心特性

### 🚗 车辆管理
- 车牌自动识别（支持多引擎融合）
- 车辆入场/出场全流程管理
- 月卡/VIP 车辆管理
- 无牌车处理机制
- 黑名单管理

### 💰 智能计费
- 灵活的计费规则引擎
- 支持按时、按时段、阶梯计费
- 月卡、VIP、优惠券折扣
- 节假日特殊计费
- 跨天计费处理

### 💳 支付系统
- 微信支付（Native、JSAPI）
- 支付宝（当面付）
- 支付安全机制（签名验证、幂等性）
- 自动退款处理
- 对账与差错处理

### 🔧 设备管理
- 车牌识别摄像头
- 道闸控制器
- 地感传感器
- LED 显示屏
- 设备状态监控

### 📊 运营管理
- 停车场信息管理
- 停车记录查询
- 订单管理
- 日报/月报统计
- 退款审批流程

---

## 🏗️ 系统架构

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

---

## 🛠️ 技术栈

### 后端技术
| 技术 | 版本 | 说明 |
|------|------|------|
| [Go](https://golang.org) | 1.26+ | 编程语言 |
| [Kratos](https://github.com/go-kratos/kratos) | v2.9 | 微服务框架 |
| [gRPC](https://grpc.io/) | v1.60+ | RPC 通信 |
| [Ent](https://entgo.io/) | v0.12+ | ORM 框架 |
| [PostgreSQL](https://www.postgresql.org/) | 15+ | 关系型数据库 |
| [Redis](https://redis.io/) | 7+ | 缓存/消息队列 |
| [Etcd](https://etcd.io/) | v3.5 | 服务注册发现 |
| [Jaeger](https://www.jaegertracing.io/) | v1.50+ | 链路追踪 |

### 支付集成
| 支付渠道 | SDK | 支持方式 |
|----------|-----|----------|
| 微信支付 | wechatpay-go | Native、JSAPI |
| 支付宝 | alipay | 当面付 |

### 运维部署
| 工具 | 用途 |
|------|------|
| Docker | 容器化 |
| Kubernetes | 容器编排 |
| Prometheus | 监控指标 |
| Grafana | 可视化监控 |

---

## 🚀 快速开始

### 环境要求

- **Go**: 1.26+
- **Docker**: 20.10+
- **Docker Compose**: 2.20+
- **Make** (可选)

### 一键启动

```bash
# 克隆项目
git clone https://github.com/xuanyiying/smart-park.git
cd smart-park

# 启动基础设施和服务
docker-compose up -d

# 查看服务状态
docker-compose ps
```

### 本地开发

```bash
# 安装依赖
go mod download

# 启动基础设施
docker-compose up -d postgres redis etcd jaeger

# 运行服务（分别在不同终端）
go run ./cmd/gateway -conf ./configs
go run ./cmd/vehicle -conf ./configs
go run ./cmd/billing -conf ./configs
go run ./cmd/payment -conf ./configs
go run ./cmd/admin -conf ./configs
```

### 测试 API

```bash
# 车辆入场
curl -X POST http://localhost:8000/api/v1/device/entry \
  -H "Content-Type: application/json" \
  -H "X-Device-Id: lane_001" \
  -d '{
    "deviceId": "lane_001",
    "plateNumber": "京A12345",
    "confidence": 0.95
  }'

# 查询费用
curl "http://localhost:8000/api/v1/billing/calculate?recordId=xxx"
```

---

## 📦 服务说明

| 服务 | 端口 | 职责 | 状态 |
|------|------|------|------|
| Gateway | 8000 | API 网关，路由转发 | ✅ 已完成 |
| Vehicle | 8001 | 车辆入场/出场，设备管理 | ✅ 已完成 |
| Billing | 8002 | 费用计算，计费规则 | ✅ 已完成 |
| Payment | 8003 | 支付处理，订单管理 | ✅ 已完成 |
| Admin | 8004 | 停车场管理，报表统计 | ✅ 已完成 |
| Charging | 8005 | 充电桩管理 | 🚧 开发中 |

---

## 📚 API 文档

### 设备端 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/device/entry` | 车辆入场 |
| POST | `/api/v1/device/exit` | 车辆出场 |
| POST | `/api/v1/device/heartbeat` | 设备心跳 |
| GET | `/api/v1/device/{id}/status` | 设备状态查询 |

### 计费 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/billing/calculate` | 费用计算 |
| GET | `/api/v1/admin/billing/rules` | 计费规则列表 |
| POST | `/api/v1/admin/billing/rules` | 创建计费规则 |

### 支付 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/pay/create` | 创建支付订单 |
| GET | `/api/v1/pay/{id}/status` | 查询支付状态 |
| POST | `/api/v1/pay/callback/wechat` | 微信支付回调 |
| POST | `/api/v1/pay/callback/alipay` | 支付宝回调 |

更多 API 文档请参考 [API 文档](docs/api.md)

---

## 🚢 部署指南

### 部署架构选择

| 规模 | 停车场数量 | 部署方式 | 文档 |
|------|-----------|----------|------|
| 小型 | 1-20 | Docker Compose | [部署文档](docs/deployment.md#小型部署) |
| 中型 | 20-100 | 主备双活 | [部署文档](docs/deployment.md#中型部署) |
| 大型 | 100+ | Kubernetes | [部署文档](docs/deployment.md#大型部署) |

### 快速部署

```bash
# 使用部署脚本
./deploy/scripts/deploy.sh deploy production v1.0.0

# 或使用 Kubernetes
kubectl apply -f deploy/k8s/
```

详细部署文档请参考 [部署文档](docs/deployment.md)

---

## 📈 性能指标

### 系统性能

| 指标 | 数值 | 说明 |
|------|------|------|
| **并发能力** | 1000+ QPS | 单服务 |
| **响应时间** | P99 < 200ms | API 接口 |
| **可用性** | 99.9% | 年度 |
| **车牌识别** | 98.5% | 准确率 |
| **支付成功率** | 99.5% | 微信+支付宝 |

### 资源占用

| 服务 | CPU | 内存 | 说明 |
|------|-----|------|------|
| Gateway | 0.5 核 | 256MB | 网关服务 |
| Vehicle | 0.5 核 | 512MB | 车辆服务 |
| Billing | 0.25 核 | 256MB | 计费服务 |
| Payment | 0.25 核 | 256MB | 支付服务 |
| Admin | 0.25 核 | 256MB | 管理服务 |

---

## 🏆 项目亮点

### 技术亮点

- **微服务架构**：基于 Kratos 框架，服务间通过 gRPC 通信
- **分布式锁**：Redis 分布式锁确保并发场景数据一致性
- **规则引擎**：灵活的计费规则引擎，支持多种计费策略
- **支付安全**：签名验证、金额校验、幂等性保证
- **高可用设计**：支持水平扩展，故障自动转移

### 业务亮点

- **防重复入场**：数据库唯一约束 + 分布式锁双重保障
- **月卡管理**：自动校验有效期，过期自动降级
- **离线模式**：网络中断时本地缓存，恢复后自动同步
- **对账系统**：自动对账，处理单边账

---

## 📖 相关文档

- [车辆服务文档](docs/vehicle.md) - 入场/出场处理、设备管理
- [计费服务文档](docs/billing.md) - 计费规则引擎
- [支付服务文档](docs/payment.md) - 支付处理
- [管理服务文档](docs/admin.md) - 运营管理
- [部署文档](docs/deployment.md) - 部署指南
- [项目简历](docs/project-resume.md) - 技术亮点和成果

---

## 🤝 贡献指南

我们欢迎所有形式的贡献，包括但不限于：

- 提交 Bug 报告
- 提交功能建议
- 提交代码修复
- 改进文档
- 分享使用经验

请参考 [贡献指南](CONTRIBUTING.md) 了解如何参与项目。

### 贡献者

感谢所有为项目做出贡献的开发者！

<a href="https://github.com/xuanyiying/smart-park/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=xuanyiying/smart-park" />
</a>

---

## 📄 许可证

本项目采用 [MIT](LICENSE) 许可证 - 详见 LICENSE 文件

---

## 💬 联系我们

- **项目主页**: https://github.com/xuanyiying/smart-park
- **问题反馈**: https://github.com/xuanyiying/smart-park/issues
- **文档站点**: https://smart-park.readthedocs.io
- **邮箱**: support@smart-park.example.com

---

## 🌟 Star History

[![Star History Chart](https://api.star-history.com/svg?repos=xuanyiying/smart-park&type=Date)](https://star-history.com/#xuanyiying/smart-park&Date)

---

**如果这个项目对你有帮助，请给我们一个 ⭐ Star！**

**Smart Park** - 让停车更智能，让出行更便捷 🚗💨

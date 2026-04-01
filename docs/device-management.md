# 设备管理平台技术文档

## 1. 概述

设备管理平台是一个用于管理和监控智能停车场设备的系统，提供设备监控、远程升级、故障诊断和多协议接入等功能。

### 1.1 功能特性

- **设备监控**：实时监控设备状态、在线状态、性能指标
- **远程升级**：支持固件版本管理和远程升级部署
- **故障诊断**：智能故障分析和告警管理
- **多协议接入**：支持HTTP、MQTT等多种设备协议

### 1.2 技术架构

设备管理平台采用微服务架构，基于Go语言和Kratos框架开发，主要组件包括：

- **API层**：提供RESTful API和gRPC接口
- **服务层**：实现业务逻辑
- **数据层**：存储设备信息、固件版本、告警等数据
- **协议适配层**：支持多种设备协议接入

## 2. 系统架构

### 2.1 核心组件

| 组件 | 职责 | 位置 |
|------|------|------|
| 设备API | 提供设备管理相关的API接口 | api/device/v1/ |
| 设备服务 | 实现设备管理的业务逻辑 | internal/device/service/ |
| 设备业务逻辑 | 实现设备管理的核心业务逻辑 | internal/device/biz/ |
| 设备数据访问 | 实现设备数据的存储和检索 | internal/device/data/ |
| 协议适配器 | 实现多种设备协议的接入 | internal/device/proto/ |
| 设备服务入口 | 设备服务的启动入口 | cmd/device/ |

### 2.2 数据流

1. 设备通过HTTP或MQTT协议发送心跳和数据
2. 设备服务接收并处理设备数据
3. 设备服务将数据存储到数据库
4. 前端通过API查询设备状态和历史数据
5. 管理员通过API执行远程操作（如升级固件、发送命令）

## 3. API文档

### 3.1 设备管理API

| API路径 | 方法 | 功能 |
|---------|------|------|
| /api/v1/devices | GET | 列出所有设备 |
| /api/v1/devices/{deviceId} | GET | 获取设备详情 |
| /api/v1/devices | POST | 创建设备 |
| /api/v1/devices/{deviceId} | PUT | 更新设备信息 |
| /api/v1/devices/{deviceId} | DELETE | 删除设备 |

### 3.2 设备监控API

| API路径 | 方法 | 功能 |
|---------|------|------|
| /api/v1/devices/{deviceId}/status | GET | 获取设备状态 |
| /api/v1/devices/status | GET | 列出设备状态 |
| /api/v1/devices/heartbeat | POST | 设备心跳 |

### 3.3 固件管理API

| API路径 | 方法 | 功能 |
|---------|------|------|
| /api/v1/firmware/versions | GET | 列出固件版本 |
| /api/v1/firmware/versions | POST | 创建固件版本 |
| /api/v1/devices/{deviceId}/firmware/update | POST | 更新设备固件 |
| /api/v1/devices/{deviceId}/firmware/status | GET | 获取固件更新状态 |

### 3.4 告警管理API

| API路径 | 方法 | 功能 |
|---------|------|------|
| /api/v1/devices/alerts | GET | 列出设备告警 |
| /api/v1/devices/alerts/{alertId} | GET | 获取告警详情 |
| /api/v1/devices/alerts/{alertId}/acknowledge | POST | 确认告警 |

### 3.5 设备命令API

| API路径 | 方法 | 功能 |
|---------|------|------|
| /api/v1/devices/{deviceId}/command | POST | 发送命令到设备 |

## 4. 数据库模型

### 4.1 设备表 (device)

| 字段 | 类型 | 描述 |
|------|------|------|
| device_id | string | 设备ID |
| device_type | string | 设备类型 |
| status | string | 设备状态 |
| online | bool | 在线状态 |
| last_heartbeat | timestamp | 最后心跳时间 |
| lot_id | string | 停车场ID |
| lane_id | string | 车道ID |
| protocol | string | 通信协议 |
| config | json | 设备配置 |
| firmware_version | string | 固件版本 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

### 4.2 固件版本表 (firmware_version)

| 字段 | 类型 | 描述 |
|------|------|------|
| id | string | 固件ID |
| device_type | string | 设备类型 |
| version | string | 固件版本 |
| url | string | 固件下载地址 |
| checksum | string | 固件校验和 |
| description | string | 固件描述 |
| is_active | bool | 是否激活 |
| created_at | timestamp | 创建时间 |

### 4.3 固件更新表 (firmware_update)

| 字段 | 类型 | 描述 |
|------|------|------|
| update_id | string | 更新ID |
| device_id | string | 设备ID |
| firmware_version | string | 固件版本 |
| status | string | 更新状态 |
| progress | string | 更新进度 |
| error_message | string | 错误信息 |
| started_at | timestamp | 开始时间 |
| completed_at | timestamp | 完成时间 |

### 4.4 告警表 (device_alert)

| 字段 | 类型 | 描述 |
|------|------|------|
| alert_id | string | 告警ID |
| device_id | string | 设备ID |
| type | string | 告警类型 |
| severity | string | 告警级别 |
| message | string | 告警消息 |
| status | string | 告警状态 |
| created_at | timestamp | 创建时间 |
| acknowledged_at | timestamp | 确认时间 |
| acknowledged_by | string | 确认人 |
| notes | string | 备注 |
| metadata | json | 元数据 |

### 4.5 命令表 (command)

| 字段 | 类型 | 描述 |
|------|------|------|
| command_id | string | 命令ID |
| device_id | string | 设备ID |
| command | string | 命令内容 |
| params | json | 命令参数 |
| status | string | 命令状态 |
| created_at | timestamp | 创建时间 |
| executed_at | timestamp | 执行时间 |
| result | string | 执行结果 |

### 4.6 设备指标表 (device_metric)

| 字段 | 类型 | 描述 |
|------|------|------|
| id | string | 指标ID |
| device_id | string | 设备ID |
| metric | string | 指标名称 |
| value | string | 指标值 |
| unit | string | 指标单位 |
| timestamp | timestamp | 时间戳 |

### 4.7 设备状态历史表 (device_status_history)

| 字段 | 类型 | 描述 |
|------|------|------|
| id | string | 记录ID |
| device_id | string | 设备ID |
| status | string | 状态 |
| online | bool | 在线状态 |
| firmware_version | string | 固件版本 |
| timestamp | timestamp | 时间戳 |
| metadata | json | 元数据 |

## 5. 协议适配器

### 5.1 HTTP协议适配器

HTTP协议适配器支持通过HTTP请求与设备通信，主要功能包括：

- 设备注册和认证
- 命令下发
- 数据采集
- 状态监控

### 5.2 MQTT协议适配器

MQTT协议适配器支持通过MQTT消息与设备通信，主要功能包括：

- 设备连接管理
- 消息发布和订阅
- 命令下发
- 数据采集

### 5.3 协议注册

设备管理平台支持动态注册新的协议适配器，通过ProtocolManager统一管理。

## 6. 前端界面

### 6.1 设备列表

显示所有设备的基本信息，包括设备名称、类型、状态、位置等。

### 6.2 设备监控

实时显示设备的运行状态、性能指标、网络状态等。

### 6.3 远程升级

管理固件版本，执行设备固件升级，查看升级进度和历史。

### 6.4 故障诊断

显示设备的健康状态、最近告警、预测分析等。

## 7. 部署与集成

### 7.1 依赖项

- Go 1.18+
- Kratos 2.0+
- PostgreSQL
- Redis (可选，用于缓存)
- MQTT Broker (可选，用于MQTT协议)

### 7.2 配置文件

设备服务的配置文件位于 `configs/device.yaml`，主要配置项包括：

- 服务端口
- 数据库连接
- MQTT broker配置
- 日志配置

### 7.3 启动命令

```bash
# 启动设备服务
go run cmd/device/main.go -conf configs/device.yaml
```

## 8. 开发指南

### 8.1 代码结构

设备管理平台的代码结构遵循Kratos框架的标准结构：

- `api/` - API定义
- `internal/` - 内部实现
  - `biz/` - 业务逻辑
  - `data/` - 数据访问
  - `service/` - 服务层
  - `proto/` - 协议适配器
- `cmd/` - 命令行工具

### 8.2 新增设备协议

要新增设备协议，需要：

1. 实现 `ProtocolAdapter` 接口
2. 在 `proto/init.go` 中注册适配器
3. 配置设备使用新协议

### 8.3 测试

设备管理平台的测试包括：

- 单元测试：测试核心业务逻辑
- 集成测试：测试服务间交互
- E2E测试：测试完整流程

## 9. 监控与维护

### 9.1 日志

设备服务的日志包括：

- 访问日志：记录API访问
- 业务日志：记录业务操作
- 错误日志：记录错误信息

### 9.2 监控指标

设备管理平台的监控指标包括：

- 设备在线率
- API响应时间
- 固件升级成功率
- 告警数量

### 9.3 常见问题

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| 设备离线 | 网络问题或设备故障 | 检查网络连接和设备状态 |
| 升级失败 | 固件版本不兼容 | 检查固件版本和设备兼容性 |
| 告警频繁 | 设备配置错误 | 检查设备配置和阈值设置 |

## 10. 总结

设备管理平台是一个功能完善的设备管理系统，支持设备监控、远程升级、故障诊断和多协议接入等功能。通过采用微服务架构和模块化设计，系统具有良好的可扩展性和可维护性。

未来可以考虑的功能扩展：

- 支持更多设备协议
- 实现设备自动发现
- 增强故障预测能力
- 提供更多数据分析和可视化功能

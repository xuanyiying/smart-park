# 硬件厂商支持文档

## 概述

Smart Park 停车管理系统支持多种硬件厂商的设备，通过厂商适配器框架实现统一的设备管理和监控。本文档详细说明系统支持的硬件厂商、设备类型、配置方法以及集成流程。

## 支持的硬件厂商

### 1. 海康威视（Hikvision）

**设备类型**：
- 车牌识别相机
- 道闸控制设备
- 视频监控设备

**配置参数**：
- `ip`：设备IP地址
- `port`：设备端口，默认80
- `username`：设备登录用户名
- `password`：设备登录密码
- `channel`：相机通道号

**适配器功能**：
- 开闸/关闸控制
- 设备状态查询
- 自定义命令发送

### 2. 大华（Dahua）

**设备类型**：
- 车牌识别相机
- 道闸控制设备
- 停车场管理设备

**配置参数**：
- `ip`：设备IP地址
- `port`：设备端口，默认80
- `username`：设备登录用户名
- `password`：设备登录密码
- `channel`：相机通道号

**适配器功能**：
- 开闸/关闸控制
- 设备状态查询
- 自定义命令发送

### 3. 捷顺（Jieshun）

**设备类型**：
- 道闸控制设备
- 停车场收费设备
- 门禁控制设备

**配置参数**：
- `ip`：设备IP地址
- `port`：设备端口，默认80
- `username`：设备登录用户名
- `password`：设备登录密码
- `device_id`：设备ID

**适配器功能**：
- 开闸/关闸控制
- 设备状态查询
- 自定义命令发送

### 4. 科拓（Ketuo）

**设备类型**：
- 车牌识别相机
- 道闸控制设备
- 停车场管理系统

**配置参数**：
- `ip`：设备IP地址
- `port`：设备端口，默认80
- `username`：设备登录用户名
- `password`：设备登录密码
- `device_type`：设备类型

**适配器功能**：
- 开闸/关闸控制
- 设备状态查询
- 自定义命令发送

### 5. 蓝卡（Lanka）

**设备类型**：
- 道闸控制设备
- 停车场收费设备
- 门禁控制设备

**配置参数**：
- `ip`：设备IP地址
- `port`：设备端口，默认80
- `username`：设备登录用户名
- `password`：设备登录密码
- `serial_number`：设备序列号

**适配器功能**：
- 开闸/关闸控制
- 设备状态查询
- 自定义命令发送

## 厂商适配器框架

### 适配器接口

系统定义了统一的设备适配器接口，所有厂商适配器都实现此接口：

```go
type DeviceAdapter interface {
    // OpenGate opens the gate
    OpenGate(ctx context.Context, deviceID string) error

    // CloseGate closes the gate
    CloseGate(ctx context.Context, deviceID string) error

    // GetDeviceStatus gets the device status
    GetDeviceStatus(ctx context.Context, deviceID string) (map[string]interface{}, error)

    // SendCommand sends a custom command to the device
    SendCommand(ctx context.Context, deviceID string, command string, params map[string]interface{}) (map[string]interface{}, error)

    // GetManufacturer returns the manufacturer name
    GetManufacturer() string

    // GetModel returns the device model
    GetModel() string
}
```

### 适配器注册

厂商适配器通过 `AdapterFactory` 进行注册和管理：

```go
// 注册海康威视适配器
adapterFactory.Register("Hikvision", "DS-2CD2T45FWD-I8", NewHikvisionAdapter("DS-2CD2T45FWD-I8"))

// 注册大华适配器
adapterFactory.Register("Dahua", "DH-IPC-HFW4433M-I1", NewDahuaAdapter("DH-IPC-HFW4433M-I1"))

// 注册捷顺适配器
adapterFactory.Register("Jieshun", "JSKT-2001", NewJieshunAdapter("JSKT-2001"))

// 注册科拓适配器
adapterFactory.Register("Ketuo", "KT-CP01", NewKetuoAdapter("KT-CP01"))

// 注册蓝卡适配器
adapterFactory.Register("Lanka", "LK-DZ01", NewLankaAdapter("LK-DZ01"))
```

## 设备管理API

### 1. 创建设备

**API端点**：`POST /api/v1/devices`

**请求体**：

```json
{
  "deviceId": "device-001",
  "deviceType": "camera",
  "lotId": "lot-001",
  "laneId": "lane-001",
  "status": "active",
  "manufacturer": "Hikvision",
  "model": "DS-2CD2T45FWD-I8",
  "firmwareVersion": "v1.0.0",
  "vendorSpecificConfig": {
    "ip": "192.168.1.100",
    "port": "80",
    "username": "admin",
    "password": "123456",
    "channel": "1"
  }
}
```

### 2. 获取设备状态

**API端点**：`GET /api/v1/device/{deviceId}/status`

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "deviceId": "device-001",
    "online": true,
    "status": "active",
    "lastHeartbeat": "2024-01-01T00:00:00Z"
  }
}
```

### 3. 发送命令

**API端点**：`POST /api/v1/device/{deviceId}/command`

**请求体**：

```json
{
  "command": "open_gate",
  "params": {
    "duration": 5
  }
}
```

## 固件管理

### 1. 创建固件

**API端点**：`POST /api/v1/firmwares`

**请求体**：

```json
{
  "firmwareId": "fw-20240101-001",
  "manufacturer": "Hikvision",
  "model": "DS-2CD2T45FWD-I8",
  "version": "v2.0.0",
  "url": "https://example.com/firmware/hikvision/ds-2cd2t45fwd-i8-v2.0.0.bin",
  "size": 10485760,
  "md5": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "description": "Updated firmware with new features",
  "status": "published"
}
```

### 2. 获取最新固件

**API端点**：`GET /api/v1/firmwares/latest?manufacturer=Hikvision&model=DS-2CD2T45FWD-I8`

**响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": "123e4567-e89b-12d3-a456-426614174000",
    "firmwareId": "fw-20240101-001",
    "manufacturer": "Hikvision",
    "model": "DS-2CD2T45FWD-I8",
    "version": "v2.0.0",
    "url": "https://example.com/firmware/hikvision/ds-2cd2t45fwd-i8-v2.0.0.bin",
    "size": 10485760,
    "md5": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "description": "Updated firmware with new features",
    "status": "published",
    "releaseDate": "2024-01-01T00:00:00Z",
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-01T00:00:00Z"
  }
}
```

## 设备性能监控

系统支持设备性能监控，包括：

- **CPU使用率**：设备CPU使用情况
- **内存使用率**：设备内存使用情况
- **网络状态**：设备网络连接状态
- **存储空间**：设备存储使用情况
- **温度**：设备运行温度

## 故障诊断

系统支持设备故障诊断，包括：

- **自动诊断**：定期检查设备状态，发现异常自动告警
- **故障处理建议**：根据故障类型提供处理建议
- **历史故障记录**：记录设备历史故障信息

## 集成流程

### 1. 设备注册

1. 在系统中创建设备记录，填写设备基本信息
2. 配置厂商特定参数
3. 测试设备连接

### 2. 设备监控

1. 设备定期发送心跳
2. 系统监控设备状态
3. 异常情况自动告警

### 3. 固件升级

1. 上传新固件
2. 选择目标设备
3. 执行固件升级
4. 验证升级结果

## 最佳实践

### 1. 设备配置

- 使用固定IP地址
- 确保网络连接稳定
- 定期备份设备配置

### 2. 监控设置

- 设置合理的心跳间隔
- 配置适当的告警阈值
- 定期检查设备状态

### 3. 固件管理

- 定期更新设备固件
- 测试固件兼容性
- 备份设备配置

## 故障排查

### 常见问题

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| 设备离线 | 网络连接问题 | 检查网络连接，重启设备 |
| 开闸失败 | 权限问题 | 检查设备权限配置 |
| 识别率低 | 相机角度问题 | 调整相机角度，清洁镜头 |
| 固件升级失败 | 网络不稳定 | 确保网络稳定，重新升级 |

### 日志分析

系统会记录设备操作日志，可通过日志分析排查问题：

- 设备通信日志
- 命令执行日志
- 状态变更日志

## 总结

Smart Park 停车管理系统通过统一的厂商适配器框架，支持多种硬件厂商的设备，实现了设备的集中管理、监控和维护。系统提供了丰富的API接口，方便与其他系统集成，为停车场管理提供了可靠的技术支持。
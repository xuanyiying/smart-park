# Vehicle Service 车辆服务

## 模块概述

车辆服务是 Smart Park 系统的核心服务之一，负责处理车辆入场/出场全流程、设备管理、停车记录管理等核心业务。该服务与硬件设备直接交互，是系统与物理世界连接的桥梁。

## 核心功能

### 1. 车辆入场处理 (Entry)

#### 业务流程

```
车辆到达 → 地感检测 → 车牌识别 → 防重复入场检查 → 创建入场记录 → 抬杆放行
```

#### 关键技术实现

**防重复入场机制**

```go
// 入场前检查是否存在未出场记录
existingRecord, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
if existingRecord != nil {
    // 同一车辆重复入场：直接返回已有记录，不再创建新记录
    return &v1.EntryData{
        PlateNumber:    req.PlateNumber,
        Allowed:        false,
        GateOpen:       false,
        DisplayMessage: "车辆已在场内",
    }, nil
}
```

**技术决策说明**：
- 采用数据库唯一性约束 + 业务层双重检查
- 分布式锁确保并发场景下不会创建重复记录
- 支持无牌车入场（plate_number 可为空）

### 2. 车辆出场处理 (Exit)

#### 业务流程

```
车辆到达 → 车牌识别 → 查询入场记录 → 月卡校验 → 费用计算 → 支付/放行 → 开闸
```

#### 关键技术实现

**车牌匹配校验（防逃费）**

```go
// 出场时校验是否存在对应的入场记录
record, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
if record == nil {
    // 无入场记录：可能是逃费或识别错误
    return &v1.ExitData{
        PlateNumber:    req.PlateNumber,
        Allowed:        false,
        GateOpen:       false,
        DisplayMessage: "无入场记录，请人工处理",
    }, nil
}
```

**月卡有效期校验**

```go
// 月卡车辆出场时校验有效期
if vehicle.VehicleType == VehicleTypeMonthly {
    if vehicle.MonthlyValidUntil != nil && vehicle.MonthlyValidUntil.After(time.Now()) {
        finalAmount = 0  // 月卡有效，免费放行
    } else {
        // 月卡过期，标记为临时车收费
        record.Metadata["chargeAs"] = VehicleTypeTemporary
        record.Metadata["monthlyExpired"] = true
    }
}
```

### 3. 设备管理

#### 支持的设备类型

| 设备类型 | 功能 | 通信协议 |
|----------|------|----------|
| 车牌识别摄像头 | 车牌识别、图像抓拍 | HTTP/SDK |
| 道闸控制器 | 抬杆/落杆控制 | MQTT/继电器 |
| 地感传感器 | 车辆检测 | IO信号 |
| LED显示屏 | 信息显示 | HTTP/串口 |

#### 设备认证机制

采用 HMAC-SHA256 签名认证：

```
signature = HMAC-SHA256(deviceId + timestamp + bodyJson, deviceSecret)
Header: X-Device-Id, X-Timestamp, X-Signature
```

**技术决策说明**：
- 签名有效期 5 分钟，防止重放攻击
- 设备密钥线下分配，云端加密存储
- 支持设备临时禁用（维修场景）

### 4. 分布式锁实现

基于 Redis 的分布式锁，确保并发场景数据一致性：

```go
// 锁 Key 设计
lockKey := fmt.Sprintf("parking:v1:lock:%s:%s", lockType, identifier)

// 获取锁
acquired, err := uc.lockRepo.AcquireLock(ctx, lockKey, owner, ttl)

// 释放锁（使用 Lua 脚本保证原子性）
err := uc.lockRepo.ReleaseLock(ctx, lockKey, owner)
```

**应用场景**：
- 车辆入场：防止同一车牌并发入场创建多条记录
- 车辆出场：防止同一记录并发计费产生重复订单
- 设备控制：防止同一道闸并发接收冲突指令

## 数据模型

### 核心实体

```go
// ParkingRecord 停车记录
type ParkingRecord struct {
    ID                uuid.UUID
    LotID             uuid.UUID
    VehicleID         *uuid.UUID      // 关联车辆（无牌车为空）
    PlateNumber       *string         // 车牌号（支持无牌车）
    PlateNumberSource string          // 来源：camera/manual/offline
    EntryLaneID       uuid.UUID
    EntryTime         time.Time
    EntryImageURL     *string
    ExitLaneID        *uuid.UUID
    ExitTime          *time.Time
    ExitImageURL      *string
    ExitDeviceID      *string
    ParkingDuration   int             // 停车时长（秒）
    RecordStatus      string          // entry/exiting/exited/paid/refunded
    ExitStatus        string          // unpaid/paid/refunded/waived
    Metadata          map[string]interface{}
}

// Device 设备信息
type Device struct {
    ID           uuid.UUID
    DeviceID     string          // 设备唯一标识
    LotID        uuid.UUID
    DeviceType   string          // camera/gate/sensor/display
    DeviceSecret string          // HMAC密钥（加密存储）
    Enabled      bool            // 是否启用
    Status       string          // active/offline/disabled
    LastHeartbeat *time.Time
}

// Lane 车道信息
type Lane struct {
    ID          uuid.UUID
    LotID       uuid.UUID
    LaneCode    string
    Direction   string          // entry/exit
    DeviceID    *string
}
```

## 应用场景区景

### 场景一：商业综合体停车场

**业务特点**：
- 车流量大，高峰期并发高
- 多种车辆类型（临时车、月租车、员工车、VIP车）
- 复杂的计费规则（分时段、节假日、消费减免）

**解决方案**：
- 分布式锁确保高并发下数据一致性
- 灵活的计费规则引擎对接
- 支持多种优惠叠加

### 场景二：住宅小区停车场

**业务特点**：
- 月卡车辆占比高
- 需要访客预约功能
- 夜间停车需求大

**解决方案**：
- 月卡有效期自动校验
- 过期月卡自动降级为临时车
- 夜间优惠时段支持

### 场景三：无人值守停车场

**业务特点**：
- 完全自动化运行
- 网络可能不稳定
- 需要离线容错能力

**解决方案**：
- 离线模式本地缓存
- 网络恢复后自动同步
- 异常场景自动告警

## 技术挑战与解决方案

### 挑战一：高并发入场/出场

**问题描述**：
早晚高峰时段，单车道可能达到 300-500 车次/小时，系统需要处理高并发请求。

**解决方案**：

1. **数据库优化**
   ```sql
   -- 复合索引优化查询
   CREATE INDEX idx_parking_records_plate_entry ON parking_records(plate_number, entry_time DESC);
   CREATE INDEX idx_parking_records_lot_status ON parking_records(lot_id, record_status) WHERE record_status IN ('entry', 'exiting');
   ```

2. **缓存策略**
   - 热点车辆信息缓存（1小时）
   - 费率规则缓存（24小时）
   - 设备状态缓存（5分钟）

3. **连接池优化**
   - PostgreSQL 连接池：max_connections=100
   - Redis 连接池：max_connections=50

### 挑战二：车牌识别准确率

**问题描述**：
实际场景中，车牌识别受光照、角度、污损等因素影响，准确率可能下降至 90% 以下。

**解决方案**：

1. **多引擎融合识别**
   ```go
   // 并行调用本地识别和云端识别
   results, _ := errgroup.Execute(
       func() { return localRecognizer.Recognize(image) },
       func() { return cloudRecognizer.Recognize(image) },
   )
   
   // 置信度融合策略
   if localResult.Confidence >= 0.9 {
       return localResult  // 本地高置信度直接返回
   }
   if cloudResult.Confidence >= 0.8 {
       return cloudResult  // 云端更准确
   }
   ```

2. **人工审核兜底**
   - 低置信度（<0.8）触发人工确认
   - 管理后台提供快速修正入口

3. **无牌车处理**
   - 支持无牌车入场（记录为 NULL）
   - 出口通过入场时间匹配

### 挑战三：网络中断容错

**问题描述**：
停车场网络可能不稳定，需要保证断网情况下系统仍能正常运行。

**解决方案**：

1. **离线模式设计**
   ```go
   // 设备本地 SQLite 缓存
   type OfflineRecord struct {
       OfflineID   string
       RecordID    uuid.UUID
       DeviceID    string
       OpenTime    time.Time
       SyncStatus  string  // pending_sync/synced/sync_failed
   }
   ```

2. **数据同步机制**
   - 网络恢复后自动同步离线数据
   - 支持断点续传
   - 冲突检测与解决

3. **降级策略**
   - 离线时允许欠费放行
   - 记录欠费车辆，下次入场追缴

### 挑战四：并发计费一致性

**问题描述**：
同一车辆可能在多个出口同时触发计费，需要防止重复计费。

**解决方案**：

1. **分布式悲观锁**
   ```go
   lockKey := fmt.Sprintf("parking:v1:lock:exit:%s", recordID)
   acquired, _ := lockRepo.AcquireLock(ctx, lockKey, owner, 30*time.Second)
   if !acquired {
       return fmt.Errorf("billing in progress")
   }
   defer lockRepo.ReleaseLock(ctx, lockKey, owner)
   ```

2. **数据库乐观锁**
   ```sql
   UPDATE parking_records 
   SET record_status = 'exiting', payment_lock = payment_lock + 1
   WHERE id = ? AND payment_lock = ?
   ```

3. **幂等性设计**
   - 订单创建使用 record_id 作为唯一键
   - 支付回调使用 transaction_id 去重

## API 接口文档

### 入场接口

```http
POST /api/v1/device/entry
Content-Type: application/json
X-Device-Id: lane_001
X-Timestamp: 1710916800000
X-Signature: sha256=...

{
  "deviceId": "lane_001",
  "plateNumber": "京A12345",
  "plateImageUrl": "https://cdn.example.com/entry/xxx.jpg",
  "confidence": 0.95,
  "timestamp": "2026-03-20T10:00:00Z"
}
```

**响应**：
```json
{
  "code": 0,
  "data": {
    "recordId": "rec_xxx",
    "plateNumber": "京A12345",
    "allowed": true,
    "gateOpen": true,
    "displayMessage": "欢迎光临"
  }
}
```

### 出场接口

```http
POST /api/v1/device/exit
Content-Type: application/json
X-Device-Id: lane_002

{
  "deviceId": "lane_002",
  "plateNumber": "京A12345",
  "plateImageUrl": "https://cdn.example.com/exit/xxx.jpg",
  "confidence": 0.93,
  "timestamp": "2026-03-20T12:00:00Z"
}
```

**响应**：
```json
{
  "code": 0,
  "data": {
    "recordId": "rec_xxx",
    "plateNumber": "京A12345",
    "parkingDuration": 7200,
    "amount": 15.00,
    "discountAmount": 0,
    "finalAmount": 15.00,
    "allowed": true,
    "gateOpen": false,
    "displayMessage": "请缴费"
  }
}
```

## 配置说明

```yaml
# configs/vehicle.yaml
server:
  http:
    addr: 0.0.0.0:8001
  grpc:
    addr: 0.0.0.0:9001

data:
  database:
    driver: postgres
    source: postgres://postgres:postgres@localhost:5432/parking?sslmode=disable
  redis:
    addr: localhost:6379
    password: ""
    db: 0

mqtt:
  broker: tcp://localhost:1883
  client_id: vehicle_service
  username: ""
  password: ""

lock:
  ttl: 30s  # 分布式锁超时时间
```

## 测试策略

### 单元测试

```bash
# 运行车辆服务单元测试
go test ./internal/vehicle/... -v

# 运行入场场景测试
go test ./internal/vehicle/biz -run TestEntry -v

# 运行出场场景测试
go test ./internal/vehicle/biz -run TestExit -v
```

### 集成测试

```bash
# 启动测试环境
docker-compose -f deploy/docker-compose.test.yml up -d

# 运行集成测试
go test ./tests/e2e -tags=integration -v
```

### 压力测试

```bash
# 使用 wrk 进行压力测试
wrk -t12 -c400 -d30s -s entry_test.lua http://localhost:8000/api/v1/device/entry
```

## 监控指标

| 指标 | 类型 | 说明 |
|------|------|------|
| vehicle_entry_total | Counter | 入场总次数 |
| vehicle_exit_total | Counter | 出场总次数 |
| vehicle_entry_duration | Histogram | 入场处理耗时 |
| vehicle_exit_duration | Histogram | 出场处理耗时 |
| device_online_status | Gauge | 设备在线状态 |
| lock_acquire_failures | Counter | 锁获取失败次数 |

## 相关文档

- [计费服务文档](billing.md)
- [支付服务文档](payment.md)
- [部署文档](deployment.md)

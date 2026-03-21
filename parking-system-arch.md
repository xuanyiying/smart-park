# 停车场收费系统架构实现及演进方案

## 文档信息

| 项目 | 内容 |
|------|------|
| 版本 | v1.4 |
| 日期 | 2026-03-20 |
| 状态 | 规划中 |

---

## 一、行业背景与市场分析

### 1.1 市场规模

| 指标 | 数据 |
|------|------|
| 市场规模（2025） | 约 334 亿元 |
| 年复合增长率 | 约 20.65% |
| 长期市场规模 | 千亿级 |
| 中游服务商规模 | 200-300 亿元 |
| 下游停车场运营 | 千亿级以上 |

### 1.2 竞争格局

```
行业公司数量估算：5000-10000 家

一线品牌：捷顺、海康威视、大华、科拓、ETCP
二线品牌：艾科智泊、立方、红门、启功、安居宝、道尔

市场特点：
- 高度分散，未形成垄断
- 地方保护主义严重
- SaaS 化趋势明显
- 价格战激烈
```

### 1.3 收费模式

| 模式 | 价格区间 | 适用场景 |
|------|----------|----------|
| 买断制 | ¥3-50万/套 | 大型停车场 |
| SaaS 年费 | ¥3000-30000/年/车道 | 中小型 |
| 硬件+软件 | 打包价 | 新建/改造 |

---

## 二、系统架构设计

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户层                                   │
├───────────────┬───────────────┬───────────────┬──────────────────┤
│   车主持卡人   │   物业管理员   │   运营方      │   系统管理员      │
│  （小程序）   │  （PC管理后台） │  （数据看板） │  （配置中心）     │
└───────────────┴───────────────┴───────────────┴──────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         服务层                                   │
├───────────────┬───────────────┬───────────────┬──────────────────┤
│  网关服务      │  识别服务      │  计费服务      │   支付服务        │
│  (Gateway)    │  (OCR)        │  (Billing)    │   (Payment)      │
├───────────────┼───────────────┼───────────────┼──────────────────┤
│  车辆服务      │  订单服务      │  通知服务      │   数据服务        │
│  (Vehicle)    │  (Order)      │  (Notify)     │   (Data)         │
└───────────────┴───────────────┴───────────────┴──────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         数据层                                   │
├───────────────┬───────────────┬───────────────┬──────────────────┤
│   PostgreSQL  │    Redis      │   消息队列     │   文件存储        │
│   (主数据)    │   (缓存/会话)  │  (异步任务)   │   (日志/图片)    │
└───────────────┴───────────────┴───────────────┴──────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                         设备层                                   │
├───────────────┬───────────────┬───────────────┬──────────────────┤
│  车牌识别摄像头 │    道闸控制    │   地感传感器  │   显示屏/缴费机  │
│   (HTTP/SDK)  │   (继电器)    │   (IO信号)   │    (LED/POST)   │
└───────────────┴───────────────┴───────────────┴──────────────────┘
```

### 2.2 技术架构（小型起步）

```
                        ┌─────────────────┐
                        │   用户端小程序   │
                        │  （车主缴费）    │
                        └────────┬────────┘
                                 │
                        ┌────────▼────────┐
                        │   云服务器       │
                        │  (2核4G足够)    │
                        │  Go + Kratos    │
                        └────────┬────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
┌───────▼───────┐    ┌───────▼───────┐    ┌───────▼───────┐
│   摄像头 SDK  │    │   闸机控制    │    │   支付网关    │
│  （厂商提供）  │    │   （继电器）  │    │  微信/支付宝  │
└───────────────┘    └───────────────┘    └───────────────┘
```

### 2.3 核心模块设计

#### 2.3.1 车辆入场模块

```
┌──────────────────────────────────────────────────────────────────────┐
│                      车辆入场流程                                    │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   车辆到达                                                            │
│      │                                                               │
│      ▼                                                               │
│  ┌─────────┐    ┌─────────┐    ┌──────────────┐                      │
│  │ 检测车辆 │───▶│ 车牌识别 │───▶│ 检查未出场记录 │                      │
│  │ (地感)  │    │ (OCR)   │    │ (防双重入场)  │                      │
│  └─────────┘    └─────────┘    └──────┬───────┘                      │
│                                       │                              │
│                               ┌───────┴───────┐                      │
│                               │ 已有未出场记录？│                      │
│                               └───────┬───────┘                      │
│                            是         │         否                   │
│                      ┌────────────────┴────────────────┐             │
│                      ▼                                  ▼             │
│              ┌──────────────┐                 ┌─────────────┐        │
│              │ 返回已有记录  │                 │ 创建入场记录  │        │
│              │ 直接抬杆放行  │                 │ 更新车位     │        │
│              └──────────────┘                 └──────┬──────┘        │
│                                                      │               │
│                                                      ▼               │
│                                              ┌─────────────┐        │
│                                              │ 保存图片    │        │
│                                              │ 抬杆放行    │        │
│                                              │ 更新车位    │        │
│                                              └─────────────┘        │
└──────────────────────────────────────────────────────────────────────┘
```

入场防重逻辑（代码实现）：
```go
// HandleEntry 处理车辆入场
func (s *VehicleService) HandleEntry(ctx context.Context, req *EntryRequest) (*EntryResult, error) {
	// 1. 检查该车牌是否有未出场记录（record_status IN ('entry', 'exiting')）
	existing, err := s.recordRepo.FindActiveByPlate(ctx, req.LotID, req.PlateNumber)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		// 同一车辆重复入场：直接返回已有记录，不再创建新记录
		return &EntryResult{
			RecordID:       existing.ID.String(),
			PlateNumber:    req.PlateNumber,
			Allowed:        true,
			GateOpen:       true,
			DisplayMessage: "欢迎光临",
			IsDuplicate:    true,
		}, nil
	}

	// 2. 确定车牌来源
	plateSource := "manual"
	if req.Confidence >= 0.7 {
		plateSource = "camera"
	}

	// 3. 创建新入场记录
	record, err := s.recordRepo.Create(ctx, &ParkingRecord{
		LotID:            req.LotID,
		EntryLaneID:      req.DeviceID,
		PlateNumber:      &req.PlateNumber,
		PlateNumberSource: &plateSource,
		EntryTime:        time.Now(),
		EntryImageURL:    &req.PlateImageURL,
		RecordStatus:     "entry",
	})
	if err != nil {
		return nil, err
	}

	return &EntryResult{
		RecordID:       record.ID.String(),
		PlateNumber:    req.PlateNumber,
		Allowed:        true,
		GateOpen:       true,
		DisplayMessage: "欢迎光临",
		IsDuplicate:    false,
	}, nil
}
```

#### 2.3.2 车辆出场模块

```
┌──────────────────────────────────────────────────────────────────────┐
│                      车辆出场流程                                    │
├──────────────────────────────────────────────────────────────────────┤
│                                                                      │
│   车辆到达                                                            │
│      │                                                               │
│      ▼                                                               │
│  ┌─────────┐    ┌─────────┐    ┌──────────────┐                      │
│  │ 检测车辆 │───▶│ 车牌识别 │───▶│ 查询入场记录  │                      │
│  │ (地感)  │    │ (OCR)   │    │ (防逃费验证)  │                      │
│  └─────────┘    └─────────┘    └──────┬───────┘                      │
│                                       │                              │
│                               ┌───────┴───────┐                      │
│                               │ 车牌匹配？    │                      │
│                               └───────┬───────┘                      │
│                            否         │         是                   │
│                      ┌────────────────┴────────────────┐             │
│                      ▼                                  ▼             │
│              ┌──────────────┐                 ┌─────────────┐        │
│              │ 人工核验车牌  │                 │ 月卡有效期   │        │
│              │ (非逃费)     │                 │ 校验        │        │
│              └──────┬───────┘                 └──────┬──────┘        │
│                     │                                │               │
│                     │ 匹配                           │               │
│                     └──────────────┬─────────────────┘               │
│                                    ▼                                 │
│                             ┌─────────────┐                          │
│                             │ 计算费用     │                          │
│                             │ (计费引擎)   │                          │
│                             └──────┬──────┘                          │
│                                    ▼                                 │
│                             ┌─────────────┐                          │
│                             │ 优惠折扣     │                          │
│                             │ (如果有)    │                          │
│                             └──────┬──────┘                          │
│                                    ▼                                 │
│                             ┌─────────────┐                          │
│                             │ 生成订单     │                          │
│                             └──────┬──────┘                          │
│                                    │                                │
│                    ┌───────────────┼───────────────┐                │
│                    ▼               ▼               ▼                │
│             ┌──────────┐   ┌──────────┐   ┌──────────┐              │
│             │ 扫码支付   │   │ 月卡免费  │   │ 离线放行  │              │
│             │ (微信/支)  │   │ (核验)   │   │ (欠费)   │              │
│             └────┬─────┘   └────┬─────┘   └────┬─────┘              │
│                  │              │              │                    │
│                  │              │              │                    │
│                  │      支付成功 │              │                    │
│                  │              │              │                    │
│                  └──────────────┴──────────────┘                    │
│                                │                                    │
│                                ▼                                    │
│                         ┌──────────────┐                            │
│                         │ 开闸放行      │                            │
│                         │ 更新状态      │                            │
│                         └──────┬───────┘                            │
│                                │                                    │
│                                ▼                                    │
│                         ┌──────────────┐                            │
│                         │ 车辆驶过地感  │                            │
│                         │ 落杆关闭闸机  │                            │
│                         └──────────────┘                            │
└──────────────────────────────────────────────────────────────────────┘
```

出场车牌匹配校验（代码实现）：
```go
// HandleExit 处理车辆出场
func (s *VehicleService) HandleExit(ctx context.Context, req *ExitRequest) (*ExitResult, error) {
	// 1. 查询入场记录（按车牌 + 时间范围，注意字段名是 record_status）
	records, err := s.recordRepo.FindByPlateAndLot(ctx, req.LotID, req.PlateNumber, []string{"entry", "exiting"})
	if err != nil {
		return nil, err
	}

	// 2. 车牌匹配校验（防逃费：A车入场、B车冒名出场）
	if len(records) == 0 {
		s.logException(ctx, "exit_no_match", map[string]interface{}{
			"plateNumber": req.PlateNumber,
			"deviceId":    req.DeviceID,
			"imageUrl":    req.PlateImageURL,
		})

		if req.Confidence < 0.8 {
			return &ExitResult{
				Allowed:        false,
				Reason:         "low_confidence_manual_verify",
				DisplayMessage: "请确认车牌后手动操作",
			}, nil
		}

		return &ExitResult{
			Allowed:        true,
			GateOpen:       true,
			Reason:         "no_match_offline_exit",
			OfflineAmount:  s.calculateDefaultFee(req),
		}, nil
	}

	// 3. 取最近的入场记录
	entryRecord := records[0]

	// 4. 更新出场信息（设置 record_status = 'exiting'）
	now := time.Now()
	entryRecord.RecordStatus = "exiting"
	entryRecord.ExitTime = &now
	entryRecord.ExitImageURL = &req.PlateImageURL
	entryRecord.ExitLaneID = &req.DeviceID
	entryRecord.ExitDeviceID = &req.DeviceID
	if err := s.recordRepo.Update(ctx, entryRecord); err != nil {
		return nil, err
	}

	// 5. 月卡有效期校验
	// 注意：无牌车（plate_number 为空）无法关联月卡，直接跳过校验
	// 月卡查询用 vehicle_id 而非 plate_number，避免无牌月卡车无法识别
	if req.PlateNumber != "" && entryRecord.VehicleID != nil {
		validation, err := s.validateMonthlyCard(ctx, *entryRecord.VehicleID, req.PlateNumber, now)
		if err == nil && validation.Valid {
			// 月卡有效：触发开闸并返回
			entryRecord.RecordStatus = "paid"
			entryRecord.ExitStatus = "paid"
			s.recordRepo.Update(ctx, entryRecord)
			s.openGate(ctx, req.DeviceID, entryRecord.ID.String())
			return &ExitResult{
				RecordID:       entryRecord.ID.String(),
				Allowed:        true,
				GateOpen:       true,
				Amount:         0,
				DisplayMessage: "月卡车辆，一路平安",
			}, nil
		}

		// 月卡过期：修改 record 中的标记（用于计费引擎识别）
		if validation != nil && validation.Reason == "monthly_expired" {
			entryRecord.RecordMetadata = map[string]interface{}{"chargeAs": "temporary"}
			s.recordRepo.Update(ctx, entryRecord)
		}
	}

	// 6. 计算费用（传入 record，计费引擎会自动读取 record_metadata 判断是否过期月卡）
	bill, err := s.calculateFeeWithLock(ctx, entryRecord.ID.String())
	if err != nil {
		// 计费异常：允许离线放行
		return &ExitResult{
			RecordID:      entryRecord.ID.String(),
			Allowed:       true,
			GateOpen:      true,
			Reason:        "billing_error_offline",
			OfflineAmount: s.calculateDefaultFee(req),
		}, nil
	}

	// 7. 月卡过期但已在计费中处理，返回待支付
	return &ExitResult{
		RecordID:       entryRecord.ID.String(),
		Allowed:        true,
		GateOpen:       false, // 等待支付后开闸
		Amount:         bill.FinalAmount,
		DiscountAmount: bill.DiscountAmount,
		ExitDeviceID:   req.DeviceID,
	}, nil
}
```

#### 2.3.3 计费引擎设计

```go
// BillingRule 计费规则配置
type BillingRule struct {
    ID         string
    Name       string
    Type       string // "time", "period", "monthly", "coupon", "vip"
    Conditions []Condition
    Actions    []Action
    Priority   int
    IsActive   bool
}

// Condition 条件
type Condition struct {
    Field    string
    Operator string
    Value    interface{}
}

// Action 动作
type Action struct {
    Type       string
    Amount     float64
    Unit       string
    Percentage float64
    Value      int // free_duration 使用
}

// 示例规则
var rules = []BillingRule{
    {
        ID:   "rule_001",
        Name: "临时车标准收费",
        Type: "time",
        Conditions: []Condition{
            {Field: "vehicle_type", Operator: "==", Value: "temporary"},
        },
        Actions: []Action{
            {Type: "base_rate", Amount: 5, Unit: "first_hour"},
            {Type: "hourly_rate", Amount: 3, Unit: "hour"},
            {Type: "max_daily", Amount: 50},
        },
    },
    {
        ID:   "rule_002",
        Name: "月卡车辆",
        Type: "monthly",
        Conditions: []Condition{
            {Field: "vehicle_type", Operator: "==", Value: "monthly"},
        },
        Actions: []Action{
            {Type: "fixed_rate", Amount: 300, Unit: "month"},
            {Type: "free_duration", Value: 86400}, // 每月包含86400秒（约1天）免费停车
        },
    },
    {
        ID:   "rule_003",
        Name: "夜间优惠",
        Type: "period",
        Conditions: []Condition{
            {Field: "exit_time", Operator: "between", Value: []string{"22:00", "08:00"}},
        },
        Actions: []Action{
            {Type: "discount", Percentage: 50},
        },
    },
}
```

### 2.4 数据库设计

```sql
-- 停车场表
CREATE TABLE parking_lots (
  id UUID PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  address VARCHAR(255),
  lanes INTEGER NOT NULL DEFAULT 1,
  status VARCHAR(20) DEFAULT 'active',
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- 车道表
CREATE TABLE lanes (
  id UUID PRIMARY KEY,
  lot_id UUID REFERENCES parking_lots(id),
  lane_no INTEGER NOT NULL,
  direction VARCHAR(10) NOT NULL, -- 'entry' or 'exit'
  status VARCHAR(20) DEFAULT 'active',
  device_config JSONB
);

-- 车辆表
CREATE TABLE vehicles (
  id UUID PRIMARY KEY,
  plate_number VARCHAR(20) NOT NULL UNIQUE,
  vehicle_type VARCHAR(20) DEFAULT 'temporary', -- 'temporary', 'monthly', 'vip'
  owner_name VARCHAR(100),
  owner_phone VARCHAR(20),
  monthly_valid_until DATE,
  created_at TIMESTAMP DEFAULT NOW()
);

-- 入场记录表
CREATE TABLE parking_records (
  id UUID PRIMARY KEY,
  lot_id UUID REFERENCES parking_lots(id),
  lane_id UUID REFERENCES lanes(id),
  vehicle_id UUID REFERENCES vehicles(id),
  plate_number VARCHAR(20),          -- 可为空，支持无牌车
  plate_number_source VARCHAR(10),   -- 'camera' | 'manual' | 'offline'
  entry_time TIMESTAMP NOT NULL,
  entry_image_url VARCHAR(255),
  record_status VARCHAR(20) DEFAULT 'entry',  -- 'entry' | 'exiting' | 'exited' | 'paid'
  exit_time TIMESTAMP,
  exit_image_url VARCHAR(255),
  exit_lane_id UUID,
  exit_device_id VARCHAR(64),        -- 出场设备ID（用于支付后自动开闸）
  parking_duration INTEGER,          -- 秒
  exit_status VARCHAR(20) DEFAULT 'unpaid',  -- 'unpaid' | 'paid' | 'refunded' | 'waived'
  payment_lock INTEGER DEFAULT 0,   -- 并发锁版本号，乐观锁
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- 索引（解决出场按车牌查询全表扫描问题）
CREATE INDEX idx_parking_records_plate_entry ON parking_records(plate_number, entry_time DESC);
CREATE INDEX idx_parking_records_lot_status ON parking_records(lot_id, record_status) WHERE record_status IN ('entry', 'exiting');
CREATE INDEX idx_parking_records_exit ON parking_records(exit_time DESC) WHERE exit_time IS NOT NULL;

-- 订单表
CREATE TABLE orders (
  id UUID PRIMARY KEY,
  record_id UUID REFERENCES parking_records(id),
  lot_id UUID REFERENCES parking_lots(id),
  vehicle_id UUID REFERENCES vehicles(id),
  plate_number VARCHAR(20) NOT NULL,
  amount DECIMAL(10,2) NOT NULL,
  discount_amount DECIMAL(10,2) DEFAULT 0,
  final_amount DECIMAL(10,2) NOT NULL,
  status VARCHAR(20) DEFAULT 'pending', -- 'pending'|'paid'|'refunding'|'refunded'|'failed'
  pay_time TIMESTAMP,
  pay_method VARCHAR(20),               -- 'wechat' | 'alipay' | 'cash'
  transaction_id VARCHAR(64),           -- 支付渠道交易号
  paid_amount DECIMAL(10,2),            -- 实际支付金额（回调写入，防篡改）
  refunded_at TIMESTAMP,                -- 退款时间
  refund_transaction_id VARCHAR(64),    -- 退款渠道流水号
  created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_pay_time ON orders(pay_time DESC) WHERE pay_time IS NOT NULL;

-- 费率配置表
CREATE TABLE billing_rules (
  id UUID PRIMARY KEY,
  lot_id UUID REFERENCES parking_lots(id),
  rule_name VARCHAR(100) NOT NULL,
  rule_config JSONB NOT NULL,
  priority INTEGER DEFAULT 0,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMP DEFAULT NOW()
);

-- 离线放行记录表（网络恢复后同步）
CREATE TABLE offline_sync_records (
  id UUID PRIMARY KEY,
  offline_id VARCHAR(64) NOT NULL UNIQUE,  -- 设备本地流水号
  record_id UUID REFERENCES parking_records(id),
  lot_id UUID REFERENCES parking_lots(id),
  device_id VARCHAR(64) NOT NULL,
  gate_id VARCHAR(64) NOT NULL,
  open_time TIMESTAMP NOT NULL,
  sync_amount DECIMAL(10,2),
  sync_status VARCHAR(20) DEFAULT 'pending_sync', -- 'pending_sync'|'synced'|'sync_failed'
  sync_error TEXT,
  retry_count INTEGER DEFAULT 0,
  synced_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

-- 分布式锁表（用于出场计费等关键操作）
CREATE TABLE distributed_locks (
  lock_key VARCHAR(128) PRIMARY KEY,
  owner VARCHAR(64) NOT NULL,
  expire_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_locks_expire ON distributed_locks(expire_at);

-- 退款审批记录表（管理端异常退款用）
CREATE TABLE refund_approvals (
  id UUID PRIMARY KEY,
  order_id UUID REFERENCES orders(id),
  applicant VARCHAR(64) NOT NULL,           -- 申请人
  approver VARCHAR(64),                      -- 审批人
  amount DECIMAL(10,2) NOT NULL,
  reason TEXT NOT NULL,
  refund_method VARCHAR(20) DEFAULT 'original', -- 'original' 原路返回 | 'manual' 人工
  status VARCHAR(20) DEFAULT 'pending',    -- 'pending'|'approved'|'rejected'
  approved_at TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);

-- 设备注册表（存储 HMAC secret）
CREATE TABLE device_registry (
  id UUID PRIMARY KEY,
  device_id VARCHAR(64) NOT NULL UNIQUE,
  lot_id UUID REFERENCES parking_lots(id),
  device_secret VARCHAR(128) NOT NULL,      -- HMAC 密钥（加密存储）
  device_type VARCHAR(20),                  -- 'camera'|'gate'|'display'|'payment_kiosk'|'sensor'
  enabled BOOLEAN DEFAULT true,             -- 临时禁用（维修时可设 false）
  status VARCHAR(20) DEFAULT 'active',      -- 'active'|'offline'|'disabled'
  last_heartbeat TIMESTAMP,
  created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX idx_device_lot ON device_registry(lot_id);
```

### 2.5 API 接口设计

#### 2.5.1 接口规范

```
RESTful API 设计规范：

Base URL: /api/v1

认证方式：JWT Bearer Token
请求格式：Content-Type: application/json
响应格式：统一响应体

{
  "code": 0,        // 状态码：0成功，非0失败
  "message": "OK",   // 消息
  "data": {}        // 数据体
}

分页格式：
{
  "code": 0,
  "data": {
    "list": [],
    "total": 100,
    "page": 1,
    "pageSize": 20
  }
}
```

#### 2.5.2 设备对接 API

**设备认证方案：HMAC-SHA256 签名**

每个设备分配唯一 `deviceSecret`，请求时对参数生成签名：
```
signature = HMAC-SHA256(deviceId + timestamp + bodyJson, deviceSecret)
Header: X-Device-Id, X-Timestamp, X-Signature
```
签名有效期 5 分钟，防重放攻击。设备注册时通过线下分配 secret，云端存储 `deviceId -> deviceSecret` 映射。

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/device/entry | 入场事件上报 |
| POST | /api/v1/device/exit | 出场事件上报 |
| POST | /api/v1/device/heartbeat | 设备心跳 |
| GET | /api/v1/device/{deviceId}/status | 设备状态查询 |
| POST | /api/v1/device/{deviceId}/command | 设备控制指令 |

**入场事件上报**

```
POST /api/v1/device/entry
X-Device-Id: lane_001
X-Timestamp: 1710916800000
X-Signature: sha256=deviceId+timestamp+body 签名结果

请求：
{
  "deviceId": "lane_001",
  "plateNumber": "京A12345",
  "plateImageUrl": "https://cdn.xxx.com/entry/xxx.jpg",
  "confidence": 0.95,
  "vehicleType": "temporary",
  "timestamp": "2026-03-20T10:00:00Z"
}

响应：
{
  "code": 0,
  "message": "OK",
  "data": {
    "recordId": "rec_xxx",
    "plateNumber": "京A12345",
    "allowed": true,
    "gateOpen": true,
    "displayMessage": "欢迎光临"
  }
}
```

**出场事件上报**

```
POST /api/v1/device/exit
X-Device-Id: lane_002
X-Timestamp: 1710917800000
X-Signature: sha256=...

请求：
{
  "deviceId": "lane_002",
  "plateNumber": "京A12345",
  "plateImageUrl": "https://cdn.xxx.com/exit/xxx.jpg",
  "confidence": 0.93,
  "timestamp": "2026-03-20T12:00:00Z"
}

响应：
{
  "code": 0,
  "message": "OK",
  "data": {
    "recordId": "rec_xxx",
    "plateNumber": "京A12345",
    "parkingDuration": 7200,
    "amount": 15.00,
    "discountAmount": 0,
    "finalAmount": 15.00,
    "allowed": true,
    "gateOpen": true,
    "displayMessage": "一路平安"
  }
}
```

**设备控制指令**（服务端下发，云端 -> 设备）

```
POST /api/v1/device/{deviceId}/command
X-Device-Id: lane_001
X-Timestamp: 1710916800000
X-Signature: sha256=...

请求：
{
  "command": "open_gate" | "close_gate" | "voice_broadcast",
  "params": {
    "message": "请关注通道"
  }
}

响应：
{
  "code": 0,
  "message": "OK",
  "data": {
    "commandId": "cmd_xxx",
    "status": "success"
  }
}
```

#### 2.5.3 支付 API

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | /api/v1/pay/create | 创建支付订单 |
| POST | /api/v1/pay/callback/{method} | 支付回调（微信/支付宝服务器调用） |
| GET | /api/v1/pay/{orderId}/status | 支付状态查询（车主端） |
| POST | /api/v1/pay/{orderId}/refund | 车主退款（原路返回，自动处理） |

**创建支付订单**

```
POST /api/v1/pay/create

请求：
{
  "recordId": "rec_xxx",
  "amount": 15.00,
  "payMethod": "wechat" | "alipay",
  "openId": "oxxxxx"
}

响应：
{
  "code": 0,
  "message": "OK",
  "data": {
    "orderId": "ord_xxx",
    "amount": 15.00,
    "payUrl": "weixin://wxpay/xxx",  // 微信支付链接
    "qrCode": "https://qr.alipay.com/xxx",  // 支付宝二维码
    "expireTime": "2026-03-20T12:30:00Z"
  }
}
```

#### 2.5.4 管理后台 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/admin/lots | 停车场列表 |
| POST | /api/v1/admin/lots | 创建停车场 |
| GET | /api/v1/admin/lots/{id} | 停车场详情 |
| PUT | /api/v1/admin/lots/{id} | 更新停车场 |
| GET | /api/v1/admin/records | 入出场记录 |
| GET | /api/v1/admin/orders | 订单列表 |
| GET | /api/v1/admin/orders/{id} | 订单详情 |
| POST | /api/v1/admin/orders/{id}/refund | 退款审批（需 LOT_ADMIN 权限） |
| GET | /api/v1/admin/vehicles | 车辆列表 |
| POST | /api/v1/admin/vehicles | 录入车辆 |
| GET | /api/v1/admin/billing/rules | 费率规则 |
| POST | /api/v1/admin/billing/rules | 创建规则 |
| PUT | /api/v1/admin/billing/rules/{id} | 更新规则 |
| GET | /api/v1/admin/reports/daily | 日报表 |
| GET | /api/v1/admin/reports/monthly | 月报表 |

#### 2.5.5 车主小程序 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | /api/v1/user/records | 查询停车记录（车牌 + 停车场） |
| GET | /api/v1/user/records/{id} | 单条停车详情 |
| POST | /api/v1/user/plates | 绑定车牌（实名认证后） |
| GET | /api/v1/user/plates | 我的车牌列表 |
| DELETE | /api/v1/user/plates/{id} | 解绑车牌 |
| POST | /api/v1/user/pay/create | 扫码缴费（基于 openId 发起支付） |
| GET | /api/v1/user/orders | 我的订单列表 |
| GET | /api/v1/user/orders/{id} | 订单详情 |
| POST | /api/v1/user/orders/{id}/refund | 车主退款（原路返回） |
| GET | /api/v1/user/monthly-card | 月卡信息查询 |
| POST | /api/v1/user/monthly-card/purchase | 月卡购买/续费 |

认证方式：微信小程序 `openId` + 阿里云/微信云短信验证码，JWT 签发给车主端。

### 2.6 模块详细设计

#### 2.6.1 车牌识别模块

```typescript
// 识别服务架构
interface PlateRecognizer {
  // 车牌识别
  recognize(imageUrl: string): Promise<RecognitionResult>;

  // 识别结果校验
  validate(result: RecognitionResult): Promise<ValidationResult>;

  // 异常车牌处理
  handleException(plate: string, imageUrl: string): Promise<ExceptionHandleResult>;
}

interface RecognitionResult {
  plateNumber: string;      // 车牌号
  confidence: number;        // 置信度 0-1
  plateColor: string;        // 车牌颜色
  vehicleColor: string;      // 车辆颜色
  vehicleType: string;       // 车辆类型
  imageUrl: string;          // 识别图片
}

// 识别策略
class RecognitionStrategy {
  // 多引擎融合识别（并行调用，快速返回）
  async fuseRecognize(imageUrl: string): Promise<RecognitionResult> {
    // 1. 并行请求本地识别和云端识别
    const [localResult, cloudResult] = await Promise.allSettled([
      this.localRecognize(imageUrl),
      this.cloudRecognize(imageUrl)
    ]);

    // 2. 提取结果（云服务超时不阻塞）
    const local = localResult.status === 'fulfilled' ? localResult.value : null;
    const cloud = cloudResult.status === 'fulfilled' ? cloudResult.value : null;

    // 3. 置信度融合策略
    // 本地高置信度：直接返回（最快）
    if (local && local.confidence >= 0.9) {
      return local;
    }

    // 云端可用：优先用云端（更准确）
    if (cloud && cloud.confidence >= 0.8) {
      return cloud;
    }

    // 云端超时或低置信度：降级用本地结果
    if (local) {
      return { ...local, source: 'local_fallback' };
    }

    // 双引擎都失败：抛异常，走人工审核
    throw new Error('Both recognition engines failed');
  }
}
```

#### 2.6.2 计费引擎模块

```typescript
// 计费引擎核心
class BillingEngine {
  // 计算停车费用
  async calculateFee(record: ParkingRecord): Promise<Bill> {
    // 1. 获取所有适用规则
    const rules = await this.getApplicableRules(record.lotId);

    // 2. 按优先级排序（优先级高的在前）
    const sortedRules = rules.sort((a, b) => b.priority - a.priority);

    // 3. 互斥匹配：只应用第一条匹配的规则
    // 规则按 type 分组：time/period 为一类，monthly/vip 为一类
    // 不同类别可叠加，同类别取最高优先级
    let primaryRule: BillingRule | null = null;
    let primaryBill: { amount: number; discount: number } = { amount: 0, discount: 0 };

    for (const rule of sortedRules) {
      if (!this.matchConditions(rule, record)) continue;
      if (!primaryRule) {
        // 第一条匹配规则作为主规则
        primaryRule = rule;
        primaryBill = this.applyRule(rule, record);
      } else if (rule.type === 'monthly' || rule.type === 'vip') {
        // 月卡/VIP 类型可与临时费叠加（先收月费，出场时按时间补差）
        const secondaryBill = this.applyRule(rule, record);
        // 月卡免停时段内直接返回月费；超出时段则叠加临时费
        if (record.parkingDuration <= this.getFreeDuration(rule)) {
          return {
            recordId: record.id,
            baseAmount: secondaryBill.amount,
            discountAmount: 0,
            finalAmount: secondaryBill.amount,
            rules: [rule],
            billType: 'monthly'
          };
        }
        // 超出时段：月费 + 超出部分临时费
        primaryBill.amount += secondaryBill.amount;
      } else if (rule.type === 'coupon') {
        // 优惠卷叠加到当前计算结果
        const couponBill = this.applyRule(rule, record);
        primaryBill.discount += couponBill.discount;
      } else if (rule.type === 'period' && primaryRule.type !== 'period') {
        // 时段优惠（如夜间优惠）可与标准费率叠加
        const periodBill = this.applyRule(rule, record);
        primaryBill.discount += periodBill.discount;
      }
    }

    if (!primaryRule) {
      throw new Error('No applicable billing rule found');
    }

    // 4. 费用校验与封顶
    let finalAmount = Math.max(0, primaryBill.amount - primaryBill.discount);
    finalAmount = this.applyCap(finalAmount, record, primaryRule);
    finalAmount = this.validateAmount(finalAmount);

    return {
      recordId: record.id,
      baseAmount: primaryBill.amount,
      discountAmount: primaryBill.discount,
      finalAmount,
      rules: primaryRule ? [primaryRule] : [],
      billType: primaryRule.type
    };
  }

  // 费用封顶（如每日最高50元）
  // 正确逻辑：超过封顶天数才生效，且按实际封顶次数计算
  // 停25小时：ceil(25/24)=2天，封顶2次→min(amount, 50×2=100)
  // 停24小时：ceil(24/24)=1天，封顶1次→min(amount, 50×1=50)
  // 封顶生效时：丢弃按小时叠加的费用，只收封顶费用
  private applyCap(amount: number, record: ParkingRecord, rule: BillingRule): number {
    const maxDaily = this.getActionParam(rule, 'max_daily');
    if (!maxDaily) return amount;

    const days = Math.ceil(record.parkingDuration / 86400);
    const capTotal = maxDaily * days;
    if (amount > capTotal) {
      return capTotal;  // 超出封顶时，只收封顶费用
    }
    return amount;  // 未超封顶，按实际费用收
  }

  // 获取月卡免费时长（秒）
  private getFreeDuration(rule: BillingRule): number {
    return rule.actions
      .find(a => a.type === 'free_duration')?.value || 0;
  }

  private getActionParam(rule: BillingRule, type: string): number | undefined {
    return rule.actions.find(a => a.type === type)?.amount;
  }
}

// 规则配置示例
interface BillingRule {
  id: string;
  name: string;
  type: 'time' | 'period' | 'monthly' | 'coupon' | 'vip';
  conditions: Condition[];
  actions: Action[];
  priority: number;
  isActive: boolean;
}

  // 规则类型
  type Condition = {
    field: 'vehicle_type' | 'parking_duration' | 'exit_time' | 'entry_time';
    operator: '==' | '!=' | '>' | '<' | 'between' | 'in' | 'contains';
    value: any;
  };

// 月卡有效期校验（出场时必须调用）
async function validateMonthlyCard(record: ParkingRecord): Promise<ValidationResult> {
  const vehicle = await this.vehicleRepository.findByPlate(record.plateNumber);
  if (!vehicle) return { valid: false, reason: 'vehicle_not_found' };

  if (vehicle.vehicle_type !== 'monthly' && vehicle.vehicle_type !== 'vip') {
    return { valid: true }; // 非月卡，无需校验
  }

  const now = record.exitTime || new Date();
  if (!vehicle.monthly_valid_until) {
    return { valid: false, reason: 'no_validity_period' };
  }

  if (new Date(vehicle.monthly_valid_until) < now) {
    // 月卡过期，转为临时车标准收费
    return {
      valid: false,
      reason: 'monthly_expired',
      chargeAs: 'temporary',
      expiredDays: Math.ceil((now.getTime() - new Date(vehicle.monthly_valid_until).getTime()) / 86400000)
    };
  }

  // 月卡有效期内，免收停车费（或按 free_duration 减免）
  return { valid: true, freeDuration: 0, discount: 0 };
}

// 并发控制：出场计费使用分布式悲观锁
async function calculateFeeWithLock(recordId: string): Promise<Bill> {
  const lockKey = `parking:lock:exit:${recordId}`;
  const lock = await this.acquireLock(lockKey, 30000); // 30秒超时
  if (!lock) throw new Error('Failed to acquire lock, another exit in progress');

  try {
    const record = await this.recordRepository.findById(recordId);
    if (record.record_status === 'paid') {
      throw new Error('Already paid');
    }
    return await this.calculateFee(record);
  } finally {
    await this.releaseLock(lockKey);
  }
}

type Action = {
  type: 'base_rate' | 'hourly_rate' | 'max_daily' | 'discount' | 'fixed_rate' | 'free_duration';
  amount?: number;
  unit?: string;
  percentage?: number;
  maxAmount?: number;
  value?: number;  // free_duration 使用 value（秒）而非 amount
};
```

#### 2.6.3 支付模块

```typescript
// 支付服务
class PaymentService {
  // 创建支付订单
  async createPayment(params: PaymentParams): Promise<PaymentOrder> {
    // 1. 校验订单有效性
    await this.validateOrder(params.recordId);

    // 2. 创建本地订单
    const order = await this.orderRepository.create({
      recordId: params.recordId,
      lotId: params.lotId,
      plateNumber: params.plateNumber,
      amount: params.amount,
      finalAmount: params.amount,
      status: 'pending',
      expireTime: Date.now() + 30 * 60 * 1000 // 30分钟过期
    });

    // 3. 调用第三方支付（notifyUrl 由服务端配置，不从参数传入，防伪造）
    const payResult = await this.thirdPartyPay.create({
      orderId: order.id,
      amount: params.amount,
      payMethod: params.payMethod,
      openId: params.openId,
      notifyUrl: this.config.paymentCallbackUrl  // 系统配置，不接受外部传入
    });

    // 4. 返回支付信息
    return {
      orderId: order.id,
      payUrl: payResult.payUrl,
      qrCode: payResult.qrCode,
      expireTime: order.expireTime
    };
  }

  // 支付回调处理
  async handleCallback(params: CallbackParams): Promise<void> {
    // 1. 验签（防伪造）
    const valid = await this.verifySignature(params);
    if (!valid) throw new Error('Invalid signature');

    // 2. 幂等性校验（防重复回调）
    const existing = await this.getByTransactionId(params.transactionId);
    if (existing) return;

    // 3. 查询本地订单并加悲观锁
    const order = await this.orderRepository.findByIdForUpdate(params.orderId);
    if (!order) throw new Error('Order not found');
    if (order.status === 'paid') return; // 已被其他回调处理

    // 4. 金额校验（防小额支付攻击）
    const paidAmount = parseFloat(params.paidAmount || params.totalFee || '0');
    if (Math.abs(paidAmount - parseFloat(order.finalAmount)) > 0.01) {
      await this.logSecurityEvent('amount_mismatch', {
        orderId: order.id,
        expected: order.finalAmount,
        received: paidAmount,
        transactionId: params.transactionId
      });
      throw new Error('Amount mismatch');
    }

    // 5. 支付状态校验（防状态伪造）
    if (params.tradeState !== 'SUCCESS') {
      await this.orderRepository.update(order.id, { status: 'failed' });
      return;
    }

    // 6. 更新订单状态
    await this.orderRepository.update(order.id, {
      status: 'paid',
      payTime: new Date(),
      transactionId: params.transactionId,
      paidAmount: paidAmount
    });

    // 7. 更新入场记录状态（使用正确的字段名 exit_status）
    await this.recordRepository.update(order.recordId, {
      record_status: 'paid',
      exit_status: 'paid',
      payment_lock: order.payment_lock + 1
    });

    // 8. 自动触发开闸（扫码支付后自动放行）
    // 获取出场车道设备（需从 record 中获取设备关联信息）
    const record = await this.recordRepository.findById(order.recordId);
    const exitDeviceId = record.exit_device_id || record.lane_id;
    if (exitDeviceId) {
      try {
        await this.deviceControlService.openGate(exitDeviceId, order.recordId);
      } catch (e) {
        // 开闸失败：记录异常，车主可手动扫码再次触发
        await this.logDeviceEvent('auto_open_gate_failed', {
          recordId: order.recordId,
          deviceId: exitDeviceId,
          error: e.message
        });
      }
    }

    // 9. 发送通知
    await this.notifyService.sendPaymentSuccess(order);
  }
}
```

#### 2.6.4 设备控制模块

```typescript
// 设备控制服务
class DeviceControlService {
  // 发送开闸指令
  async openGate(deviceId: string, recordId: string): Promise<boolean> {
    const device = await this.getDevice(deviceId);

    // 0. 设备状态校验（维修/禁用时禁止开闸）
    if (!device.enabled) {
      throw new Error(`Device ${deviceId} is disabled`);
    }
    if (device.status === 'offline') {
      throw new Error(`Device ${deviceId} is offline`);
    }

    // 1. 乐观锁校验：防止同一入场记录被并发开闸
    const record = await this.recordRepository.findById(recordId);
    if (record.exit_status === 'paid' && record.record_status === 'paid') {
      console.warn(`Record ${recordId} already processed`);
      return false;
    }

    // 2. 离线模式下本地控制
    if (!await this.isOnline()) {
      const offlineId = await this.localControl.openGateOffline(device.gateId, recordId);
      await this.offlineRecordRepository.create({
        offlineId,
        recordId,
        deviceId,
        gateId: device.gateId,
        openTime: new Date(),
        status: 'pending_sync',
        offlineAmount: record.finalAmount
      });
      return true;
    }

    // 3. 在线模式云端控制
    const result = await this.sendCommand(deviceId, {
      command: 'open_gate',
      timeout: 5000
    });

    if (!result.success) {
      const offlineId = await this.localControl.openGateOffline(device.gateId, recordId);
      await this.offlineRecordRepository.create({
        offlineId,
        recordId,
        deviceId,
        gateId: device.gateId,
        openTime: new Date(),
        status: 'pending_sync'
      });
    }

    return true;
  }

  // 发送关闸指令（车辆驶过地感后触发）
  async closeGate(deviceId: string): Promise<boolean> {
    const device = await this.getDevice(deviceId);
    return this.localControl.closeGate(device.gateId);
  }

  // 地感信号处理（地感检测到车辆通过）
  async handleGroundSensorEvent(sensorId: string, event: GroundSensorEvent): Promise<void> {
    const device = await this.getDeviceBySensor(sensorId);

    if (event.type === 'vehicle_passed') {
      // 车辆完全通过地感：安全关闭闸机
      // 同一通道不会多次触发，因为闸机应已关闭
      await this.closeGate(device.gateId);

      // 更新设备状态
      await this.deviceRepository.update(device.id, {
        lastSensorEvent: new Date()
      });
    }
  }

  // 设备心跳处理
  async handleHeartbeat(deviceId: string, status: DeviceStatus): Promise<void> {
    await this.deviceRepository.update(deviceId, {
      online: true,
      lastHeartbeat: new Date(),
      status: status
    });
    await this.checkOfflineDevices();
  }

  // 离线数据同步（网络恢复后）
  async syncOfflineRecords(): Promise<SyncResult> {
    const pending = await this.offlineRecordRepository.findByStatus('pending_sync');
    let synced = 0, failed = 0;

    for (const record of pending) {
      try {
        await this.recordService.replayFromOffline(record);
        if (record.offlineAmount > 0) {
          await this.notifyOfflinePayment(record);
        }
        await this.offlineRecordRepository.update(record.id, { status: 'synced' });
        synced++;
      } catch (e) {
        await this.offlineRecordRepository.update(record.id, {
          status: 'sync_failed',
          error: e.message,
          retryCount: record.retryCount + 1
        });
        failed++;
      }
    }

    return { synced, failed, total: pending.length };
  }
}

// 地感事件类型
interface GroundSensorEvent {
  type: 'vehicle_detected' | 'vehicle_passed' | 'vehicle_clear';
  sensorId: string;
  timestamp: Date;
}

// 离线放行记录表（设备本地 SQLite，云端同步）
interface OfflineRecord {
  id: string;
  offlineId: string;
  recordId: string;
  deviceId: string;
  gateId: string;
  openTime: Date;
  offlineAmount: number;
  status: 'pending_sync' | 'synced' | 'sync_failed';
  retryCount: number;
  createdAt: Date;
}
```

### 2.7 消息队列设计

```
┌──────────────────────────────────────────────────────────────┐
│                       消息队列架构                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐                                             │
│  │ 入场事件    │──────┬──────────────────────────────────┐   │
│  └─────────────┘      │                                  │   │
│                      ▼                                  ▼   │
│              ┌───────────────┐                  ┌──────────┐ │
│              │ entry.events  │                  │ 计费队列  │ │
│              └───────────────┘                  └────┬─────┘ │
│                                                     │        │
│  ┌─────────────┐                                    ▼        │
│  │ 出场事件    │──────────────────┬─────────────────────────┐ │
│  └─────────────┘                  │                         │ │
│                                   ▼                         ▼ │
│                          ┌───────────────┐           ┌────────────┐
│                          │ payment.queue│           │ 通知队列   │
│                          └───────┬───────┘           └─────┬────┘
│                                  │                        │
│                                  ▼                        ▼ │
│                          ┌───────────────┐           ┌────────────┐
│                          │ 支付处理      │           │ 短信/推送  │
│                          └───────────────┘           └───────────┘
└──────────────────────────────────────────────────────────────┘

消费者组设计：
- billing-consumer-group     : 计费处理
- payment-consumer-group    : 支付处理
- notification-consumer-group: 通知处理
- data-sync-consumer-group  : 数据同步

消费语义说明：

┌─────────────────────────────────────────────────────────────┐
│                  各阶段消息队列选型与语义                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  第一阶段（Redis Streams）：                                   │
│  - 语义：至少一次（at-least-once）                             │
│  - 去重：消费前检查 record_id 是否已处理                        │
│  - 幂等：入场/出场记录创建用 record_id 做唯一约束              │
│  - 补偿：消费失败重试3次后进入死信队列（DLQ）                   │
│                                                              │
│  第二阶段（Kafka）：                                           │
│  - 语义：至少一次（at-least-once）                             │
│  - 精确一次：仅在支付结果同步使用事务型 producer，幂等 producer  │
│  - 顺序性：同一 partition 内有序（按 record_id partition）      │
│  - 消费者：billing/payment/notification 各自独立 consumer group │
│  - DLQ：消费失败消息进入 __consumer_offsets + dead-letter-topic│
│                                                              │
│  注意事项：                                                    │
│  - 第一阶段切 Kafka 时，record_id 生成规则不变，保证兼容        │
│  - 支付回调必须保证幂等，transaction_id 唯一索引                │
└─────────────────────────────────────────────────────────────┘
```

### 2.8 缓存设计

```
Redis 缓存策略：

┌─────────────────────────────────────────────────────────────┐
│                     缓存分层                                │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  L1 (热点数据)  │ 毫秒级  │ 识别结果、费率配置                │
│  L2 (中间数据)  │ 秒级    │ 车辆信息、订单状态                 │
│  L3 (持久化)   │ 分钟级  │ 统计报表、汇总数据                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘

Key 设计规范（带版本号，schema 变更时递增版本）：
  parking:v1:lot:{lotId}              # 停车场信息
  parking:v1:vehicle:{plate}          # 车辆信息
  parking:v1:record:{recordId}        # 入场记录
  parking:v1:order:{orderId}          # 订单信息
  parking:v1:rule:{lotId}             # 费率规则
  parking:v1:daily:stats:{lotId}:{date}  # 日统计
  parking:v1:device:secret:{deviceId}    # 设备 HMAC 密钥（加密存储）
  parking:v1:lock:exit:{recordId}        # 出场计费分布式锁
  parking:v1:offline:queue             # 离线数据待同步队列

过期策略：
  - 识别结果：5分钟
  - 订单信息：30分钟（支付后删除）
  - 车辆信息：1小时（带版本号，变更时主动 invalidate）
  - 费率配置：24小时（变更时主动 invalidate cache）
  - 分布式锁：30秒（自动过期防死锁）
```

### 2.9 安全设计

#### 2.9.1 安全架构

```
┌──────────────────────────────────────────────────────────────┐
│                       安全防护体系                            │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐       │
│  │   WAF防火墙  │  │  DDoS防护   │  │  IPS入侵防御 │       │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘       │
│         │                │                │                 │
│         └────────────────┴────────────────┘                 │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   API Gateway                           │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │  │
│  │  │ 认证鉴权  │ │ 频率限制  │ │ 参数校验  │ │ 日志审计  │  │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   业务服务层                            │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │  │
│  │  │ 数据加密  │ │ 敏感脱敏  │ │ 事务控制  │ │ 业务风控  │  │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
│                          │                                   │
│                          ▼                                   │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   数据存储层                            │  │
│  │  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐  │  │
│  │  │ 存储加密  │ │ 备份恢复  │ │ 访问控制  │ │ 审计日志  │  │  │
│  │  └──────────┘ └──────────┘ └──────────┘ └──────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

#### 2.9.2 认证与授权

```typescript
// JWT 认证配置
interface JWTConfig {
  publicKey: process.env.JWT_PUBLIC_KEY;      // RS256 非对称，公钥
  privateKey: process.env.JWT_PRIVATE_KEY;    // RS256 非对称，私钥
  expiresIn: '24h';
  refreshExpiresIn: '7d';
  algorithm: 'RS256';                         // 生产必须用非对称算法
}

// 权限等级
enum PermissionLevel {
  SUPER_ADMIN = 100,    // 超级管理员
  LOT_ADMIN = 50,       // 停车场管理员
  OPERATOR = 30,        // 操作员
  VIEWER = 10           // 只读
}

// 权限控制示例
const permissions = {
  'admin:lots:create': PermissionLevel.LOT_ADMIN,
  'admin:lots:update': PermissionLevel.LOT_ADMIN,
  'admin:orders:refund': PermissionLevel.LOT_ADMIN,
  'admin:records:view': PermissionLevel.OPERATOR,
  'admin:reports:view': PermissionLevel.VIEWER
};
```

#### 2.9.3 支付安全

```typescript
// 支付安全措施
interface PaymentSecurity {
  // 1. 签名验证
  verifySign(params: any, sign: string): boolean;

  // 2. 回调验签
  verifyCallback(params: any): Promise<boolean>;

  // 3. 订单幂等性
  idempotentCreate(orderId: string): Promise<boolean>;

  // 4. 金额校验
  validateAmount(amount: number): boolean;

  // 5. 交易超时
  checkTransactionTimeout(orderId: string): boolean;

  // 6. 风险监控
  monitorRisk(orderId: string, amount: number): RiskLevel;
}

// 支付风控规则
const riskRules = [
  { condition: 'amount > 1000', action: 'verify_identity' },
  { condition: 'frequency > 10/hour', action: 'limit_request' },
  { condition: 'same_plate > 5/day', action: 'alert' }
];

// 退款审批流程
interface RefundWorkflow {
  // ============================================================
  // 路径一：车主端退款（原路返回，自动处理，无需审批）
  // 触发场景：付款后发现金额错误、临时车想取消等
  // 条件：订单状态为 paid，且未超出退款时效（如 30 分钟）
  // ============================================================
  async autoRefund(orderId: string, requestorId: string): Promise<RefundResult> {
    const order = await this.orderRepository.findById(orderId);
    if (order.status !== 'paid') throw new Error('Order not paid');

    // 1. 检查退款时效（默认 30 分钟内可自退）
    const refundWindowMs = 30 * 60 * 1000;
    if (order.payTime && (Date.now() - new Date(order.payTime).getTime()) > refundWindowMs) {
      throw new Error('Refund window expired, please contact admin');
    }

    // 2. 检查是否正在处理中
    if (order.status === 'refunding') throw new Error('Refund in progress');

    // 3. 更新订单状态为退款中
    await this.orderRepository.update(orderId, { status: 'refunding' });

    try {
      // 4. 调用支付渠道退款（原路返回）
      // 微信：申请退款接口 → 资金退回原支付账户
      // 支付宝：退款接口 → 资金退回原支付账户
      const refundResult = await this.thirdPartyPay.refund({
        orderId: order.id,
        transactionId: order.transactionId,
        refundAmount: order.finalAmount,      // 全额退款
        refundReason: 'user_request'
      });

      // 5. 更新订单状态
      await this.orderRepository.update(orderId, {
        status: 'refunded',
        refundedAt: new Date(),
        refundTransactionId: refundResult.refundTransactionId
      });

      // 6. 更新入场记录状态
      await this.recordRepository.update(order.recordId, {
        record_status: 'refunded',
        exit_status: 'refunded'
      });

      // 7. 审计日志（自动退款也记录）
      await this.auditLog.record({
        action: 'auto_refund',
        orderId,
        amount: order.finalAmount,
        requestor: requestorId,
        refundMethod: order.payMethod,    // 原路返回
        timestamp: new Date()
      });

      return { success: true, refundTransactionId: refundResult.refundTransactionId };

    } catch (e) {
      // 退款失败：回滚状态，降级到人工退款
      await this.orderRepository.update(orderId, { status: 'paid' });
      await this.logError('auto_refund_failed', { orderId, error: e.message });

      // 自动创建人工退款审批单
      await this.createManualRefundApproval(order, requestorId, 'auto_refund_failed');
      throw new Error('Refund failed, manual review required');
    }
  }

  // ============================================================
  // 路径二：管理端退款（需审批，适用于异常退款）
  // 触发场景：超时退款、部分退款、投诉退款
  // ============================================================
  async createRefundRequest(orderId: string, amount: number, reason: string, applicant: string): Promise<RefundApproval> {
    const order = await this.orderRepository.findById(orderId);
    if (order.status !== 'paid') throw new Error('Order not paid');
    if (amount > parseFloat(order.finalAmount)) throw new Error('Amount exceeds paid');

    return await this.refundApprovalRepository.create({
      orderId,
      applicant,
      amount,
      reason,
      refundMethod: 'original',           // 默认原路返回
      status: 'pending'
    });
  }

  // LOT_ADMIN 审批通过后执行退款
  async approveRefund(approvalId: string, approver: string): Promise<void> {
    const approval = await this.refundApprovalRepository.findById(approvalId);
    if (approval.status !== 'pending') throw new Error('Already processed');

    await this.refundApprovalRepository.update(approvalId, {
      status: 'approved',
      approver,
      approvedAt: new Date()
    });

    // 执行退款（原路返回）
    const order = await this.orderRepository.findById(approval.orderId);
    await this.orderRepository.update(approval.orderId, { status: 'refunding' });

    try {
      const refundResult = await this.thirdPartyPay.refund({
        orderId: order.id,
        transactionId: order.transactionId,
        refundAmount: approval.amount,
        refundReason: approval.reason
      });

      await this.orderRepository.update(approval.orderId, {
        status: 'refunded',
        refundedAt: new Date(),
        refundTransactionId: refundResult.refundTransactionId
      });

      await this.recordRepository.update(order.recordId, {
        exit_status: 'refunded'
      });

      await this.auditLog.record({
        action: 'refund_approved',
        approvalId,
        orderId: approval.orderId,
        amount: approval.amount,
        refundMethod: order.payMethod,
        approver,
        timestamp: new Date()
      });

    } catch (e) {
      await this.orderRepository.update(approval.orderId, { status: 'paid' });
      throw new Error('Refund execution failed: ' + e.message);
    }
  }

  async rejectRefund(approvalId: string, approver: string, reason: string): Promise<void> {
    await this.refundApprovalRepository.update(approvalId, {
      status: 'rejected',
      approver,
      approvedAt: new Date(),
      rejectReason: reason
    });
  }
}
```

#### 2.9.4 数据安全

数据安全策略：

| 分类 | 敏感等级 | 示例 |
|------|----------|------|
| 公开数据 | 无敏感信息 | 车牌识别日志 |
| 内部数据 | 需登录查看 | 订单信息、车辆信息 |
| 敏感数据 | 加密存储 | 支付信息、身份证 |
| 机密数据 | 严格访问控制 | 商户密钥、用户支付密码 |

加密策略：
- 传输加密：TLS 1.3
- 存储加密：AES-256-GCM
- 密钥管理：阿里云 KMS / AWS KMS
- 敏感字段：手机号、身份证、支付信息加密存储

#### 2.9.5 多租户数据隔离策略

隔离模型：逻辑隔离（同一数据库，不同 lot_id）

| 层次 | 实现方式 | 要点 |
|------|----------|------|
| 接入层 | 中间件解析 JWT | 从 token 中提取 tenant_id，所有 DB 查询强制注入 WHERE lot_id = :tenant_id |
| 服务层 | Repository 统一拦截 | 所有查询自动带上租户上下文，跨租户查询需显式声明并记录审计日志 |
| 数据层 | 行级安全策略（RLS） | Redis 按租户隔离 key 前缀 |
| 监控层 | 独立统计 | 每个租户的用量独立统计，支持按租户分账 |

实现要点：
- 中间件层注入 ctx.tenantId，所有 Repository 查询自动拼接 lot_id
- 超级管理员（SUPER_ADMIN）可跨租户查询，但需记录操作日志
- 设备 secret 按 lot_id 隔离，设备只能操作其所属停车场的车道
- 账单/统计按租户独立计算，不混用

### 2.10 运维监控体系

#### 2.10.1 监控架构

```
┌──────────────────────────────────────────────────────────────┐
│                       监控体系                                │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   数据采集层                            │  │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐       │  │
│  │  │ Metrics │ │  Logs  │ │ Traces │ │ Events │       │  │
│  │  │ (指标)  │ │ (日志) │ │ (链路) │ │ (事件) │       │  │
│  │  └────┬───┘ └───┬────┘ └───┬────┘ └───┬────┘       │  │
│  └───────┼─────────┼─────────┼─────────┼────────────────┘  │
│          └─────────┴─────────┴─────────┘                      │
│                         │                                     │
│                         ▼                                     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                   Prometheus + Grafana                │  │
│  └──────────────────────────────────────────────────────┘  │
│                         │                                     │
│                         ▼                                     │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                     告警中心                          │  │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐          │  │
│  │  │  短信    │  │  邮件    │  │  钉钉/飞书│          │  │
│  │  └──────────┘  └──────────┘  └──────────┘          │  │
│  └──────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

#### 2.10.2 核心监控指标

| 监控类别 | 指标项 | 告警阈值 | 处理SLA |
|----------|--------|----------|---------|
| **服务可用性** | 系统 uptime | < 99.9% | 立即告警 |
| **接口性能** | API 响应时间 P99 | > 500ms | 15分钟内 |
| **接口性能** | API 错误率 | > 1% | 立即告警 |
| **业务指标** | 支付成功率 | < 99.5% | 立即告警 |
| **业务指标** | 车牌识别率 | < 95% | 2小时内 |
| **资源使用** | CPU 使用率 | > 80% | 30分钟内 |
| **资源使用** | 内存使用率 | > 85% | 30分钟内 |
| **资源使用** | 磁盘使用率 | > 90% | 1小时内 |
| **数据库** | 连接数 | > 80% max | 30分钟内 |
| **数据库** | 慢查询 | > 100ms | 2小时内 |

#### 2.10.3 日志规范

```typescript
// 日志格式规范
interface LogEntry {
  timestamp: string;          // ISO8601 时间
  level: 'debug' | 'info' | 'warn' | 'error';
  service: string;           // 服务名称
  traceId: string;           // 链路追踪ID
  spanId: string;            // 跨度ID
  userId?: string;           // 用户ID
  action: string;            // 操作类型
  resource: string;          // 资源类型
  resourceId?: string;       // 资源ID
  method: string;            // HTTP方法
  path: string;              // 请求路径
  statusCode: number;        // 响应码
  duration: number;          // 耗时ms
  requestIp: string;         // 请求IP
  userAgent: string;         // UA
  params?: object;           // 请求参数
  result?: object;           // 响应结果
  error?: ErrorInfo;         // 错误信息
}

// 日志级别策略
const logLevelStrategy = {
  'payment:*': 'warn',      // 支付相关warn级别
  'order:*': 'info',         // 订单相关info级别
  'device:*': 'debug',       // 设备相关debug级别
  '*.error': 'error'         // 错误都是error级别
};
```

#### 2.10.4 应急预案

```
应急预案分类：

┌─────────────────────────────────────────────────────────────┐
│                     故障分级                                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  P0 - 严重   │ 全站不可用  │ 支付完全中断 │ 恢复时间 < 30min │
│  P1 - 重大   │ 核心功能异常│ 识别率下降   │ 恢复时间 < 2h   │
│  P2 - 一般   │ 非核心故障  │ 报表延迟     │ 恢复时间 < 24h  │
│  P3 - 轻微   │ 体验问题    │ 界面显示异常 │ 恢复时间 < 72h  │
│                                                              │
└─────────────────────────────────────────────────────────────┘

应急预案流程：

┌──────────────────────────────────────────────────────────────┐
│                     故障处理流程                              │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  监控告警 ──▶ 确认故障 ──▶ 启动应急预案 ──▶ 故障定位        │
│      │                                        │              │
│      │                                        ▼              │
│  通知相关人 ◀── 升级汇报 ◀────── 恢复服务 ──▶ 问题修复      │
│      │                                        │              │
│      │                                        ▼              │
│      │                               ┌─────────────┐         │
│      └──────────────────────────────▶│  故障报告   │         │
│                                      └─────────────┘         │
└──────────────────────────────────────────────────────────────┘

常见故障处理：

1. OCR识别服务故障
   - 切换到备用OCR服务商
   - 开启人工审核通道
   - 记录异常后续处理

2. 支付服务故障
   - 开启离线放行模式
   - 记录欠费后续补缴
   - 切换备用支付通道

3. 数据库故障
   - 切换到只读副本
   - 启用缓存数据兜底
   - 故障转移到大容灾

 4. 网络中断
    - 设备本地独立运行
    - 数据本地缓存
    - 网络恢复后同步

#### 2.10.5 应急回滚方案

```
┌──────────────────────────────────────────────────────────────┐
│                     数据库 Schema 迁移回滚                    │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  原则：永远向后兼容（Backward Compatible）                    │
│                                                              │
│  迁移规范：                                                  │
│  1. 新增字段：始终提供默认值，代码向前兼容                     │
│  2. 删除字段：先代码废弃（deprecate），再沉默一个版本后删除    │
│  3. 修改字段：先加新字段，迁移数据，再删除旧字段              │
│  4. 索引：异步创建（CREATE INDEX CONCURRENTLY），不影响线上   │
│                                                              │
│  回滚流程：                                                  │
│  migration_down → 确认数据完整性 → 验证业务功能               │
│                                                              │
│  禁止操作：                                                  │
│  - 不允许在高峰期（业务高峰、支付高峰）执行 DDL               │
│  - 不允许删除带数据的字段                                      │
│  - 不允许修改有索引的字段类型                                  │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                     配置变更回滚                              │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  所有配置通过环境变量或配置中心（Nacos/Apollo）管理            │
│  - 配置变更前快照当前版本                                       │
│  - 支持秒级回滚到上一版本                                       │
│  - 计费规则变更：改后不影响进行中的订单                         │
│  - 支付渠道切换：保留旧渠道为 fallback                         │
│                                                              │
│  回滚触发条件：                                               │
│  - 错误率上升 > 1%                                            │
│  - P99 响应时间 > 2x 基线                                      │
│  - 业务指标异常（如支付成功率 < 99%）                           │
└──────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────┐
│                     服务版本回滚                              │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  蓝绿部署：2 套环境，新版验证通过后切换流量                     │
│  - 回滚：切换流量回旧版本，不重新部署                          │
│  - 切换时间 < 30s                                              │
│                                                              │
│  灰度发布：按设备/用户比例灰度                                │
│  - 发现问题立即切回 0% 灰度                                    │
│  - 自动降级兜底逻辑保持可用                                    │
└──────────────────────────────────────────────────────────────┘
```

---

## 三、系统演进规划

### 3.1 版本路线图

```
Month 1 ─── MVP v1.0
          - 车牌识别
          - 基础计费
          - 微信/支付宝支付
          - 简单后台

Month 3 ─── v1.5
          - 多种计费规则
          - 月卡管理
          - 优惠卷系统
          - 数据报表

Month 6 ─── v2.0
          - 多停车场支持
          - 车位引导
          - 会员体系
          - API开放

Month 12 ── v2.5
          - 城市级平台
          - 车位预约
          - 充电桩对接
          - 数据运营

Month 24 ── v3.0
          - AI智能定价
          - 无人值守全套
          - 生态系统
```

### 3.2 架构演进

#### 第一阶段：单体架构（Month 0-6）

```
┌─────────────────────────────────┐
│         单体应用                 │
│  ┌───────────────────────────┐  │
│  │ Gateway │ OCR │ Billing   │  │
│  │ Order  │ Pay │ Vehicle   │  │
│  │ Admin  │ Notify│ Data    │  │
│  └───────────────────────────┘  │
│              │                  │
│  ┌───────────┴───────────┐     │
│  │    PostgreSQL + Redis  │     │
│  └───────────────────────┘     │
└─────────────────────────────────┘

特点：
- 快速开发
- 简单部署
- 适合 1-20 个停车场
- 2-4 台服务器
```

#### 第二阶段：服务拆分（Month 6-12）

```
┌─────────────────────────────────────────────────────┐
│                    API Gateway                        │
└─────────────────────────────────────────────────────┘
           │           │           │           │
           ▼           ▼           ▼           ▼
    ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐
    │  识别服务  │ │  计费服务  │ │  支付服务  │ │  车辆服务  │
    └───────────┘ └───────────┘ └───────────┘ └───────────┘
           │           │           │           │
           ▼           ▼           ▼           ▼
    ┌─────────────────────────────────────────────────┐
    │              消息队列 (Redis Streams)           │
    └─────────────────────────────────────────────────┘
           │           │           │           │
           ▼           ▼           ▼           ▼
    ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐
    │  订单服务  │ │  通知服务  │ │  数据服务  │ │  用户服务  │
    └───────────┘ └───────────┘ └───────────┘ └───────────┘
                        │
                        ▼
                ┌─────────────┐
                │ PostgreSQL   │
                │ (读写分离)   │
                └─────────────┘

特点：
- 微服务架构
- 独立扩展
- 适合 20-100 个停车场
- 6-10 台服务器
```

#### 第三阶段：多活架构（Month 12+）

```
┌─────────────────────────────────────────────────────────────┐
│                     全球负载均衡 (GSLB)                       │
└─────────────────────────────────────────────────────────────┘
                    │                        │
          ┌─────────┴─────────┐    ┌─────────┴─────────┐
          │    华北集群        │    │    华南集群        │
          │  (主)             │    │  (备)             │
          └─────────┬─────────┘    └─────────┬─────────┘
                    │                        │
    ┌───────────────┼────────────────────────┼───────────────┐
    │               │                        │               │
    ▼               ▼                        ▼               ▼
┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐    ┌────────┐
│ 网关集群 │    │识别集群 │    │计费集群 │    │支付集群 │    │订单集群 │
└────────┘    └────────┘    └────────┘    └────────┘    └────────┘
                      │                        │
                      ▼                        ▼
              ┌─────────────────┐    ┌─────────────────┐
              │  Kafka 集群     │    │  Redis 集群     │
              │  (消息队列)     │    │  (缓存/会话)    │
              └─────────────────┘    └─────────────────┘
                      │
                      ▼
              ┌─────────────────┐
              │ TiDB 分布式数据库│
              │ (多主架构)       │
              └─────────────────┘

特点：
- 多地多活
- 弹性扩展
- 适合 100+ 停车场
- K8s 容器编排
```

### 3.3 部署方案

#### 3.3.1 小型方案（1-20停车场）

```
┌─────────────────────────────────────────────────────────────┐
│                     单机部署架构                             │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│                    ┌─────────────────┐                      │
│                    │   负载均衡      │                      │
│                    │  (SLB/Nginx)   │                      │
│                    └────────┬────────┘                      │
│                             │                               │
│                    ┌────────┴────────┐                      │
│                    │   云服务器       │                      │
│                    │   (4核8G)       │                      │
│                    │  ┌───────────┐  │                      │
│                    │  │ 应用服务   │  │                      │
│                    │  │ (Docker)  │  │                      │
│                    │  └───────────┘  │                      │
│                    └────────┬────────┘                      │
│                             │                               │
│         ┌───────────────────┼───────────────────┐           │
│         │                   │                   │           │
│         ▼                   ▼                   ▼           │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    │
│  │ PostgreSQL  │    │    Redis    │    │    OSS      │    │
│  │  (云数据库)  │    │   (缓存)    │    │  (文件存储)  │    │
│  └─────────────┘    └─────────────┘    └─────────────┘    │
│                                                              │
│  云服务商：阿里云/腾讯云（单区域）                            │
│  成本：¥500-1000/月                                          │
└─────────────────────────────────────────────────────────────┘
```

#### 3.3.2 中型方案（20-100停车场）

```
┌─────────────────────────────────────────────────────────────┐
│                     主备部署架构                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│                    ┌─────────────────┐                      │
│                    │   全局负载均衡   │                      │
│                    │    (GSLB)       │                      │
│                    └────────┬────────┘                      │
│                             │                               │
│         ┌───────────────────┴───────────────────┐           │
│         │                                       │           │
│  ┌──────┴──────┐                        ┌──────┴──────┐    │
│  │  主可用区    │                        │   备可用区   │    │
│  └──────┬──────┘                        └──────┬──────┘    │
│         │                                       │           │
│  ┌──────┴──────┐                        ┌──────┴──────┐    │
│  │  Nginx集群  │                        │  Nginx集群  │    │
│  └──────┬──────┘                        └──────┬──────┘    │
│         │                                       │           │
│  ┌──────┴──────┐                        ┌──────┴──────┐    │
│  │ 应用服务器集群│                        │ 应用服务器集群│    │
│  │ (K8s Pod)  │                        │ (K8s Pod)   │    │
│  └──────┬──────┘                        └──────┬──────┘    │
│         │                                       │           │
│  ┌──────┴──────┐                        ┌──────┴──────┐    │
│  │  Redis集群   │◀────────────────────▶│  Redis集群   │    │
│  └──────┬──────┘       数据同步         └──────┬──────┘    │
│         │                                       │           │
│  ┌──────┴──────┐                        ┌──────┴──────┐    │
│  │ PostgreSQL  │                        │ PostgreSQL  │    │
│  │  (主从)     │       异步复制          │  (从属)     │    │
│  └─────────────┘                        └─────────────┘    │
│                                                              │
│  云服务商：阿里云/腾讯云（双区域）                            │
│  成本：¥5000-10000/月                                        │
└─────────────────────────────────────────────────────────────┘
```

#### 3.3.3 Docker Compose 部署配置

```yaml
# docker-compose.yml
version: '3.8'

services:
  app:
    image: parking-system:latest
    container_name: parking-app
    user: "1000:1000"                      # 非 root 用户运行
    read_only: true                        # 根文件系统只读
    tmpfs:
      - /tmp:rw,noexec,nosuid,size=50m     # 临时文件 tmpfs
    ports:
      - "127.0.0.1:8080:8080"              # 仅本地暴露，不对公网
    environment:
      - NODE_ENV=production
    env_file: .env.production              # 密钥通过 env_file 注入，不写死在镜像
    secrets:                               # Docker Secrets 管理密钥
      - db_password
      - redis_password
      - jwt_secret
    depends_on:
      db:
        condition: service_healthy
      cache:
        condition: service_healthy
    restart: unless-stopped
    networks:
      - parking-network
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  db:
    image: postgres:15-alpine
    container_name: parking-db
    user: "999:999"                        # postgres 系统用户
    environment:
      - POSTGRES_DB=parking
      - POSTGRES_USER=parking_user
      - POSTGRES_PASSWORD_FILE=/run/secrets/db_password
    volumes:
      - pg_data:/var/lib/postgresql/data
      - ./backup:/backup:ro               # 备份只读挂载
    command: >
      postgres
      -c max_connections=100
      -c shared_buffers=128MB
      -c wal_level=replica
      -c archive_mode=on
    restart: unless-stopped
    networks:
      - parking-network
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U parking_user -d parking"]
      interval: 10s
      timeout: 5s
      retries: 5

  cache:
    image: redis:7-alpine
    container_name: parking-cache
    command: >
      redis-server
      --requirepass-file /run/secrets/redis_password
      --appendonly yes
      --maxmemory 256mb
      --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
      - ./redis.conf:/usr/local/etc/redis/redis.conf:ro
    restart: unless-stopped
    networks:
      - parking-network
    healthcheck:
      test: ["CMD", "redis-cli", "-a", "$${REDIS_PASSWORD}", "ping"]
      env:
        REDIS_PASSWORD_FILE: /run/secrets/redis_password
      setup: ["CMD", "sh", "-c", "echo $$(cat /run/secrets/redis_password) > /tmp/redis_pass && chmod 400 /tmp/redis_pass"]
      interval: 10s
      timeout: 5s
      retries: 3

  nginx:
    image: nginx:alpine
    container_name: parking-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
      - nginx_cache:/var/cache/nginx
    depends_on:
      - app
    restart: unless-stopped
    networks:
      - parking-network

volumes:
  pg_data:
  redis_data:
  nginx_cache:

networks:
  parking-network:
    driver: bridge

secrets:
  db_password:
    file: ./secrets/db_password.txt
  redis_password:
    file: ./secrets/redis_password.txt
  jwt_secret:
    file: ./secrets/jwt_secret.txt
```

#### 3.3.4 Kubernetes 部署配置

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: parking-app
  namespace: parking-system
spec:
  replicas: 3
  selector:
    matchLabels:
      app: parking-app
  template:
    metadata:
      labels:
        app: parking-app
    spec:
      containers:
      - name: app
        image: parking-system:latest
        ports:
        - containerPort: 8080
        env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: parking-secrets
              key: database-url
        - name: JWT_PRIVATE_KEY
          valueFrom:
            secretKeyRef:
              name: parking-secrets
              key: jwt-private-key
        - name: JWT_PUBLIC_KEY
          valueFrom:
            secretKeyRef:
              name: parking-secrets
              key: jwt-public-key
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: parking-app-svc
  namespace: parking-system
spec:
  selector:
    app: parking-app
  ports:
  - port: 80
    targetPort: 8080
  type: ClusterIP
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: parking-app-hpa
  namespace: parking-system
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: parking-app
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

---

## 四、实施计划

### 4.1 团队组建

| 阶段 | 人员配置 | 成本/月 |
|------|----------|---------|
| **启动期（0-3月）** | 创始人全栈 + 1兼职前端 | ¥0-15000 |
| **成长期（3-6月）** | 1后端 + 1前端 + 1销售 | ¥30000-50000 |
| **稳定期（6-12月）** | 2后端 + 1前端 + 2销售 + 1实施 | ¥80000-120000 |
| **扩张期（12月+）** | 完整团队 8-10 人 | ¥200000+ |

### 4.2 开发计划

#### Sprint 1-2（Week 1-4）：基础架构

| 任务 | 负责 | 状态 |
|------|------|------|
| 项目框架搭建 | 后端 | □ |
| 数据库设计 | 后端 | □ |
| 基础 API 编写 | 后端 | □ |
| 前端页面框架 | 前端 | □ |
| 权限系统 | 前端 | □ |

#### Sprint 3-4（Week 5-8）：核心功能

| 任务 | 负责 | 状态 |
|------|------|------|
| 车牌识别接入 | 后端 | □ |
| 入场流程实现 | 后端 | □ |
| 出场流程实现 | 后端 | □ |
| 计费逻辑开发 | 后端 | □ |
| 支付对接（微信/支付宝） | 后端 | □ |

#### Sprint 5-6（Week 9-12）：用户体验

| 任务 | 负责 | 状态 |
|------|------|------|
| 车主小程序开发 | 前端 | □ |
| 管理后台完善 | 前端 | □ |
| 数据报表开发 | 前端 | □ |
| 消息通知功能 | 后端 | □ |
| 系统测试 | QA | □ |

#### Sprint 7-8（Week 13-16）：部署运维

| 任务 | 负责 | 状态 |
|------|------|------|
| 云服务器部署 | 运维 | □ |
| 域名/SSL 配置 | 运维 | □ |
| 监控报警配置 | 运维 | □ |
| 首批客户试用 | 销售 | □ |
| 问题修复优化 | 全员 | □ |

### 4.3 成本预算

```
第一年运营成本估算：

┌─────────────────────────────────────────────────┐
│  项目                      │  金额/年            │
├─────────────────────────────────────────────────┤
│  云服务器（2核4G）          │  ¥3,000            │
│  云数据库（2核4G）          │  ¥5,000            │
│  Redis 缓存                │  ¥2,000            │
│  OCR API（10万次/年）       │  ¥10,000           │
│  域名/SSL                  │  ¥500              │
│  微信认证（小程序）         │  ¥300              │
│  支付宝商户注册             │  ¥0                │
│  监控/日志服务              │  ¥2,000            │
│  ---------------------------------------------- │
│  合计                      │  ¥22,800/年        │
└─────────────────────────────────────────────────┘

注：OCR 按每月约 8,333 次（10万次/年）估算，月均约 ¥833，年度 ¥10,000。
若高峰月达 1万次，月均成本约 ¥1,000，全年约 ¥12,000。
```

### 4.4 客户获取计划

#### 阶段一：冷启动（Month 1-3）

```
目标：获取首批 3-5 个客户

策略：
1. 熟人关系优先
   - 朋友的小区/停车场
   - 愿意免费试用的物业

2. 试用换案例
   - 第1个客户：免费1年，换案例授权
   - 第2-3个客户：半价，换转介绍

3. 快速交付
   - 3天内完成部署
   - 7天内稳定运行
   - 30天内收回尾款
```

#### 阶段二：渠道建设（Month 3-6）

```
目标：客户数达到 10-20 个

策略：
1. 发展代理商
   - 地方物业资源合作
   - 20-30% 返点

2. 硬件捆绑
   - 与硬件商合作
   - 卖硬件送软件

3. 行业活动
   - 参加地方安防展
   - 行业分享建立品牌
```

#### 阶段三：规模化（Month 6-12）

```
目标：区域领先，月收入 10 万+

策略：
1. 城市合伙人
   - 城市独家代理
   - 利益深度绑定

2. 口碑传播
   - 每个客户背后 8 个潜在客户
   - 推荐奖励机制

3. 产品矩阵
   - 推出高中低三档产品
   - 满足不同客户需求
```

---

## 五、定价方案

### 5.1 产品分级

| 版本 | 价格 | 功能 |
|------|------|------|
| **基础版** | ¥3,000/年/车道 | 车牌识别、基础计费、微信支付、简单报表 |
| **标准版** | ¥8,000/年/车道 | 全部基础功能、月卡管理、优惠卷、数据报表、API |
| **高级版** | ¥15,000/年/车道 | 全部标准功能、会员体系、车位引导、多停车场管理 |
| **定制版** | 单独报价 | 私有化部署、功能定制、驻场开发 |

### 5.2 套餐示例

```
小区停车场（2进2出，4车道）
┌─────────────────────────────────────────────────┐
│  基础版：4 × ¥3,000 = ¥12,000/年               │
│  标准版：4 × ¥8,000 = ¥32,000/年               │
│  高级版：4 × ¥15,000 = ¥60,000/年              │
└─────────────────────────────────────────────────┘

商场停车场（3进3出，6车道）
┌─────────────────────────────────────────────────┐
│  标准版：6 × ¥8,000 = ¥48,000/年               │
│  高级版：6 × ¥15,000 = ¥90,000/年              │
└─────────────────────────────────────────────────┘
```

---

## 六、风险与应对

### 6.1 技术风险

| 风险 | 影响 | 应对措施 |
|------|------|----------|
| 车牌识别率低 | 用户体验差 | 接入百度/腾讯OCR，兜底人工 |
| 支付失败 | 资金损失 | 预锁定机制，异步回调验证 |
| 系统崩溃 | 服务中断 | 多级监控，自动告警，快速恢复 |
| 数据丢失 | 业务受损 | 多地备份，定时快照，binlog备份 |

### 6.2 业务风险

| 风险 | 影响 | 应对措施 |
|------|------|----------|
| 客户欠费 | 现金流断裂 | 预付费机制，欠费自动停服务 |
| 竞争对手价格战 | 利润下降 | 差异化服务，不跟进低价 |
| 大厂渠道下沉 | 市场挤压 | 聚焦细分领域，深耕服务 |
| 人员流动 | 服务中断 | 文档标准化，关键岗位备份 |

### 6.3 市场风险

| 风险 | 影响 | 应对措施 |
|------|------|----------|
| 政策变化 | 业务受限 | 密切跟踪，灵活调整 |
| 甲方拖欠款 | 资金压力 | 合同明确条款，分期收款 |
| 技术变革 | 竞争力下降 | 持续投入研发，保持领先 |

---

## 七、关键成功因素

```
1. 产品核心能力
   ├── 识别准确率 > 98%
   ├── 系统稳定性 > 99.9%
   └── 故障响应时间 < 30分钟

2. 商业模式
   ├── SaaS 订阅，稳定现金流
   ├── 高续费率 > 80%
   └── 客户生命周期价值 > 3倍获客成本

3. 市场策略
   ├── 先做好本地市场
   ├── 建立口碑和案例
   └── 渠道合作快速复制

4. 运营效率
   ├── 交付周期 < 7天
   ├── 客户成功体系
   └── 自动化运维
```

---

## 八、里程碑规划

```
Timeline  │  客户数  │  月收入  │  团队    │  关键成果
──────────┼──────────┼──────────┼──────────┼──────────────────
Month 3   │   0-3    │   ¥0     │  1-2人   │  MVP上线
Month 6   │   5-10   │  ¥3-5万  │  3-4人   │  口碑建立
Month 12  │   20+    │  ¥10-20万│  6-8人   │  区域品牌
Month 24  │   50+    │  ¥30-50万│  10-15人 │  行业认可
Month 36  │  100+    │  ¥100万+ │  20+人   │  融资扩张
```

---

## 附录

### A. 技术栈选型

| 层级 | 技术方案 |
|------|----------|
| 后端框架 | NestJS / Spring Boot / Go |
| 数据库 | PostgreSQL / MySQL |
| 缓存 | Redis |
| 消息队列 | Redis Streams / RabbitMQ / RocketMQ / Kafka |
| 文件存储 | 阿里云 OSS / 七牛云 / AWS S3 |
| CDN | 阿里云 CDN / Cloudflare |
| OCR | 百度车牌识别 / 腾讯云 OCR / 阿里云 OCR |
| 支付 | 微信支付 / 支付宝 / 银联 |
| 监控 | Prometheus + Grafana |
| 日志 | ELK Stack / Loki + Grafana |
| CI/CD | GitHub Actions / GitLab CI / Jenkins |
| 容器编排 | Docker / Kubernetes |
| 服务网格 | Istio / Kong |

### B. 硬件选型参考

| 设备 | 推荐品牌 | 价格区间 | 选型建议 |
|------|----------|----------|----------|
| 车牌识别摄像头 | 海康/大华/宇视/科拓 | ¥2000-8000/套 | 优先选支持ONVIF协议的 |
| AI识别盒子 | 比特大陆/华为昇腾 | ¥5000-15000/台 | 大型项目可选本地识别 |
| 道闸 | 捷顺/红门/启功/百胜 | ¥3000-15000/套 | 看开闸速度（0.3s/0.6s） |
| 地感线圈 | 国产通用 | ¥500-1000/个 | 优先选数字式地感 |
| 高速道闸 | 捷顺/富士 | ¥8000-20000/套 | 商业体/写字楼推荐 |
| LED显示屏 | 国产通用 | ¥1000-3000/块 | 选双行显示的 |
| 自助缴费机 | 科拓/ETCP/自在泊 | ¥8000-20000/台 | 支持多种支付方式 |
| 车位引导摄像头 | 海康/科拓 | ¥1500-3000/套 | 选支持鱼眼矫正的 |
| 车位锁 | 酷太/智谷 | ¥300-800/个 | 共享车位需要 |

### C. 车牌识别率对比

| 方案 | 白天识别率 | 夜间识别率 | 恶劣天气 | 价格 | 适用场景 |
|------|------------|------------|----------|------|----------|
| 海康SDK | 98% | 95% | 90% | 免费 | 预算有限 |
| 百度OCR | 99% | 96% | 92% | ¥0.1/次 | 中小型 |
| 腾讯云 | 99% | 97% | 93% | ¥0.12/次 | 中大型 |
| 本地AI盒子 | 99.5% | 98% | 96% | ¥10000+/台 | 大型/高可靠性 |

### D. 第三方服务成本估算

| 服务 | 免费额度 | 超出单价 | 月均成本（1000次/天） |
|------|----------|----------|----------------------|
| 百度车牌识别 | 1000次/天 | ¥0.1/次 | ¥3,000 |
| 腾讯云OCR | 1000次/天 | ¥0.12/次 | ¥2,400 |
| 阿里云短信 | 100条/天 | ¥0.04/条 | ¥600 |
| 微信支付 | - | 0.6% | 交易额×0.6% |
| 支付宝 | - | 0.6% | 交易额×0.6% |

### E. 性能指标参考

| 指标 | 小型方案 | 中型方案 | 大型方案 |
|------|----------|----------|----------|
| API响应时间P99 | < 500ms | < 200ms | < 100ms |
| 车牌识别延迟 | < 1s | < 500ms | < 300ms |
| 并发支持 | 100 QPS | 1000 QPS | 10000+ QPS |
| 系统可用性 | 99.5% | 99.9% | 99.99% |
| 数据持久性 | 99.9% | 99.99% | 99.999% |

### F. 常见问题FAQ

```
Q: 如何提升车牌识别率？
A: 1. 选择高质量摄像头（400万像素以上）
   2. 合理安装角度（俯角15-30度）
   3. 良好光照条件
   4. 接入多个OCR服务做融合识别

Q: 遇到无牌车怎么处理？
A: 1. 放行并记录"无牌车"标识
   2. 出口手动输入入场时间
   3. 建议安装车牌识别盒子的"无牌车检测"功能

Q: 网络中断怎么办？
A: 1. 设备本地缓存识别数据
   2. 开启离线模式（允许欠费先走）
   3. 网络恢复后自动同步
   4. 欠费车辆加入黑名单

Q: 如何防止逃费？
A: 1. 双摄像头抓拍（前后牌照）
   2. 地感防砸车
   3. 埋地雷线圈检测
   4. 黑名单机制
   5. 欠费追缴流程
```

### G. 行业术语表

| 术语 | 解释 |
|------|------|
| 车道 | 每个出入口为一个车道 |
| OCR | 光学字符识别，用于车牌识别 |
| SDK | 软件开发包，硬件厂商提供 |
| API | 应用程序接口，对外服务 |
| NVR | 网络视频录像机 |
| ONVIF | 开放型网络视频接口标准 |
| PoE | 网线供电 |
| 对讲分机 | 车主与控制中心通话设备 |
| 道闸 | 拦车器，又称栏杆机 |
| 地感 | 检测车辆通过的传感器 |
| 抬杆 | 开闸放行 |
| 落杆 | 关闭通道 |

### H. 相关资源

```
学习资料：
- 海康威视停车场方案：https://www.hikvision.com
- 捷顺科技解决方案：https://www.jieshun.cn
- 科拓停车官网：https://www.keytop.com.cn

行业报告：
- 中国停车行业协会官网
- 艾瑞咨询-智慧停车行业报告
- 各地方政府停车收费政策文件

开发文档：
- 微信支付开发文档：https://pay.weixin.qq.com
- 支付宝开放平台：https://open.alipay.com
- 百度车牌识别：https://cloud.baidu.com
```

---

*文档版本：v1.4*
*最后更新：2026-03-20*
*作者：Parking System Team*

---

## 附录 I：v1.1 → v1.2 修复记录

| # | 问题 | 修复方案 |
|---|------|----------|
| 1 | 支付回调未校验金额 | 增加 `paidAmount` 与 `order.finalAmount` 比对，误差 > 0.01 拒绝 |
| 2 | 设备 API 无认证 | 引入 HMAC-SHA256 签名，5 分钟防重放，设备注册表存储密钥 |
| 3 | 出场查询无索引 | 新增 `(plate_number, entry_time)`、`(lot_id, record_status)` 联合索引 |
| 4 | 计费引擎规则累加 | 改为互斥匹配，相同类别取最高优先级，不同类别（time+monthly+coupon）可叠加 |
| 5 | exitStatus 字段不存在 | DB schema 修正为 `record_status` + `exit_status` 两个独立字段 |
| 6 | 月卡有效期未校验 | 新增 `validateMonthlyCard()` 出场时校验并降级为临时车 |
| 7 | 出场无并发控制 | 新增分布式悲观锁 `parking:lock:exit:{recordId}`，30 秒自动过期 |
| 8 | 离线放行无同步机制 | 新增 `offline_sync_records` 表，设备本地 SQLite 缓存，网络恢复后 replay |
| 9 | API 路径不一致 | 统一 `/api/v1/device/`、`/api/v1/pay/`、`/api/v1/admin/` 前缀 |
| 10 | Redis Key 无版本 | 统一加 `v1` 版本前缀，变更时主动 invalidate |
| 11 | Docker 配置安全隐患 | 非 root 用户运行、只读根文件系统、secrets 管理、健康检查 |
| 12 | 多租户隔离缺失 | 增加接入层/服务层/数据层三层隔离方案，RLS 行级安全 |
| 13 | 应急回滚方案缺失 | 新增 Schema 迁移规范、配置回滚、服务版本回滚（蓝绿部署） |
| 14 | 无牌车 DB 支持 | `plate_number` 改为可空，新增 `plate_number_source` 字段 |
| 15 | 消息队列消费语义不明 | 明确第一阶段 Redis Streams（至少一次）、第二阶段 Kafka（事务 producer 精确一次） |
| 16 | 退款无审批流程 | 新增 `refund_approvals` 表，LOT_ADMIN 审批 + 审计日志 |

---

## 附录 J：v1.2 审查修复记录（第二轮）

| # | 问题 | 修复方案 |
|---|------|----------|
| 17 | 客户端退款路径冗余 | 车主端自动退款（原路返回，30 分钟时效），管理端需审批用于异常退款 |
| 18 | 支付回调未触发开闸 | 支付成功后自动调用 `DeviceControlService.openGate()`，失败则降级为手动扫码 |
| 19 | 出场未校验车牌匹配 | 增加 `validatePlateMatch()`，无匹配记录时人工核验或离线放行 |
| 20 | JWT 使用 HS256 | 改为 RS256 非对称签名，客户端只持有公钥 |
| 21 | 代码块闭合错误 | 数据安全/多租户章节改为表格，消除嵌套 code block |
| 22 | 双重入场无处理 | 入场前检查 `findActiveByPlate()`，已有未出场记录直接返回 |
| 23 | 容灾同步策略错误 | 从库改为异步复制，主库故障不影响从库可用性 |
| 24 | 成本估算计算错误 | 百度车牌识别 1000 次/天 × ¥0.1 × 30 天 = ¥3,000/月 |
| 25 | 入场 API 重复请求体 | 删除重复 JSON，保留"请求："和"响应："格式 |
| 26 | 地感防砸落杆逻辑缺失 | 新增 `handleGroundSensorEvent()`，车辆通过地感后自动关闸 |
| 27 | 车主小程序接口缺失 | 新增 2.5.5 车主端 API（查记录、绑车牌、扫码缴费、退款） |
| 28 | OCR 融合策略串行延迟 | 改为 `Promise.allSettled()` 并行调用本地+云端，高置信度立即返回 |
| 29 | 闸机落杆逻辑缺失 | 地感事件触发 `closeGate()`，出场流程图增加落杆环节 |
| 30 | 设备维修无禁用机制 | `device_registry` 增加 `enabled: boolean`，维修时可设为 false |

---

## 附录 K：v1.3 审查修复记录（第三轮）

| # | 问题 | 修复方案 |
|---|------|----------|
| 31 | 出场 API 请求体重复 | 删除 HEAD 下重复 JSON，保留"请求："和"响应："格式 |
| 32 | 月卡校验后 gateOpen 未触发实际开闸 | 月卡有效时调用 `openGate(deviceId, recordId)` 再返回 |
| 33 | openGate 未校验设备 enabled 状态 | 新增 `device.enabled` 和 `status === 'offline'` 校验 |
| 34 | 无牌月卡车无法关联月卡信息 | 月卡查询改用 `vehicle_id` 而非 `plate_number`，无牌车跳过校验 |
| 35 | 计费封顶逻辑不够精确 | 封顶生效时只收 `maxDaily × ceil(days)`，不再叠加按小时费用 |
| 36 | 分布式锁/离线队列 Key 缺少版本号 | 统一加 `v1` 前缀：`parking:v1:lock:exit`、`parking:v1:offline:queue` |
| 37 | Redis healthcheck 不支持从 Secret 文件读取密码 | 改用环境变量注入密码，或 setup 脚本写入临时文件 |
| 38 | OCR 年预算与月均成本矛盾 | 附录年预算改为"10万次/年"，并注明月均约 ¥833，年度 ¥10,000 |
| 39 | 出场查询字段名 status 应为 record_status | 代码中 `findByPlateAndLot` 参数修正为 `{ record_status: [...] }` |
| 40 | K8s Secret 引用不完整 | 补充 `JWT_PRIVATE_KEY` 和 `JWT_PUBLIC_KEY` Secret 注入 |
| 41 | 计费规则示例 free_exit 无定义 | 改为 `free_duration`，并在 `Action` 类型中补全 `value?: number` 字段 |
| 42 | 支付创建 notifyUrl 由调用方传入 | 删除参数，改用服务端 `this.config.paymentCallbackUrl` 配置 |
| 43 | 月卡过期 vehicle_type 赋值未重新计费 | 改用 `record_metadata.chargeAs = 'temporary'` 标记，计费引擎自动识别 |

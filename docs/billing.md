# Billing Service 计费服务

## 模块概述

计费服务是 Smart Park 系统的核心计费引擎，负责处理所有停车费用的计算。服务采用规则引擎设计，支持灵活配置多种计费规则，包括按时计费、时段优惠、月卡管理、VIP折扣等，满足不同停车场的个性化计费需求。

## 核心功能

### 1. 计费规则引擎

#### 规则类型

| 规则类型 | 说明 | 适用场景 |
|----------|------|----------|
| `time` | 按时计费 | 临时车标准收费 |
| `period` | 时段计费 | 夜间优惠、节假日收费 |
| `monthly` | 月卡计费 | 月租车辆管理 |
| `vip` | VIP计费 | 特权车辆优惠 |
| `discount` | 折扣规则 | 优惠券、活动减免 |
| `exemption` | 免费规则 | 免费时段、特殊车辆 |

#### 规则结构

```go
// BillingRule 计费规则
type BillingRule struct {
    ID         uuid.UUID
    LotID      uuid.UUID              // 所属停车场
    RuleName   string                 // 规则名称
    RuleType   string                 // 规则类型
    Conditions string                 // 条件 JSON
    Actions    string                 // 动作 JSON
    Priority   int                    // 优先级（数值越大优先级越高）
    IsActive   bool                   // 是否启用
}

// Condition 条件定义
type Condition struct {
    Type       string                 // and/or/vehicle_type/duration_min/time_range/day_of_week/holiday
    Field      string                 // 字段名
    Operator   string                 // 操作符：==/!=/>/</>=/<=
    Value      interface{}            // 值
    Conditions []*Condition           // 嵌套条件
}

// Action 计费动作
type Action struct {
    Type    string   // fixed/per_hour/per_minute/percentage/cap/ceil/max_daily/min_charge/free_duration
    Amount  float64  // 金额
    Percent float64  // 百分比
    Unit    string   // 单位
    Cap     float64  // 封顶金额
    Value   float64  // 特定值（如免费时长秒数）
}
```

### 2. 条件评估引擎

#### 支持的条件类型

```go
// 车辆类型条件
{Type: "vehicle_type", Value: "temporary"}

// 时长条件（分钟）
{Type: "duration_min", Operator: "gte", Value: 60}

// 时段条件
{Type: "time_range", Value: {"start": 22.0, "end": 8.0}}

// 星期条件
{Type: "day_of_week", Value: [1, 2, 3, 4, 5]}  // 工作日

// 节假日条件
{Type: "holiday", Value: true}

// 组合条件
{
    Type: "and",
    Conditions: [
        {Type: "vehicle_type", Value: "temporary"},
        {Type: "duration_min", Operator: "gte", Value: 30}
    ]
}
```

#### 条件评估实现

```go
func EvaluateCondition(cond *Condition, ctx *BillingContext) bool {
    switch cond.Type {
    case "and":
        for _, c := range cond.Conditions {
            if !EvaluateCondition(c, ctx) {
                return false
            }
        }
        return true
        
    case "vehicle_type":
        return ctx.VehicleType == cond.Value
        
    case "duration_min":
        minutes := ctx.Duration.Minutes()
        switch cond.Operator {
        case "gte":
            return minutes >= cond.Value.(float64)
        case "lte":
            return minutes <= cond.Value.(float64)
        }
        
    case "time_range":
        hour := float64(ctx.ExitTime.Hour()) + float64(ctx.ExitTime.Minute())/60.0
        start := cond.Value.(map[string]interface{})["start"].(float64)
        end := cond.Value.(map[string]interface{})["end"].(float64)
        return hour >= start && hour <= end
        
    // ... 其他条件类型
    }
}
```

### 3. 计费动作执行

#### 支持的动作类型

```go
func applyActions(actions []*Action, duration time.Duration, exitTime time.Time) float64 {
    var amount float64
    hours := duration.Hours()
    minutes := duration.Minutes()
    
    for _, a := range actions {
        switch a.Type {
        case "fixed":           // 固定费用
            amount += a.Amount
            
        case "per_hour":        // 按小时计费
            amount += hours * a.Amount
            
        case "per_minute":      // 按分钟计费
            amount += minutes * a.Amount
            
        case "percentage":      // 百分比折扣
            amount -= amount * (a.Percent / 100)
            
        case "cap":             // 封顶金额
            if amount > a.Cap {
                amount = a.Cap
            }
            
        case "max_daily":       // 每日封顶
            days := int(hours/24) + 1
            maxAmount := a.Amount * float64(days)
            if amount > maxAmount {
                amount = maxAmount
            }
            
        case "min_charge":      // 最低收费
            if amount < a.Amount {
                amount = a.Amount
            }
            
        case "free_duration":   // 免费时长
            freeMinutes := a.Value / 60
            if duration.Minutes() <= float64(freeMinutes) {
                amount = 0
            }
            
        case "first_hour_free": // 首小时免费
            if hours <= 1 {
                amount = 0
            }
        }
    }
    
    return amount
}
```

### 4. 规则优先级与叠加

#### 规则匹配策略

```go
func (uc *BillingUseCase) CalculateFee(ctx context.Context, req *v1.CalculateFeeRequest) (*v1.BillData, error) {
    // 1. 获取所有启用规则
    rules, _ := uc.repo.GetRulesByLotID(ctx, lotID)
    
    // 2. 按优先级排序
    sort.Slice(rules, func(i, j int) bool {
        return rules[i].Priority > rules[j].Priority
    })
    
    var baseAmount float64
    var discountAmount float64
    
    for _, rule := range rules {
        // 3. 条件匹配
        cond, _ := ParseConditions(rule.Conditions)
        if !EvaluateCondition(cond, billingCtx) {
            continue
        }
        
        // 4. 执行动作
        actions, _ := ParseActions(rule.Actions)
        ruleAmount := applyActions(actions, duration, exitTime)
        
        // 5. 规则叠加策略
        switch rule.RuleType {
        case "base", "time":
            // 基础费用取最小值（互斥）
            if baseAmount == 0 || ruleAmount < baseAmount {
                baseAmount = ruleAmount
            }
        case "discount", "exemption":
            // 折扣可叠加
            discountAmount += ruleAmount
        case "monthly":
            // 月卡全额减免
            if req.VehicleType == "monthly" {
                discountAmount = baseAmount
            }
        }
    }
    
    // 6. 计算最终金额
    finalAmount := baseAmount - discountAmount
    if finalAmount < 0 {
        finalAmount = 0
    }
    
    return &v1.BillData{
        BaseAmount:     baseAmount,
        DiscountAmount: discountAmount,
        FinalAmount:    finalAmount,
    }, nil
}
```

## 应用场景区景

### 场景一：标准临时车计费

**需求**：
- 首小时 5 元
- 之后每小时 3 元
- 每日封顶 50 元
- 15 分钟内免费

**规则配置**：
```json
{
  "ruleName": "临时车标准收费",
  "ruleType": "time",
  "conditions": {
    "type": "vehicle_type",
    "value": "temporary"
  },
  "actions": [
    {"type": "free_duration", "value": 900},
    {"type": "fixed", "amount": 5},
    {"type": "per_hour", "amount": 3},
    {"type": "max_daily", "amount": 50}
  ],
  "priority": 100
}
```

### 场景二：夜间优惠

**需求**：
- 22:00 - 08:00 出场享受 5 折优惠
- 可与标准计费叠加

**规则配置**：
```json
{
  "ruleName": "夜间优惠",
  "ruleType": "discount",
  "conditions": {
    "type": "time_range",
    "value": {"start": 22.0, "end": 8.0}
  },
  "actions": [
    {"type": "percentage", "percent": 50}
  ],
  "priority": 50
}
```

### 场景三：月卡车辆

**需求**：
- 月卡有效期内免费
- 过期后按临时车收费

**规则配置**：
```json
{
  "ruleName": "月卡车辆",
  "ruleType": "monthly",
  "conditions": {
    "type": "vehicle_type",
    "value": "monthly"
  },
  "actions": [
    {"type": "fixed", "amount": 0}
  ],
  "priority": 200
}
```

**过期处理**：
```go
// 出场时校验月卡有效期
if vehicle.VehicleType == VehicleTypeMonthly {
    if vehicle.MonthlyValidUntil != nil && vehicle.MonthlyValidUntil.After(time.Now()) {
        finalAmount = 0  // 有效期内免费
    } else {
        // 过期标记，按临时车重新计费
        record.Metadata["chargeAs"] = VehicleTypeTemporary
        // 重新调用计费引擎
        feeResult, _ = uc.billingClient.CalculateFee(ctx, record, VehicleTypeTemporary)
    }
}
```

### 场景四：工作日/周末差异化收费

**需求**：
- 工作日：首小时 5 元，之后 3 元/小时
- 周末：首小时 3 元，之后 2 元/小时

**规则配置**：
```json
{
  "ruleName": "工作日收费",
  "ruleType": "time",
  "conditions": {
    "type": "and",
    "conditions": [
      {"type": "vehicle_type", "value": "temporary"},
      {"type": "day_of_week", "value": [1, 2, 3, 4, 5]}
    ]
  },
  "actions": [
    {"type": "fixed", "amount": 5},
    {"type": "per_hour", "amount": 3}
  ],
  "priority": 100
}
```

```json
{
  "ruleName": "周末收费",
  "ruleType": "time",
  "conditions": {
    "type": "and",
    "conditions": [
      {"type": "vehicle_type", "value": "temporary"},
      {"type": "day_of_week", "value": [6, 0]}
    ]
  },
  "actions": [
    {"type": "fixed", "amount": 3},
    {"type": "per_hour", "amount": 2}
  ],
  "priority": 100
}
```

## 技术挑战与解决方案

### 挑战一：复杂规则的性能优化

**问题描述**：
停车场可能配置数十条计费规则，每次计费都需要遍历匹配，性能可能成为瓶颈。

**解决方案**：

1. **规则索引**
   ```go
   // 按规则类型建立索引
   type RuleIndex struct {
       ByType    map[string][]*BillingRule
       ByLotID   map[uuid.UUID][]*BillingRule
       ActiveOnly []*BillingRule
   }
   ```

2. **缓存策略**
   ```go
   // 费率规则缓存 24 小时
   rules, err := uc.cache.GetOrSet(ctx, 
       fmt.Sprintf("parking:v1:rules:%s", lotID),
       func() ([]*BillingRule, error) {
           return uc.repo.GetRulesByLotID(ctx, lotID)
       },
       24*time.Hour,
   )
   ```

3. **预编译条件**
   ```go
   // 规则创建时预解析条件
   func (r *BillingRule) Precompile() error {
       cond, err := ParseConditions(r.Conditions)
       if err != nil {
           return err
       }
       r.parsedCondition = cond
       return nil
   }
   ```

### 挑战二：跨天计费处理

**问题描述**：
车辆可能停车跨越多天，需要正确处理每日封顶和跨天计费。

**解决方案**：

```go
func calculateCrossDayFee(entryTime, exitTime time.Time, dailyCap float64) float64 {
    totalDays := int(exitTime.Sub(entryTime).Hours() / 24) + 1
    var totalAmount float64
    
    for i := 0; i < totalDays; i++ {
        dayStart := entryTime.Add(time.Duration(i) * 24 * time.Hour)
        dayEnd := dayStart.Add(24 * time.Hour)
        if dayEnd.After(exitTime) {
            dayEnd = exitTime
        }
        
        dayDuration := dayEnd.Sub(dayStart)
        dayAmount := calculateDailyFee(dayDuration)
        
        // 每日封顶
        if dayAmount > dailyCap {
            dayAmount = dailyCap
        }
        
        totalAmount += dayAmount
    }
    
    return totalAmount
}
```

### 挑战三：规则冲突处理

**问题描述**：
多条规则可能同时匹配，需要明确的冲突解决策略。

**解决方案**：

1. **优先级机制**
   - 数值越大优先级越高
   - 同优先级按创建时间排序

2. **规则类型互斥策略**
   ```go
   // 基础费用规则互斥（取最小值）
   if rule.RuleType == "base" || rule.RuleType == "time" {
       if baseAmount == 0 || ruleAmount < baseAmount {
           baseAmount = ruleAmount
       }
   }
   
   // 折扣规则可叠加
   if rule.RuleType == "discount" {
       discountAmount += ruleAmount
   }
   ```

3. **规则预览功能**
   - 管理端提供规则预览
   - 模拟各种场景的费用计算
   - 提前发现规则冲突

### 挑战四：动态规则生效

**问题描述**：
计费规则可能在运营期间调整，需要确保调整不影响进行中的停车记录。

**解决方案**：

1. **版本控制**
   ```sql
   ALTER TABLE billing_rules ADD COLUMN version INT DEFAULT 1;
   ALTER TABLE billing_rules ADD COLUMN effective_from TIMESTAMP;
   ALTER TABLE billing_rules ADD COLUMN effective_to TIMESTAMP;
   ```

2. **快照机制**
   ```go
   // 入场时记录当前费率版本
   record.BillingRuleVersion = getCurrentRuleVersion(lotID)
   
   // 出场时使用入场时的版本计算
   rules := uc.repo.GetRulesByVersion(ctx, lotID, record.BillingRuleVersion)
   ```

3. **灰度发布**
   - 新规则先对部分车辆生效
   - 观察无误后全量发布

## API 接口文档

### 计算费用

```http
POST /api/v1/billing/calculate
Content-Type: application/json

{
  "recordId": "rec_xxx",
  "lotId": "lot_xxx",
  "entryTime": 1710916800,
  "exitTime": 1710924000,
  "vehicleType": "temporary"
}
```

**响应**：
```json
{
  "code": 0,
  "data": {
    "recordId": "rec_xxx",
    "baseAmount": 15.00,
    "discountAmount": 5.00,
    "finalAmount": 10.00,
    "appliedRules": [
      {
        "ruleId": "rule_001",
        "ruleName": "临时车标准收费",
        "amount": 15.00
      },
      {
        "ruleId": "rule_003",
        "ruleName": "夜间优惠",
        "amount": -5.00
      }
    ]
  }
}
```

### 创建计费规则

```http
POST /api/v1/admin/billing/rules
Content-Type: application/json

{
  "lotId": "lot_xxx",
  "ruleName": "工作日收费",
  "ruleType": "time",
  "conditionsJson": "{\"type\":\"vehicle_type\",\"value\":\"temporary\"}",
  "actionsJson": "[{\"type\":\"fixed\",\"amount\":5},{\"type\":\"per_hour\",\"amount\":3}]",
  "priority": 100,
  "isActive": true
}
```

## 配置说明

```yaml
# configs/billing.yaml
server:
  http:
    addr: 0.0.0.0:8002
  grpc:
    addr: 0.0.0.0:9002

data:
  database:
    driver: postgres
    source: postgres://postgres:postgres@localhost:5432/parking?sslmode=disable

billing:
  defaultCurrency: "CNY"
  minChargeAmount: 0.01
  maxChargeAmount: 10000.00
  roundDecimals: 2
```

## 监控指标

| 指标 | 类型 | 说明 |
|------|------|------|
| billing_calculate_total | Counter | 计费总次数 |
| billing_calculate_duration | Histogram | 计费耗时 |
| billing_rule_matches | Counter | 规则匹配次数（按规则ID） |
| billing_amount_histogram | Histogram | 费用分布 |

## 相关文档

- [车辆服务文档](vehicle.md)
- [支付服务文档](payment.md)
- [部署文档](deployment.md)

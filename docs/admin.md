# Admin Service 管理服务

## 模块概述

管理服务是 Smart Park 系统的运营管理中枢，提供停车场管理、车辆管理、停车记录查询、订单管理、报表统计、退款审批等功能。该服务主要面向停车场管理员和运营人员，通过 Web 管理后台提供全面的运营管理能力。

## 核心功能

### 1. 停车场管理

#### 功能特性

```go
// ParkingLot 停车场实体
type ParkingLot struct {
    ID          uuid.UUID
    Name        string          // 停车场名称
    Address     string          // 地址
    TotalSpaces int             // 总车位数
    AvailableSpaces int         // 可用车位数
    Status      string          // active/inactive/maintenance
    ContactName string          // 联系人
    ContactPhone string         // 联系电话
    BusinessHours string        // 营业时间
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

#### 管理功能

| 功能 | 说明 | 权限 |
|------|------|------|
| 创建停车场 | 录入停车场基础信息 | SUPER_ADMIN |
| 编辑停车场 | 修改停车场配置 | LOT_ADMIN |
| 删除停车场 | 软删除停车场 | SUPER_ADMIN |
| 车位管理 | 配置总车位数、分区管理 | LOT_ADMIN |
| 状态管理 | 启用/停用/维护模式 | LOT_ADMIN |

### 2. 车辆管理

#### 车辆类型

```go
const (
    VehicleTypeTemporary = "temporary"  // 临时车
    VehicleTypeMonthly   = "monthly"    // 月卡车
    VehicleTypeVIP       = "vip"        // VIP车
    VehicleTypeStaff     = "staff"      // 员工车
    VehicleTypeBlacklist = "blacklist"  // 黑名单
)
```

#### 月卡管理

```go
// Vehicle 车辆实体
type Vehicle struct {
    ID                uuid.UUID
    PlateNumber       string          // 车牌号
    VehicleType       string          // 车辆类型
    OwnerName         string          // 车主姓名
    OwnerPhone        string          // 车主电话
    MonthlyValidFrom  *time.Time      // 月卡生效日期
    MonthlyValidUntil *time.Time      // 月卡过期日期
    MonthlyFee        float64         // 月卡费用
    LotID             uuid.UUID       // 所属停车场
    Status            string          // active/inactive
    Remarks           string          // 备注
}
```

**月卡续费流程**：
```
查询月卡信息 → 选择续费时长 → 计算续费金额 → 支付 → 更新有效期
```

### 3. 停车记录管理

#### 记录查询

支持多维度查询：
- 按车牌号查询
- 按时间段查询
- 按停车场查询
- 按车辆类型查询
- 按支付状态查询

```go
// RecordQuery 记录查询参数
type RecordQuery struct {
    PlateNumber   string
    LotID         uuid.UUID
    StartTime     time.Time
    EndTime       time.Time
    VehicleType   string
    RecordStatus  string  // entry/exiting/exited/paid/refunded
    Page          int
    PageSize      int
}

// 查询实现
func (uc *AdminUseCase) QueryParkingRecords(ctx context.Context, query *RecordQuery) ([]*ParkingRecord, int64, error) {
    // 构建查询条件
    predicates := []predicate.ParkingRecord{
        parkingrecord.CreatedAtGTE(query.StartTime),
        parkingrecord.CreatedAtLTE(query.EndTime),
    }
    
    if query.PlateNumber != "" {
        predicates = append(predicates, 
            parkingrecord.PlateNumberContains(query.PlateNumber))
    }
    
    if query.LotID != uuid.Nil {
        predicates = append(predicates, 
            parkingrecord.LotID(query.LotID))
    }
    
    // 执行分页查询
    return uc.recordRepo.Query(ctx, predicates, query.Page, query.PageSize)
}
```

### 4. 订单管理

#### 订单查询与统计

```go
// OrderQuery 订单查询参数
type OrderQuery struct {
    OrderID       string
    PlateNumber   string
    LotID         uuid.UUID
    Status        string  // pending/paid/refunded/failed
    PayMethod     string  // wechat/alipay/cash
    StartTime     time.Time
    EndTime       time.Time
    Page          int
    PageSize      int
}
```

#### 订单统计报表

```go
// OrderStatistics 订单统计
type OrderStatistics struct {
    TotalOrders       int64
    TotalAmount       float64
    PaidOrders        int64
    PaidAmount        float64
    RefundedOrders    int64
    RefundedAmount    float64
    WechatOrders      int64
    WechatAmount      float64
    AlipayOrders      int64
    AlipayAmount      float64
}

// 按日统计
func (uc *AdminUseCase) GetDailyStatistics(ctx context.Context, lotID uuid.UUID, date string) (*OrderStatistics, error) {
    startTime, _ := time.Parse("2006-01-02", date)
    endTime := startTime.Add(24 * time.Hour)
    
    return uc.orderRepo.Statistics(ctx, lotID, startTime, endTime)
}
```

### 5. 报表统计

#### 日报表

```go
// DailyReport 日报表
type DailyReport struct {
    Date              string
    LotID             uuid.UUID
    LotName           string
    
    // 流量统计
    EntryCount        int64   // 入场车次
    ExitCount         int64   // 出场车次
    
    // 收入统计
    TotalRevenue      float64 // 总收入
    CashRevenue       float64 // 现金收入
    WechatRevenue     float64 // 微信收入
    AlipayRevenue     float64 // 支付宝收入
    
    // 车辆类型分布
    TemporaryCount    int64   // 临时车数量
    MonthlyCount      int64   // 月卡车数量
    VIPCount          int64   // VIP车数量
    
    // 平均停留时长
    AvgParkingDuration float64 // 分钟
}
```

#### 月报表

```go
// MonthlyReport 月报表
type MonthlyReport struct {
    Year              int
    Month             int
    LotID             uuid.UUID
    
    DailyReports      []*DailyReport
    
    // 汇总统计
    TotalEntryCount   int64
    TotalRevenue      float64
    TotalOrders       int64
    
    // 趋势分析
    PeakDay           string  // 流量最高日期
    PeakRevenueDay    string  // 收入最高日期
}
```

### 6. 退款审批

#### 退款审批流程

```
申请人提交退款申请 → 审批人审核 → 审批通过/拒绝 → 执行退款 → 更新订单状态
```

```go
// RefundApproval 退款审批
type RefundApproval struct {
    ID                uuid.UUID
    OrderID           uuid.UUID
    Applicant         string          // 申请人
    Approver          string          // 审批人
    Amount            float64         // 退款金额
    Reason            string          // 退款原因
    RefundMethod      string          // original/manual
    Status            string          // pending/approved/rejected
    ApprovedAt        *time.Time
    RejectReason      string
    CreatedAt         time.Time
}

// 创建退款申请
func (uc *AdminUseCase) CreateRefundApproval(ctx context.Context, req *CreateRefundApprovalRequest) (*RefundApproval, error) {
    // 校验订单
    order, err := uc.orderRepo.GetByID(ctx, req.OrderID)
    if err != nil {
        return nil, err
    }
    
    if order.Status != "paid" {
        return nil, fmt.Errorf("order is not paid")
    }
    
    // 创建审批记录
    approval := &RefundApproval{
        ID:           uuid.New(),
        OrderID:      req.OrderID,
        Applicant:    req.Applicant,
        Amount:       req.Amount,
        Reason:       req.Reason,
        RefundMethod: "original",
        Status:       "pending",
    }
    
    return uc.refundApprovalRepo.Create(ctx, approval)
}

// 审批退款
func (uc *AdminUseCase) ApproveRefund(ctx context.Context, approvalID uuid.UUID, approver string) error {
    approval, err := uc.refundApprovalRepo.GetByID(ctx, approvalID)
    if err != nil {
        return err
    }
    
    if approval.Status != "pending" {
        return fmt.Errorf("approval already processed")
    }
    
    // 调用支付服务执行退款
    if err := uc.paymentClient.Refund(ctx, approval.OrderID.String()); err != nil {
        return err
    }
    
    // 更新审批状态
    approval.Status = "approved"
    approval.Approver = approver
    now := time.Now()
    approval.ApprovedAt = &now
    
    return uc.refundApprovalRepo.Update(ctx, approval)
}
```

## 应用场景区景

### 场景一：商业综合体运营管理

**业务特点**：
- 多停车场统一管理
- 复杂的计费策略配置
- 会员体系管理
- 营销活动支持

**解决方案**：
- 支持多停车场数据隔离与汇总
- 灵活的计费规则配置
- VIP/月卡分级管理
- 优惠券与折扣规则

### 场景二：物业停车场管理

**业务特点**：
- 固定车辆为主（月卡车）
- 访客车辆管理
- 简单的计费规则
- 成本敏感

**解决方案**：
- 批量导入月卡车辆
- 访客预约与登记
- 标准计费模板
- 低成本部署方案

### 场景三：路边停车管理

**业务特点**：
- 无固定出入口
- 巡检员手动录入
- 欠费追缴
- 与城市交通系统对接

**解决方案**：
- 移动端巡检 App
- 手动开单功能
- 欠费车辆黑名单
- 开放 API 对接

## 技术挑战与解决方案

### 挑战一：大数据量查询性能

**问题描述**：
大型停车场每日可能产生数万条记录，报表查询可能涉及百万级数据。

**解决方案**：

1. **数据库优化**
   ```sql
   -- 复合索引
   CREATE INDEX idx_records_lot_time ON parking_records(lot_id, entry_time DESC);
   CREATE INDEX idx_orders_lot_time ON orders(lot_id, pay_time DESC);
   
   -- 分区表（按时间分区）
   CREATE TABLE parking_records_2024_01 PARTITION OF parking_records
       FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
   ```

2. **读写分离**
   ```go
   // 查询走从库
   func (r *AdminRepo) QueryRecords(ctx context.Context, query *RecordQuery) ([]*ParkingRecord, error) {
       return r.readDB.QueryContext(ctx, sql, args...)
   }
   
   // 写入走主库
   func (r *AdminRepo) CreateRecord(ctx context.Context, record *ParkingRecord) error {
       return r.writeDB.ExecContext(ctx, sql, args...)
   }
   ```

3. **报表预计算**
   ```go
   // 定时任务预计算日报表
   func (uc *AdminUseCase) PrecomputeDailyReport(ctx context.Context, date string) error {
       stats := uc.calculateDailyStats(ctx, date)
       return uc.reportRepo.SaveDailyReport(ctx, date, stats)
   }
   ```

4. **缓存策略**
   ```go
   // 热点数据缓存
   func (uc *AdminUseCase) GetLotStatistics(ctx context.Context, lotID uuid.UUID) (*LotStatistics, error) {
       key := fmt.Sprintf("parking:v1:lot_stats:%s", lotID)
       
       // 先查缓存
       if cached, err := uc.cache.Get(ctx, key); err == nil {
           return cached.(*LotStatistics), nil
       }
       
       // 缓存未命中，查数据库
       stats, err := uc.calculateLotStatistics(ctx, lotID)
       if err != nil {
           return nil, err
       }
       
       // 写入缓存（5分钟过期）
       uc.cache.Set(ctx, key, stats, 5*time.Minute)
       return stats, nil
   }
   ```

### 挑战二：权限管理

**问题描述**：
不同角色需要不同的操作权限，如超级管理员、停车场管理员、操作员、财务等。

**解决方案**：

1. **RBAC 权限模型**
   ```go
   // 角色定义
   const (
       RoleSuperAdmin = "super_admin"    // 超级管理员
       RoleLotAdmin   = "lot_admin"      // 停车场管理员
       RoleOperator   = "operator"       // 操作员
       RoleFinance    = "finance"        // 财务
       RoleViewer     = "viewer"         // 只读用户
   )
   
   // 权限定义
   var permissions = map[string][]string{
       RoleSuperAdmin: {"*"},
       RoleLotAdmin: {
           "lot:read", "lot:update",
           "vehicle:*",
           "record:read",
           "order:*",
           "report:read",
           "refund:approve",
       },
       RoleOperator: {
           "record:read",
           "vehicle:read",
           "order:read",
       },
       RoleFinance: {
           "order:*",
           "report:*",
           "refund:*",
       },
   }
   ```

2. **中间件鉴权**
   ```go
   func AuthMiddleware(permissions ...string) middleware.Middleware {
       return func(handler middleware.Handler) middleware.Handler {
           return func(ctx context.Context, req interface{}) (interface{}, error) {
               // 获取当前用户角色
               userRole := ctx.Value("user_role").(string)
               
               // 校验权限
               if !hasPermission(userRole, permissions) {
                   return nil, fmt.Errorf("permission denied")
               }
               
               return handler(ctx, req)
           }
       }
   }
   ```

### 挑战三：数据安全与审计

**问题描述**：
敏感操作需要记录审计日志，防止数据泄露和非法操作。

**解决方案**：

1. **审计日志**
   ```go
   // 审计日志实体
   type AuditLog struct {
       ID            uuid.UUID
       UserID        string
       UserName      string
       Action        string      // create/update/delete/login
       ResourceType  string      // lot/vehicle/order/rule
       ResourceID    string
       OldValue      string      // JSON
       NewValue      string      // JSON
       IPAddress     string
       UserAgent     string
       CreatedAt     time.Time
   }
   
   // 记录审计日志
   func (uc *AdminUseCase) LogAudit(ctx context.Context, action, resourceType, resourceID string, oldVal, newVal interface{}) {
       log := &AuditLog{
           ID:           uuid.New(),
           UserID:       ctx.Value("user_id").(string),
           UserName:     ctx.Value("user_name").(string),
           Action:       action,
           ResourceType: resourceType,
           ResourceID:   resourceID,
           OldValue:     toJSON(oldVal),
           NewValue:     toJSON(newVal),
           IPAddress:    ctx.Value("client_ip").(string),
           UserAgent:    ctx.Value("user_agent").(string),
           CreatedAt:    time.Now(),
       }
       
       uc.auditLogRepo.Create(ctx, log)
   }
   ```

2. **敏感数据脱敏**
   ```go
   // 手机号脱敏
   func MaskPhone(phone string) string {
       if len(phone) != 11 {
           return phone
       }
       return phone[:3] + "****" + phone[7:]
   }
   
   // 车牌号脱敏（对外展示）
   func MaskPlateNumber(plate string) string {
       if len(plate) < 7 {
           return plate
       }
       return plate[:2] + "**" + plate[4:]
   }
   ```

## API 接口文档

### 停车场管理

```http
# 创建停车场
POST /api/v1/admin/lots
Content-Type: application/json

{
  "name": "测试停车场",
  "address": "北京市朝阳区xxx",
  "totalSpaces": 500,
  "contactName": "张三",
  "contactPhone": "13800138000"
}

# 获取停车场列表
GET /api/v1/admin/lots?page=1&pageSize=20

# 获取停车场详情
GET /api/v1/admin/lots/{id}

# 更新停车场
PUT /api/v1/admin/lots/{id}
Content-Type: application/json

{
  "name": "测试停车场（更新）",
  "totalSpaces": 600
}

# 删除停车场
DELETE /api/v1/admin/lots/{id}
```

### 车辆管理

```http
# 录入车辆
POST /api/v1/admin/vehicles
Content-Type: application/json

{
  "plateNumber": "京A12345",
  "vehicleType": "monthly",
  "ownerName": "李四",
  "ownerPhone": "13900139000",
  "monthlyValidUntil": "2026-12-31",
  "lotId": "lot_xxx"
}

# 获取车辆列表
GET /api/v1/admin/vehicles?lotId=lot_xxx&vehicleType=monthly&page=1

# 更新车辆信息
PUT /api/v1/admin/vehicles/{id}

# 删除车辆
DELETE /api/v1/admin/vehicles/{id}
```

### 记录查询

```http
# 查询停车记录
GET /api/v1/admin/records?lotId=lot_xxx&startTime=2026-03-01&endTime=2026-03-31&page=1

# 查询订单
GET /api/v1/admin/orders?lotId=lot_xxx&status=paid&page=1
```

### 报表统计

```http
# 日报表
GET /api/v1/admin/reports/daily?lotId=lot_xxx&date=2026-03-20

# 月报表
GET /api/v1/admin/reports/monthly?lotId=lot_xxx&year=2026&month=3

# 收入统计
GET /api/v1/admin/reports/revenue?lotId=lot_xxx&startDate=2026-03-01&endDate=2026-03-31
```

### 退款审批

```http
# 创建退款申请
POST /api/v1/admin/orders/{orderId}/refund
Content-Type: application/json

{
  "amount": 15.00,
  "reason": "收费错误"
}

# 审批退款
POST /api/v1/admin/refund-approvals/{approvalId}/approve
Content-Type: application/json

{
  "approver": "admin_xxx"
}

# 拒绝退款
POST /api/v1/admin/refund-approvals/{approvalId}/reject
Content-Type: application/json

{
  "rejectReason": "不符合退款条件"
}
```

## 配置说明

```yaml
# configs/admin.yaml
server:
  http:
    addr: 0.0.0.0:8004
  grpc:
    addr: 0.0.0.0:9004

data:
  database:
    driver: postgres
    source: postgres://postgres:postgres@localhost:5432/parking?sslmode=disable

admin:
  defaultPageSize: 20
  maxPageSize: 100
  reportCacheTTL: 5m
  auditLogRetention: 90d  # 审计日志保留时间
```

## 监控指标

| 指标 | 类型 | 说明 |
|------|------|------|
| admin_api_requests_total | Counter | API 请求总数 |
| admin_api_duration | Histogram | API 响应耗时 |
| admin_record_query_duration | Histogram | 记录查询耗时 |
| admin_report_generation_duration | Histogram | 报表生成耗时 |
| admin_login_total | Counter | 登录次数 |
| admin_audit_log_total | Counter | 审计日志数量 |

## 相关文档

- [车辆服务文档](vehicle.md)
- [计费服务文档](billing.md)
- [支付服务文档](payment.md)
- [部署文档](deployment.md)

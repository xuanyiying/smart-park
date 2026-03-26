# Smart Park 代码重构计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 重构 vehicle 服务，解除跨服务依赖，拆分大型 UseCase，提升代码质量

**Architecture:** 
- 创建 Billing 服务客户端，通过 gRPC 调用替代直接依赖
- 拆分 VehicleUseCase 为 EntryExitUseCase、DeviceUseCase、VehicleQueryUseCase
- 提取配置化参数，移除硬编码

**Tech Stack:** Go, Kratos, Wire, gRPC

---

## 文件结构变更

### 新增文件
- `internal/vehicle/client/billing/client.go` - Billing 服务 gRPC 客户端
- `internal/vehicle/biz/entry_exit.go` - 入场出场业务逻辑
- `internal/vehicle/biz/device.go` - 设备管理业务逻辑
- `internal/vehicle/biz/vehicle_query.go` - 车辆查询业务逻辑

### 修改文件
- `internal/vehicle/biz/vehicle.go` - 删除，功能拆分
- `internal/vehicle/biz/biz.go` - 更新 ProviderSet
- `internal/vehicle/data/data.go` - 移除 BillingRuleRepo
- `internal/vehicle/data/vehicle.go` - 移除 billingRuleRepo
- `internal/vehicle/service/vehicle.go` - 更新依赖注入
- `cmd/vehicle/wire.go` - 添加客户端注入

---

## Task 1: 创建 Billing 服务客户端

**Files:**
- Create: `internal/vehicle/client/billing/client.go`

- [ ] **Step 1: 创建 Billing 客户端接口和实现**

```go
package billing

import (
	"context"

	"github.com/go-kratos/kratos/v2/log"
	billingv1 "github.com/xuanyiying/smart-park/api/billing/v1"
)

// Client defines the interface for billing service client.
type Client interface {
	CalculateFee(ctx context.Context, recordID string, lotID string, entryTime, exitTime int64, vehicleType string) (*FeeResult, error)
}

// FeeResult represents the fee calculation result.
type FeeResult struct {
	BaseAmount     float64
	DiscountAmount float64
	FinalAmount    float64
}

// billingClient implements Client interface.
type billingClient struct {
	client billingv1.BillingServiceClient
	log    *log.Helper
}

// NewClient creates a new billing client.
func NewClient(client billingv1.BillingServiceClient, logger log.Logger) Client {
	return &billingClient{
		client: client,
		log:    log.NewHelper(logger),
	}
}

// CalculateFee calculates parking fee via billing service.
func (c *billingClient) CalculateFee(ctx context.Context, recordID string, lotID string, entryTime, exitTime int64, vehicleType string) (*FeeResult, error) {
	resp, err := c.client.CalculateFee(ctx, &billingv1.CalculateFeeRequest{
		RecordId:    recordID,
		LotId:       lotID,
		EntryTime:   entryTime,
		ExitTime:    exitTime,
		VehicleType: vehicleType,
	})
	if err != nil {
		c.log.WithContext(ctx).Errorf("failed to calculate fee: %v", err)
		return nil, err
	}

	return &FeeResult{
		BaseAmount:     resp.Data.BaseAmount,
		DiscountAmount: resp.Data.DiscountAmount,
		FinalAmount:    resp.Data.FinalAmount,
	}, nil
}
```

- [ ] **Step 2: 创建 client ProviderSet**

创建 `internal/vehicle/client/client.go`:

```go
package client

import (
	"github.com/google/wire"
	"github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
)

// ProviderSet is the provider set for clients.
var ProviderSet = wire.NewSet(
	billing.NewClient,
)
```

---

## Task 2: 拆分 VehicleUseCase - 创建 EntryExitUseCase

**Files:**
- Create: `internal/vehicle/biz/entry_exit.go`

- [ ] **Step 1: 创建 EntryExitUseCase**

```go
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/xuanyiying/smart-park/internal/vehicle/client/billing"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
	"github.com/xuanyiying/smart-park/pkg/lock"

	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// EntryExitUseCase handles vehicle entry and exit business logic.
type EntryExitUseCase struct {
	vehicleRepo  VehicleRepo
	billingClient billing.Client
	mqttClient   mqtt.Client
	lockRepo     lock.LockRepo
	log          *log.Helper
}

// NewEntryExitUseCase creates a new EntryExitUseCase.
func NewEntryExitUseCase(vehicleRepo VehicleRepo, billingClient billing.Client, mqttClient mqtt.Client, lockRepo lock.LockRepo, logger log.Logger) *EntryExitUseCase {
	return &EntryExitUseCase{
		vehicleRepo:   vehicleRepo,
		billingClient: billingClient,
		mqttClient:    mqttClient,
		lockRepo:      lockRepo,
		log:           log.NewHelper(logger),
	}
}

// Entry handles vehicle entry.
func (uc *EntryExitUseCase) Entry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryData, error) {
	uc.logEntryStart(req.DeviceId, req.PlateNumber, req.Confidence)

	if req.PlateNumber == "" {
		return nil, fmt.Errorf("plate number is required")
	}

	var result *v1.EntryData
	lockKey := lock.GenerateLockKey(LockTypeEntry, req.PlateNumber)

	if err := uc.withDistributedLock(ctx, lockKey, func() error {
		return uc.vehicleRepo.WithTx(ctx, func(ctx context.Context) error {
			var err error
			result, err = uc.processEntryTransaction(ctx, req)
			return err
		})
	}); err != nil {
		return nil, err
	}

	return result, nil
}

// Exit handles vehicle exit.
func (uc *EntryExitUseCase) Exit(ctx context.Context, req *v1.ExitRequest) (*v1.ExitData, error) {
	uc.logExitStart(req.DeviceId, req.PlateNumber, req.Confidence)

	if req.PlateNumber == "" {
		return nil, fmt.Errorf("plate number is required")
	}

	var result *v1.ExitData
	lockKey := lock.GenerateLockKey(LockTypeExit, req.PlateNumber)

	if err := uc.withDistributedLock(ctx, lockKey, func() error {
		return uc.vehicleRepo.WithTx(ctx, func(ctx context.Context) error {
			var err error
			result, err = uc.processExitTransaction(ctx, req)
			return err
		})
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (uc *EntryExitUseCase) processEntryTransaction(ctx context.Context, req *v1.EntryRequest) (*v1.EntryData, error) {
	lane, err := uc.vehicleRepo.GetLaneByDeviceCode(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get lane info: %w", err)
	}
	uc.log.WithContext(ctx).Infof("[ENTRY] Found lane - LaneID: %s, LotID: %s", lane.ID, lane.LotID)

	vehicle, _ := uc.vehicleRepo.GetVehicleByPlate(ctx, req.PlateNumber)

	existingRecord, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing entry: %w", err)
	}
	if existingRecord != nil {
		uc.log.WithContext(ctx).Warnf("[ENTRY] Duplicate entry - PlateNumber: %s", req.PlateNumber)
		return &v1.EntryData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: MsgDuplicateEntry,
		}, nil
	}

	record := uc.createParkingRecord(req, lane, vehicle)
	if err := uc.vehicleRepo.CreateParkingRecord(ctx, record); err != nil {
		return nil, fmt.Errorf("failed to create parking record: %w", err)
	}

	return uc.buildEntryResponse(record, req.PlateNumber, vehicle), nil
}

func (uc *EntryExitUseCase) processExitTransaction(ctx context.Context, req *v1.ExitRequest) (*v1.ExitData, error) {
	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get device info: %w", err)
	}

	lane, err := uc.vehicleRepo.GetLaneByDeviceCode(ctx, req.DeviceId)
	if err != nil {
		return nil, fmt.Errorf("failed to get lane info: %w", err)
	}

	record, err := uc.vehicleRepo.GetEntryRecord(ctx, req.PlateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get entry record: %w", err)
	}
	if record == nil {
		return &v1.ExitData{
			PlateNumber:    req.PlateNumber,
			Allowed:        false,
			GateOpen:       false,
			DisplayMessage: MsgNoEntryRecord,
		}, nil
	}

	exitTime := time.Now()
	duration := int(exitTime.Sub(record.EntryTime).Seconds())

	if err := uc.updateParkingRecordForExit(ctx, record, req, device, lane, exitTime, duration); err != nil {
		return nil, err
	}

	vehicle, vehicleType := uc.getVehicleInfo(ctx, req.PlateNumber)
	amount, discountAmount, finalAmount := uc.calculateExitFee(ctx, record, lane, exitTime, vehicle, vehicleType)

	return uc.buildExitResponse(record, req, duration, amount, discountAmount, finalAmount), nil
}

func (uc *EntryExitUseCase) calculateExitFee(ctx context.Context, record *ParkingRecord, lane *Lane, exitTime time.Time, vehicle *Vehicle, vehicleType string) (float64, float64, float64) {
	// 通过 Billing 服务计算费用
	feeResult, err := uc.billingClient.CalculateFee(ctx, record.ID.String(), lane.LotID.String(), 
		record.EntryTime.Unix(), exitTime.Unix(), vehicleType)
	if err != nil {
		uc.log.WithContext(ctx).Errorf("[EXIT] Failed to calculate fee: %v", err)
		return 0, 0, 0
	}

	finalAmount := feeResult.FinalAmount

	// 月卡车辆免费
	if vehicle != nil && vehicle.VehicleType == VehicleTypeMonthly && 
		vehicle.MonthlyValidUntil != nil && vehicle.MonthlyValidUntil.After(time.Now()) {
		finalAmount = 0
	}

	return feeResult.BaseAmount, feeResult.DiscountAmount, finalAmount
}

// ... 其他辅助方法
```

---

## Task 3: 创建 DeviceUseCase

**Files:**
- Create: `internal/vehicle/biz/device.go`

- [ ] **Step 1: 创建设备管理 UseCase**

```go
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// DeviceUseCase handles device management business logic.
type DeviceUseCase struct {
	vehicleRepo VehicleRepo
	log         *log.Helper
}

// NewDeviceUseCase creates a new DeviceUseCase.
func NewDeviceUseCase(vehicleRepo VehicleRepo, logger log.Logger) *DeviceUseCase {
	return &DeviceUseCase{
		vehicleRepo: vehicleRepo,
		log:         log.NewHelper(logger),
	}
}

// Heartbeat handles device heartbeat.
func (uc *DeviceUseCase) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) error {
	if req.DeviceId == "" {
		return fmt.Errorf("device id is required")
	}
	if err := uc.vehicleRepo.UpdateDeviceHeartbeat(ctx, req.DeviceId); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to update heartbeat: %v", err)
		return fmt.Errorf("failed to update device heartbeat: %w", err)
	}
	return nil
}

// GetDeviceStatus retrieves device status.
func (uc *DeviceUseCase) GetDeviceStatus(ctx context.Context, deviceID string) (*v1.DeviceStatus, error) {
	if deviceID == "" {
		return nil, fmt.Errorf("device id is required")
	}

	device, err := uc.vehicleRepo.GetDeviceByCode(ctx, deviceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %w", err)
	}
	if device == nil {
		return nil, fmt.Errorf("device not found: %s", deviceID)
	}

	online := true
	if device.LastHeartbeat != nil {
		online = time.Since(*device.LastHeartbeat) < DeviceOnlineThreshold
	}

	var lastHeartbeat string
	if device.LastHeartbeat != nil {
		lastHeartbeat = device.LastHeartbeat.Format(time.RFC3339)
	}

	return &v1.DeviceStatus{
		DeviceId:      device.DeviceID,
		Online:        online,
		Status:        device.Status,
		LastHeartbeat: lastHeartbeat,
	}, nil
}
```

---

## Task 4: 创建 VehicleQueryUseCase

**Files:**
- Create: `internal/vehicle/biz/vehicle_query.go`

- [ ] **Step 1: 创建车辆查询 UseCase**

```go
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// VehicleQueryUseCase handles vehicle query business logic.
type VehicleQueryUseCase struct {
	vehicleRepo VehicleRepo
	log         *log.Helper
}

// NewVehicleQueryUseCase creates a new VehicleQueryUseCase.
func NewVehicleQueryUseCase(vehicleRepo VehicleRepo, logger log.Logger) *VehicleQueryUseCase {
	return &VehicleQueryUseCase{
		vehicleRepo: vehicleRepo,
		log:         log.NewHelper(logger),
	}
}

// GetVehicleInfo retrieves vehicle information.
func (uc *VehicleQueryUseCase) GetVehicleInfo(ctx context.Context, plateNumber string) (*v1.VehicleInfo, error) {
	if plateNumber == "" {
		return nil, fmt.Errorf("plate number is required")
	}

	vehicle, err := uc.vehicleRepo.GetVehicleByPlate(ctx, plateNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get vehicle: %w", err)
	}
	if vehicle == nil {
		return nil, fmt.Errorf("vehicle not found: %s", plateNumber)
	}

	var monthlyValidUntil string
	if vehicle.MonthlyValidUntil != nil {
		monthlyValidUntil = vehicle.MonthlyValidUntil.Format(time.RFC3339)
	}

	return &v1.VehicleInfo{
		PlateNumber:       vehicle.PlateNumber,
		VehicleType:       vehicle.VehicleType,
		OwnerName:         vehicle.OwnerName,
		OwnerPhone:        vehicle.OwnerPhone,
		MonthlyValidUntil: monthlyValidUntil,
	}, nil
}
```

---

## Task 5: 更新 biz ProviderSet

**Files:**
- Modify: `internal/vehicle/biz/biz.go`

- [ ] **Step 1: 更新 ProviderSet**

```go
package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// ProviderSet is the provider set for biz layer.
var ProviderSet = wire.NewSet(
	NewEntryExitUseCase,
	NewDeviceUseCase,
	NewVehicleQueryUseCase,
	NewCommandUseCase,
	NewLogger,
)

// NewLogger creates a new logger helper.
func NewLogger(logger log.Logger) *log.Helper {
	return log.NewHelper(logger)
}
```

---

## Task 6: 创建 CommandUseCase

**Files:**
- Create: `internal/vehicle/biz/command.go`

- [ ] **Step 1: 创建命令发送 UseCase**

```go
package biz

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"github.com/xuanyiying/smart-park/internal/vehicle/data/mqtt"
	v1 "github.com/xuanyiying/smart-park/api/vehicle/v1"
)

// CommandUseCase handles device command business logic.
type CommandUseCase struct {
	mqttClient mqtt.Client
	log        *log.Helper
}

// NewCommandUseCase creates a new CommandUseCase.
func NewCommandUseCase(mqttClient mqtt.Client, logger log.Logger) *CommandUseCase {
	return &CommandUseCase{
		mqttClient: mqttClient,
		log:        log.NewHelper(logger),
	}
}

// SendCommand sends a command to a device via MQTT.
func (uc *CommandUseCase) SendCommand(ctx context.Context, deviceID string, command string, params map[string]string) (*v1.CommandData, error) {
	cmd := &mqtt.Command{
		CommandID: uuid.New().String(),
		DeviceID:  deviceID,
		Command:   mqtt.CommandType(command),
		Params:    params,
		Timestamp: time.Now().Unix(),
		Priority:  1,
	}

	if err := uc.mqttClient.PublishCommand(ctx, cmd); err != nil {
		uc.log.WithContext(ctx).Errorf("failed to publish command: %v", err)
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	uc.log.WithContext(ctx).Infof("command sent to device %s, command_id: %s", deviceID, cmd.CommandID)

	return &v1.CommandData{
		CommandId: cmd.CommandID,
		Status:    "sent",
	}, nil
}
```

---

## Task 7: 删除旧的 vehicle.go 文件

**Files:**
- Delete: `internal/vehicle/biz/vehicle.go`

- [ ] **Step 1: 删除旧文件**

```bash
rm internal/vehicle/biz/vehicle.go
```

---

## Task 8: 移除 data 层的 BillingRuleRepo

**Files:**
- Modify: `internal/vehicle/data/data.go`
- Modify: `internal/vehicle/data/vehicle.go`

- [ ] **Step 1: 更新 data.go**

```go
var ProviderSet = wire.NewSet(
	NewData,
	NewVehicleRepo,
	// 删除: NewBillingRuleRepo
)
```

- [ ] **Step 2: 从 vehicle.go 移除 billingRuleRepo**

删除 billingRuleRepo 相关的代码（约 40 行）。

---

## Task 9: 更新 service 层

**Files:**
- Modify: `internal/vehicle/service/vehicle.go`

- [ ] **Step 1: 更新 VehicleService 结构体**

```go
type VehicleService struct {
	v1.UnimplementedVehicleServiceServer

	entryExitUseCase *biz.EntryExitUseCase
	deviceUseCase    *biz.DeviceUseCase
	vehicleUseCase   *biz.VehicleQueryUseCase
	commandUseCase   *biz.CommandUseCase
	log              *log.Helper
}

// NewVehicleService creates a new VehicleService.
func NewVehicleService(
	entryExitUseCase *biz.EntryExitUseCase,
	deviceUseCase *biz.DeviceUseCase,
	vehicleUseCase *biz.VehicleQueryUseCase,
	commandUseCase *biz.CommandUseCase,
	logger log.Logger,
) *VehicleService {
	return &VehicleService{
		entryExitUseCase: entryExitUseCase,
		deviceUseCase:    deviceUseCase,
		vehicleUseCase:   vehicleUseCase,
		commandUseCase:   commandUseCase,
		log:              log.NewHelper(logger),
	}
}
```

- [ ] **Step 2: 更新方法调用**

```go
func (s *VehicleService) Entry(ctx context.Context, req *v1.EntryRequest) (*v1.EntryResponse, error) {
	data, err := s.entryExitUseCase.Entry(ctx, req)
	// ...
}

func (s *VehicleService) Exit(ctx context.Context, req *v1.ExitRequest) (*v1.ExitResponse, error) {
	data, err := s.entryExitUseCase.Exit(ctx, req)
	// ...
}

func (s *VehicleService) Heartbeat(ctx context.Context, req *v1.HeartbeatRequest) (*v1.HeartbeatResponse, error) {
	if err := s.deviceUseCase.Heartbeat(ctx, req); err != nil {
		// ...
	}
}

func (s *VehicleService) GetDeviceStatus(ctx context.Context, req *v1.GetDeviceStatusRequest) (*v1.GetDeviceStatusResponse, error) {
	status, err := s.deviceUseCase.GetDeviceStatus(ctx, req.DeviceId)
	// ...
}

func (s *VehicleService) GetVehicleInfo(ctx context.Context, req *v1.GetVehicleInfoRequest) (*v1.GetVehicleInfoResponse, error) {
	info, err := s.vehicleUseCase.GetVehicleInfo(ctx, req.PlateNumber)
	// ...
}

func (s *VehicleService) SendCommand(ctx context.Context, req *v1.SendCommandRequest) (*v1.SendCommandResponse, error) {
	data, err := s.commandUseCase.SendCommand(ctx, req.DeviceId, req.Command, req.Params)
	// ...
}
```

---

## Task 10: 更新 wire.go

**Files:**
- Modify: `cmd/vehicle/wire.go`

- [ ] **Step 1: 添加客户端注入**

```go
import (
	// ... 现有导入
	"github.com/xuanyiying/smart-park/internal/vehicle/client"
)

func initApp(entClient *ent.Client, billingClient billingv1.BillingServiceClient, logger log.Logger) (*app, func(), error) {
	wire.Build(
		// Client layer
		client.ProviderSet,

		// Data layer
		data.ProviderSet,

		// Business layer
		biz.ProviderSet,

		// Service layer
		service.ProviderSet,

		// gRPC and HTTP servers
		grpc.NewServer,
		http.NewServer,

		// Service registration
		wire.Bind(new(v1.VehicleServiceServer), new(*service.VehicleService)),

		// App
		newApp,
	)
	return nil, nil, nil
}
```

---

## Task 11: 验证编译

- [ ] **Step 1: 运行 go mod tidy**

```bash
go mod tidy
```

- [ ] **Step 2: 生成 wire 代码**

```bash
cd cmd/vehicle && wire
```

- [ ] **Step 3: 编译验证**

```bash
go build ./cmd/vehicle
```

---

## 注意事项

1. **保持向后兼容**: 确保 proto 接口不变
2. **错误处理**: 保持原有的错误包装方式
3. **日志记录**: 保持原有的日志格式
4. **测试**: 重构后需要更新测试文件

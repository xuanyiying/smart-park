# 硬件接入就绪清单（道闸 / 相机 / MQTT）

本文档梳理"接入真实硬件前必须确认的事项"。软件侧全流程（入场 → 计费 →
支付 → 结算 → 开闸）已由 e2e 与集成测试覆盖（`tests/e2e`、`tests/integration`），
尚未被真机验证的只有设备侧协议对接。

## 1. MQTT 契约（已由代码定义，需与设备固件对齐）

| 方向 | Topic | QoS | 载荷 |
|---|---|---|---|
| 服务 → 设备（下行命令） | `smart-park/device/{deviceID}/command` | 1 | `Command` JSON（见下） |
| 设备 → 服务（命令回执） | 订阅侧按 `CommandResult` 解析 | 1 | `CommandResult` JSON |
| 服务存活遗嘱（LWT） | `smart-park/service/{clientID}/status` | 1 | `{"status":"offline"}`（retained） |

下行 `Command` 载荷结构（`internal/vehicle/data/mqtt/client.go`）：

```json
{
  "command_id": "uuid",
  "device_id":  "GATE001",
  "command":    "open_gate",
  "params":     {"record_id": "..."},
  "timestamp":  1690000000,
  "priority":   0
}
```

**接入前必须与固件确认**：
- 命令名枚举（当前服务端发送 `open_gate`；固件侧支持的命令集合、是否大小写敏感）
- QoS 与 retained 策略是否符合设备端实现
- 命令回执是否回传、超时阈值（服务端 `sendDeviceCommandWithRetry` 为 3 次指数退避重试）
- 设备认证：每台设备使用随机 `device_secret`（`dev_` 前缀 + 32 字节随机数），
  固件侧需实现对应的 MQTT 用户名/密码或 TLS 客户端证书方案

## 2. 相机/车牌识别上报契约

入场/出场由**相机侧完成车牌识别**后调用：

```
POST /api/v1/device/entry   POST /api/v1/device/exit
{
  "deviceId":      "CAM001",          // 必须与设备表 device_id 一致
  "plateNumber":   "京A12345",
  "plateImageUrl": "http://...",      // 抓拍图 URL
  "confidence":    0.95,              // 低于配置阈值(默认0.7)会被拒收
  "timestamp":     "..."              // RFC3339
}
```

- 识别置信度阈值：`vehicle` 配置 `biz.Config.MinConfidence`（默认 0.7）
- 无牌车：`plateNumber` 允许为空（记录 `vehicle_id` 为空）
- 离线补传：车辆表支持 `plate_number_source=offline`，固件需本地缓存断网期间的记录

## 3. 已就绪的降级模式（软件侧保护）

| 故障 | 行为 | 位置 |
|---|---|---|
| MQTT 未配置/不可达 | 入场照常落库，开闸命令重试 3 次失败后记 `MANUAL_REVIEW_REQUIRED` 日志，由人工放行 | `vehicle/biz/entry_exit.go` |
| billing 不可达 | 出场报"计费失败"，**不放行**（不白放车） | `vehicle/biz/entry_exit.go` |
| payment → vehicle 断连 | 支付成功只落库不开闸，启动时 Warn 日志 | `cmd/payment/main.go` |
| charging 不可达 | 充电订单结算后跳过会话确认（仅 Warn） | `payment/biz/callback.go` |

**上线前建议**：对以上 Warn/`MANUAL_REVIEW_REQUIRED` 日志配置告警
（见 `docs/20-monitoring-alerting.md`），降级发生时人工介入。

## 4. 接入真机前的检查步骤

1. **联调环境**：`docker compose -f deploy/docker-compose.yml up -d` 起 MQTT
   broker（EMQX/Mosquitto），确认 `configs/vehicle.yaml` 的 `mqtt.broker` 指向它
2. **模拟设备**：用 `mosquitto_pub` 向 `smart-park/device/GATE001/command`
   订阅验证下行命令可达；回执报文按 `CommandResult` 结构发送
3. **真机固件**：按上文契约实现命令解析与回执；先在测试停车场
   （`SeedData` 预置的 CAM001/GATE001 等设备码）上联调
4. **全链路回归**：真机在场时跑 `./scripts/test.sh --integration`，再人工
   驾车走一遍 入场 → 缴费 → 出场
5. **升级核验**：生产库升级前运行
   `./scripts/verify_billing_rule_type_upgrade.sh <库名>`（billing 枚举扩展
   无需 DDL，但以脚本核验为准）

## 5. 测试覆盖现状

| 链路 | 测试 | 层级 |
|---|---|---|
| 入场→出场→计费→建单→结算回写→无感放行 | `tests/e2e/full_flow_test.go` | 跨服务真实 HTTP/gRPC + 真实 DB/Redis |
| 充电计费/计费规则/回调幂等/租户隔离 | `tests/integration/` | 单服务真实 DB |
| 回调验签→结算→outbox→开闸 | `internal/payment/biz` 单元测试 | mock（验签需真实商户证书，无法本地集成） |
| 设备固件协议 | ❌ 待真机联调 | — |

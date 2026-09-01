# Smart Park 代码功能完整性与生产可用性审计报告

## 修复状态（审计后更新）

以下问题已修复并通过全量测试（`go build ./...`、`go vet ./...`、`go test ./...` 全绿）：

| 编号 | 修复内容 |
|---|---|
| C1 | 对账接入真实渠道查询（`wechat.Client.QueryOrder` / `alipay.Client.QueryOrder`），`checkMissingOrders` 不再返回空 |
| C2 | `processRefund` 调用真实退款接口；修正多收/少收方向错误；少收差异不再通过改订单金额抹平 |
| C3 | 微信回调改为 APIv3 平台证书验签 + AES-GCM 解密；支付宝改为全参数排序验签，哈希算法由配置声明 |
| C4 | 回调幂等下沉到数据库状态机（订单状态条件更新 + 唯一约束），替代进程内 map |
| C5 | 出场异常一律拦截并输出 `MANUAL_REVIEW_REQUIRED`；入场异常放行但强制记录待复核事件 |
| C6 | MQTT 未配置时车辆服务拒绝启动，不再静默降级 Mock |
| C7 | 五家厂商适配器通过 MQTT 命令通道真实下发指令，设备状态由心跳判定，不再硬编码 `online` |
| C8 | 网关接入 JWT 鉴权中间件并落实 `skip_paths`；WebSocket 身份取自签名 claims，不再读 URL 参数；未配置公钥拒绝启动 |
| C9 | 计费种子数据改为引擎期望的格式（条件带 `type`/`value`，动作为数组）；`free_duration`/`first_hour_free` 门槛语义修复；`time_range` 支持跨零点 |
| H3 | 出场校验记录的 `ExitStatus=paid`，已缴费车主可正常出场 |
| H4 | 分布式锁增加看门狗续期（TTL/3 间隔），临界区超时不再丢锁 |
| H10 | admin 种子数据幂等；设备密钥改为 `crypto/rand` 随机生成；种子改为按配置的 `seed_lot_id` 触发，不再绑定硬编码停车场 |
| H11 | Dockerfile 构建路径、`-conf` 配置加载、compose 路由服务名、etcd 端口全部修正；compose 补齐 SP_* 环境变量覆盖与必需变量强校验 |
| H12 | JWT 配置在网关启动时强校验；Grafana/支付密钥改为必需环境变量注入 |
| H13 | E2E 测试重写为真实断言；支付回调时间校验、车辆入场空指针等失败测试全部修复 |

**第二轮修复（继续改造）**：

| 编号 | 修复内容 |
|---|---|
| H8 | 支付订单巡检调度器落地：超时订单主动查单后关闭（`pending→failed` 条件更新），渠道已收款但回调丢失的订单自动补单入账并触发开闸；网关查询失败保留 `pending` 待重试，绝不误关；与回调共享条件更新路径，两路竞争不会双记。已接入 `cmd/payment/main.go`，间隔可配置 |
| H9 | 峰值预测改为数据驱动：阈值 = 历史均值 + 1 倍标准差（不再写死 100），概率 = 该小时/最忙小时归一化（不再除以 500），置信度随样本量变化（不再恒为 0.85）；`GetRevenueTrend` 增加空数据除零保护与 `limit` 上限；无历史数据时诚实返回空峰值与 0 置信度 |

**第三轮修复（遗留项推进）**：

| 编号 | 修复内容 |
|---|---|
| H5 | Ent 全局租户拦截器落地：新增 `pkg/tenant/interceptor.go`（查询拦截器 `TenantQueryInterceptor` + 写操作钩子 `TenantMutationHook`，无租户上下文时放行、有租户时自动注入 `tenant_id` 过滤），并注入 billing/vehicle/admin/charging/user 五个服务；拦截器把租户过滤下沉到 Ent client，仓库层不再依赖手工调用 `TenantFilter`；配套单元测试覆盖放行/注入/显式租户保留/更新限制/全局表豁免 |
| H6 | 通知渠道真实实现：`EmailNotifier` 走 SMTP（支持 465 隐式 TLS 与 587 STARTTLS，MIME 头 UTF-8 编码）；`SMSNotifier` 走 Aliyun 兼容签名网关（HMAC-SHA1 + POP 规范化，`FOR` 智能模板参数）；`WechatNotifier` 走微信公众号模板消息（access_token 带缓存）。未配置凭据时返回 `ErrChannelNotConfigured` 而非静默"queued"；配置由 `configs/notification.yaml` 的 `notification:` 段加载 |
| H7 | 评估后不引入 Seata 分布式事务（详见下），改为落地本地消息表（outbox）：新增 `pkg/outbox`（事务内写入、`FOR UPDATE SKIP LOCKED` 并发领取、指数退避重试、死信状态），接入 payment 结算链路——`MarkOrderPaid` 与 `order.settled` 事件同事务提交，dispatcher 消费事件触发开闸；撤除 seata-go 依赖、`pkg/seata` 占位包、`configs/seata.yaml` 与 compose 中的 seata-server 服务 |
| H1 | 金额 float64 全链路改为 int64 分：proto（billing/payment/vehicle/admin/analytics/charging/user 的金额字段 double→int64）、ent schema（订单/对账/退款审批/充电会话/统计表金额字段 Float→Int64）、biz 领域模型与 data 层全部改 int64 分；计费引擎内部保留 float64（费率/时长天然小数），仅在 API 边界经 `yuanToCents` 一次性转分；微信 SDK 直接使用分、支付宝侧新增 `centsToYuanString` 整数格式化；回调金额校验与对账比较全部改为整数相等；存量数据迁移脚本 `scripts/migrate_amount_to_cents.sql`（元×100 转分，幂等探测列类型） |

**Seata 决策**：审计建议 H7 引入官方 Go SDK，经评估不引入。理由：计费—支付—出场链路已由"条件更新状态机（幂等）+ 巡检补单 + 每日对账"实现最终一致性，没有需要 2PC/TCC 强一致的资金场景；seata-go 仍在 Apache 孵化期，其 AT 数据源代理不支持 PostgreSQL；部署 TC server 的高可用与运维成本远超收益。剩余缺口（本地事务与外部副作用之间的原子交接）由 outbox 覆盖。

**遗留（需外部配合或产品定义）**：H2（tiered/flat_rate 等动作的覆盖语义需产品定义）、M14 管理后台前端（独立工程，`web/` 目录已有工程骨架）。

---

## 执行摘要

本次审计针对 `/Users/yiying/dev-app/smart-park`（Go + Kratos 微服务智慧停车系统，约 470 个 `internal/` 下的 Go 文件、9 个服务入口、1 个 Next.js 前端），目标是核实功能完整性、生产可用性，并识别一切 demo/玩具级实现。

结论是：**该项目目前不具备生产上线条件**。正向信号是工程骨架扎实——`go build ./...` 与 `go vet ./...` 全部通过，微信支付/支付宝真实 SDK 已完成接入，Ent Schema 与迁移、OpenTelemetry 链路追踪、Prometheus/Grafana/Loki 可观测栈均已就位。但在这层"看起来很完整"的外壳之下，存在 9 个阻断级（Critical）问题，其中数个会直接造成资金损失或安全越权。

最严重的几项：支付对账的核心方法是注释写着"由于是模拟环境，我们简化处理"的空实现；对账自动退款分支会伪造一个 `"wechat_refund_" + uuid` 的交易号并把订单改成"已退款"，而真实退款接口从未被调用；支付回调验签在微信侧混用了 APIv2 的 MD5 方式（下单却是 APIv3），在支付宝侧只挑 5 个字段拼签串、并通过判断公钥文本里是否含字符串 "RSA2" 来决定哈希算法；生产入口在 MQTT 未配置时静默降级为 Mock 客户端；五家设备厂商适配器（海康、大华、捷顺、科拓、蓝卡）的开闸方法全部只做 `fmt.Printf`；网关服务没有任何鉴权中间件；计费规则引擎与种子数据的数据格式互不兼容，导致所有规则静默失效、实际计费永远落到硬编码的"5 元/小时 2 元"。

按业务域评估的功能完成度：支付渠道接单 85%、车辆出入场流程 60%、计费引擎 55%、多租户 50%、报表分析 45%、通知服务 20%（仅站内信可用）、设备控制 5%（纯桩）、管理后台前端 0%（`site/` 是营销落地页而非后台）、对账 10%。

## 审计方法

采用静态代码分析结合可执行的验证手段：全量编译（`go build ./...`）与静态检查（`go vet ./...`）；全量单元测试执行（`go test ./...`）并逐项复现失败用例；针对 demo 特征做全局指纹检索（`fmt.Print`、`math/rand`、`TODO/FIXME`、mock/stub、硬编码凭据、模拟/占位注释）；对关键链路（入场/出场、计费、支付回调、对账、租户隔离、网关路由、部署配置）逐文件通读并交叉验证调用方与被调用方的契约一致性。所有结论均附有 `文件:行号` 证据，未使用推测。

## 一、阻断级问题（Critical，必须修复后才能上线）

### C1 支付对账是"模拟环境"空实现

`internal/payment/biz/reconciliation.go` 中三个核心方法全部是空壳，注释直接写明是模拟：

```181:206:internal/payment/biz/reconciliation.go
func (uc *ReconciliationUseCase) reconcileWechatOrder(...) error {
	// 这里应该调用微信支付的查询接口，验证交易是否真实存在
	// 由于是模拟环境，我们简化处理
	uc.log.WithContext(ctx).Infof("对账微信订单: %s, 交易号: %s", order.ID, order.TransactionID)
	// 模拟对账成功
	return nil
}
```

`reconcileAlipayOrder`（191-197）同样只打日志后 `return nil`，`checkMissingOrders`（199-206）永远返回空切片。后果是：只要订单状态是 `paid`，对账结果恒为 `matched`（167-176 的金额比对只比较本地字段 `FinalAmount` 与 `PaidAmount`，从不与渠道对账单比对），**漏单、单边账、渠道已退款但本地未同步等情况一个都发现不了**。而 `wechat.Client.QueryOrder`（`wechat/client.go:87`）与 `alipay.Client.QueryOrder`（`alipay/client.go:109`）都已实现，属于"有轮子没装车"。

### C2 对账自动退款伪造成功并真实改账

这是本次审计中最危险的一处。`processRefund` 从未调用任何退款接口，而是拼一个假的交易号返回成功：

```355:382:internal/payment/biz/reconciliation.go
func (uc *ReconciliationUseCase) processRefund(ctx context.Context, order *Order, amount float64, reason string) (string, error) {
	// ...
	case MethodWechat:
		if uc.wechatClient != nil {
			// 调用微信退款接口
			// 由于是模拟环境，我们简化处理
			uc.log.WithContext(ctx).Infof("调用微信退款接口: 订单 %s, 交易号 %s, 金额 %.2f", ...)
			// 模拟退款成功
			return "wechat_refund_" + uuid.New().String(), nil
		}
```

而 `FixMismatchedOrders` 会拿这个假交易号把订单写成终态：

```297:317:internal/payment/biz/reconciliation.go
			refundTransactionID, err := uc.processRefund(ctx, order, refundAmount, "对账金额不匹配，自动退款")
			// ...
			order.Status = string(StatusRefunded)
			order.RefundTransactionID = refundTransactionID
			// ...
			reconciliation.Notes = "已退款，对账匹配"
```

结果是：钱一分没退给用户，系统账单却显示"已退款、对账匹配"。真实退款方法 `wechat.Client.Refund`（`wechat/client.go:139`）与 `alipay.Client.Refund`（`alipay/client.go:134`）均可用，接上即可。

### C3 支付回调验签协议不匹配、逻辑不可用

微信侧：下单用的是官方 APIv3 SDK（`wechat/client.go:14-17` 导入 `wechatpay-apiv3/wechatpay-go`），而回调验签走的是 APIv2 的 MD5 方式：

```235:248:internal/payment/biz/callback.go
func (uc *PaymentUseCase) verifyWechatSign(req *v1.WechatCallbackRequest) error {
	if uc.config == nil || uc.config.WechatKey == "" {
		return fmt.Errorf("wechat key not configured")
	}
	signData := buildWechatSignString(req)
	expectedSign := calculateMD5(signData + "&key=" + uc.config.WechatKey)
```

APIv3 的通知是 `resource` 字段经 AES-GCM 加密的密文，需用平台证书验签后再解密才能得到 `transaction_id`/`total_fee`。当前实现既无法正确验签，也拿不到解密后的真实字段，意味着 `req.TotalFee`、`req.TransactionId` 的来源完全不可信。同时 MD5 在微信支付侧已被废弃。

支付宝侧：`buildAlipaySignString`（478-488）只挑了 `trade_status`、`trade_no`、`out_trade_no`、`total_amount`、`gmt_payment` 五个字段拼接，而支付宝要求对**除 `sign`/`sign_type` 外的全部回调参数**排序后拼接。参数不全会导致验签失败，更糟的是未参与签名的字段可被攻击者自由篡改。

更离谱的是哈希算法的判定方式：

```471:476:internal/payment/biz/callback.go
func (uc *PaymentUseCase) getAlipayHashAlgorithm() crypto.Hash {
	if strings.Contains(uc.config.AlipayPublicKey, "RSA2") {
		return crypto.SHA256
	}
	return crypto.SHA1
}
```

PEM 编码的公钥文本里不可能出现 "RSA2" 字样，该分支恒为 `SHA1`。使用 RSA2（SHA256）的商户必然验签失败。

### C4 回调幂等依赖进程内存

```34:38:internal/payment/biz/callback.go
var (
	processedCallbacks = &callbackDeduplication{
		callbacks: make(map[string]time.Time),
	}
)
```

这是一个包级全局 map，既不在 Redis 也不在数据库里。多副本部署时，渠道的重复通知会被每个副本各处理一次；进程重启后去重记录全部丢失。正确做法应基于订单状态机 + 数据库条件更新（或唯一约束）实现幂等。此外 `cleanup()`（59-66）在每次 `markProcessed` 时持写锁全量遍历，回调量上来后是明显的锁竞争点，且 24 小时窗口内内存无上限增长。

### C5 出场/入场异常时"免费放行"兜底，且无人工复核

```280:300:internal/vehicle/biz/entry_exit.go
func (uc *EntryExitUseCase) createFallbackExitResponse(req *v1.ExitRequest) *v1.ExitData {
	// In fallback mode, we allow exit but mark it for manual review
	return &v1.ExitData{
		PlateNumber:    req.PlateNumber,
		Allowed:        true,
		GateOpen:       true,
		DisplayMessage: uc.config.Messages.FallbackMode,
	}
}
```

注释声称"mark it for manual review"，但函数体内**没有任何标记动作**——不写库、不发告警、不生成待复核工单，仅返回放行。触发路径包括：数据库错误（`handleEntryError:195-198`、`handleExitError:242-245`）与**计费失败**（`handleExitError:238-241`）。也就是说，只要 billing 服务抖动或超时，出口就会免费抬杆，且不产生任何可追责的记录。生产环境应改为：计费失败时保持拦截、引导人工收费并留痕。

### C6 生产入口在 MQTT 未配置时静默降级为 Mock 客户端

```145:152:cmd/vehicle/main.go
	} else {
		logHelper.Warn("mqtt config not provided, using mock client")
		mqttClient = mqtt.NewMockMQTTClient()
		if err := mqttClient.Connect(); err != nil {
			logHelper.Errorf("failed to connect mock MQTT client: %v", err)
			os.Exit(1)
		}
	}
```

Mock 客户端的 `PublishCommand` 只是睡 100 毫秒后往 channel 里塞一个成功状态：

```219:242:internal/vehicle/data/mqtt/client.go
func (c *MockMQTTClient) PublishCommand(ctx context.Context, cmd *Command) error {
	// ...
	go func() {
		time.Sleep(100 * time.Millisecond)
		c.results <- &CommandResult{
			// ...
			Status:    "delivered",
		}
	}()
	return nil
}
```

这意味着开闸指令从未真正下发，但上游收到的是成功。真机部署后会出现"系统显示已开闸、道闸纹丝不动"。配置缺失时应当直接 `os.Exit(1)`，正如代码对 billing 端点缺失所做的那样（`cmd/vehicle/main.go:178-181`）。

### C7 五家设备厂商适配器全部是 `fmt.Printf` 空壳

`internal/vehicle/device/` 下的 `lanka.go`、`hikvision.go`、`ketuo.go`、`dahua.go`、`jieshun.go` 结构完全一致，四个方法都只打印一行然后返回：

```21:48:internal/vehicle/device/lanka.go
func (a *LankaAdapter) OpenGate(ctx context.Context, deviceID string) error {
	// Lanka specific implementation
	fmt.Printf("Opening gate for Lanka device: %s\n", deviceID)
	// Implement Lanka specific gate opening logic here
	return nil
}
// ...
func (a *LankaAdapter) GetDeviceStatus(...) (map[string]interface{}, error) {
	fmt.Printf("Getting status for Lanka device: %s\n", deviceID)
	return map[string]interface{}{
		"status":     "online",
		// ...
	}, nil
}
```

`GetDeviceStatus` 无条件硬编码返回 `"online"`，导致设备心跳监控与离线告警彻底失效——设备实际已断线，系统永远显示在线。道闸、相机、显示屏三种设备类型均无真实控制协议实现。

### C8 网关零鉴权，WebSocket 可伪装任意用户与租户

`cmd/gateway/main.go` 中检索 `JWT|Auth|Middleware|RateLimit|CORS` 关键字**零命中**，即网关没有任何认证、鉴权、限流或跨域中间件。而 `configs/gateway.yaml` 里却配置了：

```21:29:configs/gateway.yaml
# JWT 认证配置
jwt:
  public_key_path: "./keys/public.pem"
  # 跳过认证的路径 (可选)
  skip_paths:
    - "/health"
    - "/ready"
    - "/api/v1/user/login"
    - "/api/v1/user/register"
```

这段配置代码从未读取，属于典型的"死配置"——会让人误以为鉴权已生效。更直接的漏洞在 WebSocket 入口：

```176:191:internal/gateway/service/gateway.go
func (s *GatewayService) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	tenantID := r.URL.Query().Get("tenant_id")

	opts := ws.UpgradeOptions{
		UserID:   userID,
		TenantID: tenantID,
	}
```

身份完全由 URL 查询参数自报，不做任何校验，任意人可订阅任意用户/租户的实时推送。

### C9 计费规则引擎与种子数据格式不兼容，规则全部静默失效

引擎侧期望 Actions 是一个数组：

```51:61:internal/billing/biz/billing.go
func ParseActions(jsonStr string) ([]*Action, error) {
	if jsonStr == "" {
		return nil, nil
	}
	var actions []*Action
	if err := json.Unmarshal([]byte(jsonStr), &actions); err != nil {
		return nil, fmt.Errorf("failed to parse actions: %w", err)
	}
```

而种子数据写入的是对象：

```32:33:internal/billing/data/seed.go
		Conditions: `{"vehicle_type": "small"}`,
		Actions:    `{"base_fee": 5, "hourly_rate": 2, "max_daily": 50}`,
```

`json.Unmarshal` 把对象解析进 `[]*Action` 必然报错，`CalculateFee` 只打一条 Warn 就跳过该规则（339-342）。条件侧同理：引擎按 `cond.Type` 分支（69-237），期望 `{"type":"vehicle_type","value":"small"}`，而种子数据的 `{"vehicle_type":"small"}` 解析后 `Type` 为空字符串，落到 `default: return false`。

双重失效的结果是 `baseAmount` 恒为 0，最终全部落到硬编码兜底：

```591:597:internal/billing/biz/billing.go
func calculateDefaultFee(hours float64) float64 {
	if hours < 1 {
		return 5
	}
	return hours * 2
}
```

即生产环境实际计费是写死的"1 小时内 5 元，超出 2 元/小时"，运营在后台配置的所有规则都不生效。仓库里已有失败测试佐证这是已知缺陷：`go test ./internal/billing/biz/` 中 `free_duration_exceeds_limit` 期望 3.33 实际 0，`first_hour_free_over_1_hour` 期望 20 实际 0。

## 二、高危问题（High）

**H1 金额全程使用 float64。** 计费的 `Action.Amount`、`baseAmount`、`finalAmount`，支付的 `Order.FinalAmount`/`PaidAmount`，回调的 `validateAmount`（`callback.go:190-200`，以 0.01 元容差比较）全部是浮点数。而微信客户端接口收的是 `amount int64`（分），元与分之间存在隐式换算风险。`ceilToDecimal`（`billing.go:600-606`）用 `int(amount*100+0.999999)` 实现向上取整，属于近似 hack（例如 1.000001 会被算成 1.00 而非 1.01）。建议全链路改用 `int64` 分或 `decimal` 库。

**H2 计费动作之间互相覆盖。** `applyActions`（`billing.go:454-589`）顺序执行，其中 `free_duration`（488-496）、`first_hour_free`（502-509）、`flat_rate`（582-584）、`tiered`（510-544）、`time_segment`（545-566）都是对 `amount` 直接赋值而非增量叠加，组合规则时前面算出的结果被整体丢弃。这正是 C9 中两个失败测试的直接根因。

**H3 出场不校验支付状态。** `EntryExitUseCase` 的字段只有 `vehicleRepo`、`billingClient`、`mqttClient`、`lockRepo`、`adapterFactory`（`entry_exit.go:49-70`），没有 payment 客户端。`buildExitResponse`（502-523）仅凭 `finalAmount == 0` 决定是否抬杆。车主即便已通过小程序扫码缴清，出场时仍会被判为未缴费、道闸不开。README 宣称的"扫码缴费、无感支付"在出场环节实际不成立。

**H4 分布式锁无续期，TTL 硬编码 10 秒。** `withDistributedLock`（`entry_exit.go:379-400`）只有 Acquire/Release，没有看门狗续期；`DefaultConfig()` 里 `LockTTL: 10 * time.Second`（`config.go:40`）。而锁保护的事务内包含跨服务 gRPC 计费调用，耗时超过 10 秒时锁已过期，第二个请求可并发进入，防重复入场/出场的保证失效。

**H5 多租户隔离靠"自觉"，租户 ID 可被客户端任意指定。** `pkg/tenant/middleware.go:22-40` 的 `HeaderExtractor` 直接信任 `X-Tenant-ID`；中间件只校验租户存在且处于 active（106-128），**不校验当前登录用户是否属于该租户**，任意已登录用户传他人租户 ID 即可跨租户读写。隔离过滤也非自动生效：`pkg/tenant/isolation.go:10-19` 的 `TenantFilter` 在 `tenantID == uuid.Nil` 时返回 nil（不加任何过滤），且需要每个查询手工调用，未使用 Ent 全局拦截器统一注入，极易漏加。网关也没有剥离客户端传入的 `X-Tenant-ID`。

**H6 通知渠道全部是日志桩。** `internal/notification/biz/channels.go:30-85` 中 `EmailNotifier`、`SMSNotifier`、`WechatNotifier` 的 `Send` 方法都只执行 `Infow("email notification queued")` 之类后 `return nil`。注释写着 "queued"，但没有任何队列、没有 SDK、没有失败重试、没有发送记录落库。只有 `InAppNotifier` 真正写库。支付成功、月卡到期、入场提醒等通知在生产环境一条都发不出去。

**H7 分布式事务是占位实现。** `pkg/seata/seata.go` 全文件只有 10 行，注释即写明 "This is a placeholder implementation"，`InitSeata` 只打一条 Warn 就 `return nil`。跨服务的"计费—支付—出场"一致性没有任何保障机制。

**H8 全项目没有任何定时任务调度。** 在 `internal/` 下检索 `ticker|cron|time.AfterFunc` 零命中（仅 gateway 服务发现用到 goroutine）。直接影响：订单超时关闭、主动查单补单、每日自动对账、月卡到期提醒，全部没有触发入口。叠加 C1 与 H6 后，资金类兜底能力整体缺失。

**H9 预测类接口返回硬编码的"AI 结果"。** `internal/analytics/biz/analytics.go:150-166` 中，峰值预测使用硬编码阈值 `count > 100`、硬编码除数 `count / 500.0` 冒充概率、`Confidence: 0.85` 写死，没有模型、没有训练、没有对历史基线的自适应。此外 `avgRevenue := totalRevenue / float64(len(points))`（128 行）在 points 为空时得到 NaN；`GetRevenueTrend` 未对 `req.Period` 与 `req.Limit` 设上限，可被构造大 Limit 打爆内存。

**H10 种子数据含可预测密钥、硬编码停车场，且 admin 种子无幂等。** `internal/vehicle/data/seed.go:79` 的 `SetDeviceSecret("secret_" + d.deviceID)` 使设备密钥可被枚举推断（如 `secret_GATE001`），设备接入认证形同虚设。`seed.go:26` 与 `billing/data/seed.go:24` 都把数据绑定到硬编码 UUID `11111111-1111-1111-1111-111111111111`，运维新建的停车场不会拥有车道、设备与计费规则。最需注意的是 admin 的种子缺少幂等判断：

```490:506:internal/admin/data/admin.go
func (r *adminRepo) SeedData(ctx context.Context) error {
	lotID := uuid.New()
	_, err := r.clientFromCtx(ctx).ParkingLot.Create().
		SetID(lotID).
		SetName("测试停车场").
		SetAddress("测试地址").
		// ...
```

vehicle 与 billing 的 `SeedData` 都有 `if count > 0 { return nil }` 幂等保护，admin 没有——**每重启一次服务就往生产库里塞一个名为"测试停车场"的停车场**。

**H11 部署配置写成不可用状态。** 四处硬伤彼此独立：其一，`deploy/docker/Dockerfile.vehicle:17` 执行 `go build -o /app/vehicle-svc ./cmd/vehicle-svc`，而 `cmd/` 目录下只有 `vehicle`（已核对目录），`docker build` 必然失败。其二，`deploy/docker-compose.yml:145` 等处传 `KRATOS_CONF=../../configs/gateway.yaml`，但代码只读 `-conf` 命令行 flag（`cmd/vehicle/main.go:37-39`），从不读该环境变量；且容器 `WORKDIR` 为 `/app`，`../../configs` 解析成 `/configs`，路径不存在，服务启动即 `os.Exit(1)`。其三，`configs/gateway.yaml:5-19` 的路由 target 全是 `localhost:800X`，在容器网络中指向网关自身，所有转发失败。其四，同文件 33 行 etcd 端点写的是 `localhost:2380`（peer 端口），客户端端口应为 2379。

**H12 配置与密钥管理不达标。** 所有 `configs/*.yaml` 均为明文弱口令：`password=postgres`、`password: ""`、`sslmode=disable`、`api_key: ""`（见 `admin.yaml:8`、`billing.yaml:8`、`vehicle.yaml:23`、`payment.yaml:29` 等）。`configs/user.yaml:37-38` 把 JWT 公私钥路径留空，导致 `pkg/auth/jwt.go:108-129` 在 `GenerateToken`/`ParseToken` 时直接报 "private key not configured"，**所有需鉴权的接口在默认配置下完全不可用**，而服务又能正常启动——属于"启动成功但功能全废"，应在启动时做强校验。`deploy/k8s/secrets.yaml` 使用 `${DB_PASSWORD}`/`${JWT_SECRET}` 占位（`kubectl apply` 不做变量替换），私钥位置是注释占位符。

**H13 测试资产本身是假的，且已有 5 个包测试失败。** `tests/e2e/e2e_test.go` 的全部 8 个用例形如：

```31:38:tests/e2e/e2e_test.go
func (s *E2ETestSuite) TestVehicleEntryFlow() {
	req := httptest.NewRequest("POST", "/api/v1/device/entry", nil)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	assert.Equal(s.T(), http.StatusOK, w.Code)
}
```

请求构造出来后**从未发给任何 handler**，`s.server` 始终为 nil，而 `httptest.NewRecorder()` 的默认 `Code` 就是 200——断言恒真，E2E 覆盖率为零。`go test ./...` 另有 5 个包失败：

| 包 | 失败用例 | 现象与根因 |
|---|---|---|
| `internal/billing/biz` | `TestApplyActions_FreeDuration/free_duration_exceeds_limit`、`first_hour_free_over_1_hour` | 期望 3.33/20，实际均为 0，即 H2 的动作覆盖缺陷 |
| `internal/payment/wechat`、`internal/payment/alipay` | `TestNewClient`、`TestConfigValidation/valid_config` | 缺少 `test_private.pem`/`valid_private.pem` 测试固件，支付链路从未真正跑通过 |
| `internal/payment/biz` | `TestValidateWechatCallbackTime/valid_recent_time`、`TestValidateAlipayCallbackTime/valid_recent_time` | 合法的最近时间被判为 "callback time is too old (>24h)"，时间处理逻辑与测试预期不符 |
| `internal/vehicle/biz` | `TestEntryExitUseCase_Entry` | `entry_exit.go:269` 因 `mqttClient` 为 nil 触发空指针 panic，生产代码缺少 nil 保护 |

## 三、中低优先级问题（Medium / Info）

**M1** `entry_exit.go:101/139` 的 `sendDeviceCommandWithRetry` 在请求线程内 `time.Sleep`（累计最多 600ms），无指数退避与整体超时；失败仅打 Warn，不告警不落库，形成"记录已放行但道闸未开"的静默不一致。

**M2** `entry_exit.go:309-313` 中 `GetVehicleByPlate` 出错时按"无车辆信息"继续，未区分 NotFound 与数据库故障。数据库抖动时月卡/VIP 判定丢失，车主被按临时车收费。

**M3** 业务配置硬编码在 Go 代码里：`DefaultConfig()` 中 `MinConfidence: 0.7`、`LockTTL: 10s` 及全部中文提示文案，且 `NewEntryExitUseCase`（`entry_exit.go:67`）直接调用 `DefaultConfig()`，配置文件中的值无法注入，运维无法调整。

**M4** `billing.go:315` 将 `IsHoliday: false` 写死，但条件引擎支持 `holiday` 类型（224-225），节假日规则永不生效，且项目内没有节假日日历数据源。

**M5** 计费不支持跨天分割：README 宣称支持"跨天计费"，实际只按总时长线性计算；`max_daily`（475-483）用 `math.Ceil(hours/24)` 近似天数而非自然日。

**M6** `time_range` 条件（159-170）用 `start <= hour <= end` 判断，跨零点时段（如 22:00-06:00）永不匹配——而种子数据里的"夜间优惠"恰好就是 22-6。

**M7** `sortRulesByPriority`（442-451）使用冒泡排序，且每次计费都从数据库拉取全部规则，无缓存。

**M8** `GetBillingRules`（678-680）对非法 UUID 直接返回空列表，掩盖调用方错误。

**M9** 网关代理健壮性：`createProxy`（73-97）未给 `ReverseProxy` 设置 Transport 超时与连接池上限；`HealthCheck`（119-149）用 `strings.Contains(err.Error(), "connection refused")` 判断存活，极其脆弱；`ReadinessProbe`（198-216）要求**所有**后端路由健康才返回 200，任一服务下线即让网关整体被摘流量，易引发雪崩。

**M10** `GetServiceTarget`（`router.go:129-135`）固定取 `instances[0].Endpoints[0]`，无负载均衡、无健康实例剔除、无熔断与重试。

**M11** MQTT 客户端细节：`SetCleanSession(true)`（82 行）导致设备离线期间指令丢失；未设置 `OnConnect` 回调，自动重连后订阅关系不会重建；`token.Wait()`（143/164 行）无超时可永久阻塞请求线程；`Disconnect`（106-116）关闭 `results` channel 后再写入会 panic；未启用 TLS、未配置遗嘱消息。

**M12** 黑名单功能完全缺失：全仓库检索 `blacklist`/`黑名单` 零命中，但 README 将其列为核心特性之一。

**M13** 无车位余量校验：入场流程（302-335）既不检查剩余车位也不做满位拒绝。

**M14** `site/` 是营销落地页而非管理后台。`site/src/lib/constants.ts` 中"万达广场""首都机场T3""京A12345"、客户证言、三档价格表、图表数据全部写死；`DashboardContent.tsx:55` 的柱状图直接渲染硬编码数组 `[40, 65, 45, 80, 55, 90, 70]`，搜索框与通知铃铛（26-30 行）没有任何事件处理。而 `README.md:206` 写着"管理后台: http://localhost:3000"。**项目目前没有任何可用的管理后台前端**，运营人员无法实际操作停车场、车辆、订单、计费规则。

**M15** README 中的性能与案例数据（99.5% 车牌识别率、1000+ 部署站点、99.9% 可用性、1000+ QPS）没有任何压测报告或监控数据支撑，属于宣传口径，不应作为交付承诺。

## 四、综合分析与修复路线

这些问题的分布呈现出一个清晰模式：**架构层（proto 定义、分层结构、依赖注入、可观测性脚手架）完成度高，而业务闭环层（真实设备交互、资金对账、异常兜底、权限校验）大面积停留在演示状态**。典型表现是"接口签名齐全、SDK 已引入、方法体为空"，以及"配置写在 yaml 里但代码从不读取"。这类代码能通过编译与静态检查，也能在单人演示中表现正常，一旦进入真实场景（多副本、渠道回调、设备故障、并发出入场）就会系统性失效。

其中三条链路的风险最高，建议按此顺序处置：

第一条是**资金链路**。C1/C2/C3/C4 叠加的后果是"渠道真实扣款、本地对账显示正常、退款声称成功但实际未退、重复回调可能重复入账"。修复要点：把 `reconcileWechatOrder`/`reconcileAlipayOrder`/`checkMissingOrders` 接到已实现的 `QueryOrder` 上；`processRefund` 改为调用 `wechat.Client.Refund` 与 `alipay.Client.Refund` 并以渠道返回为准；微信回调改用 APIv3 的平台证书验签 + AES-GCM 解密；支付宝回调按"全参数排除 sign/sign_type 后排序拼接"重建待签串，哈希算法改由配置项显式声明；回调幂等下沉到数据库状态机。同时将全链路金额从 `float64` 换成 `int64`（分）。

第二条是**设备控制与出入场链路**。C6/C7/C5 叠加意味着"道闸不会真的开、设备永远显示在线、出错就免费放行"。修复要点：MQTT 缺失时改为启动失败而非降级；厂商适配器实现真实协议（至少完成海康、大华两家主流）；`GetDeviceStatus` 返回真实心跳状态；取消"异常即免费放行"的兜底，改为拦截 + 人工处理通道 + 落库留痕；补齐出场对支付状态的校验（接入 payment 客户端，支持已缴费放行与无感支付）。

第三条是**安全与多租户链路**。C8/H5 叠加意味着"网关不鉴权、WebSocket 可伪装身份、租户 ID 可任意指定"。修复要点：网关接入 JWT 中间件并落实 `skip_paths`；WebSocket 升级前校验 token，租户与用户身份从 token 解析而非 URL 参数；租户中间件增加"用户—租户归属"校验；用 Ent 拦截器（或软删除 mixin）对所有查询自动注入 `tenant_id`，避免依赖手工调用 `TenantFilter`。

除此之外，建议把 C9（引擎与种子数据格式对齐）与 H10（admin 种子幂等 + 设备密钥随机化）放在同一批次处理，因为两者都会直接污染生产数据。部署侧（H11/H12）需要在修复后做一次真实的 `docker-compose up` 全链路冒烟，当前配置连构建都过不去。

## 五、结论

**不可上线。** 项目具备良好的工程骨架与真实的技术选型，但核心业务闭环存在系统性缺口：支付对账与退款是伪造的、设备控制是打印桩、网关没有鉴权、计费规则实际不生效、管理后台前端不存在。这些不是可以靠配置调优绕过的性能问题，而是功能本身未实现或实现错误，会直接导致资金损失、安全越权与现场运营事故。

若以"最小可用生产版本"为目标，必须完成 9 个 Critical 项与 H3、H5、H10、H11 四个 High 项；若要支撑真实的无人值守停车场运营，还需完成 H6（通知）、H8（定时调度）、H13（前端后台）与 M12/M13（黑名单、车位余量）。以当前代码规模估算，这是数个迭代的工作量，而非一次性修补。

## 六、审计局限

本次审计以静态分析为主，未连接真实数据库与中间件做运行时验证，因此以下几类问题需要后续通过集成测试或压测确认：并发场景下分布式锁的实际失效窗口、Ent 生成代码中的查询性能与 N+1 情况、网关在高并发下的代理行为、以及各服务在真实渠道沙箱环境中的端到端支付链路。此外，`internal/charging/`（充电服务，53 个文件）与 `internal/multitenancy/data/`、`internal/admin/data/` 的下沉实现仅做了抽样通读，未逐文件覆盖；`internal/analytics/data/` 的 SQL 聚合语句未逐条审查其正确性与索引使用情况。

## 附录：关键证据位置

| 编号 | 文件:行号 | 一句话摘要 |
|---|---|---|
| C1 | `internal/payment/biz/reconciliation.go:181-206` | 对账三方法为"模拟环境"空实现 |
| C2 | `internal/payment/biz/reconciliation.go:355-382` | 退款伪造交易号却把订单改成已退款 |
| C3 | `internal/payment/biz/callback.go:235-248, 471-488` | 微信 APIv2 MD5 验签、支付宝五字段拼签、"RSA2" 判定 |
| C4 | `internal/payment/biz/callback.go:34-38` | 回调幂等使用进程内全局 map |
| C5 | `internal/vehicle/biz/entry_exit.go:280-300` | 异常兜底免费放行，无人工复核 |
| C6 | `cmd/vehicle/main.go:145-152` | MQTT 未配置时降级 Mock 客户端 |
| C7 | `internal/vehicle/device/lanka.go:21-61` 等 5 个文件 | 厂商适配器仅 `fmt.Printf` |
| C8 | `internal/gateway/service/gateway.go:176-191`、`configs/gateway.yaml:21-29` | 网关无鉴权，WS 身份取自 URL 参数 |
| C9 | `internal/billing/data/seed.go:32-33`、`internal/billing/biz/billing.go:51-61, 591-597` | 种子数据格式与引擎不兼容，计费落硬编码 |
| H3 | `internal/vehicle/biz/entry_exit.go:49-70, 502-523` | 出场不校验支付状态 |
| H4 | `internal/vehicle/biz/entry_exit.go:379-400`、`config.go:40` | 锁无续期，TTL 硬编码 10s |
| H5 | `pkg/tenant/middleware.go:22-40`、`pkg/tenant/isolation.go:10-19` | 租户 ID 可伪造，隔离靠手工调用 |
| H6 | `internal/notification/biz/channels.go:30-85` | 邮件/短信/微信通知均为日志桩 |
| H7 | `pkg/seata/seata.go:5-10` | 分布式事务为占位实现 |
| H9 | `internal/analytics/biz/analytics.go:150-166` | 峰值预测硬编码阈值与置信度 |
| H10 | `internal/admin/data/admin.go:490-506`、`internal/vehicle/data/seed.go:79` | admin 种子无幂等、设备密钥可枚举 |
| H11 | `deploy/docker/Dockerfile.vehicle:17`、`deploy/docker-compose.yml:145`、`configs/gateway.yaml:5-19, 33` | 构建路径、配置路径、路由 target、etcd 端口均错 |
| H13 | `tests/e2e/e2e_test.go:31-38` | E2E 用例断言恒真，零覆盖 |
| M14 | `site/src/lib/constants.ts:249-314`、`site/src/components/preview/contents/DashboardContent.tsx:55` | 前端为营销页，数据全硬编码 |

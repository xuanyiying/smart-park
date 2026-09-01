# Smart Park 前端功能完善方案

## Context
合并充电桩服务开发分支后，后端新增了大量设备管理（故障/日志/性能/升级/厂商/固件）、支付对账等功能，但前端仍停留在基础 CRUD 阶段。同时存在旧 `src/app/dashboard/` 目录重复、设备 service 中 legacy 函数冗余等问题。需要完善前端以覆盖后端 API 能力。

## 实施计划

### 阶段1: 代码清理 + Service 基础扩展

**1.1 删除旧 dashboard 目录**
- 删除 `src/app/dashboard/` 整个目录（4 个重复页面）

**1.2 清理 device.ts**
- 移除 `listDevicesLegacy`、`getDeviceStatusLegacy`、`sendCommandLegacy`、`sendHeartbeat` 和相关废弃类型
- 扩展 `DeviceInfo` 接口，新增 `manufacturer`、`model`、`firmwareVersion`、`faultInfo`、`heartbeatCount`、`offlineCount`、`lastOnline`、`vendorSpecificConfig` 字段

**1.3 新建 `src/services/payment.ts`**
- 支付创建、状态查询、退款、日核对、核对报告、修正不匹配订单
- 新建 `src/constants/payment.ts`（核对状态/支付方式标签映射）

**1.4 扩展 `src/services/device.ts`**
- 新增接口：Manufacturer、Firmware、DeviceFault、DeviceLog、DevicePerformance、UpgradeStatusData、DeviceStatsData
- 新增 22 个 API 函数覆盖厂商/固件/故障/日志/性能/升级/配置

**1.5 扩展 `src/constants/device.ts`**
- 新增故障严重级别、故障状态、日志级别、升级状态标签映射

### 阶段2: 页面实现

**2.1 设备列表页改造** — `src/app/(dashboard)/devices/page.tsx`
- 新增厂商/固件版本/故障状态列
- 操作列增加"详情"按钮，跳转 `/devices/[id]`

**2.2 新建设备详情页** — `src/app/(dashboard)/devices/[id]/page.tsx`
- 顶部设备基本信息卡片
- Tabs 四个标签：概览（配置+统计）、故障（列表+解决操作）、日志（列表+分页）、性能（recharts AreaChart）

**2.3 新增设备升级弹窗** — `src/components/modules/device-upgrade-dialog.tsx`
- 选择固件版本 → 提交升级 → 轮询升级状态（用 react-query refetchInterval）

**2.4 新增设备配置弹窗** — `src/components/modules/device-config-dialog.tsx`
- key-value 编辑 vendorSpecificConfig → 调用 updateDeviceConfig

**2.5 扩展设备创建/编辑弹窗** — `src/components/modules/device-dialog.tsx`
- 新增厂商（Select）、型号（Input）、固件版本（Input）字段

**2.6 新建厂商管理页** — `src/app/(dashboard)/devices/manufacturers/page.tsx`
- 标准 CRUD 列表页

**2.7 新建固件管理页** — `src/app/(dashboard)/devices/firmwares/page.tsx`
- 列表页，支持按厂商/型号筛选

**2.8 新建支付对账页** — `src/app/(dashboard)/payments/page.tsx`
- 日期选择 + 执行日核对按钮
- 统计卡片（匹配/不匹配/待核对）
- 核对结果表格 + 修正操作

**2.9 新建支付订单页** — `src/app/(dashboard)/payments/orders/page.tsx`
- 订单列表，按状态筛选，退款操作

**2.10 充电监控增强** — `src/app/(dashboard)/charging/monitor/page.tsx`
- 添加"停止充电"按钮、缩短刷新间隔、实时充电时长

**2.11 充电价格微调** — `src/app/(dashboard)/charging/prices/page.tsx`
- 价格卡片增加删除按钮、日期格式化

**2.12 Dashboard 增强** — `src/app/(dashboard)/page.tsx` + `src/components/modules/trend-chart.tsx`
- 趋势区域改为双列：左列 TrafficBarChart 柱状图、右列 RevenueTrendChart 面积图
- 并发请求替代串行循环

### 阶段3: 导航更新

**3.1 更新侧边栏** — `src/components/layout/sidebar.tsx`
- "设备控制"改为含子菜单的组：设备列表、厂商管理、固件管理
- 新增"支付管理"菜单：支付订单、支付对账

## 关键文件

| 文件 | 操作 |
|------|------|
| `src/services/device.ts` | 大幅扩展 |
| `src/services/payment.ts` | 新建 |
| `src/constants/device.ts` | 扩展 |
| `src/constants/payment.ts` | 新建 |
| `src/app/(dashboard)/devices/page.tsx` | 改造 |
| `src/app/(dashboard)/devices/[id]/page.tsx` | 新建 |
| `src/app/(dashboard)/devices/manufacturers/page.tsx` | 新建 |
| `src/app/(dashboard)/devices/firmwares/page.tsx` | 新建 |
| `src/app/(dashboard)/payments/page.tsx` | 新建 |
| `src/app/(dashboard)/payments/orders/page.tsx` | 新建 |
| `src/components/modules/device-upgrade-dialog.tsx` | 新建 |
| `src/components/modules/device-config-dialog.tsx` | 新建 |
| `src/components/modules/device-dialog.tsx` | 扩展 |
| `src/components/modules/trend-chart.tsx` | 扩展 |
| `src/components/layout/sidebar.tsx` | 更新 |
| `src/app/(dashboard)/page.tsx` | 增强 |
| `src/app/(dashboard)/charging/monitor/page.tsx` | 增强 |
| `src/app/(dashboard)/charging/prices/page.tsx` | 微调 |
| `src/app/dashboard/` | 删除 |

## 验证方式
1. `pnpm build` 编译通过
2. 各页面可正常加载、无 TypeScript 错误
3. 侧边栏导航正确显示新菜单
4. 设备详情页四个 Tab 按需加载数据
5. 支付对账页执行核对后刷新列表

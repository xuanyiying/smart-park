# 充电管理与计费功能 Spec

## Why
Smart Park 系统已定义了充电服务的 API 和后端基础实现，但缺少前端管理界面和完整的计费集成。需要添加充电桩管理、充电会话监控、充电价格配置等功能，以及与支付系统的集成，使停车场能够管理电动汽车充电服务并收取相应费用。

## What Changes
- 添加充电桩管理前端页面（列表、新增、编辑、详情）
- 添加充电枪管理功能
- 添加充电会话监控页面
- 添加充电价格配置页面（分时电价、服务费）
- 添加充电订单管理页面
- 完善充电计费与支付系统集成
- 在 Dashboard 添加充电统计卡片
- 添加充电相关导航菜单

## Impact
- Affected specs: 前端路由、API 客户端、计费系统、支付系统
- Affected code:
  - `web/src/app/(dashboard)/charging/**` - 新增充电管理页面
  - `web/src/services/charging.ts` - 充电服务 API
  - `web/src/components/layout.tsx` - 导航菜单更新
  - `internal/charging/biz/charging.go` - 计费逻辑完善
  - `internal/charging/service/charging.go` - 服务接口完善
  - `api/payment/v1/payment.proto` - 充电支付接口扩展

## ADDED Requirements

### Requirement: 充电桩管理
系统应提供充电桩的完整 CRUD 管理功能。

#### Scenario: 查看充电桩列表
- **WHEN** 管理员访问充电桩管理页面
- **THEN** 系统显示所有充电桩列表，包含名称、类型、状态、可用枪数等信息

#### Scenario: 新增充电桩
- **WHEN** 管理员填写充电桩信息并提交
- **THEN** 系统创建充电桩记录并刷新列表

#### Scenario: 编辑充电桩
- **WHEN** 管理员修改充电桩信息并保存
- **THEN** 系统更新充电桩记录

#### Scenario: 查看充电桩详情
- **WHEN** 管理员点击充电桩名称
- **THEN** 系统显示充电桩详情，包含充电枪列表

### Requirement: 充电枪管理
系统应支持管理充电桩下的充电枪。

#### Scenario: 查看充电枪列表
- **WHEN** 管理员查看充电桩详情
- **THEN** 系统显示该充电桩下的所有充电枪及状态

#### Scenario: 新增充电枪
- **WHEN** 管理员为充电桩添加充电枪
- **THEN** 系统创建充电枪记录

### Requirement: 充电价格配置
系统应支持配置充电电价和服务费。

#### Scenario: 配置分时电价
- **WHEN** 管理员设置不同时段的电价
- **THEN** 系统保存价格配置并在计费时应用

#### Scenario: 配置服务费
- **WHEN** 管理员设置充电服务费
- **THEN** 系统在充电费用计算时包含服务费

### Requirement: 充电会话监控
系统应提供充电会话的实时监控功能。

#### Scenario: 查看活跃会话
- **WHEN** 管理员访问充电监控页面
- **THEN** 系统显示所有正在充电的会话

#### Scenario: 查看历史会话
- **WHEN** 管理员筛选历史记录
- **THEN** 系统显示已完成/取消的充电会话

### Requirement: 充电订单管理
系统应提供充电订单的查询和管理功能。

#### Scenario: 查看充电订单
- **WHEN** 管理员访问充电订单页面
- **THEN** 系统显示所有充电订单及支付状态

#### Scenario: 查看订单详情
- **WHEN** 管理员点击订单
- **THEN** 系统显示充电详情、费用明细、支付信息

### Requirement: 充电计费集成
系统应在充电结束时自动计算费用并生成支付订单。

#### Scenario: 充电结束计费
- **WHEN** 用户停止充电
- **THEN** 系统根据充电量、电价、服务费计算总费用

#### Scenario: 生成支付订单
- **WHEN** 费用计算完成
- **THEN** 系统创建支付订单并返回支付参数

#### Scenario: 支付确认
- **WHEN** 用户完成支付
- **THEN** 系统更新订单状态并通知充电服务

### Requirement: Dashboard 充电统计
Dashboard 应展示充电相关的统计数据。

#### Scenario: 查看充电统计
- **WHEN** 管理员访问 Dashboard
- **THEN** 系统显示今日充电次数、充电量、充电收入等统计

## MODIFIED Requirements

### Requirement: 导航菜单
导航菜单应添加充电管理相关入口。

**修改内容**:
- 添加"充电管理"菜单组
- 包含：充电桩、充电监控、充电订单、价格配置

### Requirement: 支付服务
支付服务应支持充电订单的支付处理。

**修改内容**:
- 扩展支付接口支持充电订单类型
- 添加充电订单查询接口

## REMOVED Requirements
无

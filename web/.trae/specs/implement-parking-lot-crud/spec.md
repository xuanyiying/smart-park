# 停车场管理 CRUD 功能规范

## Why
当前停车场管理页面仅有基础列表展示，缺少完整的增删改查功能，用户无法通过界面管理停车场数据。

## What Changes
- 创建停车场弹窗组件，支持新增和编辑模式
- 改造列表页面，使用 React Query hooks 并集成 CRUD 操作
- 实现详情页面，展示完整信息并提供编辑入口

## Impact
- Affected code:
  - `src/components/modules/parking-lot-dialog.tsx` (新建)
  - `src/app/(dashboard)/parking-lots/page.tsx` (修改)
  - `src/app/(dashboard)/parking-lots/[id]/page.tsx` (修改)

## ADDED Requirements

### Requirement: 停车场弹窗组件
系统应提供停车场新增/编辑弹窗组件，包含以下功能：
- 表单字段：name (名称)、address (地址)、lanes (车道数)
- 支持新增模式和编辑模式
- 编辑模式下回显现有数据
- 表单验证：name、address 必填，lanes 必须为正整数

#### Scenario: 新增停车场成功
- **WHEN** 用户点击新增按钮，填写有效数据并提交
- **THEN** 调用 createParkingLot API，成功后关闭弹窗、刷新列表、显示成功提示

#### Scenario: 编辑停车场成功
- **WHEN** 用户点击编辑按钮，修改数据并提交
- **THEN** 调用 updateParkingLot API，成功后关闭弹窗、刷新数据、显示成功提示

#### Scenario: 表单验证失败
- **WHEN** 用户提交时 name 或 address 为空，或 lanes 不是正整数
- **THEN** 显示验证错误提示，阻止提交

### Requirement: 停车场列表页面
系统应提供完整的停车场列表管理功能：
- 使用 useApiQuery 获取列表数据
- 新增按钮触发弹窗
- 每行提供编辑和查看详情操作
- 使用 useApiMutation 处理提交

#### Scenario: 加载列表
- **WHEN** 页面加载时
- **THEN** 自动获取并展示停车场列表

#### Scenario: 操作成功反馈
- **WHEN** 新增或编辑操作成功
- **THEN** 刷新列表数据并显示 toast 成功提示

### Requirement: 停车场详情页面
系统应提供停车场详情展示页面：
- 根据 URL 参数 id 获取停车场详情
- 展示完整信息（名称、地址、车道数、状态、创建/更新时间）
- 提供编辑按钮和返回列表按钮

#### Scenario: 查看详情
- **WHEN** 用户从列表点击查看详情
- **THEN** 显示该停车场的完整信息

#### Scenario: 从详情页编辑
- **WHEN** 用户在详情页点击编辑
- **THEN** 打开编辑弹窗，回显当前数据

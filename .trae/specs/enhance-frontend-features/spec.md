# 前端功能完善 Spec

## Why
前端项目处于原型阶段，缺少认证机制、CRUD 操作未实现、部分页面使用 mock 数据，无法投入生产使用。需要完善核心功能以满足运营管理需求。

## What Changes
- 添加 API 客户端认证 token 处理和请求拦截器
- 实现登录页面和认证流程
- 替换 Dashboard、设备、报表页面的 mock 数据为真实 API
- 实现停车场/车辆/计费规则的 CRUD 操作
- 添加 next.config.ts API 代理配置
- 使用 react-query 管理数据请求和缓存
- 实现 Dashboard 图表组件
- 添加全局错误边界和 Toast 提示

## Impact
- Affected specs: 认证系统、数据管理、用户交互
- Affected code: 
  - `web/src/lib/api.ts` - API 客户端重构
  - `web/src/services/*.ts` - 服务层完善
  - `web/src/app/(dashboard)/**/*.tsx` - 页面组件
  - `web/src/components/` - 新增通用组件
  - `web/next.config.ts` - 代理配置

## ADDED Requirements

### Requirement: 认证系统
系统应提供完整的认证机制，包括登录、token 管理、自动刷新和登出。

#### Scenario: 用户登录
- **WHEN** 用户输入正确的凭证并提交
- **THEN** 系统返回 token 并存储到 localStorage，跳转到 Dashboard

#### Scenario: Token 过期
- **WHEN** API 返回 401 错误
- **THEN** 系统自动清除 token 并跳转到登录页

### Requirement: CRUD 操作
系统应提供完整的数据增删改查功能。

#### Scenario: 新增停车场
- **WHEN** 管理员填写停车场信息并提交
- **THEN** 系统创建停车场并刷新列表

#### Scenario: 编辑计费规则
- **WHEN** 管理员修改规则并保存
- **THEN** 系统更新规则并显示成功提示

### Requirement: 数据可视化
Dashboard 应展示真实统计数据和趋势图表。

#### Scenario: 加载 Dashboard
- **WHEN** 用户访问 Dashboard 页面
- **THEN** 系统从 API 获取统计数据并渲染图表

### Requirement: 错误处理
系统应提供统一的错误处理和用户反馈机制。

#### Scenario: API 请求失败
- **WHEN** API 请求返回错误
- **THEN** 系统显示 Toast 错误提示，不中断用户操作

## MODIFIED Requirements

### Requirement: API 客户端
API 客户端应支持认证 token 自动注入、错误统一处理和请求重试。

### Requirement: 设备管理页面
设备管理页面应调用真实 API 获取设备列表和状态，而非使用 mock 数据。

### Requirement: 报表统计页面
报表统计页面应调用后端报表 API 获取真实数据，支持日期范围筛选。

## REMOVED Requirements

### Requirement: Mock 数据
**Reason**: 所有 mock 数据应替换为真实 API 调用
**Migration**: 删除各页面中的 mock 数据定义，调用对应 API

# Tasks

- [x] Task 1: 创建充电服务前端 API 客户端
  - [x] SubTask 1.1: 创建 `web/src/services/charging.ts` 充电服务 API
  - [x] SubTask 1.2: 定义充电相关 TypeScript 类型
  - [x] SubTask 1.3: 添加充电 API 到 api.ts 统一导出

- [x] Task 2: 创建充电桩管理页面
  - [x] SubTask 2.1: 创建 `web/src/app/(dashboard)/charging/stations/page.tsx` 充电桩列表页
  - [x] SubTask 2.2: 实现新增充电桩弹窗组件
  - [x] SubTask 2.3: 实现编辑充电桩功能
  - [x] SubTask 2.4: 创建 `web/src/app/(dashboard)/charging/stations/[id]/page.tsx` 充电桩详情页

- [x] Task 3: 创建充电枪管理组件
  - [x] SubTask 3.1: 创建充电枪列表组件
  - [x] SubTask 3.2: 实现新增充电枪功能
  - [x] SubTask 3.3: 实现充电枪状态显示

- [x] Task 4: 创建充电价格配置页面
  - [x] SubTask 4.1: 创建 `web/src/app/(dashboard)/charging/prices/page.tsx` 价格配置页
  - [x] SubTask 4.2: 实现分时电价配置表单
  - [x] SubTask 4.3: 实现价格配置列表展示

- [x] Task 5: 创建充电监控页面
  - [x] SubTask 5.1: 创建 `web/src/app/(dashboard)/charging/monitor/page.tsx` 充电监控页
  - [x] SubTask 5.2: 实现活跃会话列表
  - [x] SubTask 5.3: 实现会话状态实时显示

- [x] Task 6: 创建充电订单管理页面
  - [x] SubTask 6.1: 创建 `web/src/app/(dashboard)/charging/orders/page.tsx` 充电订单页
  - [x] SubTask 6.2: 实现订单列表及筛选
  - [x] SubTask 6.3: 实现订单详情弹窗

- [x] Task 7: 更新导航菜单
  - [x] SubTask 7.1: 在 `web/src/components/layout/sidebar.tsx` 添加充电管理菜单组
  - [x] SubTask 7.2: 添加菜单图标和路由配置

- [x] Task 8: Dashboard 添加充电统计
  - [x] SubTask 8.1: 在 Dashboard 添加充电统计卡片
  - [x] SubTask 8.2: 集成充电统计数据 API

- [x] Task 9: 后端充电计费完善
  - [x] SubTask 9.1: 完善充电费用计算逻辑（分时电价和服务费配置）
  - [x] SubTask 9.2: 添加充电订单创建接口（ListAllSessions）
  - [x] SubTask 9.3: 实现充电支付回调处理

- [x] Task 10: 支付系统集成
  - [x] SubTask 10.1: 扩展支付 proto 支持充电订单
  - [x] SubTask 10.2: 实现充电订单支付创建
  - [x] SubTask 10.3: 实现充电支付状态同步

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 2]
- [Task 4] depends on [Task 1]
- [Task 5] depends on [Task 1]
- [Task 6] depends on [Task 1]
- [Task 7] depends on [Task 2, Task 4, Task 5, Task 6]
- [Task 8] depends on [Task 1]
- [Task 9] depends on [Task 1]
- [Task 10] depends on [Task 9]

# Tasks

- [x] Task 1: 重构 API 客户端，添加认证和错误处理
  - [x] SubTask 1.1: 添加 token 自动注入和 401 处理
  - [x] SubTask 1.2: 添加请求/响应拦截器
  - [x] SubTask 1.3: 添加 next.config.ts API 代理配置

- [x] Task 2: 实现认证系统和登录页面
  - [x] SubTask 2.1: 创建 auth store (zustand)
  - [x] SubTask 2.2: 创建登录页面 `/login`
  - [x] SubTask 2.3: 添加认证中间件保护路由

- [x] Task 3: 配置 React Query 并封装 hooks
  - [x] SubTask 3.1: 配置 QueryClient Provider
  - [x] SubTask 3.2: 创建通用 hooks (useQuery, useMutation)

- [x] Task 4: 添加全局 UI 组件
  - [x] SubTask 4.1: 添加 Toast 组件 (sonner)
  - [x] SubTask 4.2: 添加错误边界组件

- [x] Task 5: 完善 Dashboard 页面
  - [x] SubTask 5.1: 创建统计 API 服务
  - [x] SubTask 5.2: 替换 mock 数据为真实 API
  - [x] SubTask 5.3: 实现趋势图表组件 (recharts)

- [x] Task 6: 完善停车场管理页面
  - [x] SubTask 6.1: 实现新增停车场弹窗
  - [x] SubTask 6.2: 实现编辑停车场功能
  - [x] SubTask 6.3: 完善停车场详情页

- [x] Task 7: 完善车辆管理页面
  - [x] SubTask 7.1: 实现新增车辆弹窗
  - [x] SubTask 7.2: 实现编辑车辆功能

- [x] Task 8: 完善计费规则页面
  - [x] SubTask 8.1: 实现新增计费规则弹窗
  - [x] SubTask 8.2: 实现编辑计费规则功能
  - [x] SubTask 8.3: 实现删除计费规则功能

- [x] Task 9: 完善设备控制页面
  - [x] SubTask 9.1: 创建设备列表 API 服务
  - [x] SubTask 9.2: 替换 mock 数据为真实 API

- [x] Task 10: 完善报表统计页面
  - [x] SubTask 10.1: 调用真实报表 API
  - [x] SubTask 10.2: 实现日期范围筛选功能
  - [x] SubTask 10.3: 实现导出功能

- [x] Task 11: 完善用户管理页面
  - [x] SubTask 11.1: 创建管理后台用户服务
  - [x] SubTask 11.2: 实现新增/编辑/删除用户功能

# Task Dependencies
- [Task 2] depends on [Task 1]
- [Task 3] depends on [Task 1]
- [Task 5] depends on [Task 3]
- [Task 6] depends on [Task 3, Task 4]
- [Task 7] depends on [Task 3, Task 4]
- [Task 8] depends on [Task 3, Task 4]
- [Task 9] depends on [Task 3]
- [Task 10] depends on [Task 3]
- [Task 11] depends on [Task 3, Task 4]

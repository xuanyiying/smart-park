# Tasks

- [ ] Task 1: 创建停车场弹窗组件
  - [ ] SubTask 1.1: 创建 `src/components/modules/parking-lot-dialog.tsx` 文件
  - [ ] SubTask 1.2: 实现 Dialog 组件集成，包含表单字段 name、address、lanes
  - [ ] SubTask 1.3: 实现新增和编辑两种模式的数据初始化逻辑
  - [ ] SubTask 1.4: 添加表单验证（name、address 必填，lanes 正整数）
  - [ ] SubTask 1.5: 集成 useApiMutation 处理提交逻辑

- [ ] Task 2: 改造停车场列表页面
  - [ ] SubTask 2.1: 将手动 fetch 改为 useApiQuery 获取列表
  - [ ] SubTask 2.2: 集成停车场弹窗组件，绑定新增按钮点击事件
  - [ ] SubTask 2.3: 添加编辑操作，传递数据到弹窗
  - [ ] SubTask 2.4: 添加查看详情链接跳转

- [ ] Task 3: 实现停车场详情页面
  - [ ] SubTask 3.1: 使用 useApiQuery 根据 id 获取详情
  - [ ] SubTask 3.2: 展示完整停车场信息（名称、地址、车道数、状态、时间）
  - [ ] SubTask 3.3: 添加编辑按钮触发弹窗
  - [ ] SubTask 3.4: 添加返回列表按钮

# Task Dependencies
- Task 2 依赖 Task 1（列表页需要使用弹窗组件）
- Task 3 依赖 Task 1（详情页需要使用弹窗组件）
- Task 1 可独立开始

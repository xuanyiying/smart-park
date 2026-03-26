# Smart Park - 待完善功能清单

> 最后更新: 2026-03-23

## 概述

本文档追踪项目中待实现或待完善的功能，用于指导开发优先级。

---

## 🔴 高优先级 - MVP 必需

### 1. Repository 层实现

**负责人**: 后端
**状态**: ✅ 已完成

**已实现**:
- [x] vehicle data 层 - vehicle.go (347行)
- [x] billing data 层 - billing.go (157行)
- [x] payment data 层 - payment.go (已修复)
- [x] admin data 层 - admin.go (454行，已修复)
- [x] Proto 生成 - 所有服务.proto已编译为.go
- [x] third_party - 下载google api proto定义
- [x] 修复 ent 类型与 biz 层类型不匹配
- [x] 修复 Desc 函数引用
- [x] 修复 predicate 使用方式
- [x] 修复 cmd 服务启动配置 (grpc.Address, http.Address)
- [x] 移除未生成的 HTTP server 注册 (proto无http标注)
- [x] 安装 etcd registry 依赖

### 2. 车主端 API

**负责人**: 后端
**状态**: 🔲 待实现

- [ ] Proto 定义创建
  - [ ] user.proto - 车主端服务定义

- [ ] Service 实现
  - [ ] 用户认证 (JWT)
  - [ ] 车牌绑定/解绑
  - [ ] 停车记录查询
  - [ ] 扫码支付 (基于 openId)
  - [ ] 月卡信息查询
  - [ ] 月卡购买/续费
  - [ ] 订单管理

- [ ] Gateway 路由配置
  - [ ] 添加 `/api/v1/user` → user-svc 路由

### 3. 支付 SDK 集成

**负责人**: 后端
**状态**: 🔲 部分完成

- [ ] 微信支付
  - [x] Proto 定义
  - [x] Service 占位
  - [ ] SDK 集成
  - [ ] 回调验签
  - [ ] 沙箱/生产环境切换

- [ ] 支付宝
  - [x] Proto 定义
  - [x] Service 占位
  - [ ] SDK 集成
  - [ ] 回调验签
  - [ ] 沙箱/生产环境切换

### 4. 设备控制服务

**负责人**: 后端
**状态**: 🔲 占位实现

- [ ] MQTT/WebSocket 通信
  - [ ] MQTT Broker 连接
  - [ ] 指令发布/订阅
  - [ ] 心跳维护

- [ ] 指令下发
  - [ ] 开闸指令
  - [ ] 关闸指令
  - [ ] 语音播报
  - [ ] LED 显示

- [ ] 离线模式
  - [ ] 本地 SQLite 缓存
  - [ ] 网络恢复同步
  - [ ] 欠费记录

---

## 🟡 中优先级 - 完整功能

### 5. 计费引擎完善

**负责人**: 后端
**状态**: 🔲 简化实现

- [ ] 规则配置解析
  - [ ] Conditions JSON 解析引擎
  - [ ] Actions JSON 执行器
  - [ ] 条件匹配 (==, !=, >, <, between, in)

- [ ] 计费规则类型
  - [x] time - 按时计费
  - [x] vip - VIP折扣
  - [x] monthly - 月卡
  - [ ] period - 时段优惠 (夜间)
  - [ ] coupon - 优惠券
  - [ ] 规则叠加逻辑

- [ ] 封顶逻辑完善
  - [ ] 按天封顶
  - [ ] 封顶次数计算
  - [ ] 超出封顶费用

### 6. 退款审批流

**负责人**: 后端
**状态**: 🔲 Schema就绪

- [ ] 退款服务集成
  - [ ] RefundApproval 表关联
  - [ ] 审批状态流转
  - [ ] 审批历史记录

- [ ] 权限控制
  - [ ] LOT_ADMIN 角色
  - [ ] 退款金额限制
  - [ ] 审批时效

- [ ] 审计日志
  - [ ] 申请记录
  - [ ] 审批记录
  - [ ] 执行记录

### 7. 通知服务

**负责人**: 后端
**状态**: 🔲 占位服务

- [ ] 短信通知
  - [ ] 阿里云/腾讯云短信 SDK
  - [ ] 通知模板
  - [ ] 发送队列

- [ ] 微信消息推送
  - [ ] 模板消息
  - [ ] 小程序订阅消息

- [ ] 通知触发
  - [ ] 入场通知
  - [ ] 出场待缴费通知
  - [ ] 支付成功通知
  - [ ] 月卡到期提醒

---

## 🟢 低优先级 - 增强功能

### 8. 缓存层

**负责人**: 后端
**状态**: 🔲 待规划

- [ ] Redis 集成
  - [ ] 车辆信息缓存
  - [ ] 计费规则缓存
  - [ ] 分布式锁

- [ ] 缓存策略
  - [ ] L1/L2/L3 分层
  - [ ] Key 版本管理
  - [ ] 主动 invalidate

### 9. 消息队列

**负责人**: 后端
**状态**: 🔲 架构设计完成

- [ ] Redis Streams 实现
  - [ ] 入场事件
  - [ ] 出场事件
  - [ ] 支付事件

- [ ] 消费者组
  - [ ] 计费消费组
  - [ ] 通知消费组
  - [ ] 数据同步消费组

### 10. 监控告警

**负责人**: 运维
**状态**: 🔲 待规划

- [ ] 指标采集
  - [ ] Prometheus metrics
  - [ ] 业务指标
  - [ ] 系统指标

- [ ] 日志聚合
  - [ ] Loki + Grafana
  - [ ] 日志格式规范
  - [ ] 链路追踪 (Jaeger)

- [ ] 告警规则
  - [ ] 服务可用性
  - [ ] 支付成功率
  - [ ] OCR识别率

### 11. 前端开发

**负责人**: 前端
**状态**: 🔲 待开始

- [ ] 管理后台
  - [ ] 停车场管理
  - [ ] 车辆管理
  - [ ] 订单管理
  - [ ] 数据报表
  - [ ] 计费配置

- [ ] 车主小程序
  - [ ] 车牌管理
  - [ ] 停车记录
  - [ ] 在线缴费
  - [ ] 月卡购买

---

## 📋 完成的功能

### 已完成 ✅

- [x] 项目框架搭建 (Kratos)
- [x] Proto API 定义
  - [x] vehicle.proto
  - [x] billing.proto
  - [x] payment.proto
  - [x] admin.proto
- [x] Proto 编译
  - [x] 生成 *_pb.go 文件
  - [x] 生成 *_grpc_pb.go 文件
  - [x] 生成 *_pb.gw.go 文件 (HTTP gateway)
- [x] Business Logic 层 (biz)
  - [x] vehicle useCase
  - [x] billing useCase
  - [x] payment useCase
  - [x] admin useCase
- [x] Service 层 (service)
  - [x] gRPC service 实现
- [x] Data 层 (data) - 大部分完成，待修复类型匹配
- [x] Gateway 路由服务 - 基础功能完成
- [x] 数据库 Schema 设计 (Ent) - 已生成
- [x] Docker Compose 部署配置
- [x] 架构文档编写 (parking-system-arch.md)
- [x] 功能对照表文档 (parking-system-arch.md 附录L)
- [x] 项目结构优化
  - [x] cmd/*/wire_gen.go 清理重复声明
  - [x] internal/conf 包创建
  - [x] pkg/mq/adapter.go 修复
  - [x] pkg/logger/logger.go 修复
  - [x] pkg/trace/trace.go 修复

---

## 🐛 已知问题

1. **Ent类型与Biz层类型不匹配** - 需要调整ent生成的类型或转换逻辑
2. **支付回调未验签** - 直接更新订单状态，存在安全隐患
3. **设备指令未下发** - SendCommand 仅记录日志
4. **车主端缺失** - 用户认证、车牌绑定、扫码支付未实现
5. **Redis/Etcd 未集成** - 服务发现使用静态配置
6. **Wire依赖注入** - 部分服务wire_gen.go有冲突

---

## 📝 版本历史

| 版本 | 日期 | 更新内容 |
|------|------|----------|
| v0.1 | 2026-03-20 | 项目初始化，基础架构搭建 |
| v0.2 | 2026-03-23 | 功能对照表文档创建，Repository层大部分实现 |
| v0.3 | 2026-03-23 | Proto编译成功，修复多个pkg包问题 |

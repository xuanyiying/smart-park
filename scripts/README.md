# Smart Park 开发脚本

本目录包含用于开发环境的一键启动和管理脚本。

## 开发环境模式（推荐）

### 快速启动开发环境

开发环境使用**单体应用模式**，所有服务集成在一个进程中，简化开发流程。

```bash
# 启动开发环境（自动构建并启动）
./scripts/start-dev.sh

# 重新构建并启动
./scripts/start-dev.sh --rebuild

# 停止开发环境
./scripts/stop-dev.sh

# 停止开发环境（包括基础设施）
./scripts/stop-dev.sh --all
```

**开发环境特点：**
- ✅ **单体应用** - 所有微服务集成在一个进程中
- ✅ **快速启动** - 无需启动多个服务
- ✅ **简化调试** - 所有日志集中在一个文件
- ✅ **自动检测** - 自动跳过已构建的二进制文件
- ✅ **端口统一** - 所有 API 通过 8000 端口访问

## 生产环境模式

### 1. start-all.sh - 一键启动所有服务（微服务模式）

启动所有服务，包括基础设施、后端微服务和前端。

```bash
# 启动所有服务（包括构建）
./scripts/start-all.sh

# 跳过构建步骤，直接启动
./scripts/start-all.sh --skip-build
```

**启动的服务：**
- 基础设施：PostgreSQL、Redis、Etcd、Jaeger
- 后端服务：Gateway、Vehicle、Billing、Payment、Admin
- 前端：Next.js 开发服务器

### 2. stop-all.sh - 停止所有服务

停止所有运行中的服务。

```bash
# 停止后端服务（保留基础设施）
./scripts/stop-all.sh

# 停止所有服务（包括基础设施）
./scripts/stop-all.sh --all
```

### 3. status.sh - 查看服务状态

查看所有服务的运行状态。

```bash
# 查看所有服务状态
./scripts/status.sh

# 查看特定服务的日志
./scripts/status.sh --logs gateway
./scripts/status.sh --logs vehicle
./scripts/status.sh --logs frontend

# 查看所有服务的日志
./scripts/status.sh --all-logs
```

## 服务端口

| 服务 | 端口 | 说明 |
|------|------|------|
| Frontend | 3000 | Next.js 开发服务器 |
| Gateway | 8000 | API 网关 |
| Vehicle | 8001 | 车辆服务 |
| Billing | 8002 | 计费服务 |
| Payment | 8003 | 支付服务 |
| Admin | 8004 | 管理服务 |
| PostgreSQL | 5432 | 数据库 |
| Redis | 6379 | 缓存/消息队列 |
| Etcd | 2379 | 服务发现 |
| Jaeger | 16686 | 链路追踪 UI |

## 日志文件

所有服务的日志都保存在 `logs/` 目录下：

```
logs/
├── gateway.log
├── vehicle.log
├── billing.log
├── payment.log
├── admin.log
└── frontend.log
```

## 使用示例

### 完整的开发流程

```bash
# 1. 启动所有服务
./scripts/start-all.sh

# 2. 查看服务状态
./scripts/status.sh

# 3. 开发过程中查看日志
./scripts/status.sh --logs gateway

# 4. 停止所有服务
./scripts/stop-all.sh
```

### 仅重启后端服务

```bash
# 1. 停止后端服务
./scripts/stop-all.sh

# 2. 重新启动（跳过构建）
./scripts/start-all.sh --skip-build
```

## 注意事项

1. **依赖检查**：启动脚本会自动检查必要的依赖（go、docker、pnpm）
2. **端口冲突**：如果端口已被占用，脚本会跳过该服务的启动
3. **基础设施**：PostgreSQL 和 Redis 需要手动启动或使用已有的容器
4. **进程管理**：服务的 PID 保存在 `/tmp/smart-park-*.pid` 文件中

## 故障排查

### 服务无法启动

1. 检查端口是否被占用：
   ```bash
   lsof -i :8000
   ```

2. 查看服务日志：
   ```bash
   ./scripts/status.sh --logs gateway
   ```

3. 检查基础设施是否运行：
   ```bash
   docker ps | grep -E "postgres|redis|etcd|jaeger"
   ```

### 清理僵尸进程

```bash
# 停止所有服务并清理
./scripts/stop-all.sh --all

# 手动清理特定端口
lsof -ti:8000 | xargs kill -9
```

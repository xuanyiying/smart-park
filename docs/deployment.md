# Smart Park 部署文档

## 概述

本文档提供 Smart Park 智慧停车管理系统的完整部署指南，包括开发环境搭建、测试环境部署和生产环境部署方案。

## 部署架构

### 小型部署（1-20 个停车场）

适用于单个或少量停车场的场景，采用单机 Docker Compose 部署。

```
┌─────────────────────────────────────────────────────────┐
│                    云服务器（4核8G）                      │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Gateway   │  │   Vehicle   │  │   Billing   │     │
│  │   :8000     │  │   :8001     │  │   :8002     │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Payment   │  │    Admin    │  │   Web UI    │     │
│  │   :8003     │  │   :8004     │  │   :3000     │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  PostgreSQL │  │    Redis    │  │    Etcd     │     │
│  │   :5432     │  │   :6379     │  │   :2379     │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
```

**硬件要求**：
- CPU：4 核
- 内存：8 GB
- 磁盘：100 GB SSD
- 带宽：10 Mbps

### 中型部署（20-100 个停车场）

适用于区域级运营，采用主备双活架构。

```
┌─────────────────────────────────────────────────────────────┐
│                      负载均衡（SLB）                          │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┴─────────────┐
        │                           │
┌───────▼────────┐        ┌────────▼────────┐
│   主可用区      │        │   备可用区       │
│  ┌──────────┐  │        │  ┌──────────┐   │
│  │ Gateway  │  │        │  │ Gateway  │   │
│  │ Vehicle  │  │◀──────▶│  │ Vehicle  │   │
│  │ Billing  │  │  同步   │  │ Billing  │   │
│  │ Payment  │  │        │  │ Payment  │   │
│  │ Admin    │  │        │  │ Admin    │   │
│  └────┬─────┘  │        │  └────┬─────┘   │
│       │        │        │       │         │
│  ┌────▼─────┐  │        │  ┌────▼─────┐   │
│  │PostgreSQL│  │        │  │PostgreSQL│   │
│  │  (主)    │  │        │  │  (备)    │   │
│  └──────────┘  │        │  └──────────┘   │
└────────────────┘        └─────────────────┘
```

**硬件要求**：
- 主可用区：8 核 16 GB × 2 台
- 备可用区：8 核 16 GB × 2 台
- 云数据库：PostgreSQL 高可用版
- 云缓存：Redis 集群版

### 大型部署（100+ 个停车场）

适用于城市级或全国性运营，采用 Kubernetes 多活集群。

```
┌─────────────────────────────────────────────────────────────┐
│                     全局负载均衡（GSLB）                       │
└─────────────────────┬───────────────────────────────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
┌───────▼─────┐ ┌────▼────┐ ┌──────▼──────┐
│  华北集群    │ │ 华东集群 │ │   华南集群   │
│  (K8s)      │ │  (K8s)  │ │   (K8s)     │
└───────┬─────┘ └────┬────┘ └──────┬──────┘
        │            │             │
        └────────────┼─────────────┘
                     │
        ┌────────────┼────────────┐
        │            │            │
┌───────▼────┐ ┌────▼────┐ ┌─────▼─────┐
│  TiDB集群  │ │Kafka集群│ │ Redis集群 │
│ (分布式DB) │ │(消息队列)│ │  (缓存)   │
└────────────┘ └─────────┘ └───────────┘
```

## 环境准备

### 1. 安装 Docker 和 Docker Compose

```bash
# Ubuntu/Debian
curl -fsSL https://get.docker.com | sh
sudo usermod -aG docker $USER

# 安装 Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/download/v2.24.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

### 2. 安装 Go（开发环境）

```bash
# 下载并安装 Go 1.26
wget https://go.dev/dl/go1.26.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.0.linux-amd64.tar.gz

# 配置环境变量
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
source ~/.bashrc
```

### 3. 克隆代码仓库

```bash
git clone https://github.com/xuanyiying/smart-park.git
cd smart-park
```

## 开发环境部署

### 1. 启动基础设施

```bash
# 启动 PostgreSQL、Redis、Etcd、Jaeger
docker-compose -f deploy/docker-compose.yml up -d postgres redis etcd jaeger

# 查看服务状态
docker-compose -f deploy/docker-compose.yml ps
```

### 2. 初始化数据库

```bash
# 进入 PostgreSQL 容器
docker exec -it smart-park-postgres psql -U postgres -d parking

# 执行数据库迁移（通过服务自动执行）
go run ./cmd/vehicle -conf ./configs
go run ./cmd/billing -conf ./configs
go run ./cmd/payment -conf ./configs
go run ./cmd/admin -conf ./configs
```

### 3. 本地运行服务

```bash
# 使用 tmux 或同时打开多个终端窗口

# 终端 1：运行网关
go run ./cmd/gateway -conf ./configs

# 终端 2：运行车辆服务
go run ./cmd/vehicle -conf ./configs

# 终端 3：运行计费服务
go run ./cmd/billing -conf ./configs

# 终端 4：运行支付服务
go run ./cmd/payment -conf ./configs

# 终端 5：运行管理服务
go run ./cmd/admin -conf ./configs
```

### 4. 验证服务状态

```bash
# 检查网关健康状态
curl http://localhost:8000/health

# 测试入场接口
curl -X POST http://localhost:8000/api/v1/device/entry \
  -H "Content-Type: application/json" \
  -H "X-Device-Id: test_lane_001" \
  -d '{
    "deviceId": "test_lane_001",
    "plateNumber": "京A12345",
    "confidence": 0.95
  }'
```

## 测试环境部署

### 1. 构建 Docker 镜像

```bash
# 构建所有服务镜像
docker-compose -f deploy/docker-compose.yml build

# 查看构建的镜像
docker images | grep smart-park
```

### 2. 配置环境变量

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件，以下变量为强校验必填（缺失时容器启动失败）：
# - GRAFANA_PASSWORD     Grafana 管理员密码
# - SP_MQTT_BROKER       MQTT broker 地址（车辆服务道闸指令依赖）
# - SP_WECHAT_API_KEY    微信支付 API 密钥
# - SP_JWT_SECRET        JWT 签名密钥
#
# 注意：根目录 .env.example 中的变量名与 compose 实际消费的变量名不完全一致，
# 以 deploy/docker-compose.yml 中的 ${VAR:?} 强校验项为准。
```

### 3. 部署完整服务栈

```bash
# 启动所有服务
docker-compose -f deploy/docker-compose.yml up -d

# 查看日志
docker-compose -f deploy/docker-compose.yml logs -f

# 查看特定服务日志
docker-compose -f deploy/docker-compose.yml logs -f vehicle
```

### 4. 运行集成测试

```bash
# 执行集成测试
go test ./tests/e2e -tags=integration -v

# 压力测试
wrk -t12 -c400 -d60s http://localhost:8000/api/v1/device/entry
```

## 生产环境部署

### 1. 服务器准备

#### 安全加固

```bash
# 更新系统
sudo apt update && sudo apt upgrade -y

# 配置防火墙
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 8000:8004/tcp
sudo ufw enable

# 禁用 root 登录
sudo sed -i 's/PermitRootLogin yes/PermitRootLogin no/' /etc/ssh/sshd_config
sudo systemctl restart sshd
```

### 2. 生产配置

仓库中不存在 `configs/production/` 目录。生产配置直接修改各服务 `configs/*.yaml`，敏感字段（数据库连接串、密码、JWT 密钥等）通过 `SP_*` 环境变量覆盖（如 `SP_DATABASE_SOURCE`、`SP_JWT_SECRET`），详见各配置文件顶部的环境变量说明。

### 3. Docker Compose 生产配置

生产部署统一使用 `deploy/docker-compose.yml`（仓库中无 prod 变体文件），环境差异通过 `SP_*` 环境变量与 `.env` 覆盖：

```bash
# 生产环境部署
./deploy/scripts/deploy.sh deploy production v1.0.0
```

### 4. 部署脚本

`deploy/scripts/deploy.sh` 固定使用 `deploy/docker-compose.yml`，提供以下子命令：

```bash
./deploy.sh deploy [env] [version]   # 部署服务（默认 development latest）
./deploy.sh build                    # 构建 Docker 镜像
./deploy.sh push                     # 推送镜像到仓库
./deploy.sh update                   # 滚动更新服务
./deploy.sh rollback                 # 回滚到上一个版本
./deploy.sh status                   # 查看服务状态
./deploy.sh logs [service]           # 查看日志
./deploy.sh stop                     # 停止所有服务
```

### 5. 监控配置

#### Prometheus 配置

```yaml
# deploy/prometheus/prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'smart-park'
    static_configs:
      - targets: 
        - 'gateway:8000'
        - 'vehicle:8001'
        - 'billing:8002'
        - 'payment:8003'
        - 'admin:8004'
    metrics_path: /metrics
```

#### Grafana 仪表板

```json
{
  "dashboard": {
    "title": "Smart Park Monitoring",
    "panels": [
      {
        "title": "API Request Rate",
        "targets": [
          {
            "expr": "rate(http_requests_total[5m])"
          }
        ]
      },
      {
        "title": "Service Latency",
        "targets": [
          {
            "expr": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))"
          }
        ]
      }
    ]
  }
}
```

## Kubernetes 部署

### 1. 命名空间配置

```yaml
# deploy/k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: smart-park
  labels:
    name: smart-park
```

### 2. ConfigMap 配置

```yaml
# deploy/k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: smart-park-config
  namespace: smart-park
data:
  # 扁平结构，与 configs/*.yaml 及 pkg/config.Load 保持一致；
  # 敏感字段通过 SP_* 环境变量覆盖（完整内容见 deploy/k8s/configmap.yaml）
  gateway.yaml: |
    server:
      port: 8000
      timeout: 60
    routes:
      - path: /api/v1/device
        target: vehicle:8001
      - path: /api/v1/billing
        target: billing:8002
      - path: /api/v1/pay
        target: payment:8003
      - path: /api/v1/admin
        target: admin:8004
    jwt:
      public_key_path: "/keys/public.pem"
    etcd:
      endpoints:
        - "etcd:2379"
```

### 3. Secret 配置

```yaml
# deploy/k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: smart-park-secrets
  namespace: smart-park
type: Opaque
stringData:
  database-url: "postgres://user:pass@host:5432/parking"
  jwt-private-key: |
    -----BEGIN RSA PRIVATE KEY-----
    ...
    -----END RSA PRIVATE KEY-----
```

### 4. Deployment 配置

```yaml
# deploy/k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway
  namespace: smart-park
spec:
  replicas: 3
  selector:
    matchLabels:
      app: gateway
  template:
    metadata:
      labels:
        app: gateway
    spec:
      containers:
      - name: gateway
        image: smart-park/gateway:latest
        ports:
        - containerPort: 8000
        # 配置通过 -conf flag 加载（镜像内 /app/configs/ 提供），非环境变量注入
        command: ["./gateway-svc", "-conf", "/app/configs/gateway.yaml"]
        volumeMounts:
        - name: config
          mountPath: /app/configs
          readOnly: true
        - name: secrets
          mountPath: /secrets
          readOnly: true
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: smart-park-config
      - name: secrets
        secret:
          secretName: smart-park-secrets
```

### 5. Service 配置

```yaml
# deploy/k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: gateway
  namespace: smart-park
spec:
  selector:
    app: gateway
  ports:
  - port: 80
    targetPort: 8000
  type: ClusterIP

---
apiVersion: v1
kind: Service
metadata:
  name: gateway-lb
  namespace: smart-park
spec:
  selector:
    app: gateway
  ports:
  - port: 80
    targetPort: 8000
  type: LoadBalancer
```

### 6. HPA 自动扩缩容

```yaml
# deploy/k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: gateway-hpa
  namespace: smart-park
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: gateway
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### 7. 部署到 Kubernetes

```bash
# 应用配置
kubectl apply -f deploy/k8s/namespace.yaml
kubectl apply -f deploy/k8s/configmap.yaml
kubectl apply -f deploy/k8s/secrets.yaml
kubectl apply -f deploy/k8s/deployment.yaml
kubectl apply -f deploy/k8s/service.yaml
kubectl apply -f deploy/k8s/hpa.yaml

# 查看部署状态
kubectl get pods -n smart-park
kubectl get svc -n smart-park
kubectl get hpa -n smart-park
```

## 备份与恢复

### 数据库备份

> 说明：当前仓库未包含 backup.sh / restore.sh 脚本（deploy/scripts/ 下仅有 deploy.sh），以下为规划步骤，脚本需自行实现。

```bash
#!/bin/bash
# 规划：deploy/scripts/backup.sh

BACKUP_DIR="/backup/parking"
DATE=$(date +%Y%m%d_%H%M%S)

# PostgreSQL 备份
docker exec smart-park-postgres pg_dump -U postgres parking > ${BACKUP_DIR}/parking_${DATE}.sql

# 压缩备份
gzip ${BACKUP_DIR}/parking_${DATE}.sql

# 保留最近 7 天备份
find ${BACKUP_DIR} -name "parking_*.sql.gz" -mtime +7 -delete
```

### 数据恢复

```bash
#!/bin/bash
# 规划：deploy/scripts/restore.sh

BACKUP_FILE=$1

# 解压备份
gunzip ${BACKUP_FILE}

# 恢复数据库
docker exec -i smart-park-postgres psql -U postgres parking < ${BACKUP_FILE%.gz}
```

## 故障排查

### 常见问题

#### 1. 服务无法启动

```bash
# 检查日志
docker-compose logs <service-name>

# 检查端口占用
netstat -tlnp | grep 8000

# 检查配置文件
yamlint configs/*.yaml
```

#### 2. 数据库连接失败

```bash
# 检查数据库服务
docker-compose ps postgres

# 测试连接
docker exec -it smart-park-postgres psql -U postgres -d parking -c "SELECT 1"

# 检查网络
docker network inspect smart-park_default
```

#### 3. 性能问题

```bash
# 查看资源使用
docker stats

# 数据库慢查询
docker exec smart-park-postgres psql -U postgres -c "SELECT * FROM pg_stat_activity WHERE state = 'active';"

# Redis 监控
docker exec smart-park-redis redis-cli info stats
```

## 升级维护

### 滚动升级

```bash
# 1. 更新镜像版本
export VERSION=v1.2.0

# 2. 滚动更新服务
docker-compose -f deploy/docker-compose.yml up -d --no-deps --scale gateway=2 gateway
docker-compose -f deploy/docker-compose.yml up -d --no-deps vehicle billing payment admin

# 3. 验证服务健康
curl http://localhost:8000/health
```

### 数据库迁移

```bash
# 无独立迁移命令：表结构由各服务启动时通过 Ent Schema.Create 自动执行

# 一次性数据迁移脚本（金额元转分，已执行）
# scripts/migrate_amount_to_cents.sql
```

## 安全建议

1. **使用 HTTPS**：生产环境必须使用 HTTPS
2. **密钥管理**：使用 Docker Secrets 或 K8s Secrets 管理敏感信息
3. **网络隔离**：使用私有网络，限制服务间访问
4. **定期更新**：及时更新基础镜像和依赖库
5. **日志审计**：启用审计日志，定期审查
6. **访问控制**：配置防火墙规则，限制访问来源

## 联系支持

如有部署问题，请联系：
- 技术支持：support@smart-park.example.com
- 文档反馈：https://github.com/xuanyiying/smart-park/issues

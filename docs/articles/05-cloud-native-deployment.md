# 云原生部署：从 Docker Compose 到 Kubernetes

## 引言

在智慧停车系统的实际落地过程中，不同规模的停车场对部署方案有着截然不同的需求。小型停车场可能只有几十个车位，预算有限，需要简单易维护的单机部署方案；中型连锁停车场可能拥有数十个分店，需要主备双活架构保证业务连续性；而大型停车集团可能管理着数百个停车场，需要 Kubernetes 多集群部署实现弹性伸缩和跨地域容灾。

本文以 Smart Park 智慧停车管理系统为例，系统性地介绍从 Docker Compose 单机部署到 Kubernetes 多集群部署的完整演进路径。文章面向运维工程师和 DevOps 从业者，将深入探讨不同规模场景下的架构设计、配置管理、自动化部署和运维最佳实践。

文章首先介绍基于 Docker Compose 的小型部署方案，适合快速验证和小规模应用；然后探讨主备双活的中型部署方案，满足业务连续性要求；最后详细讲解 Kubernetes 多集群部署方案，支持大规模生产环境。同时，我们还将介绍完整的 CI/CD 流程设计，实现从代码提交到生产部署的全自动化。

## 一、小型部署方案：Docker Compose 单机部署

### 1.1 单机部署架构

对于小型停车场或开发测试环境，单机部署是最简单高效的选择。Smart Park 采用 Docker Compose 进行服务编排，将所有微服务、数据库、缓存和监控组件部署在单台服务器上。

```mermaid
graph TB
    Client[客户端] --> Gateway[API Gateway:8000]
    
    subgraph Docker Compose
        Gateway --> Vehicle[Vehicle Service:8001]
        Gateway --> Billing[Billing Service:8002]
        Gateway --> Payment[Payment Service:8003]
        Gateway --> Admin[Admin Service:8004]
        
        Vehicle --> PostgreSQL[(PostgreSQL)]
        Billing --> PostgreSQL
        Payment --> PostgreSQL
        Admin --> PostgreSQL
        
        Vehicle --> Redis[(Redis)]
        Billing --> Redis
        Payment --> Redis
        
        Gateway --> Etcd[Etcd]
        Vehicle --> Etcd
        
        Gateway --> Jaeger[Jaeger]
        Vehicle --> Jaeger
        Billing --> Jaeger
        Payment --> Jaeger
        Admin --> Jaeger
    end
```

这种架构的优势在于：
- **部署简单**：一条命令即可启动所有服务
- **资源占用低**：适合 2核4G 的入门级服务器
- **易于调试**：所有服务在同一主机，日志集中查看
- **快速验证**：适合开发测试和概念验证

### 1.2 Docker Compose 配置详解

Smart Park 的 Docker Compose 配置文件位于 `deploy/docker-compose.yml`，采用分层设计，将基础设施和业务服务分离。

**基础设施服务配置**：

```yaml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: smart-park-postgres
    environment:
      POSTGRES_DB: ${POSTGRES_DB:-parking}
      POSTGRES_USER: ${POSTGRES_USER:-postgres}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-postgres}
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-postgres}"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: smart-park-redis
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  etcd:
    image: quay.io/coreos/etcd:v3.5.9
    container_name: smart-park-etcd
    environment:
      ETCD_NAME: etcd
      ETCD_ADVERTISE_CLIENT_URLS: http://etcd:2379
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
    ports:
      - "2379:2379"

  jaeger:
    image: jaegertracing/all-in-one:1.52
    container_name: smart-park-jaeger
    environment:
      COLLECTOR_OTTL_ENABLED: true
    ports:
      - "16686:16686"
      - "4317:4317"
      - "4318:4318"

volumes:
  postgres_data:
  redis_data:
```

**业务服务配置**：

```yaml
services:
  gateway:
    build:
      context: ../../
      dockerfile: deploy/docker/Dockerfile.gateway
    container_name: smart-park-gateway
    ports:
      - "8000:8000"
    depends_on:
      - postgres
      - redis
      - etcd
    environment:
      - KRATOS_CONF=../../configs/gateway.yaml

  vehicle:
    build:
      context: ../../
      dockerfile: deploy/docker/Dockerfile.vehicle
    container_name: smart-park-vehicle
    ports:
      - "8001:8001"
    depends_on:
      - postgres
      - redis
    environment:
      - KRATOS_CONF=../../configs/vehicle.yaml

  billing:
    build:
      context: ../../
      dockerfile: deploy/docker/Dockerfile.billing
    container_name: smart-park-billing
    ports:
      - "8002:8002"
    depends_on:
      - postgres
    environment:
      - KRATOS_CONF=../../configs/billing.yaml

  payment:
    build:
      context: ../../
      dockerfile: deploy/docker/Dockerfile.payment
    container_name: smart-park-payment
    ports:
      - "8003:8003"
    depends_on:
      - postgres
    environment:
      - KRATOS_CONF=../../configs/payment.yaml

  admin:
    build:
      context: ../../
      dockerfile: deploy/docker/Dockerfile.admin
    container_name: smart-park-admin
    ports:
      - "8004:8004"
    depends_on:
      - postgres
    environment:
      - KRATOS_CONF=../../configs/admin.yaml
```

### 1.3 服务编排和依赖管理

Docker Compose 通过 `depends_on` 字段管理服务启动顺序，但这只能保证容器启动，不能保证服务就绪。Smart Park 使用健康检查机制确保服务依赖正确处理。

**健康检查配置要点**：

1. **数据库健康检查**：PostgreSQL 使用 `pg_isready` 命令检查数据库是否可连接
2. **Redis 健康检查**：使用 `redis-cli ping` 检查 Redis 是否响应
3. **应用服务健康检查**：每个微服务实现 `/health` 和 `/ready` 端点

**启动顺序控制**：

```bash
# 启动基础设施
docker-compose up -d postgres redis etcd jaeger

# 等待数据库就绪
sleep 10

# 启动业务服务
docker-compose up -d gateway vehicle billing payment admin
```

### 1.4 资源限制和健康检查

在生产环境中，必须为容器设置资源限制，防止某个服务占用过多资源影响其他服务。

```yaml
services:
  gateway:
    deploy:
      resources:
        limits:
          cpus: '1.0'
          memory: 512M
        reservations:
          cpus: '0.5'
          memory: 256M
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

**资源限制建议**：

| 服务 | CPU 限制 | 内存限制 | 说明 |
|------|---------|---------|------|
| Gateway | 1.0 核 | 512MB | API 网关，处理所有外部请求 |
| Vehicle | 1.0 核 | 512MB | 车辆服务，处理进出场业务 |
| Billing | 0.5 核 | 256MB | 计费服务，计算逻辑为主 |
| Payment | 0.5 核 | 256MB | 支付服务，调用第三方接口 |
| Admin | 0.5 核 | 256MB | 管理服务，后台管理功能 |

### 1.5 Dockerfile 最佳实践

Smart Park 采用多阶段构建优化镜像大小，所有服务使用统一的构建模式。

```dockerfile
# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make protobuf

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/gateway ./cmd/gateway

# Final stage
FROM alpine:latest

WORKDIR /app

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/gateway .
COPY configs/gateway.yaml /app/configs/gateway.yaml

EXPOSE 8000

CMD ["./gateway"]
```

**镜像优化要点**：
- 使用 Alpine 基础镜像，最终镜像大小控制在 50MB 以内
- 多阶段构建，编译环境和运行环境分离
- 静态编译，使用 `CGO_ENABLED=0` 消除 C 依赖
- 只复制必要的二进制文件和配置文件

## 二、中型部署方案：主备双活架构

### 2.1 主备架构设计

对于中型停车场或连锁企业，单机部署无法满足业务连续性要求。Smart Park 采用主备双活架构，在保证成本可控的前提下实现高可用。

```mermaid
graph TB
    subgraph 负载均衡层
        LB[Nginx/HAProxy]
    end
    
    subgraph 主节点
        MasterGateway[Gateway Master]
        MasterVehicle[Vehicle Master]
        MasterBilling[Billing Master]
        MasterPayment[Payment Master]
        MasterAdmin[Admin Master]
    end
    
    subgraph 备节点
        SlaveGateway[Gateway Slave]
        SlaveVehicle[Vehicle Slave]
        SlaveBilling[Billing Slave]
        SlavePayment[Payment Slave]
        SlaveAdmin[Admin Slave]
    end
    
    subgraph 数据层
        PGMaster[(PostgreSQL Master)]
        PGSlave[(PostgreSQL Slave)]
        RedisMaster[(Redis Master)]
        RedisSlave[(Redis Slave)]
    end
    
    Client[客户端] --> LB
    LB --> MasterGateway
    LB --> SlaveGateway
    
    MasterGateway --> MasterVehicle
    MasterGateway --> MasterBilling
    MasterVehicle --> PGMaster
    MasterBilling --> PGMaster
    MasterVehicle --> RedisMaster
    
    SlaveGateway --> SlaveVehicle
    SlaveGateway --> SlaveBilling
    SlaveVehicle --> PGSlave
    SlaveBilling --> PGSlave
    SlaveVehicle --> RedisSlave
    
    PGMaster -.->|同步复制| PGSlave
    RedisMaster -.->|主从复制| RedisSlave
```

### 2.2 数据库主从复制

PostgreSQL 主从复制是中型部署的核心，Smart Park 采用流复制实现数据的实时同步。

**主库配置（postgresql.conf）**：

```conf
listen_addresses = '*'
wal_level = replica
max_wal_senders = 3
wal_keep_segments = 64
synchronous_commit = on
synchronous_standby_names = 'slave1'
```

**主库访问控制（pg_hba.conf）**：

```conf
host replication replicator <slave_ip>/32 md5
```

**从库配置（recovery.conf）**：

```conf
standby_mode = on
primary_conninfo = 'host=<master_ip> port=5432 user=replicator password=<password>'
trigger_file = '/tmp/postgresql.trigger'
```

**Docker Compose 主从配置**：

```yaml
version: '3.8'

services:
  postgres-master:
    image: postgres:15-alpine
    container_name: smart-park-postgres-master
    environment:
      POSTGRES_DB: parking
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_master_data:/var/lib/postgresql/data
      - ./postgres/master/postgresql.conf:/etc/postgresql/postgresql.conf
      - ./postgres/master/pg_hba.conf:/etc/postgresql/pg_hba.conf
    command: postgres -c config_file=/etc/postgresql/postgresql.conf
    ports:
      - "5432:5432"

  postgres-slave:
    image: postgres:15-alpine
    container_name: smart-park-postgres-slave
    environment:
      POSTGRES_DB: parking
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      PGUSER: replicator
      PGPASSWORD: ${REPLICATOR_PASSWORD}
    volumes:
      - postgres_slave_data:/var/lib/postgresql/data
      - ./postgres/slave/postgresql.conf:/etc/postgresql/postgresql.conf
    command: postgres -c config_file=/etc/postgresql/postgresql.conf
    depends_on:
      - postgres-master
    ports:
      - "5433:5432"

volumes:
  postgres_master_data:
  postgres_slave_data:
```

### 2.3 Redis 集群配置

Smart Park 使用 Redis Sentinel 实现主从自动切换，保证缓存服务的高可用。

**Redis 主从配置**：

```yaml
services:
  redis-master:
    image: redis:7-alpine
    container_name: smart-park-redis-master
    command: redis-server --appendonly yes
    ports:
      - "6379:6379"
    volumes:
      - redis_master_data:/data

  redis-slave:
    image: redis:7-alpine
    container_name: smart-park-redis-slave
    command: redis-server --slaveof redis-master 6379 --appendonly yes
    depends_on:
      - redis-master
    ports:
      - "6380:6379"
    volumes:
      - redis_slave_data:/data

  redis-sentinel:
    image: redis:7-alpine
    container_name: smart-park-redis-sentinel
    command: redis-sentinel /etc/redis/sentinel.conf
    depends_on:
      - redis-master
      - redis-slave
    volumes:
      - ./redis/sentinel.conf:/etc/redis/sentinel.conf
    ports:
      - "26379:26379"

volumes:
  redis_master_data:
  redis_slave_data:
```

**Sentinel 配置（sentinel.conf）**：

```conf
sentinel monitor smart-park redis-master 6379 2
sentinel down-after-milliseconds smart-park 5000
sentinel failover-timeout smart-park 60000
sentinel parallel-syncs smart-park 1
```

### 2.4 负载均衡策略

Smart Park 使用 Nginx 作为负载均衡器，实现请求的智能分发。

**Nginx 配置**：

```nginx
upstream gateway_cluster {
    least_conn;
    server 192.168.1.10:8000 weight=3;
    server 192.168.1.11:8000 weight=2 backup;
    keepalive 32;
}

upstream vehicle_cluster {
    least_conn;
    server 192.168.1.10:8001;
    server 192.168.1.11:8001;
    keepalive 16;
}

server {
    listen 80;
    server_name api.smartpark.com;

    location /api/v1/device/ {
        proxy_pass http://vehicle_cluster;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_connect_timeout 3s;
        proxy_read_timeout 30s;
    }

    location /api/v1/billing/ {
        proxy_pass http://billing_cluster;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/v1/pay/ {
        proxy_pass http://payment_cluster;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api/v1/admin/ {
        proxy_pass http://admin_cluster;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location / {
        proxy_pass http://gateway_cluster;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

**负载均衡策略说明**：
- **least_conn**：最少连接数算法，适合长连接场景
- **weight**：权重配置，主节点权重更高
- **backup**：备用节点，仅在主节点故障时启用
- **keepalive**：保持连接池，减少连接建立开销

## 三、大型部署方案：Kubernetes 多集群

### 3.1 Kubernetes 架构设计

对于大型停车集团，Smart Park 采用 Kubernetes 进行容器编排，实现自动化部署、弹性伸缩和跨集群容灾。

```mermaid
graph TB
    subgraph 用户访问层
        User[用户]
        DNS[DNS 解析]
        CDN[CDN 加速]
    end
    
    subgraph 接入层
        Ingress[Ingress Controller]
    end
    
    subgraph Kubernetes Cluster
        subgraph Gateway Namespace
            GW1[Gateway Pod 1]
            GW2[Gateway Pod 2]
            GW3[Gateway Pod N]
        end
        
        subgraph Vehicle Namespace
            V1[Vehicle Pod 1]
            V2[Vehicle Pod 2]
        end
        
        subgraph Billing Namespace
            B1[Billing Pod 1]
            B2[Billing Pod 2]
        end
        
        subgraph Payment Namespace
            P1[Payment Pod 1]
            P2[Payment Pod 2]
        end
        
        subgraph Admin Namespace
            A1[Admin Pod 1]
            A2[Admin Pod 2]
        end
    end
    
    subgraph 数据层
        PGCluster[(PostgreSQL Cluster)]
        RedisCluster[(Redis Cluster)]
        EtcdCluster[(Etcd Cluster)]
    end
    
    subgraph 监控层
        Prometheus[Prometheus]
        Grafana[Grafana]
        Jaeger[Jaeger]
    end
    
    User --> DNS
    DNS --> CDN
    CDN --> Ingress
    Ingress --> GW1
    Ingress --> GW2
    Ingress --> GW3
    
    GW1 --> V1
    GW1 --> B1
    GW1 --> P1
    GW1 --> A1
    
    V1 --> PGCluster
    V1 --> RedisCluster
    B1 --> PGCluster
    P1 --> PGCluster
    A1 --> PGCluster
    
    Prometheus --> GW1
    Prometheus --> V1
    Prometheus --> B1
    Jaeger --> GW1
```

### 3.2 Namespace 和资源隔离

Smart Park 使用命名空间实现不同环境和服务的隔离。

**命名空间定义（namespace.yaml）**：

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: smart-park
  labels:
    name: smart-park
    app.kubernetes.io/name: smart-park
    app.kubernetes.io/version: "1.0.0"
```

**多环境命名空间策略**：

```yaml
# 开发环境
apiVersion: v1
kind: Namespace
metadata:
  name: smart-park-dev
  labels:
    env: development

---
# 测试环境
apiVersion: v1
kind: Namespace
metadata:
  name: smart-park-staging
  labels:
    env: staging

---
# 生产环境
apiVersion: v1
kind: Namespace
metadata:
  name: smart-park-prod
  labels:
    env: production
```

### 3.3 Deployment 配置

Smart Park 的每个微服务都配置了 Deployment，实现声明式部署和滚动更新。

**Gateway Deployment 配置**：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway
  namespace: smart-park
  labels:
    app: gateway
spec:
  replicas: 2
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
          name: http
        - containerPort: 9000
          name: grpc
        env:
        - name: KRATOS_CONF
          value: "/configs/gateway.yaml"
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: smart-park-secrets
              key: db-password
        volumeMounts:
        - name: config
          mountPath: /configs
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
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /ready
            port: 8000
          initialDelaySeconds: 5
          periodSeconds: 5
          timeoutSeconds: 3
          failureThreshold: 3
      volumes:
      - name: config
        configMap:
          name: smart-park-config
          items:
          - key: gateway.yaml
            path: gateway.yaml
```

**Deployment 关键配置说明**：

1. **副本数管理**：通过 `replicas` 控制实例数量
2. **资源限制**：设置 requests 和 limits 防止资源争抢
3. **健康检查**：
   - livenessProbe：检测容器是否存活，失败则重启
   - readinessProbe：检测容器是否就绪，失败则从 Service 移除
4. **配置管理**：通过 ConfigMap 挂载配置文件
5. **密钥管理**：通过 Secret 注入敏感信息

### 3.4 Service 配置

Kubernetes Service 为 Pod 提供稳定的访问入口和负载均衡。

**Service 配置示例**：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: gateway
  namespace: smart-park
  labels:
    app: gateway
spec:
  selector:
    app: gateway
  ports:
  - name: http
    port: 80
    targetPort: 8000
  - name: grpc
    port: 90
    targetPort: 9000
  type: ClusterIP

---
apiVersion: v1
kind: Service
metadata:
  name: gateway-lb
  namespace: smart-park
  labels:
    app: gateway
spec:
  selector:
    app: gateway
  ports:
  - name: http
    port: 80
    targetPort: 8000
  type: LoadBalancer
```

**Service 类型选择**：
- **ClusterIP**：内部服务访问，适合微服务间通信
- **LoadBalancer**：外部访问，适合对外暴露的 API 网关
- **NodePort**：通过节点端口访问，适合测试环境

### 3.5 ConfigMap 配置管理

Smart Park 使用 ConfigMap 统一管理所有服务的配置文件。

**ConfigMap 配置**：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: smart-park-config
  namespace: smart-park
data:
  gateway.yaml: |
    server:
      http:
        addr: 0.0.0.0:8000
        timeout: 1s
      grpc:
        addr: 0.0.0.0:9000
        timeout: 1s
    data:
      database:
        driver: postgres
        source: postgres://postgres:${DB_PASSWORD}@postgres:5432/parking?sslmode=disable
      redis:
        addr: redis:6379
        password: ""
        db: 0
        read_timeout: 0.2s
        write_timeout: 0.2s
    trace:
      endpoint: http://jaeger:14268/api/traces

  vehicle.yaml: |
    server:
      http:
        addr: 0.0.0.0:8001
      grpc:
        addr: 0.0.0.0:9001
    data:
      database:
        driver: postgres
        source: postgres://postgres:${DB_PASSWORD}@postgres:5432/parking?sslmode=disable
      redis:
        addr: redis:6379
    lock:
      ttl: 30s

  billing.yaml: |
    server:
      http:
        addr: 0.0.0.0:8002
      grpc:
        addr: 0.0.0.0:9002
    data:
      database:
        driver: postgres
        source: postgres://postgres:${DB_PASSWORD}@postgres:5432/parking?sslmode=disable
      redis:
        addr: redis:6379

  payment.yaml: |
    server:
      http:
        addr: 0.0.0.0:8003
      grpc:
        addr: 0.0.0.0:9003
    data:
      database:
        driver: postgres
        source: postgres://postgres:${DB_PASSWORD}@postgres:5432/parking?sslmode=disable
      redis:
        addr: redis:6379
    payment:
      orderExpiration: 30m
      refundWindow: 30m

  admin.yaml: |
    server:
      http:
        addr: 0.0.0.0:8004
      grpc:
        addr: 0.0.0.0:9004
    data:
      database:
        driver: postgres
        source: postgres://postgres:${DB_PASSWORD}@postgres:5432/parking?sslmode=disable
      redis:
        addr: redis:6379
    admin:
      defaultPageSize: 20
      maxPageSize: 100
      reportCacheTTL: 5m
```

**ConfigMap 更新策略**：

```bash
# 更新 ConfigMap
kubectl create configmap smart-park-config --from-file=configs/ -n smart-park --dry-run=client -o yaml | kubectl apply -f -

# 滚动重启 Pod 使配置生效
kubectl rollout restart deployment/gateway -n smart-park
kubectl rollout restart deployment/vehicle -n smart-park
```

### 3.6 Secret 密钥管理

敏感信息通过 Kubernetes Secret 管理，支持多种加密方式。

**Secret 配置模板**：

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: smart-park-secrets
  namespace: smart-park
type: Opaque
stringData:
  db-password: "${DB_PASSWORD}"
  jwt-secret: "${JWT_SECRET}"
  wechat-private-key: |
    -----BEGIN PRIVATE KEY-----
    # 替换为实际的微信支付私钥
    -----END PRIVATE KEY-----
  alipay-private-key: |
    -----BEGIN RSA PRIVATE KEY-----
    # 替换为实际的支付宝私钥
    -----END RSA PRIVATE KEY-----
  alipay-public-key: |
    -----BEGIN PUBLIC KEY-----
    # 替换为实际的支付宝公钥
    -----END PUBLIC KEY-----
```

**生产环境 Secret 创建方式**：

```bash
# 使用 kubectl 创建 Secret（推荐）
kubectl create secret generic smart-park-secrets \
  --namespace=smart-park \
  --from-literal=db-password=<your-db-password> \
  --from-literal=jwt-secret=<your-jwt-secret> \
  --from-file=wechat-private-key=<path-to-wechat-key> \
  --from-file=alipay-private-key=<path-to-alipay-key> \
  --from-file=alipay-public-key=<path-to-alipay-public-key>

# 使用 Sealed Secrets 加密
kubeseal --format=yaml < secret.yaml > sealed-secret.yaml
```

### 3.7 HPA 自动扩缩容

Smart Park 配置了 Horizontal Pod Autoscaler（HPA），根据负载自动调整 Pod 数量。

**HPA 配置**：

```yaml
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
  behavior:
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 100
        periodSeconds: 60
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
```

**HPA 策略说明**：

| 服务 | 最小副本 | 最大副本 | CPU 阈值 | 内存阈值 | 说明 |
|------|---------|---------|---------|---------|------|
| Gateway | 2 | 10 | 70% | 80% | API 网关，流量入口 |
| Vehicle | 2 | 10 | 70% | 80% | 车辆服务，高峰期扩容 |
| Billing | 2 | 5 | 70% | - | 计费服务，计算密集 |
| Payment | 2 | 5 | 70% | - | 支付服务，IO 密集 |
| Admin | 2 | 5 | 70% | - | 管理服务，后台访问 |

**扩缩容行为配置**：
- **scaleUp**：扩容策略，稳定窗口 60 秒，每次最多扩容 100%
- **scaleDown**：缩容策略，稳定窗口 300 秒，每次最多缩容 10%

### 3.8 多集群部署策略

对于跨地域的大型停车集团，Smart Park 采用多集群部署实现异地容灾。

**多集群架构**：

```mermaid
graph LR
    subgraph 用户
        User[用户请求]
    end
    
    subgraph 全局负载均衡
        GSLB[全局 DNS 负载均衡]
    end
    
    subgraph 北京集群
        BJIngress[Ingress]
        BJGateway[Gateway]
        BJSvc[Services]
        BJDB[(Database)]
    end
    
    subgraph 上海集群
        SHIngress[Ingress]
        SHGateway[Gateway]
        SHSvc[Services]
        SHDB[(Database)]
    end
    
    subgraph 广州集群
        GZIngress[Ingress]
        GZGateway[Gateway]
        GZSvc[Services]
        GZDB[(Database)]
    end
    
    User --> GSLB
    GSLB --> BJIngress
    GSLB --> SHIngress
    GSLB --> GZIngress
    
    BJIngress --> BJGateway
    BJGateway --> BJSvc
    BJSvc --> BJDB
    
    SHIngress --> SHGateway
    SHGateway --> SHSvc
    SHSvc --> SHDB
    
    GZIngress --> GZGateway
    GZGateway --> GZSvc
    GZSvc --> GZDB
    
    BJDB -.->|数据同步| SHDB
    SHDB -.->|数据同步| GZDB
```

**多集群管理工具**：

```bash
# 使用 kubefed 管理多集群
kubefedctl join beijing-cluster --cluster-context beijing --host-cluster-context beijing
kubefedctl join shanghai-cluster --cluster-context shanghai --host-cluster-context beijing
kubefedctl join guangzhou-cluster --cluster-context guangzhou --host-cluster-context beijing

# 创建联邦 Deployment
kubefedctl deploy gateway --namespace smart-park --replicas 6
```

## 四、CI/CD 流程设计

### 4.1 CI/CD 架构

Smart Park 采用 GitHub Actions 构建完整的 CI/CD 流水线，实现从代码提交到生产部署的全自动化。

```mermaid
graph LR
    A[代码提交] --> B[代码检查]
    B --> C[单元测试]
    C --> D[构建镜像]
    D --> E[推送镜像]
    E --> F{环境选择}
    F -->|开发环境| G[自动部署 Dev]
    F -->|测试环境| H[自动部署 Staging]
    F -->|生产环境| I[人工审批]
    I --> J[灰度发布]
    J --> K[全量发布]
    J --> L[自动回滚]
```

### 4.2 GitHub Actions 配置

**CI 流水线配置**：

```yaml
name: CI

on:
  push:
    branches: [ main, develop ]
  pull_request:
    branches: [ main ]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.22'
    
    - name: Run golangci-lint
      uses: golangci/golangci-lint-action@v3
      with:
        version: latest

  test:
    runs-on: ubuntu-latest
    needs: lint
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Go
      uses: actions/setup-go@v4
      with:
        go-version: '1.22'
    
    - name: Run tests
      run: go test -v -race -coverprofile=coverage.out ./...
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      with:
        file: ./coverage.out

  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v2
    
    - name: Login to Docker Hub
      uses: docker/login-action@v2
      with:
        username: ${{ secrets.DOCKER_USERNAME }}
        password: ${{ secrets.DOCKER_PASSWORD }}
    
    - name: Build and push Gateway
      uses: docker/build-push-action@v4
      with:
        context: .
        file: deploy/docker/Dockerfile.gateway
        push: true
        tags: |
          smartpark/gateway:${{ github.sha }}
          smartpark/gateway:latest
        cache-from: type=gha
        cache-to: type=gha,mode=max
    
    - name: Build and push Vehicle
      uses: docker/build-push-action@v4
      with:
        context: .
        file: deploy/docker/Dockerfile.vehicle
        push: true
        tags: smartpark/vehicle:${{ github.sha }}
    
    - name: Build and push Billing
      uses: docker/build-push-action@v4
      with:
        context: .
        file: deploy/docker/Dockerfile.billing
        push: true
        tags: smartpark/billing:${{ github.sha }}
    
    - name: Build and push Payment
      uses: docker/build-push-action@v4
      with:
        context: .
        file: deploy/docker/Dockerfile.payment
        push: true
        tags: smartpark/payment:${{ github.sha }}
    
    - name: Build and push Admin
      uses: docker/build-push-action@v4
      with:
        context: .
        file: deploy/docker/Dockerfile.admin
        push: true
        tags: smartpark/admin:${{ github.sha }}
```

**CD 流水线配置**：

```yaml
name: CD

on:
  push:
    branches: [ main ]
    tags:
      - 'v*'

jobs:
  deploy-dev:
    runs-on: ubuntu-latest
    environment: development
    steps:
    - uses: actions/checkout@v3
    
    - name: Deploy to Development
      run: |
        echo "Deploying to development environment..."
        ./deploy/scripts/deploy.sh development ${{ github.sha }}
    
  deploy-staging:
    runs-on: ubuntu-latest
    environment: staging
    needs: deploy-dev
    steps:
    - uses: actions/checkout@v3
    
    - name: Deploy to Staging
      run: |
        echo "Deploying to staging environment..."
        ./deploy/scripts/deploy.sh staging ${{ github.sha }}
    
    - name: Run integration tests
      run: |
        echo "Running integration tests..."
        go test -v ./tests/integration/...

  deploy-production:
    runs-on: ubuntu-latest
    environment: production
    needs: deploy-staging
    if: startsWith(github.ref, 'refs/tags/')
    steps:
    - uses: actions/checkout@v3
    
    - name: Deploy to Production
      run: |
        echo "Deploying to production environment..."
        ./deploy/scripts/deploy.sh production ${{ github.ref_name }}
    
    - name: Health check
      run: |
        echo "Running health check..."
        ./deploy/scripts/deploy.sh health
```

### 4.3 镜像构建和推送

Smart Park 使用 Docker Buildx 构建多平台镜像，支持 AMD64 和 ARM64 架构。

**多平台构建配置**：

```yaml
- name: Set up QEMU
  uses: docker/setup-qemu-action@v2

- name: Set up Docker Buildx
  uses: docker/setup-buildx-action@v2

- name: Build and push multi-arch images
  uses: docker/build-push-action@v4
  with:
    context: .
    file: deploy/docker/Dockerfile.gateway
    platforms: linux/amd64,linux/arm64
    push: true
    tags: smartpark/gateway:${{ github.sha }}
    cache-from: type=gha
    cache-to: type=gha,mode=max
```

### 4.4 自动化部署脚本

Smart Park 提供了完整的部署脚本，支持多种部署操作。

**部署脚本核心功能**：

```bash
#!/bin/bash
# Smart Park 部署脚本
# 用法: ./deploy.sh [environment] [version]

set -e

ENV=${1:-development}
VERSION=${2:-latest}

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装"
        exit 1
    fi
    
    log_info "依赖检查通过"
}

# 构建镜像
build_images() {
    log_info "构建 Docker 镜像..."
    export VERSION=${VERSION}
    docker-compose -f deploy/docker-compose.yml build --parallel
    log_info "镜像构建完成"
}

# 部署服务
deploy_services() {
    log_info "部署 Smart Park ${VERSION} 到 ${ENV} 环境..."
    
    # 启动基础设施
    log_info "启动基础设施..."
    docker-compose -f deploy/docker-compose.yml up -d postgres redis etcd jaeger
    
    # 等待数据库就绪
    log_info "等待数据库就绪..."
    sleep 10
    
    # 启动业务服务
    log_info "启动业务服务..."
    docker-compose -f deploy/docker-compose.yml up -d gateway vehicle billing payment admin
    
    log_info "服务部署完成"
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    services=("gateway:8000" "vehicle:8001" "billing:8002" "payment:8003" "admin:8004")
    
    for service in "${services[@]}"; do
        IFS=':' read -r name port <<< "$service"
        
        max_attempts=30
        attempt=1
        
        while [ $attempt -le $max_attempts ]; do
            if curl -s "http://localhost:${port}/health" > /dev/null 2>&1; then
                log_info "${name} 服务健康"
                break
            fi
            
            if [ $attempt -eq $max_attempts ]; then
                log_error "${name} 服务健康检查失败"
                return 1
            fi
            
            log_info "${name} 服务未就绪，等待中... (${attempt}/${max_attempts})"
            sleep 2
            ((attempt++))
        done
    done
    
    log_info "所有服务健康检查通过"
}

# 滚动更新
rolling_update() {
    log_info "执行滚动更新..."
    
    services=("gateway" "vehicle" "billing" "payment" "admin")
    
    for service in "${services[@]}"; do
        log_info "更新 ${service} 服务..."
        
        # 拉取最新镜像
        docker-compose -f deploy/docker-compose.yml pull ${service}
        
        # 重启服务
        docker-compose -f deploy/docker-compose.yml up -d --no-deps ${service}
        
        # 等待服务就绪
        sleep 5
        
        log_info "${service} 服务更新完成"
    done
    
    log_info "滚动更新完成"
}

# 回滚部署
rollback() {
    log_info "执行回滚..."
    
    # 使用上一个版本
    PREV_VERSION=$(docker images --format "{{.Tag}}" smart-park/gateway | grep -v "latest" | head -1)
    
    if [ -z "$PREV_VERSION" ]; then
        log_error "未找到上一个版本"
        exit 1
    fi
    
    log_info "回滚到版本: ${PREV_VERSION}"
    
    export VERSION=${PREV_VERSION}
    docker-compose -f deploy/docker-compose.yml up -d
    
    log_info "回滚完成"
}

# 主函数
main() {
    command=$1
    shift
    
    case $command in
        deploy)
            check_dependencies
            build_images
            deploy_services
            health_check
            ;;
        update)
            check_dependencies
            rolling_update
            health_check
            ;;
        rollback)
            check_dependencies
            rollback
            ;;
        health)
            health_check
            ;;
        *)
            echo "用法: ./deploy.sh {deploy|update|rollback|health}"
            ;;
    esac
}

main "$@"
```

### 4.5 灰度发布和回滚

Smart Park 支持灰度发布（金丝雀发布），逐步将流量切换到新版本。

**Kubernetes 灰度发布配置**：

```yaml
# 稳定版本
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway-stable
  namespace: smart-park
spec:
  replicas: 9
  selector:
    matchLabels:
      app: gateway
      version: stable
  template:
    metadata:
      labels:
        app: gateway
        version: stable
    spec:
      containers:
      - name: gateway
        image: smartpark/gateway:v1.0.0

---
# 金丝雀版本
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway-canary
  namespace: smart-park
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gateway
      version: canary
  template:
    metadata:
      labels:
        app: gateway
        version: canary
    spec:
      containers:
      - name: gateway
        image: smartpark/gateway:v1.1.0

---
# Service 配置
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
```

**灰度发布流程**：

```bash
# 1. 部署金丝雀版本（10% 流量）
kubectl apply -f gateway-canary.yaml

# 2. 监控金丝雀版本指标
kubectl logs -f deployment/gateway-canary -n smart-park

# 3. 逐步增加金丝雀版本副本数
kubectl scale deployment gateway-canary --replicas=3 -n smart-park  # 25% 流量
kubectl scale deployment gateway-canary --replicas=5 -n smart-park  # 50% 流量

# 4. 全量切换
kubectl scale deployment gateway-stable --replicas=0 -n smart-park
kubectl scale deployment gateway-canary --replicas=10 -n smart-park

# 5. 清理旧版本
kubectl delete deployment gateway-stable -n smart-park
kubectl label deployment gateway-canary version=stable -n smart-park
```

**自动回滚配置**：

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gateway
  namespace: smart-park
spec:
  replicas: 2
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  revisionHistoryLimit: 10
  progressDeadlineSeconds: 600
```

**回滚操作**：

```bash
# 查看部署历史
kubectl rollout history deployment/gateway -n smart-park

# 回滚到上一个版本
kubectl rollout undo deployment/gateway -n smart-park

# 回滚到指定版本
kubectl rollout undo deployment/gateway -n smart-park --to-revision=2

# 查看回滚状态
kubectl rollout status deployment/gateway -n smart-park
```

## 五、最佳实践

### 5.1 部署方案选择指南

根据停车场规模和业务需求选择合适的部署方案：

| 规模 | 车位数 | 日均流量 | 推荐方案 | 服务器配置 | 成本估算 |
|------|--------|---------|---------|-----------|---------|
| 小型 | < 100 | < 1000 次 | Docker Compose 单机 | 2核4G | ¥200-500/月 |
| 中型 | 100-500 | 1000-5000 次 | 主备双活 | 2台 4核8G | ¥1000-2000/月 |
| 大型 | > 500 | > 5000 次 | Kubernetes 集群 | 5台 8核16G+ | ¥5000+/月 |

**选择建议**：

1. **小型停车场**：
   - 单机部署即可满足需求
   - 定期备份数据库
   - 配置监控告警

2. **中型连锁**：
   - 主备双活保证业务连续性
   - 数据库主从复制
   - 负载均衡分发流量

3. **大型集团**：
   - Kubernetes 集群管理
   - 多地域部署
   - 自动扩缩容
   - 完善的监控体系

### 5.2 常见问题和解决方案

**问题 1：数据库连接池耗尽**

症状：服务报错 "too many connections"

解决方案：

```yaml
# PostgreSQL 配置
max_connections = 200

# 应用配置
data:
  database:
    max_open_conns: 50
    max_idle_conns: 10
    conn_max_lifetime: 300s
```

**问题 2：Redis 内存溢出**

症状：Redis 报错 "OOM command not allowed when used memory > 'maxmemory'"

解决方案：

```conf
# redis.conf
maxmemory 2gb
maxmemory-policy allkeys-lru
```

**问题 3：服务启动顺序问题**

症状：服务启动失败，依赖服务未就绪

解决方案：

```yaml
# 使用 depends_on + healthcheck
depends_on:
  postgres:
    condition: service_healthy
  redis:
    condition: service_healthy
```

**问题 4：Kubernetes Pod 频繁重启**

症状：Pod 状态频繁变化，服务不稳定

解决方案：

```yaml
# 调整健康检查参数
livenessProbe:
  initialDelaySeconds: 60  # 增加初始延迟
  periodSeconds: 20        # 降低检查频率
  failureThreshold: 5      # 增加失败阈值
```

### 5.3 安全加固建议

**1. 网络安全**

```yaml
# Kubernetes NetworkPolicy
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: smart-park-network-policy
  namespace: smart-park
spec:
  podSelector: {}
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: smart-park
  egress:
  - to:
    - namespaceSelector:
        matchLabels:
          name: smart-park
```

**2. 镜像安全**

```yaml
# 使用只读根文件系统
securityContext:
  readOnlyRootFilesystem: true
  runAsNonRoot: true
  runAsUser: 1000

# 使用安全上下文
securityContext:
  allowPrivilegeEscalation: false
  capabilities:
    drop:
    - ALL
```

**3. 密钥管理**

```bash
# 使用 Sealed Secrets 加密
kubeseal --format=yaml < secret.yaml > sealed-secret.yaml

# 使用 External Secrets Operator
# 从 AWS Secrets Manager / Azure Key Vault 同步密钥
```

**4. RBAC 权限控制**

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: smart-park-role
  namespace: smart-park
rules:
- apiGroups: [""]
  resources: ["pods", "services", "configmaps"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments"]
  verbs: ["get", "list", "watch", "update"]
```

**5. 审计日志**

```yaml
# Kubernetes 审计策略
apiVersion: audit.k8s.io/v1
kind: Policy
rules:
- level: Metadata
  resources:
  - group: ""
    resources: ["secrets"]
```

## 总结

本文系统性地介绍了 Smart Park 智慧停车管理系统从 Docker Compose 单机部署到 Kubernetes 多集群部署的完整演进路径。通过三种不同规模的部署方案，满足了从小型停车场到大型停车集团的多样化需求。

**核心要点回顾**：

1. **小型部署方案**：使用 Docker Compose 实现单机部署，适合快速验证和小规模应用，具有部署简单、资源占用低的优势。

2. **中型部署方案**：采用主备双活架构，通过数据库主从复制、Redis Sentinel 和负载均衡实现高可用，满足业务连续性要求。

3. **大型部署方案**：基于 Kubernetes 构建容器化平台，通过 Deployment、Service、ConfigMap、Secret 和 HPA 实现自动化部署、配置管理和弹性伸缩，支持多集群跨地域部署。

4. **CI/CD 流程**：使用 GitHub Actions 构建完整的持续集成和持续部署流水线，实现从代码提交到生产部署的全自动化，支持灰度发布和快速回滚。

5. **最佳实践**：提供了部署方案选择指南、常见问题解决方案和安全加固建议，帮助运维团队快速定位和解决问题。

**未来展望**：

随着云原生技术的不断发展，Smart Park 的部署方案将持续演进：

1. **Serverless 架构**：探索基于 Knative 的 Serverless 部署，实现按需计费和自动扩缩容到零。

2. **Service Mesh**：引入 Istio 服务网格，实现更精细的流量管理、熔断降级和可观测性。

3. **GitOps**：采用 ArgoCD 实现 GitOps 部署模式，通过 Git 仓库作为唯一事实来源管理集群状态。

4. **多云部署**：支持 AWS、Azure、阿里云等多云部署，避免厂商锁定，实现跨云容灾。

5. **边缘计算**：结合边缘计算技术，将部分计算下沉到停车场本地，降低网络延迟，提升用户体验。

云原生部署是一个持续优化的过程，需要根据业务发展和技术演进不断调整和改进。希望本文能够为运维工程师和 DevOps 从业者提供有价值的参考，帮助大家构建稳定、高效、安全的云原生应用部署体系。

## 参考资料

- [Kubernetes 官方文档](https://kubernetes.io/docs/)
- [Docker Compose 官方文档](https://docs.docker.com/compose/)
- [Kratos 微服务框架](https://go-kratos.dev/)
- [GitHub Actions 文档](https://docs.github.com/en/actions)
- [PostgreSQL 高可用方案](https://www.postgresql.org/docs/current/high-availability.html)
- [Redis Sentinel 文档](https://redis.io/docs/management/sentinel/)
- [Kubernetes HPA 文档](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/)
- [Sealed Secrets](https://github.com/bitnami-labs/sealed-secrets)
- [ArgoCD GitOps 工具](https://argo-cd.readthedocs.io/)

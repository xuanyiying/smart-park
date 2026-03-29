# 回滚方案文档 (Rollback Procedure)

本文档描述 Smart Park 系统的回滚流程和应急预案。

---

## 1. 回滚场景分类

### 1.1 应用层回滚
- 服务启动失败
- 功能异常导致业务中断
- 性能严重下降

### 1.2 数据库回滚
- 数据迁移失败
- 数据不一致
- 紧急数据修复

### 1.3 配置回滚
- 配置错误导致服务异常
- 环境变量配置错误
- Secrets 更新失败

---

## 2. Kubernetes 部署回滚

### 2.1 快速回滚到上一版本

```bash
# 查看部署历史
kubectl rollout history deployment/gateway -n smart-park
kubectl rollout history deployment/vehicle -n smart-park
kubectl rollout history deployment/billing -n smart-park
kubectl rollout history deployment/payment -n smart-park
kubectl rollout history deployment/admin -n smart-park

# 回滚到上一版本
kubectl rollout undo deployment/gateway -n smart-park
kubectl rollout undo deployment/vehicle -n smart-park
kubectl rollout undo deployment/billing -n smart-park
kubectl rollout undo deployment/payment -n smart-park
kubectl rollout undo deployment/admin -n smart-park

# 回滚到指定版本
kubectl rollout undo deployment/gateway -n smart-park --to-revision=2
```

### 2.2 回滚所有服务

```bash
#!/bin/bash
# rollback-all.sh - 回滚所有服务到上一版本

SERVICES=("gateway" "vehicle" "billing" "payment" "admin")

for svc in "${SERVICES[@]}"; do
    echo "Rolling back $svc..."
    kubectl rollout undo deployment/$svc -n smart-park
done

echo "Waiting for rollouts to complete..."
for svc in "${SERVICES[@]}"; do
    kubectl rollout status deployment/$svc -n smart-park
done

echo "Rollback complete!"
```

### 2.3 验证回滚状态

```bash
# 检查 Pod 状态
kubectl get pods -n smart-park

# 检查服务日志
kubectl logs -f deployment/gateway -n smart-park --tail=100

# 检查服务健康状态
kubectl exec -it deployment/gateway -n smart-park -- curl -s http://localhost:8000/health
```

---

## 3. Docker Compose 部署回滚

### 3.1 回滚到指定镜像版本

```bash
# 停止当前服务
docker-compose -f deploy/docker-compose.yml down

# 修改镜像版本标签
# 编辑 docker-compose.yml，将镜像标签改为目标版本
# 例如: smart-park/gateway:v0.3.0 -> smart-park/gateway:v0.2.0

# 重新启动服务
docker-compose -f deploy/docker-compose.yml up -d
```

### 3.2 使用回滚脚本

```bash
#!/bin/bash
# docker-rollback.sh - Docker Compose 回滚脚本

VERSION=${1:-v0.2.0}

echo "Rolling back to version $VERSION..."

# 停止服务
docker-compose -f deploy/docker-compose.yml down

# 更新镜像标签
export IMAGE_TAG=$VERSION

# 重新启动
docker-compose -f deploy/docker-compose.yml up -d

# 等待服务就绪
sleep 30

# 健康检查
for svc in gateway vehicle billing payment admin; do
    curl -s http://localhost:8000/health || echo "$svc health check failed"
done

echo "Rollback to $VERSION complete!"
```

---

## 4. 数据库回滚

### 4.1 Ent 迁移回滚

```bash
# Ent 不支持自动回滚，需要手动执行 SQL

# 1. 备份当前数据库
pg_dump -h localhost -U postgres parking > backup_$(date +%Y%m%d_%H%M%S).sql

# 2. 连接数据库
psql -h localhost -U postgres -d parking

# 3. 执行回滚 SQL (示例)
BEGIN;
-- 回滚具体表结构变更
-- ALTER TABLE ... DROP COLUMN ...;
-- DROP TABLE IF EXISTS new_table;
ROLLBACK; -- 确认无误后使用 COMMIT
```

### 4.2 数据恢复

```bash
# 从备份恢复
psql -h localhost -U postgres -d parking < backup_20240526_120000.sql

# 或使用 pg_restore (自定义格式备份)
pg_restore -h localhost -U postgres -d parking backup.dump
```

### 4.3 紧急数据修复

```sql
-- 修复错误数据
BEGIN;

-- 示例：修复订单状态
UPDATE orders 
SET status = 'pending' 
WHERE id = 'xxx' AND status = 'error';

-- 示例：修复计费金额
UPDATE parking_records
SET fee = calculated_fee
WHERE id = 'xxx';

COMMIT;
```

---

## 5. 配置回滚

### 5.1 ConfigMap 回滚

```bash
# 查看当前 ConfigMap
kubectl get configmap smart-park-config -n smart-park -o yaml

# 从备份恢复 ConfigMap
kubectl apply -f deploy/k8s/configmap-backup.yaml

# 重启服务使配置生效
kubectl rollout restart deployment/gateway -n smart-park
kubectl rollout restart deployment/vehicle -n smart-park
```

### 5.2 Secrets 回滚

```bash
# 查看当前 Secrets (不显示值)
kubectl get secrets smart-park-secrets -n smart-park

# 从备份恢复 Secrets
kubectl apply -f deploy/k8s/secrets-backup.yaml

# 或者重新创建
kubectl delete secrets smart-park-secrets -n smart-park
kubectl create secret generic smart-park-secrets \
  --namespace=smart-park \
  --from-literal=db-password=<password> \
  --from-literal=jwt-secret=<secret>
```

---

## 6. 紧急回滚清单

### 6.1 回滚前检查

- [ ] 确认回滚原因和影响范围
- [ ] 通知相关团队成员
- [ ] 备份当前状态（数据库、配置）
- [ ] 确认目标回滚版本
- [ ] 准备回滚脚本

### 6.2 回滚步骤

1. **停止流量入口**
   ```bash
   # 如果使用 Ingress
   kubectl annotate ingress smart-park-ingress -n smart-park nginx.ingress.kubernetes.io/server-snippet="return 503;"
   ```

2. **执行回滚**
   ```bash
   ./scripts/rollback-all.sh
   ```

3. **验证服务状态**
   ```bash
   kubectl get pods -n smart-park
   kubectl logs -f deployment/gateway -n smart-park --tail=50
   ```

4. **恢复流量**
   ```bash
   kubectl annotate ingress smart-park-ingress -n smart-park nginx.ingress.kubernetes.io/server-snippet-
   ```

5. **监控告警**
   - 检查 Prometheus 指标
   - 验证业务功能
   - 确认无错误日志

### 6.3 回滚后验证

```bash
# 运行冒烟测试
./scripts/smoke-test.sh

# 检查关键业务流程
curl -X POST http://localhost:8000/api/v1/device/entry -d '{"deviceId":"test","plateNumber":"测试车牌"}'
curl http://localhost:8000/api/v1/billing/calculate?recordId=test
```

---

## 7. 回滚脚本集合

### 7.1 完整回滚脚本

```bash
#!/bin/bash
# full-rollback.sh - 完整回滚脚本

set -e

VERSION=${1:-previous}
NAMESPACE="smart-park"
SERVICES=("gateway" "vehicle" "billing" "payment" "admin")

echo "=== Smart Park Rollback Script ==="
echo "Target version: $VERSION"
echo "Namespace: $NAMESPACE"

# 1. 备份当前状态
echo "[1/5] Backing up current state..."
kubectl get all -n $NAMESPACE -o yaml > backup_$(date +%Y%m%d_%H%M%S).yaml

# 2. 停止流量
echo "[2/5] Stopping traffic..."
kubectl scale deployment gateway -n $NAMESPACE --replicas=0

# 3. 执行回滚
echo "[3/5] Rolling back services..."
for svc in "${SERVICES[@]}"; do
    if [ "$VERSION" = "previous" ]; then
        kubectl rollout undo deployment/$svc -n $NAMESPACE
    else
        kubectl set image deployment/$svc $svc=smart-park/$svc:$VERSION -n $NAMESPACE
    fi
done

# 4. 等待就绪
echo "[4/5] Waiting for services to be ready..."
for svc in "${SERVICES[@]}"; do
    kubectl rollout status deployment/$svc -n $NAMESPACE --timeout=300s
done

# 5. 恢复流量
echo "[5/5] Restoring traffic..."
kubectl scale deployment gateway -n $NAMESPACE --replicas=2

# 验证
echo "Verifying deployment..."
kubectl get pods -n $NAMESPACE

echo "=== Rollback Complete ==="
```

### 7.2 冒烟测试脚本

```bash
#!/bin/bash
# smoke-test.sh - 冒烟测试

BASE_URL=${1:-http://localhost:8000}

echo "Running smoke tests against $BASE_URL..."

# 健康检查
echo "Testing health endpoint..."
curl -sf "$BASE_URL/health" || { echo "Health check failed!"; exit 1; }

# 车辆入场
echo "Testing vehicle entry..."
RESPONSE=$(curl -sf -X POST "$BASE_URL/api/v1/device/entry" \
  -H "Content-Type: application/json" \
  -d '{"deviceId":"smoke-test-001","plateNumber":"京A00001"}') || { echo "Entry test failed!"; exit 1; }

echo "Smoke tests passed!"
```

---

## 8. 联系方式

| 角色 | 姓名 | 联系方式 |
|------|------|----------|
| 技术负责人 | - | - |
| 运维负责人 | - | - |
| DBA | - | - |

---

**最后更新**: 2024-05-26  
**维护者**: Smart Park Team

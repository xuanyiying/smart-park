# 📋 发布检查清单 (Release Checklist)

本检查清单确保 Smart Park 项目在每次发布前都经过完整的验证和准备。

---

## ✅ 1. 代码质量检查 (Code Quality)

- [ ] **静态代码分析**
  ```bash
  # Go 代码检查
  golangci-lint run ./...
  
  # 检查是否有未使用的代码
  go vet ./...
  ```

- [ ] **代码格式统一**
  ```bash
  # 格式化所有 Go 文件
  gofmt -w -s .
  
  # 检查格式化差异
  gofmt -d .
  ```

- [ ] **循环复杂度检查**
  - 确保函数圈复杂度 < 10
  - 复杂函数已添加注释说明

- [ ] **安全扫描**
  ```bash
  # 检查依赖漏洞
  govulncheck ./...
  
  # 检查硬编码密钥
  gitleaks detect --source . --verbose
  ```

---

## ✅ 2. 测试覆盖率 (Test Coverage)

- [ ] **单元测试**
  ```bash
  # 运行所有测试并生成覆盖率报告
  go test -v -race -coverprofile=coverage.out ./...
  go tool cover -html=coverage.out -o coverage.html
  ```

- [ ] **覆盖率要求**
  - 核心业务逻辑覆盖率 ≥ 80%
  - 整体项目覆盖率 ≥ 70%
  - 关键路径（计费、支付）覆盖率 ≥ 90%

- [ ] **集成测试**
  ```bash
  # 启动测试环境
  docker-compose -f deploy/docker-compose.yml up -d
  
  # 运行集成测试
  go test -tags=integration ./...
  ```

- [ ] **性能测试**
  ```bash
  # 基准测试
  go test -bench=. -benchmem ./...
  
  # 压力测试（使用 hey 或 ab）
  hey -n 10000 -c 100 http://localhost:8000/api/v1/health
  ```

---

## ✅ 3. 文档完整性 (Documentation)

- [ ] **README 文件**
  - [ ] 中英文 README 已更新
  - [ ] 快速开始指南清晰
  - [ ] 架构图已更新
  - [ ] Badge 显示正确

- [ ] **API 文档**
  - [ ] Proto 文件已更新并生成代码
  - [ ] Swagger/OpenAPI 文档已生成
  - [ ] API 变更已记录

- [ ] **服务文档**
  - [ ] `docs/vehicle.md` 和 `docs/vehicle_EN.md`
  - [ ] `docs/billing.md` 和 `docs/billing_EN.md`
  - [ ] `docs/payment.md` 和 `docs/payment_EN.md`
  - [ ] `docs/admin.md` 和 `docs/admin_EN.md`

- [ ] **部署文档**
  - [ ] Docker 部署文档
  - [ ] Kubernetes 部署文档
  - [ ] 环境变量说明

- [ ] **变更日志**
  - [ ] CHANGELOG.md 已更新
  - [ ] 版本号已更新（遵循 SemVer）
  - [ ] Breaking Changes 已标注

---

## ✅ 4. 依赖管理 (Dependencies)

- [ ] **Go 模块检查**
  ```bash
  # 整理依赖
  go mod tidy
  
  # 验证依赖完整性
  go mod verify
  
  # 检查可升级的依赖
  go list -u -m all
  ```

- [ ] **Docker 镜像版本**
  - [ ] 基础镜像版本已更新到最新稳定版
  - [ ] 无已知安全漏洞的镜像版本

- [ ] **第三方服务依赖**
  - [ ] 微信支付 SDK 版本
  - [ ] 支付宝 SDK 版本
  - [ ] 数据库驱动版本
  - [ ] Redis 客户端版本

---

## ✅ 5. 配置检查 (Configuration)

- [ ] **环境变量**
  - [ ] `.env.example` 已更新
  - [ ] 所有必需的环境变量已文档化
  - [ ] 默认值已合理设置

- [ ] **配置文件**
  ```bash
  # 检查配置文件语法
  # configs/vehicle.yaml
  # configs/billing.yaml
  # configs/payment.yaml
  # configs/admin.yaml
  # configs/gateway.yaml
  ```

- [ ] **Kubernetes 配置**
  - [ ] Namespace 配置正确
  - [ ] ConfigMap 已更新
  - [ ] Secrets 模板已提供
  - [ ] Deployment 资源配置合理
  - [ ] Service 端口配置正确
  - [ ] HPA 策略已配置

- [ ] **数据库迁移**
  ```bash
  # 检查迁移文件
  # 确保所有迁移已应用
  # 回滚脚本已测试
  ```

---

## ✅ 6. 安全审计 (Security Audit)

- [ ] **认证与授权**
  - [ ] JWT token 过期时间合理
  - [ ] 敏感接口已添加权限验证
  - [ ] API 密钥已加密存储

- [ ] **数据加密**
  - [ ] 敏感数据（密码、支付信息）已加密
  - [ ] HTTPS 已配置
  - [ ] 数据库连接使用 SSL

- [ ] **输入验证**
  - [ ] 所有用户输入已验证
  - [ ] SQL 注入防护（使用参数化查询）
  - [ ] XSS 防护

- [ ] **速率限制**
  - [ ] API 限流已配置
  - [ ] 防刷策略已实施

- [ ] **日志脱敏**
  - [ ] 敏感信息不记录到日志
  - [ ] 错误信息不泄露内部细节

---

## ✅ 7. 性能优化 (Performance Optimization)

- [ ] **数据库优化**
  - [ ] 索引已创建
  - [ ] 慢查询已优化
  - [ ] 连接池配置合理
  ```go
  // 检查连接池配置
  db.SetMaxOpenConns(100)
  db.SetMaxIdleConns(25)
  db.SetConnMaxLifetime(5 * time.Minute)
  ```

- [ ] **缓存策略**
  - [ ] Redis 缓存已配置
  - [ ] 缓存过期时间合理
  - [ ] 缓存穿透/击穿/雪崩防护

- [ ] **并发控制**
  - [ ] 分布式锁已实现
  - [ ] 幂等性已保证
  - [ ] 资源竞争已处理

- [ ] **内存优化**
  - [ ] 无内存泄漏
  - [ ] 对象池已合理使用
  - [ ] 大对象已优化

- [ ] **性能指标**
  - [ ] P99 响应时间 < 200ms
  - [ ] QPS ≥ 1000
  - [ ] CPU 使用率 < 70%
  - [ ] 内存使用率 < 80%

---

## ✅ 8. 监控与可观测性 (Monitoring & Observability)

- [ ] **日志系统**
  - [ ] 日志级别配置合理
  - [ ] 结构化日志（JSON 格式）
  - [ ] 日志轮转已配置
  - [ ] 关键操作已记录

- [ ] **指标监控**
  ```bash
  # 检查 Prometheus 指标
  # HTTP 请求指标
  # 数据库连接指标
  # 缓存命中率指标
  # 业务指标（订单量、收入等）
  ```

- [ ] **链路追踪**
  - [ ] OpenTelemetry 已集成
  - [ ] Trace ID 已传递
  - [ ] Jaeger/Zipkin 可查看链路

- [ ] **告警配置**
  - [ ] 错误率告警
  - [ ] 响应时间告警
  - [ ] 资源使用率告警
  - [ ] 业务异常告警

- [ ] **健康检查**
  - [ ] `/health` 端点已实现
  - [ ] `/ready` 端点已实现
  - [ ] Kubernetes 探针已配置

---

## ✅ 9. 发布流程 (Release Process)

- [ ] **版本标签**
  ```bash
  # 创建 Git 标签
  git tag -a v0.3.0 -m "Release v0.3.0: Feature description"
  git push origin v0.3.0
  ```

- [ ] **Docker 镜像构建**
  ```bash
  # 构建并推送镜像
  docker build -t smart-park/vehicle:v0.3.0 -f deploy/docker/vehicle/Dockerfile .
  docker push smart-park/vehicle:v0.3.0
  
  # 所有服务镜像
  # - vehicle-svc
  # - billing-svc
  # - payment-svc
  # - admin-svc
  # - gateway-svc
  ```

- [ ] **GitHub Release**
  - [ ] Release Notes 已编写
  - [ ] 更新内容已列出
  - [ ] Breaking Changes 已说明
  - [ ] 升级指南已提供

- [ ] **回滚方案**
  - [ ] 回滚脚本已准备
  - [ ] 数据库回滚方案已测试
  - [ ] 快速回滚流程已文档化

- [ ] **发布验证**
  - [ ] 冒烟测试通过
  - [ ] 核心功能验证
  - [ ] 监控面板正常
  - [ ] 告警系统正常

- [ ] **通知相关方**
  - [ ] 团队成员已通知
  - [ ] 用户公告已发布（如有）
  - [ ] 文档站点已更新

---

## 📊 发布检查评分表

| 检查项 | 权重 | 得分 | 备注 |
|--------|------|------|------|
| 1. 代码质量 | 15% | /15 | |
| 2. 测试覆盖率 | 20% | /20 | |
| 3. 文档完整性 | 10% | /10 | |
| 4. 依赖管理 | 5% | /5 | |
| 5. 配置检查 | 10% | /10 | |
| 6. 安全审计 | 20% | /20 | **关键** |
| 7. 性能优化 | 10% | /10 | |
| 8. 监控与可观测性 | 5% | /5 | |
| 9. 发布流程 | 5% | /5 | |
| **总计** | **100%** | **/100** | **≥85 分可发布** |

---

## 🚀 快速检查脚本

```bash
#!/bin/bash
# release-check.sh - 快速发布检查脚本

set -e

echo "🔍 开始发布检查..."

# 1. 代码质量
echo "✅ 1. 代码质量检查..."
golangci-lint run ./... || exit 1

# 2. 测试
echo "✅ 2. 运行测试..."
go test -v -race -coverprofile=coverage.out ./... || exit 1

# 3. 依赖
echo "✅ 3. 检查依赖..."
go mod tidy
go mod verify

# 4. 安全
echo "✅ 4. 安全扫描..."
govulncheck ./... || echo "⚠️  发现漏洞，请检查"

# 5. 构建
echo "✅ 5. 构建项目..."
go build -o bin/ ./cmd/... || exit 1

echo "🎉 检查完成！"
```

---

## 📝 检查记录

| 版本号 | 检查日期 | 检查人 | 得分 | 发布状态 | 备注 |
|--------|----------|--------|------|----------|------|
| v0.3.0 | 2024-05-26 | | /100 | ⏳ 待发布 | |
| v0.2.0 | | | /100 | ✅ 已发布 | |

---

## 📚 相关文档

- [部署文档](./deploy/README.md)
- [变更日志](./CHANGELOG.md)
- [版本规范](./VERSIONING.md)
- [回滚流程](./docs/rollback-procedure.md)

---

**最后更新**: 2024-05-26  
**维护者**: Smart Park Team

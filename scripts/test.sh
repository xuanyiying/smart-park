#!/usr/bin/env bash
# 一键测试入口：静态检查 + 单元测试 + 覆盖率门禁
# 用法:
#   ./scripts/test.sh                 # 完整流程（vet + lint(可选) + test + 覆盖率门禁）
#   ./scripts/test.sh --integration   # 额外运行集成测试（需要本地 postgres/redis，可用 deploy/docker-compose.yml 起基础设施）
#   ./scripts/test.sh --no-lint       # 跳过 golangci-lint（未安装时自动跳过）
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

COVERAGE_FILE="coverage.out"
# 棘轮式门禁：以当前水平（排除生成代码后约 16.8%）卡住不回退，随补测逐步上调。
# RELEASE_CHECKLIST 的目标（整体 >=70%、核心业务 >=80%）是终点而非起点。
MIN_COVERAGE=${MIN_COVERAGE:-16}
RUN_INTEGRATION=false
RUN_LINT=true

for arg in "$@"; do
  case "$arg" in
    --integration) RUN_INTEGRATION=true ;;
    --no-lint)     RUN_LINT=false ;;
    *) echo "未知参数: $arg"; echo "用法: $0 [--integration] [--no-lint]"; exit 2 ;;
  esac
done

echo "==> [1/4] go vet ./..."
go vet ./...

if [ "$RUN_LINT" = true ]; then
  if command -v golangci-lint >/dev/null 2>&1; then
    echo "==> [2/4] golangci-lint run"
    golangci-lint run --timeout=5m
  else
    echo "==> [2/4] 未安装 golangci-lint，跳过（brew install golangci-lint）"
  fi
else
  echo "==> [2/4] 按参数跳过 lint"
fi

echo "==> [3/4] go test（单元测试，-race + 覆盖率）"
rm -f "$COVERAGE_FILE"
go test -race -count=1 -covermode=atomic -coverprofile="$COVERAGE_FILE" ./...

if [ "$RUN_INTEGRATION" = true ]; then
  echo "==> [3.5/4] go test -tags=integration（集成测试，需 postgres/redis）"
  INTEGRATION=1 go test -tags=integration -count=1 ./tests/...
fi

echo "==> [4/4] 覆盖率门禁（阈值 ${MIN_COVERAGE}%，已排除 ent/pb 等生成代码）"
grep -Ev "/data/ent/|\.pb\.go|\.pb\.gw\.go" "$COVERAGE_FILE" > coverage.filtered.out
TOTAL=$(go tool cover -func=coverage.filtered.out | tail -1 | awk '{print substr($3, 1, length($3)-1)}')
echo "总覆盖率: ${TOTAL}%"
awk -v t="$TOTAL" -v m="$MIN_COVERAGE" 'BEGIN { exit (t+0 >= m+0) ? 0 : 1 }' \
  || { echo "❌ 覆盖率 ${TOTAL}% 低于门禁 ${MIN_COVERAGE}%"; exit 1; }
echo "✅ 测试通过，覆盖率 ${TOTAL}% >= ${MIN_COVERAGE}%"

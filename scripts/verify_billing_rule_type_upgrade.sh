#!/usr/bin/env bash
# billing rule_type 枚举扩展升级验证脚本（对应 2026-09 修复：
# rule_type 曾缺失 base/discount/exemption/override，导致优惠规则落库降级为 time）。
#
# 结论：ent 的 enum 在 PostgreSQL 中存储为 varchar（无原生 ENUM 类型、无 CHECK 约束），
# 升级无需任何 DDL，部署新版本代码即可写入新枚举值。本脚本用于升级前人工核验。
#
# 用法: ./scripts/verify_billing_rule_type_upgrade.sh [数据库名，默认 parking]
#   可通过 PGHOST/PGPORT/PGUSER/PGPASSWORD 覆盖连接参数
set -euo pipefail

DB_NAME="${1:-parking}"
PSQL="docker exec -i infra-postgres psql -U postgres -d $DB_NAME -tAc"
if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^infra-postgres$'; then
  # 非 docker 环境：直接使用本机 psql
  PSQL="psql -d $DB_NAME -tAc"
fi

echo "==> 检查 $DB_NAME.billing_rules.rule_type 列类型"
COL_TYPE=$($PSQL "SELECT data_type FROM information_schema.columns WHERE table_name='billing_rules' AND column_name='rule_type'")
if [ "$COL_TYPE" != "character varying" ]; then
  echo "❌ rule_type 不是 varchar（实际: $COL_TYPE），需要人工评估迁移方案"
  exit 1
fi
echo "✅ rule_type 为 varchar，接受任意字符串值"

echo "==> 检查是否存在 CHECK 约束限制枚举值"
CHECKS=$($PSQL "SELECT count(*) FROM pg_constraint WHERE conrelid='billing_rules'::regclass AND contype='c' AND pg_get_constraintdef(oid) LIKE '%rule_type%'")
if [ "$CHECKS" != "0" ]; then
  echo "❌ 存在 rule_type 相关 CHECK 约束，需要先 DROP CONSTRAINT"
  $PSQL "SELECT conname, pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='billing_rules'::regclass AND contype='c'"
  exit 1
fi
echo "✅ 无 CHECK 约束"

echo "==> 验证新枚举值可写入（在事务中回滚，不产生数据）"
INSERT_OUT=$($PSQL "
BEGIN;
INSERT INTO billing_rules (id, tenant_id, lot_id, rule_name, rule_type, priority, is_active, created_at, updated_at)
VALUES (gen_random_uuid(), gen_random_uuid(), gen_random_uuid(), 'upgrade-check', 'discount', 0, true, now(), now())
RETURNING rule_type;
ROLLBACK;" 2>&1)
if ! echo "$INSERT_OUT" | grep -q '^discount$'; then
  echo "❌ 新枚举值写入失败: $INSERT_OUT"
  exit 1
fi
echo "✅ 新枚举值（discount 等）可直接写入"
echo ""
echo "结论：无需 DDL 迁移，直接部署新版本 billing 服务即可。"

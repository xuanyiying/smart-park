-- =============================================================================
-- H1 金额 float64(元) → int64(分) 数据迁移
--
-- 说明：
--   * 本脚本适用于"金额字段当前存的是元，需要按 ×100 转换为分"的存量库。
--   * 若存量字段已经是分（从未在旧格式下运行过），请勿执行，直接由
--     ent.Schema.Create 的自动迁移把列类型 float8 → bigint 即可（列内数值不变）。
--   * 迁移必须先于新版本服务启动执行；新代码以分为单位读写这些列，
--     未迁移的存量行会被当作巨额的"分"处理，导致对账与退款异常。
--
-- 执行方式（在应用服务停止后、新版本启动前）：
--   psql "$SP_DATABASE_SOURCE" -f scripts/migrate_amount_to_cents.sql
-- 或：
--   docker exec -i smart-park-postgres psql -U postgres -d parking \
--       -f /dev/stdin < scripts/migrate_amount_to_cents.sql
--
-- 幂等性：每个表都先用 pg_typeof 探测，只在列仍是 double precision 时执行，
-- 重复执行安全。
-- =============================================================================

BEGIN;

-- 1) 支付服务：订单金额
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='orders' AND column_name='amount'
                 AND data_type='double precision') THEN
        ALTER TABLE orders ALTER COLUMN amount TYPE bigint USING (amount * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='orders' AND column_name='discount_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE orders ALTER COLUMN discount_amount TYPE bigint USING (discount_amount * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='orders' AND column_name='final_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE orders ALTER COLUMN final_amount TYPE bigint USING (final_amount * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='orders' AND column_name='paid_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE orders ALTER COLUMN paid_amount TYPE bigint USING (paid_amount * 100);
    END IF;
END $$;

-- 2) 支付服务：对账记录
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='reconciliations' AND column_name='order_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE reconciliations ALTER COLUMN order_amount TYPE bigint USING (order_amount * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='reconciliations' AND column_name='paid_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE reconciliations ALTER COLUMN paid_amount TYPE bigint USING (paid_amount * 100);
    END IF;
END $$;

-- 3) 支付服务：退款审批
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='refund_approvals' AND column_name='amount'
                 AND data_type='double precision') THEN
        ALTER TABLE refund_approvals ALTER COLUMN amount TYPE bigint USING (amount * 100);
    END IF;
END $$;

-- 4) 充电服务：充电会话金额
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='sessions' AND column_name='cost'
                 AND data_type='double precision') THEN
        ALTER TABLE sessions ALTER COLUMN cost TYPE bigint USING (cost * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='sessions' AND column_name='service_fee'
                 AND data_type='double precision') THEN
        ALTER TABLE sessions ALTER COLUMN service_fee TYPE bigint USING (service_fee * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='sessions' AND column_name='total_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE sessions ALTER COLUMN total_amount TYPE bigint USING (total_amount * 100);
    END IF;
END $$;

-- 5) 车辆服务：离线同步金额
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='offline_sync_records' AND column_name='sync_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE offline_sync_records ALTER COLUMN sync_amount TYPE bigint USING (sync_amount * 100);
    END IF;
END $$;

-- 6) 分析服务：统计金额
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='daily_stats' AND column_name='total_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE daily_stats ALTER COLUMN total_amount TYPE bigint USING (total_amount * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='daily_stats' AND column_name='total_discount'
                 AND data_type='double precision') THEN
        ALTER TABLE daily_stats ALTER COLUMN total_discount TYPE bigint USING (total_discount * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='daily_stats' AND column_name='net_amount'
                 AND data_type='double precision') THEN
        ALTER TABLE daily_stats ALTER COLUMN net_amount TYPE bigint USING (net_amount * 100);
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.columns
               WHERE table_schema='public' AND table_name='hourly_stats' AND column_name='revenue'
                 AND data_type='double precision') THEN
        ALTER TABLE hourly_stats ALTER COLUMN revenue TYPE bigint USING (revenue * 100);
    END IF;
END $$;

COMMIT;

-- 校验：应输出 bigint
SELECT table_name, column_name, data_type
FROM information_schema.columns
WHERE table_schema='public'
  AND column_name IN ('amount','final_amount','paid_amount','order_amount','discount_amount',
                      'total_amount','total_discount','net_amount','cost','service_fee','sync_amount','revenue')
ORDER BY table_name, column_name;

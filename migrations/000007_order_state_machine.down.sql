DROP INDEX IF EXISTS idx_orders_status_updated_at;
DROP INDEX IF EXISTS idx_orders_one_active_per_driver;

ALTER TABLE orders
    DROP COLUMN IF EXISTS version;

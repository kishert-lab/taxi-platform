ALTER TABLE taxi_park_settings
    DROP CONSTRAINT IF EXISTS taxi_park_settings_scheduled_positive,
    DROP COLUMN IF EXISTS allow_scheduled_driver_preassignment,
    DROP COLUMN IF EXISTS scheduled_expire_after_minutes,
    DROP COLUMN IF EXISTS scheduled_activation_before_minutes,
    DROP COLUMN IF EXISTS scheduled_min_before_minutes,
    DROP COLUMN IF EXISTS scheduled_orders_enabled;

DROP INDEX IF EXISTS idx_orders_preassigned_driver_scheduled_at;
DROP INDEX IF EXISTS idx_orders_taxi_park_scheduled_at;
DROP INDEX IF EXISTS idx_orders_activation_at;
DROP INDEX IF EXISTS idx_orders_scheduled_at;
DROP INDEX IF EXISTS idx_orders_scheduled_status;
DROP INDEX IF EXISTS idx_orders_order_type;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_scheduled_status_check,
    DROP CONSTRAINT IF EXISTS orders_order_type_check,
    DROP COLUMN IF EXISTS scheduled_expired_at,
    DROP COLUMN IF EXISTS scheduled_cancelled_at,
    DROP COLUMN IF EXISTS activated_at,
    DROP COLUMN IF EXISTS scheduled_cancel_reason,
    DROP COLUMN IF EXISTS scheduled_created_by,
    DROP COLUMN IF EXISTS preassigned_driver_id,
    DROP COLUMN IF EXISTS scheduled_status,
    DROP COLUMN IF EXISTS scheduled_timezone,
    DROP COLUMN IF EXISTS activation_at,
    DROP COLUMN IF EXISTS scheduled_at,
    DROP COLUMN IF EXISTS order_type;

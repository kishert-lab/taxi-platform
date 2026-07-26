DROP INDEX IF EXISTS idx_orders_assigned_tariff_id;

ALTER TABLE orders
    DROP COLUMN IF EXISTS actual_duration_seconds,
    DROP COLUMN IF EXISTS actual_distance_meters,
    DROP COLUMN IF EXISTS assigned_tariff_id;

DROP INDEX IF EXISTS idx_taxi_park_tariffs_active_class;

ALTER TABLE taxi_park_tariffs
    DROP CONSTRAINT IF EXISTS taxi_park_tariffs_pricing_values_check,
    DROP CONSTRAINT IF EXISTS taxi_park_tariffs_fixed_price_non_negative,
    DROP CONSTRAINT IF EXISTS taxi_park_tariffs_pricing_mode_check;

ALTER TABLE taxi_park_tariffs
    DROP COLUMN IF EXISTS fixed_price_cents,
    DROP COLUMN IF EXISTS pricing_mode,
    DROP COLUMN IF EXISTS car_class_id;

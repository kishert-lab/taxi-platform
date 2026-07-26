ALTER TABLE taxi_park_tariffs
    ADD COLUMN IF NOT EXISTS car_class_id UUID REFERENCES car_classes(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS pricing_mode TEXT NOT NULL DEFAULT 'distance_time',
    ADD COLUMN IF NOT EXISTS fixed_price_cents BIGINT NOT NULL DEFAULT 0;

UPDATE taxi_park_tariffs tariff
SET car_class_id = class.id
FROM car_classes class
WHERE tariff.car_class_id IS NULL
  AND class.code = 'economy'
  AND class.deleted_at IS NULL;

ALTER TABLE taxi_park_tariffs
    DROP CONSTRAINT IF EXISTS taxi_park_tariffs_pricing_mode_check,
    DROP CONSTRAINT IF EXISTS taxi_park_tariffs_fixed_price_non_negative,
    DROP CONSTRAINT IF EXISTS taxi_park_tariffs_pricing_values_check;

ALTER TABLE taxi_park_tariffs
    ADD CONSTRAINT taxi_park_tariffs_pricing_mode_check CHECK (
        pricing_mode IN ('fixed', 'distance', 'time', 'distance_time')
    ),
    ADD CONSTRAINT taxi_park_tariffs_fixed_price_non_negative CHECK (fixed_price_cents >= 0),
    ADD CONSTRAINT taxi_park_tariffs_pricing_values_check CHECK (
        (
            pricing_mode = 'fixed'
            AND fixed_price_cents > 0
        )
        OR (
            pricing_mode = 'distance'
            AND price_per_km_cents > 0
        )
        OR (
            pricing_mode = 'time'
            AND price_per_minute_cents > 0
        )
        OR (
            pricing_mode = 'distance_time'
            AND (price_per_km_cents > 0 OR price_per_minute_cents > 0)
        )
    );

CREATE INDEX IF NOT EXISTS idx_taxi_park_tariffs_active_class
ON taxi_park_tariffs (taxi_park_id, car_class_id)
WHERE is_active = true;

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS assigned_tariff_id UUID REFERENCES taxi_park_tariffs(id) ON DELETE RESTRICT,
    ADD COLUMN IF NOT EXISTS actual_distance_meters BIGINT,
    ADD COLUMN IF NOT EXISTS actual_duration_seconds BIGINT;

CREATE INDEX IF NOT EXISTS idx_orders_assigned_tariff_id
ON orders (assigned_tariff_id)
WHERE assigned_tariff_id IS NOT NULL;

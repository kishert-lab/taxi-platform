DROP INDEX IF EXISTS taxi_park_tariffs_active_priority_unique;
ALTER TABLE orders DROP COLUMN IF EXISTS fare_mode;
ALTER TABLE orders DROP COLUMN IF EXISTS agreed_price_cents;
ALTER TABLE taxi_park_tariffs
    DROP CONSTRAINT IF EXISTS taxi_park_tariffs_fare_mode_check,
    DROP COLUMN IF EXISTS priority,
    DROP COLUMN IF EXISTS fare_mode;

-- Existing orders used their selected tariff with actual trip metrics at completion.
-- Preserve that behavior explicitly for preexisting tariffs.
ALTER TABLE taxi_park_tariffs
    ADD COLUMN fare_mode TEXT NOT NULL DEFAULT 'metered',
    ADD COLUMN priority INTEGER NOT NULL DEFAULT 0;

ALTER TABLE taxi_park_tariffs
    ADD CONSTRAINT taxi_park_tariffs_fare_mode_check
    CHECK (fare_mode IN ('fixed_quote', 'metered'));

-- Make the old newest-first selection deterministic and explicit before
-- enforcing one winner per park, class, fare mode and priority.
WITH ranked AS (
    SELECT id,
           row_number() OVER (
               PARTITION BY taxi_park_id, car_class_id, fare_mode
               ORDER BY created_at, id
           ) - 1 AS assigned_priority
    FROM taxi_park_tariffs
)
UPDATE taxi_park_tariffs tariff
SET priority = ranked.assigned_priority
FROM ranked
WHERE tariff.id = ranked.id;

CREATE UNIQUE INDEX taxi_park_tariffs_active_priority_unique
    ON taxi_park_tariffs (taxi_park_id, car_class_id, fare_mode, priority)
    WHERE is_active = true AND car_class_id IS NOT NULL;

ALTER TABLE orders
    ADD COLUMN fare_mode TEXT NOT NULL DEFAULT 'metered'
    CHECK (fare_mode IN ('fixed_quote', 'metered'));

ALTER TABLE orders
    ADD COLUMN agreed_price_cents BIGINT
    CHECK (agreed_price_cents IS NULL OR agreed_price_cents > 0);

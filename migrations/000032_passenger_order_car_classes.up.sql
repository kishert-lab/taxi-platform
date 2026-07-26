CREATE TABLE car_classes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    base_price NUMERIC(12,2) NOT NULL,
    price_per_km NUMERIC(12,2) NOT NULL,
    price_per_minute NUMERIC(12,2) NOT NULL,
    minimum_price NUMERIC(12,2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    sort_order INTEGER NOT NULL DEFAULT 100,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uq_car_classes_code UNIQUE (code),
    CONSTRAINT car_classes_prices_non_negative CHECK (
        base_price >= 0 AND price_per_km >= 0 AND price_per_minute >= 0 AND minimum_price >= 0
    )
);

CREATE TRIGGER car_classes_set_updated_at
BEFORE UPDATE ON car_classes
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_car_classes_active_sort
ON car_classes (is_active, sort_order)
WHERE deleted_at IS NULL;

INSERT INTO car_classes (code, name, description, base_price, price_per_km, price_per_minute, minimum_price, is_active, sort_order)
VALUES
    ('economy', 'Эконом', 'Базовый класс автомобиля', 120, 18, 6, 180, TRUE, 10),
    ('comfort', 'Комфорт', 'Повышенный комфорт поездки', 180, 24, 8, 260, TRUE, 20),
    ('business', 'Бизнес', 'Автомобили бизнес-класса', 300, 35, 12, 420, TRUE, 30),
    ('minivan', 'Минивэн', 'Поездки для компаний и багажа', 260, 30, 10, 360, TRUE, 40)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS car_id UUID NULL REFERENCES cars(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS park_id UUID NULL REFERENCES taxi_parks(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS car_class_id UUID NULL REFERENCES car_classes(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS pickup_entrance TEXT NULL,
    ADD COLUMN IF NOT EXISTS pickup_comment TEXT NULL,
    ADD COLUMN IF NOT EXISTS passenger_location_sharing_enabled BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_orders_car_class_id
ON orders (car_class_id)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_orders_park_id
ON orders (park_id)
WHERE deleted_at IS NULL;

CREATE TEMP TABLE tmp_legacy_passenger_map (
    legacy_user_id UUID PRIMARY KEY,
    target_passenger_id UUID NOT NULL
) ON COMMIT DROP;

INSERT INTO tmp_legacy_passenger_map (legacy_user_id, target_passenger_id)
SELECT refs.legacy_user_id,
       COALESCE(existing_passenger.id, refs.legacy_user_id) AS target_passenger_id
FROM (
    SELECT DISTINCT passenger_id AS legacy_user_id
    FROM orders
    WHERE passenger_id IS NOT NULL
    UNION
    SELECT DISTINCT passenger_id AS legacy_user_id
    FROM order_ratings
    WHERE passenger_id IS NOT NULL
    UNION
    SELECT DISTINCT passenger_id AS legacy_user_id
    FROM passenger_ratings
    WHERE passenger_id IS NOT NULL
    UNION
    SELECT DISTINCT passenger_id AS legacy_user_id
    FROM chat_threads
    WHERE passenger_id IS NOT NULL
    UNION
    SELECT DISTINCT passenger_id AS legacy_user_id
    FROM order_financial_transactions
    WHERE passenger_id IS NOT NULL
) refs
JOIN users legacy_user ON legacy_user.id = refs.legacy_user_id
LEFT JOIN passengers existing_passenger
    ON existing_passenger.phone = legacy_user.phone
   AND existing_passenger.deleted_at IS NULL;

INSERT INTO passengers (
    id,
    phone,
    name,
    email,
    avatar_url,
    is_active,
    phone_verified_at,
    last_login_at,
    created_at,
    updated_at
)
SELECT
    legacy_user.id,
    legacy_user.phone,
    NULLIF(trim(concat_ws(' ', COALESCE(legacy_user.first_name, ''), COALESCE(legacy_user.last_name, ''))), '') AS name,
    NULLIF(legacy_user.email, '') AS email,
    NULLIF(legacy_user.profile_photo_url, '') AS avatar_url,
    legacy_user.is_active,
    legacy_user.phone_confirmed_at,
    legacy_user.last_login_at,
    legacy_user.created_at,
    legacy_user.updated_at
FROM tmp_legacy_passenger_map mapping
JOIN users legacy_user ON legacy_user.id = mapping.legacy_user_id
LEFT JOIN passengers existing_passenger ON existing_passenger.id = mapping.target_passenger_id
WHERE mapping.target_passenger_id = mapping.legacy_user_id
  AND existing_passenger.id IS NULL;

UPDATE orders record
SET passenger_id = mapping.target_passenger_id
FROM tmp_legacy_passenger_map mapping
WHERE record.passenger_id = mapping.legacy_user_id
  AND record.passenger_id <> mapping.target_passenger_id;

UPDATE order_ratings record
SET passenger_id = mapping.target_passenger_id
FROM tmp_legacy_passenger_map mapping
WHERE record.passenger_id = mapping.legacy_user_id
  AND record.passenger_id <> mapping.target_passenger_id;

UPDATE passenger_ratings record
SET passenger_id = mapping.target_passenger_id
FROM tmp_legacy_passenger_map mapping
WHERE record.passenger_id = mapping.legacy_user_id
  AND record.passenger_id <> mapping.target_passenger_id;

UPDATE chat_threads record
SET passenger_id = mapping.target_passenger_id
FROM tmp_legacy_passenger_map mapping
WHERE record.passenger_id = mapping.legacy_user_id
  AND record.passenger_id <> mapping.target_passenger_id;

UPDATE order_financial_transactions record
SET passenger_id = mapping.target_passenger_id
FROM tmp_legacy_passenger_map mapping
WHERE record.passenger_id = mapping.legacy_user_id
  AND record.passenger_id <> mapping.target_passenger_id;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_passenger_id_fkey;
ALTER TABLE orders
    ADD CONSTRAINT orders_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES passengers(id) ON DELETE RESTRICT;

ALTER TABLE order_ratings DROP CONSTRAINT IF EXISTS order_ratings_passenger_id_fkey;
ALTER TABLE order_ratings
    ADD CONSTRAINT order_ratings_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES passengers(id) ON DELETE RESTRICT;

ALTER TABLE passenger_ratings DROP CONSTRAINT IF EXISTS passenger_ratings_passenger_id_fkey;
ALTER TABLE passenger_ratings
    ADD CONSTRAINT passenger_ratings_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES passengers(id) ON DELETE RESTRICT;

ALTER TABLE chat_threads DROP CONSTRAINT IF EXISTS chat_threads_passenger_id_fkey;
ALTER TABLE chat_threads
    ADD CONSTRAINT chat_threads_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES passengers(id) ON DELETE SET NULL;

ALTER TABLE order_financial_transactions DROP CONSTRAINT IF EXISTS order_financial_transactions_passenger_id_fkey;
ALTER TABLE order_financial_transactions
    ADD CONSTRAINT order_financial_transactions_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES passengers(id) ON DELETE SET NULL;

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

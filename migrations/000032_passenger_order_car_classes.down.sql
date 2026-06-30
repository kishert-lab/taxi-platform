ALTER TABLE order_financial_transactions DROP CONSTRAINT IF EXISTS order_financial_transactions_passenger_id_fkey;
ALTER TABLE order_financial_transactions
    ADD CONSTRAINT order_financial_transactions_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE chat_threads DROP CONSTRAINT IF EXISTS chat_threads_passenger_id_fkey;
ALTER TABLE chat_threads
    ADD CONSTRAINT chat_threads_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE passenger_ratings DROP CONSTRAINT IF EXISTS passenger_ratings_passenger_id_fkey;
ALTER TABLE passenger_ratings
    ADD CONSTRAINT passenger_ratings_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE order_ratings DROP CONSTRAINT IF EXISTS order_ratings_passenger_id_fkey;
ALTER TABLE order_ratings
    ADD CONSTRAINT order_ratings_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES users(id) ON DELETE RESTRICT;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_passenger_id_fkey;
ALTER TABLE orders
    ADD CONSTRAINT orders_passenger_id_fkey
    FOREIGN KEY (passenger_id) REFERENCES users(id) ON DELETE RESTRICT;

DROP INDEX IF EXISTS idx_orders_park_id;
DROP INDEX IF EXISTS idx_orders_car_class_id;

ALTER TABLE orders
    DROP COLUMN IF EXISTS passenger_location_sharing_enabled,
    DROP COLUMN IF EXISTS pickup_comment,
    DROP COLUMN IF EXISTS pickup_entrance,
    DROP COLUMN IF EXISTS car_class_id,
    DROP COLUMN IF EXISTS park_id,
    DROP COLUMN IF EXISTS car_id;

DROP INDEX IF EXISTS idx_car_classes_active_sort;
DROP TRIGGER IF EXISTS car_classes_set_updated_at ON car_classes;
DROP TABLE IF EXISTS car_classes;

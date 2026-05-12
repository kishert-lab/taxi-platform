ALTER TABLE orders
    ADD COLUMN version INTEGER NOT NULL DEFAULT 1;

CREATE UNIQUE INDEX idx_orders_one_active_per_driver
ON orders (driver_id)
WHERE driver_id IS NOT NULL
  AND status IN ('driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress')
  AND deleted_at IS NULL;

CREATE INDEX idx_orders_status_updated_at
ON orders (status, updated_at)
WHERE deleted_at IS NULL;

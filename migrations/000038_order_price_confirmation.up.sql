ALTER TABLE orders
    ADD COLUMN price_confirmation_state TEXT,
    ADD COLUMN proposed_price_cents BIGINT,
    ADD COLUMN price_confirmation_expires_at TIMESTAMPTZ,
    ADD COLUMN declined_driver_ids UUID[] NOT NULL DEFAULT '{}';

ALTER TABLE orders
    ADD CONSTRAINT orders_price_confirmation_state_check
    CHECK (price_confirmation_state IS NULL OR price_confirmation_state IN ('pending', 'confirmed'));

ALTER TABLE orders
    ADD CONSTRAINT orders_proposed_price_positive_check
    CHECK (proposed_price_cents IS NULL OR proposed_price_cents > 0);

CREATE INDEX orders_pending_price_confirmation_idx
    ON orders (price_confirmation_expires_at)
    WHERE price_confirmation_state = 'pending';

CREATE UNIQUE INDEX orders_one_active_order_per_driver_idx
    ON orders (driver_id)
    WHERE driver_id IS NOT NULL AND deleted_at IS NULL
      AND status IN ('driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress');

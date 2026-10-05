-- Migration 000038 may already be recorded on installations that received an
-- earlier version of that file. Repair the required schema without rewriting
-- migration history.
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS price_confirmation_state TEXT,
    ADD COLUMN IF NOT EXISTS proposed_price_cents BIGINT,
    ADD COLUMN IF NOT EXISTS price_confirmation_expires_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS declined_driver_ids UUID[] NOT NULL DEFAULT '{}';

UPDATE orders
SET declined_driver_ids = '{}'
WHERE declined_driver_ids IS NULL;

ALTER TABLE orders
    ALTER COLUMN declined_driver_ids SET DEFAULT '{}',
    ALTER COLUMN declined_driver_ids SET NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'orders_price_confirmation_state_check'
    ) THEN
        ALTER TABLE orders
            ADD CONSTRAINT orders_price_confirmation_state_check
            CHECK (price_confirmation_state IS NULL OR price_confirmation_state IN ('pending', 'confirmed'));
    END IF;

    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'orders_proposed_price_positive_check'
    ) THEN
        ALTER TABLE orders
            ADD CONSTRAINT orders_proposed_price_positive_check
            CHECK (proposed_price_cents IS NULL OR proposed_price_cents > 0);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS orders_pending_price_confirmation_idx
    ON orders (price_confirmation_expires_at)
    WHERE price_confirmation_state = 'pending';

CREATE UNIQUE INDEX IF NOT EXISTS orders_one_active_order_per_driver_idx
    ON orders (driver_id)
    WHERE driver_id IS NOT NULL AND deleted_at IS NULL
      AND status IN ('driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress');

DROP INDEX IF EXISTS orders_pending_price_confirmation_idx;
DROP INDEX IF EXISTS orders_one_active_order_per_driver_idx;
ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_proposed_price_positive_check,
    DROP CONSTRAINT IF EXISTS orders_price_confirmation_state_check,
    DROP COLUMN IF EXISTS price_confirmation_expires_at,
    DROP COLUMN IF EXISTS proposed_price_cents,
    DROP COLUMN IF EXISTS price_confirmation_state,
    DROP COLUMN IF EXISTS declined_driver_ids;

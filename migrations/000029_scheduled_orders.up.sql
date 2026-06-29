ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS order_type VARCHAR(50) NOT NULL DEFAULT 'instant',
    ADD COLUMN IF NOT EXISTS scheduled_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS activation_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS scheduled_timezone VARCHAR(100) NULL,
    ADD COLUMN IF NOT EXISTS scheduled_status VARCHAR(50) NULL,
    ADD COLUMN IF NOT EXISTS preassigned_driver_id UUID NULL REFERENCES drivers(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS scheduled_created_by UUID NULL REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS scheduled_cancel_reason TEXT NULL,
    ADD COLUMN IF NOT EXISTS activated_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS scheduled_cancelled_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS scheduled_expired_at TIMESTAMPTZ NULL;

ALTER TABLE orders
    DROP CONSTRAINT IF EXISTS orders_order_type_check,
    DROP CONSTRAINT IF EXISTS orders_scheduled_status_check;

ALTER TABLE orders
    ADD CONSTRAINT orders_order_type_check CHECK (order_type IN ('instant', 'scheduled')),
    ADD CONSTRAINT orders_scheduled_status_check CHECK (
        scheduled_status IS NULL OR scheduled_status IN (
            'scheduled_new',
            'scheduled_confirmed',
            'scheduled_driver_assigned',
            'scheduled_waiting_activation',
            'scheduled_activated',
            'scheduled_cancelled',
            'scheduled_expired',
            'scheduled_failed'
        )
    );

CREATE INDEX IF NOT EXISTS idx_orders_order_type ON orders(order_type);
CREATE INDEX IF NOT EXISTS idx_orders_scheduled_status ON orders(scheduled_status);
CREATE INDEX IF NOT EXISTS idx_orders_scheduled_at ON orders(scheduled_at);
CREATE INDEX IF NOT EXISTS idx_orders_activation_at ON orders(activation_at);
CREATE INDEX IF NOT EXISTS idx_orders_taxi_park_scheduled_at ON orders(((metadata->>'taxi_park_id')), scheduled_at);
CREATE INDEX IF NOT EXISTS idx_orders_preassigned_driver_scheduled_at ON orders(preassigned_driver_id, scheduled_at);

ALTER TABLE taxi_park_settings
    ADD COLUMN IF NOT EXISTS scheduled_orders_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS scheduled_min_before_minutes INTEGER NOT NULL DEFAULT 15,
    ADD COLUMN IF NOT EXISTS scheduled_activation_before_minutes INTEGER NOT NULL DEFAULT 20,
    ADD COLUMN IF NOT EXISTS scheduled_expire_after_minutes INTEGER NOT NULL DEFAULT 15,
    ADD COLUMN IF NOT EXISTS allow_scheduled_driver_preassignment BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE taxi_park_settings
    DROP CONSTRAINT IF EXISTS taxi_park_settings_scheduled_positive;

ALTER TABLE taxi_park_settings
    ADD CONSTRAINT taxi_park_settings_scheduled_positive CHECK (
        scheduled_min_before_minutes > 0
        AND scheduled_activation_before_minutes > 0
        AND scheduled_expire_after_minutes > 0
    );

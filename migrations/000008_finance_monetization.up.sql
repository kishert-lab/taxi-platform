ALTER TABLE taxi_parks
    ADD COLUMN IF NOT EXISTS commission_percent NUMERIC(5,2),
    ADD COLUMN IF NOT EXISTS balance_cents BIGINT NOT NULL DEFAULT 0;

ALTER TABLE drivers
    ADD COLUMN IF NOT EXISTS taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS commission_percent NUMERIC(5,2);

ALTER TABLE cities
    ADD COLUMN IF NOT EXISTS commission_percent NUMERIC(5,2);

ALTER TABLE tariffs
    ADD COLUMN IF NOT EXISTS commission_percent NUMERIC(5,2);

ALTER TABLE taxi_parks
    ADD CONSTRAINT taxi_parks_commission_percent_range
    CHECK (commission_percent IS NULL OR (commission_percent >= 0 AND commission_percent <= 100));

ALTER TABLE drivers
    ADD CONSTRAINT drivers_commission_percent_range
    CHECK (commission_percent IS NULL OR (commission_percent >= 0 AND commission_percent <= 100));

ALTER TABLE cities
    ADD CONSTRAINT cities_commission_percent_range
    CHECK (commission_percent IS NULL OR (commission_percent >= 0 AND commission_percent <= 100));

ALTER TABLE tariffs
    ADD CONSTRAINT tariffs_commission_percent_range
    CHECK (commission_percent IS NULL OR (commission_percent >= 0 AND commission_percent <= 100));

CREATE TABLE driver_balances (
    driver_id UUID PRIMARY KEY REFERENCES drivers(id) ON DELETE RESTRICT,
    available_balance_cents BIGINT NOT NULL DEFAULT 0,
    pending_balance_cents BIGINT NOT NULL DEFAULT 0,
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    version INTEGER NOT NULL DEFAULT 1,
    CONSTRAINT driver_balances_non_negative CHECK (available_balance_cents >= 0 AND pending_balance_cents >= 0)
);

CREATE TRIGGER driver_balances_set_updated_at
BEFORE UPDATE ON driver_balances
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE financial_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID REFERENCES orders(id) ON DELETE RESTRICT,
    driver_id UUID REFERENCES drivers(id) ON DELETE RESTRICT,
    taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE RESTRICT,
    transaction_type TEXT NOT NULL,
    gross_amount_cents BIGINT NOT NULL DEFAULT 0,
    commission_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
    commission_amount_cents BIGINT NOT NULL DEFAULT 0,
    net_amount_cents BIGINT NOT NULL DEFAULT 0,
    currency CHAR(3) NOT NULL DEFAULT 'RUB',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT financial_transactions_type_check CHECK (
        transaction_type IN ('commission', 'driver_income', 'refund', 'manual_adjustment', 'withdrawal')
    ),
    CONSTRAINT financial_transactions_amounts_non_negative CHECK (
        gross_amount_cents >= 0 AND commission_amount_cents >= 0 AND net_amount_cents >= 0
    )
);

CREATE UNIQUE INDEX idx_financial_transactions_order_type
ON financial_transactions (order_id, transaction_type)
WHERE order_id IS NOT NULL;

CREATE INDEX idx_financial_transactions_driver_created_at
ON financial_transactions (driver_id, created_at DESC)
WHERE driver_id IS NOT NULL;

CREATE INDEX idx_financial_transactions_taxi_park_created_at
ON financial_transactions (taxi_park_id, created_at DESC)
WHERE taxi_park_id IS NOT NULL;

CREATE INDEX idx_financial_transactions_type_created_at
ON financial_transactions (transaction_type, created_at DESC);

CREATE TABLE finance_audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id UUID REFERENCES financial_transactions(id) ON DELETE RESTRICT,
    order_id UUID REFERENCES orders(id) ON DELETE RESTRICT,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_finance_audit_events_order_created_at
ON finance_audit_events (order_id, created_at DESC);

CREATE INDEX idx_drivers_taxi_park
ON drivers (taxi_park_id)
WHERE taxi_park_id IS NOT NULL AND deleted_at IS NULL;


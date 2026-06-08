CREATE TABLE taxi_park_finance_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE CASCADE,
    driver_commission_percent NUMERIC(5,2) NOT NULL DEFAULT 0.00,
    platform_service_fee_percent NUMERIC(5,2) NOT NULL DEFAULT 1.00,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT taxi_park_finance_settings_driver_commission_range CHECK (
        driver_commission_percent >= 0 AND driver_commission_percent <= 100
    ),
    CONSTRAINT taxi_park_finance_settings_platform_fee_range CHECK (
        platform_service_fee_percent >= 0 AND platform_service_fee_percent <= 100
    )
);

CREATE UNIQUE INDEX uq_taxi_park_finance_settings_taxi_park_id
ON taxi_park_finance_settings (taxi_park_id);

CREATE INDEX idx_taxi_park_finance_settings_active
ON taxi_park_finance_settings (taxi_park_id, is_active);

CREATE TRIGGER taxi_park_finance_settings_set_updated_at
BEFORE UPDATE ON taxi_park_finance_settings
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

INSERT INTO taxi_park_finance_settings (taxi_park_id, driver_commission_percent, platform_service_fee_percent, is_active)
SELECT id, COALESCE(commission_percent, 0.00), 1.00, TRUE
FROM taxi_parks
WHERE deleted_at IS NULL
ON CONFLICT (taxi_park_id) DO NOTHING;

CREATE TABLE order_financial_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT,
    taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE SET NULL,
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
    passenger_id UUID REFERENCES users(id) ON DELETE SET NULL,
    order_total_amount NUMERIC(12,2) NOT NULL,
    driver_commission_percent NUMERIC(5,2) NOT NULL,
    taxi_park_commission_amount NUMERIC(12,2) NOT NULL,
    driver_income_amount NUMERIC(12,2) NOT NULL,
    platform_service_fee_percent NUMERIC(5,2) NOT NULL,
    platform_service_fee_amount NUMERIC(12,2) NOT NULL,
    taxi_park_income_amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_order_financial_transactions_order_id UNIQUE (order_id),
    CONSTRAINT order_financial_transactions_order_total_amount_non_negative CHECK (order_total_amount >= 0),
    CONSTRAINT order_financial_transactions_driver_commission_percent_range CHECK (
        driver_commission_percent >= 0 AND driver_commission_percent <= 100
    ),
    CONSTRAINT order_financial_transactions_platform_fee_percent_range CHECK (
        platform_service_fee_percent >= 0 AND platform_service_fee_percent <= 100
    ),
    CONSTRAINT order_financial_transactions_taxi_park_commission_amount_non_negative CHECK (taxi_park_commission_amount >= 0),
    CONSTRAINT order_financial_transactions_driver_income_amount_non_negative CHECK (driver_income_amount >= 0),
    CONSTRAINT order_financial_transactions_platform_service_fee_amount_non_negative CHECK (platform_service_fee_amount >= 0),
    CONSTRAINT order_financial_transactions_taxi_park_income_amount_non_negative CHECK (taxi_park_income_amount >= 0),
    CONSTRAINT order_financial_transactions_status_check CHECK (
        status IN ('pending', 'calculated', 'confirmed', 'cancelled', 'refunded')
    )
);

CREATE INDEX idx_order_financial_transactions_taxi_park_id
ON order_financial_transactions (taxi_park_id);

CREATE INDEX idx_order_financial_transactions_driver_id
ON order_financial_transactions (driver_id);

CREATE INDEX idx_order_financial_transactions_created_at
ON order_financial_transactions (created_at DESC);

CREATE INDEX idx_order_financial_transactions_status
ON order_financial_transactions (status);

CREATE TRIGGER order_financial_transactions_set_updated_at
BEFORE UPDATE ON order_financial_transactions
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE taxi_park_balance_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    transaction_id UUID REFERENCES financial_transactions(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    direction VARCHAR(10) NOT NULL,
    balance_after NUMERIC(12,2) NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT taxi_park_balance_ledger_amount_non_negative CHECK (amount >= 0),
    CONSTRAINT taxi_park_balance_ledger_balance_after_non_negative CHECK (balance_after >= 0),
    CONSTRAINT taxi_park_balance_ledger_direction_check CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT taxi_park_balance_ledger_type_check CHECK (
        type IN ('order_commission_income', 'driver_payout', 'adjustment', 'refund')
    )
);

CREATE INDEX idx_taxi_park_balance_ledger_taxi_park_id
ON taxi_park_balance_ledger (taxi_park_id);

CREATE INDEX idx_taxi_park_balance_ledger_order_id
ON taxi_park_balance_ledger (order_id);

CREATE INDEX idx_taxi_park_balance_ledger_transaction_id
ON taxi_park_balance_ledger (transaction_id);

CREATE INDEX idx_taxi_park_balance_ledger_created_at
ON taxi_park_balance_ledger (created_at DESC);

CREATE TABLE driver_balance_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE SET NULL,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    transaction_id UUID REFERENCES financial_transactions(id) ON DELETE SET NULL,
    type VARCHAR(50) NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    direction VARCHAR(10) NOT NULL,
    balance_after NUMERIC(12,2) NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT driver_balance_ledger_amount_non_negative CHECK (amount >= 0),
    CONSTRAINT driver_balance_ledger_balance_after_non_negative CHECK (balance_after >= 0),
    CONSTRAINT driver_balance_ledger_direction_check CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT driver_balance_ledger_type_check CHECK (
        type IN ('order_income', 'payout', 'correction', 'refund')
    )
);

CREATE INDEX idx_driver_balance_ledger_driver_id
ON driver_balance_ledger (driver_id);

CREATE INDEX idx_driver_balance_ledger_taxi_park_id
ON driver_balance_ledger (taxi_park_id);

CREATE INDEX idx_driver_balance_ledger_order_id
ON driver_balance_ledger (order_id);

CREATE INDEX idx_driver_balance_ledger_transaction_id
ON driver_balance_ledger (transaction_id);

CREATE INDEX idx_driver_balance_ledger_created_at
ON driver_balance_ledger (created_at DESC);

CREATE TABLE taxi_park_platform_fee_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE CASCADE,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    transaction_id UUID REFERENCES financial_transactions(id) ON DELETE SET NULL,
    invoice_id UUID,
    type VARCHAR(50) NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    direction VARCHAR(10) NOT NULL,
    balance_after NUMERIC(12,2) NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT taxi_park_platform_fee_ledger_amount_non_negative CHECK (amount >= 0),
    CONSTRAINT taxi_park_platform_fee_ledger_balance_after_non_negative CHECK (balance_after >= 0),
    CONSTRAINT taxi_park_platform_fee_ledger_direction_check CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT taxi_park_platform_fee_ledger_type_check CHECK (
        type IN ('platform_service_fee_accrual', 'platform_service_fee_payment', 'adjustment', 'refund')
    )
);

CREATE INDEX idx_taxi_park_platform_fee_ledger_taxi_park_id
ON taxi_park_platform_fee_ledger (taxi_park_id);

CREATE INDEX idx_taxi_park_platform_fee_ledger_order_id
ON taxi_park_platform_fee_ledger (order_id);

CREATE INDEX idx_taxi_park_platform_fee_ledger_transaction_id
ON taxi_park_platform_fee_ledger (transaction_id);

CREATE INDEX idx_taxi_park_platform_fee_ledger_created_at
ON taxi_park_platform_fee_ledger (created_at DESC);

CREATE TABLE platform_balance_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE SET NULL,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    transaction_id UUID REFERENCES financial_transactions(id) ON DELETE SET NULL,
    invoice_id UUID,
    type VARCHAR(50) NOT NULL,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    direction VARCHAR(10) NOT NULL,
    balance_after NUMERIC(12,2) NOT NULL,
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT platform_balance_ledger_amount_non_negative CHECK (amount >= 0),
    CONSTRAINT platform_balance_ledger_balance_after_non_negative CHECK (balance_after >= 0),
    CONSTRAINT platform_balance_ledger_direction_check CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT platform_balance_ledger_type_check CHECK (
        type IN ('service_fee_receivable', 'service_fee_payment_received', 'adjustment', 'refund')
    )
);

CREATE INDEX idx_platform_balance_ledger_taxi_park_id
ON platform_balance_ledger (taxi_park_id);

CREATE INDEX idx_platform_balance_ledger_order_id
ON platform_balance_ledger (order_id);

CREATE INDEX idx_platform_balance_ledger_transaction_id
ON platform_balance_ledger (transaction_id);

CREATE INDEX idx_platform_balance_ledger_created_at
ON platform_balance_ledger (created_at DESC);

CREATE TABLE driver_payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
    taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE RESTRICT,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    period_from TIMESTAMPTZ,
    period_to TIMESTAMPTZ,
    payment_method VARCHAR(100),
    payment_document_number VARCHAR(100),
    comment TEXT,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT driver_payouts_amount_non_negative CHECK (amount >= 0),
    CONSTRAINT driver_payouts_status_check CHECK (status IN ('created', 'approved', 'paid', 'cancelled'))
);

CREATE INDEX idx_driver_payouts_driver_id
ON driver_payouts (driver_id);

CREATE INDEX idx_driver_payouts_taxi_park_id
ON driver_payouts (taxi_park_id);

CREATE INDEX idx_driver_payouts_status
ON driver_payouts (status);

CREATE INDEX idx_driver_payouts_created_at
ON driver_payouts (created_at DESC);

CREATE TRIGGER driver_payouts_set_updated_at
BEFORE UPDATE ON driver_payouts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE platform_invoices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE RESTRICT,
    amount NUMERIC(12,2) NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'RUB',
    period_from TIMESTAMPTZ NOT NULL,
    period_to TIMESTAMPTZ NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    invoice_number VARCHAR(100) NOT NULL,
    document_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT platform_invoices_amount_non_negative CHECK (amount >= 0),
    CONSTRAINT platform_invoices_status_check CHECK (status IN ('draft', 'issued', 'paid', 'overdue', 'cancelled'))
);

CREATE UNIQUE INDEX uq_platform_invoices_period
ON platform_invoices (taxi_park_id, period_from, period_to);

CREATE INDEX idx_platform_invoices_taxi_park_id
ON platform_invoices (taxi_park_id);

CREATE INDEX idx_platform_invoices_status
ON platform_invoices (status);

CREATE INDEX idx_platform_invoices_created_at
ON platform_invoices (created_at DESC);

CREATE TRIGGER platform_invoices_set_updated_at
BEFORE UPDATE ON platform_invoices
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE taxi_park_platform_fee_ledger
    ADD CONSTRAINT taxi_park_platform_fee_ledger_invoice_id_fkey
    FOREIGN KEY (invoice_id) REFERENCES platform_invoices(id) ON DELETE SET NULL;

ALTER TABLE platform_balance_ledger
    ADD CONSTRAINT platform_balance_ledger_invoice_id_fkey
    FOREIGN KEY (invoice_id) REFERENCES platform_invoices(id) ON DELETE SET NULL;

CREATE TABLE finance_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE SET NULL,
    driver_id UUID REFERENCES drivers(id) ON DELETE SET NULL,
    order_id UUID REFERENCES orders(id) ON DELETE SET NULL,
    payout_id UUID REFERENCES driver_payouts(id) ON DELETE SET NULL,
    invoice_id UUID REFERENCES platform_invoices(id) ON DELETE SET NULL,
    type VARCHAR(100) NOT NULL,
    number VARCHAR(100) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'created',
    file_url TEXT,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_finance_documents_taxi_park_id
ON finance_documents (taxi_park_id);

CREATE INDEX idx_finance_documents_driver_id
ON finance_documents (driver_id);

CREATE INDEX idx_finance_documents_order_id
ON finance_documents (order_id);

CREATE INDEX idx_finance_documents_payout_id
ON finance_documents (payout_id);

CREATE INDEX idx_finance_documents_invoice_id
ON finance_documents (invoice_id);

CREATE INDEX idx_finance_documents_created_at
ON finance_documents (created_at DESC);

CREATE INDEX idx_finance_documents_status
ON finance_documents (status);

CREATE TRIGGER finance_documents_set_updated_at
BEFORE UPDATE ON finance_documents
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

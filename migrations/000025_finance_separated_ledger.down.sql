DROP TRIGGER IF EXISTS finance_documents_set_updated_at ON finance_documents;
DROP TABLE IF EXISTS finance_documents;

ALTER TABLE platform_balance_ledger DROP CONSTRAINT IF EXISTS platform_balance_ledger_invoice_id_fkey;
ALTER TABLE taxi_park_platform_fee_ledger DROP CONSTRAINT IF EXISTS taxi_park_platform_fee_ledger_invoice_id_fkey;

DROP TRIGGER IF EXISTS platform_invoices_set_updated_at ON platform_invoices;
DROP TABLE IF EXISTS platform_invoices;

DROP TRIGGER IF EXISTS driver_payouts_set_updated_at ON driver_payouts;
DROP TABLE IF EXISTS driver_payouts;

DROP TABLE IF EXISTS platform_balance_ledger;
DROP TABLE IF EXISTS taxi_park_platform_fee_ledger;
DROP TABLE IF EXISTS driver_balance_ledger;
DROP TABLE IF EXISTS taxi_park_balance_ledger;

DROP TRIGGER IF EXISTS order_financial_transactions_set_updated_at ON order_financial_transactions;
DROP TABLE IF EXISTS order_financial_transactions;

DROP TRIGGER IF EXISTS taxi_park_finance_settings_set_updated_at ON taxi_park_finance_settings;
DROP TABLE IF EXISTS taxi_park_finance_settings;

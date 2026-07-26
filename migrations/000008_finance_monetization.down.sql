DROP INDEX IF EXISTS idx_drivers_taxi_park;
DROP INDEX IF EXISTS idx_finance_audit_events_order_created_at;
DROP TABLE IF EXISTS finance_audit_events;
DROP INDEX IF EXISTS idx_financial_transactions_type_created_at;
DROP INDEX IF EXISTS idx_financial_transactions_taxi_park_created_at;
DROP INDEX IF EXISTS idx_financial_transactions_driver_created_at;
DROP INDEX IF EXISTS idx_financial_transactions_order_type;
DROP TABLE IF EXISTS financial_transactions;
DROP TRIGGER IF EXISTS driver_balances_set_updated_at ON driver_balances;
DROP TABLE IF EXISTS driver_balances;

ALTER TABLE tariffs DROP CONSTRAINT IF EXISTS tariffs_commission_percent_range;
ALTER TABLE cities DROP CONSTRAINT IF EXISTS cities_commission_percent_range;
ALTER TABLE drivers DROP CONSTRAINT IF EXISTS drivers_commission_percent_range;
ALTER TABLE taxi_parks DROP CONSTRAINT IF EXISTS taxi_parks_commission_percent_range;

ALTER TABLE tariffs DROP COLUMN IF EXISTS commission_percent;
ALTER TABLE cities DROP COLUMN IF EXISTS commission_percent;
ALTER TABLE drivers DROP COLUMN IF EXISTS commission_percent;
ALTER TABLE drivers DROP COLUMN IF EXISTS taxi_park_id;
ALTER TABLE taxi_parks DROP COLUMN IF EXISTS balance_cents;
ALTER TABLE taxi_parks DROP COLUMN IF EXISTS commission_percent;


DROP INDEX IF EXISTS idx_passenger_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_passenger_refresh_tokens_passenger_id;
DROP INDEX IF EXISTS idx_passenger_auth_codes_used_at;
DROP INDEX IF EXISTS idx_passenger_auth_codes_expires_at;
DROP INDEX IF EXISTS idx_passenger_auth_codes_phone;
DROP INDEX IF EXISTS idx_passengers_phone;

DROP TABLE IF EXISTS passenger_refresh_tokens;

DROP TRIGGER IF EXISTS passenger_auth_codes_set_updated_at ON passenger_auth_codes;
DROP TABLE IF EXISTS passenger_auth_codes;

DROP TRIGGER IF EXISTS passengers_set_updated_at ON passengers;
DROP TABLE IF EXISTS passengers;

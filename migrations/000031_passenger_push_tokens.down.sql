DROP INDEX IF EXISTS idx_passenger_push_tokens_passenger_active;
DROP INDEX IF EXISTS idx_passenger_push_tokens_unique_active;
DROP TRIGGER IF EXISTS passenger_push_tokens_set_updated_at ON passenger_push_tokens;
DROP TABLE IF EXISTS passenger_push_tokens;

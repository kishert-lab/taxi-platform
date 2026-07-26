DROP INDEX IF EXISTS idx_user_verification_codes_expires_at;
DROP INDEX IF EXISTS idx_user_verification_codes_lookup;
DROP INDEX IF EXISTS idx_taxi_parks_owner_user_id;
DROP INDEX IF EXISTS idx_taxi_parks_city_status;

DROP TABLE IF EXISTS user_verification_codes;
DROP TABLE IF EXISTS taxi_parks;

DROP INDEX IF EXISTS idx_users_registration_type;
DROP INDEX IF EXISTS idx_users_email_unique;

ALTER TABLE users
    DROP COLUMN IF EXISTS phone_confirmed_at,
    DROP COLUMN IF EXISTS email_confirmed_at,
    DROP COLUMN IF EXISTS is_email_confirmed,
    DROP COLUMN IF EXISTS registration_type,
    DROP COLUMN IF EXISTS email;

DROP TYPE IF EXISTS verification_status;
DROP TYPE IF EXISTS verification_purpose;
DROP TYPE IF EXISTS verification_channel;
DROP TYPE IF EXISTS registration_type;

-- PostgreSQL does not support removing a single enum value from user_role safely.
-- The 'taxi_park' value is intentionally left in place on down migration.

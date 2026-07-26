DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_enum
        WHERE enumlabel = 'taxi_park'
          AND enumtypid = 'user_role'::regtype
    ) THEN
        ALTER TYPE user_role ADD VALUE 'taxi_park';
    END IF;
END $$;

CREATE TYPE registration_type AS ENUM ('passenger', 'driver', 'taxi_park');
CREATE TYPE verification_channel AS ENUM ('sms', 'email');
CREATE TYPE verification_purpose AS ENUM ('registration', 'login', 'email_confirm', 'phone_change', 'password_reset');
CREATE TYPE verification_status AS ENUM ('pending', 'verified', 'expired', 'blocked');

ALTER TABLE users
    ADD COLUMN email TEXT,
    ADD COLUMN registration_type registration_type,
    ADD COLUMN is_email_confirmed BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN email_confirmed_at TIMESTAMPTZ,
    ADD COLUMN phone_confirmed_at TIMESTAMPTZ;

UPDATE users
SET registration_type = CASE
    WHEN role = 'driver' THEN 'driver'::registration_type
    ELSE 'passenger'::registration_type
END
WHERE registration_type IS NULL;

ALTER TABLE users
    ALTER COLUMN registration_type SET NOT NULL;

CREATE UNIQUE INDEX idx_users_email_unique
ON users (lower(email))
WHERE email IS NOT NULL AND deleted_at IS NULL;

CREATE INDEX idx_users_registration_type
ON users (registration_type)
WHERE deleted_at IS NULL;

CREATE TABLE taxi_parks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_user_id UUID NOT NULL UNIQUE REFERENCES users(id) ON DELETE RESTRICT,
    city_id UUID NOT NULL REFERENCES cities(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    legal_name TEXT,
    tax_id TEXT,
    contact_phone TEXT NOT NULL,
    contact_email TEXT NOT NULL,
    is_verified BOOLEAN NOT NULL DEFAULT false,
    verification_status verification_status NOT NULL DEFAULT 'pending',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT taxi_parks_name_not_blank CHECK (length(trim(name)) > 0),
    CONSTRAINT taxi_parks_contact_phone_not_blank CHECK (length(trim(contact_phone)) > 0),
    CONSTRAINT taxi_parks_contact_email_not_blank CHECK (length(trim(contact_email)) > 0)
);

CREATE TRIGGER taxi_parks_set_updated_at
BEFORE UPDATE ON taxi_parks
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE user_verification_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    target TEXT NOT NULL,
    channel verification_channel NOT NULL,
    purpose verification_purpose NOT NULL,
    code_hash TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_sent_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT user_verification_codes_target_not_blank CHECK (length(trim(target)) > 0),
    CONSTRAINT user_verification_codes_attempts_valid CHECK (attempts >= 0 AND max_attempts > 0)
);

CREATE INDEX idx_taxi_parks_city_status
ON taxi_parks (city_id, verification_status)
WHERE deleted_at IS NULL;

CREATE INDEX idx_taxi_parks_owner_user_id
ON taxi_parks (owner_user_id)
WHERE deleted_at IS NULL;

CREATE INDEX idx_user_verification_codes_lookup
ON user_verification_codes (target, channel, purpose, created_at DESC)
WHERE consumed_at IS NULL;

CREATE INDEX idx_user_verification_codes_expires_at
ON user_verification_codes (expires_at)
WHERE consumed_at IS NULL;

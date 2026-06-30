CREATE TABLE passengers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(32) NOT NULL UNIQUE,
    name VARCHAR(255),
    email VARCHAR(255),
    avatar_url TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    phone_verified_at TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE TRIGGER passengers_set_updated_at
BEFORE UPDATE ON passengers
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE passenger_auth_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    phone VARCHAR(32) NOT NULL,
    code_hash TEXT NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 5,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT passenger_auth_codes_attempts_valid CHECK (attempts >= 0 AND max_attempts > 0)
);

CREATE TRIGGER passenger_auth_codes_set_updated_at
BEFORE UPDATE ON passenger_auth_codes
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE passenger_refresh_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    passenger_id UUID NOT NULL REFERENCES passengers(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_passengers_phone
ON passengers (phone)
WHERE deleted_at IS NULL;

CREATE INDEX idx_passenger_auth_codes_phone
ON passenger_auth_codes (phone);

CREATE INDEX idx_passenger_auth_codes_expires_at
ON passenger_auth_codes (expires_at)
WHERE used_at IS NULL;

CREATE INDEX idx_passenger_auth_codes_used_at
ON passenger_auth_codes (used_at);

CREATE INDEX idx_passenger_refresh_tokens_passenger_id
ON passenger_refresh_tokens (passenger_id);

CREATE INDEX idx_passenger_refresh_tokens_expires_at
ON passenger_refresh_tokens (expires_at);

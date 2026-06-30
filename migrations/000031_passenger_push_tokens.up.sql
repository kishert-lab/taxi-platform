CREATE TABLE passenger_push_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    passenger_id UUID NOT NULL REFERENCES passengers(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    platform TEXT NOT NULL,
    device_id TEXT,
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT passenger_push_tokens_token_not_blank CHECK (length(trim(token)) > 0),
    CONSTRAINT passenger_push_tokens_platform_not_blank CHECK (length(trim(platform)) > 0)
);

CREATE TRIGGER passenger_push_tokens_set_updated_at
BEFORE UPDATE ON passenger_push_tokens
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE UNIQUE INDEX idx_passenger_push_tokens_unique_active
ON passenger_push_tokens (passenger_id, token)
WHERE deleted_at IS NULL;

CREATE INDEX idx_passenger_push_tokens_passenger_active
ON passenger_push_tokens (passenger_id, is_active)
WHERE deleted_at IS NULL;

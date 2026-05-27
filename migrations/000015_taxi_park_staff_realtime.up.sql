CREATE TABLE IF NOT EXISTS taxi_park_staff (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role user_role NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT taxi_park_staff_supported_role CHECK (role IN ('dispatcher', 'taxi_park')),
    CONSTRAINT taxi_park_staff_unique_user UNIQUE (taxi_park_id, user_id)
);

CREATE TRIGGER taxi_park_staff_set_updated_at
BEFORE UPDATE ON taxi_park_staff
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE INDEX IF NOT EXISTS idx_taxi_park_staff_park_role_active
ON taxi_park_staff (taxi_park_id, role, is_active)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_taxi_park_staff_user_active
ON taxi_park_staff (user_id, is_active)
WHERE deleted_at IS NULL;

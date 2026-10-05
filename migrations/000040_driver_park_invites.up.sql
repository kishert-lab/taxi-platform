CREATE TABLE driver_park_invites (
 id UUID PRIMARY KEY,
 driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
 from_taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE RESTRICT,
 to_taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE RESTRICT,
 status TEXT NOT NULL CHECK (status IN ('pending','accepted','declined','cancelled','expired')),
 created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
 created_at TIMESTAMPTZ NOT NULL,
 expires_at TIMESTAMPTZ NOT NULL CHECK (expires_at > created_at),
 accepted_at TIMESTAMPTZ,
 declined_at TIMESTAMPTZ,
 cancelled_at TIMESTAMPTZ,
 cancellation_reason TEXT NOT NULL DEFAULT '',
 CHECK (from_taxi_park_id IS DISTINCT FROM to_taxi_park_id)
);
CREATE UNIQUE INDEX driver_park_invites_pending ON driver_park_invites(driver_id,to_taxi_park_id) WHERE status='pending';
CREATE INDEX driver_park_invites_driver ON driver_park_invites(driver_id,created_at DESC);
CREATE INDEX driver_park_invites_park ON driver_park_invites(to_taxi_park_id,created_at DESC);

CREATE TABLE driver_park_history (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE RESTRICT,
 taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE RESTRICT,
 joined_at TIMESTAMPTZ NOT NULL,
 left_at TIMESTAMPTZ,
 transfer_invite_id UUID REFERENCES driver_park_invites(id) ON DELETE RESTRICT,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 CHECK (left_at IS NULL OR left_at >= joined_at)
);
CREATE UNIQUE INDEX driver_park_history_current ON driver_park_history(driver_id) WHERE left_at IS NULL;
INSERT INTO driver_park_history(driver_id,taxi_park_id,joined_at)
 SELECT id,taxi_park_id,created_at FROM drivers WHERE taxi_park_id IS NOT NULL AND deleted_at IS NULL;

-- Extend the existing balance model; a NULL park is the independent driver's account.
ALTER TABLE driver_balances DROP CONSTRAINT driver_balances_pkey;
ALTER TABLE driver_balances ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 ADD COLUMN taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE RESTRICT;
UPDATE driver_balances b SET taxi_park_id=d.taxi_park_id FROM drivers d WHERE d.id=b.driver_id;
ALTER TABLE driver_balances ADD CONSTRAINT driver_balances_driver_park_unique UNIQUE NULLS NOT DISTINCT(driver_id,taxi_park_id);

CREATE TABLE driver_push_tokens (
 token TEXT PRIMARY KEY,
 user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 platform TEXT NOT NULL CHECK(platform IN ('android','ios','web')),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX driver_push_tokens_user ON driver_push_tokens(user_id);

INSERT INTO permissions(code,description) VALUES('taxi_park.drivers.invite','Invite drivers to the current taxi park');
INSERT INTO role_permissions(role,permission_code) VALUES
 ('taxi_park','taxi_park.drivers.invite'),('dispatcher','taxi_park.drivers.invite');

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS must_change_password BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE drivers
    ADD COLUMN IF NOT EXISTS birth_date DATE,
    ADD COLUMN IF NOT EXISTS license_series TEXT,
    ADD COLUMN IF NOT EXISTS license_issued_at DATE,
    ADD COLUMN IF NOT EXISTS license_expires_at DATE,
    ADD COLUMN IF NOT EXISTS driving_experience_from DATE,
    ADD COLUMN IF NOT EXISTS verification_status TEXT NOT NULL DEFAULT 'draft',
    ADD COLUMN IF NOT EXISTS taxi_park_comment TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'drivers_verification_status_check'
    ) THEN
        ALTER TABLE drivers
            ADD CONSTRAINT drivers_verification_status_check
            CHECK (verification_status IN ('draft', 'pending_verification', 'verified', 'rejected', 'blocked', 'archived'));
    END IF;
END $$;

ALTER TABLE cars
    ALTER COLUMN driver_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS taxi_park_id UUID REFERENCES taxi_parks(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS vin TEXT,
    ADD COLUMN IF NOT EXISTS sts TEXT,
    ADD COLUMN IF NOT EXISTS pts TEXT,
    ADD COLUMN IF NOT EXISTS car_class TEXT,
    ADD COLUMN IF NOT EXISTS verification_status TEXT NOT NULL DEFAULT 'draft',
    ADD COLUMN IF NOT EXISTS owner_details TEXT,
    ADD COLUMN IF NOT EXISTS osago_expires_at DATE,
    ADD COLUMN IF NOT EXISTS diagnostic_card_expires_at DATE,
    ADD COLUMN IF NOT EXISTS taxi_permit_number TEXT,
    ADD COLUMN IF NOT EXISTS regional_registry_number TEXT,
    ADD COLUMN IF NOT EXISTS permit_region TEXT,
    ADD COLUMN IF NOT EXISTS permit_issued_at DATE,
    ADD COLUMN IF NOT EXISTS permit_expires_at DATE;

UPDATE cars c
SET taxi_park_id = d.taxi_park_id
FROM drivers d
WHERE c.driver_id = d.id
  AND c.taxi_park_id IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'cars_verification_status_check'
    ) THEN
        ALTER TABLE cars
            ADD CONSTRAINT cars_verification_status_check
            CHECK (verification_status IN ('draft', 'pending_verification', 'verified', 'rejected', 'blocked', 'archived'));
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS car_driver_assignments (
    car_id UUID NOT NULL REFERENCES cars(id) ON DELETE CASCADE,
    driver_id UUID NOT NULL REFERENCES drivers(id) ON DELETE CASCADE,
    taxi_park_id UUID NOT NULL REFERENCES taxi_parks(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (car_id, driver_id)
);

CREATE INDEX IF NOT EXISTS idx_drivers_taxi_park_verification
ON drivers (taxi_park_id, verification_status)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cars_taxi_park_verification
ON cars (taxi_park_id, verification_status)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_car_driver_assignments_driver
ON car_driver_assignments (driver_id);

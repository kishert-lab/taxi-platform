ALTER TABLE drivers
    ADD COLUMN IF NOT EXISTS license_category TEXT,
    ADD COLUMN IF NOT EXISTS has_no_taxi_work_restrictions BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS federal_law_580_compliant BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS regional_requirements_compliant BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS medical_check_passed BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS pretrip_control_required BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS pretrip_control_passed BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS no_transport_ban BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS verification_checked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS verification_checked_by UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE cars
    ADD COLUMN IF NOT EXISTS taxi_permit_verified BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS regional_registry_verified BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS regional_requirements_compliant BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS has_taxi_color_scheme BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS has_orange_roof_lamp BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS has_passenger_info BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS osago_verified BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS diagnostic_card_verified BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS technical_state_verified BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS localization_compliant BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS legal_use_basis_verified BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS verification_checked_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS verification_checked_by UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_drivers_compliance_ready
ON drivers (taxi_park_id, verification_status, license_expires_at)
WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cars_compliance_ready
ON cars (taxi_park_id, verification_status, permit_expires_at, osago_expires_at)
WHERE deleted_at IS NULL;

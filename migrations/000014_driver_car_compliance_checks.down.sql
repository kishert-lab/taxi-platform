DROP INDEX IF EXISTS idx_cars_compliance_ready;
DROP INDEX IF EXISTS idx_drivers_compliance_ready;

ALTER TABLE cars DROP COLUMN IF EXISTS verification_checked_by;
ALTER TABLE cars DROP COLUMN IF EXISTS verification_checked_at;
ALTER TABLE cars DROP COLUMN IF EXISTS legal_use_basis_verified;
ALTER TABLE cars DROP COLUMN IF EXISTS localization_compliant;
ALTER TABLE cars DROP COLUMN IF EXISTS technical_state_verified;
ALTER TABLE cars DROP COLUMN IF EXISTS diagnostic_card_verified;
ALTER TABLE cars DROP COLUMN IF EXISTS osago_verified;
ALTER TABLE cars DROP COLUMN IF EXISTS has_passenger_info;
ALTER TABLE cars DROP COLUMN IF EXISTS has_orange_roof_lamp;
ALTER TABLE cars DROP COLUMN IF EXISTS has_taxi_color_scheme;
ALTER TABLE cars DROP COLUMN IF EXISTS regional_requirements_compliant;
ALTER TABLE cars DROP COLUMN IF EXISTS regional_registry_verified;
ALTER TABLE cars DROP COLUMN IF EXISTS taxi_permit_verified;

ALTER TABLE drivers DROP COLUMN IF EXISTS verification_checked_by;
ALTER TABLE drivers DROP COLUMN IF EXISTS verification_checked_at;
ALTER TABLE drivers DROP COLUMN IF EXISTS no_transport_ban;
ALTER TABLE drivers DROP COLUMN IF EXISTS pretrip_control_passed;
ALTER TABLE drivers DROP COLUMN IF EXISTS pretrip_control_required;
ALTER TABLE drivers DROP COLUMN IF EXISTS medical_check_passed;
ALTER TABLE drivers DROP COLUMN IF EXISTS regional_requirements_compliant;
ALTER TABLE drivers DROP COLUMN IF EXISTS federal_law_580_compliant;
ALTER TABLE drivers DROP COLUMN IF EXISTS has_no_taxi_work_restrictions;
ALTER TABLE drivers DROP COLUMN IF EXISTS license_category;

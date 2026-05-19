DROP INDEX IF EXISTS idx_car_driver_assignments_driver;
DROP INDEX IF EXISTS idx_cars_taxi_park_verification;
DROP INDEX IF EXISTS idx_drivers_taxi_park_verification;
DROP TABLE IF EXISTS car_driver_assignments;

ALTER TABLE cars DROP CONSTRAINT IF EXISTS cars_verification_status_check;
ALTER TABLE cars DROP COLUMN IF EXISTS permit_expires_at;
ALTER TABLE cars DROP COLUMN IF EXISTS permit_issued_at;
ALTER TABLE cars DROP COLUMN IF EXISTS permit_region;
ALTER TABLE cars DROP COLUMN IF EXISTS regional_registry_number;
ALTER TABLE cars DROP COLUMN IF EXISTS taxi_permit_number;
ALTER TABLE cars DROP COLUMN IF EXISTS diagnostic_card_expires_at;
ALTER TABLE cars DROP COLUMN IF EXISTS osago_expires_at;
ALTER TABLE cars DROP COLUMN IF EXISTS owner_details;
ALTER TABLE cars DROP COLUMN IF EXISTS verification_status;
ALTER TABLE cars DROP COLUMN IF EXISTS car_class;
ALTER TABLE cars DROP COLUMN IF EXISTS pts;
ALTER TABLE cars DROP COLUMN IF EXISTS sts;
ALTER TABLE cars DROP COLUMN IF EXISTS vin;
ALTER TABLE cars DROP COLUMN IF EXISTS taxi_park_id;
ALTER TABLE cars ALTER COLUMN driver_id SET NOT NULL;

ALTER TABLE drivers DROP CONSTRAINT IF EXISTS drivers_verification_status_check;
ALTER TABLE drivers DROP COLUMN IF EXISTS taxi_park_comment;
ALTER TABLE drivers DROP COLUMN IF EXISTS verification_status;
ALTER TABLE drivers DROP COLUMN IF EXISTS driving_experience_from;
ALTER TABLE drivers DROP COLUMN IF EXISTS license_expires_at;
ALTER TABLE drivers DROP COLUMN IF EXISTS license_issued_at;
ALTER TABLE drivers DROP COLUMN IF EXISTS license_series;
ALTER TABLE drivers DROP COLUMN IF EXISTS birth_date;

ALTER TABLE users DROP COLUMN IF EXISTS must_change_password;

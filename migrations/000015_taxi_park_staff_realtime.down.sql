DROP INDEX IF EXISTS idx_taxi_park_staff_user_active;
DROP INDEX IF EXISTS idx_taxi_park_staff_park_role_active;
DROP TRIGGER IF EXISTS taxi_park_staff_set_updated_at ON taxi_park_staff;
DROP TABLE IF EXISTS taxi_park_staff;

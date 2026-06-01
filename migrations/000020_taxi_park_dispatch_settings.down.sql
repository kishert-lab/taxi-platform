ALTER TABLE taxi_park_settings
    DROP CONSTRAINT IF EXISTS taxi_park_settings_dispatch_positive,
    DROP COLUMN IF EXISTS dispatch_recovery_interval_sec,
    DROP COLUMN IF EXISTS dispatch_worker_poll_timeout_sec,
    DROP COLUMN IF EXISTS dispatch_accept_lock_ttl_sec,
    DROP COLUMN IF EXISTS dispatch_offer_ttl_sec,
    DROP COLUMN IF EXISTS dispatch_driver_location_max_age_sec,
    DROP COLUMN IF EXISTS dispatch_max_drivers_per_offer,
    DROP COLUMN IF EXISTS dispatch_radius_attempts_meters,
    DROP COLUMN IF EXISTS dispatch_radius_step_meters,
    DROP COLUMN IF EXISTS dispatch_max_radius_meters,
    DROP COLUMN IF EXISTS dispatch_initial_radius_meters;

ALTER TABLE taxi_park_settings
    ADD COLUMN dispatch_initial_radius_meters INTEGER NOT NULL DEFAULT 10000,
    ADD COLUMN dispatch_max_radius_meters INTEGER NOT NULL DEFAULT 100000,
    ADD COLUMN dispatch_radius_step_meters INTEGER NOT NULL DEFAULT 1000,
    ADD COLUMN dispatch_radius_attempts_meters JSONB NOT NULL DEFAULT '[10000,30000,50000,100000]'::jsonb,
    ADD COLUMN dispatch_max_drivers_per_offer INTEGER NOT NULL DEFAULT 5,
    ADD COLUMN dispatch_driver_location_max_age_sec INTEGER NOT NULL DEFAULT 120,
    ADD COLUMN dispatch_offer_ttl_sec INTEGER NOT NULL DEFAULT 60,
    ADD COLUMN dispatch_accept_lock_ttl_sec INTEGER NOT NULL DEFAULT 90,
    ADD COLUMN dispatch_worker_poll_timeout_sec INTEGER NOT NULL DEFAULT 30,
    ADD COLUMN dispatch_recovery_interval_sec INTEGER NOT NULL DEFAULT 30,
    ADD CONSTRAINT taxi_park_settings_dispatch_positive CHECK (
        dispatch_initial_radius_meters > 0
        AND dispatch_max_radius_meters >= dispatch_initial_radius_meters
        AND dispatch_radius_step_meters > 0
        AND dispatch_max_drivers_per_offer > 0
        AND dispatch_driver_location_max_age_sec > 0
        AND dispatch_offer_ttl_sec > 0
        AND dispatch_accept_lock_ttl_sec > 0
        AND dispatch_worker_poll_timeout_sec > 0
        AND dispatch_recovery_interval_sec > 0
        AND jsonb_typeof(dispatch_radius_attempts_meters) = 'array'
        AND jsonb_array_length(dispatch_radius_attempts_meters) > 0
    );

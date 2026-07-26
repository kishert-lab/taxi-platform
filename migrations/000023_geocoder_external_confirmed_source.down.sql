UPDATE local_geo_points
SET source = 'user_confirmed'
WHERE source = 'external_confirmed';

ALTER TABLE local_geo_points
    DROP CONSTRAINT IF EXISTS local_geo_points_source_check;

ALTER TABLE local_geo_points
    ADD CONSTRAINT local_geo_points_source_check
    CHECK (source IN ('user_confirmed', 'driver_confirmed', 'dispatcher_confirmed', 'admin'));

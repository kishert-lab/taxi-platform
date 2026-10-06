DO $$
DECLARE
    economy_car_class_id UUID;
BEGIN
    SELECT id
    INTO economy_car_class_id
    FROM car_classes
    WHERE code = 'economy'
      AND deleted_at IS NULL
    LIMIT 1;

    IF economy_car_class_id IS NULL THEN
        RAISE EXCEPTION 'cannot backfill taxi park tariffs: active economy car class is missing';
    END IF;

    UPDATE taxi_park_tariffs
    SET car_class_id = economy_car_class_id
    WHERE car_class_id IS NULL;
END
$$;

ALTER TABLE taxi_park_tariffs
    ALTER COLUMN car_class_id SET NOT NULL;

UPDATE orders AS order_record
SET car_class_id = tariff.car_class_id
FROM taxi_park_tariffs AS tariff
WHERE order_record.car_class_id IS NULL
  AND order_record.metadata ->> 'taxi_park_tariff_id' ~* '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'
  AND tariff.id = (order_record.metadata ->> 'taxi_park_tariff_id')::uuid
  AND tariff.car_class_id IS NOT NULL;

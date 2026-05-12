INSERT INTO cities (id, name, region, country_code, timezone, center)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'Demo City',
    'Demo Region',
    'RU',
    'Asia/Yekaterinburg',
    ST_SetSRID(ST_MakePoint(60.597465, 56.838011), 4326)::geography
)
ON CONFLICT (name, region, country_code) DO NOTHING;

INSERT INTO tariffs (
    id,
    city_id,
    name,
    base_price,
    price_per_km,
    price_per_minute,
    minimum_price,
    free_waiting_minutes,
    rules
)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    'Economy',
    90.00,
    18.00,
    5.00,
    120.00,
    3,
    '{"night_multiplier": 1.15, "bad_weather_multiplier": 1.2}'::jsonb
)
ON CONFLICT (city_id, name) DO NOTHING;

INSERT INTO users (id, phone, role, first_name, last_name, is_phone_confirmed, is_active)
VALUES
    ('33333333-3333-3333-3333-333333333333', '+70000000001', 'admin', 'Admin', 'User', true, true),
    ('44444444-4444-4444-4444-444444444444', '+70000000002', 'dispatcher', 'Demo', 'Dispatcher', true, true),
    ('55555555-5555-5555-5555-555555555555', '+70000000003', 'passenger', 'Demo', 'Passenger', true, true),
    ('66666666-6666-6666-6666-666666666666', '+70000000004', 'driver', 'Demo', 'Driver', true, true)
ON CONFLICT (phone, role) DO NOTHING;

INSERT INTO drivers (id, user_id, city_id, status, rating, completed_orders_count, license_number, is_verified)
VALUES (
    '77777777-7777-7777-7777-777777777777',
    '66666666-6666-6666-6666-666666666666',
    '11111111-1111-1111-1111-111111111111',
    'offline',
    4.95,
    120,
    'DEMO-DRIVER-LICENSE',
    true
)
ON CONFLICT (user_id) DO NOTHING;

INSERT INTO cars (id, driver_id, brand, model, color, plate_number, year, seats, is_active)
VALUES (
    '88888888-8888-8888-8888-888888888888',
    '77777777-7777-7777-7777-777777777777',
    'Lada',
    'Vesta',
    'White',
    'A001AA196',
    2022,
    4,
    true
)
ON CONFLICT (plate_number) DO NOTHING;

INSERT INTO driver_locations (driver_id, city_id, location, heading, speed_mps, accuracy_meters, recorded_at)
VALUES (
    '77777777-7777-7777-7777-777777777777',
    '11111111-1111-1111-1111-111111111111',
    ST_SetSRID(ST_MakePoint(60.600000, 56.840000), 4326)::geography,
    90,
    0.00,
    8.00,
    now()
)
ON CONFLICT (driver_id) DO UPDATE SET
    location = EXCLUDED.location,
    heading = EXCLUDED.heading,
    speed_mps = EXCLUDED.speed_mps,
    accuracy_meters = EXCLUDED.accuracy_meters,
    recorded_at = EXCLUDED.recorded_at;

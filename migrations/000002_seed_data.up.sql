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


INSERT INTO cities (name, region, country_code, timezone, center)
VALUES
    (
        'Пермь',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.229443, 58.010455), 4326)::geography
    ),
    (
        'Березники',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.804015, 59.407922), 4326)::geography
    ),
    (
        'Соликамск',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.768503, 59.648333), 4326)::geography
    ),
    (
        'Чайковский',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.150795, 56.778061), 4326)::geography
    ),
    (
        'Кунгур',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.959313, 57.428321), 4326)::geography
    ),
    (
        'Лысьва',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.808706, 58.099598), 4326)::geography
    ),
    (
        'Краснокамск',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.754515, 58.080185), 4326)::geography
    ),
    (
        'Чусовой',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.819318, 58.297472), 4326)::geography
    ),
    (
        'Добрянка',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.413137, 58.469635), 4326)::geography
    ),
    (
        'Чернушка',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.076361, 56.507224), 4326)::geography
    ),
    (
        'Кудымкар',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.655695, 59.016860), 4326)::geography
    ),
    (
        'Верещагино',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.658074, 58.078636), 4326)::geography
    ),
    (
        'Оса',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.468760, 57.288999), 4326)::geography
    ),
    (
        'Нытва',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.328655, 57.943683), 4326)::geography
    ),
    (
        'Очер',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.716175, 57.885199), 4326)::geography
    ),
    (
        'Губаха',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.554444, 58.837024), 4326)::geography
    ),
    (
        'Горнозаводск',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(58.374340, 58.374253), 4326)::geography
    ),
    (
        'Александровск',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.588642, 59.161378), 4326)::geography
    ),
    (
        'Кизел',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.647673, 59.051140), 4326)::geography
    ),
    (
        'Чердынь',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.479019, 60.402836), 4326)::geography
    )
ON CONFLICT (name, region, country_code) DO NOTHING;


INSERT INTO cities (name, region, country_code, timezone, center)
VALUES
    -- Крупные и средние города
    (
        'Пермь',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.229443, 58.010455), 4326)::geography
    ),
    (
        'Березники',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.804015, 59.407922), 4326)::geography
    ),
    (
        'Соликамск',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.768503, 59.648333), 4326)::geography
    ),
    (
        'Чайковский',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.150795, 56.778061), 4326)::geography
    ),
    (
        'Кунгур',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.959313, 57.428321), 4326)::geography
    ),
    (
        'Лысьва',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.808706, 58.099598), 4326)::geography
    ),
    (
        'Краснокамск',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.754515, 58.080185), 4326)::geography
    ),
    (
        'Чусовой',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.819318, 58.297472), 4326)::geography
    ),
    (
        'Добрянка',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.413137, 58.469635), 4326)::geography
    ),
    (
        'Чернушка',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.076361, 56.507224), 4326)::geography
    ),
    (
        'Кудымкар',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.655695, 59.016860), 4326)::geography
    ),
    (
        'Верещагино',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.658074, 58.078636), 4326)::geography
    ),
    (
        'Оса',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.468760, 57.288999), 4326)::geography
    ),
    (
        'Нытва',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.328655, 57.943683), 4326)::geography
    ),
    (
        'Очер',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.716175, 57.885199), 4326)::geography
    ),
    (
        'Губаха',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.554444, 58.837024), 4326)::geography
    ),
    (
        'Горнозаводск',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(58.374340, 58.374253), 4326)::geography
    ),
    (
        'Александровск',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.588642, 59.161378), 4326)::geography
    ),
    (
        'Кизел',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.647673, 59.051140), 4326)::geography
    ),
    (
        'Чердынь',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.479019, 60.402836), 4326)::geography
    ),

    -- Малые районные / муниципальные центры
    (
        'Барда',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.596139, 56.927777), 4326)::geography
    ),
    (
        'Берёзовка',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.306107, 57.604578), 4326)::geography
    ),
    (
        'Большая Соснова',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.601509, 57.668250), 4326)::geography
    ),
    (
        'Гайны',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.324750, 60.307178), 4326)::geography
    ),
    (
        'Елово',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.923924, 57.053614), 4326)::geography
    ),
    (
        'Ильинский',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.694772, 58.567739), 4326)::geography
    ),
    (
        'Карагай',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.936452, 58.268765), 4326)::geography
    ),
    (
        'Коса',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.997484, 59.944227), 4326)::geography
    ),
    (
        'Кочёво',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.316585, 59.598514), 4326)::geography
    ),
    (
        'Куеда',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.588310, 56.432709), 4326)::geography
    ),
    (
        'Орда',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.909741, 57.195186), 4326)::geography
    ),
    (
        'Октябрьский',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.203445, 56.516147), 4326)::geography
    ),
    (
        'Сива',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.376484, 58.382313), 4326)::geography
    ),
    (
        'Суксун',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.394759, 57.143099), 4326)::geography
    ),
    (
        'Уинское',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.574213, 56.879152), 4326)::geography
    ),
    (
        'Усолье',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.685014, 59.426755), 4326)::geography
    ),
    (
        'Усть-Кишерть',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.249478, 57.365459), 4326)::geography
    ),
    (
        'Частые',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.970897, 57.288983), 4326)::geography
    ),
    (
        'Юрла',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.322936, 59.324655), 4326)::geography
    ),
    (
        'Юсьва',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(54.955660, 58.958333), 4326)::geography
    ),

    -- Рабочие посёлки / важные локальные центры
    (
        'Полазна',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.414249, 58.292247), 4326)::geography
    ),
    (
        'Звёздный',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.314074, 57.732603), 4326)::geography
    ),
    (
        'Ферма',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.302687, 57.901681), 4326)::geography
    ),
    (
        'Кондратово',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.103670, 57.981292), 4326)::geography
    ),
    (
        'Лобаново',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.284972, 57.856916), 4326)::geography
    ),
    (
        'Култаево',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.939362, 57.894893), 4326)::geography
    ),
    (
        'Гамово',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.105442, 57.864885), 4326)::geography
    ),
    (
        'Сылва',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(56.774920, 58.027454), 4326)::geography
    ),
    (
        'Уральский',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.620368, 57.897509), 4326)::geography
    ),
    (
        'Майский',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(55.695679, 57.993632), 4326)::geography
    ),
    (
        'Сарс',
        'Пермский край',
        'RU',
        'Asia/Yekaterinburg',
        ST_SetSRID(ST_MakePoint(57.134316, 56.550275), 4326)::geography
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

INSERT INTO permissions (code, description)
VALUES
    ('taxi_park.drivers.create', 'Taxi park can create own drivers'),
    ('taxi_park.finance.view', 'Taxi park can view own finance')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role, permission_code)
VALUES
    ('taxi_park', 'taxi_park.profile.manage'),
    ('taxi_park', 'taxi_park.drivers.create'),
    ('taxi_park', 'taxi_park.drivers.manage'),
    ('taxi_park', 'taxi_park.cars.manage'),
    ('taxi_park', 'taxi_park.orders.view'),
    ('taxi_park', 'taxi_park.earnings.view'),
    ('taxi_park', 'taxi_park.finance.view')
ON CONFLICT (role, permission_code) DO NOTHING;

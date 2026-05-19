CREATE TABLE permissions (
    code TEXT PRIMARY KEY,
    description TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT permissions_code_not_blank CHECK (length(trim(code)) > 0)
);

CREATE TABLE role_permissions (
    role user_role NOT NULL,
    permission_code TEXT NOT NULL REFERENCES permissions(code) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role, permission_code)
);

INSERT INTO permissions (code, description)
VALUES
    ('passenger.orders.create', 'Passenger can create an order'),
    ('passenger.orders.current.view', 'Passenger can view current order status'),
    ('passenger.orders.history.view', 'Passenger can view trip history'),
    ('passenger.orders.cancel', 'Passenger can cancel own order'),
    ('passenger.trips.rate', 'Passenger can rate completed trip'),
    ('passenger.driver.contact.view', 'Passenger can see assigned driver contact'),

    ('driver.auth.authorize', 'Driver can authorize in driver application'),
    ('driver.verification.submit', 'Driver can submit verification data'),
    ('driver.status.online', 'Driver can go online'),
    ('driver.status.offline', 'Driver can go offline'),
    ('driver.location.update', 'Driver can update location'),
    ('driver.orders.receive', 'Driver can receive order offers'),
    ('driver.orders.accept', 'Driver can accept order'),
    ('driver.orders.reject', 'Driver can reject order'),
    ('driver.trips.status.update', 'Driver can update trip status'),
    ('driver.orders.complete', 'Driver can complete order'),
    ('driver.earnings.view', 'Driver can view earnings'),
    ('driver.balance.view', 'Driver can view balance'),
    ('driver.transactions.view', 'Driver can view financial transactions'),
    ('driver.orders.history.view', 'Driver can view order history'),

    ('admin.cities.manage', 'Admin can manage cities'),
    ('admin.zones.manage', 'Admin can manage zones'),
    ('admin.tariffs.manage', 'Admin can manage tariffs'),
    ('admin.drivers.manage', 'Admin can add and manage drivers'),
    ('admin.drivers.block', 'Admin can block drivers'),
    ('admin.orders.view', 'Admin can view orders'),
    ('admin.statistics.view', 'Admin can view statistics'),
    ('admin.orders.driver.assign', 'Admin can manually assign drivers'),
    ('admin.commissions.manage', 'Admin can manage commissions'),
    ('admin.finance.view', 'Admin can view finance overview'),
    ('admin.complaints.view', 'Admin can view complaints'),
    ('admin.ratings.view', 'Admin can view ratings'),

    ('dispatcher.orders.create', 'Dispatcher can create order by phone request'),
    ('dispatcher.passengers.find', 'Dispatcher can find passenger by phone'),
    ('dispatcher.orders.driver.assign', 'Dispatcher can assign nearest driver'),
    ('dispatcher.orders.address.update', 'Dispatcher can update order address'),
    ('dispatcher.orders.cancel', 'Dispatcher can cancel order'),
    ('dispatcher.drivers.contact', 'Dispatcher can contact driver'),

    ('taxi_park.profile.manage', 'Taxi park can manage own profile'),
    ('taxi_park.drivers.create', 'Taxi park can create own drivers'),
    ('taxi_park.drivers.manage', 'Taxi park can manage own drivers'),
    ('taxi_park.cars.manage', 'Taxi park can manage own cars'),
    ('taxi_park.orders.view', 'Taxi park can view own fleet orders'),
    ('taxi_park.earnings.view', 'Taxi park can view own fleet earnings'),
    ('taxi_park.finance.view', 'Taxi park can view own finance')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role, permission_code)
VALUES
    ('passenger', 'passenger.orders.create'),
    ('passenger', 'passenger.orders.current.view'),
    ('passenger', 'passenger.orders.history.view'),
    ('passenger', 'passenger.orders.cancel'),
    ('passenger', 'passenger.trips.rate'),
    ('passenger', 'passenger.driver.contact.view'),

    ('driver', 'driver.auth.authorize'),
    ('driver', 'driver.verification.submit'),
    ('driver', 'driver.status.online'),
    ('driver', 'driver.status.offline'),
    ('driver', 'driver.location.update'),
    ('driver', 'driver.orders.receive'),
    ('driver', 'driver.orders.accept'),
    ('driver', 'driver.orders.reject'),
    ('driver', 'driver.trips.status.update'),
    ('driver', 'driver.orders.complete'),
    ('driver', 'driver.earnings.view'),
    ('driver', 'driver.balance.view'),
    ('driver', 'driver.transactions.view'),
    ('driver', 'driver.orders.history.view'),

    ('admin', 'admin.cities.manage'),
    ('admin', 'admin.zones.manage'),
    ('admin', 'admin.tariffs.manage'),
    ('admin', 'admin.drivers.manage'),
    ('admin', 'admin.drivers.block'),
    ('admin', 'admin.orders.view'),
    ('admin', 'admin.statistics.view'),
    ('admin', 'admin.orders.driver.assign'),
    ('admin', 'admin.commissions.manage'),
    ('admin', 'admin.finance.view'),
    ('admin', 'admin.complaints.view'),
    ('admin', 'admin.ratings.view'),

    ('dispatcher', 'dispatcher.orders.create'),
    ('dispatcher', 'dispatcher.passengers.find'),
    ('dispatcher', 'dispatcher.orders.driver.assign'),
    ('dispatcher', 'dispatcher.orders.address.update'),
    ('dispatcher', 'dispatcher.orders.cancel'),
    ('dispatcher', 'dispatcher.drivers.contact'),

    ('taxi_park', 'taxi_park.profile.manage'),
    ('taxi_park', 'taxi_park.drivers.create'),
    ('taxi_park', 'taxi_park.drivers.manage'),
    ('taxi_park', 'taxi_park.cars.manage'),
    ('taxi_park', 'taxi_park.orders.view'),
    ('taxi_park', 'taxi_park.earnings.view'),
    ('taxi_park', 'taxi_park.finance.view')
ON CONFLICT (role, permission_code) DO NOTHING;

CREATE INDEX idx_role_permissions_permission_code
ON role_permissions (permission_code);

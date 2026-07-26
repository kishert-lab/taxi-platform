INSERT INTO permissions (code, description)
VALUES
    ('driver.balance.view', 'Driver can view balance'),
    ('driver.transactions.view', 'Driver can view financial transactions'),
    ('admin.finance.view', 'Admin can view finance overview')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role, permission_code)
VALUES
    ('driver', 'driver.balance.view'),
    ('driver', 'driver.transactions.view'),
    ('admin', 'admin.finance.view')
ON CONFLICT (role, permission_code) DO NOTHING;

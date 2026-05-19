DELETE FROM role_permissions
WHERE (role = 'driver' AND permission_code IN ('driver.balance.view', 'driver.transactions.view'))
   OR (role = 'admin' AND permission_code = 'admin.finance.view');

DELETE FROM permissions
WHERE code IN ('driver.balance.view', 'driver.transactions.view', 'admin.finance.view');

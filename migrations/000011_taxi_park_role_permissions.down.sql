DELETE FROM role_permissions
WHERE role = 'taxi_park'
  AND permission_code IN ('taxi_park.drivers.create', 'taxi_park.finance.view');

DELETE FROM permissions
WHERE code IN ('taxi_park.drivers.create', 'taxi_park.finance.view');

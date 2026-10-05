-- Do not silently merge balances or erase transfer history during a rollback.
DO $$ BEGIN
 IF EXISTS (SELECT 1 FROM driver_park_invites) OR EXISTS (
  SELECT 1 FROM driver_balances GROUP BY driver_id HAVING count(*) > 1
 ) THEN RAISE EXCEPTION 'Driver transfers exist; rollback requires an explicit data recovery plan'; END IF;
END $$;
DELETE FROM role_permissions WHERE permission_code='taxi_park.drivers.invite';
DELETE FROM permissions WHERE code='taxi_park.drivers.invite';
DROP TABLE driver_push_tokens;
DROP TABLE driver_park_history;
DROP TABLE driver_park_invites;
ALTER TABLE driver_balances DROP CONSTRAINT driver_balances_driver_park_unique;
ALTER TABLE driver_balances DROP COLUMN id, DROP COLUMN taxi_park_id;
ALTER TABLE driver_balances ADD PRIMARY KEY(driver_id);

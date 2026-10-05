package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/driverinvite"
	"time"
)

type PostgresDriverParkInviteRepository struct{ pool *pgxpool.Pool }

func NewPostgresDriverParkInviteRepository(pool *pgxpool.Pool) *PostgresDriverParkInviteRepository {
	return &PostgresDriverParkInviteRepository{pool}
}

type driverInviteTransaction struct{ transaction pgx.Tx }

func (repository *PostgresDriverParkInviteRepository) Transact(ctx context.Context, callback func(driverinvite.Transaction) error) error {
	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin invite transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)
	if err := callback(&driverInviteTransaction{transaction}); err != nil {
		return err
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit invite transaction: %w", err)
	}
	return nil
}
func inviteQueryError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrParkInviteNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}
func (store *driverInviteTransaction) ActorPark(ctx context.Context, actor uuid.UUID) (driverinvite.Park, error) {
	var parkID uuid.UUID
	err := store.transaction.QueryRow(ctx, `SELECT tp.id FROM taxi_parks tp JOIN users u ON u.id=$1 AND u.is_active AND u.deleted_at IS NULL
 WHERE tp.deleted_at IS NULL AND ((tp.owner_user_id=u.id AND u.role='taxi_park') OR EXISTS (
 SELECT 1 FROM taxi_park_staff s WHERE s.taxi_park_id=tp.id AND s.user_id=u.id AND s.is_active AND s.deleted_at IS NULL AND s.role IN ('dispatcher','taxi_park') AND u.role IN ('dispatcher','taxi_park')))
 ORDER BY tp.id LIMIT 1`, actor).Scan(&parkID)
	if errors.Is(err, pgx.ErrNoRows) {
		return driverinvite.Park{}, domain.ErrParkInviteForbidden
	}
	if err != nil {
		return driverinvite.Park{}, fmt.Errorf("resolve invite actor park: %w", err)
	}
	return store.Park(ctx, parkID)
}
func (store *driverInviteTransaction) Park(ctx context.Context, id uuid.UUID) (driverinvite.Park, error) {
	var park driverinvite.Park
	err := store.transaction.QueryRow(ctx, `SELECT p.id,p.city_id,p.name,(p.deleted_at IS NULL AND COALESCE(s.is_active,true))
 FROM taxi_parks p LEFT JOIN taxi_park_settings s ON s.taxi_park_id=p.id WHERE p.id=$1 FOR SHARE OF p`, id).Scan(&park.ID, &park.CityID, &park.Name, &park.Active)
	if err != nil {
		return park, inviteQueryError("select invite park", err)
	}
	// Lock settings too, so park deactivation cannot race with acceptance.
	var active bool
	err = store.transaction.QueryRow(ctx, `SELECT is_active FROM taxi_park_settings WHERE taxi_park_id=$1 FOR SHARE`, id).Scan(&active)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return park, fmt.Errorf("lock invite park settings: %w", err)
	}
	if err == nil {
		park.Active = park.Active && active
	}
	return park, nil
}
func (store *driverInviteTransaction) LockDriver(ctx context.Context, userID uuid.UUID, phone string) (driverinvite.Driver, error) {
	return store.scanDriver(ctx, `WHERE (u.id=$1 OR ($2<>'' AND u.phone=$2)) AND u.role='driver' AND u.is_active AND u.deleted_at IS NULL AND d.deleted_at IS NULL FOR UPDATE OF d`, userID, phone)
}
func (store *driverInviteTransaction) LockInviteDriver(ctx context.Context, id uuid.UUID) (driverinvite.Driver, error) {
	return store.scanDriver(ctx, `WHERE d.id=(SELECT driver_id FROM driver_park_invites WHERE id=$1) AND d.deleted_at IS NULL FOR UPDATE OF d`, id)
}
func (store *driverInviteTransaction) scanDriver(ctx context.Context, where string, args ...any) (driverinvite.Driver, error) {
	var driver driverinvite.Driver
	err := store.transaction.QueryRow(ctx, `SELECT d.id,d.user_id,d.taxi_park_id,trim(COALESCE(u.first_name,'')||' '||COALESCE(u.last_name,'')),d.status FROM drivers d JOIN users u ON u.id=d.user_id `+where, args...).Scan(&driver.ID, &driver.UserID, &driver.ParkID, &driver.Name, &driver.Status)
	if err != nil {
		return driver, inviteQueryError("lock invite driver", err)
	}
	return driver, nil
}

const inviteSelect = `SELECT i.id,i.driver_id,i.from_taxi_park_id,i.to_taxi_park_id,COALESCE(previous.name,''),target.name,
 trim(COALESCE(u.first_name,'')||' '||COALESCE(u.last_name,'')),i.status,i.created_by_user_id,i.created_at,i.expires_at,i.accepted_at,i.declined_at,i.cancelled_at,i.cancellation_reason
 FROM driver_park_invites i JOIN drivers d ON d.id=i.driver_id JOIN users u ON u.id=d.user_id
 JOIN taxi_parks target ON target.id=i.to_taxi_park_id LEFT JOIN taxi_parks previous ON previous.id=i.from_taxi_park_id `

func scanInvite(row pgx.Row) (domain.DriverParkInvite, error) {
	var invite domain.DriverParkInvite
	err := row.Scan(&invite.ID, &invite.DriverID, &invite.FromTaxiParkID, &invite.ToTaxiParkID, &invite.FromTaxiParkName, &invite.ToTaxiParkName, &invite.DriverName, &invite.Status, &invite.CreatedByUserID, &invite.CreatedAt, &invite.ExpiresAt, &invite.AcceptedAt, &invite.DeclinedAt, &invite.CancelledAt, &invite.CancellationReason)
	if err != nil {
		return invite, inviteQueryError("scan park invite", err)
	}
	return invite, nil
}
func (store *driverInviteTransaction) Invite(ctx context.Context, id uuid.UUID) (domain.DriverParkInvite, error) {
	return scanInvite(store.transaction.QueryRow(ctx, inviteSelect+`WHERE i.id=$1 FOR UPDATE OF i`, id))
}
func (store *driverInviteTransaction) Pending(ctx context.Context, driver, park uuid.UUID) (*domain.DriverParkInvite, error) {
	invite, err := scanInvite(store.transaction.QueryRow(ctx, inviteSelect+`WHERE i.driver_id=$1 AND i.to_taxi_park_id=$2 AND i.status='pending'`, driver, park))
	if errors.Is(err, domain.ErrParkInviteNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &invite, nil
}
func (store *driverInviteTransaction) Insert(ctx context.Context, invite domain.DriverParkInvite) error {
	_, err := store.transaction.Exec(ctx, `INSERT INTO driver_park_invites(id,driver_id,from_taxi_park_id,to_taxi_park_id,status,created_by_user_id,created_at,expires_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, invite.ID, invite.DriverID, invite.FromTaxiParkID, invite.ToTaxiParkID, invite.Status, invite.CreatedByUserID, invite.CreatedAt, invite.ExpiresAt)
	if err != nil {
		return fmt.Errorf("insert park invite: %w", err)
	}
	return nil
}
func (store *driverInviteTransaction) Save(ctx context.Context, invite domain.DriverParkInvite) error {
	_, err := store.transaction.Exec(ctx, `UPDATE driver_park_invites SET status=$2,accepted_at=$3,declined_at=$4,cancelled_at=$5,cancellation_reason=$6 WHERE id=$1`, invite.ID, invite.Status, invite.AcceptedAt, invite.DeclinedAt, invite.CancelledAt, invite.CancellationReason)
	if err != nil {
		return fmt.Errorf("update park invite: %w", err)
	}
	return nil
}
func (store *driverInviteTransaction) Expire(ctx context.Context, driver uuid.UUID, now time.Time) error {
	_, err := store.transaction.Exec(ctx, `UPDATE driver_park_invites SET status='expired' WHERE driver_id=$1 AND status='pending' AND expires_at<=$2`, driver, now)
	if err != nil {
		return fmt.Errorf("expire park invites: %w", err)
	}
	return nil
}
func (store *driverInviteTransaction) HasOrders(ctx context.Context, driver uuid.UUID) (bool, error) {
	var exists bool
	err := store.transaction.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM orders WHERE (driver_id=$1 OR preassigned_driver_id=$1) AND status NOT IN ('completed','cancelled','failed') AND deleted_at IS NULL)`, driver).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check driver transfer orders: %w", err)
	}
	return exists, nil
}
func (store *driverInviteTransaction) Transfer(ctx context.Context, driver driverinvite.Driver, park driverinvite.Park, invite domain.DriverParkInvite, now time.Time) error {
	statements := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO driver_park_history(driver_id,taxi_park_id,joined_at) SELECT id,taxi_park_id,created_at FROM drivers WHERE id=$1 AND taxi_park_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM driver_park_history WHERE driver_id=$1 AND left_at IS NULL)`, []any{driver.ID}},
		{`UPDATE driver_park_history SET left_at=$2 WHERE driver_id=$1 AND left_at IS NULL`, []any{driver.ID, now}},
		{`DELETE FROM car_driver_assignments WHERE driver_id=$1`, []any{driver.ID}},
		{`UPDATE cars SET driver_id=NULL WHERE driver_id=$1`, []any{driver.ID}},
		{`UPDATE drivers SET taxi_park_id=$2,city_id=$3,status='offline',commission_percent=NULL,taxi_park_comment=NULL,verification_checked_by=NULL,verification_checked_at=NULL,verification_status='pending_verification',is_verified=false WHERE id=$1`, []any{driver.ID, park.ID, park.CityID}},
		{`INSERT INTO driver_park_history(driver_id,taxi_park_id,joined_at,transfer_invite_id) VALUES($1,$2,$3,$4)`, []any{driver.ID, park.ID, now, invite.ID}},
		{`INSERT INTO driver_balances(driver_id,taxi_park_id) VALUES($1,$2) ON CONFLICT(driver_id,taxi_park_id) DO NOTHING`, []any{driver.ID, park.ID}},
	}
	for index, statement := range statements {
		if _, err := store.transaction.Exec(ctx, statement.query, statement.args...); err != nil {
			return fmt.Errorf("persist driver transfer step %d: %w", index, err)
		}
	}
	return nil
}
func (store *driverInviteTransaction) CancelOtherPending(ctx context.Context, driver, id uuid.UUID, now time.Time) error {
	_, err := store.transaction.Exec(ctx, `UPDATE driver_park_invites SET status=CASE WHEN expires_at<=$3 THEN 'expired' ELSE 'cancelled' END,cancelled_at=CASE WHEN expires_at>$3 THEN $3 END,cancellation_reason='driver_transferred_to_another_park' WHERE driver_id=$1 AND id<>$2 AND status='pending'`, driver, id, now)
	if err != nil {
		return fmt.Errorf("cancel superseded park invites: %w", err)
	}
	return nil
}
func (store *driverInviteTransaction) Audit(ctx context.Context, actor uuid.UUID, event string, invite domain.DriverParkInvite) error {
	payload, err := json.Marshal(invite)
	if err != nil {
		return fmt.Errorf("marshal invite audit: %w", err)
	}
	_, err = store.transaction.Exec(ctx, `INSERT INTO finance_audit_events(actor_user_id,event_type,payload) VALUES($1,$2,$3)`, actor, event, payload)
	if err != nil {
		return fmt.Errorf("insert invite audit: %w", err)
	}
	return nil
}
func (repository *PostgresDriverParkInviteRepository) List(ctx context.Context, actor uuid.UUID, park bool) ([]domain.DriverParkInvite, error) {
	result := make([]domain.DriverParkInvite, 0)
	err := repository.Transact(ctx, func(transaction driverinvite.Transaction) error {
		store := transaction.(*driverInviteTransaction)
		where := `WHERE u.id=$1 AND u.is_active AND u.deleted_at IS NULL AND d.deleted_at IS NULL`
		id := actor
		if park {
			target, err := store.ActorPark(ctx, actor)
			if err != nil {
				return err
			}
			id = target.ID
			where = `WHERE i.to_taxi_park_id=$1`
		}
		rows, err := store.transaction.Query(ctx, inviteSelect+where+` ORDER BY i.created_at DESC LIMIT 200`, id)
		if err != nil {
			return fmt.Errorf("list park invites: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			invite, err := scanInvite(rows)
			if err != nil {
				return err
			}
			result = append(result, invite)
		}
		if err := rows.Err(); err != nil {
			return fmt.Errorf("iterate park invites: %w", err)
		}
		return nil
	})
	return result, err
}
func (repository *PostgresDriverParkInviteRepository) SavePushToken(ctx context.Context, user uuid.UUID, token, platform string) error {
	tag, err := repository.pool.Exec(ctx, `INSERT INTO driver_push_tokens(user_id,token,platform) SELECT u.id,$2,$3 FROM users u JOIN drivers d ON d.user_id=u.id WHERE u.id=$1 AND u.role='driver' AND u.is_active AND u.deleted_at IS NULL AND d.deleted_at IS NULL ON CONFLICT(token) DO UPDATE SET user_id=EXCLUDED.user_id,platform=EXCLUDED.platform,updated_at=now()`, user, token, platform)
	if err != nil {
		return fmt.Errorf("save driver push token: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrParkInviteNotFound
	}
	return nil
}
func (repository *PostgresDriverParkInviteRepository) DeletePushToken(ctx context.Context, user uuid.UUID, token string) error {
	_, err := repository.pool.Exec(ctx, `DELETE FROM driver_push_tokens WHERE user_id=$1 AND token=$2`, user, token)
	if err != nil {
		return fmt.Errorf("delete driver push token: %w", err)
	}
	return nil
}

func (repository *PostgresDriverParkInviteRepository) ListDriverPushTokens(ctx context.Context, driver uuid.UUID) ([]string, error) {
	rows, err := repository.pool.Query(ctx, `SELECT t.token FROM driver_push_tokens t JOIN drivers d ON d.user_id=t.user_id WHERE d.id=$1`, driver)
	if err != nil {
		return nil, fmt.Errorf("list driver push tokens: %w", err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var token string
		if err := rows.Scan(&token); err != nil {
			return nil, fmt.Errorf("scan driver push token: %w", err)
		}
		result = append(result, token)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate driver push tokens: %w", err)
	}
	return result, nil
}
func (repository *PostgresDriverParkInviteRepository) ListParkInviteRecipients(ctx context.Context, park uuid.UUID) ([]uuid.UUID, error) {
	rows, err := repository.pool.Query(ctx, `SELECT p.owner_user_id FROM taxi_parks p JOIN users u ON u.id=p.owner_user_id WHERE p.id=$1 AND u.is_active AND u.deleted_at IS NULL UNION SELECT s.user_id FROM taxi_park_staff s JOIN users u ON u.id=s.user_id WHERE s.taxi_park_id=$1 AND s.is_active AND s.deleted_at IS NULL AND u.is_active AND u.deleted_at IS NULL`, park)
	if err != nil {
		return nil, fmt.Errorf("list park invite recipients: %w", err)
	}
	defer rows.Close()
	result := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan park invite recipient: %w", err)
		}
		result = append(result, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate park invite recipients: %w", err)
	}
	return result, nil
}

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/dispatch"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	orderapp "github.com/kishert-lab/taxi-platform/internal/order"
)

type PostgresDispatchOrderRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresDispatchOrderRepository(pool *pgxpool.Pool) *PostgresDispatchOrderRepository {
	return &PostgresDispatchOrderRepository{pool: pool}
}

func (repository *PostgresDispatchOrderRepository) GetOrderByID(ctx context.Context, orderID uuid.UUID) (domain.Order, error) {
	order, err := scanDispatchOrder(repository.pool.QueryRow(ctx, `SELECT `+dispatchOrderSelectColumns+` FROM orders WHERE id = $1 AND deleted_at IS NULL`, orderID))
	if err != nil {
		return domain.Order{}, fmt.Errorf("select dispatch order by id: %w", err)
	}
	return order, nil
}

func (repository *PostgresDispatchOrderRepository) GetCurrentOrderByPassengerID(ctx context.Context, passengerID uuid.UUID) (domain.Order, error) {
	const query = `SELECT ` + dispatchOrderSelectColumns + `
		FROM orders
		WHERE passenger_id = $1
		  AND status IN ('created', 'searching', 'driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress')
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1`

	order, err := scanDispatchOrder(repository.pool.QueryRow(ctx, query, passengerID))
	if err != nil {
		return domain.Order{}, fmt.Errorf("select current passenger order: %w", err)
	}
	return order, nil
}

func (repository *PostgresDispatchOrderRepository) GetCurrentOrderByDriverID(ctx context.Context, driverID uuid.UUID) (domain.Order, error) {
	const query = `SELECT ` + dispatchOrderSelectColumns + `
		FROM orders
		WHERE driver_id = $1
		  AND status IN ('driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress')
		  AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT 1`

	order, err := scanDispatchOrder(repository.pool.QueryRow(ctx, query, driverID))
	if err != nil {
		return domain.Order{}, fmt.Errorf("select current driver order: %w", err)
	}
	return order, nil
}

func (repository *PostgresDispatchOrderRepository) MarkOrderSearching(ctx context.Context, orderID uuid.UUID) error {
	const query = `
		UPDATE orders
		SET status = 'searching',
		    version = version + 1
		WHERE id = $1
		  AND status IN ('created', 'searching')
		  AND driver_id IS NULL
		  AND deleted_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("update order searching: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("update order searching: %w", pgx.ErrNoRows)
	}
	return nil
}

func (repository *PostgresDispatchOrderRepository) GetCandidateFare(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID) (dispatch.CandidateFare, error) {
	var fare dispatch.CandidateFare
	var tariff domain.TaxiParkTariff
	var carClassID pgtype.UUID
	err := repository.pool.QueryRow(ctx, `
		SELECT t.id, t.taxi_park_id, t.car_class_id, t.name, COALESCE(t.description, ''),
		       t.pricing_mode, t.fare_mode, t.priority, t.base_price_cents,
		       t.fixed_price_cents, t.price_per_km_cents, t.price_per_minute_cents,
		       t.minimum_price_cents, t.fixed_routes, t.is_active, t.created_at, t.updated_at,
		       (o.metadata->'pricing_snapshot'->>'route_distance_meters')::bigint,
		       (o.metadata->'pricing_snapshot'->>'route_duration_seconds')::bigint
		FROM orders o
		JOIN drivers d ON d.id = $2
		JOIN taxi_park_tariffs t ON t.taxi_park_id = d.taxi_park_id
		WHERE o.id = $1 AND o.status = 'searching' AND o.deleted_at IS NULL
		  AND t.car_class_id = o.car_class_id AND t.fare_mode = o.fare_mode AND t.is_active
		ORDER BY t.priority DESC, t.id DESC LIMIT 1`, orderID, driverID).Scan(
		&tariff.ID, &tariff.TaxiParkID, &carClassID, &tariff.Name, &tariff.Description,
		&tariff.PricingMode, &tariff.FareMode, &tariff.Priority, &tariff.BasePrice.Amount,
		&tariff.FixedPrice.Amount, &tariff.PricePerKM.Amount, &tariff.PricePerMinute.Amount,
		&tariff.MinimumPrice.Amount, &tariff.FixedRoutes, &tariff.IsActive, &tariff.CreatedAt, &tariff.UpdatedAt,
		&fare.DistanceMeters, &fare.DurationSeconds)
	if err != nil {
		return dispatch.CandidateFare{}, fmt.Errorf("get candidate park fare: %w", err)
	}
	if carClassID.Valid {
		id := uuid.UUID(carClassID.Bytes)
		tariff.CarClassID = &id
	}
	tariff.BasePrice.Currency = "RUB"
	tariff.FixedPrice.Currency = "RUB"
	tariff.PricePerKM.Currency = "RUB"
	tariff.PricePerMinute.Currency = "RUB"
	tariff.MinimumPrice.Currency = "RUB"
	fare.Tariff = tariff
	return fare, nil
}

const assignDriverQuery = `
		WITH reserved AS (UPDATE orders
		SET driver_id = $2,
		    car_id = (
		      SELECT c.id
		      FROM drivers d
		      JOIN cars c ON c.taxi_park_id = d.taxi_park_id
		      LEFT JOIN car_driver_assignments cda ON cda.car_id = c.id
		      WHERE d.id = $2
		        AND (c.driver_id = d.id OR cda.driver_id = d.id)
		        AND c.deleted_at IS NULL
		        AND c.is_active = true
		        AND c.verification_status = 'verified'
		        AND COALESCE(c.permit_expires_at, current_date + interval '1 day') >= current_date
		        AND COALESCE(c.osago_expires_at, current_date + interval '1 day') >= current_date
		        AND EXISTS (
		            SELECT 1 FROM car_classes requested_class
		            JOIN car_classes driver_class ON driver_class.code = c.car_class
		            WHERE requested_class.id = orders.car_class_id
		              AND driver_class.sort_order >= requested_class.sort_order
		              AND requested_class.is_active = true AND requested_class.deleted_at IS NULL
		              AND driver_class.is_active = true AND driver_class.deleted_at IS NULL
		        )
		      ORDER BY (c.driver_id = d.id) DESC, c.created_at DESC
		      LIMIT 1
		    ),
		    park_id = (
		      SELECT d.taxi_park_id
		      FROM drivers d
		      WHERE d.id = $2
		    ),
		    assigned_tariff_id = (
		      SELECT tariff.id
		      FROM drivers d
		      JOIN taxi_park_tariffs tariff ON tariff.taxi_park_id = d.taxi_park_id
		      WHERE d.id = $2
		        AND tariff.car_class_id = orders.car_class_id
		        AND tariff.fare_mode = orders.fare_mode
		        AND tariff.is_active = true
		        ORDER BY tariff.priority DESC, tariff.id DESC
		      LIMIT 1
		    ),
		    metadata = jsonb_set(
		        COALESCE(metadata, '{}'::jsonb),
		        '{assigned_tariff_snapshot}',
		        COALESCE((
		            SELECT jsonb_build_object(
		                'id', tariff.id,
		                'taxi_park_id', tariff.taxi_park_id,
		                'car_class_id', tariff.car_class_id,
		                'pricing_mode', tariff.pricing_mode,
		                'fare_mode', tariff.fare_mode,
		                'priority', tariff.priority,
		                'base_price_cents', tariff.base_price_cents,
		                'fixed_price_cents', tariff.fixed_price_cents,
		                'price_per_km_cents', tariff.price_per_km_cents,
		                'price_per_minute_cents', tariff.price_per_minute_cents,
		                'minimum_price_cents', tariff.minimum_price_cents,
		                'fixed_routes', tariff.fixed_routes,
			            'captured_at', $3::timestamptz
		            )
		            FROM drivers d
		            JOIN taxi_park_tariffs tariff ON tariff.taxi_park_id = d.taxi_park_id
		            WHERE d.id = $2
		              AND tariff.car_class_id = orders.car_class_id
		              AND tariff.fare_mode = orders.fare_mode
		              AND tariff.is_active = true
		            ORDER BY tariff.priority DESC, tariff.id DESC
		            LIMIT 1
		        ), 'null'::jsonb),
		        true
		    ),
		    status = 'driver_assigned',
		    accepted_at = $3::timestamptz,
		    price_confirmation_state = CASE WHEN $10::boolean THEN 'pending' ELSE 'confirmed' END,
		    proposed_price_cents = $6::bigint,
		    agreed_price_cents = CASE WHEN $10::boolean OR fare_mode <> 'fixed_quote' THEN NULL ELSE $6::bigint END,
		    price_confirmation_expires_at = CASE WHEN $10::boolean THEN $7::timestamptz ELSE NULL END,
		    version = version + 1
		WHERE id = $1
		  AND status = 'searching'
		  AND (metadata->'pricing_snapshot'->>'route_distance_meters')::bigint = $8
		  AND (metadata->'pricing_snapshot'->>'route_duration_seconds')::bigint = $9
		  AND driver_id IS NULL
		  AND deleted_at IS NULL
		  AND EXISTS (
		    SELECT 1
		    FROM drivers d
		    JOIN taxi_park_tariffs tariff ON tariff.taxi_park_id = d.taxi_park_id
		    WHERE d.id = $2
		      AND d.status = 'online'
		      AND d.is_verified = true
		      AND d.verification_status = 'verified'
		      AND d.deleted_at IS NULL
		      AND EXISTS (
		          SELECT 1 FROM taxi_parks tp
		          LEFT JOIN taxi_park_settings settings ON settings.taxi_park_id = tp.id
		          WHERE tp.id = d.taxi_park_id AND tp.deleted_at IS NULL
		            AND COALESCE(settings.is_active, true)
		      )
		      AND EXISTS (
		          SELECT 1 FROM driver_locations dl
		          WHERE dl.driver_id = d.id
		            AND dl.updated_at >= now() - interval '30 seconds'
		      )
		      AND NOT EXISTS (
		          SELECT 1 FROM orders active_order
		          WHERE active_order.driver_id = d.id
		            AND active_order.status IN ('driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress')
		            AND active_order.deleted_at IS NULL
		      )
		      AND EXISTS (
		          SELECT 1 FROM cars c
		          LEFT JOIN car_driver_assignments cda ON cda.car_id = c.id
		          JOIN car_classes driver_class ON driver_class.code = c.car_class
		          JOIN car_classes requested_class ON requested_class.id = orders.car_class_id
		          WHERE c.taxi_park_id = d.taxi_park_id
		            AND (c.driver_id = d.id OR cda.driver_id = d.id)
		            AND c.is_active = true AND c.deleted_at IS NULL
		            AND c.verification_status = 'verified'
		            AND COALESCE(c.permit_expires_at, current_date + interval '1 day') >= current_date
		            AND COALESCE(c.osago_expires_at, current_date + interval '1 day') >= current_date
		            AND driver_class.sort_order >= requested_class.sort_order
		      )
		      AND tariff.car_class_id = orders.car_class_id
		      AND tariff.fare_mode = orders.fare_mode
		      AND tariff.is_active = true
		      AND tariff.id = $4 AND tariff.updated_at = $5
		      AND NOT EXISTS (
		          SELECT 1 FROM taxi_park_tariffs higher
		          WHERE higher.taxi_park_id = tariff.taxi_park_id
		            AND higher.car_class_id = tariff.car_class_id
		            AND higher.fare_mode = tariff.fare_mode AND higher.is_active
		            AND (higher.priority > tariff.priority OR (higher.priority = tariff.priority AND higher.id > tariff.id))
		      )
		  )
		RETURNING id)
		INSERT INTO order_events (order_id, actor_driver_id, event_type, payload, created_at)
		SELECT id, $2, 'order.updated',
		       jsonb_build_object(
		           'event', CASE WHEN $10::boolean THEN 'price_confirmation_requested' ELSE 'price_confirmed' END,
		           'driver_id', $2,
		           'price_cents', $6::bigint,
		           'expires_at', CASE WHEN $10::boolean THEN $7::timestamptz ELSE NULL END
		       ), $3::timestamptz
		FROM reserved`

func (repository *PostgresDispatchOrderRepository) AssignDriver(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID, acceptedAt time.Time, fare dispatch.CandidateFare, priceCents int64, confirmationExpiresAt time.Time, requiresPassengerPriceConfirmation bool) (bool, error) {

	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin driver assignment: %w", err)
	}
	defer rollbackTx(ctx, transaction)
	var lockedDriver uuid.UUID
	if err := transaction.QueryRow(ctx, `SELECT id FROM drivers WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, driverID).Scan(&lockedDriver); err != nil {
		return false, fmt.Errorf("lock assigned driver: %w", err)
	}
	commandTag, err := transaction.Exec(ctx, assignDriverQuery, orderID, driverID, acceptedAt, fare.Tariff.ID, fare.Tariff.UpdatedAt, priceCents, confirmationExpiresAt, fare.DistanceMeters, fare.DurationSeconds, requiresPassengerPriceConfirmation)
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) && postgresError.Code == "23505" {
			return false, nil
		}
		return false, fmt.Errorf("assign driver atomically: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit driver assignment: %w", err)
	}
	return commandTag.RowsAffected() == 1, nil
}

func (repository *PostgresDispatchOrderRepository) ConfirmDriverPrice(ctx context.Context, orderID uuid.UUID, passengerID uuid.UUID) (bool, error) {
	commandTag, err := repository.pool.Exec(ctx, `
		WITH confirmed AS (UPDATE orders
		SET price_confirmation_state = 'confirmed',
		    agreed_price_cents = CASE WHEN fare_mode = 'fixed_quote' THEN proposed_price_cents ELSE NULL END,
		    version = version + 1
		WHERE id = $1 AND passenger_id = $2 AND status = 'driver_assigned'
		  AND price_confirmation_state = 'pending'
		  AND price_confirmation_expires_at > now()
		  AND proposed_price_cents > 0 AND deleted_at IS NULL
		RETURNING id, driver_id, proposed_price_cents, fare_mode)
		INSERT INTO order_events (order_id, event_type, payload, created_at)
		SELECT id, 'order.updated',
		       jsonb_build_object('event', 'price_confirmed', 'driver_id', driver_id, 'price_cents', proposed_price_cents, 'fare_mode', fare_mode), now()
		FROM confirmed`, orderID, passengerID)
	if err != nil {
		return false, fmt.Errorf("confirm driver park price: %w", err)
	}
	return commandTag.RowsAffected() == 1, nil
}

func (repository *PostgresDispatchOrderRepository) ReleaseDriverReservation(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID) (bool, error) {
	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin driver reservation release: %w", err)
	}
	defer rollbackTx(ctx, transaction)
	commandTag, err := transaction.Exec(ctx, `
		UPDATE orders
		SET status = 'searching', driver_id = NULL, car_id = NULL, park_id = NULL,
		    assigned_tariff_id = NULL, accepted_at = NULL,
		    price_confirmation_state = NULL, proposed_price_cents = NULL,
		    price_confirmation_expires_at = NULL, agreed_price_cents = NULL,
		    metadata = COALESCE(metadata, '{}'::jsonb) - 'assigned_tariff_snapshot',
		    declined_driver_ids = array_append(declined_driver_ids, $2),
		    version = version + 1
		WHERE id = $1 AND driver_id = $2 AND status = 'driver_assigned'
		  AND price_confirmation_state = 'pending' AND deleted_at IS NULL`, orderID, driverID)
	if err != nil {
		return false, fmt.Errorf("release order driver reservation: %w", err)
	}
	if commandTag.RowsAffected() != 1 {
		return false, nil
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO order_events (order_id, actor_driver_id, event_type, payload, created_at)
		VALUES ($1, $2::uuid, 'order.updated', jsonb_build_object('event', 'price_declined', 'driver_id', $2::uuid), now())`, orderID, driverID); err != nil {
		return false, fmt.Errorf("record declined driver price: %w", err)
	}
	if _, err := transaction.Exec(ctx, `UPDATE drivers SET status = 'online' WHERE id = $1 AND status = 'busy' AND deleted_at IS NULL`, driverID); err != nil {
		return false, fmt.Errorf("mark released driver online: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit driver reservation release: %w", err)
	}
	return true, nil
}

func (repository *PostgresDispatchOrderRepository) TransitionOrderStatus(ctx context.Context, transition domain.OrderTransition) (domain.Order, bool, error) {
	const query = `
		UPDATE orders
		SET status = $4,
		    version = version + 1,
		    cancelled_at = CASE WHEN $4 = 'cancelled' THEN $5 ELSE cancelled_at END,
		    cancellation_reason = CASE WHEN $4 = 'cancelled' THEN $6 ELSE cancellation_reason END,
		    started_at = CASE WHEN $4 = 'in_progress' THEN $5 ELSE started_at END,
		    completed_at = CASE WHEN $4 = 'completed' THEN $5 ELSE completed_at END
		WHERE id = $1
		  AND status = $2
		  AND (
		      status <> 'driver_assigned'
		      OR fare_mode = 'metered'
		      OR COALESCE(price_confirmation_state, 'confirmed') = 'confirmed'
		      OR $4 = 'cancelled'
		  )
		  AND version = $3
		  AND deleted_at IS NULL
		RETURNING ` + dispatchOrderSelectColumns

	order, err := scanDispatchOrder(repository.pool.QueryRow(
		ctx,
		query,
		transition.OrderID,
		transition.FromStatus,
		transition.ExpectedVersion,
		transition.ToStatus,
		transition.OccurredAt,
		nullableString(transition.Reason),
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, false, nil
		}
		return domain.Order{}, false, fmt.Errorf("transition order status: %w", err)
	}
	return order, true, nil
}

func (repository *PostgresDispatchOrderRepository) IncrementDispatchAttempt(ctx context.Context, orderID uuid.UUID) error {
	const query = `
		UPDATE orders
		SET dispatch_attempt = dispatch_attempt + 1
		WHERE id = $1
		  AND status = 'searching'
		  AND deleted_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, orderID)
	if err != nil {
		return fmt.Errorf("increment dispatch attempt: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("increment dispatch attempt: %w", pgx.ErrNoRows)
	}
	return nil
}

func (repository *PostgresDispatchOrderRepository) FailOrder(ctx context.Context, orderID uuid.UUID, reason string) error {
	const query = `
		UPDATE orders
		SET status = 'failed',
		    cancellation_reason = $2,
		    cancelled_at = now(),
		    version = version + 1
		WHERE id = $1
		  AND status = 'searching'
		  AND driver_id IS NULL
		  AND deleted_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, orderID, reason)
	if err != nil {
		return fmt.Errorf("fail dispatch order: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("fail dispatch order: %w", pgx.ErrNoRows)
	}
	return nil
}

func (repository *PostgresDispatchOrderRepository) AddOrderEvent(ctx context.Context, event dispatch.OrderEvent) error {
	return repository.insertOrderEvent(ctx, event.OrderID, event.ActorUserID, event.ActorDriverID, event.EventType, event.Payload, event.CreatedAt)
}

func (repository *PostgresDispatchOrderRepository) AddStateEvent(ctx context.Context, event orderapp.OrderEvent) error {
	return repository.insertOrderEvent(ctx, event.OrderID, event.ActorUserID, event.ActorDriverID, event.EventType, event.Payload, time.Now().UTC())
}

func (repository *PostgresDispatchOrderRepository) ListExpiredPriceConfirmations(ctx context.Context, limit int) ([]uuid.UUID, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id FROM orders
		WHERE status = 'driver_assigned' AND price_confirmation_state = 'pending'
		  AND price_confirmation_expires_at <= now() AND deleted_at IS NULL
		ORDER BY price_confirmation_expires_at LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list expired price confirmations: %w", err)
	}
	defer rows.Close()
	orderIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan expired price confirmation: %w", err)
		}
		orderIDs = append(orderIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired price confirmations: %w", err)
	}
	return orderIDs, nil
}

func (repository *PostgresDispatchOrderRepository) ListSearchingOrders(ctx context.Context, limit int) ([]uuid.UUID, error) {
	const query = `
		SELECT id
		FROM orders
		WHERE status = 'searching'
		  AND driver_id IS NULL
		  AND deleted_at IS NULL
		ORDER BY updated_at ASC
		LIMIT $1`

	rows, err := repository.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("select searching orders: %w", err)
	}
	defer rows.Close()

	orderIDs := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var orderID uuid.UUID
		if err := rows.Scan(&orderID); err != nil {
			return nil, fmt.Errorf("scan searching order id: %w", err)
		}
		orderIDs = append(orderIDs, orderID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate searching order ids: %w", err)
	}
	return orderIDs, nil
}

func (repository *PostgresDispatchOrderRepository) insertOrderEvent(ctx context.Context, orderID uuid.UUID, actorUserID *uuid.UUID, actorDriverID *uuid.UUID, eventType domain.OrderEventType, payload map[string]any, createdAt time.Time) error {
	const query = `
		INSERT INTO order_events (order_id, actor_user_id, actor_driver_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`

	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal order event payload: %w", err)
	}
	if _, err := repository.pool.Exec(ctx, query, orderID, actorUserID, actorDriverID, eventType, payloadBytes, createdAt); err != nil {
		return fmt.Errorf("insert order event: %w", err)
	}
	return nil
}

const dispatchOrderSelectColumns = `
	id,
	passenger_id,
	driver_id,
	car_id,
	park_id,
	preassigned_driver_id,
	city_id,
	tariff_id,
	assigned_tariff_id,
	car_class_id,
	status,
	order_type,
	scheduled_status,
	pickup_address,
	COALESCE(pickup_entrance, '') AS pickup_entrance,
	COALESCE(pickup_comment, '') AS pickup_comment,
	ST_Y(pickup_location::geometry) AS pickup_latitude,
	ST_X(pickup_location::geometry) AS pickup_longitude,
	COALESCE(destination_address, '') AS destination_address,
	CASE WHEN destination_location IS NULL THEN NULL ELSE ST_Y(destination_location::geometry) END AS destination_latitude,
	CASE WHEN destination_location IS NULL THEN NULL ELSE ST_X(destination_location::geometry) END AS destination_longitude,
	scheduled_at,
	activation_at,
	COALESCE(scheduled_timezone, '') AS scheduled_timezone,
	requested_at,
	accepted_at,
	started_at,
	completed_at,
	cancelled_at,
	activated_at,
	scheduled_cancelled_at,
	scheduled_expired_at,
	COALESCE(cancellation_reason, '') AS cancellation_reason,
	COALESCE(scheduled_cancel_reason, '') AS scheduled_cancel_reason,
	CASE WHEN estimated_price IS NULL THEN NULL ELSE (estimated_price * 100)::bigint END AS estimated_price_cents,
	CASE WHEN final_price IS NULL THEN NULL ELSE (final_price * 100)::bigint END AS final_price_cents,
	payment_method,
	COALESCE(passenger_comment, '') AS passenger_comment,
	passenger_location_sharing_enabled,
	dispatch_attempt,
	scheduled_created_by,
	version,
	created_at,
	updated_at,
	deleted_at,
	fare_mode,
	COALESCE(price_confirmation_state, ''),
	proposed_price_cents,
	agreed_price_cents,
	price_confirmation_expires_at,
	declined_driver_ids,
	COALESCE(metadata->>'created_by_role', '') AS created_by_role`

func scanDispatchOrder(row pgx.Row) (domain.Order, error) {
	var order domain.Order
	var driverID pgtype.UUID
	var carID pgtype.UUID
	var parkID pgtype.UUID
	var preassignedDriverID pgtype.UUID
	var tariffID pgtype.UUID
	var assignedTariffID pgtype.UUID
	var carClassID pgtype.UUID
	var scheduledCreatedBy pgtype.UUID
	var destinationLatitude pgtype.Float8
	var destinationLongitude pgtype.Float8
	var scheduledAt pgtype.Timestamptz
	var activationAt pgtype.Timestamptz
	var acceptedAt pgtype.Timestamptz
	var startedAt pgtype.Timestamptz
	var completedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz
	var activatedAt pgtype.Timestamptz
	var scheduledCancelledAt pgtype.Timestamptz
	var scheduledExpiredAt pgtype.Timestamptz
	var deletedAt pgtype.Timestamptz
	var proposedPrice pgtype.Int8
	var agreedPrice pgtype.Int8
	var confirmationExpiresAt pgtype.Timestamptz
	var estimatedPrice pgtype.Int8
	var finalPrice pgtype.Int8
	var scheduledStatus pgtype.Text
	var pickupLatitude float64
	var pickupLongitude float64

	if err := row.Scan(
		&order.ID,
		&order.PassengerID,
		&driverID,
		&carID,
		&parkID,
		&preassignedDriverID,
		&order.CityID,
		&tariffID,
		&assignedTariffID,
		&carClassID,
		&order.Status,
		&order.OrderType,
		&scheduledStatus,
		&order.PickupAddress,
		&order.PickupEntrance,
		&order.PickupComment,
		&pickupLatitude,
		&pickupLongitude,
		&order.DestinationAddress,
		&destinationLatitude,
		&destinationLongitude,
		&scheduledAt,
		&activationAt,
		&order.ScheduledTimezone,
		&order.RequestedAt,
		&acceptedAt,
		&startedAt,
		&completedAt,
		&cancelledAt,
		&activatedAt,
		&scheduledCancelledAt,
		&scheduledExpiredAt,
		&order.CancellationReason,
		&order.ScheduledCancelReason,
		&estimatedPrice,
		&finalPrice,
		&order.PaymentMethod,
		&order.PassengerComment,
		&order.PassengerLocationSharingEnabled,
		&order.DispatchAttempt,
		&scheduledCreatedBy,
		&order.Version,
		&order.CreatedAt,
		&order.UpdatedAt,
		&deletedAt,
		&order.FareMode,
		&order.PriceConfirmationState,
		&proposedPrice,
		&agreedPrice,
		&confirmationExpiresAt,
		&order.DeclinedDriverIDs,
		&order.CreatedByRole,
	); err != nil {
		return domain.Order{}, err
	}
	if proposedPrice.Valid {
		order.ProposedPriceCents = &proposedPrice.Int64
	}
	if agreedPrice.Valid {
		order.AgreedPriceCents = &agreedPrice.Int64
	}
	if confirmationExpiresAt.Valid {
		order.PriceConfirmationExpiresAt = &confirmationExpiresAt.Time
	}
	if estimatedPrice.Valid {
		order.EstimatedPrice = &domain.Money{Amount: estimatedPrice.Int64, Currency: "RUB"}
	}
	if finalPrice.Valid {
		order.FinalPrice = &domain.Money{Amount: finalPrice.Int64, Currency: "RUB"}
	}

	pickupLocation, err := domain.NewCoordinates(pickupLatitude, pickupLongitude)
	if err != nil {
		return domain.Order{}, fmt.Errorf("build pickup coordinates: %w", err)
	}
	order.PickupLocation = pickupLocation

	if driverID.Valid {
		value := uuid.UUID(driverID.Bytes)
		order.DriverID = &value
	}
	if carID.Valid {
		value := uuid.UUID(carID.Bytes)
		order.CarID = &value
	}
	if parkID.Valid {
		value := uuid.UUID(parkID.Bytes)
		order.ParkID = &value
	}
	if preassignedDriverID.Valid {
		value := uuid.UUID(preassignedDriverID.Bytes)
		order.PreassignedDriverID = &value
	}
	if tariffID.Valid {
		value := uuid.UUID(tariffID.Bytes)
		order.TariffID = &value
	}
	if assignedTariffID.Valid {
		value := uuid.UUID(assignedTariffID.Bytes)
		order.AssignedTariffID = &value
	}
	if carClassID.Valid {
		value := uuid.UUID(carClassID.Bytes)
		order.CarClassID = &value
	}
	if scheduledStatus.Valid {
		value := domain.ScheduledOrderStatus(scheduledStatus.String)
		order.ScheduledStatus = &value
	}
	if destinationLatitude.Valid && destinationLongitude.Valid {
		value, err := domain.NewCoordinates(destinationLatitude.Float64, destinationLongitude.Float64)
		if err != nil {
			return domain.Order{}, fmt.Errorf("build destination coordinates: %w", err)
		}
		order.DestinationLocation = &value
	}
	if acceptedAt.Valid {
		order.AcceptedAt = &acceptedAt.Time
	}
	if scheduledAt.Valid {
		order.ScheduledAt = &scheduledAt.Time
	}
	if activationAt.Valid {
		order.ActivationAt = &activationAt.Time
	}
	if startedAt.Valid {
		order.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		order.CompletedAt = &completedAt.Time
	}
	if cancelledAt.Valid {
		order.CancelledAt = &cancelledAt.Time
	}
	if activatedAt.Valid {
		order.ActivatedAt = &activatedAt.Time
	}
	if scheduledCancelledAt.Valid {
		order.ScheduledCancelledAt = &scheduledCancelledAt.Time
	}
	if scheduledExpiredAt.Valid {
		order.ScheduledExpiredAt = &scheduledExpiredAt.Time
	}
	if scheduledCreatedBy.Valid {
		value := uuid.UUID(scheduledCreatedBy.Bytes)
		order.ScheduledCreatedBy = &value
	}
	if deletedAt.Valid {
		order.DeletedAt = &deletedAt.Time
	}
	return order, nil
}

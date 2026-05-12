package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/develoop/taxi-platform/internal/dispatch"
	"github.com/develoop/taxi-platform/internal/domain"
	orderapp "github.com/develoop/taxi-platform/internal/order"
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

func (repository *PostgresDispatchOrderRepository) AssignDriver(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID, acceptedAt time.Time) (bool, error) {
	const query = `
		UPDATE orders
		SET driver_id = $2,
		    status = 'driver_assigned',
		    accepted_at = $3,
		    version = version + 1
		WHERE id = $1
		  AND status = 'searching'
		  AND driver_id IS NULL
		  AND deleted_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, orderID, driverID, acceptedAt)
	if err != nil {
		return false, fmt.Errorf("assign driver atomically: %w", err)
	}
	return commandTag.RowsAffected() == 1, nil
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
	city_id,
	tariff_id,
	status,
	pickup_address,
	ST_Y(pickup_location::geometry) AS pickup_latitude,
	ST_X(pickup_location::geometry) AS pickup_longitude,
	COALESCE(destination_address, '') AS destination_address,
	CASE WHEN destination_location IS NULL THEN NULL ELSE ST_Y(destination_location::geometry) END AS destination_latitude,
	CASE WHEN destination_location IS NULL THEN NULL ELSE ST_X(destination_location::geometry) END AS destination_longitude,
	requested_at,
	accepted_at,
	started_at,
	completed_at,
	cancelled_at,
	COALESCE(cancellation_reason, '') AS cancellation_reason,
	payment_method,
	COALESCE(passenger_comment, '') AS passenger_comment,
	dispatch_attempt,
	version,
	created_at,
	updated_at,
	deleted_at`

func scanDispatchOrder(row pgx.Row) (domain.Order, error) {
	var order domain.Order
	var driverID pgtype.UUID
	var tariffID pgtype.UUID
	var destinationLatitude pgtype.Float8
	var destinationLongitude pgtype.Float8
	var acceptedAt pgtype.Timestamptz
	var startedAt pgtype.Timestamptz
	var completedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz
	var deletedAt pgtype.Timestamptz
	var pickupLatitude float64
	var pickupLongitude float64

	if err := row.Scan(
		&order.ID,
		&order.PassengerID,
		&driverID,
		&order.CityID,
		&tariffID,
		&order.Status,
		&order.PickupAddress,
		&pickupLatitude,
		&pickupLongitude,
		&order.DestinationAddress,
		&destinationLatitude,
		&destinationLongitude,
		&order.RequestedAt,
		&acceptedAt,
		&startedAt,
		&completedAt,
		&cancelledAt,
		&order.CancellationReason,
		&order.PaymentMethod,
		&order.PassengerComment,
		&order.DispatchAttempt,
		&order.Version,
		&order.CreatedAt,
		&order.UpdatedAt,
		&deletedAt,
	); err != nil {
		return domain.Order{}, err
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
	if tariffID.Valid {
		value := uuid.UUID(tariffID.Bytes)
		order.TariffID = &value
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
	if startedAt.Valid {
		order.StartedAt = &startedAt.Time
	}
	if completedAt.Valid {
		order.CompletedAt = &completedAt.Time
	}
	if cancelledAt.Valid {
		order.CancelledAt = &cancelledAt.Time
	}
	if deletedAt.Valid {
		order.DeletedAt = &deletedAt.Time
	}
	return order, nil
}

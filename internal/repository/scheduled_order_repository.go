package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PostgresScheduledOrderRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresScheduledOrderRepository(pool *pgxpool.Pool) *PostgresScheduledOrderRepository {
	return &PostgresScheduledOrderRepository{pool: pool}
}

func (repository *PostgresScheduledOrderRepository) ListDueOrderIDs(ctx context.Context, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := repository.pool.Query(ctx, `
		SELECT id
		FROM orders
		WHERE order_type = 'scheduled'
		  AND scheduled_status IN ('scheduled_confirmed', 'scheduled_driver_assigned')
		  AND activation_at IS NOT NULL
		  AND activation_at <= now()
		  AND deleted_at IS NULL
		ORDER BY activation_at ASC
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("select due scheduled order ids: %w", err)
	}
	defer rows.Close()

	orderIDs := make([]uuid.UUID, 0, limit)
	for rows.Next() {
		var orderID uuid.UUID
		if err := rows.Scan(&orderID); err != nil {
			return nil, fmt.Errorf("scan due scheduled order id: %w", err)
		}
		orderIDs = append(orderIDs, orderID)
	}
	return orderIDs, rows.Err()
}

func (repository *PostgresScheduledOrderRepository) ActivateOrder(ctx context.Context, orderID uuid.UUID) (domain.Order, bool, error) {
	order, err := scanDispatchOrder(repository.pool.QueryRow(ctx, `
		UPDATE orders
		SET scheduled_status = 'scheduled_activated',
		    activated_at = now(),
		    status = CASE
		        WHEN preassigned_driver_id IS NULL THEN 'created'::order_status
		        ELSE 'driver_assigned'::order_status
		    END,
		    driver_id = COALESCE(driver_id, preassigned_driver_id),
		    accepted_at = CASE
		        WHEN preassigned_driver_id IS NULL THEN accepted_at
		        ELSE COALESCE(accepted_at, now())
		    END,
		    version = version + 1
		WHERE id = $1
		  AND order_type = 'scheduled'
		  AND scheduled_status IN ('scheduled_confirmed', 'scheduled_driver_assigned')
		  AND activation_at IS NOT NULL
		  AND activation_at <= now()
		  AND deleted_at IS NULL
		RETURNING `+dispatchOrderSelectColumns, orderID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, false, nil
		}
		return domain.Order{}, false, fmt.Errorf("activate scheduled order: %w", err)
	}
	return order, true, nil
}

func (repository *PostgresScheduledOrderRepository) ExpirePendingOrders(ctx context.Context, limit int) ([]domain.Order, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := repository.pool.Query(ctx, `
		WITH candidates AS (
			SELECT o.id
			FROM orders o
			JOIN taxi_park_settings s ON s.taxi_park_id::text = o.metadata->>'taxi_park_id'
			WHERE o.order_type = 'scheduled'
			  AND o.scheduled_status IN ('scheduled_confirmed', 'scheduled_driver_assigned')
			  AND o.scheduled_at IS NOT NULL
			  AND o.scheduled_at + make_interval(mins => s.scheduled_expire_after_minutes) <= now()
			  AND o.deleted_at IS NULL
			ORDER BY o.scheduled_at ASC
			LIMIT $1
		)
		UPDATE orders o
		SET scheduled_status = 'scheduled_expired',
		    scheduled_expired_at = now(),
		    version = version + 1
		FROM candidates
		WHERE o.id = candidates.id
		RETURNING
			o.id,
			o.passenger_id,
			o.driver_id,
			o.preassigned_driver_id,
			o.city_id,
			o.tariff_id,
			o.status,
			o.order_type,
			o.scheduled_status,
			o.pickup_address,
			ST_Y(o.pickup_location::geometry) AS pickup_latitude,
			ST_X(o.pickup_location::geometry) AS pickup_longitude,
			COALESCE(o.destination_address, '') AS destination_address,
			CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_Y(o.destination_location::geometry) END AS destination_latitude,
			CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_X(o.destination_location::geometry) END AS destination_longitude,
			o.scheduled_at,
			o.activation_at,
			COALESCE(o.scheduled_timezone, '') AS scheduled_timezone,
			o.requested_at,
			o.accepted_at,
			o.started_at,
			o.completed_at,
			o.cancelled_at,
			o.activated_at,
			o.scheduled_cancelled_at,
			o.scheduled_expired_at,
			COALESCE(o.cancellation_reason, '') AS cancellation_reason,
			COALESCE(o.scheduled_cancel_reason, '') AS scheduled_cancel_reason,
			o.payment_method,
			COALESCE(o.passenger_comment, '') AS passenger_comment,
			o.dispatch_attempt,
			o.scheduled_created_by,
			o.version,
			o.created_at,
			o.updated_at,
			o.deleted_at`, limit)
	if err != nil {
		return nil, fmt.Errorf("expire pending scheduled orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0)
	for rows.Next() {
		order, scanErr := scanDispatchOrder(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan expired scheduled order: %w", scanErr)
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (repository *PostgresScheduledOrderRepository) GetTaxiParkSettingsByOrderID(ctx context.Context, orderID uuid.UUID) (domain.TaxiParkSettings, error) {
	settings, err := scanTaxiParkSettings(repository.pool.QueryRow(ctx, `
		SELECT `+taxiParkSettingsColumns+`
		FROM orders o
		JOIN taxi_park_settings s ON s.taxi_park_id::text = o.metadata->>'taxi_park_id'
		JOIN taxi_parks p ON p.id = s.taxi_park_id
		JOIN cities c ON c.id = p.city_id
		WHERE o.id = $1
		  AND o.deleted_at IS NULL`, orderID))
	if err != nil {
		return domain.TaxiParkSettings{}, fmt.Errorf("select taxi park settings by order id: %w", err)
	}
	return settings, nil
}

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

	"github.com/kishert-lab/taxi-platform/internal/domain"
	driverapp "github.com/kishert-lab/taxi-platform/internal/driver"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geo"
)

type PostgresDriverMobileRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresDriverMobileRepository(pool *pgxpool.Pool) *PostgresDriverMobileRepository {
	return &PostgresDriverMobileRepository{pool: pool}
}

func (repository *PostgresDriverMobileRepository) GetProfileByUserID(ctx context.Context, userID uuid.UUID) (driverapp.Profile, error) {
	profile, err := scanDriverMobileProfile(repository.pool.QueryRow(ctx, driverMobileProfileQuery(`
		WHERE d.user_id = $1
		  AND d.deleted_at IS NULL
		  AND u.deleted_at IS NULL`), userID))
	if err != nil {
		return driverapp.Profile{}, mapDriverMobileScanError("select driver profile", err)
	}
	return profile, nil
}

func (repository *PostgresDriverMobileRepository) UpdateProfileByUserID(ctx context.Context, userID uuid.UUID, patch driverapp.ProfilePatch) (driverapp.Profile, error) {
	profile, err := scanDriverMobileProfile(repository.pool.QueryRow(ctx, `
		WITH updated_driver AS (
			UPDATE drivers
			SET license_number = COALESCE($4::text, license_number)
			WHERE user_id = $1 AND deleted_at IS NULL
			RETURNING id
		),
		updated_user AS (
			UPDATE users
			SET first_name = COALESCE($2::text, first_name),
			    last_name = COALESCE($3::text, last_name),
			    profile_photo_url = COALESCE($5::text, profile_photo_url)
			WHERE id = $1 AND deleted_at IS NULL
			RETURNING id
		)
		`+driverMobileProfileSelect+`
		WHERE d.user_id = $1
		  AND d.id IN (SELECT id FROM updated_driver)
		  AND u.id IN (SELECT id FROM updated_user)`,
		userID,
		patch.FirstName,
		patch.LastName,
		patch.LicenseNumber,
		patch.PhotoURL,
	))
	if err != nil {
		return driverapp.Profile{}, mapDriverMobileScanError("update driver profile", err)
	}
	return profile, nil
}

func (repository *PostgresDriverMobileRepository) SetStatusByUserID(ctx context.Context, userID uuid.UUID, status domain.DriverStatus) (driverapp.Profile, error) {
	if status == domain.DriverStatusOnline {
		return repository.markOnline(ctx, userID)
	}
	if status == domain.DriverStatusOffline {
		return repository.markOffline(ctx, userID)
	}
	return driverapp.Profile{}, fmt.Errorf("unsupported driver status update: %w", domain.ErrInvalidDriverStatus)
}

func (repository *PostgresDriverMobileRepository) ListCarsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Car, error) {
	var driverID uuid.UUID
	if err := repository.pool.QueryRow(ctx, `
		SELECT d.id
		FROM drivers d
		JOIN users u ON u.id = d.user_id
		WHERE d.user_id = $1
		  AND d.deleted_at IS NULL
		  AND u.deleted_at IS NULL`, userID).Scan(&driverID); err != nil {
		return nil, mapDriverMobileScanError("select driver before list cars", err)
	}

	rows, err := repository.pool.Query(ctx, `
		SELECT DISTINCT c.id, c.taxi_park_id, c.driver_id, c.brand, c.model, COALESCE(c.year, 0),
		       c.plate_number, COALESCE(c.vin, ''), COALESCE(c.sts, ''), COALESCE(c.pts, ''),
		       c.color, COALESCE(c.car_class, ''), c.verification_status, COALESCE(c.owner_details, ''),
		       c.osago_expires_at, c.diagnostic_card_expires_at, COALESCE(c.taxi_permit_number, ''),
		       COALESCE(c.regional_registry_number, ''), COALESCE(c.permit_region, ''),
		       c.permit_issued_at, c.permit_expires_at, c.taxi_permit_verified, c.regional_registry_verified,
		       c.regional_requirements_compliant, c.has_taxi_color_scheme, c.has_orange_roof_lamp,
		       c.has_passenger_info, c.osago_verified, c.diagnostic_card_verified, c.technical_state_verified,
		       c.localization_compliant, c.legal_use_basis_verified, c.verification_checked_at,
		       c.verification_checked_by, c.is_active, c.created_at, c.updated_at
		FROM cars c
		LEFT JOIN car_driver_assignments cda ON cda.car_id = c.id
		WHERE c.deleted_at IS NULL
		  AND (c.driver_id = $1 OR cda.driver_id = $1)
		ORDER BY c.created_at DESC`, driverID)
	if err != nil {
		return nil, fmt.Errorf("select driver cars: %w", err)
	}
	defer rows.Close()

	cars := make([]domain.Car, 0)
	for rows.Next() {
		car, err := scanTaxiParkCar(rows)
		if err != nil {
			return nil, fmt.Errorf("scan driver car: %w", err)
		}
		attachedDriverIDs, err := repository.carAssignments(ctx, car.ID)
		if err != nil {
			return nil, err
		}
		car.AttachedDriverIDs = attachedDriverIDs
		cars = append(cars, car)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate driver cars: %w", err)
	}
	return cars, nil
}

func (repository *PostgresDriverMobileRepository) GetCurrentOrderByUserID(ctx context.Context, userID uuid.UUID) (driverapp.CurrentOrder, error) {
	order, err := scanDriverCurrentOrder(repository.pool.QueryRow(ctx, `
		SELECT o.id,
		       d.id,
		       p.id,
		       trim(concat_ws(' ', COALESCE(p.first_name, ''), COALESCE(p.last_name, ''))) AS passenger_name,
		       p.phone,
		       COALESCE(p.profile_photo_url, '') AS passenger_photo_url,
		       p.rating::float8,
		       p.ratings_count,
		       o.pickup_address,
		       ST_Y(o.pickup_location::geometry) AS pickup_latitude,
		       ST_X(o.pickup_location::geometry) AS pickup_longitude,
		       COALESCE(o.destination_address, '') AS destination_address,
		       CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_Y(o.destination_location::geometry) END AS destination_latitude,
		       CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_X(o.destination_location::geometry) END AS destination_longitude,
		       o.status,
		       CASE
		           WHEN o.final_price IS NOT NULL THEN (o.final_price * 100)::bigint
		           WHEN o.estimated_price IS NOT NULL THEN (o.estimated_price * 100)::bigint
		           ELSE NULL
		       END AS price_amount,
		       COALESCE(o.passenger_comment, '') AS passenger_comment,
		       o.version,
		       o.created_at
		FROM orders o
		JOIN drivers d ON d.id = o.driver_id
		JOIN users p ON p.id = o.passenger_id
		WHERE d.user_id = $1
		  AND d.deleted_at IS NULL
		  AND o.deleted_at IS NULL
		  AND o.status IN ('driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress')
		ORDER BY o.created_at DESC
		LIMIT 1`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return driverapp.CurrentOrder{}, driverapp.ErrCurrentOrderNotFound
		}
		return driverapp.CurrentOrder{}, fmt.Errorf("select current driver order: %w", err)
	}
	return order, nil
}

func (repository *PostgresDriverMobileRepository) GetOrderByUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (driverapp.CurrentOrder, error) {
	order, err := scanDriverCurrentOrder(repository.pool.QueryRow(ctx, `
		SELECT o.id,
		       d.id,
		       p.id,
		       trim(concat_ws(' ', COALESCE(p.first_name, ''), COALESCE(p.last_name, ''))) AS passenger_name,
		       p.phone,
		       COALESCE(p.profile_photo_url, '') AS passenger_photo_url,
		       p.rating::float8,
		       p.ratings_count,
		       o.pickup_address,
		       ST_Y(o.pickup_location::geometry) AS pickup_latitude,
		       ST_X(o.pickup_location::geometry) AS pickup_longitude,
		       COALESCE(o.destination_address, '') AS destination_address,
		       CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_Y(o.destination_location::geometry) END AS destination_latitude,
		       CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_X(o.destination_location::geometry) END AS destination_longitude,
		       o.status,
		       CASE
		           WHEN o.final_price IS NOT NULL THEN (o.final_price * 100)::bigint
		           WHEN o.estimated_price IS NOT NULL THEN (o.estimated_price * 100)::bigint
		           ELSE NULL
		       END AS price_amount,
		       COALESCE(o.passenger_comment, '') AS passenger_comment,
		       o.version,
		       o.created_at
		FROM orders o
		JOIN drivers d ON d.id = o.driver_id
		JOIN users p ON p.id = o.passenger_id
		WHERE o.id = $1
		  AND d.user_id = $2
		  AND d.deleted_at IS NULL
		  AND o.deleted_at IS NULL`, orderID, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return driverapp.CurrentOrder{}, driverapp.ErrCurrentOrderNotFound
		}
		return driverapp.CurrentOrder{}, fmt.Errorf("select driver order details: %w", err)
	}
	return order, nil
}

func (repository *PostgresDriverMobileRepository) ListOrderHistoryByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]driverapp.CurrentOrder, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := repository.pool.Query(ctx, `
		SELECT o.id,
		       d.id,
		       p.id,
		       trim(concat_ws(' ', COALESCE(p.first_name, ''), COALESCE(p.last_name, ''))) AS passenger_name,
		       p.phone,
		       COALESCE(p.profile_photo_url, '') AS passenger_photo_url,
		       p.rating::float8,
		       p.ratings_count,
		       o.pickup_address,
		       ST_Y(o.pickup_location::geometry) AS pickup_latitude,
		       ST_X(o.pickup_location::geometry) AS pickup_longitude,
		       COALESCE(o.destination_address, '') AS destination_address,
		       CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_Y(o.destination_location::geometry) END AS destination_latitude,
		       CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_X(o.destination_location::geometry) END AS destination_longitude,
		       o.status,
		       CASE
		           WHEN o.final_price IS NOT NULL THEN (o.final_price * 100)::bigint
		           WHEN o.estimated_price IS NOT NULL THEN (o.estimated_price * 100)::bigint
		           ELSE NULL
		       END AS price_amount,
		       COALESCE(o.passenger_comment, '') AS passenger_comment,
		       o.version,
		       o.created_at
		FROM orders o
		JOIN drivers d ON d.id = o.driver_id
		JOIN users p ON p.id = o.passenger_id
		WHERE d.user_id = $1
		  AND d.deleted_at IS NULL
		  AND o.deleted_at IS NULL
		  AND o.status IN ('completed', 'cancelled', 'failed')
		ORDER BY COALESCE(o.completed_at, o.cancelled_at, o.updated_at, o.created_at) DESC
		LIMIT $2`, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("select driver order history: %w", err)
	}
	defer rows.Close()

	orders := make([]driverapp.CurrentOrder, 0, limit)
	for rows.Next() {
		order, err := scanDriverCurrentOrder(rows)
		if err != nil {
			return nil, fmt.Errorf("scan driver order history: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate driver order history: %w", err)
	}
	return orders, nil
}

func selectDriverCurrentOrderByID(ctx context.Context, transaction pgx.Tx, userID uuid.UUID, orderID uuid.UUID) (driverapp.CurrentOrder, error) {
	return scanDriverCurrentOrder(transaction.QueryRow(ctx, `
		SELECT o.id,
		       d.id,
		       p.id,
		       trim(concat_ws(' ', COALESCE(p.first_name, ''), COALESCE(p.last_name, ''))) AS passenger_name,
		       p.phone,
		       COALESCE(p.profile_photo_url, '') AS passenger_photo_url,
		       p.rating::float8,
		       p.ratings_count,
		       o.pickup_address,
		       ST_Y(o.pickup_location::geometry) AS pickup_latitude,
		       ST_X(o.pickup_location::geometry) AS pickup_longitude,
		       COALESCE(o.destination_address, '') AS destination_address,
		       CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_Y(o.destination_location::geometry) END AS destination_latitude,
		       CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_X(o.destination_location::geometry) END AS destination_longitude,
		       o.status,
		       CASE
		           WHEN o.final_price IS NOT NULL THEN (o.final_price * 100)::bigint
		           WHEN o.estimated_price IS NOT NULL THEN (o.estimated_price * 100)::bigint
		           ELSE NULL
		       END AS price_amount,
		       COALESCE(o.passenger_comment, '') AS passenger_comment,
		       o.version,
		       o.created_at
		FROM orders o
		JOIN drivers d ON d.id = o.driver_id
		JOIN users p ON p.id = o.passenger_id
		WHERE o.id = $1
		  AND d.user_id = $2
		  AND d.deleted_at IS NULL
		  AND o.deleted_at IS NULL`, orderID, userID))
}

func transitionDispatchOrderStatus(ctx context.Context, transaction pgx.Tx, transition domain.OrderTransition, finalPriceCents *int64) (domain.Order, bool, error) {
	order, err := scanDispatchOrder(transaction.QueryRow(ctx, `
		UPDATE orders
		SET status = $4::order_status,
		    version = version + 1,
		    cancelled_at = CASE WHEN $4::order_status = 'cancelled'::order_status THEN $5 ELSE cancelled_at END,
		    cancellation_reason = CASE WHEN $4::order_status = 'cancelled'::order_status THEN $6::text ELSE cancellation_reason END,
		    started_at = CASE WHEN $4::order_status = 'in_progress'::order_status THEN $5 ELSE started_at END,
		    completed_at = CASE WHEN $4::order_status = 'completed'::order_status THEN $5 ELSE completed_at END,
		    final_price = CASE WHEN $4::order_status = 'completed'::order_status THEN ($7::bigint::numeric / 100) ELSE final_price END
		WHERE id = $1
		  AND status = $2::order_status
		  AND version = $3
		  AND deleted_at IS NULL
		RETURNING `+dispatchOrderSelectColumns,
		transition.OrderID,
		transition.FromStatus,
		transition.ExpectedVersion,
		transition.ToStatus,
		transition.OccurredAt,
		nullableString(transition.Reason),
		finalPriceCents,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Order{}, false, nil
		}
		return domain.Order{}, false, fmt.Errorf("transition driver order status: %w", err)
	}
	return order, true, nil
}

func insertDriverMobileOrderEvent(ctx context.Context, transaction pgx.Tx, transition domain.OrderTransition, updated domain.Order) error {
	payloadBytes, err := json.Marshal(map[string]any{
		"order_id":    updated.ID,
		"from_status": transition.FromStatus,
		"to_status":   transition.ToStatus,
		"version":     updated.Version,
		"reason":      transition.Reason,
		"occurred_at": transition.OccurredAt,
	})
	if err != nil {
		return fmt.Errorf("marshal driver order transition event: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO order_events (order_id, actor_user_id, actor_driver_id, event_type, payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		updated.ID,
		transition.ActorUserID,
		transition.ActorDriverID,
		domain.EventTypeForOrderStatus(transition.ToStatus),
		payloadBytes,
		transition.OccurredAt,
	); err != nil {
		return fmt.Errorf("insert driver order transition event: %w", err)
	}
	return nil
}

func markDriverOnlineInTx(ctx context.Context, transaction pgx.Tx, driverID uuid.UUID) error {
	if _, err := transaction.Exec(ctx, `
		UPDATE drivers
		SET status = 'online'
		WHERE id = $1
		  AND status IN ('busy', 'online')
		  AND deleted_at IS NULL`, driverID); err != nil {
		return fmt.Errorf("mark driver online after order close: %w", err)
	}
	return nil
}

func selectDriverDomainOrderForTransition(ctx context.Context, transaction pgx.Tx, userID uuid.UUID, orderID uuid.UUID) (domain.Order, error) {
	var order domain.Order
	var driverID uuid.UUID
	if err := transaction.QueryRow(ctx, `
		SELECT o.id, o.passenger_id, o.driver_id, o.status, o.version
		FROM orders o
		JOIN drivers d ON d.id = o.driver_id
		WHERE o.id = $1
		  AND d.user_id = $2
		  AND o.deleted_at IS NULL
		  AND d.deleted_at IS NULL
		FOR UPDATE OF o`, orderID, userID).Scan(
		&order.ID,
		&order.PassengerID,
		&driverID,
		&order.Status,
		&order.Version,
	); err != nil {
		return domain.Order{}, mapDriverMobileScanError("select driver order for transition", err)
	}
	order.DriverID = &driverID
	return order, nil
}

func (repository *PostgresDriverMobileRepository) TransitionOrderByUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID, toStatus domain.OrderStatus, reason string, finalPriceCents *int64) (driverapp.CurrentOrder, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return driverapp.CurrentOrder{}, fmt.Errorf("begin driver order transition: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	currentOrder, err := selectDriverDomainOrderForTransition(ctx, transaction, userID, orderID)
	if err != nil {
		return driverapp.CurrentOrder{}, err
	}
	transition, err := domain.NewOrderTransition(currentOrder, toStatus, &userID, currentOrder.DriverID, reason, time.Now().UTC())
	if err != nil {
		return driverapp.CurrentOrder{}, err
	}

	updated, changed, err := transitionDispatchOrderStatus(ctx, transaction, transition, finalPriceCents)
	if err != nil {
		return driverapp.CurrentOrder{}, err
	}
	if !changed {
		return driverapp.CurrentOrder{}, fmt.Errorf("driver order transition concurrent update: %w", domain.ErrInvalidOrderStatusTransition)
	}
	if err := insertDriverMobileOrderEvent(ctx, transaction, transition, updated); err != nil {
		return driverapp.CurrentOrder{}, err
	}
	if toStatus == domain.OrderStatusCancelled || toStatus == domain.OrderStatusCompleted {
		if err := markDriverOnlineInTx(ctx, transaction, *updated.DriverID); err != nil {
			return driverapp.CurrentOrder{}, err
		}
	}

	order, err := selectDriverCurrentOrderByID(ctx, transaction, userID, orderID)
	if err != nil {
		return driverapp.CurrentOrder{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return driverapp.CurrentOrder{}, fmt.Errorf("commit driver order transition: %w", err)
	}
	return order, nil
}

func (repository *PostgresDriverMobileRepository) ListRoutePointsByUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) ([]driverapp.RoutePoint, error) {
	var orderExists bool
	if err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM orders o
			JOIN drivers d ON d.id = o.driver_id
			WHERE o.id = $1
			  AND d.user_id = $2
			  AND o.deleted_at IS NULL
			  AND d.deleted_at IS NULL
		)`, orderID, userID).Scan(&orderExists); err != nil {
		return nil, fmt.Errorf("check driver order before route points: %w", err)
	}
	if !orderExists {
		return nil, driverapp.ErrCurrentOrderNotFound
	}

	rows, err := repository.pool.Query(ctx, `
		SELECT rp.id,
		       ST_Y(rp.location::geometry) AS latitude,
		       ST_X(rp.location::geometry) AS longitude,
		       rp.heading,
		       rp.speed_mps,
		       rp.accuracy_meters,
		       rp.recorded_at
		FROM order_route_points rp
		JOIN orders o ON o.id = rp.order_id
		JOIN drivers d ON d.id = rp.driver_id AND d.id = o.driver_id
		WHERE rp.order_id = $1
		  AND d.user_id = $2
		  AND o.deleted_at IS NULL
		  AND d.deleted_at IS NULL
		ORDER BY rp.recorded_at ASC`, orderID, userID)
	if err != nil {
		return nil, fmt.Errorf("select driver order route points: %w", err)
	}
	defer rows.Close()

	points := make([]driverapp.RoutePoint, 0)
	for rows.Next() {
		var point driverapp.RoutePoint
		var latitude float64
		var longitude float64
		var heading pgtype.Float8
		var speedMPS pgtype.Float8
		var accuracyMeters pgtype.Float8
		if err := rows.Scan(
			&point.ID,
			&latitude,
			&longitude,
			&heading,
			&speedMPS,
			&accuracyMeters,
			&point.RecordedAt,
		); err != nil {
			return nil, fmt.Errorf("scan driver order route point: %w", err)
		}
		point.Location = domain.Coordinates{Latitude: latitude, Longitude: longitude}
		point.Heading = nullableFloat64(heading)
		point.SpeedMPS = nullableFloat64(speedMPS)
		point.AccuracyMeters = nullableFloat64(accuracyMeters)
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate driver order route points: %w", err)
	}
	return points, nil
}

func (repository *PostgresDriverMobileRepository) GetOrderRouteUploadAccess(ctx context.Context, orderID uuid.UUID) (driverapp.OrderRouteUploadAccess, error) {
	var access driverapp.OrderRouteUploadAccess
	var driverID pgtype.UUID
	var driverUserID pgtype.UUID

	if err := repository.pool.QueryRow(ctx, `
		SELECT o.id,
		       o.status,
		       d.id,
		       d.user_id
		FROM orders o
		LEFT JOIN drivers d ON d.id = o.driver_id AND d.deleted_at IS NULL
		WHERE o.id = $1
		  AND o.deleted_at IS NULL`, orderID).Scan(
		&access.OrderID,
		&access.Status,
		&driverID,
		&driverUserID,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return driverapp.OrderRouteUploadAccess{}, driverapp.ErrCurrentOrderNotFound
		}
		return driverapp.OrderRouteUploadAccess{}, fmt.Errorf("select order route upload access: %w", err)
	}

	if !driverID.Valid || !driverUserID.Valid {
		return driverapp.OrderRouteUploadAccess{}, driverapp.ErrOrderAccessDenied
	}

	parsedDriverID, err := uuid.FromBytes(driverID.Bytes[:])
	if err != nil {
		return driverapp.OrderRouteUploadAccess{}, fmt.Errorf("parse route upload driver id: %w", err)
	}
	parsedDriverUserID, err := uuid.FromBytes(driverUserID.Bytes[:])
	if err != nil {
		return driverapp.OrderRouteUploadAccess{}, fmt.Errorf("parse route upload driver user id: %w", err)
	}

	access.DriverID = parsedDriverID
	access.DriverUserID = parsedDriverUserID
	return access, nil
}

func (repository *PostgresDriverMobileRepository) AppendRoutePointByUserID(ctx context.Context, userID uuid.UUID, update geoservice.DriverLocationUpdate) error {
	commandTag, err := repository.pool.Exec(ctx, `
		INSERT INTO order_route_points (
			order_id, driver_id, location, heading, speed_mps, accuracy_meters, recorded_at
		)
		SELECT o.id, d.id, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		       $4, $5, $6, $7
		FROM drivers d
		JOIN orders o ON o.driver_id = d.id
		WHERE d.user_id = $1
		  AND d.deleted_at IS NULL
		  AND o.deleted_at IS NULL
		  AND o.status = 'in_progress'
		ORDER BY o.updated_at DESC
		LIMIT 1`,
		userID,
		update.Location.Latitude,
		update.Location.Longitude,
		update.Heading,
		update.SpeedMPS,
		update.AccuracyMeters,
		update.RecordedAt,
	)
	if err != nil {
		return fmt.Errorf("append driver route point: %w", err)
	}
	_ = commandTag
	return nil
}

func (repository *PostgresDriverMobileRepository) AppendOrderRoutePoints(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID, points []driverapp.OrderRouteAppendPoint) (driverapp.AppendOrderRoutePointsResult, error) {
	if len(points) == 0 {
		return driverapp.AppendOrderRoutePointsResult{OrderID: orderID}, nil
	}

	payload := make([]map[string]any, 0, len(points))
	for _, point := range points {
		payload = append(payload, map[string]any{
			"latitude":        point.Location.Latitude,
			"longitude":       point.Location.Longitude,
			"heading":         point.Heading,
			"speed_mps":       point.SpeedMPS,
			"accuracy_meters": point.AccuracyMeters,
			"recorded_at":     point.RecordedAt.UTC(),
		})
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return driverapp.AppendOrderRoutePointsResult{}, fmt.Errorf("marshal order route points payload: %w", err)
	}

	var acceptedPoints int
	if err := repository.pool.QueryRow(ctx, `
		WITH input AS (
			SELECT latitude,
			       longitude,
			       heading,
			       speed_mps,
			       accuracy_meters,
			       recorded_at
			FROM jsonb_to_recordset($3::jsonb) AS point(
				latitude double precision,
				longitude double precision,
				heading smallint,
				speed_mps numeric,
				accuracy_meters numeric,
				recorded_at timestamptz
			)
		),
		inserted AS (
			INSERT INTO order_route_points (
				order_id, driver_id, location, heading, speed_mps, accuracy_meters, recorded_at
			)
			SELECT $1,
			       $2,
			       ST_SetSRID(ST_MakePoint(input.longitude, input.latitude), 4326)::geography,
			       input.heading,
			       input.speed_mps,
			       input.accuracy_meters,
			       input.recorded_at
			FROM input
			ON CONFLICT DO NOTHING
			RETURNING 1
		)
		SELECT COUNT(*)::int
		FROM inserted`,
		orderID,
		driverID,
		payloadBytes,
	).Scan(&acceptedPoints); err != nil {
		return driverapp.AppendOrderRoutePointsResult{}, fmt.Errorf("bulk insert order route points: %w", err)
	}

	return driverapp.AppendOrderRoutePointsResult{
		OrderID:        orderID,
		AcceptedPoints: acceptedPoints,
		IgnoredPoints:  len(points) - acceptedPoints,
	}, nil
}

func nullableFloat64(value pgtype.Float8) *float64 {
	if !value.Valid {
		return nil
	}
	result := value.Float64
	return &result
}

func (repository *PostgresDriverMobileRepository) markOnline(ctx context.Context, userID uuid.UUID) (driverapp.Profile, error) {
	profile, err := scanDriverMobileProfile(repository.pool.QueryRow(ctx, `
		WITH updated_driver AS (
			UPDATE drivers d
		    SET status = 'online'
		    FROM taxi_parks tp
		    WHERE d.user_id = $1
		      AND tp.id = d.taxi_park_id
		      AND d.status IN ('offline', 'paused', 'online')
		      AND d.taxi_park_id IS NOT NULL
		      AND d.is_verified = true
		      AND d.verification_status = 'verified'
		      AND d.deleted_at IS NULL
		      AND tp.deleted_at IS NULL
		      AND EXISTS (
		      	SELECT 1
		      	FROM taxi_park_settings tps
		      	WHERE tps.taxi_park_id = tp.id
		      	  AND tps.is_active = true
		      	UNION ALL
		      	SELECT 1
		      	WHERE NOT EXISTS (
		      		SELECT 1
		      		FROM taxi_park_settings tps
		      		WHERE tps.taxi_park_id = tp.id
		      	)
		      )
		      AND EXISTS (
		      	SELECT 1
		      	FROM cars c
		      	LEFT JOIN car_driver_assignments cda ON cda.car_id = c.id
		      	WHERE c.taxi_park_id = d.taxi_park_id
		      	  AND (c.driver_id = d.id OR cda.driver_id = d.id)
		      	  AND c.verification_status = 'verified'
		      	  AND c.is_active = true
		      	  AND c.deleted_at IS NULL
		      	  AND COALESCE(c.permit_expires_at, current_date + interval '1 day') >= current_date
		      	  AND COALESCE(c.osago_expires_at, current_date + interval '1 day') >= current_date
		      )
		    RETURNING d.id
		)
		`+driverMobileProfileSelect+`
		WHERE d.user_id = $1
		  AND d.id IN (SELECT id FROM updated_driver)`, userID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			exists, existsErr := repository.driverExists(ctx, userID)
			if existsErr != nil {
				return driverapp.Profile{}, existsErr
			}
			if exists {
				return driverapp.Profile{}, driverapp.ErrDriverNotAvailable
			}
		}
		return driverapp.Profile{}, mapDriverMobileScanError("mark driver online", err)
	}
	return profile, nil
}

func (repository *PostgresDriverMobileRepository) markOffline(ctx context.Context, userID uuid.UUID) (driverapp.Profile, error) {
	profile, err := scanDriverMobileProfile(repository.pool.QueryRow(ctx, `
		WITH updated_driver AS (
			UPDATE drivers
			SET status = 'offline'
			WHERE user_id = $1
			  AND status IN ('online', 'paused', 'offline')
			  AND deleted_at IS NULL
			RETURNING id
		)
		`+driverMobileProfileSelect+`
		WHERE d.user_id = $1
		  AND d.id IN (SELECT id FROM updated_driver)`, userID))
	if err != nil {
		return driverapp.Profile{}, mapDriverMobileScanError("mark driver offline", err)
	}
	return profile, nil
}

func (repository *PostgresDriverMobileRepository) driverExists(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	if err := repository.pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM drivers WHERE user_id = $1 AND deleted_at IS NULL)`, userID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check driver exists: %w", err)
	}
	return exists, nil
}

const driverMobileProfileSelect = `
		SELECT d.id,
		       d.user_id,
		       d.city_id,
		       u.phone,
		       COALESCE(u.first_name, ''),
		       COALESCE(u.last_name, ''),
		       COALESCE(u.profile_photo_url, ''),
		       d.status,
		       d.rating::float8,
		       d.ratings_count,
		       COALESCE(d.license_number, ''),
		       d.is_verified,
		       d.verification_status,
		       d.taxi_park_id,
		       tps.is_active,
		       d.has_no_taxi_work_restrictions,
		       d.federal_law_580_compliant,
		       d.regional_requirements_compliant,
		       d.medical_check_passed,
		       d.pretrip_control_required,
		       d.pretrip_control_passed,
		       d.no_transport_ban,
		       profile_car.id,
		       COALESCE(profile_car.brand, ''),
		       COALESCE(profile_car.model, ''),
		       COALESCE(profile_car.year, 0),
		       COALESCE(profile_car.plate_number, ''),
		       COALESCE(profile_car.color, ''),
		       COALESCE(profile_car.car_class, ''),
		       profile_car.verification_status,
		       COALESCE(profile_car.is_active, false),
		       profile_car.osago_expires_at,
		       profile_car.permit_expires_at
		FROM drivers d
		JOIN users u ON u.id = d.user_id
		LEFT JOIN taxi_park_settings tps ON tps.taxi_park_id = d.taxi_park_id
		LEFT JOIN LATERAL (
			SELECT c.id,
			       c.brand,
			       c.model,
			       c.year,
			       c.plate_number,
			       c.color,
			       c.car_class,
			       c.verification_status,
			       c.is_active,
			       c.osago_expires_at,
			       c.permit_expires_at
			FROM cars c
			LEFT JOIN car_driver_assignments cda ON cda.car_id = c.id AND cda.driver_id = d.id
			WHERE c.deleted_at IS NULL
			  AND (c.driver_id = d.id OR cda.driver_id IS NOT NULL)
			ORDER BY
				(c.verification_status = 'verified') DESC,
				c.is_active DESC,
				(c.driver_id = d.id) DESC,
				c.created_at DESC
			LIMIT 1
		) profile_car ON true`

func driverMobileProfileQuery(whereClause string) string {
	return driverMobileProfileSelect + "\n" + whereClause
}

func scanDriverMobileProfile(row pgx.Row) (driverapp.Profile, error) {
	var profile driverapp.Profile
	var taxiParkID pgtype.UUID
	var taxiParkIsActive pgtype.Bool
	var carID pgtype.UUID
	var carVerificationStatus pgtype.Text
	var osagoExpiresAt pgtype.Date
	var permitExpiresAt pgtype.Date
	var carBrand string
	var carModel string
	var carYear int
	var carPlateNumber string
	var carColor string
	var carClass string
	var carIsActive bool
	if err := row.Scan(
		&profile.DriverID,
		&profile.UserID,
		&profile.CityID,
		&profile.Phone,
		&profile.FirstName,
		&profile.LastName,
		&profile.PhotoURL,
		&profile.Status,
		&profile.Rating,
		&profile.RatingsCount,
		&profile.LicenseNumber,
		&profile.IsVerified,
		&profile.VerificationStatus,
		&taxiParkID,
		&taxiParkIsActive,
		&profile.HasNoTaxiWorkRestrictions,
		&profile.FederalLaw580Compliant,
		&profile.RegionalRequirementsCompliant,
		&profile.MedicalCheckPassed,
		&profile.PretripControlRequired,
		&profile.PretripControlPassed,
		&profile.NoTransportBan,
		&carID,
		&carBrand,
		&carModel,
		&carYear,
		&carPlateNumber,
		&carColor,
		&carClass,
		&carVerificationStatus,
		&carIsActive,
		&osagoExpiresAt,
		&permitExpiresAt,
	); err != nil {
		return driverapp.Profile{}, err
	}
	if taxiParkID.Valid {
		value, err := uuid.FromBytes(taxiParkID.Bytes[:])
		if err != nil {
			return driverapp.Profile{}, err
		}
		profile.TaxiParkID = &value
	}
	if taxiParkIsActive.Valid {
		value := taxiParkIsActive.Bool
		profile.TaxiParkIsActive = &value
	}
	if carID.Valid {
		value, err := uuid.FromBytes(carID.Bytes[:])
		if err != nil {
			return driverapp.Profile{}, err
		}
		profile.Car = &driverapp.ProfileCar{
			ID:                 value,
			Brand:              carBrand,
			Model:              carModel,
			Year:               carYear,
			PlateNumber:        carPlateNumber,
			Color:              carColor,
			CarClass:           carClass,
			VerificationStatus: domain.VerificationLifecycleStatus(carVerificationStatus.String),
			IsActive:           carIsActive,
			OSAGOExpiresAt:     datePtr(osagoExpiresAt),
			PermitExpiresAt:    datePtr(permitExpiresAt),
		}
	}
	return profile, nil
}

func scanDriverCurrentOrder(row pgx.Row) (driverapp.CurrentOrder, error) {
	var order driverapp.CurrentOrder
	var destinationLatitude pgtype.Float8
	var destinationLongitude pgtype.Float8
	var priceAmount pgtype.Int8
	var pickupLatitude float64
	var pickupLongitude float64

	if err := row.Scan(
		&order.OrderID,
		&order.DriverID,
		&order.PassengerID,
		&order.PassengerName,
		&order.PassengerPhone,
		&order.PassengerPhotoURL,
		&order.PassengerRating,
		&order.PassengerRatingsCount,
		&order.PickupAddress,
		&pickupLatitude,
		&pickupLongitude,
		&order.DestinationAddress,
		&destinationLatitude,
		&destinationLongitude,
		&order.Status,
		&priceAmount,
		&order.Comment,
		&order.Version,
		&order.CreatedAt,
	); err != nil {
		return driverapp.CurrentOrder{}, err
	}

	order.PickupLocation = domain.Coordinates{
		Latitude:  pickupLatitude,
		Longitude: pickupLongitude,
	}
	if destinationLatitude.Valid && destinationLongitude.Valid {
		order.DestinationLocation = &domain.Coordinates{
			Latitude:  destinationLatitude.Float64,
			Longitude: destinationLongitude.Float64,
		}
	}
	if priceAmount.Valid {
		order.Price = &domain.Money{
			Amount:   priceAmount.Int64,
			Currency: "RUB",
		}
	}

	return order, nil
}

func (repository *PostgresDriverMobileRepository) carAssignments(ctx context.Context, carID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := repository.pool.Query(ctx, `SELECT driver_id FROM car_driver_assignments WHERE car_id = $1`, carID)
	if err != nil {
		return nil, fmt.Errorf("select driver car assignments: %w", err)
	}
	defer rows.Close()

	driverIDs := make([]uuid.UUID, 0)
	for rows.Next() {
		var driverID uuid.UUID
		if err := rows.Scan(&driverID); err != nil {
			return nil, fmt.Errorf("scan driver car assignment: %w", err)
		}
		driverIDs = append(driverIDs, driverID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate driver car assignments: %w", err)
	}
	return driverIDs, nil
}

func mapDriverMobileScanError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return driverapp.ErrDriverNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}

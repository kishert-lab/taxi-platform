package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	passengerapp "github.com/kishert-lab/taxi-platform/internal/passenger"
)

type PostgresPassengerOrderRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPassengerOrderRepository(pool *pgxpool.Pool) *PostgresPassengerOrderRepository {
	return &PostgresPassengerOrderRepository{pool: pool}
}

func (repository *PostgresPassengerOrderRepository) ListActiveCarClasses(ctx context.Context) ([]domain.CarClass, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT `+carClassSelectColumns+`
		FROM car_classes
		WHERE is_active = true
		  AND deleted_at IS NULL
		ORDER BY sort_order ASC, created_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("select active car classes: %w", err)
	}
	defer rows.Close()

	items := make([]domain.CarClass, 0)
	for rows.Next() {
		item, err := scanCarClass(rows)
		if err != nil {
			return nil, fmt.Errorf("scan active car class: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active car classes: %w", err)
	}
	return items, nil
}

func (repository *PostgresPassengerOrderRepository) GetActiveCarClassByID(ctx context.Context, carClassID uuid.UUID) (domain.CarClass, error) {
	item, err := scanCarClass(repository.pool.QueryRow(ctx, `
		SELECT `+carClassSelectColumns+`
		FROM car_classes
		WHERE id = $1
		  AND is_active = true
		  AND deleted_at IS NULL`, carClassID))
	if err != nil {
		return domain.CarClass{}, fmt.Errorf("select active car class by id: %w", err)
	}
	return item, nil
}

func (repository *PostgresPassengerOrderRepository) EstimateRoute(ctx context.Context, pickup geodomain.Coordinates, destination geodomain.Coordinates) (float64, error) {
	var distanceMeters float64
	if err := repository.pool.QueryRow(ctx, `
		SELECT ST_Distance(
			ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
			ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography
		)`, pickup.Latitude, pickup.Longitude, destination.Latitude, destination.Longitude).Scan(&distanceMeters); err != nil {
		return 0, fmt.Errorf("estimate route distance: %w", err)
	}
	return distanceMeters / 1000.0, nil
}

func (repository *PostgresPassengerOrderRepository) CreatePassengerOrder(ctx context.Context, record passengerapp.CreateOrderRecord) (passengerapp.PassengerOrderDetails, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("begin passenger order create transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	metadataBytes, err := json.Marshal(map[string]any{
		"created_by_role": "passenger",
	})
	if err != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("marshal passenger order metadata: %w", err)
	}

	var insertedOrderID uuid.UUID
	err = transaction.QueryRow(ctx, `
		INSERT INTO orders (
			passenger_id,
			city_id,
			status,
			pickup_address,
			pickup_entrance,
			pickup_comment,
			pickup_location,
			destination_address,
			destination_location,
			estimated_price,
			payment_method,
			passenger_comment,
			passenger_location_sharing_enabled,
			car_class_id,
			dispatch_attempt,
			version,
			metadata
		)
		VALUES (
			$1,
			$2,
			'searching',
			$3,
			NULLIF($4, ''),
			NULLIF($5, ''),
			ST_SetSRID(ST_MakePoint($7, $6), 4326)::geography,
			$8,
			ST_SetSRID(ST_MakePoint($10, $9), 4326)::geography,
			$11::bigint::numeric / 100,
			$12,
			NULLIF($13, ''),
			$14,
			$15,
			0,
			1,
			$16::jsonb
		)
		RETURNING id`,
		record.PassengerID,
		record.CityID,
		record.PickupAddress,
		record.PickupEntrance,
		record.PickupComment,
		record.PickupLocation.Latitude,
		record.PickupLocation.Longitude,
		record.DestinationAddress,
		record.DestinationLocation.Latitude,
		record.DestinationLocation.Longitude,
		record.EstimatedPrice.Amount,
		record.PaymentMethod,
		record.PassengerComment,
		record.PassengerLocationSharingEnabled,
		record.CarClassID,
		string(metadataBytes),
	).Scan(&insertedOrderID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return passengerapp.PassengerOrderDetails{}, pgx.ErrNoRows
		}
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("insert passenger order: %w", err)
	}

	if err := insertPassengerOrderEvents(ctx, transaction, insertedOrderID, record.PassengerID); err != nil {
		return passengerapp.PassengerOrderDetails{}, err
	}

	orderDetails, err := repository.getPassengerOrderByID(ctx, transaction, record.PassengerID, insertedOrderID)
	if err != nil {
		return passengerapp.PassengerOrderDetails{}, err
	}

	if err := transaction.Commit(ctx); err != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("commit passenger order create transaction: %w", err)
	}

	return orderDetails, nil
}

func (repository *PostgresPassengerOrderRepository) GetCurrentPassengerOrder(ctx context.Context, passengerID uuid.UUID) (passengerapp.PassengerOrderDetails, error) {
	return repository.getPassengerOrderByID(ctx, repository.pool, passengerID, uuid.Nil)
}

func (repository *PostgresPassengerOrderRepository) ListPassengerOrderHistory(ctx context.Context, passengerID uuid.UUID, limit int) ([]passengerapp.PassengerOrderDetails, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := repository.pool.Query(ctx, passengerOrderSelectQuery+`
		WHERE o.passenger_id = $1
		  AND o.deleted_at IS NULL
		  AND o.status IN ('completed', 'cancelled', 'failed')
		ORDER BY COALESCE(o.completed_at, o.cancelled_at, o.updated_at, o.created_at) DESC
		LIMIT $2`, passengerID, limit)
	if err != nil {
		return nil, fmt.Errorf("select passenger order history: %w", err)
	}
	defer rows.Close()

	items := make([]passengerapp.PassengerOrderDetails, 0, limit)
	for rows.Next() {
		item, err := scanPassengerOrderDetails(rows)
		if err != nil {
			return nil, fmt.Errorf("scan passenger order history: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate passenger order history: %w", err)
	}
	return items, nil
}

func (repository *PostgresPassengerOrderRepository) GetPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (passengerapp.PassengerOrderDetails, error) {
	return repository.getPassengerOrderByID(ctx, repository.pool, passengerID, orderID)
}

func (repository *PostgresPassengerOrderRepository) CancelPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, reason string, cancelledAt time.Time) (passengerapp.PassengerOrderDetails, error) {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("begin passenger order cancel transaction: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	var updatedOrderID uuid.UUID
	err = transaction.QueryRow(ctx, `
		UPDATE orders
		SET status = 'cancelled',
		    cancelled_at = $4,
		    cancellation_reason = NULLIF($3, ''),
		    version = version + 1
		WHERE id = $1
		  AND passenger_id = $2
		  AND status IN ('searching', 'driver_assigned', 'driver_arriving', 'driver_waiting')
		  AND deleted_at IS NULL
		RETURNING id`,
		orderID,
		passengerID,
		strings.TrimSpace(reason),
		cancelledAt,
	).Scan(&updatedOrderID)
	if err != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("update passenger order cancelled: %w", err)
	}

	payloadBytes, marshalErr := json.Marshal(map[string]any{
		"order_id":            updatedOrderID,
		"status":              domain.OrderStatusCancelled,
		"cancelled_by":        "passenger",
		"cancellation_reason": strings.TrimSpace(reason),
		"occurred_at":         cancelledAt,
	})
	if marshalErr != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("marshal passenger cancel order event: %w", marshalErr)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO order_events (order_id, actor_user_id, event_type, payload, created_at)
		VALUES ($1, $2, 'order.cancelled', $3::jsonb, $4)`,
		updatedOrderID,
		nil,
		string(payloadBytes),
		cancelledAt,
	); err != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("insert passenger cancel order event: %w", err)
	}

	orderDetails, err := repository.getPassengerOrderByID(ctx, transaction, passengerID, updatedOrderID)
	if err != nil {
		return passengerapp.PassengerOrderDetails{}, err
	}
	if err := transaction.Commit(ctx); err != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("commit passenger order cancel transaction: %w", err)
	}
	return orderDetails, nil
}

func (repository *PostgresPassengerOrderRepository) getPassengerOrderByID(ctx context.Context, queryer queryRower, passengerID uuid.UUID, orderID uuid.UUID) (passengerapp.PassengerOrderDetails, error) {
	query := passengerOrderSelectQuery + `
		WHERE o.passenger_id = $1
		  AND o.deleted_at IS NULL`
	args := []any{passengerID}

	if orderID == uuid.Nil {
		query += `
		  AND o.status IN ('created', 'searching', 'driver_assigned', 'driver_arriving', 'driver_waiting', 'in_progress')
		ORDER BY o.created_at DESC
		LIMIT 1`
	} else {
		query += `
		  AND o.id = $2`
		args = append(args, orderID)
	}

	item, err := scanPassengerOrderDetails(queryer.QueryRow(ctx, query, args...))
	if err != nil {
		return passengerapp.PassengerOrderDetails{}, fmt.Errorf("select passenger order details: %w", err)
	}
	return item, nil
}

type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func insertPassengerOrderEvents(ctx context.Context, transaction pgx.Tx, orderID uuid.UUID, _ uuid.UUID) error {
	createdPayload, err := json.Marshal(map[string]any{
		"order_id": orderID,
		"status":   domain.OrderStatusCreated,
	})
	if err != nil {
		return fmt.Errorf("marshal passenger order created event: %w", err)
	}
	searchingPayload, err := json.Marshal(map[string]any{
		"order_id": orderID,
		"status":   domain.OrderStatusSearching,
		"version":  1,
	})
	if err != nil {
		return fmt.Errorf("marshal passenger order searching event: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO order_events (order_id, actor_user_id, event_type, payload)
		VALUES ($1, $2, 'order.created', $3::jsonb),
		       ($1, $2, 'order.searching', $4::jsonb)`,
		orderID,
		nil,
		string(createdPayload),
		string(searchingPayload),
	); err != nil {
		return fmt.Errorf("insert passenger order events: %w", err)
	}
	return nil
}

const carClassSelectColumns = `
	id,
	code,
	name,
	description,
	(base_price * 100)::bigint,
	(price_per_km * 100)::bigint,
	(price_per_minute * 100)::bigint,
	(minimum_price * 100)::bigint,
	is_active,
	sort_order,
	created_at,
	updated_at`

func scanCarClass(row pgx.Row) (domain.CarClass, error) {
	var item domain.CarClass
	if err := row.Scan(
		&item.ID,
		&item.Code,
		&item.Name,
		&item.Description,
		&item.BasePrice.Amount,
		&item.PricePerKM.Amount,
		&item.PricePerMinute.Amount,
		&item.MinimumPrice.Amount,
		&item.IsActive,
		&item.SortOrder,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return domain.CarClass{}, err
	}
	item.BasePrice.Currency = "RUB"
	item.PricePerKM.Currency = "RUB"
	item.PricePerMinute.Currency = "RUB"
	item.MinimumPrice.Currency = "RUB"
	return item, nil
}

const passengerOrderSelectQuery = `
	SELECT
		o.id,
		o.passenger_id,
		o.driver_id,
		o.car_id,
		o.park_id,
		o.city_id,
		o.tariff_id,
		o.car_class_id,
		o.status,
		o.order_type,
		o.pickup_address,
		COALESCE(o.pickup_entrance, '') AS pickup_entrance,
		COALESCE(o.pickup_comment, '') AS pickup_comment,
		ST_Y(o.pickup_location::geometry) AS pickup_latitude,
		ST_X(o.pickup_location::geometry) AS pickup_longitude,
		COALESCE(o.destination_address, '') AS destination_address,
		CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_Y(o.destination_location::geometry) END AS destination_latitude,
		CASE WHEN o.destination_location IS NULL THEN NULL ELSE ST_X(o.destination_location::geometry) END AS destination_longitude,
		o.requested_at,
		o.accepted_at,
		o.started_at,
		o.completed_at,
		o.cancelled_at,
		COALESCE(o.cancellation_reason, '') AS cancellation_reason,
		CASE WHEN o.estimated_price IS NULL THEN NULL ELSE (o.estimated_price * 100)::bigint END AS estimated_price_amount,
		CASE WHEN o.final_price IS NULL THEN NULL ELSE (o.final_price * 100)::bigint END AS final_price_amount,
		o.payment_method,
		COALESCE(o.passenger_comment, '') AS passenger_comment,
		o.passenger_location_sharing_enabled,
		o.dispatch_attempt,
		o.version,
		o.created_at,
		o.updated_at,
		cc.id,
		COALESCE(cc.code, ''),
		COALESCE(cc.name, ''),
		COALESCE(cc.description, ''),
		CASE WHEN cc.id IS NULL THEN NULL ELSE (cc.base_price * 100)::bigint END AS class_base_price_amount,
		CASE WHEN cc.id IS NULL THEN NULL ELSE (cc.price_per_km * 100)::bigint END AS class_price_per_km_amount,
		CASE WHEN cc.id IS NULL THEN NULL ELSE (cc.price_per_minute * 100)::bigint END AS class_price_per_minute_amount,
		CASE WHEN cc.id IS NULL THEN NULL ELSE (cc.minimum_price * 100)::bigint END AS class_minimum_price_amount,
		COALESCE(cc.is_active, false),
		COALESCE(cc.sort_order, 0),
		cc.created_at,
		cc.updated_at,
		d.id,
		COALESCE(d_user.first_name || CASE WHEN COALESCE(d_user.last_name, '') <> '' THEN ' ' || d_user.last_name ELSE '' END, '') AS driver_name,
		COALESCE(d_user.phone, '') AS driver_phone,
		COALESCE(d_user.profile_photo_url, '') AS driver_avatar_url,
		COALESCE(d.rating::float8, 0),
		COALESCE(d.ratings_count, 0),
		car.id,
		COALESCE(car.brand, ''),
		COALESCE(car.model, ''),
		COALESCE(car.color, ''),
		COALESCE(car.plate_number, ''),
		COALESCE(car.car_class, '')
	FROM orders o
	LEFT JOIN car_classes cc ON cc.id = o.car_class_id
	LEFT JOIN drivers d ON d.id = o.driver_id AND d.deleted_at IS NULL
	LEFT JOIN users d_user ON d_user.id = d.user_id AND d_user.deleted_at IS NULL
	LEFT JOIN cars car ON car.id = o.car_id AND car.deleted_at IS NULL`

func scanPassengerOrderDetails(row pgx.Row) (passengerapp.PassengerOrderDetails, error) {
	var details passengerapp.PassengerOrderDetails
	var order domain.Order
	var driverID pgtype.UUID
	var carID pgtype.UUID
	var parkID pgtype.UUID
	var tariffID pgtype.UUID
	var classID pgtype.UUID
	var destinationLatitude pgtype.Float8
	var destinationLongitude pgtype.Float8
	var acceptedAt pgtype.Timestamptz
	var startedAt pgtype.Timestamptz
	var completedAt pgtype.Timestamptz
	var cancelledAt pgtype.Timestamptz
	var estimatedPriceAmount pgtype.Int8
	var finalPriceAmount pgtype.Int8
	var classBasePriceAmount pgtype.Int8
	var classPricePerKMAmount pgtype.Int8
	var classPricePerMinuteAmount pgtype.Int8
	var classMinimumPriceAmount pgtype.Int8
	var carClassID pgtype.UUID
	var carClassActive bool
	var carClassSortOrder int
	var carClassCreatedAt pgtype.Timestamptz
	var carClassUpdatedAt pgtype.Timestamptz
	var assignedDriverID pgtype.UUID
	var assignedCarID pgtype.UUID
	var pickupLatitude float64
	var pickupLongitude float64
	var driverName string
	var driverPhone string
	var driverAvatarURL string
	var driverRating float64
	var driverRatingsCount int
	var assignedCarBrand string
	var assignedCarModel string
	var assignedCarColor string
	var assignedCarPlateNumber string
	var assignedCarClass string
	var carClassCode string
	var carClassName string
	var carClassDescription string

	if err := row.Scan(
		&order.ID,
		&order.PassengerID,
		&driverID,
		&carID,
		&parkID,
		&order.CityID,
		&tariffID,
		&classID,
		&order.Status,
		&order.OrderType,
		&order.PickupAddress,
		&order.PickupEntrance,
		&order.PickupComment,
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
		&estimatedPriceAmount,
		&finalPriceAmount,
		&order.PaymentMethod,
		&order.PassengerComment,
		&order.PassengerLocationSharingEnabled,
		&order.DispatchAttempt,
		&order.Version,
		&order.CreatedAt,
		&order.UpdatedAt,
		&carClassID,
		&carClassCode,
		&carClassName,
		&carClassDescription,
		&classBasePriceAmount,
		&classPricePerKMAmount,
		&classPricePerMinuteAmount,
		&classMinimumPriceAmount,
		&carClassActive,
		&carClassSortOrder,
		&carClassCreatedAt,
		&carClassUpdatedAt,
		&assignedDriverID,
		&driverName,
		&driverPhone,
		&driverAvatarURL,
		&driverRating,
		&driverRatingsCount,
		&assignedCarID,
		&assignedCarBrand,
		&assignedCarModel,
		&assignedCarColor,
		&assignedCarPlateNumber,
		&assignedCarClass,
	); err != nil {
		return passengerapp.PassengerOrderDetails{}, err
	}

	order.PickupLocation = domain.Coordinates{Latitude: pickupLatitude, Longitude: pickupLongitude}
	if destinationLatitude.Valid && destinationLongitude.Valid {
		order.DestinationLocation = &domain.Coordinates{
			Latitude:  destinationLatitude.Float64,
			Longitude: destinationLongitude.Float64,
		}
	}
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
	if tariffID.Valid {
		value := uuid.UUID(tariffID.Bytes)
		order.TariffID = &value
	}
	if classID.Valid {
		value := uuid.UUID(classID.Bytes)
		order.CarClassID = &value
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
	if estimatedPriceAmount.Valid {
		order.EstimatedPrice = &domain.Money{Amount: estimatedPriceAmount.Int64, Currency: "RUB"}
	}
	if finalPriceAmount.Valid {
		order.FinalPrice = &domain.Money{Amount: finalPriceAmount.Int64, Currency: "RUB"}
	}

	if carClassID.Valid {
		classUUID := uuid.UUID(carClassID.Bytes)
		carClass := &domain.CarClass{
			ID:          classUUID,
			Code:        carClassCode,
			Name:        carClassName,
			Description: carClassDescription,
			IsActive:    carClassActive,
			SortOrder:   carClassSortOrder,
		}
		if classBasePriceAmount.Valid {
			carClass.BasePrice = domain.Money{Amount: classBasePriceAmount.Int64, Currency: "RUB"}
		}
		if classPricePerKMAmount.Valid {
			carClass.PricePerKM = domain.Money{Amount: classPricePerKMAmount.Int64, Currency: "RUB"}
		}
		if classPricePerMinuteAmount.Valid {
			carClass.PricePerMinute = domain.Money{Amount: classPricePerMinuteAmount.Int64, Currency: "RUB"}
		}
		if classMinimumPriceAmount.Valid {
			carClass.MinimumPrice = domain.Money{Amount: classMinimumPriceAmount.Int64, Currency: "RUB"}
		}
		if carClassCreatedAt.Valid {
			carClass.CreatedAt = carClassCreatedAt.Time
		}
		if carClassUpdatedAt.Valid {
			carClass.UpdatedAt = carClassUpdatedAt.Time
		}
		details.CarClass = carClass
	}

	if assignedDriverID.Valid {
		driverUUID := uuid.UUID(assignedDriverID.Bytes)
		details.Driver = &passengerapp.PassengerAssignedDriver{
			ID:           driverUUID,
			Name:         strings.TrimSpace(driverName),
			Phone:        driverPhone,
			AvatarURL:    driverAvatarURL,
			Rating:       driverRating,
			RatingsCount: driverRatingsCount,
		}
	}

	if assignedCarID.Valid {
		carUUID := uuid.UUID(assignedCarID.Bytes)
		details.Car = &passengerapp.PassengerAssignedCar{
			ID:          carUUID,
			Brand:       assignedCarBrand,
			Model:       assignedCarModel,
			Color:       assignedCarColor,
			PlateNumber: assignedCarPlateNumber,
			CarClass:    assignedCarClass,
		}
	}

	details.Order = order
	return details, nil
}

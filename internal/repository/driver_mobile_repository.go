package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	driverapp "github.com/kishert-lab/taxi-platform/internal/driver"
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

func (repository *PostgresDriverMobileRepository) GetCurrentOrderByUserID(ctx context.Context, userID uuid.UUID) (driverapp.CurrentOrder, error) {
	order, err := scanDriverCurrentOrder(repository.pool.QueryRow(ctx, `
		SELECT o.id,
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
			  AND d.verification_status NOT IN ('blocked', 'archived', 'rejected')
			  AND d.deleted_at IS NULL
			  AND tp.deleted_at IS NULL
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
		       d.is_verified
		FROM drivers d
		JOIN users u ON u.id = d.user_id`

func driverMobileProfileQuery(whereClause string) string {
	return driverMobileProfileSelect + "\n" + whereClause
}

func scanDriverMobileProfile(row pgx.Row) (driverapp.Profile, error) {
	var profile driverapp.Profile
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
	); err != nil {
		return driverapp.Profile{}, err
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

func mapDriverMobileScanError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return driverapp.ErrDriverNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}

package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/geo"
)

type PostgresDriverLocationRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresDriverLocationRepository(pool *pgxpool.Pool) *PostgresDriverLocationRepository {
	return &PostgresDriverLocationRepository{pool: pool}
}

func (repository *PostgresDriverLocationRepository) UpdateDriverLocation(ctx context.Context, update geo.DriverLocationUpdate) error {
	const query = `
		INSERT INTO driver_locations (
			driver_id,
			city_id,
			location,
			heading,
			speed_mps,
			accuracy_meters,
			recorded_at
		)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5, $6, $7, $8)
		ON CONFLICT (driver_id) DO UPDATE SET
			city_id = EXCLUDED.city_id,
			location = EXCLUDED.location,
			heading = EXCLUDED.heading,
			speed_mps = EXCLUDED.speed_mps,
			accuracy_meters = EXCLUDED.accuracy_meters,
			recorded_at = EXCLUDED.recorded_at`

	if _, err := repository.pool.Exec(
		ctx,
		query,
		update.DriverID,
		update.CityID,
		update.Location.Longitude,
		update.Location.Latitude,
		update.Heading,
		update.SpeedMPS,
		update.AccuracyMeters,
		update.RecordedAt,
	); err != nil {
		return fmt.Errorf("upsert driver location: %w", err)
	}
	return nil
}

func (repository *PostgresDriverLocationRepository) MarkStaleDriversOffline(ctx context.Context, staleBefore time.Time, limit int) (int, error) {
	const query = `
		WITH stale_drivers AS (
			SELECT d.id
			FROM drivers d
			INNER JOIN driver_locations dl ON dl.driver_id = d.id
			LEFT JOIN taxi_park_settings s ON s.taxi_park_id = d.taxi_park_id
			WHERE d.status = 'online'
			  AND dl.updated_at < CASE
			  	WHEN NULLIF(s.dispatch_driver_location_max_age_sec, 0) IS NULL THEN $1
			  	ELSE now() - (s.dispatch_driver_location_max_age_sec * interval '1 second')
			  END
			  AND d.deleted_at IS NULL
			LIMIT $2
		)
		UPDATE drivers
		SET status = 'offline'
		WHERE id IN (SELECT id FROM stale_drivers)`

	commandTag, err := repository.pool.Exec(ctx, query, staleBefore, limit)
	if err != nil {
		return 0, fmt.Errorf("mark stale drivers offline: %w", err)
	}
	return int(commandTag.RowsAffected()), nil
}

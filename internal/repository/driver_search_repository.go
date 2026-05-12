package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/develoop/taxi-platform/internal/dispatch"
	"github.com/develoop/taxi-platform/internal/domain"
)

type PostgresDriverSearchRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresDriverSearchRepository(pool *pgxpool.Pool) *PostgresDriverSearchRepository {
	return &PostgresDriverSearchRepository{pool: pool}
}

func (repository *PostgresDriverSearchRepository) FindNearestOnlineDrivers(ctx context.Context, query dispatch.NearestDriversQuery) ([]dispatch.DriverCandidate, error) {
	excludeIDs := query.ExcludeIDs
	if excludeIDs == nil {
		excludeIDs = []uuid.UUID{}
	}

	const sqlQuery = `
		WITH pickup AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS location
		)
		SELECT
			d.id,
			ST_Distance(dl.location, pickup.location) AS distance_meters,
			ST_Y(dl.location::geometry) AS latitude,
			ST_X(dl.location::geometry) AS longitude
		FROM drivers d
		INNER JOIN driver_locations dl ON dl.driver_id = d.id
		CROSS JOIN pickup
		WHERE d.city_id = $3
		  AND d.status = 'online'
		  AND d.is_verified = true
		  AND d.deleted_at IS NULL
		  AND dl.updated_at >= now() - make_interval(secs => $7)
		  AND ST_DWithin(dl.location, pickup.location, $4)
		  AND NOT (d.id = ANY($5::uuid[]))
		ORDER BY ST_Distance(dl.location, pickup.location) ASC
		LIMIT $6`

	rows, err := repository.pool.Query(
		ctx,
		sqlQuery,
		query.Pickup.Longitude,
		query.Pickup.Latitude,
		query.CityID,
		query.RadiusMeters,
		excludeIDs,
		query.Limit,
		int(query.LocationMaxAge.Seconds()),
	)
	if err != nil {
		return nil, fmt.Errorf("query nearest online drivers: %w", err)
	}
	defer rows.Close()

	candidates := make([]dispatch.DriverCandidate, 0, query.Limit)
	for rows.Next() {
		var candidate dispatch.DriverCandidate
		var latitude float64
		var longitude float64

		if err := rows.Scan(&candidate.DriverID, &candidate.DistanceMeters, &latitude, &longitude); err != nil {
			return nil, fmt.Errorf("scan nearest online driver: %w", err)
		}

		location, err := domain.NewCoordinates(latitude, longitude)
		if err != nil {
			return nil, fmt.Errorf("scan nearest online driver location: %w", err)
		}
		candidate.Location = location
		candidates = append(candidates, candidate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate nearest online drivers: %w", err)
	}

	return candidates, nil
}

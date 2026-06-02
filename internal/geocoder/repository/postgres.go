// Package repository provides PostgreSQL/PostGIS persistence for geocoder data.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (repository *PostgresRepository) ResolveActorCity(ctx context.Context, actorUserID uuid.UUID, actorRole string) (geoservice.CityContext, bool, error) {
	switch actorRole {
	case "taxi_park":
		return repository.resolveTaxiParkOwnerCity(ctx, actorUserID)
	case "dispatcher":
		return repository.resolveDispatcherCity(ctx, actorUserID)
	default:
		return geoservice.CityContext{}, false, nil
	}
}

func (repository *PostgresRepository) ResolveCity(ctx context.Context, cityID uuid.UUID) (geoservice.CityContext, bool, error) {
	return scanCityContext(repository.pool.QueryRow(ctx, `
		SELECT c.id, c.name, ST_Y(c.center::geometry), ST_X(c.center::geometry)
		FROM cities c
		WHERE c.id = $1
		  AND c.deleted_at IS NULL
		LIMIT 1`,
		cityID,
	))
}

func (repository *PostgresRepository) resolveTaxiParkOwnerCity(ctx context.Context, ownerUserID uuid.UUID) (geoservice.CityContext, bool, error) {
	return scanCityContext(repository.pool.QueryRow(ctx, `
		SELECT c.id, c.name, ST_Y(c.center::geometry), ST_X(c.center::geometry)
		FROM taxi_parks p
		JOIN cities c ON c.id = p.city_id
		WHERE p.owner_user_id = $1
		  AND p.deleted_at IS NULL
		  AND c.deleted_at IS NULL
		LIMIT 1`,
		ownerUserID,
	))
}

func (repository *PostgresRepository) resolveDispatcherCity(ctx context.Context, dispatcherUserID uuid.UUID) (geoservice.CityContext, bool, error) {
	return scanCityContext(repository.pool.QueryRow(ctx, `
		SELECT c.id, c.name, ST_Y(c.center::geometry), ST_X(c.center::geometry)
		FROM taxi_park_employees e
		JOIN taxi_parks p ON p.id = e.taxi_park_id
		JOIN cities c ON c.id = p.city_id
		WHERE e.user_id = $1
		  AND e.role = 'dispatcher'
		  AND e.status = 'active'
		  AND e.deleted_at IS NULL
		  AND p.deleted_at IS NULL
		  AND c.deleted_at IS NULL
		LIMIT 1`,
		dispatcherUserID,
	))
}

func (repository *PostgresRepository) SearchLocalPoints(ctx context.Context, request geoservice.LocalSearchRequest) ([]geodomain.SearchResult, error) {
	limit := request.Limit
	if limit <= 0 {
		limit = 10
	}
	var latitude any
	var longitude any
	if request.Focus != nil {
		latitude = request.Focus.Latitude
		longitude = request.Focus.Longitude
	}
	rows, err := repository.pool.Query(ctx, `
		SELECT id, city_id, name, address, ST_Y(location::geometry), ST_X(location::geometry),
		       confidence, trust_level
		FROM local_geo_points
		WHERE deleted_at IS NULL
		  AND trust_level IN ('confirmed', 'trusted')
		  AND ($2::uuid IS NULL OR city_id = $2)
		  AND (
		      normalized_name ILIKE '%' || $1 || '%'
		      OR lower(address) ILIKE '%' || $1 || '%'
		  )
		ORDER BY
		  CASE WHEN trust_level = 'trusted' THEN 0 ELSE 1 END,
		  CASE
		    WHEN $3::double precision IS NULL OR $4::double precision IS NULL THEN 0
		    ELSE ST_Distance(location, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography)
		  END,
		  confirmation_count DESC,
		  updated_at DESC
		LIMIT $5`,
		request.NormalizedQuery,
		request.CityID,
		latitude,
		longitude,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("query local geo points: %w", err)
	}
	defer rows.Close()

	results := make([]geodomain.SearchResult, 0)
	for rows.Next() {
		var id uuid.UUID
		var cityID uuid.UUID
		var name string
		var address string
		var lat float64
		var lon float64
		var confidence float64
		var trustLevel geodomain.TrustLevel
		if err := rows.Scan(&id, &cityID, &name, &address, &lat, &lon, &confidence, &trustLevel); err != nil {
			return nil, fmt.Errorf("scan local geo point result: %w", err)
		}
		coordinates, err := geodomain.NewCoordinates(lat, lon)
		if err != nil {
			return nil, fmt.Errorf("scan local geo point coordinates: %w", err)
		}
		pointID := id
		cityIDCopy := cityID
		results = append(results, geodomain.SearchResult{
			ID:           id.String(),
			LocalPointID: &pointID,
			Provider:     geodomain.ProviderLocal,
			Name:         name,
			Address:      address,
			CityID:       &cityIDCopy,
			Coordinates:  coordinates,
			Confidence:   confidence,
			TrustLevel:   trustLevel,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate local geo point results: %w", err)
	}
	return results, nil
}

func scanCityContext(row pgx.Row) (geoservice.CityContext, bool, error) {
	var cityID uuid.UUID
	var name string
	var latitude float64
	var longitude float64
	if err := row.Scan(&cityID, &name, &latitude, &longitude); err != nil {
		if err == pgx.ErrNoRows {
			return geoservice.CityContext{}, false, nil
		}
		return geoservice.CityContext{}, false, fmt.Errorf("scan geocoder actor city: %w", err)
	}
	center, err := geodomain.NewCoordinates(latitude, longitude)
	if err != nil {
		return geoservice.CityContext{}, false, fmt.Errorf("scan geocoder actor city center: %w", err)
	}
	return geoservice.CityContext{CityID: cityID, Name: name, Center: center}, true, nil
}

func (repository *PostgresRepository) GetExternalCache(ctx context.Context, provider geodomain.Provider, normalizedQuery string, cityID *uuid.UUID, now time.Time) ([]geodomain.SearchResult, bool, error) {
	var payload []byte
	var expiresAt time.Time
	err := repository.pool.QueryRow(ctx, `
		SELECT results, expires_at
		FROM geocoder_external_cache
		WHERE provider = $1
		  AND normalized_query = $2
		  AND (($3::uuid IS NULL AND city_id IS NULL) OR city_id = $3)
		  AND expires_at > $4
		ORDER BY updated_at DESC
		LIMIT 1`,
		provider,
		normalizedQuery,
		cityID,
		now,
	).Scan(&payload, &expiresAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("select geocoder external cache: %w", err)
	}
	var results []geodomain.SearchResult
	if err := json.Unmarshal(payload, &results); err != nil {
		return nil, false, fmt.Errorf("decode geocoder external cache: %w", err)
	}
	for index := range results {
		results[index].ExpiresAt = &expiresAt
	}
	return results, true, nil
}

func (repository *PostgresRepository) SaveExternalCache(ctx context.Context, cache geoservice.ExternalCacheRecord) error {
	results, err := json.Marshal(cache.Results)
	if err != nil {
		return fmt.Errorf("encode geocoder external cache results: %w", err)
	}
	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin geocoder cache transaction: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	if _, err := transaction.Exec(ctx, `
		DELETE FROM geocoder_external_cache
		WHERE provider = $1
		  AND normalized_query = $2
		  AND (($3::uuid IS NULL AND city_id IS NULL) OR city_id = $3)`,
		cache.Provider,
		cache.NormalizedQuery,
		cache.CityID,
	); err != nil {
		return fmt.Errorf("delete stale geocoder external cache: %w", err)
	}
	if _, err := transaction.Exec(ctx, `
		INSERT INTO geocoder_external_cache (
			provider, normalized_query, city_id, request_params, response, results, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		cache.Provider,
		cache.NormalizedQuery,
		cache.CityID,
		jsonOrEmptyObject(cache.RequestParams),
		jsonOrEmptyObject(cache.Response),
		results,
		cache.ExpiresAt,
	); err != nil {
		return fmt.Errorf("insert geocoder external cache: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit geocoder cache transaction: %w", err)
	}
	return nil
}

func (repository *PostgresRepository) ConfirmPoint(ctx context.Context, request geoservice.ConfirmPointRequest) (geodomain.LocalGeoPoint, error) {
	transaction, err := repository.pool.Begin(ctx)
	if err != nil {
		return geodomain.LocalGeoPoint{}, fmt.Errorf("begin confirm local geo point: %w", err)
	}
	defer func() { _ = transaction.Rollback(ctx) }()

	normalizedName := geodomain.NormalizeQuery(firstNonBlank(request.Name, request.Address))
	var pointID uuid.UUID
	err = transaction.QueryRow(ctx, `
		SELECT id
		FROM local_geo_points
		WHERE deleted_at IS NULL
		  AND city_id = $1
		  AND trust_level <> 'rejected'
		  AND (
		    normalized_name = $2
		    OR ST_DWithin(location, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography, 25)
		  )
		ORDER BY ST_Distance(location, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography)
		LIMIT 1`,
		request.CityID,
		normalizedName,
		request.Coordinates.Latitude,
		request.Coordinates.Longitude,
	).Scan(&pointID)
	if err != nil && err != pgx.ErrNoRows {
		return geodomain.LocalGeoPoint{}, fmt.Errorf("find existing local geo point: %w", err)
	}
	if err == pgx.ErrNoRows {
		if err := transaction.QueryRow(ctx, `
			INSERT INTO local_geo_points (
				city_id, name, normalized_name, address, location, source,
				external_provider, external_place_id, confidence, trust_level, confirmation_count
			) VALUES (
				$1, $2, $3, $4, ST_SetSRID(ST_MakePoint($6, $5), 4326)::geography, $7,
				NULLIF($8, ''), NULLIF($9, ''), $10, 'confirmed', 0
			)
			RETURNING id`,
			request.CityID,
			firstNonBlank(request.Name, request.Address),
			normalizedName,
			request.Address,
			request.Coordinates.Latitude,
			request.Coordinates.Longitude,
			request.Source,
			request.ExternalProvider,
			request.ExternalPlaceID,
			request.Confidence,
		).Scan(&pointID); err != nil {
			return geodomain.LocalGeoPoint{}, fmt.Errorf("insert local geo point: %w", err)
		}
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO local_geo_point_confirmations (
			point_id, user_id, actor_role, action, source, address, location, comment, ip, user_agent
		) VALUES (
			$1, $2, NULLIF($3, ''), 'confirm', $4, $5,
			ST_SetSRID(ST_MakePoint($7, $6), 4326)::geography, NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, '')
		)`,
		pointID,
		request.UserID,
		request.ActorRole,
		request.Source,
		request.Address,
		request.Coordinates.Latitude,
		request.Coordinates.Longitude,
		request.Comment,
		request.IP,
		request.UserAgent,
	); err != nil {
		return geodomain.LocalGeoPoint{}, fmt.Errorf("insert local geo point confirmation: %w", err)
	}
	point, err := scanLocalGeoPoint(transaction.QueryRow(ctx, `
		UPDATE local_geo_points
		SET confirmation_count = confirmation_count + 1,
		    trust_level = CASE WHEN confirmation_count + 1 >= 3 THEN 'trusted' ELSE trust_level END,
		    name = COALESCE(NULLIF($2, ''), name),
		    address = COALESCE(NULLIF($3, ''), address),
		    confidence = GREATEST(confidence, $4)
		WHERE id = $1
		RETURNING `+localGeoPointColumns,
		pointID,
		request.Name,
		request.Address,
		request.Confidence,
	))
	if err != nil {
		return geodomain.LocalGeoPoint{}, fmt.Errorf("update local geo point confirmation count: %w", err)
	}
	if err := transaction.Commit(ctx); err != nil {
		return geodomain.LocalGeoPoint{}, fmt.Errorf("commit confirm local geo point: %w", err)
	}
	return point, nil
}

func (repository *PostgresRepository) CreateLocalPoint(ctx context.Context, request geoservice.AdminLocalPointRequest) (geodomain.LocalGeoPoint, error) {
	point, err := scanLocalGeoPoint(repository.pool.QueryRow(ctx, `
		INSERT INTO local_geo_points (
			city_id, name, normalized_name, address, location, source,
			confidence, trust_level, confirmation_count
		) VALUES (
			$1, $2, $3, $4, ST_SetSRID(ST_MakePoint($6, $5), 4326)::geography, 'admin',
			1.0000, $7, CASE WHEN $7 = 'trusted' THEN 3 ELSE 1 END
		)
		RETURNING `+localGeoPointColumns,
		request.CityID,
		request.Name,
		geodomain.NormalizeQuery(request.Name),
		request.Address,
		request.Coordinates.Latitude,
		request.Coordinates.Longitude,
		request.TrustLevel,
	))
	if err != nil {
		return geodomain.LocalGeoPoint{}, fmt.Errorf("insert admin local geo point: %w", err)
	}
	return point, nil
}

func (repository *PostgresRepository) ListLocalPoints(ctx context.Context, filter geoservice.LocalPointFilter) ([]geodomain.LocalGeoPoint, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT `+localGeoPointColumns+`
		FROM local_geo_points
		WHERE deleted_at IS NULL
		  AND ($1::uuid IS NULL OR city_id = $1)
		  AND ($2::text = '' OR trust_level = $2)
		ORDER BY updated_at DESC
		LIMIT $3`,
		filter.CityID,
		filter.TrustLevel,
		filter.Limit,
	)
	if err != nil {
		return nil, fmt.Errorf("select local geo points: %w", err)
	}
	defer rows.Close()
	return scanLocalGeoPoints(rows)
}

func (repository *PostgresRepository) ApproveLocalPoint(ctx context.Context, id uuid.UUID, adminUserID *uuid.UUID) (geodomain.LocalGeoPoint, error) {
	point, err := scanLocalGeoPoint(repository.pool.QueryRow(ctx, `
		UPDATE local_geo_points
		SET trust_level = 'trusted',
		    confirmation_count = GREATEST(confirmation_count, 3),
		    approved_by = $2,
		    approved_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+localGeoPointColumns,
		id,
		adminUserID,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return geodomain.LocalGeoPoint{}, geodomain.ErrPointNotFound
		}
		return geodomain.LocalGeoPoint{}, fmt.Errorf("approve local geo point: %w", err)
	}
	return point, nil
}

func (repository *PostgresRepository) RejectLocalPoint(ctx context.Context, id uuid.UUID, adminUserID *uuid.UUID) (geodomain.LocalGeoPoint, error) {
	point, err := scanLocalGeoPoint(repository.pool.QueryRow(ctx, `
		UPDATE local_geo_points
		SET reject_count = reject_count + 1,
		    trust_level = CASE WHEN reject_count + 1 >= 5 THEN 'rejected' ELSE trust_level END,
		    rejected_by = $2,
		    rejected_at = now()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING `+localGeoPointColumns,
		id,
		adminUserID,
	))
	if err != nil {
		if err == pgx.ErrNoRows {
			return geodomain.LocalGeoPoint{}, geodomain.ErrPointNotFound
		}
		return geodomain.LocalGeoPoint{}, fmt.Errorf("reject local geo point: %w", err)
	}
	return point, nil
}

func (repository *PostgresRepository) ExportTrustedLocalPoints(ctx context.Context) ([]geodomain.LocalGeoPoint, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT `+localGeoPointColumns+`
		FROM local_geo_points
		WHERE deleted_at IS NULL AND trust_level = 'trusted'
		ORDER BY city_id, normalized_name`)
	if err != nil {
		return nil, fmt.Errorf("select trusted local geo points: %w", err)
	}
	defer rows.Close()
	return scanLocalGeoPoints(rows)
}

const localGeoPointColumns = `
	id, city_id, name, normalized_name, address, ST_Y(location::geometry), ST_X(location::geometry),
	source, COALESCE(external_provider, ''), COALESCE(external_place_id, ''), confidence, trust_level,
	confirmation_count, reject_count, approved_by, approved_at, rejected_by, rejected_at, created_at, updated_at`

func scanLocalGeoPoints(rows pgx.Rows) ([]geodomain.LocalGeoPoint, error) {
	points := make([]geodomain.LocalGeoPoint, 0)
	for rows.Next() {
		point, err := scanLocalGeoPoint(rows)
		if err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate local geo points: %w", err)
	}
	return points, nil
}

func scanLocalGeoPoint(row pgx.Row) (geodomain.LocalGeoPoint, error) {
	var point geodomain.LocalGeoPoint
	var latitude float64
	var longitude float64
	var approvedBy pgtype.UUID
	var rejectedBy pgtype.UUID
	var approvedAt pgtype.Timestamptz
	var rejectedAt pgtype.Timestamptz
	if err := row.Scan(
		&point.ID,
		&point.CityID,
		&point.Name,
		&point.NormalizedName,
		&point.Address,
		&latitude,
		&longitude,
		&point.Source,
		&point.ExternalProvider,
		&point.ExternalPlaceID,
		&point.Confidence,
		&point.TrustLevel,
		&point.ConfirmationCount,
		&point.RejectCount,
		&approvedBy,
		&approvedAt,
		&rejectedBy,
		&rejectedAt,
		&point.CreatedAt,
		&point.UpdatedAt,
	); err != nil {
		return geodomain.LocalGeoPoint{}, err
	}
	coordinates, err := geodomain.NewCoordinates(latitude, longitude)
	if err != nil {
		return geodomain.LocalGeoPoint{}, err
	}
	point.Coordinates = coordinates
	if approvedBy.Valid {
		value := uuid.UUID(approvedBy.Bytes)
		point.ApprovedBy = &value
	}
	if rejectedBy.Valid {
		value := uuid.UUID(rejectedBy.Bytes)
		point.RejectedBy = &value
	}
	if approvedAt.Valid {
		value := approvedAt.Time
		point.ApprovedAt = &value
	}
	if rejectedAt.Valid {
		value := rejectedAt.Time
		point.RejectedAt = &value
	}
	return point, nil
}

func jsonOrEmptyObject(value []byte) []byte {
	if json.Valid(value) {
		return value
	}
	return []byte("{}")
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

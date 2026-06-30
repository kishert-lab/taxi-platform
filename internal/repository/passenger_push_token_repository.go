package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PostgresPassengerPushTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPassengerPushTokenRepository(pool *pgxpool.Pool) *PostgresPassengerPushTokenRepository {
	return &PostgresPassengerPushTokenRepository{pool: pool}
}

func (repository *PostgresPassengerPushTokenRepository) Upsert(ctx context.Context, token domain.PassengerPushToken) (domain.PassengerPushToken, error) {
	const query = `
		INSERT INTO passenger_push_tokens (passenger_id, token, platform, device_id, is_active)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT (passenger_id, token) WHERE deleted_at IS NULL
		DO UPDATE SET
			platform = EXCLUDED.platform,
			device_id = EXCLUDED.device_id,
			is_active = true,
			last_seen_at = now(),
			deleted_at = NULL
		RETURNING id, passenger_id, token, platform, device_id, is_active, last_seen_at, created_at, updated_at, deleted_at`

	record, err := scanPassengerPushToken(repository.pool.QueryRow(
		ctx,
		query,
		token.PassengerID,
		token.Token,
		token.Platform,
		nullableString(token.DeviceID),
	))
	if err != nil {
		return domain.PassengerPushToken{}, fmt.Errorf("upsert passenger push token: %w", err)
	}

	return record, nil
}

func (repository *PostgresPassengerPushTokenRepository) ListActiveTokens(ctx context.Context, passengerID uuid.UUID) ([]domain.PassengerPushToken, error) {
	const query = `
		SELECT id, passenger_id, token, platform, device_id, is_active, last_seen_at, created_at, updated_at, deleted_at
		FROM passenger_push_tokens
		WHERE passenger_id = $1
		  AND is_active = true
		  AND deleted_at IS NULL`

	rows, err := repository.pool.Query(ctx, query, passengerID)
	if err != nil {
		return nil, fmt.Errorf("select passenger push tokens: %w", err)
	}
	defer rows.Close()

	tokens := make([]domain.PassengerPushToken, 0)
	for rows.Next() {
		record, scanErr := scanPassengerPushToken(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan passenger push token: %w", scanErr)
		}
		tokens = append(tokens, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate passenger push tokens: %w", err)
	}

	return tokens, nil
}

func scanPassengerPushToken(row pgx.Row) (domain.PassengerPushToken, error) {
	var token domain.PassengerPushToken
	var deviceID pgtype.Text
	var deletedAt pgtype.Timestamptz

	if err := row.Scan(
		&token.ID,
		&token.PassengerID,
		&token.Token,
		&token.Platform,
		&deviceID,
		&token.IsActive,
		&token.LastSeenAt,
		&token.CreatedAt,
		&token.UpdatedAt,
		&deletedAt,
	); err != nil {
		return domain.PassengerPushToken{}, err
	}
	token.DeviceID = deviceID.String
	if deletedAt.Valid {
		token.DeletedAt = &deletedAt.Time
	}
	return token, nil
}

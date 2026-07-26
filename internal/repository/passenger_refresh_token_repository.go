package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresPassengerRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPassengerRefreshTokenRepository(pool *pgxpool.Pool) *PostgresPassengerRefreshTokenRepository {
	return &PostgresPassengerRefreshTokenRepository{pool: pool}
}

func (repository *PostgresPassengerRefreshTokenRepository) Store(ctx context.Context, passengerID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO passenger_refresh_tokens (passenger_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`

	if _, err := repository.pool.Exec(ctx, query, passengerID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("insert passenger refresh token: %w", err)
	}

	return nil
}

func (repository *PostgresPassengerRefreshTokenRepository) Rotate(ctx context.Context, oldTokenHash string, passengerID uuid.UUID, newTokenHash string, newExpiresAt time.Time) error {
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin passenger refresh rotation: %w", err)
	}
	defer func() {
		rollbackErr := tx.Rollback(ctx)
		if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
			err = fmt.Errorf("rollback passenger refresh rotation: %w", rollbackErr)
		}
	}()

	tag, err := tx.Exec(ctx, `
		UPDATE passenger_refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1
		  AND passenger_id = $2
		  AND revoked_at IS NULL
		  AND expires_at > now()`,
		oldTokenHash,
		passengerID,
	)
	if err != nil {
		return fmt.Errorf("revoke old passenger refresh token: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("revoke old passenger refresh token: %w", pgx.ErrNoRows)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO passenger_refresh_tokens (passenger_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		passengerID,
		newTokenHash,
		newExpiresAt,
	); err != nil {
		return fmt.Errorf("insert rotated passenger refresh token: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit passenger refresh rotation: %w", err)
	}

	return nil
}

func (repository *PostgresPassengerRefreshTokenRepository) Revoke(ctx context.Context, tokenHash string) error {
	const query = `
		UPDATE passenger_refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1
		  AND revoked_at IS NULL`

	if _, err := repository.pool.Exec(ctx, query, tokenHash); err != nil {
		return fmt.Errorf("revoke passenger refresh token: %w", err)
	}

	return nil
}

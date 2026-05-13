package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRefreshTokenRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRefreshTokenRepository(pool *pgxpool.Pool) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{pool: pool}
}

func (repository *PostgresRefreshTokenRepository) StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	const query = `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`
	if _, err := repository.pool.Exec(ctx, query, userID, tokenHash, expiresAt); err != nil {
		return fmt.Errorf("insert refresh token: %w", err)
	}

	return nil
}

func (repository *PostgresRefreshTokenRepository) RotateRefreshToken(ctx context.Context, oldTokenHash string, userID uuid.UUID, newTokenHash string, newExpiresAt time.Time) error {
	transaction, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin refresh token rotation: %w", err)
	}
	defer rollbackTx(ctx, transaction)

	commandTag, err := transaction.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1
		  AND user_id = $2
		  AND revoked_at IS NULL
		  AND expires_at > now()`, oldTokenHash, userID)
	if err != nil {
		return fmt.Errorf("revoke old refresh token: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("revoke old refresh token: %w", pgx.ErrNoRows)
	}

	if _, err := transaction.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, userID, newTokenHash, newExpiresAt); err != nil {
		return fmt.Errorf("insert rotated refresh token: %w", err)
	}

	if err := transaction.Commit(ctx); err != nil {
		return fmt.Errorf("commit refresh token rotation: %w", err)
	}

	return nil
}

func (repository *PostgresRefreshTokenRepository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if _, err := repository.pool.Exec(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = now()
		WHERE token_hash = $1
		  AND revoked_at IS NULL`, tokenHash); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

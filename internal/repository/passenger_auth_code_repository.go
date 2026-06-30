package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PostgresPassengerAuthCodeRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresPassengerAuthCodeRepository(pool *pgxpool.Pool) *PostgresPassengerAuthCodeRepository {
	return &PostgresPassengerAuthCodeRepository{pool: pool}
}

func (repository *PostgresPassengerAuthCodeRepository) Create(ctx context.Context, code domain.PassengerAuthCode) (domain.PassengerAuthCode, error) {
	const query = `
		INSERT INTO passenger_auth_codes (phone, code_hash, max_attempts, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, phone, code_hash, attempts, max_attempts, expires_at, used_at, created_at, updated_at`

	record, err := scanPassengerAuthCode(repository.pool.QueryRow(
		ctx,
		query,
		code.Phone,
		code.CodeHash,
		code.MaxAttempts,
		code.ExpiresAt,
	))
	if err != nil {
		return domain.PassengerAuthCode{}, fmt.Errorf("insert passenger auth code: %w", err)
	}

	return record, nil
}

func (repository *PostgresPassengerAuthCodeRepository) InvalidateActiveByPhone(ctx context.Context, phone string, invalidatedAt time.Time) error {
	const query = `
		UPDATE passenger_auth_codes
		SET used_at = $2
		WHERE phone = $1
		  AND used_at IS NULL
		  AND expires_at > $2`

	if _, err := repository.pool.Exec(ctx, query, phone, invalidatedAt); err != nil {
		return fmt.Errorf("invalidate passenger auth codes: %w", err)
	}

	return nil
}

func (repository *PostgresPassengerAuthCodeRepository) GetLatestActiveByPhone(ctx context.Context, phone string) (domain.PassengerAuthCode, error) {
	const query = `
		SELECT id, phone, code_hash, attempts, max_attempts, expires_at, used_at, created_at, updated_at
		FROM passenger_auth_codes
		WHERE phone = $1
		ORDER BY created_at DESC
		LIMIT 1`

	record, err := scanPassengerAuthCode(repository.pool.QueryRow(ctx, query, phone))
	if err != nil {
		return domain.PassengerAuthCode{}, fmt.Errorf("select passenger auth code: %w", err)
	}

	return record, nil
}

func (repository *PostgresPassengerAuthCodeRepository) IncrementAttempts(ctx context.Context, codeID uuid.UUID) error {
	const query = `
		UPDATE passenger_auth_codes
		SET attempts = attempts + 1
		WHERE id = $1`

	tag, err := repository.pool.Exec(ctx, query, codeID)
	if err != nil {
		return fmt.Errorf("increment passenger auth attempts: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("increment passenger auth attempts: %w", pgx.ErrNoRows)
	}

	return nil
}

func (repository *PostgresPassengerAuthCodeRepository) MarkUsed(ctx context.Context, codeID uuid.UUID, usedAt time.Time) error {
	const query = `
		UPDATE passenger_auth_codes
		SET used_at = $2
		WHERE id = $1`

	tag, err := repository.pool.Exec(ctx, query, codeID, usedAt)
	if err != nil {
		return fmt.Errorf("mark passenger auth code used: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("mark passenger auth code used: %w", pgx.ErrNoRows)
	}

	return nil
}

func scanPassengerAuthCode(row pgx.Row) (domain.PassengerAuthCode, error) {
	var record domain.PassengerAuthCode
	var usedAt pgtype.Timestamptz

	if err := row.Scan(
		&record.ID,
		&record.Phone,
		&record.CodeHash,
		&record.Attempts,
		&record.MaxAttempts,
		&record.ExpiresAt,
		&usedAt,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return domain.PassengerAuthCode{}, err
	}
	if usedAt.Valid {
		record.UsedAt = &usedAt.Time
	}

	return record, nil
}

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

type PostgresVerificationCodeRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresVerificationCodeRepository(pool *pgxpool.Pool) *PostgresVerificationCodeRepository {
	return &PostgresVerificationCodeRepository{pool: pool}
}

func (repository *PostgresVerificationCodeRepository) CreateVerificationCode(ctx context.Context, code domain.VerificationCode) (domain.VerificationCode, error) {
	const query = `
		INSERT INTO user_verification_codes (
			user_id,
			target,
			channel,
			purpose,
			code_hash,
			attempts,
			max_attempts,
			expires_at,
			last_sent_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING
			id,
			user_id,
			target,
			channel,
			purpose,
			code_hash,
			attempts,
			max_attempts,
			expires_at,
			consumed_at,
			created_at,
			last_sent_at`

	createdCode, err := scanVerificationCode(repository.pool.QueryRow(
		ctx,
		query,
		code.UserID,
		code.Target,
		code.Channel,
		code.Purpose,
		code.CodeHash,
		code.Attempts,
		code.MaxAttempts,
		code.ExpiresAt,
		code.LastSentAt,
	))
	if err != nil {
		return domain.VerificationCode{}, fmt.Errorf("insert verification code: %w", err)
	}

	return createdCode, nil
}

func (repository *PostgresVerificationCodeRepository) GetLatestActiveCode(ctx context.Context, target string, channel domain.VerificationChannel, purpose domain.VerificationPurpose) (domain.VerificationCode, error) {
	const query = `
		SELECT
			id,
			user_id,
			target,
			channel,
			purpose,
			code_hash,
			attempts,
			max_attempts,
			expires_at,
			consumed_at,
			created_at,
			last_sent_at
		FROM user_verification_codes
		WHERE target = $1
		  AND channel = $2
		  AND purpose = $3
		  AND consumed_at IS NULL
		  AND attempts < max_attempts
		ORDER BY created_at DESC
		LIMIT 1`

	code, err := scanVerificationCode(repository.pool.QueryRow(ctx, query, target, channel, purpose))
	if err != nil {
		return domain.VerificationCode{}, fmt.Errorf("select latest active verification code: %w", err)
	}

	return code, nil
}

func (repository *PostgresVerificationCodeRepository) IncrementAttempts(ctx context.Context, codeID uuid.UUID) error {
	const query = `
		UPDATE user_verification_codes
		SET attempts = attempts + 1
		WHERE id = $1 AND consumed_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, codeID)
	if err != nil {
		return fmt.Errorf("increment verification code attempts: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("increment verification code attempts: %w", pgx.ErrNoRows)
	}

	return nil
}

func (repository *PostgresVerificationCodeRepository) ConsumeCode(ctx context.Context, codeID uuid.UUID, consumedAt time.Time) error {
	const query = `
		UPDATE user_verification_codes
		SET consumed_at = $2
		WHERE id = $1 AND consumed_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, codeID, consumedAt)
	if err != nil {
		return fmt.Errorf("consume verification code: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("consume verification code: %w", pgx.ErrNoRows)
	}

	return nil
}

func scanVerificationCode(row pgx.Row) (domain.VerificationCode, error) {
	var code domain.VerificationCode
	var userID pgtype.UUID
	var consumedAt pgtype.Timestamptz

	if err := row.Scan(
		&code.ID,
		&userID,
		&code.Target,
		&code.Channel,
		&code.Purpose,
		&code.CodeHash,
		&code.Attempts,
		&code.MaxAttempts,
		&code.ExpiresAt,
		&consumedAt,
		&code.CreatedAt,
		&code.LastSentAt,
	); err != nil {
		return domain.VerificationCode{}, err
	}

	if userID.Valid {
		code.UserID = uuid.UUID(userID.Bytes)
	}
	if consumedAt.Valid {
		code.ConsumedAt = &consumedAt.Time
	}

	return code, nil
}

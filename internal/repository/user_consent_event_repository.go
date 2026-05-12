package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/develoop/taxi-platform/internal/domain"
)

type PostgresUserConsentEventRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresUserConsentEventRepository(pool *pgxpool.Pool) *PostgresUserConsentEventRepository {
	return &PostgresUserConsentEventRepository{pool: pool}
}

func (repository *PostgresUserConsentEventRepository) CreateUserConsentEvent(ctx context.Context, event domain.UserConsentEvent) error {
	const query = `
		INSERT INTO user_consent_events (
			user_id,
			event_type,
			document_type,
			document_version,
			ip,
			user_agent,
			created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`

	if _, err := repository.pool.Exec(
		ctx,
		query,
		event.UserID,
		event.EventType,
		event.DocumentType,
		event.DocumentVersion,
		nullableString(event.IP),
		nullableString(event.UserAgent),
		event.CreatedAt,
	); err != nil {
		return fmt.Errorf("insert user consent event: %w", err)
	}

	return nil
}

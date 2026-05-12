package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresDriverStateRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresDriverStateRepository(pool *pgxpool.Pool) *PostgresDriverStateRepository {
	return &PostgresDriverStateRepository{pool: pool}
}

func (repository *PostgresDriverStateRepository) MarkDriverBusy(ctx context.Context, driverID uuid.UUID) error {
	const query = `
		UPDATE drivers
		SET status = 'busy'
		WHERE id = $1
		  AND status = 'online'
		  AND deleted_at IS NULL`

	commandTag, err := repository.pool.Exec(ctx, query, driverID)
	if err != nil {
		return fmt.Errorf("mark driver busy: %w", err)
	}
	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("mark driver busy: %w", pgx.ErrNoRows)
	}

	return nil
}

package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	auditapp "github.com/kishert-lab/taxi-platform/internal/audit"
)

type PostgresTransportRequestLogRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTransportRequestLogRepository(pool *pgxpool.Pool) *PostgresTransportRequestLogRepository {
	return &PostgresTransportRequestLogRepository{pool: pool}
}

func (repository *PostgresTransportRequestLogRepository) CreateTransportRequestLog(ctx context.Context, logEntry auditapp.TransportRequestLog) error {
	metadata, err := json.Marshal(logEntry.Metadata)
	if err != nil {
		return fmt.Errorf("marshal transport request log metadata: %w", err)
	}

	var actorUserID pgtype.UUID
	if logEntry.ActorUserID != uuid.Nil {
		actorUserID = pgtype.UUID{Bytes: logEntry.ActorUserID, Valid: true}
	}

	_, err = repository.pool.Exec(ctx, `
		INSERT INTO transport_request_logs (
			protocol,
			event_type,
			request_id,
			method,
			route,
			path,
			raw_query,
			status_code,
			duration_ms,
			client_ip,
			user_agent,
			actor_user_id,
			actor_role,
			error_message,
			metadata
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8, 0), NULLIF($9, 0), $10, $11, $12, NULLIF($13, ''), NULLIF($14, ''), $15)`,
		logEntry.Protocol,
		logEntry.EventType,
		nullableString(logEntry.RequestID),
		nullableString(logEntry.Method),
		nullableString(logEntry.Route),
		logEntry.Path,
		logEntry.RawQuery,
		logEntry.StatusCode,
		logEntry.Duration.Milliseconds(),
		nullableString(logEntry.ClientIP),
		nullableString(logEntry.UserAgent),
		actorUserID,
		string(logEntry.ActorRole),
		logEntry.ErrorMessage,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("insert transport request log: %w", err)
	}

	return nil
}

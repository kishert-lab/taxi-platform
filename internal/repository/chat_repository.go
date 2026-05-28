package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	chatapp "github.com/kishert-lab/taxi-platform/internal/chat"
	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PostgresChatRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresChatRepository(pool *pgxpool.Pool) *PostgresChatRepository {
	return &PostgresChatRepository{pool: pool}
}

func (repository *PostgresChatRepository) EnsureOrderThread(ctx context.Context, orderID uuid.UUID, chatType domain.ChatType) (domain.ChatThread, error) {
	thread, err := scanChatThread(repository.pool.QueryRow(ctx, `
		WITH order_context AS (
			SELECT o.id AS order_id,
			       (o.metadata->>'taxi_park_id')::uuid AS taxi_park_id,
			       o.passenger_id,
			       o.driver_id
			FROM orders o
			WHERE o.id = $1 AND o.deleted_at IS NULL
		),
		upserted AS (
			INSERT INTO chat_threads (chat_type, order_id, taxi_park_id, passenger_id, driver_id)
			SELECT $2, order_id, taxi_park_id, passenger_id, driver_id
			FROM order_context
			ON CONFLICT (order_id, chat_type) WHERE order_id IS NOT NULL AND deleted_at IS NULL
			DO UPDATE SET passenger_id = EXCLUDED.passenger_id,
			              driver_id = EXCLUDED.driver_id,
			              taxi_park_id = EXCLUDED.taxi_park_id
			RETURNING id, chat_type, order_id, taxi_park_id, passenger_id, driver_id, status, created_at, updated_at, closed_at
		)
		SELECT id, chat_type, order_id, taxi_park_id, passenger_id, driver_id, status, created_at, updated_at, closed_at
		FROM upserted`, orderID, chatType))
	if err != nil {
		return domain.ChatThread{}, mapChatScanError("ensure order chat thread", err)
	}
	return thread, nil
}

func (repository *PostgresChatRepository) EnsurePassengerSupportThread(ctx context.Context, passengerID uuid.UUID) (domain.ChatThread, error) {
	thread, err := scanChatThread(repository.pool.QueryRow(ctx, `
		INSERT INTO chat_threads (chat_type, passenger_id)
		VALUES ('passenger_support', $1)
		ON CONFLICT (passenger_id, chat_type) WHERE order_id IS NULL AND chat_type = 'passenger_support' AND deleted_at IS NULL
		DO UPDATE SET passenger_id = EXCLUDED.passenger_id
		RETURNING id, chat_type, order_id, taxi_park_id, passenger_id, driver_id, status, created_at, updated_at, closed_at`,
		passengerID,
	))
	if err != nil {
		return domain.ChatThread{}, mapChatScanError("ensure passenger support chat thread", err)
	}
	return thread, nil
}

func (repository *PostgresChatRepository) CreateMessage(ctx context.Context, thread domain.ChatThread, senderUserID uuid.UUID, senderRole domain.UserRole, body string) (domain.ChatMessage, error) {
	message, err := scanChatMessage(repository.pool.QueryRow(ctx, `
		INSERT INTO chat_messages (thread_id, order_id, sender_user_id, sender_role, body)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, thread_id, order_id, sender_user_id, sender_role, body, created_at, edited_at`,
		thread.ID,
		nullableUUID(thread.OrderID),
		senderUserID,
		senderRole,
		body,
	))
	if err != nil {
		return domain.ChatMessage{}, fmt.Errorf("insert chat message: %w", err)
	}
	return message, nil
}

func (repository *PostgresChatRepository) ListMessages(ctx context.Context, thread domain.ChatThread, limit int) ([]domain.ChatMessage, error) {
	rows, err := repository.pool.Query(ctx, `
		SELECT id, thread_id, order_id, sender_user_id, sender_role, body, created_at, edited_at
		FROM (
			SELECT id, thread_id, order_id, sender_user_id, sender_role, body, created_at, edited_at
			FROM chat_messages
			WHERE thread_id = $1 AND deleted_at IS NULL
			ORDER BY created_at DESC
			LIMIT $2
		) recent
		ORDER BY created_at ASC`, thread.ID, limit)
	if err != nil {
		return nil, fmt.Errorf("select chat messages: %w", err)
	}
	defer rows.Close()

	messages := make([]domain.ChatMessage, 0, limit)
	for rows.Next() {
		message, err := scanChatMessage(rows)
		if err != nil {
			return nil, fmt.Errorf("scan chat message: %w", err)
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chat messages: %w", err)
	}
	return messages, nil
}

func (repository *PostgresChatRepository) GetOrderChatContext(ctx context.Context, orderID uuid.UUID) (chatapp.OrderChatContext, error) {
	var result chatapp.OrderChatContext
	var driverID pgtype.UUID
	var driverUserID pgtype.UUID
	var taxiParkID pgtype.UUID
	if err := repository.pool.QueryRow(ctx, `
		SELECT o.id,
		       o.status,
		       o.passenger_id,
		       o.driver_id,
		       d.user_id,
		       CASE WHEN o.metadata ? 'taxi_park_id' THEN (o.metadata->>'taxi_park_id')::uuid ELSE NULL END
		FROM orders o
		LEFT JOIN drivers d ON d.id = o.driver_id AND d.deleted_at IS NULL
		WHERE o.id = $1 AND o.deleted_at IS NULL`, orderID).Scan(
		&result.OrderID,
		&result.Status,
		&result.PassengerID,
		&driverID,
		&driverUserID,
		&taxiParkID,
	); err != nil {
		return chatapp.OrderChatContext{}, mapChatScanError("select order chat context", err)
	}
	if driverID.Valid {
		value := uuid.UUID(driverID.Bytes)
		result.DriverID = &value
	}
	if driverUserID.Valid {
		value := uuid.UUID(driverUserID.Bytes)
		result.DriverUserID = &value
	}
	if taxiParkID.Valid {
		value := uuid.UUID(taxiParkID.Bytes)
		result.TaxiParkID = &value
	}
	return result, nil
}

func (repository *PostgresChatRepository) IsTaxiParkActor(ctx context.Context, taxiParkID uuid.UUID, actorUserID uuid.UUID) (bool, error) {
	var exists bool
	if err := repository.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM taxi_parks tp
			WHERE tp.id = $1
			  AND tp.owner_user_id = $2
			  AND tp.deleted_at IS NULL
			UNION
			SELECT 1
			FROM taxi_park_staff staff
			JOIN users u ON u.id = staff.user_id
			WHERE staff.taxi_park_id = $1
			  AND staff.user_id = $2
			  AND staff.role IN ('dispatcher', 'taxi_park')
			  AND staff.is_active = true
			  AND staff.deleted_at IS NULL
			  AND u.is_active = true
			  AND u.deleted_at IS NULL
		)`, taxiParkID, actorUserID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check taxi park chat actor: %w", err)
	}
	return exists, nil
}

func scanChatThread(row pgx.Row) (domain.ChatThread, error) {
	var thread domain.ChatThread
	var orderID pgtype.UUID
	var taxiParkID pgtype.UUID
	var passengerID pgtype.UUID
	var driverID pgtype.UUID
	var closedAt pgtype.Timestamptz
	if err := row.Scan(
		&thread.ID,
		&thread.Type,
		&orderID,
		&taxiParkID,
		&passengerID,
		&driverID,
		&thread.Status,
		&thread.CreatedAt,
		&thread.UpdatedAt,
		&closedAt,
	); err != nil {
		return domain.ChatThread{}, err
	}
	thread.OrderID = uuidPointer(orderID)
	thread.TaxiParkID = uuidPointer(taxiParkID)
	thread.PassengerID = uuidPointer(passengerID)
	thread.DriverID = uuidPointer(driverID)
	if closedAt.Valid {
		thread.ClosedAt = &closedAt.Time
	}
	return thread, nil
}

func scanChatMessage(row pgx.Row) (domain.ChatMessage, error) {
	var message domain.ChatMessage
	var orderID pgtype.UUID
	var editedAt pgtype.Timestamptz
	if err := row.Scan(
		&message.ID,
		&message.ThreadID,
		&orderID,
		&message.SenderUserID,
		&message.SenderRole,
		&message.Body,
		&message.CreatedAt,
		&editedAt,
	); err != nil {
		return domain.ChatMessage{}, err
	}
	message.OrderID = uuidPointer(orderID)
	if editedAt.Valid {
		message.EditedAt = &editedAt.Time
	}
	return message, nil
}

func uuidPointer(value pgtype.UUID) *uuid.UUID {
	if !value.Valid {
		return nil
	}
	result := uuid.UUID(value.Bytes)
	return &result
}

func mapChatScanError(operation string, err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return chatapp.ErrChatUnavailable
	}
	return fmt.Errorf("%s: %w", operation, err)
}

package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	wsmsg "github.com/kishert-lab/taxi-platform/internal/ws"
)

type RealtimeGateway struct {
	client *goredis.Client
	pool   *pgxpool.Pool
}

func NewRealtimeGateway(client *goredis.Client, pool *pgxpool.Pool) *RealtimeGateway {
	return &RealtimeGateway{client: client, pool: pool}
}

func (gateway *RealtimeGateway) SendToDriver(ctx context.Context, driverID uuid.UUID, eventName string, payload any) error {
	userID, err := gateway.driverUserID(ctx, driverID)
	if err != nil {
		return err
	}
	return gateway.publishToUser(ctx, userID, driverRealtimeEvent(eventName), payload)
}

func (gateway *RealtimeGateway) SendToPassenger(ctx context.Context, passengerID uuid.UUID, eventName string, payload any) error {
	return gateway.publishToUser(ctx, passengerID, passengerRealtimeEvent(eventName), payload)
}

func (gateway *RealtimeGateway) SendDriverPresenceToTaxiPark(ctx context.Context, driverID uuid.UUID, payload any) error {
	recipientUserIDs, err := gateway.taxiParkRealtimeRecipientUserIDs(ctx, driverID)
	if err != nil {
		return err
	}
	for _, userID := range recipientUserIDs {
		if err := gateway.publishToUser(ctx, userID, "driver.status_changed", payload); err != nil {
			return err
		}
	}
	return nil
}

func (gateway *RealtimeGateway) SendDriverLocationToTaxiPark(ctx context.Context, driverID uuid.UUID, payload any) error {
	recipientUserIDs, err := gateway.taxiParkRealtimeRecipientUserIDs(ctx, driverID)
	if err != nil {
		return err
	}
	for _, userID := range recipientUserIDs {
		if err := gateway.publishToUser(ctx, userID, wsmsg.EventDriverLocation, payload); err != nil {
			return err
		}
	}
	return nil
}

func (gateway *RealtimeGateway) SendToTaxiParkByOrder(ctx context.Context, orderID uuid.UUID, eventName string, payload any) error {
	recipientUserIDs, err := gateway.taxiParkRealtimeRecipientUserIDsByOrder(ctx, orderID)
	if err != nil {
		return err
	}
	for _, userID := range recipientUserIDs {
		if err := gateway.publishToUser(ctx, userID, eventName, payload); err != nil {
			return err
		}
	}
	return nil
}

func (gateway *RealtimeGateway) driverUserID(ctx context.Context, driverID uuid.UUID) (uuid.UUID, error) {
	var userID uuid.UUID
	if err := gateway.pool.QueryRow(ctx, `
		SELECT user_id
		FROM drivers
		WHERE id = $1 AND deleted_at IS NULL`, driverID).Scan(&userID); err != nil {
		if err == pgx.ErrNoRows {
			return uuid.Nil, fmt.Errorf("dispatch realtime driver user not found: %w", err)
		}
		return uuid.Nil, fmt.Errorf("select dispatch realtime driver user: %w", err)
	}
	return userID, nil
}

func (gateway *RealtimeGateway) taxiParkRealtimeRecipientUserIDs(ctx context.Context, driverID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := gateway.pool.Query(ctx, `
		WITH target_park AS (
			SELECT tp.id, tp.owner_user_id
			FROM drivers d
			JOIN taxi_parks tp ON tp.id = d.taxi_park_id
			WHERE d.id = $1
			  AND d.deleted_at IS NULL
			  AND tp.deleted_at IS NULL
		)
		SELECT owner_user_id
		FROM target_park
		UNION
		SELECT staff.user_id
		FROM taxi_park_staff staff
		JOIN users u ON u.id = staff.user_id
		JOIN target_park tp ON tp.id = staff.taxi_park_id
		WHERE staff.role IN ('dispatcher', 'taxi_park')
		  AND staff.is_active = true
		  AND staff.deleted_at IS NULL
		  AND u.is_active = true
		  AND u.deleted_at IS NULL`, driverID)
	if err != nil {
		return nil, fmt.Errorf("select taxi park realtime recipients: %w", err)
	}
	defer rows.Close()

	recipients := make([]uuid.UUID, 0)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan taxi park realtime recipient: %w", err)
		}
		recipients = append(recipients, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate taxi park realtime recipients: %w", err)
	}
	if len(recipients) == 0 {
		return nil, fmt.Errorf("driver taxi park realtime recipients not found: %w", pgx.ErrNoRows)
	}
	return recipients, nil
}

func (gateway *RealtimeGateway) taxiParkRealtimeRecipientUserIDsByOrder(ctx context.Context, orderID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := gateway.pool.Query(ctx, `
		WITH target_park AS (
			SELECT (o.metadata->>'taxi_park_id')::uuid AS id
			FROM orders o
			WHERE o.id = $1
			  AND o.metadata ? 'taxi_park_id'
			  AND o.deleted_at IS NULL
		)
		SELECT tp.owner_user_id
		FROM taxi_parks tp
		JOIN target_park target ON target.id = tp.id
		WHERE tp.deleted_at IS NULL
		UNION
		SELECT staff.user_id
		FROM taxi_park_staff staff
		JOIN users u ON u.id = staff.user_id
		JOIN target_park target ON target.id = staff.taxi_park_id
		WHERE staff.role IN ('dispatcher', 'taxi_park')
		  AND staff.is_active = true
		  AND staff.deleted_at IS NULL
		  AND u.is_active = true
		  AND u.deleted_at IS NULL`, orderID)
	if err != nil {
		return nil, fmt.Errorf("select taxi park order realtime recipients: %w", err)
	}
	defer rows.Close()

	recipients := make([]uuid.UUID, 0)
	for rows.Next() {
		var userID uuid.UUID
		if err := rows.Scan(&userID); err != nil {
			return nil, fmt.Errorf("scan taxi park order realtime recipient: %w", err)
		}
		recipients = append(recipients, userID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate taxi park order realtime recipients: %w", err)
	}
	if len(recipients) == 0 {
		return nil, fmt.Errorf("taxi park order realtime recipients not found: %w", pgx.ErrNoRows)
	}
	return recipients, nil
}

func (gateway *RealtimeGateway) publishToUser(ctx context.Context, userID uuid.UUID, eventName string, payload any) error {
	messagePayload, ok := payload.(map[string]any)
	if !ok {
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal realtime websocket payload: %w", err)
		}
		if err := json.Unmarshal(payloadBytes, &messagePayload); err != nil {
			return fmt.Errorf("decode realtime websocket payload as object: %w", err)
		}
		if messagePayload == nil {
			messagePayload = map[string]any{}
		}
	}
	message := wsmsg.NewMessage(eventName, uuid.New(), messagePayload)
	bytes, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshal realtime websocket message: %w", err)
	}
	if err := gateway.client.Publish(ctx, userWebSocketChannel(userID), bytes).Err(); err != nil {
		return fmt.Errorf("publish realtime websocket message: %w", err)
	}
	return nil
}

func userWebSocketChannel(userID uuid.UUID) string {
	return fmt.Sprintf("ws:user:%s", userID)
}

func driverRealtimeEvent(eventName string) string {
	switch eventName {
	case "order.assigned":
		return wsmsg.EventOrderAccepted
	case "order.expired":
		return wsmsg.EventOrderOfferExpired
	default:
		return eventName
	}
}

func passengerRealtimeEvent(eventName string) string {
	switch eventName {
	case "driver_assigned":
		return wsmsg.EventOrderDriverAssigned
	default:
		return eventName
	}
}

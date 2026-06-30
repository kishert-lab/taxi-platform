// Package chat contains application use cases for order and support chats.
package chat

import (
	"context"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type Repository interface {
	EnsureOrderThread(ctx context.Context, orderID uuid.UUID, chatType domain.ChatType) (domain.ChatThread, error)
	EnsurePassengerSupportThread(ctx context.Context, passengerID uuid.UUID) (domain.ChatThread, error)
	CreateMessage(ctx context.Context, thread domain.ChatThread, senderID uuid.UUID, senderRole domain.UserRole, body string) (domain.ChatMessage, error)
	ListMessages(ctx context.Context, thread domain.ChatThread, limit int) ([]domain.ChatMessage, error)
	GetOrderChatContext(ctx context.Context, orderID uuid.UUID) (OrderChatContext, error)
	IsTaxiParkActor(ctx context.Context, taxiParkID uuid.UUID, actorUserID uuid.UUID) (bool, error)
}

type RealtimeGateway interface {
	SendToDriver(ctx context.Context, driverID uuid.UUID, eventName string, payload any) error
	SendToPassenger(ctx context.Context, passengerID uuid.UUID, eventName string, payload any) error
	SendToTaxiParkByOrder(ctx context.Context, orderID uuid.UUID, eventName string, payload any) error
}

type OrderChatContext struct {
	OrderID      uuid.UUID
	Status       domain.OrderStatus
	PassengerID  uuid.UUID
	DriverID     *uuid.UUID
	DriverUserID *uuid.UUID
	TaxiParkID   *uuid.UUID
}

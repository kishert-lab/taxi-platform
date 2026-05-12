package order

import (
	"context"

	"github.com/google/uuid"

	"github.com/develoop/taxi-platform/internal/domain"
)

type Repository interface {
	GetOrderByID(ctx context.Context, orderID uuid.UUID) (domain.Order, error)
	GetCurrentOrderByPassengerID(ctx context.Context, passengerID uuid.UUID) (domain.Order, error)
	GetCurrentOrderByDriverID(ctx context.Context, driverID uuid.UUID) (domain.Order, error)
	TransitionOrderStatus(ctx context.Context, transition domain.OrderTransition) (domain.Order, bool, error)
	AddStateEvent(ctx context.Context, event OrderEvent) error
}

type DispatchController interface {
	StopDispatch(ctx context.Context, orderID uuid.UUID) error
}

type RealtimePublisher interface {
	SendToDriver(ctx context.Context, driverID uuid.UUID, eventName string, payload any) error
	SendToPassenger(ctx context.Context, passengerID uuid.UUID, eventName string, payload any) error
}

type OrderEvent struct {
	OrderID       uuid.UUID
	ActorUserID   *uuid.UUID
	ActorDriverID *uuid.UUID
	EventType     domain.OrderEventType
	Payload       map[string]any
}

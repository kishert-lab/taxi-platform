package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/develoop/taxi-platform/internal/domain"
)

type Service struct {
	repository         Repository
	dispatchController DispatchController
	realtimePublisher  RealtimePublisher
	logger             *zap.Logger
}

type NewServiceParams struct {
	Repository         Repository
	DispatchController DispatchController
	RealtimePublisher  RealtimePublisher
	Logger             *zap.Logger
}

var ErrOrderConcurrentUpdate = errors.New("order concurrent update")

func NewService(params NewServiceParams) *Service {
	logger := params.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		repository:         params.Repository,
		dispatchController: params.DispatchController,
		realtimePublisher:  params.RealtimePublisher,
		logger:             logger,
	}
}

type TransitionCommand struct {
	OrderID       uuid.UUID
	ToStatus      domain.OrderStatus
	ActorUserID   *uuid.UUID
	ActorDriverID *uuid.UUID
	Reason        string
}

func (service *Service) Transition(ctx context.Context, command TransitionCommand) (domain.Order, error) {
	currentOrder, err := service.repository.GetOrderByID(ctx, command.OrderID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("get order for transition: %w", err)
	}

	transition, err := domain.NewOrderTransition(
		currentOrder,
		command.ToStatus,
		command.ActorUserID,
		command.ActorDriverID,
		command.Reason,
		time.Now().UTC(),
	)
	if err != nil {
		return domain.Order{}, fmt.Errorf("validate order transition: %w", err)
	}

	updatedOrder, changed, err := service.repository.TransitionOrderStatus(ctx, transition)
	if err != nil {
		return domain.Order{}, fmt.Errorf("persist order transition: %w", err)
	}
	if !changed {
		return domain.Order{}, ErrOrderConcurrentUpdate
	}

	event := OrderEvent{
		OrderID:       updatedOrder.ID,
		ActorUserID:   command.ActorUserID,
		ActorDriverID: command.ActorDriverID,
		EventType:     domain.EventTypeForOrderStatus(command.ToStatus),
		Payload: map[string]any{
			"order_id":    updatedOrder.ID,
			"from_status": transition.FromStatus,
			"to_status":   transition.ToStatus,
			"version":     updatedOrder.Version,
			"reason":      command.Reason,
			"occurred_at": transition.OccurredAt,
		},
	}
	if err := service.repository.AddStateEvent(ctx, event); err != nil {
		return domain.Order{}, fmt.Errorf("persist order transition event: %w", err)
	}

	if command.ToStatus == domain.OrderStatusCancelled && service.dispatchController != nil {
		if err := service.dispatchController.StopDispatch(ctx, updatedOrder.ID); err != nil {
			return domain.Order{}, fmt.Errorf("stop dispatch after cancellation: %w", err)
		}
	}

	if err := service.publishState(ctx, updatedOrder, event.EventType); err != nil {
		return domain.Order{}, err
	}

	service.logger.Info(
		"order state transitioned",
		zap.String("order_id", updatedOrder.ID.String()),
		zap.String("from_status", string(transition.FromStatus)),
		zap.String("to_status", string(transition.ToStatus)),
		zap.Int("version", updatedOrder.Version),
	)

	return updatedOrder, nil
}

func (service *Service) CurrentForPassenger(ctx context.Context, passengerID uuid.UUID) (domain.Order, error) {
	order, err := service.repository.GetCurrentOrderByPassengerID(ctx, passengerID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("get current passenger order: %w", err)
	}
	return order, nil
}

func (service *Service) CurrentForDriver(ctx context.Context, driverID uuid.UUID) (domain.Order, error) {
	order, err := service.repository.GetCurrentOrderByDriverID(ctx, driverID)
	if err != nil {
		return domain.Order{}, fmt.Errorf("get current driver order: %w", err)
	}
	return order, nil
}

func (service *Service) publishState(ctx context.Context, order domain.Order, eventType domain.OrderEventType) error {
	payload := map[string]any{
		"order_id":  order.ID,
		"status":    order.Status,
		"version":   order.Version,
		"driver_id": order.DriverID,
	}

	if service.realtimePublisher == nil {
		return nil
	}
	if err := service.realtimePublisher.SendToPassenger(ctx, order.PassengerID, string(eventType), payload); err != nil {
		return fmt.Errorf("publish order state to passenger: %w", err)
	}
	if order.DriverID != nil {
		if err := service.realtimePublisher.SendToDriver(ctx, *order.DriverID, string(eventType), payload); err != nil {
			return fmt.Errorf("publish order state to driver: %w", err)
		}
	}

	return nil
}

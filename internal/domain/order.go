package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderStatusCreated        OrderStatus = "created"
	OrderStatusSearching      OrderStatus = "searching"
	OrderStatusDriverAssigned OrderStatus = "driver_assigned"
	OrderStatusDriverArriving OrderStatus = "driver_arriving"
	OrderStatusDriverWaiting  OrderStatus = "driver_waiting"
	OrderStatusInProgress     OrderStatus = "in_progress"
	OrderStatusCompleted      OrderStatus = "completed"
	OrderStatusCancelled      OrderStatus = "cancelled"
	OrderStatusFailed         OrderStatus = "failed"
)

type PaymentMethod string

const (
	PaymentMethodCash      PaymentMethod = "cash"
	PaymentMethodCard      PaymentMethod = "card"
	PaymentMethodCorporate PaymentMethod = "corporate"
)

type OrderEventType string

const (
	OrderEventCreated     OrderEventType = "order.created"
	OrderEventSearching   OrderEventType = "order.searching"
	OrderEventOffer       OrderEventType = "order.offer"
	OrderEventAccepted    OrderEventType = "order.accepted"
	OrderEventRejected    OrderEventType = "order.rejected"
	OrderEventCancelled   OrderEventType = "order.cancelled"
	OrderEventLocation    OrderEventType = "driver.location"
	OrderEventArrived     OrderEventType = "driver.arrived"
	OrderEventTripStarted OrderEventType = "trip.started"
	OrderEventCompleted   OrderEventType = "trip.completed"
	OrderEventFailed      OrderEventType = "trip.failed"
)

type Order struct {
	ID                  uuid.UUID
	PassengerID         uuid.UUID
	DriverID            *uuid.UUID
	CityID              uuid.UUID
	TariffID            *uuid.UUID
	Status              OrderStatus
	PickupAddress       string
	PickupLocation      Coordinates
	DestinationAddress  string
	DestinationLocation *Coordinates
	RequestedAt         time.Time
	AcceptedAt          *time.Time
	StartedAt           *time.Time
	CompletedAt         *time.Time
	CancelledAt         *time.Time
	CancellationReason  string
	EstimatedPrice      *Money
	FinalPrice          *Money
	PaymentMethod       PaymentMethod
	PassengerComment    string
	DispatchAttempt     int
	Version             int
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           *time.Time
}

type OrderRating struct {
	ID          uuid.UUID
	OrderID     uuid.UUID
	PassengerID uuid.UUID
	DriverID    uuid.UUID
	Score       int
	Comment     string
	CreatedAt   time.Time
}

var (
	ErrInvalidOrderStatus           = errors.New("invalid order status")
	ErrInvalidOrderStatusTransition = errors.New("invalid order status transition")
	ErrInvalidPaymentMethod         = errors.New("invalid payment method")
	ErrInvalidRatingScore           = errors.New("invalid rating score")
)

var terminalOrderStatuses = map[OrderStatus]struct{}{
	OrderStatusCompleted: {},
	OrderStatusCancelled: {},
	OrderStatusFailed:    {},
}

var allowedOrderStatusTransitions = map[OrderStatus][]OrderStatus{
	OrderStatusCreated:        {OrderStatusSearching},
	OrderStatusSearching:      {OrderStatusDriverAssigned, OrderStatusCancelled, OrderStatusFailed},
	OrderStatusDriverAssigned: {OrderStatusDriverArriving, OrderStatusCancelled},
	OrderStatusDriverArriving: {OrderStatusDriverWaiting, OrderStatusCancelled},
	OrderStatusDriverWaiting:  {OrderStatusInProgress, OrderStatusCancelled},
	OrderStatusInProgress:     {OrderStatusCompleted},
	OrderStatusCompleted:      {},
	OrderStatusCancelled:      {},
	OrderStatusFailed:         {},
}

type OrderTransition struct {
	OrderID         uuid.UUID
	FromStatus      OrderStatus
	ToStatus        OrderStatus
	ExpectedVersion int
	ActorUserID     *uuid.UUID
	ActorDriverID   *uuid.UUID
	Reason          string
	OccurredAt      time.Time
}

func (status OrderStatus) Validate() error {
	switch status {
	case OrderStatusCreated,
		OrderStatusSearching,
		OrderStatusDriverAssigned,
		OrderStatusDriverArriving,
		OrderStatusDriverWaiting,
		OrderStatusInProgress,
		OrderStatusCompleted,
		OrderStatusCancelled,
		OrderStatusFailed:
		return nil
	default:
		return ErrInvalidOrderStatus
	}
}

func (status OrderStatus) IsTerminal() bool {
	_, exists := terminalOrderStatuses[status]
	return exists
}

func CanTransitionOrderStatus(from OrderStatus, to OrderStatus) bool {
	if from == to {
		return true
	}

	for _, allowedStatus := range allowedOrderStatusTransitions[from] {
		if allowedStatus == to {
			return true
		}
	}

	return false
}

func EnsureOrderStatusTransition(from OrderStatus, to OrderStatus) error {
	if err := from.Validate(); err != nil {
		return fmt.Errorf("validate source order status: %w", err)
	}
	if err := to.Validate(); err != nil {
		return fmt.Errorf("validate target order status: %w", err)
	}
	if !CanTransitionOrderStatus(from, to) {
		return fmt.Errorf("%w: %s -> %s", ErrInvalidOrderStatusTransition, from, to)
	}

	return nil
}

func NewOrderTransition(order Order, toStatus OrderStatus, actorUserID *uuid.UUID, actorDriverID *uuid.UUID, reason string, occurredAt time.Time) (OrderTransition, error) {
	if err := EnsureOrderStatusTransition(order.Status, toStatus); err != nil {
		return OrderTransition{}, err
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}

	return OrderTransition{
		OrderID:         order.ID,
		FromStatus:      order.Status,
		ToStatus:        toStatus,
		ExpectedVersion: order.Version,
		ActorUserID:     actorUserID,
		ActorDriverID:   actorDriverID,
		Reason:          reason,
		OccurredAt:      occurredAt,
	}, nil
}

func EventTypeForOrderStatus(status OrderStatus) OrderEventType {
	switch status {
	case OrderStatusSearching:
		return OrderEventSearching
	case OrderStatusDriverAssigned:
		return OrderEventAccepted
	case OrderStatusDriverWaiting:
		return OrderEventArrived
	case OrderStatusInProgress:
		return OrderEventTripStarted
	case OrderStatusCompleted:
		return OrderEventCompleted
	case OrderStatusCancelled:
		return OrderEventCancelled
	case OrderStatusFailed:
		return OrderEventFailed
	default:
		return OrderEventCreated
	}
}

func (paymentMethod PaymentMethod) Validate() error {
	switch paymentMethod {
	case PaymentMethodCash, PaymentMethodCard, PaymentMethodCorporate:
		return nil
	default:
		return ErrInvalidPaymentMethod
	}
}

func NewOrderRating(orderID uuid.UUID, passengerID uuid.UUID, driverID uuid.UUID, score int, comment string) (OrderRating, error) {
	if score < 1 || score > 5 {
		return OrderRating{}, ErrInvalidRatingScore
	}

	return OrderRating{
		OrderID:     orderID,
		PassengerID: passengerID,
		DriverID:    driverID,
		Score:       score,
		Comment:     comment,
	}, nil
}

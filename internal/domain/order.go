package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type OrderStatus string
type OrderType string
type ScheduledOrderStatus string

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

const (
	OrderTypeInstant   OrderType = "instant"
	OrderTypeScheduled OrderType = "scheduled"
)

const (
	ScheduledOrderStatusNew               ScheduledOrderStatus = "scheduled_new"
	ScheduledOrderStatusConfirmed         ScheduledOrderStatus = "scheduled_confirmed"
	ScheduledOrderStatusDriverAssigned    ScheduledOrderStatus = "scheduled_driver_assigned"
	ScheduledOrderStatusWaitingActivation ScheduledOrderStatus = "scheduled_waiting_activation"
	ScheduledOrderStatusActivated         ScheduledOrderStatus = "scheduled_activated"
	ScheduledOrderStatusCancelled         ScheduledOrderStatus = "scheduled_cancelled"
	ScheduledOrderStatusExpired           ScheduledOrderStatus = "scheduled_expired"
	ScheduledOrderStatusFailed            ScheduledOrderStatus = "scheduled_failed"
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
	OrderEventUpdated     OrderEventType = "order.updated"
	OrderEventCancelled   OrderEventType = "order.cancelled"
	OrderEventLocation    OrderEventType = "driver.location"
	OrderEventArriving    OrderEventType = "driver.arriving"
	OrderEventArrived     OrderEventType = "driver.arrived"
	OrderEventTripStarted OrderEventType = "trip.started"
	OrderEventCompleted   OrderEventType = "trip.completed"
	OrderEventFailed      OrderEventType = "trip.failed"
)

type Order struct {
	ID                              uuid.UUID
	PassengerID                     uuid.UUID
	DriverID                        *uuid.UUID
	CarID                           *uuid.UUID
	ParkID                          *uuid.UUID
	PreassignedDriverID             *uuid.UUID
	CityID                          uuid.UUID
	TariffID                        *uuid.UUID
	AssignedTariffID                *uuid.UUID
	CarClassID                      *uuid.UUID
	Status                          OrderStatus
	OrderType                       OrderType
	ScheduledStatus                 *ScheduledOrderStatus
	PickupAddress                   string
	PickupEntrance                  string
	PickupComment                   string
	PickupLocation                  Coordinates
	DestinationAddress              string
	DestinationLocation             *Coordinates
	ScheduledAt                     *time.Time
	ActivationAt                    *time.Time
	ScheduledTimezone               string
	RequestedAt                     time.Time
	AcceptedAt                      *time.Time
	StartedAt                       *time.Time
	CompletedAt                     *time.Time
	CancelledAt                     *time.Time
	ActivatedAt                     *time.Time
	ScheduledCancelledAt            *time.Time
	ScheduledExpiredAt              *time.Time
	CancellationReason              string
	ScheduledCancelReason           string
	EstimatedPrice                  *Money
	FinalPrice                      *Money
	ActualDistanceMeters            *int64
	ActualDurationSeconds           *int64
	PaymentMethod                   PaymentMethod
	PassengerComment                string
	PassengerLocationSharingEnabled bool
	DispatchAttempt                 int
	ScheduledCreatedBy              *uuid.UUID
	Version                         int
	CreatedAt                       time.Time
	UpdatedAt                       time.Time
	DeletedAt                       *time.Time
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
	ErrInvalidOrderType             = errors.New("invalid order type")
	ErrInvalidScheduledOrderStatus  = errors.New("invalid scheduled order status")
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

func (orderType OrderType) Validate() error {
	switch orderType {
	case OrderTypeInstant, OrderTypeScheduled:
		return nil
	default:
		return ErrInvalidOrderType
	}
}

func (status ScheduledOrderStatus) Validate() error {
	switch status {
	case ScheduledOrderStatusNew,
		ScheduledOrderStatusConfirmed,
		ScheduledOrderStatusDriverAssigned,
		ScheduledOrderStatusWaitingActivation,
		ScheduledOrderStatusActivated,
		ScheduledOrderStatusCancelled,
		ScheduledOrderStatusExpired,
		ScheduledOrderStatusFailed:
		return nil
	default:
		return ErrInvalidScheduledOrderStatus
	}
}

func (status ScheduledOrderStatus) IsTerminal() bool {
	switch status {
	case ScheduledOrderStatusActivated,
		ScheduledOrderStatusCancelled,
		ScheduledOrderStatusExpired,
		ScheduledOrderStatusFailed:
		return true
	default:
		return false
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
	case OrderStatusDriverArriving:
		return OrderEventArriving
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

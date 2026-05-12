package dispatch

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/develoop/taxi-platform/internal/domain"
)

type OrderRepository interface {
	GetOrderByID(ctx context.Context, orderID uuid.UUID) (domain.Order, error)
	MarkOrderSearching(ctx context.Context, orderID uuid.UUID) error
	AssignDriver(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID, acceptedAt time.Time) (bool, error)
	IncrementDispatchAttempt(ctx context.Context, orderID uuid.UUID) error
	FailOrder(ctx context.Context, orderID uuid.UUID, reason string) error
	AddOrderEvent(ctx context.Context, event OrderEvent) error
}

type DriverSearchRepository interface {
	FindNearestOnlineDrivers(ctx context.Context, query NearestDriversQuery) ([]DriverCandidate, error)
}

type DriverStateRepository interface {
	MarkDriverBusy(ctx context.Context, driverID uuid.UUID) error
}

type OfferStore interface {
	SaveOffer(ctx context.Context, offer OrderOffer, ttl time.Duration) error
	GetOffer(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID) (OrderOffer, bool, error)
	ListOfferedDriverIDs(ctx context.Context, orderID uuid.UUID) ([]uuid.UUID, error)
	RemoveOffers(ctx context.Context, orderID uuid.UUID) error
}

type DispatchStateStore interface {
	BeginDispatch(ctx context.Context, orderID uuid.UUID, ttl time.Duration) (bool, error)
	FinishDispatch(ctx context.Context, orderID uuid.UUID) error
	MarkActiveOffer(ctx context.Context, orderID uuid.UUID, ttl time.Duration) error
	MarkAcceptedDriver(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID, ttl time.Duration) error
}

type TaskQueue interface {
	Publish(ctx context.Context, task DispatchTask) error
	Consume(ctx context.Context, timeout time.Duration) (DispatchTask, bool, error)
}

type TimeoutQueue interface {
	Schedule(ctx context.Context, task DispatchTask, runAt time.Time) error
	Due(ctx context.Context, now time.Time, limit int) ([]DispatchTask, error)
	Remove(ctx context.Context, task DispatchTask) error
}

type RecoveryRepository interface {
	ListSearchingOrders(ctx context.Context, limit int) ([]uuid.UUID, error)
}

type Metrics interface {
	ObserveDispatchDuration(duration time.Duration)
	ObserveDriverAcceptTime(duration time.Duration)
	ObserveDispatchRadiusAttempt(radiusMeters int)
	IncrementFailedDispatches()
	IncrementDispatchTimeouts()
	SetActiveSearches(count int)
	SetActiveOrders(count int)
	SetStaleDrivers(count int)
	SetWSConnections(count int)
	IncrementReconnects()
}

type LockManager interface {
	Acquire(ctx context.Context, key string, ttl time.Duration) (Lock, bool, error)
}

type Lock interface {
	Release(ctx context.Context) error
}

type RealtimeGateway interface {
	SendToDriver(ctx context.Context, driverID uuid.UUID, eventName string, payload any) error
	SendToPassenger(ctx context.Context, passengerID uuid.UUID, eventName string, payload any) error
}

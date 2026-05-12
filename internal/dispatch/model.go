package dispatch

import (
	"time"

	"github.com/google/uuid"

	"github.com/develoop/taxi-platform/internal/domain"
)

const (
	EventOrderOffer              = "order.offer"
	EventOrderCancelled          = "order.cancelled"
	EventOrderAssigned           = "order.assigned"
	EventOrderDispatchSearching  = "order.searching"
	EventOrderNoDriversFound     = "no_drivers_found"
	EventOrderExpired            = "order.expired"
	EventPassengerDriverAssigned = "driver_assigned"
)

type Config struct {
	InitialRadiusMeters  int
	MaxRadiusMeters      int
	RadiusStepMeters     int
	RadiusAttemptsMeters []int
	MaxDriversPerOffer   int
	DriverLocationMaxAge time.Duration
	OfferTTL             time.Duration
	AcceptLockTTL        time.Duration
	WorkerPollTimeout    time.Duration
	RecoveryInterval     time.Duration
}

type NearestDriversQuery struct {
	CityID         uuid.UUID
	Pickup         domain.Coordinates
	RadiusMeters   int
	Limit          int
	ExcludeIDs     []uuid.UUID
	LocationMaxAge time.Duration
}

type DriverCandidate struct {
	DriverID       uuid.UUID
	DistanceMeters float64
	Location       domain.Coordinates
}

type OrderOffer struct {
	OrderID        uuid.UUID
	DriverID       uuid.UUID
	Attempt        int
	RadiusMeters   int
	DistanceMeters float64
	ExpiresAt      time.Time
	CreatedAt      time.Time
}

type DispatchTask struct {
	OrderID  uuid.UUID `json:"order_id"`
	Attempt  int       `json:"attempt"`
	QueuedAt time.Time `json:"queued_at"`
}

type OrderEvent struct {
	OrderID       uuid.UUID
	ActorUserID   *uuid.UUID
	ActorDriverID *uuid.UUID
	EventType     domain.OrderEventType
	Payload       map[string]any
	CreatedAt     time.Time
}

type DispatchResult struct {
	OrderID            uuid.UUID
	Status             domain.OrderStatus
	Attempt            int
	RadiusMeters       int
	OfferedDriverIDs   []uuid.UUID
	NextRetryAfter     time.Duration
	NoDriversAvailable bool
}

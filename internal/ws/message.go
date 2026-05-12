package ws

import (
	"time"

	"github.com/google/uuid"
)

const (
	EventSyncRequired = "sync.required"

	EventOrderSearching      = "order.searching"
	EventOrderDriverAssigned = "order.driver_assigned"
	EventOrderDriverArriving = "order.driver_arriving"
	EventOrderDriverWaiting  = "order.driver_waiting"
	EventOrderTripStarted    = "order.trip_started"
	EventOrderCompleted      = "order.completed"
	EventOrderCancelled      = "order.cancelled"
	EventOrderFailed         = "order.failed"
	EventDriverLocation      = "driver.location_updated"

	EventOrderOffer          = "order.offer"
	EventOrderOfferExpired   = "order.offer_expired"
	EventOrderOfferCancelled = "order.offer_cancelled"
	EventOrderAccepted       = "order.accepted"
	EventPassengerCancelled  = "passenger.cancelled"
)

type Message struct {
	Event      string         `json:"event" example:"order.driver_assigned"`
	RequestID  uuid.UUID      `json:"request_id" example:"11111111-1111-1111-1111-111111111111"`
	OccurredAt time.Time      `json:"occurred_at" example:"2026-05-12T12:00:00Z"`
	Payload    map[string]any `json:"payload"`
}

func NewMessage(event string, requestID uuid.UUID, payload map[string]any) Message {
	return Message{
		Event:      event,
		RequestID:  requestID,
		OccurredAt: time.Now().UTC(),
		Payload:    payload,
	}
}

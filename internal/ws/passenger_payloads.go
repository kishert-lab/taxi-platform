package ws

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type PassengerOrderStatePayload struct {
	OrderID            uuid.UUID          `json:"order_id" example:"11111111-1111-1111-1111-111111111111"`
	Status             domain.OrderStatus `json:"status" example:"driver_arriving"`
	Version            int                `json:"version" example:"3"`
	DriverID           *uuid.UUID         `json:"driver_id,omitempty" example:"22222222-2222-2222-2222-222222222222"`
	CancelledBy        string             `json:"cancelled_by,omitempty" example:"driver"`
	CancellationReason string             `json:"cancellation_reason,omitempty" example:"driver unavailable"`
}

type PassengerDriverAssignedPayload struct {
	OrderID  uuid.UUID `json:"order_id" example:"11111111-1111-1111-1111-111111111111"`
	DriverID uuid.UUID `json:"driver_id" example:"22222222-2222-2222-2222-222222222222"`
}

type PassengerNoDriversPayload struct {
	OrderID uuid.UUID `json:"order_id" example:"11111111-1111-1111-1111-111111111111"`
}

type PassengerDriverLocationPayload struct {
	OrderID        uuid.UUID               `json:"order_id" example:"11111111-1111-1111-1111-111111111111"`
	DriverID       uuid.UUID               `json:"driver_id" example:"22222222-2222-2222-2222-222222222222"`
	Status         domain.OrderStatus      `json:"status" example:"driver_arriving"`
	Location       dto.CoordinatesResponse `json:"location"`
	Heading        *int16                  `json:"heading,omitempty" example:"180"`
	SpeedMPS       *float64                `json:"speed_mps,omitempty" example:"7.5"`
	AccuracyMeters *float64                `json:"accuracy_meters,omitempty" example:"8"`
	RecordedAt     time.Time               `json:"recorded_at" example:"2026-06-29T10:00:00Z"`
}

type PassengerChatMessagePayload struct {
	Message dto.ChatMessageResponse `json:"message"`
}

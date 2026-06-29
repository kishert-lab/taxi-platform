package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TaxiParkCreateScheduledOrderRequest struct {
	PassengerPhone      string                           `json:"passenger_phone,omitempty" example:"+79990000000"`
	PassengerName       string                           `json:"passenger_name,omitempty" example:"Ivan"`
	TariffID            uuid.UUID                        `json:"tariff_id" binding:"required" example:"4e99f0be-5ea1-4353-8307-d1555f588825"`
	PickupAddress       string                           `json:"pickup_address" binding:"required" example:"Lenina 10"`
	PickupLocation      *TaxiParkOrderCoordinatesRequest `json:"pickup_location" binding:"required"`
	DestinationAddress  string                           `json:"destination_address" binding:"required" example:"Airport"`
	DestinationLocation *TaxiParkOrderCoordinatesRequest `json:"destination_location,omitempty"`
	PaymentType         domain.PaymentMethod             `json:"payment_type,omitempty" example:"cash"`
	PaymentMethod       domain.PaymentMethod             `json:"payment_method,omitempty" example:"cash"`
	Comment             string                           `json:"comment,omitempty" example:"Call 5 minutes before"`
	ScheduledAt         time.Time                        `json:"scheduled_at" binding:"required" example:"2026-07-01T08:30:00+05:00"`
	Timezone            string                           `json:"timezone,omitempty" example:"Asia/Yekaterinburg"`
	PreassignedDriverID *uuid.UUID                       `json:"preassigned_driver_id,omitempty" example:"11111111-1111-1111-1111-111111111111"`
}

type TaxiParkUpdateScheduledOrderRequest struct {
	PickupAddress       *string                          `json:"pickup_address,omitempty" example:"Lenina 10"`
	PickupLocation      *TaxiParkOrderCoordinatesRequest `json:"pickup_location,omitempty"`
	DestinationAddress  *string                          `json:"destination_address,omitempty" example:"Airport"`
	DestinationLocation *TaxiParkOrderCoordinatesRequest `json:"destination_location,omitempty"`
	PaymentType         *domain.PaymentMethod            `json:"payment_type,omitempty" example:"cash"`
	PaymentMethod       *domain.PaymentMethod            `json:"payment_method,omitempty" example:"cash"`
	Comment             *string                          `json:"comment,omitempty" example:"Call 5 minutes before"`
	ScheduledAt         *time.Time                       `json:"scheduled_at,omitempty" example:"2026-07-01T08:30:00+05:00"`
	Timezone            *string                          `json:"timezone,omitempty" example:"Asia/Yekaterinburg"`
	PreassignedDriverID *uuid.UUID                       `json:"preassigned_driver_id,omitempty" example:"11111111-1111-1111-1111-111111111111"`
}

type AssignScheduledOrderDriverRequest struct {
	DriverID uuid.UUID `json:"driver_id" binding:"required" example:"11111111-1111-1111-1111-111111111111"`
}

type ScheduledOrderResponse struct {
	ID                    uuid.UUID                   `json:"id" example:"11111111-1111-1111-1111-111111111111"`
	OrderType             domain.OrderType            `json:"order_type" example:"scheduled"`
	Status                domain.OrderStatus          `json:"status" example:"created"`
	ScheduledStatus       domain.ScheduledOrderStatus `json:"scheduled_status" example:"scheduled_confirmed"`
	PassengerID           uuid.UUID                   `json:"passenger_id" example:"22222222-2222-2222-2222-222222222222"`
	DriverID              *uuid.UUID                  `json:"driver_id,omitempty" example:"33333333-3333-3333-3333-333333333333"`
	PreassignedDriverID   *uuid.UUID                  `json:"preassigned_driver_id,omitempty" example:"33333333-3333-3333-3333-333333333333"`
	TariffID              *uuid.UUID                  `json:"tariff_id,omitempty" example:"44444444-4444-4444-4444-444444444444"`
	CityID                uuid.UUID                   `json:"city_id" example:"55555555-5555-5555-5555-555555555555"`
	PickupAddress         string                      `json:"pickup_address" example:"Lenina 10"`
	PickupLocation        CoordinatesResponse         `json:"pickup_location"`
	DestinationAddress    string                      `json:"destination_address" example:"Airport"`
	DestinationLocation   *CoordinatesResponse        `json:"destination_location,omitempty"`
	PaymentMethod         domain.PaymentMethod        `json:"payment_method" example:"cash"`
	Comment               string                      `json:"comment,omitempty" example:"Call 5 minutes before"`
	ScheduledAt           time.Time                   `json:"scheduled_at" example:"2026-07-01T08:30:00+05:00"`
	ActivationAt          time.Time                   `json:"activation_at" example:"2026-07-01T08:10:00+05:00"`
	ScheduledTimezone     string                      `json:"scheduled_timezone" example:"Asia/Yekaterinburg"`
	ActivatedAt           *time.Time                  `json:"activated_at,omitempty" example:"2026-07-01T08:10:00+05:00"`
	ScheduledCancelledAt  *time.Time                  `json:"scheduled_cancelled_at,omitempty" example:"2026-07-01T07:00:00+05:00"`
	ScheduledExpiredAt    *time.Time                  `json:"scheduled_expired_at,omitempty" example:"2026-07-01T08:45:00+05:00"`
	ScheduledCancelReason string                      `json:"scheduled_cancel_reason,omitempty" example:"Passenger cancelled"`
	CreatedAt             time.Time                   `json:"created_at" example:"2026-06-28T12:00:00Z"`
	UpdatedAt             time.Time                   `json:"updated_at" example:"2026-06-28T12:00:00Z"`
}

type ScheduledOrdersResponse struct {
	Orders []ScheduledOrderResponse `json:"orders"`
}

package dto

import (
	"github.com/google/uuid"

	"github.com/develoop/taxi-platform/internal/domain"
)

type CoordinatesRequest struct {
	Latitude  float64 `json:"latitude" binding:"required,min=-90,max=90" example:"56.838011"`
	Longitude float64 `json:"longitude" binding:"required,min=-180,max=180" example:"60.597465"`
}

type EstimateOrderPriceRequest struct {
	CityID              uuid.UUID          `json:"city_id" binding:"required" example:"11111111-1111-1111-1111-111111111111"`
	TariffID            uuid.UUID          `json:"tariff_id" binding:"required" example:"22222222-2222-2222-2222-222222222222"`
	PickupLocation      CoordinatesRequest `json:"pickup_location" binding:"required"`
	DestinationLocation CoordinatesRequest `json:"destination_location" binding:"required"`
}

type EstimateOrderPriceResponse struct {
	EstimatedPrice  MoneyResponse `json:"estimated_price"`
	DistanceMeters  int64         `json:"distance_meters" example:"4300"`
	DurationSeconds int64         `json:"duration_seconds" example:"720"`
}

type CreateOrderRequest struct {
	CityID              uuid.UUID            `json:"city_id" binding:"required" example:"11111111-1111-1111-1111-111111111111"`
	TariffID            uuid.UUID            `json:"tariff_id" binding:"required" example:"22222222-2222-2222-2222-222222222222"`
	PickupAddress       string               `json:"pickup_address" binding:"required" example:"Lenina 1"`
	PickupLocation      CoordinatesRequest   `json:"pickup_location" binding:"required"`
	DestinationAddress  string               `json:"destination_address" binding:"required" example:"Mira 10"`
	DestinationLocation CoordinatesRequest   `json:"destination_location" binding:"required"`
	PaymentMethod       domain.PaymentMethod `json:"payment_method" binding:"required,oneof=cash card corporate" example:"cash"`
	PassengerComment    string               `json:"passenger_comment" example:"Entrance 2"`
}

type OrderResponse struct {
	ID                 uuid.UUID            `json:"id"`
	PassengerID        uuid.UUID            `json:"passenger_id"`
	DriverID           *uuid.UUID           `json:"driver_id,omitempty"`
	CityID             uuid.UUID            `json:"city_id"`
	TariffID           *uuid.UUID           `json:"tariff_id,omitempty"`
	Status             domain.OrderStatus   `json:"status" example:"searching"`
	Version            int                  `json:"version" example:"2"`
	PickupAddress      string               `json:"pickup_address"`
	PickupLocation     CoordinatesResponse  `json:"pickup_location"`
	DestinationAddress string               `json:"destination_address"`
	EstimatedPrice     *MoneyResponse       `json:"estimated_price,omitempty"`
	FinalPrice         *MoneyResponse       `json:"final_price,omitempty"`
	PaymentMethod      domain.PaymentMethod `json:"payment_method" example:"cash"`
}

type CurrentOrderResponse struct {
	Order OrderResponse `json:"order"`
}

type CoordinatesResponse struct {
	Latitude  float64 `json:"latitude" example:"56.838011"`
	Longitude float64 `json:"longitude" example:"60.597465"`
}

type MoneyResponse struct {
	Amount   int64  `json:"amount" example:"25000"`
	Currency string `json:"currency" example:"RUB"`
}

type CancelOrderRequest struct {
	Reason string `json:"reason" binding:"required" example:"Passenger changed plans"`
}

type RateOrderRequest struct {
	Score   int    `json:"score" binding:"required,min=1,max=5" example:"5"`
	Comment string `json:"comment" example:"Good driver"`
}

type DriverStatusRequest struct {
	Status domain.DriverStatus `json:"status" binding:"required,oneof=offline online busy paused blocked" example:"online"`
}

type DriverLocationRequest struct {
	Location       CoordinatesRequest `json:"location" binding:"required"`
	Heading        *int16             `json:"heading,omitempty" binding:"omitempty,min=0,max=359" example:"90"`
	SpeedMPS       *float64           `json:"speed_mps,omitempty" binding:"omitempty,min=0" example:"8.3"`
	AccuracyMeters *float64           `json:"accuracy_meters,omitempty" binding:"omitempty,min=0" example:"12.5"`
}

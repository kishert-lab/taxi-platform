package dto

import (
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TaxiParkOrderCoordinatesRequest struct {
	Latitude  float64 `json:"latitude" example:"56.835128"`
	Longitude float64 `json:"longitude" example:"60.598698"`
}

type TaxiParkCreateOrderRequest struct {
	PassengerPhone      string                           `json:"passenger_phone,omitempty" example:"+79990000000"`
	PassengerName       string                           `json:"passenger_name,omitempty" example:"Irina"`
	TariffID            uuid.UUID                        `json:"tariff_id" binding:"required" example:"4e99f0be-5ea1-4353-8307-d1555f588825"`
	PickupAddress       string                           `json:"pickup_address" example:"Mira 8"`
	PickupLocation      *TaxiParkOrderCoordinatesRequest `json:"pickup_location"`
	DestinationAddress  string                           `json:"destination_address" example:"Lenina 50"`
	DestinationLocation *TaxiParkOrderCoordinatesRequest `json:"destination_location"`
	PaymentType         domain.PaymentMethod             `json:"payment_type,omitempty" example:"cash"`
	PaymentMethod       domain.PaymentMethod             `json:"payment_method,omitempty" example:"cash"`
	Comment             string                           `json:"comment,omitempty" example:"Entrance 2"`
}

type TaxiParkUpdateOrderRequest struct {
	PickupAddress       *string                          `json:"pickup_address,omitempty" example:"Mira 8"`
	PickupLocation      *TaxiParkOrderCoordinatesRequest `json:"pickup_location,omitempty"`
	DestinationAddress  *string                          `json:"destination_address,omitempty" example:"Lenina 50"`
	DestinationLocation *TaxiParkOrderCoordinatesRequest `json:"destination_location,omitempty"`
	PaymentType         *domain.PaymentMethod            `json:"payment_type,omitempty" example:"cash"`
	PaymentMethod       *domain.PaymentMethod            `json:"payment_method,omitempty" example:"cash"`
	Comment             *string                          `json:"comment,omitempty" example:"Entrance 2"`
}

type TaxiParkCompleteOrderRequest struct {
	FinalPrice int64  `json:"final_price" binding:"required,min=0" example:"25000"`
	Currency   string `json:"currency" binding:"required" example:"RUB"`
}

package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TaxiParkDriverLocationCarResponse struct {
	ID                 uuid.UUID                          `json:"id" example:"55555555-5555-5555-5555-555555555555"`
	Brand              string                             `json:"brand" example:"Lada"`
	Model              string                             `json:"model" example:"Vesta"`
	PlateNumber        string                             `json:"plate_number" example:"A001AA196"`
	Color              string                             `json:"color,omitempty" example:"White"`
	CarClass           string                             `json:"car_class,omitempty" example:"economy"`
	VerificationStatus domain.VerificationLifecycleStatus `json:"verification_status" example:"verified"`
	IsActive           bool                               `json:"is_active" example:"true"`
}

type TaxiParkDriverLocationResponse struct {
	DriverID           uuid.UUID                          `json:"driver_id" example:"22222222-2222-2222-2222-222222222222"`
	UserID             uuid.UUID                          `json:"user_id" example:"66666666-6666-6666-6666-666666666666"`
	Name               string                             `json:"name" example:"Ivan Petrov"`
	Phone              string                             `json:"phone" example:"+79990000001"`
	Status             domain.DriverStatus                `json:"status" example:"online"`
	VerificationStatus domain.VerificationLifecycleStatus `json:"verification_status" example:"verified"`
	Rating             float64                            `json:"rating" example:"4.95"`
	Location           *CoordinatesResponse               `json:"location,omitempty"`
	Heading            *int16                             `json:"heading,omitempty" example:"90"`
	SpeedMPS           *float64                           `json:"speed_mps,omitempty" example:"8.3"`
	AccuracyMeters     *float64                           `json:"accuracy_meters,omitempty" example:"12.5"`
	RecordedAt         *time.Time                         `json:"recorded_at,omitempty" example:"2026-05-12T12:00:00Z"`
	UpdatedAt          *time.Time                         `json:"updated_at,omitempty" example:"2026-05-12T12:00:00Z"`
	IsStale            bool                               `json:"is_stale" example:"false"`
	Car                *TaxiParkDriverLocationCarResponse `json:"car,omitempty"`
}

type TaxiParkDriverLocationsResponse struct {
	Drivers []TaxiParkDriverLocationResponse `json:"drivers"`
}

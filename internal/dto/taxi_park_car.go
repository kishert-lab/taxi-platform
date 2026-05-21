package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TaxiParkCarRequest struct {
	PrimaryDriverID               *uuid.UUID                         `json:"primary_driver_id,omitempty" example:"22222222-2222-2222-2222-222222222222"`
	AttachedDriverIDs             []uuid.UUID                        `json:"attached_driver_ids,omitempty"`
	Brand                         string                             `json:"brand" binding:"required" example:"Lada"`
	Model                         string                             `json:"model" binding:"required" example:"Vesta"`
	Year                          int                                `json:"year,omitempty" example:"2023"`
	PlateNumber                   string                             `json:"plate_number" binding:"required" example:"A001AA196"`
	VIN                           string                             `json:"vin,omitempty" example:"XTA00000000000000"`
	STS                           string                             `json:"sts,omitempty" example:"9911000000"`
	PTS                           string                             `json:"pts,omitempty" example:"77AA000000"`
	Color                         string                             `json:"color" binding:"required" example:"White"`
	CarClass                      string                             `json:"car_class,omitempty" example:"economy"`
	VerificationStatus            domain.VerificationLifecycleStatus `json:"verification_status,omitempty" binding:"omitempty,oneof=draft pending_verification verified rejected blocked archived" example:"pending_verification"`
	OwnerDetails                  string                             `json:"owner_details,omitempty" example:"Owned by taxi park"`
	OwnerOrLegalBasis             string                             `json:"owner_or_legal_basis,omitempty" example:"Lease agreement"`
	OSAGOExpiresAt                string                             `json:"osago_expires_at,omitempty" example:"2027-01-31"`
	DiagnosticCardExpiresAt       string                             `json:"diagnostic_card_expires_at,omitempty" example:"2027-01-31"`
	TaxiPermitNumber              string                             `json:"taxi_permit_number,omitempty" example:"TAXI-66-000001"`
	RegionalRegistryNumber        string                             `json:"regional_registry_number,omitempty" example:"66-123456"`
	PermitRegion                  string                             `json:"permit_region,omitempty" example:"Sverdlovsk Oblast"`
	PermitIssuedAt                string                             `json:"permit_issued_at,omitempty" example:"2026-01-31"`
	PermitExpiresAt               string                             `json:"permit_expires_at,omitempty" example:"2031-01-31"`
	TaxiPermitVerified            bool                               `json:"taxi_permit_verified" example:"true"`
	RegionalRegistryVerified      bool                               `json:"regional_registry_verified" example:"true"`
	RegionalRequirementsCompliant bool                               `json:"regional_requirements_compliant" example:"true"`
	HasTaxiColorScheme            bool                               `json:"has_taxi_color_scheme" example:"true"`
	HasOrangeRoofLamp             bool                               `json:"has_orange_roof_lamp" example:"true"`
	HasPassengerInfo              bool                               `json:"has_passenger_info" example:"true"`
	OSAGOVerified                 bool                               `json:"osago_verified" example:"true"`
	DiagnosticCardVerified        bool                               `json:"diagnostic_card_verified" example:"true"`
	TechnicalStateVerified        bool                               `json:"technical_state_verified" example:"true"`
	LocalizationCompliant         bool                               `json:"localization_compliant" example:"true"`
	LegalUseBasisVerified         bool                               `json:"legal_use_basis_verified" example:"true"`
	IsActive                      *bool                              `json:"is_active,omitempty" example:"true"`
}

type TaxiParkCarPatchRequest struct {
	PrimaryDriverID               *uuid.UUID                          `json:"primary_driver_id,omitempty" example:"22222222-2222-2222-2222-222222222222"`
	AttachedDriverIDs             []uuid.UUID                         `json:"attached_driver_ids,omitempty"`
	Brand                         *string                             `json:"brand,omitempty" example:"Lada"`
	Model                         *string                             `json:"model,omitempty" example:"Vesta"`
	Year                          *int                                `json:"year,omitempty" example:"2023"`
	PlateNumber                   *string                             `json:"plate_number,omitempty" example:"A001AA196"`
	VIN                           *string                             `json:"vin,omitempty" example:"XTA00000000000000"`
	STS                           *string                             `json:"sts,omitempty" example:"9911000000"`
	PTS                           *string                             `json:"pts,omitempty" example:"77AA000000"`
	Color                         *string                             `json:"color,omitempty" example:"White"`
	CarClass                      *string                             `json:"car_class,omitempty" example:"economy"`
	VerificationStatus            *domain.VerificationLifecycleStatus `json:"verification_status,omitempty" binding:"omitempty,oneof=draft pending_verification verified rejected blocked archived" example:"verified"`
	OwnerDetails                  *string                             `json:"owner_details,omitempty" example:"Leased by taxi park"`
	OwnerOrLegalBasis             *string                             `json:"owner_or_legal_basis,omitempty" example:"Lease agreement"`
	OSAGOExpiresAt                *string                             `json:"osago_expires_at,omitempty" example:"2027-01-31"`
	DiagnosticCardExpiresAt       *string                             `json:"diagnostic_card_expires_at,omitempty" example:"2027-01-31"`
	TaxiPermitNumber              *string                             `json:"taxi_permit_number,omitempty" example:"TAXI-66-000001"`
	RegionalRegistryNumber        *string                             `json:"regional_registry_number,omitempty" example:"66-123456"`
	PermitRegion                  *string                             `json:"permit_region,omitempty" example:"Sverdlovsk Oblast"`
	PermitIssuedAt                *string                             `json:"permit_issued_at,omitempty" example:"2026-01-31"`
	PermitExpiresAt               *string                             `json:"permit_expires_at,omitempty" example:"2031-01-31"`
	TaxiPermitVerified            *bool                               `json:"taxi_permit_verified,omitempty" example:"true"`
	RegionalRegistryVerified      *bool                               `json:"regional_registry_verified,omitempty" example:"true"`
	RegionalRequirementsCompliant *bool                               `json:"regional_requirements_compliant,omitempty" example:"true"`
	HasTaxiColorScheme            *bool                               `json:"has_taxi_color_scheme,omitempty" example:"true"`
	HasOrangeRoofLamp             *bool                               `json:"has_orange_roof_lamp,omitempty" example:"true"`
	HasPassengerInfo              *bool                               `json:"has_passenger_info,omitempty" example:"true"`
	OSAGOVerified                 *bool                               `json:"osago_verified,omitempty" example:"true"`
	DiagnosticCardVerified        *bool                               `json:"diagnostic_card_verified,omitempty" example:"true"`
	TechnicalStateVerified        *bool                               `json:"technical_state_verified,omitempty" example:"true"`
	LocalizationCompliant         *bool                               `json:"localization_compliant,omitempty" example:"true"`
	LegalUseBasisVerified         *bool                               `json:"legal_use_basis_verified,omitempty" example:"true"`
	IsActive                      *bool                               `json:"is_active,omitempty" example:"true"`
}

type TaxiParkCarResponse struct {
	ID                            uuid.UUID                          `json:"id" example:"55555555-5555-5555-5555-555555555555"`
	TaxiParkID                    uuid.UUID                          `json:"taxi_park_id" example:"44444444-4444-4444-4444-444444444444"`
	PrimaryDriverID               *uuid.UUID                         `json:"primary_driver_id,omitempty" example:"22222222-2222-2222-2222-222222222222"`
	AttachedDriverIDs             []uuid.UUID                        `json:"attached_driver_ids"`
	Brand                         string                             `json:"brand" example:"Lada"`
	Model                         string                             `json:"model" example:"Vesta"`
	Year                          int                                `json:"year,omitempty" example:"2023"`
	PlateNumber                   string                             `json:"plate_number" example:"A001AA196"`
	VIN                           string                             `json:"vin,omitempty" example:"XTA00000000000000"`
	STS                           string                             `json:"sts,omitempty" example:"9911000000"`
	PTS                           string                             `json:"pts,omitempty" example:"77AA000000"`
	Color                         string                             `json:"color" example:"White"`
	CarClass                      string                             `json:"car_class,omitempty" example:"economy"`
	VerificationStatus            domain.VerificationLifecycleStatus `json:"verification_status" example:"verified"`
	OwnerDetails                  string                             `json:"owner_details,omitempty" example:"Owned by taxi park"`
	OSAGOExpiresAt                *time.Time                         `json:"osago_expires_at,omitempty" example:"2027-01-31T00:00:00Z"`
	DiagnosticCardExpiresAt       *time.Time                         `json:"diagnostic_card_expires_at,omitempty" example:"2027-01-31T00:00:00Z"`
	TaxiPermitNumber              string                             `json:"taxi_permit_number,omitempty" example:"TAXI-66-000001"`
	RegionalRegistryNumber        string                             `json:"regional_registry_number,omitempty" example:"66-123456"`
	PermitRegion                  string                             `json:"permit_region,omitempty" example:"Sverdlovsk Oblast"`
	PermitIssuedAt                *time.Time                         `json:"permit_issued_at,omitempty" example:"2026-01-31T00:00:00Z"`
	PermitExpiresAt               *time.Time                         `json:"permit_expires_at,omitempty" example:"2031-01-31T00:00:00Z"`
	TaxiPermitVerified            bool                               `json:"taxi_permit_verified" example:"true"`
	RegionalRegistryVerified      bool                               `json:"regional_registry_verified" example:"true"`
	RegionalRequirementsCompliant bool                               `json:"regional_requirements_compliant" example:"true"`
	HasTaxiColorScheme            bool                               `json:"has_taxi_color_scheme" example:"true"`
	HasOrangeRoofLamp             bool                               `json:"has_orange_roof_lamp" example:"true"`
	HasPassengerInfo              bool                               `json:"has_passenger_info" example:"true"`
	OSAGOVerified                 bool                               `json:"osago_verified" example:"true"`
	DiagnosticCardVerified        bool                               `json:"diagnostic_card_verified" example:"true"`
	TechnicalStateVerified        bool                               `json:"technical_state_verified" example:"true"`
	LocalizationCompliant         bool                               `json:"localization_compliant" example:"true"`
	LegalUseBasisVerified         bool                               `json:"legal_use_basis_verified" example:"true"`
	VerificationCheckedAt         *time.Time                         `json:"verification_checked_at,omitempty" example:"2026-05-19T12:00:00Z"`
	VerificationCheckedBy         *uuid.UUID                         `json:"verification_checked_by,omitempty" example:"11111111-1111-1111-1111-111111111111"`
	IsActive                      bool                               `json:"is_active" example:"true"`
	CreatedAt                     time.Time                          `json:"created_at" example:"2026-05-19T12:00:00Z"`
	UpdatedAt                     time.Time                          `json:"updated_at" example:"2026-05-19T12:00:00Z"`
}

type TaxiParkCarsResponse struct {
	Cars []TaxiParkCarResponse `json:"cars"`
}

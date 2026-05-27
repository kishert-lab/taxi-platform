package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TaxiParkCreateDriverRequest struct {
	Phone                         string                             `json:"phone" binding:"required" example:"+79990000001"`
	Email                         string                             `json:"email,omitempty" binding:"omitempty,email" example:"driver@example.com"`
	FirstName                     string                             `json:"first_name,omitempty" example:"Ivan"`
	LastName                      string                             `json:"last_name,omitempty" example:"Petrov"`
	BirthDate                     string                             `json:"birth_date,omitempty" example:"1990-01-31"`
	Password                      string                             `json:"password,omitempty" example:"temporary-password"`
	LicenseSeries                 string                             `json:"license_series,omitempty" example:"77 01"`
	LicenseNumber                 string                             `json:"license_number,omitempty" example:"7700000000"`
	LicenseCategory               string                             `json:"license_category,omitempty" example:"B"`
	LicenseIssuedAt               string                             `json:"license_issued_at,omitempty" example:"2020-01-31"`
	LicenseExpiresAt              string                             `json:"license_expires_at,omitempty" example:"2030-01-31"`
	DrivingExperienceFrom         string                             `json:"driving_experience_from,omitempty" example:"2015-01-31"`
	HasNoTaxiWorkRestrictions     bool                               `json:"has_no_taxi_work_restrictions" example:"true"`
	FederalLaw580Compliant        bool                               `json:"federal_law_580_compliant" example:"true"`
	RegionalRequirementsCompliant bool                               `json:"regional_requirements_compliant" example:"true"`
	MedicalCheckPassed            bool                               `json:"medical_check_passed" example:"true"`
	PretripControlRequired        bool                               `json:"pretrip_control_required" example:"true"`
	PretripControlPassed          bool                               `json:"pretrip_control_passed" example:"true"`
	NoTransportBan                bool                               `json:"no_transport_ban" example:"true"`
	VerificationStatus            domain.VerificationLifecycleStatus `json:"verification_status,omitempty" binding:"omitempty,oneof=draft pending_verification verified rejected blocked archived" example:"pending_verification"`
	TaxiParkComment               string                             `json:"taxi_park_comment,omitempty" example:"Documents checked by park manager"`
	AttachedCarID                 *uuid.UUID                         `json:"attached_car_id,omitempty" example:"55555555-5555-5555-5555-555555555555"`
}

type TaxiParkCreateDriverResponse struct {
	UserID                        uuid.UUID                          `json:"user_id" example:"33333333-3333-3333-3333-333333333333"`
	DriverID                      uuid.UUID                          `json:"driver_id" example:"22222222-2222-2222-2222-222222222222"`
	TaxiParkID                    uuid.UUID                          `json:"taxi_park_id" example:"44444444-4444-4444-4444-444444444444"`
	Phone                         string                             `json:"phone" example:"+79990000001"`
	Email                         string                             `json:"email,omitempty" example:"driver@example.com"`
	Name                          string                             `json:"name,omitempty" example:"Ivan Petrov"`
	Status                        domain.DriverStatus                `json:"status" example:"offline"`
	VerificationStatus            domain.VerificationLifecycleStatus `json:"verification_status" example:"pending_verification"`
	Rating                        float64                            `json:"rating" example:"5"`
	RatingsCount                  int                                `json:"ratings_count" example:"0"`
	BirthDate                     *time.Time                         `json:"birth_date,omitempty" example:"1990-01-31T00:00:00Z"`
	LicenseSeries                 string                             `json:"license_series,omitempty" example:"77 01"`
	LicenseNumber                 string                             `json:"license_number,omitempty" example:"7700000000"`
	LicenseCategory               string                             `json:"license_category,omitempty" example:"B"`
	LicenseIssuedAt               *time.Time                         `json:"license_issued_at,omitempty" example:"2020-01-31T00:00:00Z"`
	LicenseExpiresAt              *time.Time                         `json:"license_expires_at,omitempty" example:"2030-01-31T00:00:00Z"`
	DrivingExperienceFrom         *time.Time                         `json:"driving_experience_from,omitempty" example:"2015-01-31T00:00:00Z"`
	HasNoTaxiWorkRestrictions     bool                               `json:"has_no_taxi_work_restrictions" example:"true"`
	FederalLaw580Compliant        bool                               `json:"federal_law_580_compliant" example:"true"`
	RegionalRequirementsCompliant bool                               `json:"regional_requirements_compliant" example:"true"`
	MedicalCheckPassed            bool                               `json:"medical_check_passed" example:"true"`
	PretripControlRequired        bool                               `json:"pretrip_control_required" example:"true"`
	PretripControlPassed          bool                               `json:"pretrip_control_passed" example:"true"`
	NoTransportBan                bool                               `json:"no_transport_ban" example:"true"`
	VerificationCheckedAt         *time.Time                         `json:"verification_checked_at,omitempty" example:"2026-05-19T12:00:00Z"`
	VerificationCheckedBy         *uuid.UUID                         `json:"verification_checked_by,omitempty" example:"11111111-1111-1111-1111-111111111111"`
	IsVerified                    bool                               `json:"is_verified" example:"false"`
	TaxiParkComment               string                             `json:"taxi_park_comment,omitempty" example:"Documents checked by park manager"`
	GeneratedPassword             string                             `json:"generated_password,omitempty" example:"A1b2C3d4E5f6G7h8J9"`
	PasswordGenerated             bool                               `json:"password_generated" example:"true"`
}

type TaxiParkUpdateDriverRequest struct {
	FirstName                     *string                             `json:"first_name,omitempty" example:"Ivan"`
	LastName                      *string                             `json:"last_name,omitempty" example:"Petrov"`
	BirthDate                     *string                             `json:"birth_date,omitempty" example:"1990-01-31"`
	LicenseSeries                 *string                             `json:"license_series,omitempty" example:"77 01"`
	LicenseNumber                 *string                             `json:"license_number,omitempty" example:"7700000000"`
	LicenseCategory               *string                             `json:"license_category,omitempty" example:"B"`
	LicenseIssuedAt               *string                             `json:"license_issued_at,omitempty" example:"2020-01-31"`
	LicenseExpiresAt              *string                             `json:"license_expires_at,omitempty" example:"2030-01-31"`
	DrivingExperienceFrom         *string                             `json:"driving_experience_from,omitempty" example:"2015-01-31"`
	HasNoTaxiWorkRestrictions     *bool                               `json:"has_no_taxi_work_restrictions,omitempty" example:"true"`
	FederalLaw580Compliant        *bool                               `json:"federal_law_580_compliant,omitempty" example:"true"`
	RegionalRequirementsCompliant *bool                               `json:"regional_requirements_compliant,omitempty" example:"true"`
	MedicalCheckPassed            *bool                               `json:"medical_check_passed,omitempty" example:"true"`
	PretripControlRequired        *bool                               `json:"pretrip_control_required,omitempty" example:"true"`
	PretripControlPassed          *bool                               `json:"pretrip_control_passed,omitempty" example:"true"`
	NoTransportBan                *bool                               `json:"no_transport_ban,omitempty" example:"true"`
	VerificationStatus            *domain.VerificationLifecycleStatus `json:"verification_status,omitempty" binding:"omitempty,oneof=draft pending_verification verified rejected blocked archived" example:"verified"`
	TaxiParkComment               *string                             `json:"taxi_park_comment,omitempty" example:"Verified by park"`
	AttachedCarID                 *uuid.UUID                          `json:"attached_car_id,omitempty" example:"55555555-5555-5555-5555-555555555555"`
}

type TaxiParkBlockDriverRequest struct {
	Reason string `json:"reason" binding:"required" example:"Expired license"`
}

type TaxiParkDriverPasswordRequest struct {
	Password string `json:"password" binding:"required" example:"NewPassword123!"`
}

type TaxiParkDriverPasswordResponse struct {
	DriverID        uuid.UUID `json:"driver_id" example:"22222222-2222-2222-2222-222222222222"`
	PasswordUpdated bool      `json:"password_updated" example:"true"`
}

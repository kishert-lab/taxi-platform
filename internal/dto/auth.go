package dto

import (
	"github.com/google/uuid"

	"github.com/develoop/taxi-platform/internal/domain"
)

type StartRegistrationRequest struct {
	Phone                string                  `json:"phone" binding:"required" example:"+79990000000"`
	Email                string                  `json:"email" binding:"required,email" example:"user@example.com"`
	Password             string                  `json:"password" binding:"required,min=8" example:"strong-password"`
	FirstName            string                  `json:"first_name" binding:"required" example:"Ivan"`
	LastName             string                  `json:"last_name" example:"Petrov"`
	RegistrationType     domain.RegistrationType `json:"registration_type" binding:"required,oneof=passenger driver taxi_park" example:"passenger"`
	CityID               uuid.UUID               `json:"city_id" example:"11111111-1111-1111-1111-111111111111"`
	PersonalDataConsent  bool                    `json:"personal_data_consent" binding:"required" example:"true"`
	TermsAccepted        bool                    `json:"terms_accepted" binding:"required" example:"true"`
	PrivacyPolicyVersion string                  `json:"privacy_policy_version" binding:"required" example:"1.0"`
	TermsVersion         string                  `json:"terms_version" binding:"required" example:"1.0"`
	TaxiPark             *TaxiParkRegistration   `json:"taxi_park,omitempty"`
}

type TaxiParkRegistration struct {
	Name      string `json:"name" binding:"required_with=TaxiPark" example:"North Taxi Park"`
	LegalName string `json:"legal_name" example:"North Taxi Park LLC"`
	TaxID     string `json:"tax_id" example:"7700000000"`
}

type StartRegistrationResponse struct {
	UserID           uuid.UUID               `json:"user_id" example:"33333333-3333-3333-3333-333333333333"`
	Role             domain.UserRole         `json:"role" example:"passenger"`
	RegistrationType domain.RegistrationType `json:"registration_type" example:"passenger"`
	PhoneMasked      string                  `json:"phone_masked" example:"+7*****000"`
	EmailMasked      string                  `json:"email_masked" example:"u***@example.com"`
	Message          string                  `json:"message" example:"confirmation codes sent"`
}

type ConfirmPhoneRequest struct {
	Phone string                  `json:"phone" binding:"required" example:"+79990000000"`
	Type  domain.RegistrationType `json:"registration_type" binding:"required,oneof=passenger driver taxi_park" example:"passenger"`
	Code  string                  `json:"code" binding:"required,len=6" example:"123456"`
}

type ConfirmEmailRequest struct {
	Email string `json:"email" binding:"required,email" example:"user@example.com"`
	Code  string `json:"code" binding:"required,len=6" example:"123456"`
}

type LoginByPhoneRequest struct {
	Phone string          `json:"phone" binding:"required" example:"+79990000000"`
	Role  domain.UserRole `json:"role" binding:"required,oneof=passenger driver taxi_park admin dispatcher" example:"passenger"`
	Code  string          `json:"code" binding:"required,len=6" example:"123456"`
}

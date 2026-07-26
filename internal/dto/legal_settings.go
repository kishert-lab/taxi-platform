package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TaxiParkSettingsResponse struct {
	ID                      uuid.UUID                         `json:"id" example:"11111111-1111-1111-1111-111111111111"`
	TaxiParkID              uuid.UUID                         `json:"taxi_park_id" example:"22222222-2222-2222-2222-222222222222"`
	CityID                  uuid.UUID                         `json:"city_id" example:"33333333-3333-3333-3333-333333333333"`
	City                    TaxiParkCityDTO                   `json:"city"`
	DisplayName             string                            `json:"display_name" example:"North Taxi"`
	ShortName               string                            `json:"short_name,omitempty" example:"North"`
	SupportPhone            string                            `json:"support_phone,omitempty" example:"+79990000000"`
	SupportEmail            string                            `json:"support_email,omitempty" example:"support@example.com"`
	LegalName               string                            `json:"legal_name,omitempty" example:"ООО Северное такси"`
	LegalAddress            string                            `json:"legal_address,omitempty" example:"Екатеринбург, Ленина 1"`
	INN                     string                            `json:"inn,omitempty" example:"7700000000"`
	OGRN                    string                            `json:"ogrn,omitempty" example:"1027700000000"`
	Website                 string                            `json:"website,omitempty" example:"https://taxi.example.com"`
	LogoURL                 string                            `json:"logo_url,omitempty" example:"https://cdn.example.com/logo.png"`
	PrimaryColor            string                            `json:"primary_color,omitempty" example:"#111827"`
	SecondaryColor          string                            `json:"secondary_color,omitempty" example:"#F59E0B"`
	CommissionBasisPoints   *int32                            `json:"commission_basis_points,omitempty" example:"100"`
	CommissionPercent       string                            `json:"commission_percent,omitempty" example:"1.00"`
	MinimumOrderPrice       MoneyCentsResponse                `json:"minimum_order_price"`
	CancellationTimeoutSec  int                               `json:"cancellation_timeout_sec" example:"300"`
	DriverArrivalTimeoutSec int                               `json:"driver_arrival_timeout_sec" example:"900"`
	AllowCashPayment        bool                              `json:"allow_cash_payment" example:"true"`
	AllowCardPayment        bool                              `json:"allow_card_payment" example:"true"`
	AllowTransferPayment    bool                              `json:"allow_transfer_payment" example:"false"`
	Dispatch                TaxiParkDispatchSettingsResponse  `json:"dispatch"`
	Scheduled               TaxiParkScheduledSettingsResponse `json:"scheduled"`
	IsActive                bool                              `json:"is_active" example:"true"`
	CreatedAt               time.Time                         `json:"created_at" example:"2026-05-12T12:00:00Z"`
	UpdatedAt               time.Time                         `json:"updated_at" example:"2026-05-12T12:00:00Z"`
}

type TaxiParkCityDTO struct {
	ID          uuid.UUID           `json:"id" example:"33333333-3333-3333-3333-333333333333"`
	Name        string              `json:"name" example:"Yekaterinburg"`
	Region      string              `json:"region" example:"Sverdlovsk Oblast"`
	CountryCode string              `json:"country_code" example:"RU"`
	Timezone    string              `json:"timezone" example:"Asia/Yekaterinburg"`
	Center      CoordinatesResponse `json:"center"`
}

type TaxiParkDispatchSettingsResponse struct {
	InitialRadiusMeters     int   `json:"initial_radius_meters" example:"10000"`
	MaxRadiusMeters         int   `json:"max_radius_meters" example:"100000"`
	RadiusStepMeters        int   `json:"radius_step_meters" example:"1000"`
	RadiusAttemptsMeters    []int `json:"radius_attempts_meters" example:"10000,30000,50000,100000"`
	MaxDriversPerOffer      int   `json:"max_drivers_per_offer" example:"5"`
	DriverLocationMaxAgeSec int   `json:"driver_location_max_age_sec" example:"120"`
	OfferTTLSec             int   `json:"offer_ttl_sec" example:"60"`
	AcceptLockTTLSec        int   `json:"accept_lock_ttl_sec" example:"90"`
	WorkerPollTimeoutSec    int   `json:"worker_poll_timeout_sec" example:"30"`
	RecoveryIntervalSec     int   `json:"recovery_interval_sec" example:"30"`
}

type TaxiParkScheduledSettingsResponse struct {
	ScheduledOrdersEnabled            bool `json:"scheduled_orders_enabled" example:"true"`
	ScheduledMinBeforeMinutes         int  `json:"scheduled_min_before_minutes" example:"15"`
	ScheduledActivationBeforeMinutes  int  `json:"scheduled_activation_before_minutes" example:"20"`
	ScheduledExpireAfterMinutes       int  `json:"scheduled_expire_after_minutes" example:"15"`
	AllowScheduledDriverPreassignment bool `json:"allow_scheduled_driver_preassignment" example:"true"`
}

type TaxiParkSettingsPatchRequest struct {
	DisplayName             *string                                `json:"display_name,omitempty" example:"North Taxi"`
	ShortName               *string                                `json:"short_name,omitempty" example:"North"`
	SupportPhone            *string                                `json:"support_phone,omitempty" example:"+79990000000"`
	SupportEmail            *string                                `json:"support_email,omitempty" binding:"omitempty,email" example:"support@example.com"`
	LegalName               *string                                `json:"legal_name,omitempty" example:"ООО Северное такси"`
	LegalAddress            *string                                `json:"legal_address,omitempty" example:"Екатеринбург, Ленина 1"`
	INN                     *string                                `json:"inn,omitempty" example:"7700000000"`
	OGRN                    *string                                `json:"ogrn,omitempty" example:"1027700000000"`
	Website                 *string                                `json:"website,omitempty" binding:"omitempty,url" example:"https://taxi.example.com"`
	LogoURL                 *string                                `json:"logo_url,omitempty" binding:"omitempty,url" example:"https://cdn.example.com/logo.png"`
	PrimaryColor            *string                                `json:"primary_color,omitempty" example:"#111827"`
	SecondaryColor          *string                                `json:"secondary_color,omitempty" example:"#F59E0B"`
	CommissionBasisPoints   *int32                                 `json:"commission_basis_points,omitempty" binding:"omitempty,min=0,max=10000" example:"100"`
	MinimumOrderPriceCents  *int64                                 `json:"minimum_order_price_cents,omitempty" binding:"omitempty,min=0" example:"25000"`
	CancellationTimeoutSec  *int                                   `json:"cancellation_timeout_sec,omitempty" binding:"omitempty,min=1" example:"300"`
	DriverArrivalTimeoutSec *int                                   `json:"driver_arrival_timeout_sec,omitempty" binding:"omitempty,min=1" example:"900"`
	AllowCashPayment        *bool                                  `json:"allow_cash_payment,omitempty" example:"true"`
	AllowCardPayment        *bool                                  `json:"allow_card_payment,omitempty" example:"true"`
	AllowTransferPayment    *bool                                  `json:"allow_transfer_payment,omitempty" example:"false"`
	Dispatch                *TaxiParkDispatchSettingsPatchRequest  `json:"dispatch,omitempty"`
	Scheduled               *TaxiParkScheduledSettingsPatchRequest `json:"scheduled,omitempty"`
	IsActive                *bool                                  `json:"is_active,omitempty" example:"true"`
}

type TaxiParkDispatchSettingsPatchRequest struct {
	InitialRadiusMeters     *int  `json:"initial_radius_meters,omitempty" binding:"omitempty,min=1" example:"10000"`
	MaxRadiusMeters         *int  `json:"max_radius_meters,omitempty" binding:"omitempty,min=1" example:"100000"`
	RadiusStepMeters        *int  `json:"radius_step_meters,omitempty" binding:"omitempty,min=1" example:"1000"`
	RadiusAttemptsMeters    []int `json:"radius_attempts_meters,omitempty" binding:"omitempty,min=1,dive,min=1" example:"10000,30000,50000,100000"`
	MaxDriversPerOffer      *int  `json:"max_drivers_per_offer,omitempty" binding:"omitempty,min=1" example:"5"`
	DriverLocationMaxAgeSec *int  `json:"driver_location_max_age_sec,omitempty" binding:"omitempty,min=1" example:"120"`
	OfferTTLSec             *int  `json:"offer_ttl_sec,omitempty" binding:"omitempty,min=1" example:"60"`
	AcceptLockTTLSec        *int  `json:"accept_lock_ttl_sec,omitempty" binding:"omitempty,min=1" example:"90"`
	WorkerPollTimeoutSec    *int  `json:"worker_poll_timeout_sec,omitempty" binding:"omitempty,min=1" example:"30"`
	RecoveryIntervalSec     *int  `json:"recovery_interval_sec,omitempty" binding:"omitempty,min=1" example:"30"`
}

type TaxiParkScheduledSettingsPatchRequest struct {
	ScheduledOrdersEnabled            *bool `json:"scheduled_orders_enabled,omitempty" example:"true"`
	ScheduledMinBeforeMinutes         *int  `json:"scheduled_min_before_minutes,omitempty" binding:"omitempty,min=1" example:"15"`
	ScheduledActivationBeforeMinutes  *int  `json:"scheduled_activation_before_minutes,omitempty" binding:"omitempty,min=1" example:"20"`
	ScheduledExpireAfterMinutes       *int  `json:"scheduled_expire_after_minutes,omitempty" binding:"omitempty,min=1" example:"15"`
	AllowScheduledDriverPreassignment *bool `json:"allow_scheduled_driver_preassignment,omitempty" example:"true"`
}

type TaxiParkTariffRequest struct {
	Name                string          `json:"name" binding:"required" example:"Park Economy"`
	CarClassID          *uuid.UUID      `json:"car_class_id,omitempty" example:"33333333-3333-3333-3333-333333333333"`
	Description         string          `json:"description,omitempty" example:"Local economy tariff"`
	PricingMode         string          `json:"pricing_mode" binding:"omitempty,oneof=fixed distance time distance_time" example:"distance_time"`
	BasePriceCents      int64           `json:"base_price_cents" binding:"min=0" example:"10000"`
	FixedPriceCents     int64           `json:"fixed_price_cents" binding:"min=0" example:"25000"`
	PricePerKMCents     int64           `json:"price_per_km_cents" binding:"min=0" example:"2500"`
	PricePerMinuteCents int64           `json:"price_per_minute_cents" binding:"min=0" example:"500"`
	MinimumPriceCents   int64           `json:"minimum_price_cents" binding:"min=0" example:"18000"`
	FixedRoutes         json.RawMessage `json:"fixed_routes,omitempty" swaggertype:"array,object"`
	IsActive            *bool           `json:"is_active,omitempty" example:"true"`
}

type TaxiParkTariffPatchRequest struct {
	Name                *string         `json:"name,omitempty" example:"Park Economy"`
	CarClassID          *uuid.UUID      `json:"car_class_id,omitempty" example:"33333333-3333-3333-3333-333333333333"`
	Description         *string         `json:"description,omitempty" example:"Local economy tariff"`
	PricingMode         *string         `json:"pricing_mode,omitempty" binding:"omitempty,oneof=fixed distance time distance_time" example:"distance_time"`
	BasePriceCents      *int64          `json:"base_price_cents,omitempty" binding:"omitempty,min=0" example:"10000"`
	FixedPriceCents     *int64          `json:"fixed_price_cents,omitempty" binding:"omitempty,min=0" example:"25000"`
	PricePerKMCents     *int64          `json:"price_per_km_cents,omitempty" binding:"omitempty,min=0" example:"2500"`
	PricePerMinuteCents *int64          `json:"price_per_minute_cents,omitempty" binding:"omitempty,min=0" example:"500"`
	MinimumPriceCents   *int64          `json:"minimum_price_cents,omitempty" binding:"omitempty,min=0" example:"18000"`
	FixedRoutes         json.RawMessage `json:"fixed_routes,omitempty" swaggertype:"array,object"`
	IsActive            *bool           `json:"is_active,omitempty" example:"true"`
}

type TaxiParkTariffResponse struct {
	ID             uuid.UUID          `json:"id" example:"33333333-3333-3333-3333-333333333333"`
	TaxiParkID     uuid.UUID          `json:"taxi_park_id" example:"22222222-2222-2222-2222-222222222222"`
	CarClassID     *uuid.UUID         `json:"car_class_id,omitempty" example:"33333333-3333-3333-3333-333333333333"`
	Name           string             `json:"name" example:"Park Economy"`
	Description    string             `json:"description,omitempty" example:"Local economy tariff"`
	PricingMode    string             `json:"pricing_mode" example:"distance_time"`
	BasePrice      MoneyCentsResponse `json:"base_price"`
	FixedPrice     MoneyCentsResponse `json:"fixed_price"`
	PricePerKM     MoneyCentsResponse `json:"price_per_km"`
	PricePerMinute MoneyCentsResponse `json:"price_per_minute"`
	MinimumPrice   MoneyCentsResponse `json:"minimum_price"`
	FixedRoutes    json.RawMessage    `json:"fixed_routes" swaggertype:"object,string"`
	IsActive       bool               `json:"is_active" example:"true"`
	CreatedAt      time.Time          `json:"created_at" example:"2026-05-12T12:00:00Z"`
	UpdatedAt      time.Time          `json:"updated_at" example:"2026-05-12T12:00:00Z"`
}

type TaxiParkTariffsResponse struct {
	Tariffs []TaxiParkTariffResponse `json:"tariffs"`
}

type LegalDocumentRequest struct {
	DocumentType domain.LegalDocumentType `json:"document_type" binding:"required" example:"privacy_policy"`
	Version      string                   `json:"version" binding:"required" example:"1.1"`
	Title        string                   `json:"title" binding:"required" example:"Политика обработки персональных данных"`
	Content      string                   `json:"content" binding:"required" example:"# Политика обработки персональных данных"`
	Language     string                   `json:"language" example:"ru"`
	Activate     bool                     `json:"activate" example:"false"`
}

type LegalDocumentResponse struct {
	ID           uuid.UUID                `json:"id" example:"44444444-4444-4444-4444-444444444444"`
	DocumentType domain.LegalDocumentType `json:"document_type" example:"privacy_policy"`
	Version      string                   `json:"version" example:"1.0"`
	Title        string                   `json:"title" example:"Политика обработки персональных данных"`
	Content      string                   `json:"content" example:"# Политика обработки персональных данных"`
	Language     string                   `json:"language" example:"ru"`
	IsActive     bool                     `json:"is_active" example:"true"`
	CreatedAt    time.Time                `json:"created_at" example:"2026-05-12T12:00:00Z"`
}

type LegalDocumentsResponse struct {
	Documents []LegalDocumentResponse `json:"documents"`
}

func TaxiParkSettingsFromDomain(settings domain.TaxiParkSettings) TaxiParkSettingsResponse {
	response := TaxiParkSettingsResponse{
		ID:         settings.ID,
		TaxiParkID: settings.TaxiParkID,
		CityID:     settings.CityID,
		City: TaxiParkCityDTO{
			ID:          settings.CityID,
			Name:        settings.CityName,
			Region:      settings.CityRegion,
			CountryCode: settings.CityCountryCode,
			Timezone:    settings.CityTimezone,
			Center: CoordinatesResponse{
				Latitude:  settings.CityCenter.Latitude,
				Longitude: settings.CityCenter.Longitude,
			},
		},
		DisplayName:             settings.DisplayName,
		ShortName:               settings.ShortName,
		SupportPhone:            settings.SupportPhone,
		SupportEmail:            settings.SupportEmail,
		LegalName:               settings.LegalName,
		LegalAddress:            settings.LegalAddress,
		INN:                     settings.INN,
		OGRN:                    settings.OGRN,
		Website:                 settings.Website,
		LogoURL:                 settings.LogoURL,
		PrimaryColor:            settings.PrimaryColor,
		SecondaryColor:          settings.SecondaryColor,
		CommissionBasisPoints:   settings.CommissionBasisPoints,
		MinimumOrderPrice:       MoneyCentsFromDomain(settings.MinimumOrderPrice),
		CancellationTimeoutSec:  settings.CancellationTimeoutSec,
		DriverArrivalTimeoutSec: settings.DriverArrivalTimeoutSec,
		AllowCashPayment:        settings.AllowCashPayment,
		AllowCardPayment:        settings.AllowCardPayment,
		AllowTransferPayment:    settings.AllowTransferPayment,
		Dispatch: TaxiParkDispatchSettingsResponse{
			InitialRadiusMeters:     settings.DispatchInitialRadiusMeters,
			MaxRadiusMeters:         settings.DispatchMaxRadiusMeters,
			RadiusStepMeters:        settings.DispatchRadiusStepMeters,
			RadiusAttemptsMeters:    settings.DispatchRadiusAttemptsMeters,
			MaxDriversPerOffer:      settings.DispatchMaxDriversPerOffer,
			DriverLocationMaxAgeSec: settings.DispatchDriverLocationMaxAgeSec,
			OfferTTLSec:             settings.DispatchOfferTTLSec,
			AcceptLockTTLSec:        settings.DispatchAcceptLockTTLSec,
			WorkerPollTimeoutSec:    settings.DispatchWorkerPollTimeoutSec,
			RecoveryIntervalSec:     settings.DispatchRecoveryIntervalSec,
		},
		Scheduled: TaxiParkScheduledSettingsResponse{
			ScheduledOrdersEnabled:            settings.ScheduledOrdersEnabled,
			ScheduledMinBeforeMinutes:         settings.ScheduledMinBeforeMinutes,
			ScheduledActivationBeforeMinutes:  settings.ScheduledActivationBeforeMinutes,
			ScheduledExpireAfterMinutes:       settings.ScheduledExpireAfterMinutes,
			AllowScheduledDriverPreassignment: settings.AllowScheduledDriverPreassignment,
		},
		IsActive:  settings.IsActive,
		CreatedAt: settings.CreatedAt,
		UpdatedAt: settings.UpdatedAt,
	}
	if settings.CommissionBasisPoints != nil {
		response.CommissionPercent = FormatBasisPoints(*settings.CommissionBasisPoints)
	}
	return response
}

func TaxiParkTariffFromDomain(tariff domain.TaxiParkTariff) TaxiParkTariffResponse {
	return TaxiParkTariffResponse{
		ID:             tariff.ID,
		TaxiParkID:     tariff.TaxiParkID,
		CarClassID:     tariff.CarClassID,
		Name:           tariff.Name,
		Description:    tariff.Description,
		PricingMode:    string(tariff.PricingMode),
		BasePrice:      MoneyCentsFromDomain(tariff.BasePrice),
		FixedPrice:     MoneyCentsFromDomain(tariff.FixedPrice),
		PricePerKM:     MoneyCentsFromDomain(tariff.PricePerKM),
		PricePerMinute: MoneyCentsFromDomain(tariff.PricePerMinute),
		MinimumPrice:   MoneyCentsFromDomain(tariff.MinimumPrice),
		FixedRoutes:    json.RawMessage(tariff.FixedRoutes),
		IsActive:       tariff.IsActive,
		CreatedAt:      tariff.CreatedAt,
		UpdatedAt:      tariff.UpdatedAt,
	}
}

func LegalDocumentFromDomain(document domain.LegalDocument) LegalDocumentResponse {
	return LegalDocumentResponse{
		ID:           document.ID,
		DocumentType: document.DocumentType,
		Version:      document.Version,
		Title:        document.Title,
		Content:      document.Content,
		Language:     document.Language,
		IsActive:     document.IsActive,
		CreatedAt:    document.CreatedAt,
	}
}

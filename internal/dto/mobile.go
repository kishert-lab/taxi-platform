package dto

import (
	"io"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PassengerProfileRequest struct {
	Name     string `json:"name" binding:"required" example:"Irina"`
	Email    string `json:"email" binding:"omitempty,email" example:"irina@example.com"`
	PhotoURL string `json:"photo_url,omitempty" binding:"omitempty,url" example:"https://cdn.example.com/passengers/111/photo.jpg"`
}

type PassengerProfilePatchRequest struct {
	Name     *string `json:"name,omitempty" example:"Irina"`
	Email    *string `json:"email,omitempty" example:"irina@example.com"`
	PhotoURL *string `json:"photo_url,omitempty" binding:"omitempty,url" example:"https://cdn.example.com/passengers/111/photo.jpg"`
}

type PassengerProfileResponse struct {
	ID           uuid.UUID `json:"id" example:"11111111-1111-1111-1111-111111111111"`
	Phone        string    `json:"phone" example:"+79990000000"`
	Name         string    `json:"name" example:"Irina"`
	Email        string    `json:"email,omitempty" example:"irina@example.com"`
	PhotoURL     string    `json:"photo_url,omitempty" example:"https://cdn.example.com/passengers/111/photo.jpg"`
	Rating       float64   `json:"rating" example:"4.92"`
	RatingsCount int       `json:"ratings_count" example:"37"`
}

type DriverProfilePatchRequest struct {
	Name          *string `json:"name,omitempty" example:"Ivan"`
	LicenseNumber *string `json:"license_number,omitempty" example:"7700000000"`
	PhotoURL      *string `json:"photo_url,omitempty" binding:"omitempty,url" example:"https://cdn.example.com/drivers/222/photo.jpg"`
}

type DriverProfileResponse struct {
	ID                            uuid.UUID                          `json:"id" example:"22222222-2222-2222-2222-222222222222"`
	UserID                        uuid.UUID                          `json:"user_id" example:"33333333-3333-3333-3333-333333333333"`
	Phone                         string                             `json:"phone" example:"+79990000001"`
	Name                          string                             `json:"name" example:"Ivan"`
	PhotoURL                      string                             `json:"photo_url,omitempty" example:"https://cdn.example.com/drivers/222/photo.jpg"`
	Status                        domain.DriverStatus                `json:"status" example:"online"`
	Rating                        float64                            `json:"rating" example:"4.95"`
	RatingsCount                  int                                `json:"ratings_count" example:"112"`
	LicenseNumber                 string                             `json:"license_number,omitempty" example:"7700000000"`
	IsVerified                    bool                               `json:"is_verified" example:"true"`
	VerificationStatus            domain.VerificationLifecycleStatus `json:"verification_status" example:"verified"`
	TaxiParkID                    *uuid.UUID                         `json:"taxi_park_id,omitempty" example:"44444444-4444-4444-4444-444444444444"`
	TaxiParkIsActive              *bool                              `json:"taxi_park_is_active,omitempty" example:"true"`
	HasNoTaxiWorkRestrictions     bool                               `json:"has_no_taxi_work_restrictions" example:"true"`
	FederalLaw580Compliant        bool                               `json:"federal_law_580_compliant" example:"true"`
	RegionalRequirementsCompliant bool                               `json:"regional_requirements_compliant" example:"true"`
	MedicalCheckPassed            bool                               `json:"medical_check_passed" example:"true"`
	PretripControlRequired        bool                               `json:"pretrip_control_required" example:"false"`
	PretripControlPassed          bool                               `json:"pretrip_control_passed" example:"true"`
	NoTransportBan                bool                               `json:"no_transport_ban" example:"true"`
	Car                           *DriverProfileCarResponse          `json:"car,omitempty"`
}

type DriverProfileCarResponse struct {
	ID                 uuid.UUID                          `json:"id" example:"55555555-5555-5555-5555-555555555555"`
	Brand              string                             `json:"brand" example:"Lada"`
	Model              string                             `json:"model" example:"Vesta"`
	Year               int                                `json:"year,omitempty" example:"2023"`
	PlateNumber        string                             `json:"plate_number" example:"A001AA196"`
	Color              string                             `json:"color,omitempty" example:"White"`
	CarClass           string                             `json:"car_class,omitempty" example:"economy"`
	VerificationStatus domain.VerificationLifecycleStatus `json:"verification_status" example:"verified"`
	IsActive           bool                               `json:"is_active" example:"true"`
	OSAGOExpiresAt     *time.Time                         `json:"osago_expires_at,omitempty" example:"2027-01-31T00:00:00Z"`
	PermitExpiresAt    *time.Time                         `json:"permit_expires_at,omitempty" example:"2031-01-31T00:00:00Z"`
}

type ProfilePhotoUploadRequest struct {
	FileName    string
	ContentType string
	Size        int64
	Body        io.Reader
}

type ProfilePhotoUploadResponse struct {
	PhotoURL string `json:"photo_url" example:"https://cdn.example.com/profiles/111/photo.jpg"`
}

type OrderEstimateRequest struct {
	CityID              uuid.UUID          `json:"city_id" binding:"required" example:"11111111-1111-1111-1111-111111111111"`
	TariffID            uuid.UUID          `json:"tariff_id" binding:"required" example:"22222222-2222-2222-2222-222222222222"`
	PickupLocation      CoordinatesRequest `json:"pickup_location" binding:"required"`
	DestinationLocation CoordinatesRequest `json:"destination_location" binding:"required"`
}

type OrderEstimateResponse struct {
	TariffID    uuid.UUID `json:"tariff_id" example:"22222222-2222-2222-2222-222222222222"`
	TariffName  string    `json:"tariff_name" example:"Economy"`
	DistanceKM  float64   `json:"distance_km" example:"4.2"`
	DurationMin int64     `json:"duration_min" example:"11"`
	Price       int64     `json:"price" example:"250"`
	Currency    string    `json:"currency" example:"RUB"`
	PriceType   string    `json:"price_type" example:"estimated"`
}

type PassengerCreateOrderRequest struct {
	CityID              uuid.UUID            `json:"city_id" binding:"required" example:"11111111-1111-1111-1111-111111111111"`
	PickupLocation      CoordinatesRequest   `json:"pickup_location" binding:"required"`
	PickupAddress       string               `json:"pickup_address" binding:"required" example:"Lenina 1"`
	DestinationLocation CoordinatesRequest   `json:"destination_location" binding:"required"`
	DestinationAddress  string               `json:"destination_address" binding:"required" example:"Mira 10"`
	TariffID            uuid.UUID            `json:"tariff_id" binding:"required" example:"22222222-2222-2222-2222-222222222222"`
	PaymentType         domain.PaymentMethod `json:"payment_type" binding:"required,oneof=cash card corporate" example:"cash"`
	Comment             string               `json:"comment" example:"Entrance 2"`
	PassengerPhone      string               `json:"passenger_phone,omitempty" example:"+79990000000"`
}

type PassengerOrderResponse struct {
	OrderID          uuid.UUID           `json:"order_id" example:"44444444-4444-4444-4444-444444444444"`
	Driver           *AssignedDriverDTO  `json:"driver,omitempty"`
	Car              *CarDTO             `json:"car,omitempty"`
	PickupPoint      PointDTO            `json:"pickup_point"`
	DestinationPoint PointDTO            `json:"destination_point"`
	Status           domain.OrderStatus  `json:"status" example:"driver_arriving"`
	Price            *MoneyResponse      `json:"price,omitempty"`
	ETASeconds       *int64              `json:"eta_seconds,omitempty" example:"420"`
	AllowedActions   []string            `json:"allowed_actions" example:"cancel,call_driver"`
	Timeline         []OrderTimelineItem `json:"timeline,omitempty"`
	Version          int                 `json:"version" example:"3"`
}

type DriverOrderResponse struct {
	OrderID          uuid.UUID           `json:"order_id" example:"44444444-4444-4444-4444-444444444444"`
	Passenger        PassengerBriefDTO   `json:"passenger"`
	PickupPoint      PointDTO            `json:"pickup_point"`
	DestinationPoint PointDTO            `json:"destination_point"`
	Status           domain.OrderStatus  `json:"status" example:"driver_assigned"`
	Price            *MoneyResponse      `json:"price,omitempty"`
	Comment          string              `json:"comment,omitempty" example:"Entrance 2"`
	Timeline         []OrderTimelineItem `json:"timeline,omitempty"`
	AllowedActions   []string            `json:"allowed_actions" example:"arrived,call_passenger"`
	Version          int                 `json:"version" example:"3"`
}

type PointDTO struct {
	Address  string              `json:"address" example:"Lenina 1"`
	Location CoordinatesResponse `json:"location"`
}

type AssignedDriverDTO struct {
	ID           uuid.UUID `json:"id" example:"55555555-5555-5555-5555-555555555555"`
	Name         string    `json:"name" example:"Ivan"`
	Phone        string    `json:"phone" example:"+79990000001"`
	PhotoURL     string    `json:"photo_url,omitempty" example:"https://cdn.example.com/drivers/555/photo.jpg"`
	Rating       float64   `json:"rating" example:"4.95"`
	RatingsCount int       `json:"ratings_count" example:"112"`
}

type PassengerBriefDTO struct {
	ID           uuid.UUID `json:"id" example:"66666666-6666-6666-6666-666666666666"`
	Name         string    `json:"name" example:"Irina"`
	Phone        string    `json:"phone" example:"+79990000000"`
	PhotoURL     string    `json:"photo_url,omitempty" example:"https://cdn.example.com/passengers/666/photo.jpg"`
	Rating       float64   `json:"rating" example:"4.92"`
	RatingsCount int       `json:"ratings_count" example:"37"`
}

type CarDTO struct {
	ID          uuid.UUID `json:"id" example:"77777777-7777-7777-7777-777777777777"`
	Brand       string    `json:"brand" example:"Lada"`
	Model       string    `json:"model" example:"Vesta"`
	Color       string    `json:"color" example:"White"`
	PlateNumber string    `json:"plate_number" example:"A001AA196"`
}

type OrderTimelineItem struct {
	Status     domain.OrderStatus `json:"status" example:"driver_assigned"`
	OccurredAt time.Time          `json:"occurred_at" example:"2026-05-12T12:00:00Z"`
}

type OrderHistoryResponse struct {
	Orders []PassengerOrderResponse `json:"orders"`
}

type DriverOrderHistoryResponse struct {
	Orders []DriverOrderResponse `json:"orders"`
}

type DriverOrderOffersResponse struct {
	Offers []DriverOrderOfferResponse `json:"offers"`
}

type DriverOrderOfferResponse struct {
	OrderID          uuid.UUID          `json:"order_id" example:"44444444-4444-4444-4444-444444444444"`
	PickupPoint      PointDTO           `json:"pickup_point"`
	DestinationPoint PointDTO           `json:"destination_point"`
	Status           domain.OrderStatus `json:"status" example:"searching"`
	EstimatedPrice   *MoneyResponse     `json:"estimated_price,omitempty"`
	Attempt          int                `json:"attempt" example:"0"`
	RadiusMeters     int                `json:"radius_meters" example:"1000"`
	DistanceMeters   float64            `json:"distance_meters" example:"475.2"`
	ExpiresAt        time.Time          `json:"expires_at" example:"2026-05-12T12:00:15Z"`
	AllowedActions   []string           `json:"allowed_actions" example:"accept,reject"`
}

type OrderRoutePointResponse struct {
	ID             uuid.UUID           `json:"id" example:"88888888-8888-8888-8888-888888888888"`
	Location       CoordinatesResponse `json:"location"`
	Heading        *float64            `json:"heading,omitempty" example:"181.5"`
	SpeedMPS       *float64            `json:"speed_mps,omitempty" example:"9.4"`
	AccuracyMeters *float64            `json:"accuracy_meters,omitempty" example:"7.2"`
	RecordedAt     time.Time           `json:"recorded_at" example:"2026-05-12T12:10:00Z"`
}

type OrderRouteResponse struct {
	OrderID uuid.UUID                 `json:"order_id" example:"44444444-4444-4444-4444-444444444444"`
	Points  []OrderRoutePointResponse `json:"points"`
}

type DriverLocationBatchRequest struct {
	Locations []DriverLocationRequest `json:"locations" binding:"required,min=1,max=50"`
}

type CompleteOrderRequest struct {
	FinalPrice int64  `json:"final_price" binding:"required,min=0" example:"260"`
	Currency   string `json:"currency" binding:"required" example:"RUB"`
}

type RejectOrderRequest struct {
	Reason string `json:"reason" binding:"required" example:"Too far"`
}

type AuthLoginRequest struct {
	Phone    string          `json:"phone" binding:"required" example:"+79990000000"`
	Password string          `json:"password" binding:"required" example:"strong-password"`
	Role     domain.UserRole `json:"role" binding:"required,oneof=passenger driver taxi_park admin dispatcher" example:"passenger"`
}

type AuthEmailCodeRequest struct {
	Email string          `json:"email" binding:"required,email" example:"user@example.com"`
	Role  domain.UserRole `json:"role" binding:"required,oneof=passenger driver taxi_park admin dispatcher" example:"passenger"`
}

type AuthEmailVerifyRequest struct {
	Email string          `json:"email" binding:"required,email" example:"user@example.com"`
	Role  domain.UserRole `json:"role" binding:"required,oneof=passenger driver taxi_park admin dispatcher" example:"passenger"`
	Code  string          `json:"code" binding:"required,len=6" example:"123456"`
}

type AuthVerifyCodeRequest struct {
	Phone string          `json:"phone,omitempty" example:"+79990000000"`
	Email string          `json:"email,omitempty" example:"user@example.com"`
	Role  domain.UserRole `json:"role" binding:"required,oneof=passenger driver taxi_park admin dispatcher" example:"passenger"`
	Code  string          `json:"code" binding:"required,len=6" example:"123456"`
}

type AuthCodeSentResponse struct {
	DeliveryChannel string `json:"delivery_channel" example:"sms"`
	Message         string `json:"message" example:"verification code sent"`
	DebugCode       string `json:"debug_code,omitempty" example:"123456"`
}

type AuthTokenResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOi..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOi..."`
	TokenType    string `json:"token_type" example:"Bearer"`
	ExpiresIn    int64  `json:"expires_in" example:"900"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOi..."`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required" example:"eyJhbGciOi..."`
}

type WSMessage struct {
	Event      string         `json:"event" example:"order.driver_assigned"`
	RequestID  uuid.UUID      `json:"request_id" example:"11111111-1111-1111-1111-111111111111"`
	OccurredAt time.Time      `json:"occurred_at" example:"2026-05-12T12:00:00Z"`
	Payload    map[string]any `json:"payload"`
}

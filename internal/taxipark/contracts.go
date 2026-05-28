package taxipark

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type Repository interface {
	GetSettingsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkSettings, error)
	UpdateSettingsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error)
	ListTariffsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) ([]domain.TaxiParkTariff, error)
	CreateTariffByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error)
	UpdateTariffByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, tariffID uuid.UUID, request dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error)
	CreateOrderByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, record CreateOrderRecord) (domain.Order, error)
	GetOrderByActorUserID(ctx context.Context, actorUserID uuid.UUID, orderID uuid.UUID) (domain.Order, error)
	UpdateOrderByActorUserID(ctx context.Context, actorUserID uuid.UUID, orderID uuid.UUID, record UpdateOrderRecord) (domain.Order, error)
	CancelOrderByActorUserID(ctx context.Context, actorUserID uuid.UUID, orderID uuid.UUID, reason string) (domain.Order, error)
	CompleteOrderByActorUserID(ctx context.Context, actorUserID uuid.UUID, orderID uuid.UUID, finalPriceCents int64) (domain.Order, error)
	CreateDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, record CreateDriverRecord) (CreateDriverResult, error)
	ListDriverLocationsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, maxAge time.Duration) ([]DriverLocation, error)
	UpdateDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, record UpdateDriverRecord) (CreateDriverResult, error)
	UpdateDriverPasswordByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, passwordHash string) error
	BlockDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, reason string) error
	UnblockDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error
	ArchiveDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error
	ListCarsByDriverAndOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) ([]domain.Car, error)
	ListCarsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) ([]domain.Car, error)
	CreateCarByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, record CarRecord) (domain.Car, error)
	UpdateCarByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID, record CarPatchRecord) (domain.Car, error)
	ArchiveCarByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID) error
	AttachCarToDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error
	AssignCarToDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error
	DetachCarFromDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error
	ListDriverDocumentsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) ([]domain.TaxiParkDocument, error)
	ListCarDocumentsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID) ([]domain.TaxiParkDocument, error)
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type DispatchController interface {
	EnqueueOrder(ctx context.Context, orderID uuid.UUID) error
	StopDispatch(ctx context.Context, orderID uuid.UUID) error
}

type RealtimeGateway interface {
	SendToDriver(ctx context.Context, driverID uuid.UUID, eventName string, payload any) error
	SendToPassenger(ctx context.Context, passengerID uuid.UUID, eventName string, payload any) error
	SendToTaxiParkByOrder(ctx context.Context, orderID uuid.UUID, eventName string, payload any) error
}

type FinanceProcessor interface {
	SettleCompletedOrder(ctx context.Context, orderID uuid.UUID) (domain.OrderSettlement, error)
}

type CreateOrderRecord struct {
	PassengerPhone      string
	PassengerName       string
	TariffID            uuid.UUID
	PickupAddress       string
	PickupLocation      domain.Coordinates
	DestinationAddress  string
	DestinationLocation *domain.Coordinates
	PaymentMethod       domain.PaymentMethod
	Comment             string
}

type UpdateOrderRecord struct {
	PickupAddress       *string
	PickupLocation      *domain.Coordinates
	DestinationAddress  *string
	DestinationLocation *domain.Coordinates
	PaymentMethod       *domain.PaymentMethod
	Comment             *string
}

type CreateDriverRecord struct {
	Phone                         string
	Email                         string
	FirstName                     string
	LastName                      string
	BirthDate                     *time.Time
	LicenseSeries                 string
	PasswordHash                  string
	LicenseNumber                 string
	LicenseCategory               string
	LicenseIssuedAt               *time.Time
	LicenseExpiresAt              *time.Time
	DrivingExperienceFrom         *time.Time
	HasNoTaxiWorkRestrictions     bool
	FederalLaw580Compliant        bool
	RegionalRequirementsCompliant bool
	MedicalCheckPassed            bool
	PretripControlRequired        bool
	PretripControlPassed          bool
	NoTransportBan                bool
	VerificationStatus            domain.VerificationLifecycleStatus
	TaxiParkComment               string
	AttachedCarID                 *uuid.UUID
}

type DriverLocation struct {
	DriverID           uuid.UUID
	UserID             uuid.UUID
	Name               string
	Phone              string
	Status             domain.DriverStatus
	VerificationStatus domain.VerificationLifecycleStatus
	Rating             float64
	Location           *domain.Coordinates
	Heading            *int16
	SpeedMPS           *float64
	AccuracyMeters     *float64
	RecordedAt         *time.Time
	UpdatedAt          *time.Time
	IsStale            bool
	Car                *DriverLocationCar
}

type DriverLocationCar struct {
	ID                 uuid.UUID
	Brand              string
	Model              string
	PlateNumber        string
	Color              string
	CarClass           string
	VerificationStatus domain.VerificationLifecycleStatus
	IsActive           bool
}

type UpdateDriverRecord struct {
	FirstName                     *string
	LastName                      *string
	BirthDate                     *time.Time
	LicenseSeries                 *string
	LicenseNumber                 *string
	LicenseCategory               *string
	LicenseIssuedAt               *time.Time
	LicenseExpiresAt              *time.Time
	DrivingExperienceFrom         *time.Time
	HasNoTaxiWorkRestrictions     *bool
	FederalLaw580Compliant        *bool
	RegionalRequirementsCompliant *bool
	MedicalCheckPassed            *bool
	PretripControlRequired        *bool
	PretripControlPassed          *bool
	NoTransportBan                *bool
	VerificationStatus            *domain.VerificationLifecycleStatus
	TaxiParkComment               *string
	AttachedCarID                 *uuid.UUID
}

type CreateDriverResult struct {
	UserID                        uuid.UUID
	DriverID                      uuid.UUID
	TaxiParkID                    uuid.UUID
	Phone                         string
	Email                         string
	FirstName                     string
	LastName                      string
	Status                        domain.DriverStatus
	VerificationStatus            domain.VerificationLifecycleStatus
	Rating                        float64
	RatingsCount                  int
	BirthDate                     *time.Time
	LicenseSeries                 string
	LicenseNumber                 string
	LicenseCategory               string
	LicenseIssuedAt               *time.Time
	LicenseExpiresAt              *time.Time
	DrivingExperienceFrom         *time.Time
	HasNoTaxiWorkRestrictions     bool
	FederalLaw580Compliant        bool
	RegionalRequirementsCompliant bool
	MedicalCheckPassed            bool
	PretripControlRequired        bool
	PretripControlPassed          bool
	NoTransportBan                bool
	VerificationCheckedAt         *time.Time
	VerificationCheckedBy         *uuid.UUID
	IsVerified                    bool
	TaxiParkComment               string
	GeneratedPassword             string
	PasswordGenerated             bool
}

type CarRecord struct {
	PrimaryDriverID               *uuid.UUID
	AttachedDriverIDs             []uuid.UUID
	Brand                         string
	Model                         string
	Year                          int
	PlateNumber                   string
	VIN                           string
	STS                           string
	PTS                           string
	Color                         string
	CarClass                      string
	VerificationStatus            domain.VerificationLifecycleStatus
	OwnerDetails                  string
	OSAGOExpiresAt                *time.Time
	DiagnosticCardExpiresAt       *time.Time
	TaxiPermitNumber              string
	RegionalRegistryNumber        string
	PermitRegion                  string
	PermitIssuedAt                *time.Time
	PermitExpiresAt               *time.Time
	TaxiPermitVerified            bool
	RegionalRegistryVerified      bool
	RegionalRequirementsCompliant bool
	HasTaxiColorScheme            bool
	HasOrangeRoofLamp             bool
	HasPassengerInfo              bool
	OSAGOVerified                 bool
	DiagnosticCardVerified        bool
	TechnicalStateVerified        bool
	LocalizationCompliant         bool
	LegalUseBasisVerified         bool
	IsActive                      bool
}

type CarPatchRecord struct {
	PrimaryDriverID               *uuid.UUID
	AttachedDriverIDs             []uuid.UUID
	Brand                         *string
	Model                         *string
	Year                          *int
	PlateNumber                   *string
	VIN                           *string
	STS                           *string
	PTS                           *string
	Color                         *string
	CarClass                      *string
	VerificationStatus            *domain.VerificationLifecycleStatus
	OwnerDetails                  *string
	OSAGOExpiresAt                *time.Time
	DiagnosticCardExpiresAt       *time.Time
	TaxiPermitNumber              *string
	RegionalRegistryNumber        *string
	PermitRegion                  *string
	PermitIssuedAt                *time.Time
	PermitExpiresAt               *time.Time
	TaxiPermitVerified            *bool
	RegionalRegistryVerified      *bool
	RegionalRequirementsCompliant *bool
	HasTaxiColorScheme            *bool
	HasOrangeRoofLamp             *bool
	HasPassengerInfo              *bool
	OSAGOVerified                 *bool
	DiagnosticCardVerified        *bool
	TechnicalStateVerified        *bool
	LocalizationCompliant         *bool
	LegalUseBasisVerified         *bool
	IsActive                      *bool
}

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
	CreateDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, record CreateDriverRecord) (CreateDriverResult, error)
	UpdateDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, record UpdateDriverRecord) (CreateDriverResult, error)
	BlockDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, reason string) error
	ArchiveDriverByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error
	ListCarsByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) ([]domain.Car, error)
	CreateCarByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, record CarRecord) (domain.Car, error)
	UpdateCarByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID, record CarPatchRecord) (domain.Car, error)
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type CreateDriverRecord struct {
	Phone                 string
	Email                 string
	FirstName             string
	LastName              string
	BirthDate             *time.Time
	LicenseSeries         string
	PasswordHash          string
	LicenseNumber         string
	LicenseCategory       string
	LicenseIssuedAt       *time.Time
	LicenseExpiresAt      *time.Time
	DrivingExperienceFrom *time.Time
	HasNoTaxiWorkRestrictions  bool
	FederalLaw580Compliant     bool
	RegionalRequirementsCompliant bool
	MedicalCheckPassed         bool
	PretripControlRequired     bool
	PretripControlPassed       bool
	NoTransportBan             bool
	VerificationStatus    domain.VerificationLifecycleStatus
	TaxiParkComment       string
	AttachedCarID         *uuid.UUID
}

type UpdateDriverRecord struct {
	FirstName             *string
	LastName              *string
	BirthDate             *time.Time
	LicenseSeries         *string
	LicenseNumber         *string
	LicenseCategory       *string
	LicenseIssuedAt       *time.Time
	LicenseExpiresAt      *time.Time
	DrivingExperienceFrom *time.Time
	HasNoTaxiWorkRestrictions  *bool
	FederalLaw580Compliant     *bool
	RegionalRequirementsCompliant *bool
	MedicalCheckPassed         *bool
	PretripControlRequired     *bool
	PretripControlPassed       *bool
	NoTransportBan             *bool
	VerificationStatus    *domain.VerificationLifecycleStatus
	TaxiParkComment       *string
	AttachedCarID         *uuid.UUID
}

type CreateDriverResult struct {
	UserID                uuid.UUID
	DriverID              uuid.UUID
	TaxiParkID            uuid.UUID
	Phone                 string
	Email                 string
	FirstName             string
	LastName              string
	Status                domain.DriverStatus
	VerificationStatus    domain.VerificationLifecycleStatus
	Rating                float64
	RatingsCount          int
	BirthDate             *time.Time
	LicenseSeries         string
	LicenseNumber         string
	LicenseCategory       string
	LicenseIssuedAt       *time.Time
	LicenseExpiresAt      *time.Time
	DrivingExperienceFrom *time.Time
	HasNoTaxiWorkRestrictions  bool
	FederalLaw580Compliant     bool
	RegionalRequirementsCompliant bool
	MedicalCheckPassed         bool
	PretripControlRequired     bool
	PretripControlPassed       bool
	NoTransportBan             bool
	VerificationCheckedAt      *time.Time
	VerificationCheckedBy      *uuid.UUID
	IsVerified            bool
	TaxiParkComment       string
	GeneratedPassword     string
	PasswordGenerated     bool
}

type CarRecord struct {
	PrimaryDriverID         *uuid.UUID
	AttachedDriverIDs       []uuid.UUID
	Brand                   string
	Model                   string
	Year                    int
	PlateNumber             string
	VIN                     string
	STS                     string
	PTS                     string
	Color                   string
	CarClass                string
	VerificationStatus      domain.VerificationLifecycleStatus
	OwnerDetails            string
	OSAGOExpiresAt          *time.Time
	DiagnosticCardExpiresAt *time.Time
	TaxiPermitNumber        string
	RegionalRegistryNumber  string
	PermitRegion            string
	PermitIssuedAt          *time.Time
	PermitExpiresAt         *time.Time
	TaxiPermitVerified      bool
	RegionalRegistryVerified bool
	RegionalRequirementsCompliant bool
	HasTaxiColorScheme      bool
	HasOrangeRoofLamp       bool
	HasPassengerInfo        bool
	OSAGOVerified           bool
	DiagnosticCardVerified  bool
	TechnicalStateVerified  bool
	LocalizationCompliant   bool
	LegalUseBasisVerified   bool
	IsActive                bool
}

type CarPatchRecord struct {
	PrimaryDriverID         *uuid.UUID
	AttachedDriverIDs       []uuid.UUID
	Brand                   *string
	Model                   *string
	Year                    *int
	PlateNumber             *string
	VIN                     *string
	STS                     *string
	PTS                     *string
	Color                   *string
	CarClass                *string
	VerificationStatus      *domain.VerificationLifecycleStatus
	OwnerDetails            *string
	OSAGOExpiresAt          *time.Time
	DiagnosticCardExpiresAt *time.Time
	TaxiPermitNumber        *string
	RegionalRegistryNumber  *string
	PermitRegion            *string
	PermitIssuedAt          *time.Time
	PermitExpiresAt         *time.Time
	TaxiPermitVerified      *bool
	RegionalRegistryVerified *bool
	RegionalRequirementsCompliant *bool
	HasTaxiColorScheme      *bool
	HasOrangeRoofLamp       *bool
	HasPassengerInfo        *bool
	OSAGOVerified           *bool
	DiagnosticCardVerified  *bool
	TechnicalStateVerified  *bool
	LocalizationCompliant   *bool
	LegalUseBasisVerified   *bool
	IsActive                *bool
}

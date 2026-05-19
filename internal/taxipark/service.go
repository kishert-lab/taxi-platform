package taxipark

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

const defaultDriverPasswordLength = 18

var (
	ErrTaxiParkNotFound          = errors.New("taxi park not found")
	ErrDriverPhoneAlreadyExists  = errors.New("driver phone already exists")
	ErrInvalidDriverCreateFields = errors.New("invalid driver create fields")
)

type Service struct {
	repository     Repository
	passwordHasher PasswordHasher
}

func NewService(repository Repository, passwordHasher PasswordHasher) *Service {
	return &Service{repository: repository, passwordHasher: passwordHasher}
}

func (service *Service) GetSettings(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkSettings, error) {
	return service.repository.GetSettingsByOwnerUserID(ctx, ownerUserID)
}

func (service *Service) UpdateSettings(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error) {
	return service.repository.UpdateSettingsByOwnerUserID(ctx, ownerUserID, request)
}

func (service *Service) ListTariffs(ctx context.Context, ownerUserID uuid.UUID) ([]domain.TaxiParkTariff, error) {
	return service.repository.ListTariffsByOwnerUserID(ctx, ownerUserID)
}

func (service *Service) CreateTariff(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error) {
	return service.repository.CreateTariffByOwnerUserID(ctx, ownerUserID, request)
}

func (service *Service) UpdateTariff(ctx context.Context, ownerUserID uuid.UUID, tariffID uuid.UUID, request dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error) {
	return service.repository.UpdateTariffByOwnerUserID(ctx, ownerUserID, tariffID, request)
}

func (service *Service) CreateDriver(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkCreateDriverRequest) (CreateDriverResult, error) {
	phone, err := domain.NormalizePhone(request.Phone)
	if err != nil {
		return CreateDriverResult{}, fmt.Errorf("normalize taxi park driver phone: %w", err)
	}

	email := ""
	if strings.TrimSpace(request.Email) != "" {
		email, err = domain.NormalizeEmail(request.Email)
		if err != nil {
			return CreateDriverResult{}, fmt.Errorf("normalize taxi park driver email: %w", err)
		}
	}

	birthDate, err := parseOptionalDate(request.BirthDate)
	if err != nil {
		return CreateDriverResult{}, err
	}
	licenseIssuedAt, err := parseOptionalDate(request.LicenseIssuedAt)
	if err != nil {
		return CreateDriverResult{}, err
	}
	licenseExpiresAt, err := parseOptionalDate(request.LicenseExpiresAt)
	if err != nil {
		return CreateDriverResult{}, err
	}
	drivingExperienceFrom, err := parseOptionalDate(request.DrivingExperienceFrom)
	if err != nil {
		return CreateDriverResult{}, err
	}
	verificationStatus := request.VerificationStatus
	if verificationStatus == "" {
		verificationStatus = domain.ComplianceStatusPendingVerification
	}
	if err := verificationStatus.Validate(); err != nil {
		return CreateDriverResult{}, err
	}

	password := strings.TrimSpace(request.Password)
	passwordGenerated := false
	if password == "" {
		password, err = generateTemporaryPassword(defaultDriverPasswordLength)
		if err != nil {
			return CreateDriverResult{}, err
		}
		passwordGenerated = true
	}

	passwordHash, err := service.passwordHasher.HashPassword(password)
	if err != nil {
		return CreateDriverResult{}, fmt.Errorf("hash taxi park driver password: %w", err)
	}

	result, err := service.repository.CreateDriverByOwnerUserID(ctx, ownerUserID, CreateDriverRecord{
		Phone:                 phone,
		Email:                 email,
		FirstName:             strings.TrimSpace(request.FirstName),
		LastName:              strings.TrimSpace(request.LastName),
		BirthDate:             birthDate,
		LicenseSeries:         strings.TrimSpace(request.LicenseSeries),
		PasswordHash:          passwordHash,
		LicenseNumber:         strings.TrimSpace(request.LicenseNumber),
		LicenseCategory:       strings.TrimSpace(request.LicenseCategory),
		LicenseIssuedAt:       licenseIssuedAt,
		LicenseExpiresAt:      licenseExpiresAt,
		DrivingExperienceFrom: drivingExperienceFrom,
		HasNoTaxiWorkRestrictions:     request.HasNoTaxiWorkRestrictions,
		FederalLaw580Compliant:        request.FederalLaw580Compliant,
		RegionalRequirementsCompliant: request.RegionalRequirementsCompliant,
		MedicalCheckPassed:            request.MedicalCheckPassed,
		PretripControlRequired:        request.PretripControlRequired,
		PretripControlPassed:          request.PretripControlPassed,
		NoTransportBan:                request.NoTransportBan,
		VerificationStatus:    verificationStatus,
		TaxiParkComment:       strings.TrimSpace(request.TaxiParkComment),
		AttachedCarID:         request.AttachedCarID,
	})
	if err != nil {
		return CreateDriverResult{}, err
	}
	result.PasswordGenerated = passwordGenerated
	if passwordGenerated {
		result.GeneratedPassword = password
	}

	return result, nil
}

func (service *Service) UpdateDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, request dto.TaxiParkUpdateDriverRequest) (CreateDriverResult, error) {
	record := UpdateDriverRecord{
		FirstName:          trimStringPointer(request.FirstName),
		LastName:           trimStringPointer(request.LastName),
		LicenseSeries:      trimStringPointer(request.LicenseSeries),
		LicenseNumber:      trimStringPointer(request.LicenseNumber),
		LicenseCategory:    trimStringPointer(request.LicenseCategory),
		TaxiParkComment:    trimStringPointer(request.TaxiParkComment),
		VerificationStatus: request.VerificationStatus,
		AttachedCarID:      request.AttachedCarID,
		HasNoTaxiWorkRestrictions:     request.HasNoTaxiWorkRestrictions,
		FederalLaw580Compliant:        request.FederalLaw580Compliant,
		RegionalRequirementsCompliant: request.RegionalRequirementsCompliant,
		MedicalCheckPassed:            request.MedicalCheckPassed,
		PretripControlRequired:        request.PretripControlRequired,
		PretripControlPassed:          request.PretripControlPassed,
		NoTransportBan:                request.NoTransportBan,
	}
	var err error
	record.BirthDate, err = parseOptionalDatePointer(request.BirthDate)
	if err != nil {
		return CreateDriverResult{}, err
	}
	record.LicenseIssuedAt, err = parseOptionalDatePointer(request.LicenseIssuedAt)
	if err != nil {
		return CreateDriverResult{}, err
	}
	record.LicenseExpiresAt, err = parseOptionalDatePointer(request.LicenseExpiresAt)
	if err != nil {
		return CreateDriverResult{}, err
	}
	record.DrivingExperienceFrom, err = parseOptionalDatePointer(request.DrivingExperienceFrom)
	if err != nil {
		return CreateDriverResult{}, err
	}
	if record.VerificationStatus != nil {
		if err := (*record.VerificationStatus).Validate(); err != nil {
			return CreateDriverResult{}, err
		}
	}

	return service.repository.UpdateDriverByOwnerUserID(ctx, ownerUserID, driverID, record)
}

func (service *Service) BlockDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, reason string) error {
	return service.repository.BlockDriverByOwnerUserID(ctx, ownerUserID, driverID, strings.TrimSpace(reason))
}

func (service *Service) ArchiveDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error {
	return service.repository.ArchiveDriverByOwnerUserID(ctx, ownerUserID, driverID)
}

func (service *Service) ListCars(ctx context.Context, ownerUserID uuid.UUID) ([]domain.Car, error) {
	return service.repository.ListCarsByOwnerUserID(ctx, ownerUserID)
}

func (service *Service) CreateCar(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkCarRequest) (domain.Car, error) {
	record, err := carRecordFromRequest(request)
	if err != nil {
		return domain.Car{}, err
	}
	return service.repository.CreateCarByOwnerUserID(ctx, ownerUserID, record)
}

func (service *Service) UpdateCar(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID, request dto.TaxiParkCarPatchRequest) (domain.Car, error) {
	record, err := carPatchRecordFromRequest(request)
	if err != nil {
		return domain.Car{}, err
	}
	return service.repository.UpdateCarByOwnerUserID(ctx, ownerUserID, carID, record)
}

func generateTemporaryPassword(length int) (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz23456789!@#$%*-_"
	if length <= 0 {
		return "", ErrInvalidDriverCreateFields
	}

	password := make([]byte, length)
	for index := range password {
		value, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			return "", fmt.Errorf("generate taxi park driver password: %w", err)
		}
		password[index] = alphabet[value.Int64()]
	}

	return string(password), nil
}

func parseOptionalDate(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.DateOnly, trimmed)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid date %q", ErrInvalidDriverCreateFields, value)
	}
	return &parsed, nil
}

func parseOptionalDatePointer(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	return parseOptionalDate(*value)
}

func trimStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func carRecordFromRequest(request dto.TaxiParkCarRequest) (CarRecord, error) {
	status := request.VerificationStatus
	if status == "" {
		status = domain.ComplianceStatusPendingVerification
	}
	if err := status.Validate(); err != nil {
		return CarRecord{}, err
	}
	osagoExpiresAt, err := parseOptionalDate(request.OSAGOExpiresAt)
	if err != nil {
		return CarRecord{}, err
	}
	diagnosticExpiresAt, err := parseOptionalDate(request.DiagnosticCardExpiresAt)
	if err != nil {
		return CarRecord{}, err
	}
	permitIssuedAt, err := parseOptionalDate(request.PermitIssuedAt)
	if err != nil {
		return CarRecord{}, err
	}
	permitExpiresAt, err := parseOptionalDate(request.PermitExpiresAt)
	if err != nil {
		return CarRecord{}, err
	}
	isActive := true
	if request.IsActive != nil {
		isActive = *request.IsActive
	}
	return CarRecord{
		PrimaryDriverID:         request.PrimaryDriverID,
		AttachedDriverIDs:       request.AttachedDriverIDs,
		Brand:                   strings.TrimSpace(request.Brand),
		Model:                   strings.TrimSpace(request.Model),
		Year:                    request.Year,
		PlateNumber:             strings.TrimSpace(request.PlateNumber),
		VIN:                     strings.TrimSpace(request.VIN),
		STS:                     strings.TrimSpace(request.STS),
		PTS:                     strings.TrimSpace(request.PTS),
		Color:                   strings.TrimSpace(request.Color),
		CarClass:                strings.TrimSpace(request.CarClass),
		VerificationStatus:      status,
		OwnerDetails:            strings.TrimSpace(request.OwnerDetails),
		OSAGOExpiresAt:          osagoExpiresAt,
		DiagnosticCardExpiresAt: diagnosticExpiresAt,
		TaxiPermitNumber:        strings.TrimSpace(request.TaxiPermitNumber),
		RegionalRegistryNumber:  strings.TrimSpace(request.RegionalRegistryNumber),
		PermitRegion:            strings.TrimSpace(request.PermitRegion),
		PermitIssuedAt:          permitIssuedAt,
		PermitExpiresAt:         permitExpiresAt,
		TaxiPermitVerified:      request.TaxiPermitVerified,
		RegionalRegistryVerified:        request.RegionalRegistryVerified,
		RegionalRequirementsCompliant:   request.RegionalRequirementsCompliant,
		HasTaxiColorScheme:      request.HasTaxiColorScheme,
		HasOrangeRoofLamp:       request.HasOrangeRoofLamp,
		HasPassengerInfo:        request.HasPassengerInfo,
		OSAGOVerified:           request.OSAGOVerified,
		DiagnosticCardVerified:  request.DiagnosticCardVerified,
		TechnicalStateVerified:  request.TechnicalStateVerified,
		LocalizationCompliant:   request.LocalizationCompliant,
		LegalUseBasisVerified:   request.LegalUseBasisVerified,
		IsActive:                isActive,
	}, nil
}

func carPatchRecordFromRequest(request dto.TaxiParkCarPatchRequest) (CarPatchRecord, error) {
	record := CarPatchRecord{
		PrimaryDriverID:        request.PrimaryDriverID,
		AttachedDriverIDs:      request.AttachedDriverIDs,
		Brand:                  trimStringPointer(request.Brand),
		Model:                  trimStringPointer(request.Model),
		Year:                   request.Year,
		PlateNumber:            trimStringPointer(request.PlateNumber),
		VIN:                    trimStringPointer(request.VIN),
		STS:                    trimStringPointer(request.STS),
		PTS:                    trimStringPointer(request.PTS),
		Color:                  trimStringPointer(request.Color),
		CarClass:               trimStringPointer(request.CarClass),
		VerificationStatus:     request.VerificationStatus,
		OwnerDetails:           trimStringPointer(request.OwnerDetails),
		TaxiPermitNumber:       trimStringPointer(request.TaxiPermitNumber),
		RegionalRegistryNumber: trimStringPointer(request.RegionalRegistryNumber),
		PermitRegion:           trimStringPointer(request.PermitRegion),
		TaxiPermitVerified:     request.TaxiPermitVerified,
		RegionalRegistryVerified:       request.RegionalRegistryVerified,
		RegionalRequirementsCompliant:  request.RegionalRequirementsCompliant,
		HasTaxiColorScheme:     request.HasTaxiColorScheme,
		HasOrangeRoofLamp:      request.HasOrangeRoofLamp,
		HasPassengerInfo:       request.HasPassengerInfo,
		OSAGOVerified:          request.OSAGOVerified,
		DiagnosticCardVerified: request.DiagnosticCardVerified,
		TechnicalStateVerified: request.TechnicalStateVerified,
		LocalizationCompliant:  request.LocalizationCompliant,
		LegalUseBasisVerified:  request.LegalUseBasisVerified,
		IsActive:               request.IsActive,
	}
	var err error
	record.OSAGOExpiresAt, err = parseOptionalDatePointer(request.OSAGOExpiresAt)
	if err != nil {
		return CarPatchRecord{}, err
	}
	record.DiagnosticCardExpiresAt, err = parseOptionalDatePointer(request.DiagnosticCardExpiresAt)
	if err != nil {
		return CarPatchRecord{}, err
	}
	record.PermitIssuedAt, err = parseOptionalDatePointer(request.PermitIssuedAt)
	if err != nil {
		return CarPatchRecord{}, err
	}
	record.PermitExpiresAt, err = parseOptionalDatePointer(request.PermitExpiresAt)
	if err != nil {
		return CarPatchRecord{}, err
	}
	if record.VerificationStatus != nil {
		if err := (*record.VerificationStatus).Validate(); err != nil {
			return CarPatchRecord{}, err
		}
	}
	return record, nil
}

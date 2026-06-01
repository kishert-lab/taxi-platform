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

	dispatchapp "github.com/kishert-lab/taxi-platform/internal/dispatch"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

const defaultDriverPasswordLength = 18

var (
	ErrTaxiParkNotFound          = errors.New("taxi park not found")
	ErrTaxiParkResourceNotFound  = errors.New("taxi park resource not found")
	ErrTaxiParkResourceForbidden = errors.New("taxi park resource forbidden")
	ErrDriverPhoneAlreadyExists  = errors.New("driver phone already exists")
	ErrCarAlreadyExists          = errors.New("car already exists")
	ErrInvalidDriverCreateFields = errors.New("invalid driver create fields")
	ErrInvalidDriverPassword     = errors.New("invalid driver password")
	ErrInvalidOrderFields        = errors.New("invalid taxi park order fields")
	ErrOrderTariffNotFound       = errors.New("taxi park order tariff not found")
)

type Service struct {
	repository         Repository
	passwordHasher     PasswordHasher
	dispatchController DispatchController
	realtimeGateway    RealtimeGateway
	financeProcessor   FinanceProcessor
}

func NewService(repository Repository, passwordHasher PasswordHasher) *Service {
	return &Service{repository: repository, passwordHasher: passwordHasher}
}

func NewServiceWithDispatch(repository Repository, passwordHasher PasswordHasher, dispatchController DispatchController, realtimeGateways ...RealtimeGateway) *Service {
	service := &Service{repository: repository, passwordHasher: passwordHasher, dispatchController: dispatchController}
	if len(realtimeGateways) > 0 {
		service.realtimeGateway = realtimeGateways[0]
	}
	return service
}

func (service *Service) WithFinanceProcessor(financeProcessor FinanceProcessor) *Service {
	service.financeProcessor = financeProcessor
	return service
}

func (service *Service) GetSettings(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkSettings, error) {
	return service.repository.GetSettingsByOwnerUserID(ctx, ownerUserID)
}

func (service *Service) UpdateSettings(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error) {
	if err := validateDispatchSettingsPatch(request.Dispatch); err != nil {
		return domain.TaxiParkSettings{}, err
	}
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

func (service *Service) CreateOrder(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkCreateOrderRequest) (domain.Order, error) {
	record, err := orderRecordFromRequest(request)
	if err != nil {
		return domain.Order{}, err
	}
	settings, err := service.repository.GetSettingsByOwnerUserID(ctx, ownerUserID)
	if err != nil {
		return domain.Order{}, err
	}
	order, err := service.repository.CreateOrderByOwnerUserID(ctx, ownerUserID, record)
	if err != nil {
		return domain.Order{}, err
	}
	if service.dispatchController != nil {
		if err := service.dispatchController.EnqueueOrderWithConfig(ctx, order.ID, dispatchConfigFromTaxiParkSettings(settings)); err != nil {
			return domain.Order{}, fmt.Errorf("enqueue taxi park order dispatch: %w", err)
		}
	}
	return order, nil
}

func (service *Service) GetOrder(ctx context.Context, actorUserID uuid.UUID, orderID uuid.UUID) (domain.Order, error) {
	return service.repository.GetOrderByActorUserID(ctx, actorUserID, orderID)
}

func (service *Service) UpdateOrder(ctx context.Context, actorUserID uuid.UUID, orderID uuid.UUID, request dto.TaxiParkUpdateOrderRequest) (domain.Order, error) {
	record, err := updateOrderRecordFromRequest(request)
	if err != nil {
		return domain.Order{}, err
	}
	order, err := service.repository.UpdateOrderByActorUserID(ctx, actorUserID, orderID, record)
	if err != nil {
		return domain.Order{}, err
	}
	if err := service.publishOrderEvent(ctx, order, "order.updated"); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (service *Service) CancelOrder(ctx context.Context, actorUserID uuid.UUID, orderID uuid.UUID, request dto.CancelOrderRequest) (domain.Order, error) {
	order, err := service.repository.CancelOrderByActorUserID(ctx, actorUserID, orderID, strings.TrimSpace(request.Reason))
	if err != nil {
		return domain.Order{}, err
	}
	if service.dispatchController != nil {
		if err := service.dispatchController.StopDispatch(ctx, order.ID); err != nil {
			return domain.Order{}, fmt.Errorf("stop taxi park order dispatch after cancel: %w", err)
		}
	}
	if err := service.publishOrderEvent(ctx, order, "order.cancelled"); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (service *Service) CompleteOrder(ctx context.Context, actorUserID uuid.UUID, orderID uuid.UUID, request dto.TaxiParkCompleteOrderRequest) (domain.Order, error) {
	order, err := service.repository.CompleteOrderByActorUserID(ctx, actorUserID, orderID, request.FinalPrice)
	if err != nil {
		return domain.Order{}, err
	}
	if service.financeProcessor != nil {
		if _, err := service.financeProcessor.SettleCompletedOrder(ctx, order.ID); err != nil {
			return domain.Order{}, fmt.Errorf("settle taxi park completed order: %w", err)
		}
	}
	if err := service.publishOrderEvent(ctx, order, "order.completed"); err != nil {
		return domain.Order{}, err
	}
	return order, nil
}

func (service *Service) publishOrderEvent(ctx context.Context, order domain.Order, eventName string) error {
	if service.realtimeGateway == nil {
		return nil
	}
	payload := map[string]any{
		"order_id":  order.ID,
		"status":    order.Status,
		"version":   order.Version,
		"driver_id": order.DriverID,
	}
	if err := service.realtimeGateway.SendToTaxiParkByOrder(ctx, order.ID, eventName, payload); err != nil {
		return fmt.Errorf("publish taxi park order websocket event: %w", err)
	}
	if err := service.realtimeGateway.SendToPassenger(ctx, order.PassengerID, eventName, payload); err != nil {
		return fmt.Errorf("publish passenger order websocket event: %w", err)
	}
	if order.DriverID != nil {
		if err := service.realtimeGateway.SendToDriver(ctx, *order.DriverID, eventName, payload); err != nil {
			return fmt.Errorf("publish driver order websocket event: %w", err)
		}
	}
	return nil
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
		Phone:                         phone,
		Email:                         email,
		FirstName:                     strings.TrimSpace(request.FirstName),
		LastName:                      strings.TrimSpace(request.LastName),
		BirthDate:                     birthDate,
		LicenseSeries:                 strings.TrimSpace(request.LicenseSeries),
		PasswordHash:                  passwordHash,
		LicenseNumber:                 strings.TrimSpace(request.LicenseNumber),
		LicenseCategory:               strings.TrimSpace(request.LicenseCategory),
		LicenseIssuedAt:               licenseIssuedAt,
		LicenseExpiresAt:              licenseExpiresAt,
		DrivingExperienceFrom:         drivingExperienceFrom,
		HasNoTaxiWorkRestrictions:     request.HasNoTaxiWorkRestrictions,
		FederalLaw580Compliant:        request.FederalLaw580Compliant,
		RegionalRequirementsCompliant: request.RegionalRequirementsCompliant,
		MedicalCheckPassed:            request.MedicalCheckPassed,
		PretripControlRequired:        request.PretripControlRequired,
		PretripControlPassed:          request.PretripControlPassed,
		NoTransportBan:                request.NoTransportBan,
		VerificationStatus:            verificationStatus,
		TaxiParkComment:               strings.TrimSpace(request.TaxiParkComment),
		AttachedCarID:                 request.AttachedCarID,
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

func (service *Service) ListDriverLocations(ctx context.Context, ownerUserID uuid.UUID, maxAge time.Duration) ([]DriverLocation, error) {
	if maxAge <= 0 {
		maxAge = 30 * time.Second
	}
	return service.repository.ListDriverLocationsByOwnerUserID(ctx, ownerUserID, maxAge)
}

func (service *Service) UpdateDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, request dto.TaxiParkUpdateDriverRequest) (CreateDriverResult, error) {
	record := UpdateDriverRecord{
		FirstName:                     trimStringPointer(request.FirstName),
		LastName:                      trimStringPointer(request.LastName),
		LicenseSeries:                 trimStringPointer(request.LicenseSeries),
		LicenseNumber:                 trimStringPointer(request.LicenseNumber),
		LicenseCategory:               trimStringPointer(request.LicenseCategory),
		TaxiParkComment:               trimStringPointer(request.TaxiParkComment),
		VerificationStatus:            request.VerificationStatus,
		AttachedCarID:                 request.AttachedCarID,
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

func (service *Service) UpdateDriverPassword(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, request dto.TaxiParkDriverPasswordRequest) error {
	password := strings.TrimSpace(request.Password)
	if len(password) < 8 {
		return ErrInvalidDriverPassword
	}
	passwordHash, err := service.passwordHasher.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash taxi park driver password: %w", err)
	}
	return service.repository.UpdateDriverPasswordByOwnerUserID(ctx, ownerUserID, driverID, passwordHash)
}

func (service *Service) BlockDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, reason string) error {
	return service.repository.BlockDriverByOwnerUserID(ctx, ownerUserID, driverID, strings.TrimSpace(reason))
}

func (service *Service) ArchiveDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error {
	return service.repository.ArchiveDriverByOwnerUserID(ctx, ownerUserID, driverID)
}

func (service *Service) UnblockDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error {
	return service.repository.UnblockDriverByOwnerUserID(ctx, ownerUserID, driverID)
}

func (service *Service) ListCars(ctx context.Context, ownerUserID uuid.UUID) ([]domain.Car, error) {
	return service.repository.ListCarsByOwnerUserID(ctx, ownerUserID)
}

func (service *Service) ListTaxiParkDriverCars(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) ([]domain.Car, error) {
	return service.repository.ListCarsByDriverAndOwnerUserID(ctx, ownerUserID, driverID)
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

func (service *Service) VerifyCar(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID) (domain.Car, error) {
	verifiedStatus := domain.ComplianceStatusVerified
	return service.repository.UpdateCarByOwnerUserID(ctx, ownerUserID, carID, CarPatchRecord{
		VerificationStatus: &verifiedStatus,
	})
}

func (service *Service) ArchiveCar(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID) error {
	return service.repository.ArchiveCarByOwnerUserID(ctx, ownerUserID, carID)
}

func (service *Service) AttachCarToDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error {
	return service.repository.AttachCarToDriverByOwnerUserID(ctx, ownerUserID, driverID, carID)
}

func (service *Service) AssignCarToDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error {
	return service.repository.AssignCarToDriverByOwnerUserID(ctx, ownerUserID, driverID, carID)
}

func (service *Service) DetachCarFromDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, carID uuid.UUID) error {
	return service.repository.DetachCarFromDriverByOwnerUserID(ctx, ownerUserID, driverID, carID)
}

func (service *Service) ListDriverDocuments(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) ([]domain.TaxiParkDocument, error) {
	return service.repository.ListDriverDocumentsByOwnerUserID(ctx, ownerUserID, driverID)
}

func (service *Service) ListCarDocuments(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID) ([]domain.TaxiParkDocument, error) {
	return service.repository.ListCarDocumentsByOwnerUserID(ctx, ownerUserID, carID)
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
	if parsed, err := time.Parse(time.DateOnly, trimmed); err == nil {
		return &parsed, nil
	}
	if parsed, err := time.Parse(time.RFC3339, trimmed); err == nil {
		return &parsed, nil
	}
	return nil, fmt.Errorf("%w: invalid date %q", ErrInvalidDriverCreateFields, value)
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

func orderRecordFromRequest(request dto.TaxiParkCreateOrderRequest) (CreateOrderRecord, error) {
	if request.TariffID == uuid.Nil {
		return CreateOrderRecord{}, fmt.Errorf("%w: tariff_id is required", ErrInvalidOrderFields)
	}
	if strings.TrimSpace(request.PickupAddress) == "" {
		return CreateOrderRecord{}, fmt.Errorf("%w: pickup_address is required", ErrInvalidOrderFields)
	}
	if request.PickupLocation == nil {
		return CreateOrderRecord{}, fmt.Errorf("%w: pickup_location is required", ErrInvalidOrderFields)
	}
	if strings.TrimSpace(request.DestinationAddress) == "" {
		return CreateOrderRecord{}, fmt.Errorf("%w: destination_address is required", ErrInvalidOrderFields)
	}
	if err := validateOrderCoordinates(*request.PickupLocation); err != nil {
		return CreateOrderRecord{}, fmt.Errorf("pickup_location: %w", err)
	}
	if isZeroOrderCoordinates(*request.PickupLocation) {
		return CreateOrderRecord{}, fmt.Errorf("%w: pickup_location must be geocoded", ErrInvalidOrderFields)
	}

	var destinationLocation *domain.Coordinates
	if request.DestinationLocation != nil && !isZeroOrderCoordinates(*request.DestinationLocation) {
		if err := validateOrderCoordinates(*request.DestinationLocation); err != nil {
			return CreateOrderRecord{}, fmt.Errorf("destination_location: %w", err)
		}
		destinationLocation = &domain.Coordinates{
			Latitude:  request.DestinationLocation.Latitude,
			Longitude: request.DestinationLocation.Longitude,
		}
	}

	paymentMethod := request.PaymentMethod
	if paymentMethod == "" {
		paymentMethod = request.PaymentType
	}
	if paymentMethod == "" {
		paymentMethod = domain.PaymentMethodCash
	}
	if err := paymentMethod.Validate(); err != nil {
		return CreateOrderRecord{}, err
	}

	passengerPhone := strings.TrimSpace(request.PassengerPhone)
	if passengerPhone != "" {
		normalizedPhone, err := domain.NormalizePhone(passengerPhone)
		if err != nil {
			return CreateOrderRecord{}, err
		}
		passengerPhone = normalizedPhone
	}

	return CreateOrderRecord{
		PassengerPhone: passengerPhone,
		PassengerName:  strings.TrimSpace(request.PassengerName),
		TariffID:       request.TariffID,
		PickupAddress:  strings.TrimSpace(request.PickupAddress),
		PickupLocation: domain.Coordinates{
			Latitude:  request.PickupLocation.Latitude,
			Longitude: request.PickupLocation.Longitude,
		},
		DestinationAddress:  strings.TrimSpace(request.DestinationAddress),
		DestinationLocation: destinationLocation,
		PaymentMethod:       paymentMethod,
		Comment:             strings.TrimSpace(request.Comment),
	}, nil
}

func updateOrderRecordFromRequest(request dto.TaxiParkUpdateOrderRequest) (UpdateOrderRecord, error) {
	record := UpdateOrderRecord{
		PickupAddress:      trimStringPointer(request.PickupAddress),
		DestinationAddress: trimStringPointer(request.DestinationAddress),
		Comment:            trimStringPointer(request.Comment),
	}
	if request.PickupLocation != nil {
		coordinates, err := domain.NewCoordinates(request.PickupLocation.Latitude, request.PickupLocation.Longitude)
		if err != nil {
			return UpdateOrderRecord{}, fmt.Errorf("%w: pickup_location is invalid", ErrInvalidOrderFields)
		}
		record.PickupLocation = &coordinates
	}
	if request.DestinationLocation != nil {
		coordinates, err := domain.NewCoordinates(request.DestinationLocation.Latitude, request.DestinationLocation.Longitude)
		if err != nil {
			return UpdateOrderRecord{}, fmt.Errorf("%w: destination_location is invalid", ErrInvalidOrderFields)
		}
		record.DestinationLocation = &coordinates
	}
	paymentMethod := request.PaymentMethod
	if paymentMethod == nil {
		paymentMethod = request.PaymentType
	}
	if paymentMethod != nil {
		if err := (*paymentMethod).Validate(); err != nil {
			return UpdateOrderRecord{}, fmt.Errorf("%w: payment_method is invalid", ErrInvalidOrderFields)
		}
		record.PaymentMethod = paymentMethod
	}
	return record, nil
}

func validateOrderCoordinates(coordinates dto.TaxiParkOrderCoordinatesRequest) error {
	if coordinates.Latitude < -90 || coordinates.Latitude > 90 {
		return fmt.Errorf("%w: latitude must be between -90 and 90", ErrInvalidOrderFields)
	}
	if coordinates.Longitude < -180 || coordinates.Longitude > 180 {
		return fmt.Errorf("%w: longitude must be between -180 and 180", ErrInvalidOrderFields)
	}
	return nil
}

func isZeroOrderCoordinates(coordinates dto.TaxiParkOrderCoordinatesRequest) bool {
	return coordinates.Latitude == 0 && coordinates.Longitude == 0
}

func validateDispatchSettingsPatch(dispatchSettings *dto.TaxiParkDispatchSettingsPatchRequest) error {
	if dispatchSettings == nil {
		return nil
	}
	if dispatchSettings.InitialRadiusMeters != nil && dispatchSettings.MaxRadiusMeters != nil && *dispatchSettings.MaxRadiusMeters < *dispatchSettings.InitialRadiusMeters {
		return fmt.Errorf("%w: dispatch.max_radius_meters must be greater than or equal to initial_radius_meters", ErrInvalidDriverCreateFields)
	}
	previousRadius := 0
	for _, radius := range dispatchSettings.RadiusAttemptsMeters {
		if radius <= 0 {
			return fmt.Errorf("%w: dispatch.radius_attempts_meters must contain positive values", ErrInvalidDriverCreateFields)
		}
		if previousRadius > 0 && radius < previousRadius {
			return fmt.Errorf("%w: dispatch.radius_attempts_meters must be sorted ascending", ErrInvalidDriverCreateFields)
		}
		previousRadius = radius
	}
	return nil
}

func dispatchConfigFromTaxiParkSettings(settings domain.TaxiParkSettings) dispatchapp.Config {
	return dispatchapp.Config{
		InitialRadiusMeters:  settings.DispatchInitialRadiusMeters,
		MaxRadiusMeters:      settings.DispatchMaxRadiusMeters,
		RadiusStepMeters:     settings.DispatchRadiusStepMeters,
		RadiusAttemptsMeters: settings.DispatchRadiusAttemptsMeters,
		MaxDriversPerOffer:   settings.DispatchMaxDriversPerOffer,
		DriverLocationMaxAge: time.Duration(settings.DispatchDriverLocationMaxAgeSec) * time.Second,
		OfferTTL:             time.Duration(settings.DispatchOfferTTLSec) * time.Second,
		AcceptLockTTL:        time.Duration(settings.DispatchAcceptLockTTLSec) * time.Second,
		WorkerPollTimeout:    time.Duration(settings.DispatchWorkerPollTimeoutSec) * time.Second,
		RecoveryInterval:     time.Duration(settings.DispatchRecoveryIntervalSec) * time.Second,
	}
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
	ownerDetails := request.OwnerDetails
	if strings.TrimSpace(ownerDetails) == "" {
		ownerDetails = request.OwnerOrLegalBasis
	}
	return CarRecord{
		PrimaryDriverID:               request.PrimaryDriverID,
		AttachedDriverIDs:             request.AttachedDriverIDs,
		Brand:                         strings.TrimSpace(request.Brand),
		Model:                         strings.TrimSpace(request.Model),
		Year:                          request.Year,
		PlateNumber:                   strings.TrimSpace(request.PlateNumber),
		VIN:                           strings.TrimSpace(request.VIN),
		STS:                           strings.TrimSpace(request.STS),
		PTS:                           strings.TrimSpace(request.PTS),
		Color:                         strings.TrimSpace(request.Color),
		CarClass:                      strings.TrimSpace(request.CarClass),
		VerificationStatus:            status,
		OwnerDetails:                  strings.TrimSpace(ownerDetails),
		OSAGOExpiresAt:                osagoExpiresAt,
		DiagnosticCardExpiresAt:       diagnosticExpiresAt,
		TaxiPermitNumber:              strings.TrimSpace(request.TaxiPermitNumber),
		RegionalRegistryNumber:        strings.TrimSpace(request.RegionalRegistryNumber),
		PermitRegion:                  strings.TrimSpace(request.PermitRegion),
		PermitIssuedAt:                permitIssuedAt,
		PermitExpiresAt:               permitExpiresAt,
		TaxiPermitVerified:            request.TaxiPermitVerified,
		RegionalRegistryVerified:      request.RegionalRegistryVerified,
		RegionalRequirementsCompliant: request.RegionalRequirementsCompliant,
		HasTaxiColorScheme:            request.HasTaxiColorScheme,
		HasOrangeRoofLamp:             request.HasOrangeRoofLamp,
		HasPassengerInfo:              request.HasPassengerInfo,
		OSAGOVerified:                 request.OSAGOVerified,
		DiagnosticCardVerified:        request.DiagnosticCardVerified,
		TechnicalStateVerified:        request.TechnicalStateVerified,
		LocalizationCompliant:         request.LocalizationCompliant,
		LegalUseBasisVerified:         request.LegalUseBasisVerified,
		IsActive:                      isActive,
	}, nil
}

func carPatchRecordFromRequest(request dto.TaxiParkCarPatchRequest) (CarPatchRecord, error) {
	ownerDetails := request.OwnerDetails
	if ownerDetails == nil {
		ownerDetails = request.OwnerOrLegalBasis
	}
	record := CarPatchRecord{
		PrimaryDriverID:               request.PrimaryDriverID,
		AttachedDriverIDs:             request.AttachedDriverIDs,
		Brand:                         trimStringPointer(request.Brand),
		Model:                         trimStringPointer(request.Model),
		Year:                          request.Year,
		PlateNumber:                   trimStringPointer(request.PlateNumber),
		VIN:                           trimStringPointer(request.VIN),
		STS:                           trimStringPointer(request.STS),
		PTS:                           trimStringPointer(request.PTS),
		Color:                         trimStringPointer(request.Color),
		CarClass:                      trimStringPointer(request.CarClass),
		VerificationStatus:            request.VerificationStatus,
		OwnerDetails:                  trimStringPointer(ownerDetails),
		TaxiPermitNumber:              trimStringPointer(request.TaxiPermitNumber),
		RegionalRegistryNumber:        trimStringPointer(request.RegionalRegistryNumber),
		PermitRegion:                  trimStringPointer(request.PermitRegion),
		TaxiPermitVerified:            request.TaxiPermitVerified,
		RegionalRegistryVerified:      request.RegionalRegistryVerified,
		RegionalRequirementsCompliant: request.RegionalRequirementsCompliant,
		HasTaxiColorScheme:            request.HasTaxiColorScheme,
		HasOrangeRoofLamp:             request.HasOrangeRoofLamp,
		HasPassengerInfo:              request.HasPassengerInfo,
		OSAGOVerified:                 request.OSAGOVerified,
		DiagnosticCardVerified:        request.DiagnosticCardVerified,
		TechnicalStateVerified:        request.TechnicalStateVerified,
		LocalizationCompliant:         request.LocalizationCompliant,
		LegalUseBasisVerified:         request.LegalUseBasisVerified,
		IsActive:                      request.IsActive,
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

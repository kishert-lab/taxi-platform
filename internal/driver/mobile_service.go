// Package driver contains driver mobile application use cases.
package driver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/internal/common"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geo"
)

var (
	ErrDriverNotFound       = errors.New("driver not found")
	ErrDriverNotAvailable   = errors.New("driver not available")
	ErrCurrentOrderNotFound = errors.New("current order not found")
)

type MobileRepository interface {
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (Profile, error)
	UpdateProfileByUserID(ctx context.Context, userID uuid.UUID, patch ProfilePatch) (Profile, error)
	SetStatusByUserID(ctx context.Context, userID uuid.UUID, status domain.DriverStatus) (Profile, error)
	ListCarsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Car, error)
	GetCurrentOrderByUserID(ctx context.Context, userID uuid.UUID) (CurrentOrder, error)
}

type PresenceStore interface {
	MarkOnline(ctx context.Context, cityID uuid.UUID, driverID uuid.UUID, ttl time.Duration) error
	MarkOffline(ctx context.Context, cityID uuid.UUID, driverID uuid.UUID) error
}

type LocationService interface {
	UpdateLocation(ctx context.Context, update geoservice.DriverLocationUpdate) error
	UpdateLocationBatch(ctx context.Context, updates []geoservice.DriverLocationUpdate) error
}

type DispatchController interface {
	AcceptOffer(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID) error
}

type RealtimeGateway interface {
	SendDriverLocationToTaxiPark(ctx context.Context, driverID uuid.UUID, payload any) error
}

type MobileService struct {
	repository         MobileRepository
	presenceStore      PresenceStore
	locationService    LocationService
	dispatchController DispatchController
	realtimeGateway    RealtimeGateway
	logger             *zap.Logger
	presenceTTL        time.Duration
}

type Profile struct {
	DriverID      uuid.UUID
	UserID        uuid.UUID
	CityID        uuid.UUID
	Phone         string
	FirstName     string
	LastName      string
	PhotoURL      string
	Status        domain.DriverStatus
	Rating        float64
	RatingsCount  int
	LicenseNumber string
	IsVerified    bool
}

type ProfilePatch struct {
	FirstName     *string
	LastName      *string
	LicenseNumber *string
	PhotoURL      *string
}

type CurrentOrder struct {
	OrderID               uuid.UUID
	PassengerID           uuid.UUID
	PassengerName         string
	PassengerPhone        string
	PassengerPhotoURL     string
	PassengerRating       float64
	PassengerRatingsCount int
	PickupAddress         string
	PickupLocation        domain.Coordinates
	DestinationAddress    string
	DestinationLocation   *domain.Coordinates
	Status                domain.OrderStatus
	Price                 *domain.Money
	Comment               string
	Version               int
	CreatedAt             time.Time
}

func NewMobileService(repository MobileRepository, presenceStore PresenceStore, locationService LocationService, logger *zap.Logger) *MobileService {
	return &MobileService{
		repository:      repository,
		presenceStore:   presenceStore,
		locationService: locationService,
		logger:          logger,
		presenceTTL:     45 * time.Second,
	}
}

func NewMobileServiceWithDispatch(repository MobileRepository, presenceStore PresenceStore, locationService LocationService, dispatchController DispatchController, logger *zap.Logger, realtimeGateways ...RealtimeGateway) *MobileService {
	service := NewMobileService(repository, presenceStore, locationService, logger)
	service.dispatchController = dispatchController
	if len(realtimeGateways) > 0 {
		service.realtimeGateway = realtimeGateways[0]
	}
	return service
}

func (service *MobileService) GetDriverProfile(ctx context.Context, userID uuid.UUID) (dto.DriverProfileResponse, error) {
	profile, err := service.repository.GetProfileByUserID(ctx, userID)
	if err != nil {
		return dto.DriverProfileResponse{}, err
	}
	return profileResponse(profile), nil
}

func (service *MobileService) ListDriverCars(ctx context.Context, userID uuid.UUID) (dto.TaxiParkCarsResponse, error) {
	cars, err := service.repository.ListCarsByUserID(ctx, userID)
	if err != nil {
		return dto.TaxiParkCarsResponse{}, err
	}
	return driverCarsResponse(cars), nil
}

func (service *MobileService) UpdateDriverProfile(ctx context.Context, userID uuid.UUID, request dto.DriverProfilePatchRequest) (dto.DriverProfileResponse, error) {
	patch := ProfilePatch{
		LicenseNumber: trimStringPointer(request.LicenseNumber),
		PhotoURL:      trimStringPointer(request.PhotoURL),
	}
	if request.Name != nil {
		firstName, lastName := splitName(*request.Name)
		patch.FirstName = &firstName
		patch.LastName = &lastName
	}

	profile, err := service.repository.UpdateProfileByUserID(ctx, userID, patch)
	if err != nil {
		return dto.DriverProfileResponse{}, err
	}
	return profileResponse(profile), nil
}

func (service *MobileService) UploadDriverProfilePhoto(context.Context, uuid.UUID, dto.ProfilePhotoUploadRequest) (dto.ProfilePhotoUploadResponse, error) {
	return dto.ProfilePhotoUploadResponse{}, common.ErrNotImplemented
}

func (service *MobileService) MarkDriverOnline(ctx context.Context, userID uuid.UUID) (dto.DriverProfileResponse, error) {
	profile, err := service.repository.SetStatusByUserID(ctx, userID, domain.DriverStatusOnline)
	if err != nil {
		return dto.DriverProfileResponse{}, err
	}
	if err := service.presenceStore.MarkOnline(ctx, profile.CityID, profile.DriverID, service.presenceTTL); err != nil {
		return dto.DriverProfileResponse{}, fmt.Errorf("mark driver online presence: %w", err)
	}
	service.logger.Info("driver marked online", zap.String("driver_id", profile.DriverID.String()), zap.String("user_id", userID.String()))
	return profileResponse(profile), nil
}

func (service *MobileService) MarkDriverOffline(ctx context.Context, userID uuid.UUID) (dto.DriverProfileResponse, error) {
	profile, err := service.repository.SetStatusByUserID(ctx, userID, domain.DriverStatusOffline)
	if err != nil {
		return dto.DriverProfileResponse{}, err
	}
	if err := service.presenceStore.MarkOffline(ctx, profile.CityID, profile.DriverID); err != nil {
		return dto.DriverProfileResponse{}, fmt.Errorf("mark driver offline presence: %w", err)
	}
	service.logger.Info("driver marked offline", zap.String("driver_id", profile.DriverID.String()), zap.String("user_id", userID.String()))
	return profileResponse(profile), nil
}

func (service *MobileService) UpdateDriverLocation(ctx context.Context, userID uuid.UUID, request dto.DriverLocationRequest) error {
	profile, err := service.repository.GetProfileByUserID(ctx, userID)
	if err != nil {
		return err
	}
	update := locationUpdate(profile, request)
	if err := service.locationService.UpdateLocation(ctx, update); err != nil {
		return err
	}
	return service.publishLocationUpdated(ctx, profile, update)
}

func (service *MobileService) UpdateDriverLocationBatch(ctx context.Context, userID uuid.UUID, request dto.DriverLocationBatchRequest) error {
	profile, err := service.repository.GetProfileByUserID(ctx, userID)
	if err != nil {
		return err
	}
	updates := make([]geoservice.DriverLocationUpdate, 0, len(request.Locations))
	for _, location := range request.Locations {
		updates = append(updates, locationUpdate(profile, location))
	}
	if err := service.locationService.UpdateLocationBatch(ctx, updates); err != nil {
		return err
	}
	if len(updates) == 0 {
		return nil
	}
	return service.publishLocationUpdated(ctx, profile, updates[len(updates)-1])
}

func (service *MobileService) GetCurrentDriverOrder(ctx context.Context, userID uuid.UUID) (dto.DriverOrderResponse, error) {
	order, err := service.repository.GetCurrentOrderByUserID(ctx, userID)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return currentOrderResponse(order), nil
}

func (service *MobileService) ListDriverOrderHistory(context.Context, uuid.UUID) (dto.DriverOrderHistoryResponse, error) {
	return dto.DriverOrderHistoryResponse{}, common.ErrNotImplemented
}

func (service *MobileService) AcceptDriverOrder(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error) {
	if service.dispatchController == nil {
		return dto.DriverOrderResponse{}, common.ErrNotImplemented
	}
	profile, err := service.repository.GetProfileByUserID(ctx, userID)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.dispatchController.AcceptOffer(ctx, orderID, profile.DriverID); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return service.GetCurrentDriverOrder(ctx, userID)
}

func (service *MobileService) RejectDriverOrder(context.Context, uuid.UUID, uuid.UUID, dto.RejectOrderRequest) error {
	return common.ErrNotImplemented
}

func (service *MobileService) MarkDriverArrived(context.Context, uuid.UUID, uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (service *MobileService) StartDriverTrip(context.Context, uuid.UUID, uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (service *MobileService) CompleteDriverTrip(context.Context, uuid.UUID, uuid.UUID, dto.CompleteOrderRequest) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func (service *MobileService) RatePassenger(context.Context, uuid.UUID, uuid.UUID, dto.RateOrderRequest) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
}

func locationUpdate(profile Profile, request dto.DriverLocationRequest) geoservice.DriverLocationUpdate {
	return geoservice.DriverLocationUpdate{
		DriverID: profile.DriverID,
		CityID:   profile.CityID,
		Location: domain.Coordinates{
			Latitude:  request.Location.Latitude,
			Longitude: request.Location.Longitude,
		},
		Heading:        request.Heading,
		SpeedMPS:       request.SpeedMPS,
		AccuracyMeters: request.AccuracyMeters,
		RecordedAt:     time.Now().UTC(),
	}
}

func (service *MobileService) publishLocationUpdated(ctx context.Context, profile Profile, update geoservice.DriverLocationUpdate) error {
	if service.realtimeGateway == nil {
		return nil
	}
	payload := map[string]any{
		"driver_id": profile.DriverID,
		"user_id":   profile.UserID,
		"name":      strings.TrimSpace(strings.TrimSpace(profile.FirstName) + " " + strings.TrimSpace(profile.LastName)),
		"phone":     profile.Phone,
		"status":    profile.Status,
		"location": map[string]float64{
			"latitude":  update.Location.Latitude,
			"longitude": update.Location.Longitude,
		},
		"heading":         update.Heading,
		"speed_mps":       update.SpeedMPS,
		"accuracy_meters": update.AccuracyMeters,
		"recorded_at":     update.RecordedAt,
	}
	if err := service.realtimeGateway.SendDriverLocationToTaxiPark(ctx, profile.DriverID, payload); err != nil {
		return fmt.Errorf("publish taxi park driver location websocket event: %w", err)
	}
	return nil
}

func profileResponse(profile Profile) dto.DriverProfileResponse {
	name := strings.TrimSpace(strings.TrimSpace(profile.FirstName) + " " + strings.TrimSpace(profile.LastName))
	return dto.DriverProfileResponse{
		ID:            profile.DriverID,
		UserID:        profile.UserID,
		Phone:         profile.Phone,
		Name:          name,
		PhotoURL:      profile.PhotoURL,
		Status:        profile.Status,
		Rating:        profile.Rating,
		RatingsCount:  profile.RatingsCount,
		LicenseNumber: profile.LicenseNumber,
		IsVerified:    profile.IsVerified,
	}
}

func driverCarsResponse(cars []domain.Car) dto.TaxiParkCarsResponse {
	responseBody := dto.TaxiParkCarsResponse{Cars: make([]dto.TaxiParkCarResponse, 0, len(cars))}
	for _, car := range cars {
		responseBody.Cars = append(responseBody.Cars, dto.TaxiParkCarResponse{
			ID:                            car.ID,
			TaxiParkID:                    car.TaxiParkID,
			PrimaryDriverID:               car.PrimaryDriverID,
			AttachedDriverIDs:             car.AttachedDriverIDs,
			Brand:                         car.Brand,
			Model:                         car.Model,
			Year:                          car.Year,
			PlateNumber:                   car.PlateNumber,
			VIN:                           car.VIN,
			STS:                           car.STS,
			PTS:                           car.PTS,
			Color:                         car.Color,
			CarClass:                      car.CarClass,
			VerificationStatus:            car.VerificationStatus,
			OwnerDetails:                  car.OwnerDetails,
			OSAGOExpiresAt:                car.OSAGOExpiresAt,
			DiagnosticCardExpiresAt:       car.DiagnosticCardExpiresAt,
			TaxiPermitNumber:              car.TaxiPermitNumber,
			RegionalRegistryNumber:        car.RegionalRegistryNumber,
			PermitRegion:                  car.PermitRegion,
			PermitIssuedAt:                car.PermitIssuedAt,
			PermitExpiresAt:               car.PermitExpiresAt,
			TaxiPermitVerified:            car.TaxiPermitVerified,
			RegionalRegistryVerified:      car.RegionalRegistryVerified,
			RegionalRequirementsCompliant: car.RegionalRequirementsCompliant,
			HasTaxiColorScheme:            car.HasTaxiColorScheme,
			HasOrangeRoofLamp:             car.HasOrangeRoofLamp,
			HasPassengerInfo:              car.HasPassengerInfo,
			OSAGOVerified:                 car.OSAGOVerified,
			DiagnosticCardVerified:        car.DiagnosticCardVerified,
			TechnicalStateVerified:        car.TechnicalStateVerified,
			LocalizationCompliant:         car.LocalizationCompliant,
			LegalUseBasisVerified:         car.LegalUseBasisVerified,
			VerificationCheckedAt:         car.VerificationCheckedAt,
			VerificationCheckedBy:         car.VerificationCheckedBy,
			IsActive:                      car.IsActive,
			CreatedAt:                     car.CreatedAt,
			UpdatedAt:                     car.UpdatedAt,
		})
	}
	return responseBody
}

func currentOrderResponse(order CurrentOrder) dto.DriverOrderResponse {
	destinationLocation := dto.CoordinatesResponse{}
	if order.DestinationLocation != nil {
		destinationLocation = dto.CoordinatesResponse{
			Latitude:  order.DestinationLocation.Latitude,
			Longitude: order.DestinationLocation.Longitude,
		}
	}

	var price *dto.MoneyResponse
	if order.Price != nil {
		price = &dto.MoneyResponse{
			Amount:   order.Price.Amount,
			Currency: order.Price.Currency,
		}
	}

	return dto.DriverOrderResponse{
		OrderID: order.OrderID,
		Passenger: dto.PassengerBriefDTO{
			ID:           order.PassengerID,
			Name:         order.PassengerName,
			Phone:        order.PassengerPhone,
			PhotoURL:     order.PassengerPhotoURL,
			Rating:       order.PassengerRating,
			RatingsCount: order.PassengerRatingsCount,
		},
		PickupPoint: dto.PointDTO{
			Address: order.PickupAddress,
			Location: dto.CoordinatesResponse{
				Latitude:  order.PickupLocation.Latitude,
				Longitude: order.PickupLocation.Longitude,
			},
		},
		DestinationPoint: dto.PointDTO{
			Address:  order.DestinationAddress,
			Location: destinationLocation,
		},
		Status:         order.Status,
		Price:          price,
		Comment:        order.Comment,
		Timeline:       []dto.OrderTimelineItem{{Status: order.Status, OccurredAt: order.CreatedAt}},
		AllowedActions: driverAllowedActions(order.Status),
		Version:        order.Version,
	}
}

func driverAllowedActions(status domain.OrderStatus) []string {
	switch status {
	case domain.OrderStatusDriverAssigned, domain.OrderStatusDriverArriving:
		return []string{"arrived", "call_passenger"}
	case domain.OrderStatusDriverWaiting:
		return []string{"start", "call_passenger"}
	case domain.OrderStatusInProgress:
		return []string{"complete"}
	default:
		return []string{}
	}
}

func splitName(value string) (string, string) {
	parts := strings.Fields(value)
	if len(parts) == 0 {
		return "", ""
	}
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], strings.Join(parts[1:], " ")
}

func trimStringPointer(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

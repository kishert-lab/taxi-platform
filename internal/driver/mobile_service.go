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

type MobileService struct {
	repository      MobileRepository
	presenceStore   PresenceStore
	locationService LocationService
	logger          *zap.Logger
	presenceTTL     time.Duration
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

func (service *MobileService) GetDriverProfile(ctx context.Context, userID uuid.UUID) (dto.DriverProfileResponse, error) {
	profile, err := service.repository.GetProfileByUserID(ctx, userID)
	if err != nil {
		return dto.DriverProfileResponse{}, err
	}
	return profileResponse(profile), nil
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
	return service.locationService.UpdateLocation(ctx, locationUpdate(profile, request))
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
	return service.locationService.UpdateLocationBatch(ctx, updates)
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

func (service *MobileService) AcceptDriverOrder(context.Context, uuid.UUID, uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, common.ErrNotImplemented
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

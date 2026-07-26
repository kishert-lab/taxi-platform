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
	dispatchapp "github.com/kishert-lab/taxi-platform/internal/dispatch"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geo"
	wsmsg "github.com/kishert-lab/taxi-platform/internal/ws"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

var (
	ErrDriverNotFound       = errors.New("driver not found")
	ErrDriverNotAvailable   = errors.New("driver not available")
	ErrCurrentOrderNotFound = errors.New("current order not found")
	ErrOrderAccessDenied    = errors.New("order access denied")
	ErrOrderRouteForbidden  = errors.New("order route upload forbidden")
)

type MobileRepository interface {
	GetProfileByUserID(ctx context.Context, userID uuid.UUID) (Profile, error)
	UpdateProfileByUserID(ctx context.Context, userID uuid.UUID, patch ProfilePatch) (Profile, error)
	SetStatusByUserID(ctx context.Context, userID uuid.UUID, status domain.DriverStatus) (Profile, error)
	ListCarsByUserID(ctx context.Context, userID uuid.UUID) ([]domain.Car, error)
	GetCurrentOrderByUserID(ctx context.Context, userID uuid.UUID) (CurrentOrder, error)
	GetOrderByUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (CurrentOrder, error)
	ListOrderHistoryByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]CurrentOrder, error)
	ListRoutePointsByUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) ([]RoutePoint, error)
	GetOrderRouteUploadAccess(ctx context.Context, orderID uuid.UUID) (OrderRouteUploadAccess, error)
	TransitionOrderByUserID(ctx context.Context, userID uuid.UUID, orderID uuid.UUID, toStatus domain.OrderStatus, reason string, finalPriceCents *int64) (CurrentOrder, error)
	AppendRoutePointByUserID(ctx context.Context, userID uuid.UUID, update geoservice.DriverLocationUpdate) error
	AppendOrderRoutePoints(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID, points []OrderRouteAppendPoint) (AppendOrderRoutePointsResult, error)
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
	RejectOffer(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID, reason string) error
	ListDriverOffers(ctx context.Context, driverID uuid.UUID) ([]dispatchapp.DriverOrderOffer, error)
}

type RealtimeGateway interface {
	SendDriverPresenceToTaxiPark(ctx context.Context, driverID uuid.UUID, payload any) error
	SendDriverLocationToTaxiPark(ctx context.Context, driverID uuid.UUID, payload any) error
	SendToDriver(ctx context.Context, driverID uuid.UUID, eventName string, payload any) error
	SendToPassenger(ctx context.Context, passengerID uuid.UUID, eventName string, payload any) error
	SendToTaxiParkByOrder(ctx context.Context, orderID uuid.UUID, eventName string, payload any) error
}

type FinanceProcessor interface {
	SettleCompletedOrder(ctx context.Context, orderID uuid.UUID) (domain.OrderSettlement, error)
}

type PassengerNotifier interface {
	NotifyPassenger(ctx context.Context, passengerID uuid.UUID, notification PassengerNotification) error
}

type PassengerNotification struct {
	Title string
	Body  string
	Data  map[string]string
}

type MobileService struct {
	repository         MobileRepository
	presenceStore      PresenceStore
	locationService    LocationService
	dispatchController DispatchController
	realtimeGateway    RealtimeGateway
	financeProcessor   FinanceProcessor
	passengerNotifier  PassengerNotifier
	logger             *zap.Logger
	presenceTTL        time.Duration
}

type Profile struct {
	DriverID                      uuid.UUID
	UserID                        uuid.UUID
	CityID                        uuid.UUID
	Phone                         string
	FirstName                     string
	LastName                      string
	PhotoURL                      string
	Status                        domain.DriverStatus
	Rating                        float64
	RatingsCount                  int
	LicenseNumber                 string
	IsVerified                    bool
	VerificationStatus            domain.VerificationLifecycleStatus
	TaxiParkID                    *uuid.UUID
	TaxiParkIsActive              *bool
	HasNoTaxiWorkRestrictions     bool
	FederalLaw580Compliant        bool
	RegionalRequirementsCompliant bool
	MedicalCheckPassed            bool
	PretripControlRequired        bool
	PretripControlPassed          bool
	NoTransportBan                bool
	Car                           *ProfileCar
}

type ProfileCar struct {
	ID                 uuid.UUID
	Brand              string
	Model              string
	Year               int
	PlateNumber        string
	Color              string
	CarClass           string
	VerificationStatus domain.VerificationLifecycleStatus
	IsActive           bool
	OSAGOExpiresAt     *time.Time
	PermitExpiresAt    *time.Time
}

type ProfilePatch struct {
	FirstName     *string
	LastName      *string
	LicenseNumber *string
	PhotoURL      *string
}

type CurrentOrder struct {
	OrderID               uuid.UUID
	DriverID              uuid.UUID
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
	AssignedTariffID      *uuid.UUID
	AssignedTaxiParkID    *uuid.UUID
	PricingMode           domain.PricingMode
	Comment               string
	Version               int
	CreatedAt             time.Time
}

type RoutePoint struct {
	ID             uuid.UUID
	Location       domain.Coordinates
	Heading        *float64
	SpeedMPS       *float64
	AccuracyMeters *float64
	RecordedAt     time.Time
}

type OrderRouteUploadAccess struct {
	OrderID      uuid.UUID
	DriverID     uuid.UUID
	DriverUserID uuid.UUID
	Status       domain.OrderStatus
}

type OrderRouteAppendPoint struct {
	Location       domain.Coordinates
	Heading        *int16
	SpeedMPS       *float64
	AccuracyMeters *float64
	RecordedAt     time.Time
}

type AppendOrderRoutePointsResult struct {
	OrderID        uuid.UUID
	AcceptedPoints int
	IgnoredPoints  int
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

func (service *MobileService) WithFinanceProcessor(financeProcessor FinanceProcessor) *MobileService {
	service.financeProcessor = financeProcessor
	return service
}

func (service *MobileService) WithPassengerNotifier(passengerNotifier PassengerNotifier) *MobileService {
	service.passengerNotifier = passengerNotifier
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
	profile.Status = domain.DriverStatusOnline
	if err := service.presenceStore.MarkOnline(ctx, profile.CityID, profile.DriverID, service.presenceTTL); err != nil {
		return dto.DriverProfileResponse{}, fmt.Errorf("mark driver online presence: %w", err)
	}
	if err := service.publishPresenceChanged(ctx, profile); err != nil {
		return dto.DriverProfileResponse{}, err
	}
	service.logger.Info("driver marked online", zap.String("driver_id", profile.DriverID.String()), zap.String("user_id", userID.String()))
	return profileResponse(profile), nil
}

func (service *MobileService) MarkDriverOffline(ctx context.Context, userID uuid.UUID) (dto.DriverProfileResponse, error) {
	profile, err := service.repository.SetStatusByUserID(ctx, userID, domain.DriverStatusOffline)
	if err != nil {
		return dto.DriverProfileResponse{}, err
	}
	profile.Status = domain.DriverStatusOffline
	if err := service.presenceStore.MarkOffline(ctx, profile.CityID, profile.DriverID); err != nil {
		return dto.DriverProfileResponse{}, fmt.Errorf("mark driver offline presence: %w", err)
	}
	if err := service.publishPresenceChanged(ctx, profile); err != nil {
		return dto.DriverProfileResponse{}, err
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
	if err := service.repository.AppendRoutePointByUserID(ctx, userID, update); err != nil {
		return err
	}
	if err := service.publishLocationUpdated(ctx, profile, update); err != nil {
		return err
	}
	return service.publishPassengerDriverLocation(ctx, userID, profile, update)
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
	for _, update := range updates {
		if err := service.repository.AppendRoutePointByUserID(ctx, userID, update); err != nil {
			return err
		}
	}
	lastUpdate := updates[len(updates)-1]
	if err := service.publishLocationUpdated(ctx, profile, lastUpdate); err != nil {
		return err
	}
	return service.publishPassengerDriverLocation(ctx, userID, profile, lastUpdate)
}

func (service *MobileService) GetCurrentDriverOrder(ctx context.Context, userID uuid.UUID) (dto.DriverOrderResponse, error) {
	order, err := service.repository.GetCurrentOrderByUserID(ctx, userID)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return currentOrderResponse(order), nil
}

func (service *MobileService) GetDriverOrder(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error) {
	order, err := service.repository.GetOrderByUserID(ctx, userID, orderID)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return currentOrderResponse(order), nil
}

func (service *MobileService) ListDriverOrderHistory(ctx context.Context, userID uuid.UUID) (dto.DriverOrderHistoryResponse, error) {
	orders, err := service.repository.ListOrderHistoryByUserID(ctx, userID, 50)
	if err != nil {
		return dto.DriverOrderHistoryResponse{}, err
	}
	response := dto.DriverOrderHistoryResponse{Orders: make([]dto.DriverOrderResponse, 0, len(orders))}
	for _, order := range orders {
		response.Orders = append(response.Orders, currentOrderResponse(order))
	}
	return response, nil
}

func (service *MobileService) ListDriverOrderOffers(ctx context.Context, userID uuid.UUID) (dto.DriverOrderOffersResponse, error) {
	if service.dispatchController == nil {
		return dto.DriverOrderOffersResponse{}, common.ErrNotImplemented
	}
	profile, err := service.repository.GetProfileByUserID(ctx, userID)
	if err != nil {
		return dto.DriverOrderOffersResponse{}, err
	}
	offers, err := service.dispatchController.ListDriverOffers(ctx, profile.DriverID)
	if err != nil {
		return dto.DriverOrderOffersResponse{}, err
	}
	response := dto.DriverOrderOffersResponse{Offers: make([]dto.DriverOrderOfferResponse, 0, len(offers))}
	for _, offer := range offers {
		response.Offers = append(response.Offers, driverOrderOfferResponse(offer))
	}
	return response, nil
}

func (service *MobileService) GetDriverOrderRoute(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (dto.OrderRouteResponse, error) {
	points, err := service.repository.ListRoutePointsByUserID(ctx, userID, orderID)
	if err != nil {
		return dto.OrderRouteResponse{}, err
	}
	response := dto.OrderRouteResponse{
		OrderID: orderID,
		Points:  make([]dto.OrderRoutePointResponse, 0, len(points)),
	}
	for _, point := range points {
		response.Points = append(response.Points, dto.OrderRoutePointResponse{
			ID: point.ID,
			Location: dto.CoordinatesResponse{
				Latitude:  point.Location.Latitude,
				Longitude: point.Location.Longitude,
			},
			Heading:        point.Heading,
			SpeedMPS:       point.SpeedMPS,
			AccuracyMeters: point.AccuracyMeters,
			RecordedAt:     point.RecordedAt,
		})
	}
	return response, nil
}

func (service *MobileService) AppendOrderRoutePoints(ctx context.Context, userID uuid.UUID, orderID uuid.UUID, request dto.DriverOrderRouteBatchRequest) (dto.DriverOrderRouteBatchResponse, error) {
	access, err := service.repository.GetOrderRouteUploadAccess(ctx, orderID)
	if err != nil {
		return dto.DriverOrderRouteBatchResponse{}, err
	}
	if access.DriverUserID != userID {
		return dto.DriverOrderRouteBatchResponse{}, ErrOrderAccessDenied
	}
	if !canAppendOrderRoutePoints(access.Status) {
		return dto.DriverOrderRouteBatchResponse{}, ErrOrderRouteForbidden
	}

	points := make([]OrderRouteAppendPoint, 0, len(request.Points))
	for _, point := range request.Points {
		points = append(points, OrderRouteAppendPoint{
			Location: domain.Coordinates{
				Latitude:  point.Location.Latitude,
				Longitude: point.Location.Longitude,
			},
			Heading:        point.Heading,
			SpeedMPS:       point.SpeedMPS,
			AccuracyMeters: point.AccuracyMeters,
			RecordedAt:     point.RecordedAt.UTC(),
		})
	}

	result, err := service.repository.AppendOrderRoutePoints(ctx, orderID, access.DriverID, points)
	if err != nil {
		return dto.DriverOrderRouteBatchResponse{}, fmt.Errorf("append order route points: %w", err)
	}

	service.logger.Info(
		"driver uploaded order route points",
		zap.String("request_id", requestIDFromContext(ctx)),
		zap.String("driver_id", access.DriverID.String()),
		zap.String("order_id", orderID.String()),
		zap.Int("accepted_points", result.AcceptedPoints),
		zap.Int("ignored_points", result.IgnoredPoints),
	)

	return dto.DriverOrderRouteBatchResponse{
		OrderID:        result.OrderID,
		AcceptedPoints: result.AcceptedPoints,
		IgnoredPoints:  result.IgnoredPoints,
	}, nil
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

func (service *MobileService) RejectDriverOrder(ctx context.Context, userID uuid.UUID, orderID uuid.UUID, request dto.RejectOrderRequest) error {
	if service.dispatchController == nil {
		return common.ErrNotImplemented
	}
	profile, err := service.repository.GetProfileByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return service.dispatchController.RejectOffer(ctx, orderID, profile.DriverID, request.Reason)
}

func (service *MobileService) MarkDriverArriving(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error) {
	order, err := service.repository.TransitionOrderByUserID(ctx, userID, orderID, domain.OrderStatusDriverArriving, "", nil)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.publishOrderState(ctx, order, "order.driver_arriving"); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.notifyPassenger(ctx, order.PassengerID, PassengerNotification{
		Title: "Водитель в пути",
		Body:  "Водитель едет к месту посадки.",
		Data: map[string]string{
			"event":     "order.driver_arriving",
			"order_id":  order.OrderID.String(),
			"driver_id": order.DriverID.String(),
		},
	}); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return currentOrderResponse(order), nil
}

func (service *MobileService) MarkDriverArrived(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error) {
	order, err := service.repository.TransitionOrderByUserID(ctx, userID, orderID, domain.OrderStatusDriverWaiting, "", nil)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.publishOrderState(ctx, order, "order.driver_waiting"); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.notifyPassenger(ctx, order.PassengerID, PassengerNotification{
		Title: "Водитель ожидает",
		Body:  "Водитель прибыл и ожидает вас.",
		Data: map[string]string{
			"event":     "order.driver_waiting",
			"order_id":  order.OrderID.String(),
			"driver_id": order.DriverID.String(),
		},
	}); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return currentOrderResponse(order), nil
}

func (service *MobileService) CancelDriverOrder(ctx context.Context, userID uuid.UUID, orderID uuid.UUID, reason string) (dto.DriverOrderResponse, error) {
	order, err := service.repository.TransitionOrderByUserID(ctx, userID, orderID, domain.OrderStatusCancelled, reason, nil)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.markProfileOnline(ctx, userID); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.publishDriverCancelledOrderState(ctx, order, reason); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.notifyPassenger(ctx, order.PassengerID, PassengerNotification{
		Title: "Заказ отменен водителем",
		Body:  "Водитель отменил поездку.",
		Data: map[string]string{
			"event":               "order.cancelled",
			"order_id":            order.OrderID.String(),
			"driver_id":           order.DriverID.String(),
			"cancelled_by":        "driver",
			"cancellation_reason": strings.TrimSpace(reason),
		},
	}); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return currentOrderResponse(order), nil
}

func (service *MobileService) StartDriverTrip(ctx context.Context, userID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error) {
	order, err := service.repository.TransitionOrderByUserID(ctx, userID, orderID, domain.OrderStatusInProgress, "", nil)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.publishOrderState(ctx, order, "order.trip_started"); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return currentOrderResponse(order), nil
}

func (service *MobileService) CompleteDriverTrip(ctx context.Context, userID uuid.UUID, orderID uuid.UUID, request dto.CompleteOrderRequest) (dto.DriverOrderResponse, error) {
	finalPriceCents := request.FinalPrice
	order, err := service.repository.TransitionOrderByUserID(ctx, userID, orderID, domain.OrderStatusCompleted, "", &finalPriceCents)
	if err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if service.financeProcessor != nil {
		if _, err := service.financeProcessor.SettleCompletedOrder(ctx, orderID); err != nil {
			return dto.DriverOrderResponse{}, fmt.Errorf("settle completed order: %w", err)
		}
	}
	if err := service.markProfileOnline(ctx, userID); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	if err := service.publishOrderState(ctx, order, "order.completed"); err != nil {
		return dto.DriverOrderResponse{}, err
	}
	return currentOrderResponse(order), nil
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

func (service *MobileService) publishPresenceChanged(ctx context.Context, profile Profile) error {
	if service.realtimeGateway == nil {
		return nil
	}
	payload := map[string]any{
		"driver_id":    profile.DriverID,
		"user_id":      profile.UserID,
		"city_id":      profile.CityID,
		"taxi_park_id": profile.TaxiParkID,
		"name":         strings.TrimSpace(strings.TrimSpace(profile.FirstName) + " " + strings.TrimSpace(profile.LastName)),
		"phone":        profile.Phone,
		"status":       profile.Status,
		"changed_at":   time.Now().UTC(),
	}
	if profile.Car != nil {
		payload["car"] = map[string]any{
			"id":           profile.Car.ID,
			"brand":        profile.Car.Brand,
			"model":        profile.Car.Model,
			"plate_number": profile.Car.PlateNumber,
			"car_class":    profile.Car.CarClass,
		}
	}
	if err := service.realtimeGateway.SendDriverPresenceToTaxiPark(ctx, profile.DriverID, payload); err != nil {
		return fmt.Errorf("publish taxi park driver presence websocket event: %w", err)
	}
	return nil
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

func (service *MobileService) publishPassengerDriverLocation(ctx context.Context, userID uuid.UUID, profile Profile, update geoservice.DriverLocationUpdate) error {
	if service.realtimeGateway == nil {
		return nil
	}
	order, err := service.repository.GetCurrentOrderByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrCurrentOrderNotFound) {
			return nil
		}
		return err
	}
	payload := wsmsg.PassengerDriverLocationPayload{
		OrderID:  order.OrderID,
		DriverID: profile.DriverID,
		Status:   order.Status,
		Location: dto.CoordinatesResponse{
			Latitude:  update.Location.Latitude,
			Longitude: update.Location.Longitude,
		},
		Heading:        update.Heading,
		SpeedMPS:       update.SpeedMPS,
		AccuracyMeters: update.AccuracyMeters,
		RecordedAt:     update.RecordedAt,
	}
	if err := service.realtimeGateway.SendToPassenger(ctx, order.PassengerID, "driver.location_updated", payload); err != nil {
		return fmt.Errorf("publish passenger driver location websocket event: %w", err)
	}
	return nil
}

func (service *MobileService) markProfileOnline(ctx context.Context, userID uuid.UUID) error {
	profile, err := service.repository.GetProfileByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return service.presenceStore.MarkOnline(ctx, profile.CityID, profile.DriverID, service.presenceTTL)
}

func (service *MobileService) publishOrderState(ctx context.Context, order CurrentOrder, eventName string) error {
	if service.realtimeGateway == nil {
		return nil
	}
	payload := wsmsg.PassengerOrderStatePayload{
		OrderID:  order.OrderID,
		Status:   order.Status,
		Version:  order.Version,
		DriverID: &order.DriverID,
	}
	if order.DriverID == uuid.Nil {
		payload.DriverID = nil
	}
	if err := service.realtimeGateway.SendToPassenger(ctx, order.PassengerID, eventName, payload); err != nil {
		return fmt.Errorf("publish order state to passenger: %w", err)
	}
	if err := service.realtimeGateway.SendToTaxiParkByOrder(ctx, order.OrderID, eventName, payload); err != nil {
		return fmt.Errorf("publish order state to taxi park: %w", err)
	}
	return service.realtimeGateway.SendToDriver(ctx, order.DriverID, eventName, payload)
}

func (service *MobileService) publishDriverCancelledOrderState(ctx context.Context, order CurrentOrder, reason string) error {
	if service.realtimeGateway == nil {
		return nil
	}

	payload := wsmsg.PassengerOrderStatePayload{
		OrderID:            order.OrderID,
		Status:             order.Status,
		Version:            order.Version,
		CancelledBy:        "driver",
		CancellationReason: strings.TrimSpace(reason),
	}
	if order.DriverID != uuid.Nil {
		payload.DriverID = &order.DriverID
	}

	if err := service.realtimeGateway.SendToPassenger(ctx, order.PassengerID, "order.cancelled", payload); err != nil {
		return fmt.Errorf("publish driver cancelled order state to passenger: %w", err)
	}

	genericPayload := wsmsg.PassengerOrderStatePayload{
		OrderID:  order.OrderID,
		Status:   order.Status,
		Version:  order.Version,
		DriverID: payload.DriverID,
	}
	if err := service.realtimeGateway.SendToTaxiParkByOrder(ctx, order.OrderID, "order.cancelled", genericPayload); err != nil {
		return fmt.Errorf("publish driver cancelled order state to taxi park: %w", err)
	}
	return service.realtimeGateway.SendToDriver(ctx, order.DriverID, "order.cancelled", genericPayload)
}

func profileResponse(profile Profile) dto.DriverProfileResponse {
	name := strings.TrimSpace(strings.TrimSpace(profile.FirstName) + " " + strings.TrimSpace(profile.LastName))
	response := dto.DriverProfileResponse{
		ID:                            profile.DriverID,
		UserID:                        profile.UserID,
		Phone:                         profile.Phone,
		Name:                          name,
		PhotoURL:                      profile.PhotoURL,
		Status:                        profile.Status,
		Rating:                        profile.Rating,
		RatingsCount:                  profile.RatingsCount,
		LicenseNumber:                 profile.LicenseNumber,
		IsVerified:                    profile.IsVerified,
		VerificationStatus:            profile.VerificationStatus,
		TaxiParkID:                    profile.TaxiParkID,
		TaxiParkIsActive:              profile.TaxiParkIsActive,
		HasNoTaxiWorkRestrictions:     profile.HasNoTaxiWorkRestrictions,
		FederalLaw580Compliant:        profile.FederalLaw580Compliant,
		RegionalRequirementsCompliant: profile.RegionalRequirementsCompliant,
		MedicalCheckPassed:            profile.MedicalCheckPassed,
		PretripControlRequired:        profile.PretripControlRequired,
		PretripControlPassed:          profile.PretripControlPassed,
		NoTransportBan:                profile.NoTransportBan,
	}
	if profile.Car != nil {
		response.Car = &dto.DriverProfileCarResponse{
			ID:                 profile.Car.ID,
			Brand:              profile.Car.Brand,
			Model:              profile.Car.Model,
			Year:               profile.Car.Year,
			PlateNumber:        profile.Car.PlateNumber,
			Color:              profile.Car.Color,
			CarClass:           profile.Car.CarClass,
			VerificationStatus: profile.Car.VerificationStatus,
			IsActive:           profile.Car.IsActive,
			OSAGOExpiresAt:     profile.Car.OSAGOExpiresAt,
			PermitExpiresAt:    profile.Car.PermitExpiresAt,
		}
	}
	return response
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

func driverOrderOfferResponse(offer dispatchapp.DriverOrderOffer) dto.DriverOrderOfferResponse {
	destinationLocation := dto.CoordinatesResponse{}
	if offer.Order.DestinationLocation != nil {
		destinationLocation = dto.CoordinatesResponse{
			Latitude:  offer.Order.DestinationLocation.Latitude,
			Longitude: offer.Order.DestinationLocation.Longitude,
		}
	}

	var estimatedPrice *dto.MoneyResponse
	if offer.Order.EstimatedPrice != nil {
		estimatedPrice = &dto.MoneyResponse{
			Amount:   offer.Order.EstimatedPrice.Amount,
			Currency: offer.Order.EstimatedPrice.Currency,
		}
	}

	return dto.DriverOrderOfferResponse{
		OrderID: offer.Order.ID,
		PickupPoint: dto.PointDTO{
			Address: offer.Order.PickupAddress,
			Location: dto.CoordinatesResponse{
				Latitude:  offer.Order.PickupLocation.Latitude,
				Longitude: offer.Order.PickupLocation.Longitude,
			},
		},
		DestinationPoint: dto.PointDTO{
			Address:  offer.Order.DestinationAddress,
			Location: destinationLocation,
		},
		Status:         offer.Order.Status,
		EstimatedPrice: estimatedPrice,
		Attempt:        offer.Offer.Attempt,
		RadiusMeters:   offer.Offer.RadiusMeters,
		DistanceMeters: offer.Offer.DistanceMeters,
		ExpiresAt:      offer.Offer.ExpiresAt,
		AllowedActions: []string{"accept", "reject"},
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
	case domain.OrderStatusDriverAssigned:
		return []string{"arriving", "cancel", "call_passenger"}
	case domain.OrderStatusDriverArriving:
		return []string{"arrived", "cancel", "call_passenger"}
	case domain.OrderStatusDriverWaiting:
		return []string{"start", "cancel", "call_passenger"}
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

func canAppendOrderRoutePoints(status domain.OrderStatus) bool {
	switch status {
	case domain.OrderStatusDriverAssigned,
		domain.OrderStatusDriverArriving,
		domain.OrderStatusDriverWaiting,
		domain.OrderStatusInProgress,
		domain.OrderStatusCompleted:
		return true
	default:
		return false
	}
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(response.RequestIDValueContextKey).(string)
	return requestID
}

func (service *MobileService) notifyPassenger(ctx context.Context, passengerID uuid.UUID, notification PassengerNotification) error {
	if service.passengerNotifier == nil || passengerID == uuid.Nil {
		return nil
	}
	if err := service.passengerNotifier.NotifyPassenger(ctx, passengerID, notification); err != nil {
		return fmt.Errorf("notify passenger: %w", err)
	}
	return nil
}

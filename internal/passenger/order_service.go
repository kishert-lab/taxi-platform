package passenger

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kishert-lab/taxi-platform/internal/common"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/order"
)

var (
	ErrPassengerOrderNotFound     = errors.New("passenger order not found")
	ErrPassengerActiveOrderExists = errors.New("passenger already has active order")
	ErrPassengerCarClassRequired  = errors.New("car class is required")
	ErrPassengerCarClassNotFound  = errors.New("car class not found")
	ErrPassengerInactive          = errors.New("passenger is inactive")
)

type OrderService struct {
	passengerRepository Repository
	orderRepository     OrderRepository
	dispatchQueue       DispatchQueue
	cityResolver        CityResolver
}

func NewOrderService(
	passengerRepository Repository,
	orderRepository OrderRepository,
	dispatchQueue DispatchQueue,
	cityResolver CityResolver,
) *OrderService {
	return &OrderService{
		passengerRepository: passengerRepository,
		orderRepository:     orderRepository,
		dispatchQueue:       dispatchQueue,
		cityResolver:        cityResolver,
	}
}

func (service *OrderService) ListPassengerCarClasses(ctx context.Context, passengerID uuid.UUID) (dto.PassengerCarClassesResponse, error) {
	passengerRecord, err := service.passengerRepository.GetByID(ctx, passengerID)
	if err != nil {
		return dto.PassengerCarClassesResponse{}, fmt.Errorf("get passenger before list car classes: %w", err)
	}
	if !passengerRecord.IsActive {
		return dto.PassengerCarClassesResponse{}, ErrPassengerInactive
	}

	carClasses, err := service.orderRepository.ListActiveCarClasses(ctx)
	if err != nil {
		return dto.PassengerCarClassesResponse{}, fmt.Errorf("list active car classes: %w", err)
	}

	responseItems := make([]dto.PassengerCarClassResponse, 0, len(carClasses))
	for _, carClass := range carClasses {
		responseItems = append(responseItems, passengerCarClassDTO(carClass))
	}

	return dto.PassengerCarClassesResponse{Items: responseItems}, nil
}

func (service *OrderService) EstimatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.OrderEstimateRequest) (dto.OrderEstimateResponse, error) {
	if _, err := service.ensureActivePassenger(ctx, passengerID); err != nil {
		return dto.OrderEstimateResponse{}, err
	}

	carClassID, err := resolveRequestedCarClassID(request.CarClassID, request.TariffID)
	if err != nil {
		return dto.OrderEstimateResponse{}, err
	}
	carClass, err := service.orderRepository.GetActiveCarClassByID(ctx, carClassID)
	if err != nil {
		return dto.OrderEstimateResponse{}, mapCarClassError(err)
	}

	pickup, destination, err := orderEstimateCoordinates(request)
	if err != nil {
		return dto.OrderEstimateResponse{}, err
	}

	distanceKM, durationMinutes, priceAmount, err := service.estimateOrder(ctx, carClass, pickup, destination)
	if err != nil {
		return dto.OrderEstimateResponse{}, err
	}

	return dto.OrderEstimateResponse{
		TariffID:     carClass.ID,
		TariffName:   carClass.Name,
		CarClassID:   &carClass.ID,
		CarClassName: carClass.Name,
		CarClass:     carClass.Code,
		DistanceKM:   distanceKM,
		DurationMin:  durationMinutes,
		Price:        priceAmount / 100,
		Currency:     carClass.BasePrice.Currency,
		PriceType:    "estimated",
	}, nil
}

func (service *OrderService) CreatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error) {
	passengerRecord, err := service.ensureActivePassenger(ctx, passengerID)
	if err != nil {
		return dto.PassengerOrderResponse{}, err
	}

	carClassID, err := resolveRequestedCarClassID(request.CarClassID, request.TariffID)
	if err != nil {
		return dto.PassengerOrderResponse{}, err
	}
	carClass, err := service.orderRepository.GetActiveCarClassByID(ctx, carClassID)
	if err != nil {
		return dto.PassengerOrderResponse{}, mapCarClassError(err)
	}

	pickup, destination, err := orderCreateCoordinates(request)
	if err != nil {
		return dto.PassengerOrderResponse{}, err
	}

	cityID, err := service.resolveCityID(ctx, request.CityID, pickup)
	if err != nil {
		return dto.PassengerOrderResponse{}, err
	}

	_, _, priceAmount, err := service.estimateOrder(ctx, carClass, pickup, destination)
	if err != nil {
		return dto.PassengerOrderResponse{}, err
	}

	createdOrder, err := service.orderRepository.CreatePassengerOrder(ctx, CreateOrderRecord{
		PassengerID:                     passengerRecord.ID,
		CityID:                          cityID,
		CarClassID:                      carClass.ID,
		PickupAddress:                   strings.TrimSpace(request.PickupAddress),
		PickupEntrance:                  strings.TrimSpace(request.PickupEntrance),
		PickupComment:                   strings.TrimSpace(request.PickupComment),
		PickupLocation:                  pickup,
		DestinationAddress:              strings.TrimSpace(request.DestinationAddress),
		DestinationLocation:             destination,
		EstimatedPrice:                  domain.Money{Amount: priceAmount, Currency: carClass.BasePrice.Currency},
		PaymentMethod:                   request.PaymentType,
		PassengerComment:                strings.TrimSpace(request.Comment),
		PassengerLocationSharingEnabled: request.PassengerLocationSharingEnabled,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.PassengerOrderResponse{}, ErrPassengerActiveOrderExists
		}
		return dto.PassengerOrderResponse{}, fmt.Errorf("create passenger order: %w", err)
	}

	if service.dispatchQueue != nil {
		if err := service.dispatchQueue.EnqueueOrder(ctx, createdOrder.Order.ID); err != nil {
			return dto.PassengerOrderResponse{}, fmt.Errorf("enqueue passenger order dispatch: %w", err)
		}
	}

	return passengerOrderResponse(createdOrder), nil
}

func (service *OrderService) GetCurrentPassengerOrder(ctx context.Context, passengerID uuid.UUID) (dto.PassengerOrderResponse, error) {
	if _, err := service.ensureActivePassenger(ctx, passengerID); err != nil {
		return dto.PassengerOrderResponse{}, err
	}

	orderDetails, err := service.orderRepository.GetCurrentPassengerOrder(ctx, passengerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.PassengerOrderResponse{}, ErrPassengerOrderNotFound
		}
		return dto.PassengerOrderResponse{}, fmt.Errorf("get current passenger order: %w", err)
	}
	return passengerOrderResponse(orderDetails), nil
}

func (service *OrderService) ListPassengerOrderHistory(ctx context.Context, passengerID uuid.UUID) (dto.OrderHistoryResponse, error) {
	if _, err := service.ensureActivePassenger(ctx, passengerID); err != nil {
		return dto.OrderHistoryResponse{}, err
	}

	orders, err := service.orderRepository.ListPassengerOrderHistory(ctx, passengerID, 50)
	if err != nil {
		return dto.OrderHistoryResponse{}, fmt.Errorf("list passenger order history: %w", err)
	}

	responseOrders := make([]dto.PassengerOrderResponse, 0, len(orders))
	for _, item := range orders {
		responseOrders = append(responseOrders, passengerOrderResponse(item))
	}
	return dto.OrderHistoryResponse{Orders: responseOrders}, nil
}

func (service *OrderService) GetPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (dto.PassengerOrderResponse, error) {
	if _, err := service.ensureActivePassenger(ctx, passengerID); err != nil {
		return dto.PassengerOrderResponse{}, err
	}

	orderDetails, err := service.orderRepository.GetPassengerOrder(ctx, passengerID, orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.PassengerOrderResponse{}, ErrPassengerOrderNotFound
		}
		return dto.PassengerOrderResponse{}, fmt.Errorf("get passenger order: %w", err)
	}
	return passengerOrderResponse(orderDetails), nil
}

func (service *OrderService) CancelPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.CancelOrderRequest) (dto.PassengerOrderResponse, error) {
	if _, err := service.ensureActivePassenger(ctx, passengerID); err != nil {
		return dto.PassengerOrderResponse{}, err
	}

	cancelledOrder, err := service.orderRepository.CancelPassengerOrder(ctx, passengerID, orderID, strings.TrimSpace(request.Reason), time.Now().UTC())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.PassengerOrderResponse{}, ErrPassengerOrderNotFound
		}
		return dto.PassengerOrderResponse{}, fmt.Errorf("cancel passenger order: %w", err)
	}
	return passengerOrderResponse(cancelledOrder), nil
}

func (service *OrderService) RatePassengerOrder(context.Context, uuid.UUID, uuid.UUID, dto.RateOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, common.ErrNotImplemented
}

func (service *OrderService) ensureActivePassenger(ctx context.Context, passengerID uuid.UUID) (domain.Passenger, error) {
	passengerRecord, err := service.passengerRepository.GetByID(ctx, passengerID)
	if err != nil {
		return domain.Passenger{}, fmt.Errorf("get passenger: %w", err)
	}
	if !passengerRecord.IsActive {
		return domain.Passenger{}, ErrPassengerInactive
	}
	return passengerRecord, nil
}

func (service *OrderService) resolveCityID(ctx context.Context, cityID *uuid.UUID, pickup geodomain.Coordinates) (uuid.UUID, error) {
	if cityID != nil {
		return *cityID, nil
	}
	if service.cityResolver == nil {
		return uuid.Nil, fmt.Errorf("resolve passenger order city: city resolver is not configured")
	}
	cityContext, found, err := service.cityResolver.ResolveCityByCoordinates(ctx, pickup)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve passenger order city by pickup coordinates: %w", err)
	}
	if !found {
		return uuid.Nil, fmt.Errorf("resolve passenger order city by pickup coordinates: city not found")
	}
	return cityContext.CityID, nil
}

func (service *OrderService) estimateOrder(ctx context.Context, carClass domain.CarClass, pickup geodomain.Coordinates, destination geodomain.Coordinates) (float64, int64, int64, error) {
	distanceKM, err := service.orderRepository.EstimateRoute(ctx, pickup, destination)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("estimate passenger route distance: %w", err)
	}

	durationMinutes := int64(math.Ceil(distanceKM / 0.5))
	if durationMinutes < 1 {
		durationMinutes = 1
	}

	priceAmount := carClass.BasePrice.Amount +
		int64(math.Round(distanceKM*float64(carClass.PricePerKM.Amount))) +
		durationMinutes*carClass.PricePerMinute.Amount
	if priceAmount < carClass.MinimumPrice.Amount {
		priceAmount = carClass.MinimumPrice.Amount
	}

	return distanceKM, durationMinutes, priceAmount, nil
}

func orderEstimateCoordinates(request dto.OrderEstimateRequest) (geodomain.Coordinates, geodomain.Coordinates, error) {
	pickup, err := geodomain.NewCoordinates(request.PickupLocation.Latitude, request.PickupLocation.Longitude)
	if err != nil {
		return geodomain.Coordinates{}, geodomain.Coordinates{}, fmt.Errorf("pickup coordinates: %w", err)
	}
	destination, err := geodomain.NewCoordinates(request.DestinationLocation.Latitude, request.DestinationLocation.Longitude)
	if err != nil {
		return geodomain.Coordinates{}, geodomain.Coordinates{}, fmt.Errorf("destination coordinates: %w", err)
	}
	return pickup, destination, nil
}

func orderCreateCoordinates(request dto.PassengerCreateOrderRequest) (geodomain.Coordinates, geodomain.Coordinates, error) {
	pickup, err := geodomain.NewCoordinates(request.PickupLocation.Latitude, request.PickupLocation.Longitude)
	if err != nil {
		return geodomain.Coordinates{}, geodomain.Coordinates{}, fmt.Errorf("pickup coordinates: %w", err)
	}
	destination, err := geodomain.NewCoordinates(request.DestinationLocation.Latitude, request.DestinationLocation.Longitude)
	if err != nil {
		return geodomain.Coordinates{}, geodomain.Coordinates{}, fmt.Errorf("destination coordinates: %w", err)
	}
	return pickup, destination, nil
}

func passengerCarClassDTO(carClass domain.CarClass) dto.PassengerCarClassResponse {
	return dto.PassengerCarClassResponse{
		ID:             carClass.ID,
		Code:           carClass.Code,
		Name:           carClass.Name,
		Description:    carClass.Description,
		BasePrice:      carClass.BasePrice.Amount,
		PricePerKM:     carClass.PricePerKM.Amount,
		PricePerMinute: carClass.PricePerMinute.Amount,
		MinimumPrice:   carClass.MinimumPrice.Amount,
		Currency:       carClass.BasePrice.Currency,
		SortOrder:      carClass.SortOrder,
	}
}

func passengerOrderResponse(details PassengerOrderDetails) dto.PassengerOrderResponse {
	responseBody := dto.PassengerOrderResponse{
		OrderID:  details.Order.ID,
		CarClass: "",
		PickupPoint: dto.PointDTO{
			Address: details.Order.PickupAddress,
			Location: dto.CoordinatesResponse{
				Latitude:  details.Order.PickupLocation.Latitude,
				Longitude: details.Order.PickupLocation.Longitude,
			},
		},
		PickupEntrance: details.Order.PickupEntrance,
		PickupComment:  details.Order.PickupComment,
		DestinationPoint: dto.PointDTO{
			Address: details.Order.DestinationAddress,
		},
		Status:         details.Order.Status,
		AllowedActions: passengerAllowedActions(details.Order.Status),
		Timeline:       []dto.OrderTimelineItem{{Status: details.Order.Status, OccurredAt: details.Order.CreatedAt}},
		Version:        details.Order.Version,
	}

	if details.Order.DestinationLocation != nil {
		responseBody.DestinationPoint.Location = dto.CoordinatesResponse{
			Latitude:  details.Order.DestinationLocation.Latitude,
			Longitude: details.Order.DestinationLocation.Longitude,
		}
	}
	if details.Order.EstimatedPrice != nil {
		responseBody.Price = &dto.MoneyResponse{
			Amount:   details.Order.EstimatedPrice.Amount,
			Currency: details.Order.EstimatedPrice.Currency,
		}
	}
	if details.Order.FinalPrice != nil {
		responseBody.Price = &dto.MoneyResponse{
			Amount:   details.Order.FinalPrice.Amount,
			Currency: details.Order.FinalPrice.Currency,
		}
	}
	if details.CarClass != nil {
		responseBody.CarClassID = &details.CarClass.ID
		responseBody.CarClassName = details.CarClass.Name
		responseBody.CarClass = details.CarClass.Code
	}
	if details.Driver != nil {
		responseBody.Driver = &dto.AssignedDriverDTO{
			ID:           details.Driver.ID,
			Name:         details.Driver.Name,
			Phone:        details.Driver.Phone,
			PhotoURL:     details.Driver.AvatarURL,
			Rating:       details.Driver.Rating,
			RatingsCount: details.Driver.RatingsCount,
		}
	}
	if details.Car != nil {
		responseBody.Car = &dto.CarDTO{
			ID:          details.Car.ID,
			Brand:       details.Car.Brand,
			Model:       details.Car.Model,
			Color:       details.Car.Color,
			PlateNumber: details.Car.PlateNumber,
			CarClass:    details.Car.CarClass,
		}
	}

	return responseBody
}

func passengerAllowedActions(status domain.OrderStatus) []string {
	actions := order.AllowedPassengerActions(status)
	result := make([]string, 0, len(actions))
	for _, action := range actions {
		result = append(result, string(action))
	}
	return result
}

func resolveRequestedCarClassID(requestCarClassID *uuid.UUID, fallbackTariffID uuid.UUID) (uuid.UUID, error) {
	if requestCarClassID != nil {
		return *requestCarClassID, nil
	}
	if fallbackTariffID != uuid.Nil {
		return fallbackTariffID, nil
	}
	return uuid.Nil, ErrPassengerCarClassRequired
}

func mapCarClassError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPassengerCarClassNotFound
	}
	return fmt.Errorf("get active car class: %w", err)
}

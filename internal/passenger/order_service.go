package passenger

import (
	"context"
	"errors"
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/kishert-lab/taxi-platform/internal/common"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/order"
	"github.com/kishert-lab/taxi-platform/internal/routing"
)

var (
	ErrPassengerOrderNotFound     = errors.New("passenger order not found")
	ErrPassengerActiveOrderExists = errors.New("passenger already has active order")
	ErrPassengerCarClassRequired  = errors.New("car class is required")
	ErrPassengerCarClassNotFound  = errors.New("car class not found")
	ErrPassengerInactive          = errors.New("passenger is inactive")
	ErrPriceQuoteStale            = errors.New("passenger price quote expired or changed")
	ErrPriceQuoteRequired         = errors.New("passenger price quote is required")
)

type PriceQuoteChangedError struct {
	Updated dto.OrderEstimateResponse
}

func (err *PriceQuoteChangedError) Error() string { return ErrPriceQuoteStale.Error() }
func (err *PriceQuoteChangedError) Unwrap() error { return ErrPriceQuoteStale }

type OrderService struct {
	passengerRepository Repository
	orderRepository     OrderRepository
	dispatchQueue       DispatchQueue
	cityResolver        CityResolver
	routingService      routing.Service
}

func NewOrderService(
	passengerRepository Repository,
	orderRepository OrderRepository,
	dispatchQueue DispatchQueue,
	cityResolver CityResolver,
	routingService routing.Service,
) *OrderService {
	if routingService == nil {
		routingService = routing.UnavailableService{}
	}
	return &OrderService{
		passengerRepository: passengerRepository,
		orderRepository:     orderRepository,
		dispatchQueue:       dispatchQueue,
		cityResolver:        cityResolver,
		routingService:      routingService,
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
	if request.FareMode == "" {
		request.FareMode = domain.FareModeMetered // legacy clients omitted the mode
	}
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

	cityID, err := service.resolveCityID(ctx, request.CityID, pickup)
	if err != nil {
		return dto.OrderEstimateResponse{}, err
	}

	estimate, err := service.buildTaxiParkEstimate(ctx, carClass, cityID, pickup, destination, request.FareMode)
	if err != nil {
		return dto.OrderEstimateResponse{}, err
	}

	responseBody := dto.OrderEstimateResponse{
		FareMode:     request.FareMode,
		TariffID:     carClass.ID,
		TariffName:   carClass.Name,
		CarClassID:   &carClass.ID,
		CarClassName: carClass.Name,
		CarClass:     carClass.Code,
		DistanceKM:   estimate.DistanceKM,
		DurationMin:  estimate.DurationMinutes,
		Currency:     carClass.BasePrice.Currency,
		PriceType:    "estimated",
		Pricing:      pricingResponse(&estimate.Pricing, nil, nil),
	}
	if estimate.PriceAmount != nil {
		responseBody.Price = *estimate.PriceAmount / 100
		responseBody.PriceCents = estimate.PriceAmount
		quoteID := uuid.New()
		expiresAt := time.Now().UTC().Add(2 * time.Minute)
		if err := service.orderRepository.SavePriceQuote(ctx, PriceQuote{
			ID: quoteID, PassengerID: passengerID, CityID: cityID, CarClassID: carClass.ID,
			FareMode: request.FareMode, Pickup: pickup, Destination: destination,
			Pricing: estimate.Pricing, ExpiresAt: expiresAt,
		}); err != nil {
			return dto.OrderEstimateResponse{}, fmt.Errorf("save passenger price quote: %w", err)
		}
		responseBody.QuoteID = &quoteID
		responseBody.ExpiresAt = &expiresAt
	}

	return responseBody, nil
}

func (service *OrderService) CreatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error) {
	if request.QuoteID == nil || *request.QuoteID == uuid.Nil {
		return dto.PassengerOrderResponse{}, ErrPriceQuoteRequired
	}
	if request.FareMode == "" {
		request.FareMode = domain.FareModeMetered // legacy clients omitted the mode
	}
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
	quote, err := service.orderRepository.GetPriceQuote(ctx, *request.QuoteID, passengerID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.PassengerOrderResponse{}, ErrPriceQuoteStale
		}
		return dto.PassengerOrderResponse{}, fmt.Errorf("get passenger price quote: %w", err)
	}
	quoteStale := quote.ConsumedOrderID != nil || !time.Now().UTC().Before(quote.ExpiresAt) ||
		quote.CityID != cityID || quote.CarClassID != carClass.ID || quote.FareMode != request.FareMode ||
		quote.Pickup != pickup || quote.Destination != destination

	estimate, err := service.buildTaxiParkEstimate(ctx, carClass, cityID, pickup, destination, request.FareMode)
	if err != nil {
		return dto.PassengerOrderResponse{}, err
	}
	if quoteStale || !estimate.Pricing.PriceAvailable || !sameQuotedTerms(quote.Pricing, estimate.Pricing) {
		updated := dto.OrderEstimateResponse{
			FareMode: request.FareMode, TariffID: carClass.ID, TariffName: carClass.Name,
			CarClassID: &carClass.ID, CarClassName: carClass.Name, CarClass: carClass.Code,
			DistanceKM: estimate.DistanceKM, DurationMin: estimate.DurationMinutes,
			Currency: carClass.BasePrice.Currency, PriceType: "estimated",
			Pricing: pricingResponse(&estimate.Pricing, nil, nil),
		}
		if estimate.PriceAmount != nil {
			updated.PriceCents = estimate.PriceAmount
			updated.Price = *estimate.PriceAmount / 100
			newQuoteID := uuid.New()
			expiresAt := time.Now().UTC().Add(2 * time.Minute)
			if err := service.orderRepository.SavePriceQuote(ctx, PriceQuote{
				ID: newQuoteID, PassengerID: passengerID, CityID: cityID, CarClassID: carClass.ID,
				FareMode: request.FareMode, Pickup: pickup, Destination: destination,
				Pricing: estimate.Pricing, ExpiresAt: expiresAt,
			}); err != nil {
				return dto.PassengerOrderResponse{}, fmt.Errorf("save updated passenger price quote: %w", err)
			}
			updated.QuoteID = &newQuoteID
			updated.ExpiresAt = &expiresAt
		}
		return dto.PassengerOrderResponse{}, &PriceQuoteChangedError{Updated: updated}
	}

	createdOrder, err := service.orderRepository.CreatePassengerOrder(ctx, CreateOrderRecord{
		QuoteID:                         quote.ID,
		FareMode:                        request.FareMode,
		PassengerID:                     passengerRecord.ID,
		CityID:                          cityID,
		CarClassID:                      carClass.ID,
		PickupAddress:                   strings.TrimSpace(request.PickupAddress),
		PickupEntrance:                  strings.TrimSpace(request.PickupEntrance),
		PickupComment:                   strings.TrimSpace(request.PickupComment),
		PickupLocation:                  pickup,
		DestinationAddress:              strings.TrimSpace(request.DestinationAddress),
		DestinationLocation:             destination,
		EstimatedPrice:                  &domain.Money{Amount: quote.Pricing.EstimatedPrice.Amount, Currency: carClass.BasePrice.Currency},
		PricingSnapshot:                 &quote.Pricing,
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

func (service *OrderService) ConfirmPassengerPrice(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (dto.PassengerOrderResponse, error) {
	if _, err := service.ensureActivePassenger(ctx, passengerID); err != nil {
		return dto.PassengerOrderResponse{}, err
	}
	if service.dispatchQueue == nil {
		return dto.PassengerOrderResponse{}, fmt.Errorf("price confirmation dispatch service unavailable")
	}
	if err := service.dispatchQueue.ConfirmOrderPrice(ctx, orderID, passengerID); err != nil {
		return dto.PassengerOrderResponse{}, fmt.Errorf("confirm passenger price: %w", err)
	}
	return service.GetPassengerOrder(ctx, passengerID, orderID)
}

func (service *OrderService) DeclinePassengerPrice(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (dto.PassengerOrderResponse, error) {
	if _, err := service.ensureActivePassenger(ctx, passengerID); err != nil {
		return dto.PassengerOrderResponse{}, err
	}
	if service.dispatchQueue == nil {
		return dto.PassengerOrderResponse{}, fmt.Errorf("price confirmation dispatch service unavailable")
	}
	if err := service.dispatchQueue.DeclineOrderPrice(ctx, orderID, passengerID); err != nil {
		return dto.PassengerOrderResponse{}, fmt.Errorf("decline passenger price: %w", err)
	}
	return service.GetPassengerOrder(ctx, passengerID, orderID)
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

	durationMinutes := estimateDurationMinutes(distanceKM)
	priceAmount := calculateCarClassEstimate(carClass, distanceKM, durationMinutes)
	return distanceKM, durationMinutes, priceAmount, nil
}

func estimateDurationMinutes(distanceKM float64) int64 {
	durationMinutes := int64(math.Ceil(distanceKM / 0.5))
	if durationMinutes < 1 {
		durationMinutes = 1
	}
	return durationMinutes
}

func calculateCarClassEstimate(carClass domain.CarClass, distanceKM float64, durationMinutes int64) int64 {
	priceAmount := carClass.BasePrice.Amount +
		int64(math.Round(distanceKM*float64(carClass.PricePerKM.Amount))) +
		durationMinutes*carClass.PricePerMinute.Amount
	if priceAmount < carClass.MinimumPrice.Amount {
		priceAmount = carClass.MinimumPrice.Amount
	}
	return priceAmount
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

func passengerOrderResponse(details OrderDetails) dto.PassengerOrderResponse {
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
		Status:                     details.Order.Status,
		AllowedActions:             passengerAllowedActions(details.Order.Status),
		Timeline:                   []dto.OrderTimelineItem{{Status: details.Order.Status, OccurredAt: details.Order.CreatedAt}},
		Version:                    details.Order.Version,
		Pricing:                    pricingResponse(details.Pricing, details.Order.AssignedTariffID, details.Order.ParkID),
		PriceConfirmationState:     details.Order.PriceConfirmationState,
		ProposedPriceCents:         details.Order.ProposedPriceCents,
		AgreedPriceCents:           details.Order.AgreedPriceCents,
		PriceConfirmationExpiresAt: details.Order.PriceConfirmationExpiresAt,
	}
	responseBody.Pricing.AssignedTariffRates = details.AssignedTariffRates
	if details.Order.PriceConfirmationState == "pending" {
		responseBody.AllowedActions = append(responseBody.AllowedActions, "confirm_price", "decline_price")
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
		if responseBody.Pricing.EstimatedPrice == nil {
			responseBody.Pricing.EstimatedPrice = &dto.MoneyResponse{
				Amount:   details.Order.EstimatedPrice.Amount,
				Currency: details.Order.EstimatedPrice.Currency,
			}
			responseBody.Pricing.EstimatedPriceMin = responseBody.Pricing.EstimatedPrice
			responseBody.Pricing.EstimatedPriceMax = responseBody.Pricing.EstimatedPrice
			responseBody.Pricing.PriceAvailable = true
		}
	}
	if details.Order.AgreedPriceCents != nil {
		agreed := &dto.MoneyResponse{Amount: *details.Order.AgreedPriceCents, Currency: "RUB"}
		responseBody.Price = agreed
		responseBody.Pricing.AgreedPrice = agreed
	}
	if details.Order.FinalPrice != nil {
		responseBody.Price = &dto.MoneyResponse{
			Amount:   details.Order.FinalPrice.Amount,
			Currency: details.Order.FinalPrice.Currency,
		}
		responseBody.Pricing.FinalPrice = &dto.MoneyResponse{
			Amount:   details.Order.FinalPrice.Amount,
			Currency: details.Order.FinalPrice.Currency,
		}
		responseBody.Pricing.IsFinal = true
		responseBody.Pricing.PriceAvailable = true
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

type passengerEstimateDetails struct {
	DistanceKM      float64
	DurationMinutes int64
	PriceAmount     *int64
	Pricing         domain.OrderPricingSnapshot
}

func (details passengerEstimateDetails) estimatedMoney(currency string) *domain.Money {
	if details.PriceAmount == nil {
		return nil
	}
	return &domain.Money{Amount: *details.PriceAmount, Currency: currency}
}

func sameQuotedTerms(quoted domain.OrderPricingSnapshot, current domain.OrderPricingSnapshot) bool {
	return quoted.PriceAvailable && current.PriceAvailable &&
		reflect.DeepEqual(quoted.EstimatedPrice, current.EstimatedPrice) &&
		reflect.DeepEqual(quoted.EstimatedPriceMin, current.EstimatedPriceMin) &&
		reflect.DeepEqual(quoted.EstimatedPriceMax, current.EstimatedPriceMax) &&
		reflect.DeepEqual(quoted.RouteDistanceMeters, current.RouteDistanceMeters) &&
		reflect.DeepEqual(quoted.RouteDurationSeconds, current.RouteDurationSeconds) &&
		quoted.RouteSource == current.RouteSource && quoted.RouteDataVersion == current.RouteDataVersion &&
		reflect.DeepEqual(quoted.TariffRates, current.TariffRates)
}

func (service *OrderService) buildTaxiParkEstimate(ctx context.Context, carClass domain.CarClass, cityID uuid.UUID, pickup geodomain.Coordinates, destination geodomain.Coordinates, fareMode domain.FareMode) (passengerEstimateDetails, error) {
	if fareMode != domain.FareModeFixedQuote && fareMode != domain.FareModeMetered {
		return passengerEstimateDetails{}, fmt.Errorf("invalid fare mode %q", fareMode)
	}
	route, err := service.routingService.Route(ctx, pickup, destination)
	if err != nil {
		return unavailableTaxiParkEstimate(route, routingUnavailableReason(err), fareMode), nil
	}

	searchRadiusMeters, tariffs, err := service.resolveTaxiParkTariffs(ctx, pickup, cityID, carClass.ID, fareMode)
	if err != nil {
		return passengerEstimateDetails{}, err
	}
	if len(tariffs) == 0 {
		availableDrivers, err := service.orderRepository.HasNearbyAvailableDrivers(ctx, pickup, cityID, carClass.ID, 10000, 30*time.Second)
		if err != nil {
			return passengerEstimateDetails{}, fmt.Errorf("check available drivers for unavailable estimate: %w", err)
		}
		if !availableDrivers {
			return unavailableTaxiParkEstimate(route, "no_available_drivers", fareMode), nil
		}
		return unavailableTaxiParkEstimate(route, "no_available_park_tariffs", fareMode), nil
	}

	var totalPrice int64
	var minimumPrice int64
	var maximumPrice int64
	validTariffs := make([]domain.TaxiParkTariff, 0, len(tariffs))
	validPrices := make([]int64, 0, len(tariffs))
	excludedTariffReasons := make([]string, 0)
	seenParks := make(map[uuid.UUID]struct{}, len(tariffs))
	for _, tariff := range tariffs {
		if _, exists := seenParks[tariff.TaxiParkID]; exists {
			continue
		}
		price, calculateErr := domain.CalculateTripPrice(tariff, route.DistanceMeters, route.DurationSeconds)
		if calculateErr != nil {
			excludedTariffReasons = append(excludedTariffReasons, fmt.Sprintf("tariff %s: %v", tariff.ID, calculateErr))
			continue
		}
		if price <= 0 {
			excludedTariffReasons = append(excludedTariffReasons, fmt.Sprintf("tariff %s: nonpositive price", tariff.ID))
			continue
		}
		seenParks[tariff.TaxiParkID] = struct{}{}
		if len(validTariffs) == 0 || price < minimumPrice {
			minimumPrice = price
		}
		if len(validTariffs) == 0 || price > maximumPrice {
			maximumPrice = price
		}
		if totalPrice > math.MaxInt64-price {
			return passengerEstimateDetails{}, fmt.Errorf("sum available taxi park tariffs: overflow")
		}
		totalPrice += price
		validTariffs = append(validTariffs, tariff)
		validPrices = append(validPrices, price)
	}
	if len(validTariffs) == 0 {
		unavailable := unavailableTaxiParkEstimate(route, "no_valid_park_tariffs", fareMode)
		unavailable.Pricing.ExcludedTariffReasons = excludedTariffReasons
		return unavailable, nil
	}
	averagePrice := totalPrice/int64(len(validTariffs)) + (totalPrice%int64(len(validTariffs))+int64(len(validTariffs))/2)/int64(len(validTariffs))
	rateSnapshots := domain.SnapshotTariffRates(validTariffs)
	for index, price := range validPrices {
		rateSnapshots[index].ProjectedPrice = &domain.Money{Amount: price, Currency: "RUB"}
	}
	snapshot := domain.OrderPricingSnapshot{
		EstimatedPrice:        &domain.Money{Amount: averagePrice, Currency: "RUB"},
		EstimatedPriceMin:     &domain.Money{Amount: minimumPrice, Currency: "RUB"},
		EstimatedPriceMax:     &domain.Money{Amount: maximumPrice, Currency: "RUB"},
		EstimatedPriceSource:  domain.EstimatedPriceSourceAverageParks,
		PricingMode:           domain.PricingModeUnknown,
		FareMode:              fareMode,
		PriceAvailable:        true,
		SearchRadiusMeters:    searchRadiusMeters,
		RouteDistanceMeters:   &route.DistanceMeters,
		RouteDurationSeconds:  &route.DurationSeconds,
		RouteSource:           route.Source,
		RouteDataVersion:      route.DataVersion,
		TariffRates:           rateSnapshots,
		RoundingRule:          "distance_nearest_kopeck;time_started_minute;average_nearest_kopeck",
		TaxiParkCount:         len(validTariffs),
		ExcludedTariffReasons: excludedTariffReasons,
		CalculatedAt:          route.CalculatedAt,
	}
	return passengerEstimateDetails{
		DistanceKM:      float64(route.DistanceMeters) / 1000,
		DurationMinutes: (route.DurationSeconds + 59) / 60,
		PriceAmount:     &averagePrice,
		Pricing:         snapshot,
	}, nil
}

func (service *OrderService) resolveTaxiParkTariffs(ctx context.Context, pickup geodomain.Coordinates, cityID uuid.UUID, carClassID uuid.UUID, fareMode domain.FareMode) (*int, []domain.TaxiParkTariff, error) {
	radius := 10000
	tariffs, err := service.orderRepository.ListAvailableTaxiParkTariffs(ctx, pickup, cityID, carClassID, fareMode, radius, 30*time.Second)
	if err != nil {
		return nil, nil, fmt.Errorf("list available taxi park tariffs for estimate: %w", err)
	}
	return &radius, tariffs, nil
}

func unavailableTaxiParkEstimate(route routing.Route, reason string, fareMode domain.FareMode) passengerEstimateDetails {
	snapshot := domain.OrderPricingSnapshot{
		EstimatedPriceSource: domain.EstimatedPriceSourceUnavailable,
		PricingMode:          domain.PricingModeUnknown,
		FareMode:             fareMode,
		Message:              "Price is unavailable",
		UnavailableReason:    reason,
		RouteSource:          route.Source,
		CalculatedAt:         time.Now().UTC(),
	}
	if route.DistanceMeters > 0 {
		snapshot.RouteDistanceMeters = &route.DistanceMeters
	}
	if route.DurationSeconds > 0 {
		snapshot.RouteDurationSeconds = &route.DurationSeconds
	}
	return passengerEstimateDetails{
		DistanceKM:      float64(route.DistanceMeters) / 1000,
		DurationMinutes: (route.DurationSeconds + 59) / 60,
		Pricing:         snapshot,
	}
}

func routingUnavailableReason(err error) string {
	switch {
	case errors.Is(err, routing.ErrRouteNotFound):
		return "route_not_found"
	case errors.Is(err, routing.ErrInvalidResponse):
		return "routing_invalid_response"
	default:
		return "routing_unavailable"
	}
}

func (service *OrderService) buildPassengerEstimate(ctx context.Context, carClass domain.CarClass, cityID uuid.UUID, pickup geodomain.Coordinates, destination geodomain.Coordinates) (passengerEstimateDetails, error) {
	distanceKM, durationMinutes, priceAmount, err := service.estimateOrder(ctx, carClass, pickup, destination)
	if err != nil {
		return passengerEstimateDetails{}, err
	}

	searchRadiusMeters, priceAvailable, err := service.resolvePassengerEstimateAvailability(ctx, pickup, cityID, carClass.ID)
	if err != nil {
		return passengerEstimateDetails{}, err
	}

	mode := pricingModeForCarClass(carClass)
	snapshot := domain.OrderPricingSnapshot{
		EstimatedPriceSource: domain.EstimatedPriceSourceCarClassCatalog,
		PricingMode:          mode,
		PriceAvailable:       priceAvailable,
		IsFinal:              false,
		CalculatedAt:         time.Now().UTC(),
		SearchRadiusMeters:   searchRadiusMeters,
	}
	if !priceAvailable {
		snapshot.EstimatedPriceSource = domain.EstimatedPriceSourceUnavailable
		snapshot.Message = "Цена будет рассчитана после назначения водителя"
		return passengerEstimateDetails{
			DistanceKM:      distanceKM,
			DurationMinutes: durationMinutes,
			Pricing:         snapshot,
		}, nil
	}

	snapshot.EstimatedPrice = &domain.Money{Amount: priceAmount, Currency: carClass.BasePrice.Currency}
	snapshot.EstimatedPriceMin = &domain.Money{Amount: priceAmount, Currency: carClass.BasePrice.Currency}
	snapshot.EstimatedPriceMax = &domain.Money{Amount: priceAmount, Currency: carClass.BasePrice.Currency}

	return passengerEstimateDetails{
		DistanceKM:      distanceKM,
		DurationMinutes: durationMinutes,
		PriceAmount:     &priceAmount,
		Pricing:         snapshot,
	}, nil
}

func (service *OrderService) resolvePassengerEstimateAvailability(ctx context.Context, pickup geodomain.Coordinates, cityID uuid.UUID, carClassID uuid.UUID) (*int, bool, error) {
	for _, radius := range []int{5000, 10000} {
		exists, err := service.orderRepository.HasNearbyAvailableDrivers(ctx, pickup, cityID, carClassID, radius, 2*time.Minute)
		if err != nil {
			return nil, false, fmt.Errorf("check nearby drivers for estimate: %w", err)
		}
		if exists {
			return &radius, true, nil
		}
	}
	return nil, false, nil
}

func pricingModeForCarClass(carClass domain.CarClass) domain.PricingMode {
	hasDistance := carClass.PricePerKM.Amount > 0
	hasTime := carClass.PricePerMinute.Amount > 0
	switch {
	case !hasDistance && !hasTime:
		return domain.PricingModeFixed
	case hasDistance && hasTime:
		return domain.PricingModeDistanceTime
	case hasDistance:
		return domain.PricingModeDistance
	case hasTime:
		return domain.PricingModeTime
	default:
		return domain.PricingModeUnknown
	}
}

func pricingResponse(snapshot *domain.OrderPricingSnapshot, assignedTariffID *uuid.UUID, assignedTaxiParkID *uuid.UUID) dto.OrderPricingResponse {
	result := dto.PricingSnapshotResponse(snapshot)
	result.AssignedTariffID = assignedTariffID
	result.AssignedTaxiParkID = assignedTaxiParkID
	return result
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

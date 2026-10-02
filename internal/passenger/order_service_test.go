package passenger

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
	"github.com/kishert-lab/taxi-platform/internal/routing"
)

func TestEstimatePassengerOrderReturnsStructuredPricingWhenDriversAvailable(t *testing.T) {
	passengerID := uuid.New()
	carClassID := uuid.New()

	service := NewOrderService(
		estimatePassengerRepository{passenger: domain.Passenger{ID: passengerID, IsActive: true}},
		&estimateOrderRepository{
			carClass: domain.CarClass{
				ID:             carClassID,
				Code:           "economy",
				Name:           "Эконом",
				BasePrice:      domain.Money{Amount: 12000, Currency: "RUB"},
				PricePerKM:     domain.Money{Amount: 1800, Currency: "RUB"},
				PricePerMinute: domain.Money{Amount: 600, Currency: "RUB"},
				MinimumPrice:   domain.Money{Amount: 18000, Currency: "RUB"},
			},
			nearbyDrivers5KM: true,
		},
		nil,
		estimateCityResolver{cityID: uuid.New()},
		estimateRouteService{},
	)

	result, err := service.EstimatePassengerOrder(context.Background(), passengerID, dto.OrderEstimateRequest{
		FareMode:   domain.FareModeMetered,
		CarClassID: &carClassID,
		PickupLocation: dto.CoordinatesRequest{
			Latitude:  58.0,
			Longitude: 56.2,
		},
		DestinationLocation: dto.CoordinatesRequest{
			Latitude:  58.1,
			Longitude: 56.3,
		},
	})
	if err != nil {
		t.Fatalf("estimate passenger order: %v", err)
	}

	if !result.Pricing.PriceAvailable {
		t.Fatalf("expected available pricing, got unavailable: %#v", result.Pricing)
	}
	if result.Pricing.EstimatedPrice == nil || result.Pricing.EstimatedPrice.Amount <= 0 {
		t.Fatalf("expected structured estimated price, got %#v", result.Pricing)
	}
	if result.Pricing.EstimatedPriceSource != string(domain.EstimatedPriceSourceAverageParks) {
		t.Fatalf("unexpected pricing source: %s", result.Pricing.EstimatedPriceSource)
	}
	if result.Pricing.PricingMode != string(domain.PricingModeUnknown) {
		t.Fatalf("unexpected pricing mode: %s", result.Pricing.PricingMode)
	}
	if result.Price != result.Pricing.EstimatedPrice.Amount/100 {
		t.Fatalf("legacy price field mismatch: price=%d structured=%d", result.Price, result.Pricing.EstimatedPrice.Amount)
	}
}

func TestEstimatePassengerOrderReturnsUnavailableWhenNoNearbyDrivers(t *testing.T) {
	passengerID := uuid.New()
	carClassID := uuid.New()

	service := NewOrderService(
		estimatePassengerRepository{passenger: domain.Passenger{ID: passengerID, IsActive: true}},
		&estimateOrderRepository{
			carClass: domain.CarClass{
				ID:             carClassID,
				Code:           "economy",
				Name:           "Эконом",
				BasePrice:      domain.Money{Amount: 12000, Currency: "RUB"},
				PricePerKM:     domain.Money{Amount: 1800, Currency: "RUB"},
				PricePerMinute: domain.Money{Amount: 600, Currency: "RUB"},
				MinimumPrice:   domain.Money{Amount: 18000, Currency: "RUB"},
			},
		},
		nil,
		estimateCityResolver{cityID: uuid.New()},
		estimateRouteService{},
	)

	result, err := service.EstimatePassengerOrder(context.Background(), passengerID, dto.OrderEstimateRequest{
		FareMode:   domain.FareModeMetered,
		CarClassID: &carClassID,
		PickupLocation: dto.CoordinatesRequest{
			Latitude:  58.0,
			Longitude: 56.2,
		},
		DestinationLocation: dto.CoordinatesRequest{
			Latitude:  58.1,
			Longitude: 56.3,
		},
	})
	if err != nil {
		t.Fatalf("estimate passenger order: %v", err)
	}

	if result.Pricing.PriceAvailable {
		t.Fatalf("expected unavailable pricing, got %#v", result.Pricing)
	}
	if result.Pricing.EstimatedPrice != nil {
		t.Fatalf("expected no estimated price, got %#v", result.Pricing.EstimatedPrice)
	}
	if result.Pricing.EstimatedPriceSource != string(domain.EstimatedPriceSourceUnavailable) {
		t.Fatalf("unexpected pricing source: %s", result.Pricing.EstimatedPriceSource)
	}
	if result.Pricing.Message == "" {
		t.Fatalf("expected unavailable pricing message")
	}
}

func TestEstimateWithoutRoadRouteHasNoPriceOrQuote(t *testing.T) {
	passengerID, classID := uuid.New(), uuid.New()
	service := NewOrderService(
		estimatePassengerRepository{passenger: domain.Passenger{ID: passengerID, IsActive: true}},
		&estimateOrderRepository{carClass: domain.CarClass{ID: classID, BasePrice: domain.Money{Currency: "RUB"}}, nearbyDrivers10: true},
		nil, estimateCityResolver{cityID: uuid.New()}, missingRouteService{},
	)
	result, err := service.EstimatePassengerOrder(context.Background(), passengerID, dto.OrderEstimateRequest{
		FareMode: domain.FareModeMetered, CarClassID: &classID,
		PickupLocation: dto.CoordinatesRequest{Latitude: 58, Longitude: 56}, DestinationLocation: dto.CoordinatesRequest{Latitude: 58.1, Longitude: 56.1},
	})
	if err != nil { t.Fatalf("estimate unavailable road route: %v", err) }
	if result.Pricing.PriceAvailable || result.PriceCents != nil || result.QuoteID != nil || result.Pricing.UnavailableReason == "" {
		t.Fatalf("missing road route produced a price or quote: %#v", result)
	}
}

type missingRouteService struct{}
func (missingRouteService) Route(context.Context, geodomain.Coordinates, geodomain.Coordinates) (routing.Route, error) {
	return routing.Route{}, routing.ErrRouteNotFound
}

type estimatePassengerRepository struct {
	passenger domain.Passenger
}

func (repository estimatePassengerRepository) Create(context.Context, domain.Passenger) (domain.Passenger, error) {
	panic("unexpected call")
}

func (repository estimatePassengerRepository) GetByID(context.Context, uuid.UUID) (domain.Passenger, error) {
	return repository.passenger, nil
}

func (repository estimatePassengerRepository) GetByPhone(context.Context, string) (domain.Passenger, error) {
	panic("unexpected call")
}

func (repository estimatePassengerRepository) UpdateProfile(context.Context, uuid.UUID, string, string, string) (domain.Passenger, error) {
	panic("unexpected call")
}

func (repository estimatePassengerRepository) MarkAuthenticated(context.Context, uuid.UUID, *time.Time, time.Time) (domain.Passenger, error) {
	panic("unexpected call")
}

type estimateOrderRepository struct {
	carClass         domain.CarClass
	nearbyDrivers5KM bool
	nearbyDrivers10  bool
	tariffs          []domain.TaxiParkTariff
	requestedRadius  int
	quotes           map[uuid.UUID]PriceQuote
}

func (repository *estimateOrderRepository) SavePriceQuote(_ context.Context, quote PriceQuote) error {
	if repository.quotes == nil {
		repository.quotes = make(map[uuid.UUID]PriceQuote)
	}
	repository.quotes[quote.ID] = quote
	return nil
}
func (repository *estimateOrderRepository) GetPriceQuote(_ context.Context, quoteID uuid.UUID, passengerID uuid.UUID) (PriceQuote, error) {
	quote, ok := repository.quotes[quoteID]
	if !ok || quote.PassengerID != passengerID {
		return PriceQuote{}, ErrPriceQuoteStale
	}
	return quote, nil
}

func TestExpiredEstimateRequiresFreshConfirmation(t *testing.T) {
	passengerID, classID, cityID := uuid.New(), uuid.New(), uuid.New()
	repository := &estimateOrderRepository{
		carClass:        domain.CarClass{ID: classID, Name: "Economy", BasePrice: domain.Money{Currency: "RUB"}},
		nearbyDrivers10: true,
	}
	service := NewOrderService(estimatePassengerRepository{passenger: domain.Passenger{ID: passengerID, IsActive: true}}, repository, nil, estimateCityResolver{cityID: cityID}, estimateRouteService{})
	pickup := dto.CoordinatesRequest{Latitude: 58, Longitude: 56}
	destination := dto.CoordinatesRequest{Latitude: 58.1, Longitude: 56.1}
	estimate, err := service.EstimatePassengerOrder(context.Background(), passengerID, dto.OrderEstimateRequest{
		FareMode: domain.FareModeMetered, CarClassID: &classID, PickupLocation: pickup, DestinationLocation: destination,
	})
	if err != nil {
		t.Fatalf("create estimate: %v", err)
	}
	if estimate.QuoteID == nil || estimate.ExpiresAt == nil {
		t.Fatalf("estimate missing server quote binding: %#v", estimate)
	}
	quote := repository.quotes[*estimate.QuoteID]
	quote.ExpiresAt = time.Now().Add(-time.Second)
	repository.quotes[*estimate.QuoteID] = quote
	_, err = service.CreatePassengerOrder(context.Background(), passengerID, dto.PassengerCreateOrderRequest{
		QuoteID: estimate.QuoteID, FareMode: domain.FareModeMetered, CarClassID: &classID,
		PickupLocation: pickup, DestinationLocation: destination,
		PickupAddress: "A", DestinationAddress: "B", PaymentType: domain.PaymentMethodCash,
	})
	if !errors.Is(err, ErrPriceQuoteStale) {
		t.Fatalf("expired quote accepted: %v", err)
	}
	var changed *PriceQuoteChangedError
	if !errors.As(err, &changed) || changed.Updated.QuoteID == nil || *changed.Updated.QuoteID == *estimate.QuoteID {
		t.Fatalf("fresh terms and quote not returned: %v", err)
	}
}

func (repository *estimateOrderRepository) ListActiveCarClasses(context.Context) ([]domain.CarClass, error) {
	panic("unexpected call")
}

func (repository *estimateOrderRepository) ListAvailableCarClasses(context.Context, geodomain.Coordinates, uuid.UUID, int, time.Duration) ([]domain.CarClass, error) {
	panic("unexpected call")
}

func (repository *estimateOrderRepository) GetActiveCarClassByID(context.Context, uuid.UUID) (domain.CarClass, error) {
	return repository.carClass, nil
}

func (repository *estimateOrderRepository) EstimateRoute(context.Context, geodomain.Coordinates, geodomain.Coordinates) (float64, error) {
	return 0, nil
}

func (repository *estimateOrderRepository) HasNearbyAvailableDrivers(_ context.Context, _ geodomain.Coordinates, _ uuid.UUID, _ uuid.UUID, radiusMeters int, _ time.Duration) (bool, error) {
	if radiusMeters <= 5000 {
		return repository.nearbyDrivers5KM, nil
	}
	return repository.nearbyDrivers10, nil
}

func (repository *estimateOrderRepository) ListAvailableTaxiParkTariffs(_ context.Context, _ geodomain.Coordinates, _ uuid.UUID, carClassID uuid.UUID, fareMode domain.FareMode, radiusMeters int, _ time.Duration) ([]domain.TaxiParkTariff, error) {
	repository.requestedRadius = radiusMeters
	if repository.tariffs != nil {
		filtered := make([]domain.TaxiParkTariff, 0, len(repository.tariffs))
		for _, tariff := range repository.tariffs {
			if tariff.FareMode == fareMode { filtered = append(filtered, tariff) }
		}
		return filtered, nil
	}
	available := repository.nearbyDrivers5KM
	if radiusMeters > 5000 {
		available = repository.nearbyDrivers10 || repository.nearbyDrivers5KM
	}
	if !available {
		return nil, nil
	}
	return []domain.TaxiParkTariff{{
		ID:             uuid.New(),
		TaxiParkID:     uuid.New(),
		CarClassID:     &carClassID,
		PricingMode:    domain.PricingModeDistanceTime,
		FareMode:       fareMode,
		BasePrice:      domain.Money{Amount: 12000, Currency: "RUB"},
		PricePerKM:     domain.Money{Amount: 1800, Currency: "RUB"},
		PricePerMinute: domain.Money{Amount: 600, Currency: "RUB"},
		MinimumPrice:   domain.Money{Amount: 18000, Currency: "RUB"},
	}}, nil
}

func TestFixedAndMeteredEstimatesUseSeparateParkTariffs(t *testing.T) {
	passengerID, classID, parkID := uuid.New(), uuid.New(), uuid.New()
	repository := &estimateOrderRepository{
		carClass: domain.CarClass{ID: classID, BasePrice: domain.Money{Currency: "RUB"}},
		tariffs: []domain.TaxiParkTariff{
			{ID: uuid.New(), TaxiParkID: parkID, FareMode: domain.FareModeFixedQuote, PricingMode: domain.PricingModeFixed, FixedPrice: domain.Money{Amount: 36000, Currency: "RUB"}},
			{ID: uuid.New(), TaxiParkID: parkID, FareMode: domain.FareModeMetered, PricingMode: domain.PricingModeFixed, FixedPrice: domain.Money{Amount: 28000, Currency: "RUB"}},
		},
	}
	service := NewOrderService(estimatePassengerRepository{passenger: domain.Passenger{ID: passengerID, IsActive: true}}, repository, nil, estimateCityResolver{cityID: uuid.New()}, estimateRouteService{})
	for _, testCase := range []struct { mode domain.FareMode; expected int64 }{{domain.FareModeFixedQuote, 36000}, {domain.FareModeMetered, 28000}} {
		result, err := service.EstimatePassengerOrder(context.Background(), passengerID, dto.OrderEstimateRequest{
			FareMode: testCase.mode, CarClassID: &classID,
			PickupLocation: dto.CoordinatesRequest{Latitude: 58, Longitude: 56}, DestinationLocation: dto.CoordinatesRequest{Latitude: 58.1, Longitude: 56.1},
		})
		if err != nil { t.Fatalf("estimate %s: %v", testCase.mode, err) }
		if result.PriceCents == nil || *result.PriceCents != testCase.expected { t.Fatalf("mode %s included wrong tariff: %#v", testCase.mode, result) }
	}
}

func TestEstimateUsesEveryParkWithinTenKilometersOnce(t *testing.T) {
	passengerID, classID := uuid.New(), uuid.New()
	firstPark, secondPark, thirdPark := uuid.New(), uuid.New(), uuid.New()
	testTariff := func(parkID uuid.UUID, cents int64) domain.TaxiParkTariff {
		return domain.TaxiParkTariff{
			ID: uuid.New(), TaxiParkID: parkID, CarClassID: &classID,
			FareMode: domain.FareModeFixedQuote, PricingMode: domain.PricingModeFixed,
			FixedPrice: domain.Money{Amount: cents, Currency: "RUB"},
		}
	}
	repository := &estimateOrderRepository{
		carClass: domain.CarClass{ID: classID, BasePrice: domain.Money{Currency: "RUB"}},
		tariffs: []domain.TaxiParkTariff{
			testTariff(firstPark, 28000), testTariff(firstPark, 28000),
			testTariff(secondPark, 32000), testTariff(thirdPark, 36000),
		},
	}
	service := NewOrderService(
		estimatePassengerRepository{passenger: domain.Passenger{ID: passengerID, IsActive: true}},
		repository, nil, estimateCityResolver{cityID: uuid.New()}, estimateRouteService{},
	)
	result, err := service.EstimatePassengerOrder(context.Background(), passengerID, dto.OrderEstimateRequest{
		FareMode: domain.FareModeFixedQuote, CarClassID: &classID,
		PickupLocation:      dto.CoordinatesRequest{Latitude: 58, Longitude: 56},
		DestinationLocation: dto.CoordinatesRequest{Latitude: 58.1, Longitude: 56.1},
	})
	if err != nil {
		t.Fatalf("estimate three parks: %v", err)
	}
	if repository.requestedRadius != 10000 {
		t.Fatalf("expected full 10 km radius, got %d", repository.requestedRadius)
	}
	if result.Pricing.TaxiParkCount != 3 || result.Pricing.EstimatedPrice.Amount != 32000 || result.Pricing.EstimatedPriceMin.Amount != 28000 || result.Pricing.EstimatedPriceMax.Amount != 36000 {
		t.Fatalf("unexpected independent park prices: %#v", result.Pricing)
	}
}

func (repository *estimateOrderRepository) CreatePassengerOrder(context.Context, CreateOrderRecord) (OrderDetails, error) {
	panic("unexpected call")
}

func (repository *estimateOrderRepository) GetCurrentPassengerOrder(context.Context, uuid.UUID) (OrderDetails, error) {
	panic("unexpected call")
}

func (repository *estimateOrderRepository) ListPassengerOrderHistory(context.Context, uuid.UUID, int) ([]OrderDetails, error) {
	panic("unexpected call")
}

func (repository *estimateOrderRepository) GetPassengerOrder(context.Context, uuid.UUID, uuid.UUID) (OrderDetails, error) {
	panic("unexpected call")
}

func (repository *estimateOrderRepository) CancelPassengerOrder(context.Context, uuid.UUID, uuid.UUID, string, time.Time) (OrderDetails, error) {
	panic("unexpected call")
}

type estimateCityResolver struct {
	cityID uuid.UUID
}

type estimateRouteService struct{}

func (estimateRouteService) Route(context.Context, geodomain.Coordinates, geodomain.Coordinates) (routing.Route, error) {
	return routing.Route{DistanceMeters: 4200, DurationSeconds: 660, Source: "osrm", CalculatedAt: time.Now().UTC()}, nil
}

func (resolver estimateCityResolver) ResolveCityByCoordinates(context.Context, geodomain.Coordinates) (geoservice.CityContext, bool, error) {
	return geoservice.CityContext{CityID: resolver.cityID}, true, nil
}

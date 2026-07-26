package passenger

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
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
			distanceKM:       4.2,
			nearbyDrivers5KM: true,
		},
		nil,
		estimateCityResolver{cityID: uuid.New()},
	)

	result, err := service.EstimatePassengerOrder(context.Background(), passengerID, dto.OrderEstimateRequest{
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
	if result.Pricing.EstimatedPriceSource != string(domain.EstimatedPriceSourceCarClassCatalog) {
		t.Fatalf("unexpected pricing source: %s", result.Pricing.EstimatedPriceSource)
	}
	if result.Pricing.PricingMode != string(domain.PricingModeDistanceTime) {
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
			distanceKM: 3.5,
		},
		nil,
		estimateCityResolver{cityID: uuid.New()},
	)

	result, err := service.EstimatePassengerOrder(context.Background(), passengerID, dto.OrderEstimateRequest{
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
	distanceKM       float64
	nearbyDrivers5KM bool
	nearbyDrivers10  bool
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
	return repository.distanceKM, nil
}

func (repository *estimateOrderRepository) HasNearbyAvailableDrivers(_ context.Context, _ geodomain.Coordinates, _ uuid.UUID, _ uuid.UUID, radiusMeters int, _ time.Duration) (bool, error) {
	if radiusMeters <= 5000 {
		return repository.nearbyDrivers5KM, nil
	}
	return repository.nearbyDrivers10, nil
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

func (resolver estimateCityResolver) ResolveCityByCoordinates(context.Context, geodomain.Coordinates) (geoservice.CityContext, bool, error) {
	return geoservice.CityContext{CityID: resolver.cityID}, true, nil
}

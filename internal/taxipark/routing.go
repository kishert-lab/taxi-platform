package taxipark

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/routing"
	"time"
)

type OrderTariffReader interface {
	GetOrderTariff(context.Context, uuid.UUID, uuid.UUID) (domain.TaxiParkTariff, error)
}

func (service *Service) EstimateOrder(ctx context.Context, actorID uuid.UUID, request dto.TaxiParkCreateOrderRequest) (*domain.OrderPricingSnapshot, error) {
	record, err := orderRecordFromRequest(request)
	if err != nil {
		return nil, err
	}
	if _, err := service.repository.GetSettingsByOwnerUserID(ctx, actorID); err != nil {
		return nil, err
	}
	return service.estimateRoadPrice(ctx, actorID, record.TariffID, record.PickupLocation, record.DestinationLocation)
}

func (service *Service) WithRoutingService(provider routing.Service, reader OrderTariffReader) *Service {
	service.routingService = provider
	service.tariffReader = reader
	return service
}

func (service *Service) estimateRoadPrice(ctx context.Context, actorID, tariffID uuid.UUID, pickup domain.Coordinates, destination *domain.Coordinates) (*domain.OrderPricingSnapshot, error) {
	snapshot := &domain.OrderPricingSnapshot{EstimatedPriceSource: domain.EstimatedPriceSourceUnavailable, UnavailableReason: "routing_unavailable", CalculatedAt: time.Now().UTC()}
	if service.tariffReader == nil {
		return snapshot, nil
	}
	tariff, err := service.tariffReader.GetOrderTariff(ctx, actorID, tariffID)
	if err != nil {
		return nil, fmt.Errorf("read order tariff for road estimate: %w", err)
	}
	snapshot.TariffRates = domain.SnapshotTariffRates([]domain.TaxiParkTariff{tariff})
	snapshot.PricingMode = tariff.PricingMode
	snapshot.RoundingRule = "distance_nearest_kopeck;time_started_minute"
	if destination == nil {
		snapshot.UnavailableReason = "destination_required"
		return snapshot, nil
	}
	if service.routingService == nil {
		return snapshot, nil
	}
	route, err := service.routingService.Route(ctx, geodomain.Coordinates{Latitude: pickup.Latitude, Longitude: pickup.Longitude}, geodomain.Coordinates{Latitude: destination.Latitude, Longitude: destination.Longitude})
	if err != nil {
		return nil, fmt.Errorf("estimate dispatcher road route: %w", err)
	}
	price, err := domain.CalculateTripPrice(tariff, route.DistanceMeters, route.DurationSeconds)
	if err != nil {
		return nil, fmt.Errorf("calculate dispatcher road price: %w", err)
	}
	snapshot.EstimatedPrice = &domain.Money{Amount: price, Currency: tariff.BasePrice.Currency}
	snapshot.EstimatedPriceSource = domain.EstimatedPriceSourceFixedTariff
	snapshot.PriceAvailable = true
	snapshot.UnavailableReason = ""
	snapshot.RouteDistanceMeters = &route.DistanceMeters
	snapshot.RouteDurationSeconds = &route.DurationSeconds
	snapshot.RouteSource = route.Source
	snapshot.RouteDataVersion = route.DataVersion
	snapshot.CalculatedAt = route.CalculatedAt
	return snapshot, nil
}

package dto

import "github.com/kishert-lab/taxi-platform/internal/domain"

// PricingSnapshotResponse keeps the same money and route fields for all clients.
func PricingSnapshotResponse(snapshot *domain.OrderPricingSnapshot) OrderPricingResponse {
	if snapshot == nil {
		return OrderPricingResponse{}
	}
	money := func(value *domain.Money) *MoneyResponse {
		if value == nil {
			return nil
		}
		return &MoneyResponse{Amount: value.Amount, Currency: value.Currency}
	}
	return OrderPricingResponse{EstimatedPrice: money(snapshot.EstimatedPrice), EstimatedPriceMin: money(snapshot.EstimatedPriceMin), EstimatedPriceMax: money(snapshot.EstimatedPriceMax), EstimatedPriceSource: string(snapshot.EstimatedPriceSource), PricingMode: string(snapshot.PricingMode), PriceAvailable: snapshot.PriceAvailable, IsFinal: snapshot.IsFinal, Message: snapshot.Message, UnavailableReason: snapshot.UnavailableReason, SearchRadiusMeters: snapshot.SearchRadiusMeters, RouteDistanceMeters: snapshot.RouteDistanceMeters, RouteDurationSeconds: snapshot.RouteDurationSeconds, RouteSource: snapshot.RouteSource, RouteDataVersion: snapshot.RouteDataVersion, TaxiParkCount: snapshot.TaxiParkCount}
}

package domain

import "testing"

func TestCalculateTripPriceUsesServerTripMetrics(t *testing.T) {
	tariff := TaxiParkTariff{
		PricingMode:    PricingModeDistanceTime,
		BasePrice:      Money{Amount: 10000, Currency: "RUB"},
		PricePerKM:     Money{Amount: 2500, Currency: "RUB"},
		PricePerMinute: Money{Amount: 500, Currency: "RUB"},
		MinimumPrice:   Money{Amount: 0, Currency: "RUB"},
	}

	price, err := CalculateTripPrice(tariff, 1250, 61)
	if err != nil {
		t.Fatalf("calculate trip price: %v", err)
	}
	if price != 14125 {
		t.Fatalf("expected 14125 cents, got %d", price)
	}
}

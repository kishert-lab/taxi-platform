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

func TestCalculateTripPriceAppliesMinimumAndRoundsUpStartedMinute(t *testing.T) {
	tariff := TaxiParkTariff{
		PricingMode: PricingModeDistanceTime,
		PricePerKM:  Money{Amount: 101}, PricePerMinute: Money{Amount: 50},
		MinimumPrice: Money{Amount: 300},
	}
	price, err := CalculateTripPrice(tariff, 1001, 61)
	if err != nil {
		t.Fatal(err)
	}
	if price != 300 {
		t.Fatalf("minimum must win, got %d", price)
	}
	tariff.MinimumPrice.Amount = 0
	price, err = CalculateTripPrice(tariff, 1001, 61)
	if err != nil || price != 201 {
		t.Fatalf("expected 101 distance + 100 for two started minutes, got %d (%v)", price, err)
	}
}

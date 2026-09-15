package domain

import "fmt"

// CalculateTripPrice calculates the final fare from server-collected trip metrics.
func CalculateTripPrice(tariff TaxiParkTariff, distanceMeters, durationSeconds int64) (int64, error) {
	if distanceMeters < 0 || durationSeconds < 0 {
		return 0, fmt.Errorf("trip metrics cannot be negative")
	}

	price := tariff.BasePrice.Amount
	switch tariff.PricingMode {
	case PricingModeFixed:
		price = tariff.FixedPrice.Amount
	case PricingModeDistance:
		price += roundedDistanceCharge(distanceMeters, tariff.PricePerKM.Amount)
	case PricingModeTime:
		price += roundedMinuteCharge(durationSeconds, tariff.PricePerMinute.Amount)
	case PricingModeDistanceTime:
		price += roundedDistanceCharge(distanceMeters, tariff.PricePerKM.Amount)
		price += roundedMinuteCharge(durationSeconds, tariff.PricePerMinute.Amount)
	default:
		return 0, fmt.Errorf("unsupported pricing mode %q", tariff.PricingMode)
	}
	if price < tariff.MinimumPrice.Amount {
		price = tariff.MinimumPrice.Amount
	}
	return price, nil
}

func roundedDistanceCharge(distanceMeters, pricePerKMCents int64) int64 {
	return (distanceMeters*pricePerKMCents + 500) / 1000
}

func roundedMinuteCharge(durationSeconds, pricePerMinuteCents int64) int64 {
	if durationSeconds == 0 {
		return 0
	}
	return ((durationSeconds + 59) / 60) * pricePerMinuteCents
}

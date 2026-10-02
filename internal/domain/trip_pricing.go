package domain

import (
	"fmt"
	"math/big"
)

// CalculateTripPrice calculates the final fare from server-collected trip metrics.
func CalculateTripPrice(tariff TaxiParkTariff, distanceMeters, durationSeconds int64) (int64, error) {
	if distanceMeters < 0 || durationSeconds < 0 {
		return 0, fmt.Errorf("trip metrics cannot be negative")
	}
	if tariff.BasePrice.Amount < 0 || tariff.FixedPrice.Amount < 0 || tariff.PricePerKM.Amount < 0 || tariff.PricePerMinute.Amount < 0 || tariff.MinimumPrice.Amount < 0 {
		return 0, fmt.Errorf("tariff rates cannot be negative")
	}

	switch tariff.PricingMode {
	case PricingModeFixed:
		if tariff.FixedPrice.Amount <= 0 {
			return 0, fmt.Errorf("fixed tariff has no fixed price")
		}
		price := tariff.FixedPrice.Amount
		if price < tariff.MinimumPrice.Amount {
			price = tariff.MinimumPrice.Amount
		}
		return price, nil
	case PricingModeDistance:
		if tariff.PricePerKM.Amount <= 0 {
			return 0, fmt.Errorf("distance tariff has no kilometer rate")
		}
		return calculateMeasuredPrice(tariff, distanceMeters, 0)
	case PricingModeTime:
		if tariff.PricePerMinute.Amount <= 0 {
			return 0, fmt.Errorf("time tariff has no minute rate")
		}
		return calculateMeasuredPrice(tariff, 0, durationSeconds)
	case PricingModeDistanceTime:
		if tariff.PricePerKM.Amount == 0 && tariff.PricePerMinute.Amount == 0 {
			return 0, fmt.Errorf("distance/time tariff has no variable rate")
		}
		return calculateMeasuredPrice(tariff, distanceMeters, durationSeconds)
	default:
		return 0, fmt.Errorf("unsupported pricing mode %q", tariff.PricingMode)
	}
}

func calculateMeasuredPrice(tariff TaxiParkTariff, distanceMeters, durationSeconds int64) (int64, error) {
	price := big.NewInt(tariff.BasePrice.Amount)
	distanceCharge := new(big.Int).Mul(big.NewInt(distanceMeters), big.NewInt(tariff.PricePerKM.Amount))
	distanceCharge.Add(distanceCharge, big.NewInt(500))
	distanceCharge.Div(distanceCharge, big.NewInt(1000))
	price.Add(price, distanceCharge)
	if durationSeconds > 0 {
		minutes := new(big.Int).Add(big.NewInt(durationSeconds), big.NewInt(59))
		minutes.Div(minutes, big.NewInt(60))
		price.Add(price, minutes.Mul(minutes, big.NewInt(tariff.PricePerMinute.Amount)))
	}
	if price.Cmp(big.NewInt(tariff.MinimumPrice.Amount)) < 0 {
		price.SetInt64(tariff.MinimumPrice.Amount)
	}
	if !price.IsInt64() {
		return 0, fmt.Errorf("tariff price exceeds int64 cents")
	}
	return price.Int64(), nil
}

package domain

import "time"

type EstimatedPriceSource string
type PricingMode string

const (
	EstimatedPriceSourceAverageParks    EstimatedPriceSource = "average_parks"
	EstimatedPriceSourceCarClassCatalog EstimatedPriceSource = "car_class_catalog"
	EstimatedPriceSourceFixedTariff     EstimatedPriceSource = "fixed_tariff"
	EstimatedPriceSourceUnavailable     EstimatedPriceSource = "unavailable"
)

const (
	PricingModeUnknown      PricingMode = "unknown"
	PricingModeFixed        PricingMode = "fixed"
	PricingModeDistance     PricingMode = "distance"
	PricingModeTime         PricingMode = "time"
	PricingModeDistanceTime PricingMode = "distance_time"
)

type OrderPricingSnapshot struct {
	EstimatedPrice       *Money               `json:"estimated_price,omitempty"`
	EstimatedPriceMin    *Money               `json:"estimated_price_min,omitempty"`
	EstimatedPriceMax    *Money               `json:"estimated_price_max,omitempty"`
	EstimatedPriceSource EstimatedPriceSource `json:"estimated_price_source,omitempty"`
	PricingMode          PricingMode          `json:"pricing_mode,omitempty"`
	PriceAvailable       bool                 `json:"price_available"`
	IsFinal              bool                 `json:"is_final"`
	Message              string               `json:"message,omitempty"`
	UnavailableReason    string               `json:"unavailable_reason,omitempty"`
	SearchRadiusMeters   *int                 `json:"search_radius_meters,omitempty"`
	RouteDistanceMeters  *int64               `json:"route_distance_meters,omitempty"`
	RouteDurationSeconds *int64               `json:"route_duration_seconds,omitempty"`
	RouteSource          string               `json:"route_source,omitempty"`
	RouteDataVersion     string               `json:"route_data_version,omitempty"`
	TariffRates          []TariffRateSnapshot `json:"tariff_rates,omitempty"`
	RoundingRule         string               `json:"rounding_rule,omitempty"`
	TaxiParkCount        int                  `json:"taxi_park_count,omitempty"`
	CalculatedAt         time.Time            `json:"calculated_at,omitempty"`
}

type TariffRateSnapshot struct {
	TariffID       string      `json:"tariff_id"`
	PricingMode    PricingMode `json:"pricing_mode"`
	BasePrice      Money       `json:"base_price"`
	FixedPrice     Money       `json:"fixed_price"`
	PricePerKM     Money       `json:"price_per_km"`
	PricePerMinute Money       `json:"price_per_minute"`
	MinimumPrice   Money       `json:"minimum_price"`
}

func SnapshotTariffRates(tariffs []TaxiParkTariff) []TariffRateSnapshot {
	result := make([]TariffRateSnapshot, 0, len(tariffs))
	for _, tariff := range tariffs {
		result = append(result, TariffRateSnapshot{tariff.ID.String(), tariff.PricingMode, tariff.BasePrice, tariff.FixedPrice, tariff.PricePerKM, tariff.PricePerMinute, tariff.MinimumPrice})
	}
	return result
}

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
	SearchRadiusMeters   *int                 `json:"search_radius_meters,omitempty"`
	CalculatedAt         time.Time            `json:"calculated_at,omitempty"`
}

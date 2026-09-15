package routing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
)

var (
	ErrUnavailable     = errors.New("routing service unavailable")
	ErrRouteNotFound   = errors.New("road route not found")
	ErrInvalidResponse = errors.New("invalid routing service response")
	ErrInvalidRequest  = errors.New("invalid route request")
	ErrOutsideCoverage = errors.New("route outside coverage or snapping radius")
)

// Route contains road metrics returned by a routing provider.
type Route struct {
	DistanceMeters  int64                   `json:"distance_meters"`
	DurationSeconds int64                   `json:"duration_seconds"`
	Geometry        json.RawMessage         `json:"geometry" swaggertype:"object"`
	Source          string                  `json:"source"`
	DataVersion     string                  `json:"data_version"`
	OriginalPoints  []geodomain.Coordinates `json:"original_points"`
	SnappedPoints   []geodomain.Coordinates `json:"snapped_points"`
	CalculatedAt    time.Time               `json:"calculated_at"`
}

type Service interface {
	Route(context.Context, geodomain.Coordinates, geodomain.Coordinates) (Route, error)
}

type UnavailableService struct{}

func (UnavailableService) Route(context.Context, geodomain.Coordinates, geodomain.Coordinates) (Route, error) {
	return Route{}, ErrUnavailable
}

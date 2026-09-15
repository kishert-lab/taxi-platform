package maps

import (
	"context"
	"fmt"
	"github.com/kishert-lab/taxi-platform/configs"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/routing"
	"net/url"
	"regexp"
	"strings"
)

type Configuration struct {
	StyleURL         string    `json:"style_url"`
	Styles           []Style   `json:"styles"`
	Bounds           []float64 `json:"bounds"`
	MinZoom          int       `json:"min_zoom"`
	MaxZoom          int       `json:"max_zoom"`
	DataVersion      string    `json:"data_version"`
	UpdatedAt        string    `json:"updated_at"`
	Attribution      string    `json:"attribution"`
	SearchAvailable  bool      `json:"search_available"`
	RoutingAvailable bool      `json:"routing_available"`
}
type Style struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}
type RouteRequest struct {
	Points []geodomain.Coordinates `json:"points" binding:"required,len=2"`
}
type Service struct {
	config           configs.MapsConfig
	routing          routing.Service
	searchAvailable  bool
	routingAvailable bool
	geocoder         ReverseGeocoder
}

func New(config configs.MapsConfig, provider routing.Service, searchAvailable, routingAvailable bool) *Service {
	return &Service{config: config, routing: provider, searchAvailable: searchAvailable, routingAvailable: routingAvailable}
}

type ReverseGeocoder interface {
	Reverse(context.Context, geodomain.Coordinates) ([]geodomain.SearchResult, error)
}
type ReverseResponse struct {
	Point           geodomain.Coordinates  `json:"point"`
	Found           bool                   `json:"found"`
	Address         string                 `json:"address"`
	AddressLocation *geodomain.Coordinates `json:"address_location,omitempty"`
	Source          string                 `json:"source"`
}

func (service *Service) WithReverseGeocoder(provider ReverseGeocoder) { service.geocoder = provider }
func (service *Service) Reverse(ctx context.Context, point geodomain.Coordinates) (ReverseResponse, error) {
	if _, err := geodomain.NewCoordinates(point.Latitude, point.Longitude); err != nil {
		return ReverseResponse{}, fmt.Errorf("%w: %w", routing.ErrInvalidRequest, err)
	}
	if service.geocoder == nil {
		return ReverseResponse{}, routing.ErrUnavailable
	}
	results, err := service.geocoder.Reverse(ctx, point)
	if err != nil {
		return ReverseResponse{}, fmt.Errorf("reverse geocode: %w", err)
	}
	result := ReverseResponse{Point: point, Source: "pelias"}
	if len(results) > 0 {
		result.Found = true
		result.Address = results[0].Address
		result.AddressLocation = &results[0].Coordinates
	}
	return result, nil
}
func (service *Service) Configuration(context.Context) (Configuration, error) {
	config := service.config
	endpoint, err := url.Parse(config.PublicURL)
	if err != nil || endpoint.Scheme != "https" || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" || !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(config.DataVersion) {
		return Configuration{}, fmt.Errorf("map release is not configured: %w", routing.ErrUnavailable)
	}
	base := strings.TrimRight(config.PublicURL, "/") + "/releases/" + config.DataVersion
	return Configuration{StyleURL: base + "/style.json", Styles: []Style{{ID: "day", URL: base + "/style.json"}}, Bounds: config.Bounds, MinZoom: config.MinZoom, MaxZoom: config.MaxZoom, DataVersion: config.DataVersion, UpdatedAt: config.UpdatedAt, Attribution: "© OpenStreetMap contributors (ODbL) https://www.openstreetmap.org/copyright", SearchAvailable: service.searchAvailable, RoutingAvailable: service.routingAvailable}, nil
}
func (service *Service) Route(ctx context.Context, request RouteRequest) (routing.Route, error) {
	if len(request.Points) != 2 {
		return routing.Route{}, routing.ErrInvalidRequest
	}
	return service.routing.Route(ctx, request.Points[0], request.Points[1])
}

package osrm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/routing"
)

type Options struct {
	DataVersion   string
	MaxSnapMeters float64
	Bounds        [4]float64 // west, south, east, north; west > east crosses the antimeridian
}

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	options    Options
}

func New(rawURL string, httpClient *http.Client) (*Client, error) {
	return NewWithOptions(rawURL, httpClient, Options{MaxSnapMeters: 500, Bounds: [4]float64{-180, -90, 180, 90}})
}

func NewWithOptions(rawURL string, httpClient *http.Client, options Options) (*Client, error) {
	endpoint, err := url.Parse(strings.TrimRight(strings.TrimSpace(rawURL), "/"))
	if err != nil {
		return nil, fmt.Errorf("parse OSRM URL: %w", err)
	}
	if (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.Fragment != "" {
		return nil, fmt.Errorf("OSRM URL must be an HTTP origin or base path")
	}
	if options.MaxSnapMeters <= 0 || math.IsNaN(options.MaxSnapMeters) || math.IsInf(options.MaxSnapMeters, 0) {
		return nil, fmt.Errorf("OSRM snapping radius must be finite and positive")
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Second}
	}
	transport := *httpClient
	if transport.Timeout <= 0 {
		transport.Timeout = 3 * time.Second
	}
	return &Client{baseURL: endpoint, httpClient: &transport, options: options}, nil
}

func (client *Client) Route(ctx context.Context, origin, destination geodomain.Coordinates) (routing.Route, error) {
	points := []geodomain.Coordinates{origin, destination}
	for _, point := range points {
		if _, err := geodomain.NewCoordinates(point.Latitude, point.Longitude); err != nil {
			return routing.Route{}, fmt.Errorf("%w: %w", routing.ErrInvalidRequest, err)
		}
		bounds := client.options.Bounds
		longitudeInside := point.Longitude >= bounds[0] && point.Longitude <= bounds[2]
		if bounds[0] > bounds[2] {
			longitudeInside = point.Longitude >= bounds[0] || point.Longitude <= bounds[2]
		}
		if !longitudeInside || point.Latitude < bounds[1] || point.Latitude > bounds[3] {
			return routing.Route{}, routing.ErrOutsideCoverage
		}
	}
	endpoint := *client.baseURL
	endpoint.Path += fmt.Sprintf("/route/v1/driving/%f,%f;%f,%f", origin.Longitude, origin.Latitude, destination.Longitude, destination.Latitude)
	query := endpoint.Query()
	query.Set("overview", "full")
	query.Set("geometries", "geojson")
	query.Set("steps", "false")
	query.Set("radiuses", fmt.Sprintf("%g;%g", client.options.MaxSnapMeters, client.options.MaxSnapMeters))
	endpoint.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return routing.Route{}, fmt.Errorf("create OSRM request: %w", err)
	}
	response, err := client.httpClient.Do(request)
	if err != nil {
		return routing.Route{}, fmt.Errorf("request OSRM route: %w: %w", routing.ErrUnavailable, err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, (2<<20)+1))
	if err != nil {
		return routing.Route{}, fmt.Errorf("read OSRM response: %w: %w", routing.ErrUnavailable, err)
	}
	if len(body) > 2<<20 {
		return routing.Route{}, routing.ErrInvalidResponse
	}
	var payload routeResponse
	decodeErr := json.Unmarshal(body, &payload)
	if response.StatusCode == 400 && decodeErr == nil && payload.Code == "NoSegment" {
		return routing.Route{}, routing.ErrOutsideCoverage
	}
	if response.StatusCode != http.StatusOK {
		return routing.Route{}, fmt.Errorf("OSRM HTTP %d: %w", response.StatusCode, routing.ErrUnavailable)
	}
	if decodeErr != nil {
		return routing.Route{}, fmt.Errorf("decode OSRM response: %w: %w", routing.ErrInvalidResponse, decodeErr)
	}
	if payload.Code == "NoRoute" {
		return routing.Route{}, routing.ErrRouteNotFound
	}
	if payload.Code == "NoSegment" {
		return routing.Route{}, routing.ErrOutsideCoverage
	}
	if payload.Code != "Ok" || len(payload.Routes) == 0 || len(payload.Waypoints) != 2 {
		return routing.Route{}, routing.ErrInvalidResponse
	}
	candidate := payload.Routes[0]
	if candidate.Distance == nil || candidate.Duration == nil || !validMetric(*candidate.Distance) || !validMetric(*candidate.Duration) || candidate.Geometry.Type != "LineString" || len(candidate.Geometry.Coordinates) < 2 {
		return routing.Route{}, routing.ErrInvalidResponse
	}
	for _, pair := range candidate.Geometry.Coordinates {
		if !validPair(pair) {
			return routing.Route{}, routing.ErrInvalidResponse
		}
	}
	snapped := make([]geodomain.Coordinates, 0, 2)
	for _, point := range payload.Waypoints {
		if !validPair(point.Location) || point.Distance == nil || !validMetric(*point.Distance) {
			return routing.Route{}, routing.ErrInvalidResponse
		}
		if *point.Distance > client.options.MaxSnapMeters {
			return routing.Route{}, routing.ErrOutsideCoverage
		}
		snapped = append(snapped, geodomain.Coordinates{Longitude: point.Location[0], Latitude: point.Location[1]})
	}
	geometry, err := json.Marshal(candidate.Geometry)
	if err != nil {
		return routing.Route{}, fmt.Errorf("encode OSRM geometry: %w", err)
	}
	return routing.Route{DistanceMeters: int64(math.Round(*candidate.Distance)), DurationSeconds: int64(math.Round(*candidate.Duration)), Geometry: geometry, Source: "osrm", DataVersion: client.options.DataVersion, OriginalPoints: points, SnappedPoints: snapped, CalculatedAt: time.Now().UTC()}, nil
}

func validMetric(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value < 1e12
}
func validPair(pair []float64) bool {
	if len(pair) != 2 {
		return false
	}
	_, err := geodomain.NewCoordinates(pair[1], pair[0])
	return err == nil
}

type routeResponse struct {
	Code   string `json:"code"`
	Routes []struct {
		Distance *float64 `json:"distance"`
		Duration *float64 `json:"duration"`
		Geometry struct {
			Type        string      `json:"type"`
			Coordinates [][]float64 `json:"coordinates"`
		} `json:"geometry"`
	} `json:"routes"`
	Waypoints []struct {
		Location []float64 `json:"location"`
		Distance *float64  `json:"distance"`
	} `json:"waypoints"`
}

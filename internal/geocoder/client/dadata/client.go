// Package dadata contains DaData address geocoder clients.
package dadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
)

const (
	defaultCleanEndpoint   = "https://cleaner.dadata.ru/api/v1/clean/address"
	defaultSuggestEndpoint = "https://suggestions.dadata.ru/suggestions/api/4_1/rs/suggest/address"
	maxFocusDistanceMeters = 250000.0
)

type Client struct {
	apiKey     string
	secretKey  string
	cleanURL   string
	suggestURL string
	httpClient *http.Client
}

func New(apiKey string, secretKey string, cleanURL string, suggestURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 4 * time.Second}
	}
	if strings.TrimSpace(cleanURL) == "" {
		cleanURL = defaultCleanEndpoint
	}
	if strings.TrimSpace(suggestURL) == "" {
		suggestURL = defaultSuggestEndpoint
	}
	return &Client{
		apiKey:     strings.TrimSpace(apiKey),
		secretKey:  strings.TrimSpace(secretKey),
		cleanURL:   strings.TrimSpace(cleanURL),
		suggestURL: strings.TrimSpace(suggestURL),
		httpClient: httpClient,
	}
}

func (client *Client) Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	if client.apiKey == "" {
		return nil, nil
	}
	query := strings.TrimSpace(request.Query)
	if query == "" {
		return nil, geodomain.ErrInvalidQuery
	}

	cleanResults, cleanErr := client.clean(ctx, query)
	if cleanErr == nil && len(cleanResults) > 0 {
		return prioritizeByFocus(cleanResults, request.Focus), nil
	}
	suggestResults, suggestErr := client.suggest(ctx, request, query)
	if suggestErr == nil {
		return prioritizeByFocus(suggestResults, request.Focus), nil
	}
	if cleanErr != nil {
		return nil, fmt.Errorf("dadata clean failed: %w; suggest failed: %w", cleanErr, suggestErr)
	}
	return nil, suggestErr
}

func (client *Client) clean(ctx context.Context, query string) ([]geodomain.SearchResult, error) {
	if client.secretKey == "" {
		return nil, nil
	}
	body, err := json.Marshal([]string{query})
	if err != nil {
		return nil, fmt.Errorf("encode dadata request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.cleanURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create dadata request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Authorization", "Token "+client.apiKey)
	httpRequest.Header.Set("X-Secret", client.secretKey)

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("execute dadata request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("dadata returned status %d", response.StatusCode)
	}

	var payload []cleanAddressResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode dadata response: %w", err)
	}
	results := make([]geodomain.SearchResult, 0, len(payload))
	for _, address := range payload {
		if !hasCoordinates(address.GeoLat, address.GeoLon) {
			continue
		}
		coordinates, ok := parseCoordinates(address.GeoLat, address.GeoLon)
		if !ok {
			continue
		}
		results = append(results, geodomain.SearchResult{
			ID:              "dadata:" + query,
			Provider:        geodomain.ProviderDaData,
			Name:            firstNonBlank(streetWithHouse(address.Street, address.House), address.Result, query),
			Address:         firstNonBlank(address.Result, query),
			Coordinates:     coordinates,
			Confidence:      dadataCleanConfidence(address.QCGeo),
			ExternalPlaceID: firstNonBlank(address.FIASID, address.HouseFIASID),
		})
	}
	return results, nil
}

func (client *Client) suggest(ctx context.Context, request geodomain.SearchRequest, query string) ([]geodomain.SearchResult, error) {
	count := request.Limit
	if count <= 0 {
		count = 10
	}
	if request.Focus != nil && count < 20 {
		count = 20
	}
	body, err := json.Marshal(map[string]any{
		"query": query,
		"count": count,
	})
	if err != nil {
		return nil, fmt.Errorf("encode dadata suggest request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.suggestURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create dadata suggest request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	httpRequest.Header.Set("Authorization", "Token "+client.apiKey)

	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("execute dadata suggest request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("dadata suggest returned status %d", response.StatusCode)
	}

	var payload suggestResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode dadata suggest response: %w", err)
	}
	results := make([]geodomain.SearchResult, 0, len(payload.Suggestions))
	for _, suggestion := range payload.Suggestions {
		if !hasCoordinates(suggestion.Data.GeoLat, suggestion.Data.GeoLon) {
			continue
		}
		coordinates, ok := parseCoordinates(suggestion.Data.GeoLat, suggestion.Data.GeoLon)
		if !ok {
			continue
		}
		results = append(results, geodomain.SearchResult{
			ID:              "dadata:" + firstNonBlank(suggestion.Data.FIASID, suggestion.Value, query),
			Provider:        geodomain.ProviderDaData,
			Name:            firstNonBlank(streetWithHouse(suggestion.Data.Street, suggestion.Data.House), suggestion.Value, query),
			Address:         firstNonBlank(suggestion.UnrestrictedValue, suggestion.Value, query),
			Coordinates:     coordinates,
			Confidence:      dadataSuggestConfidence(suggestion.Data),
			ExternalPlaceID: firstNonBlank(suggestion.Data.FIASID, suggestion.Data.HouseFIASID),
		})
	}
	return results, nil
}

func prioritizeByFocus(results []geodomain.SearchResult, focus *geodomain.Coordinates) []geodomain.SearchResult {
	if focus == nil || len(results) == 0 {
		return results
	}

	type scoredResult struct {
		result   geodomain.SearchResult
		distance float64
	}
	scored := make([]scoredResult, 0, len(results))
	for _, result := range results {
		scored = append(scored, scoredResult{
			result:   result,
			distance: distanceMeters(*focus, result.Coordinates),
		})
	}

	sort.SliceStable(scored, func(left int, right int) bool {
		return scored[left].distance < scored[right].distance
	})

	filtered := make([]geodomain.SearchResult, 0, len(scored))
	for _, item := range scored {
		if item.distance <= maxFocusDistanceMeters {
			filtered = append(filtered, item.result)
		}
	}
	if len(filtered) > 0 {
		return filtered
	}

	ordered := make([]geodomain.SearchResult, 0, len(scored))
	for _, item := range scored {
		ordered = append(ordered, item.result)
	}
	return ordered
}

func distanceMeters(from geodomain.Coordinates, to geodomain.Coordinates) float64 {
	const earthRadiusMeters = 6371000.0

	latitudeFrom := from.Latitude * math.Pi / 180
	latitudeTo := to.Latitude * math.Pi / 180
	deltaLatitude := (to.Latitude - from.Latitude) * math.Pi / 180
	deltaLongitude := (to.Longitude - from.Longitude) * math.Pi / 180

	sinLatitude := math.Sin(deltaLatitude / 2)
	sinLongitude := math.Sin(deltaLongitude / 2)
	a := sinLatitude*sinLatitude + math.Cos(latitudeFrom)*math.Cos(latitudeTo)*sinLongitude*sinLongitude
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

type cleanAddressResponse struct {
	Result      string `json:"result"`
	Street      string `json:"street"`
	House       string `json:"house"`
	GeoLat      string `json:"geo_lat"`
	GeoLon      string `json:"geo_lon"`
	QCGeo       int    `json:"qc_geo"`
	FIASID      string `json:"fias_id"`
	HouseFIASID string `json:"house_fias_id"`
}

type suggestResponse struct {
	Suggestions []suggestion `json:"suggestions"`
}

type suggestion struct {
	Value             string            `json:"value"`
	UnrestrictedValue string            `json:"unrestricted_value"`
	Data              suggestionAddress `json:"data"`
}

type suggestionAddress struct {
	Street      string `json:"street"`
	House       string `json:"house"`
	GeoLat      string `json:"geo_lat"`
	GeoLon      string `json:"geo_lon"`
	FIASID      string `json:"fias_id"`
	HouseFIASID string `json:"house_fias_id"`
}

func hasCoordinates(latitude string, longitude string) bool {
	return strings.TrimSpace(latitude) != "" && strings.TrimSpace(longitude) != ""
}

func parseCoordinates(latitudeValue string, longitudeValue string) (geodomain.Coordinates, bool) {
	latitude, err := strconv.ParseFloat(latitudeValue, 64)
	if err != nil {
		return geodomain.Coordinates{}, false
	}
	longitude, err := strconv.ParseFloat(longitudeValue, 64)
	if err != nil {
		return geodomain.Coordinates{}, false
	}
	coordinates, err := geodomain.NewCoordinates(latitude, longitude)
	return coordinates, err == nil
}

func streetWithHouse(streetValue string, houseValue string) string {
	street := strings.TrimSpace(streetValue)
	house := strings.TrimSpace(houseValue)
	if street == "" {
		return house
	}
	if house == "" {
		return street
	}
	return street + ", " + house
}

func dadataCleanConfidence(qcGeo int) float64 {
	switch qcGeo {
	case 0:
		return 0.95
	case 1:
		return 0.90
	case 2:
		return 0.75
	case 3:
		return 0.60
	default:
		return 0.50
	}
}

func dadataSuggestConfidence(address suggestionAddress) float64 {
	if strings.TrimSpace(address.House) != "" {
		return 0.88
	}
	if strings.TrimSpace(address.Street) != "" {
		return 0.75
	}
	return 0.60
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

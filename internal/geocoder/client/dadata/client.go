// Package dadata contains the DaData clean address geocoder client.
package dadata

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
)

const defaultEndpoint = "https://cleaner.dadata.ru/api/v1/clean/address"

type Client struct {
	apiKey     string
	secretKey  string
	endpoint   string
	httpClient *http.Client
}

func New(apiKey string, secretKey string, endpoint string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 4 * time.Second}
	}
	if strings.TrimSpace(endpoint) == "" {
		endpoint = defaultEndpoint
	}
	return &Client{
		apiKey:     strings.TrimSpace(apiKey),
		secretKey:  strings.TrimSpace(secretKey),
		endpoint:   strings.TrimSpace(endpoint),
		httpClient: httpClient,
	}
}

func (client *Client) Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	if client.apiKey == "" || client.secretKey == "" {
		return nil, nil
	}
	query := strings.TrimSpace(request.Query)
	if query == "" {
		return nil, geodomain.ErrInvalidQuery
	}
	body, err := json.Marshal([]string{query})
	if err != nil {
		return nil, fmt.Errorf("encode dadata request: %w", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, client.endpoint, bytes.NewReader(body))
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
		if !address.hasCoordinates() {
			continue
		}
		latitude, err := strconv.ParseFloat(address.GeoLat, 64)
		if err != nil {
			continue
		}
		longitude, err := strconv.ParseFloat(address.GeoLon, 64)
		if err != nil {
			continue
		}
		coordinates, err := geodomain.NewCoordinates(latitude, longitude)
		if err != nil {
			continue
		}
		results = append(results, geodomain.SearchResult{
			ID:              "dadata:" + query,
			Provider:        geodomain.ProviderDaData,
			Name:            firstNonBlank(address.StreetWithHouse(), address.Result, query),
			Address:         firstNonBlank(address.Result, query),
			Coordinates:     coordinates,
			Confidence:      dadataConfidence(address.QCGeo),
			ExternalPlaceID: firstNonBlank(address.FIASID, address.HouseFIASID),
		})
	}
	return results, nil
}

type cleanAddressResponse struct {
	Source      string `json:"source"`
	Result      string `json:"result"`
	Street      string `json:"street"`
	House       string `json:"house"`
	GeoLat      string `json:"geo_lat"`
	GeoLon      string `json:"geo_lon"`
	QCGeo       int    `json:"qc_geo"`
	FIASID      string `json:"fias_id"`
	HouseFIASID string `json:"house_fias_id"`
}

func (response cleanAddressResponse) hasCoordinates() bool {
	return strings.TrimSpace(response.GeoLat) != "" && strings.TrimSpace(response.GeoLon) != ""
}

func (response cleanAddressResponse) StreetWithHouse() string {
	street := strings.TrimSpace(response.Street)
	house := strings.TrimSpace(response.House)
	if street == "" {
		return house
	}
	if house == "" {
		return street
	}
	return street + ", " + house
}

func dadataConfidence(qcGeo int) float64 {
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

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

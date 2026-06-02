// Package pelias contains the local Pelias geocoder HTTP client.
package pelias

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func New(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 3 * time.Second}
	}
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

func (client *Client) Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	if client.baseURL == "" {
		return nil, nil
	}
	endpoint, err := url.Parse(client.baseURL + "/v1/search")
	if err != nil {
		return nil, fmt.Errorf("parse pelias url: %w", err)
	}
	query := endpoint.Query()
	query.Set("text", request.Query)
	if request.Limit > 0 {
		query.Set("size", strconv.Itoa(request.Limit))
	}
	if request.Focus != nil {
		query.Set("focus.point.lat", strconv.FormatFloat(request.Focus.Latitude, 'f', -1, 64))
		query.Set("focus.point.lon", strconv.FormatFloat(request.Focus.Longitude, 'f', -1, 64))
	}
	endpoint.RawQuery = query.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create pelias request: %w", err)
	}
	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("execute pelias request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("pelias returned status %d", response.StatusCode)
	}

	var payload searchResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode pelias response: %w", err)
	}
	results := make([]geodomain.SearchResult, 0, len(payload.Features))
	for _, feature := range payload.Features {
		if len(feature.Geometry.Coordinates) < 2 {
			continue
		}
		coordinates, err := geodomain.NewCoordinates(feature.Geometry.Coordinates[1], feature.Geometry.Coordinates[0])
		if err != nil {
			continue
		}
		confidence := feature.Properties.Confidence
		if confidence == 0 {
			confidence = feature.Properties.MatchConfidence
		}
		results = append(results, geodomain.SearchResult{
			ID:              firstNonBlank(feature.Properties.ID, feature.Properties.GID),
			Provider:        geodomain.ProviderPelias,
			Name:            firstNonBlank(feature.Properties.Name, feature.Properties.Label),
			Address:         firstNonBlank(feature.Properties.Label, feature.Properties.Name),
			Coordinates:     coordinates,
			Confidence:      confidence,
			ExternalPlaceID: firstNonBlank(feature.Properties.GID, feature.Properties.ID),
		})
	}
	return results, nil
}

type searchResponse struct {
	Features []feature `json:"features"`
}

type feature struct {
	Geometry   geometry   `json:"geometry"`
	Properties properties `json:"properties"`
}

type geometry struct {
	Coordinates []float64 `json:"coordinates"`
}

type properties struct {
	ID              string  `json:"id"`
	GID             string  `json:"gid"`
	Name            string  `json:"name"`
	Label           string  `json:"label"`
	Confidence      float64 `json:"confidence"`
	MatchConfidence float64 `json:"match_confidence"`
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

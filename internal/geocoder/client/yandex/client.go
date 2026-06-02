// Package yandex contains a limited Yandex Geocoder client used only as a temporary fallback.
package yandex

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
	apiKey     string
	endpoint   string
	httpClient *http.Client
}

func New(apiKey string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 4 * time.Second}
	}
	return &Client{
		apiKey:     strings.TrimSpace(apiKey),
		endpoint:   "https://geocode-maps.yandex.ru/1.x/",
		httpClient: httpClient,
	}
}

func (client *Client) Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	if client.apiKey == "" {
		return nil, nil
	}
	endpoint, err := url.Parse(client.endpoint)
	if err != nil {
		return nil, fmt.Errorf("parse yandex url: %w", err)
	}
	query := endpoint.Query()
	query.Set("apikey", client.apiKey)
	query.Set("format", "json")
	query.Set("geocode", request.Query)
	if request.Limit > 0 {
		query.Set("results", strconv.Itoa(request.Limit))
	}
	if request.Focus != nil {
		query.Set("ll", strconv.FormatFloat(request.Focus.Longitude, 'f', -1, 64)+","+strconv.FormatFloat(request.Focus.Latitude, 'f', -1, 64))
		query.Set("spn", "0.2,0.2")
	}
	endpoint.RawQuery = query.Encode()

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create yandex request: %w", err)
	}
	response, err := client.httpClient.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("execute yandex request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("yandex returned status %d", response.StatusCode)
	}

	var payload yandexResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode yandex response: %w", err)
	}
	members := payload.Response.GeoObjectCollection.FeatureMember
	results := make([]geodomain.SearchResult, 0, len(members))
	for _, member := range members {
		coordinates, ok := parseYandexPosition(member.GeoObject.Point.Pos)
		if !ok {
			continue
		}
		results = append(results, geodomain.SearchResult{
			ID:              member.GeoObject.URI,
			Provider:        geodomain.ProviderYandex,
			Name:            firstNonBlank(member.GeoObject.Name, member.GeoObject.MetaDataProperty.GeocoderMetaData.Text),
			Address:         firstNonBlank(member.GeoObject.MetaDataProperty.GeocoderMetaData.Text, member.GeoObject.Description),
			Coordinates:     coordinates,
			Confidence:      yandexPrecisionConfidence(member.GeoObject.MetaDataProperty.GeocoderMetaData.Precision),
			ExternalPlaceID: member.GeoObject.URI,
		})
	}
	return results, nil
}

func parseYandexPosition(position string) (geodomain.Coordinates, bool) {
	parts := strings.Fields(position)
	if len(parts) != 2 {
		return geodomain.Coordinates{}, false
	}
	longitude, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return geodomain.Coordinates{}, false
	}
	latitude, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return geodomain.Coordinates{}, false
	}
	coordinates, err := geodomain.NewCoordinates(latitude, longitude)
	return coordinates, err == nil
}

func yandexPrecisionConfidence(precision string) float64 {
	switch precision {
	case "exact":
		return 0.90
	case "number":
		return 0.85
	case "near":
		return 0.70
	case "range":
		return 0.65
	case "street":
		return 0.60
	default:
		return 0.50
	}
}

type yandexResponse struct {
	Response struct {
		GeoObjectCollection struct {
			FeatureMember []struct {
				GeoObject struct {
					URI              string `json:"uri"`
					Name             string `json:"name"`
					Description      string `json:"description"`
					MetaDataProperty struct {
						GeocoderMetaData struct {
							Text      string `json:"text"`
							Precision string `json:"precision"`
						} `json:"GeocoderMetaData"`
					} `json:"metaDataProperty"`
					Point struct {
						Pos string `json:"pos"`
					} `json:"Point"`
				} `json:"GeoObject"`
			} `json:"featureMember"`
		} `json:"GeoObjectCollection"`
	} `json:"response"`
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

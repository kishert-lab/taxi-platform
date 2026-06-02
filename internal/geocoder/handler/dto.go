package handler

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type CoordinatesRequest struct {
	Latitude  float64 `json:"latitude" binding:"required" example:"58.010455"`
	Longitude float64 `json:"longitude" binding:"required" example:"56.229443"`
}

type CoordinatesResponse struct {
	Latitude  float64 `json:"latitude" example:"58.010455"`
	Longitude float64 `json:"longitude" example:"56.229443"`
}

type SearchResultResponse struct {
	ID              string               `json:"id" example:"pelias:address:123"`
	LocalPointID    *uuid.UUID           `json:"local_point_id,omitempty" example:"11111111-1111-1111-1111-111111111111"`
	Provider        geodomain.Provider   `json:"provider" example:"pelias"`
	Name            string               `json:"name" example:"Мира 8"`
	Address         string               `json:"address" example:"Пермь, улица Мира, 8"`
	CityID          *uuid.UUID           `json:"city_id,omitempty" example:"22222222-2222-2222-2222-222222222222"`
	Coordinates     CoordinatesResponse  `json:"coordinates"`
	Confidence      float64              `json:"confidence" example:"0.91"`
	TrustLevel      geodomain.TrustLevel `json:"trust_level,omitempty" example:"trusted"`
	ExternalPlaceID string               `json:"external_place_id,omitempty" example:"yandex:123"`
	ExpiresAt       *time.Time           `json:"expires_at,omitempty" example:"2026-06-30T12:00:00Z"`
}

type SearchResponse struct {
	Results []SearchResultResponse `json:"results"`
}

type ConfirmPointRequest struct {
	CityID           uuid.UUID             `json:"city_id" binding:"required" example:"22222222-2222-2222-2222-222222222222"`
	Name             string                `json:"name,omitempty" example:"Мира 8"`
	Address          string                `json:"address" binding:"required" example:"Пермь, улица Мира, 8"`
	Coordinates      CoordinatesRequest    `json:"coordinates" binding:"required"`
	Source           geodomain.PointSource `json:"source,omitempty" example:"dispatcher_confirmed"`
	ExternalProvider string                `json:"external_provider,omitempty" example:"yandex"`
	ExternalPlaceID  string                `json:"external_place_id,omitempty" example:"yandex:123"`
	Confidence       float64               `json:"confidence,omitempty" example:"0.9"`
	Comment          string                `json:"comment,omitempty" example:"Confirmed by dispatcher"`
}

type AdminLocalPointRequest struct {
	CityID      uuid.UUID            `json:"city_id" binding:"required" example:"22222222-2222-2222-2222-222222222222"`
	Name        string               `json:"name" binding:"required" example:"Мира 8"`
	Address     string               `json:"address" binding:"required" example:"Пермь, улица Мира, 8"`
	Coordinates CoordinatesRequest   `json:"coordinates" binding:"required"`
	TrustLevel  geodomain.TrustLevel `json:"trust_level,omitempty" example:"trusted"`
}

type LocalPointResponse struct {
	ID                uuid.UUID             `json:"id" example:"11111111-1111-1111-1111-111111111111"`
	CityID            uuid.UUID             `json:"city_id" example:"22222222-2222-2222-2222-222222222222"`
	Name              string                `json:"name" example:"Мира 8"`
	NormalizedName    string                `json:"normalized_name" example:"мира 8"`
	Address           string                `json:"address" example:"Пермь, улица Мира, 8"`
	Coordinates       CoordinatesResponse   `json:"coordinates"`
	Source            geodomain.PointSource `json:"source" example:"dispatcher_confirmed"`
	ExternalProvider  string                `json:"external_provider,omitempty" example:"yandex"`
	ExternalPlaceID   string                `json:"external_place_id,omitempty" example:"yandex:123"`
	Confidence        float64               `json:"confidence" example:"1"`
	TrustLevel        geodomain.TrustLevel  `json:"trust_level" example:"trusted"`
	ConfirmationCount int                   `json:"confirmation_count" example:"3"`
	RejectCount       int                   `json:"reject_count" example:"0"`
	CreatedAt         time.Time             `json:"created_at" example:"2026-06-01T12:00:00Z"`
	UpdatedAt         time.Time             `json:"updated_at" example:"2026-06-01T12:00:00Z"`
}

type LocalPointsResponse struct {
	Points []LocalPointResponse `json:"points"`
}

type SearchSuccessResponse struct {
	Data SearchResponse `json:"data"`
	Meta response.Meta  `json:"meta"`
}

type LocalPointSuccessResponse struct {
	Data LocalPointResponse `json:"data"`
	Meta response.Meta      `json:"meta"`
}

type LocalPointsSuccessResponse struct {
	Data LocalPointsResponse `json:"data"`
	Meta response.Meta       `json:"meta"`
}

func (request ConfirmPointRequest) toService(context *gin.Context) (geoservice.ConfirmPointRequest, error) {
	coordinates, err := geodomain.NewCoordinates(request.Coordinates.Latitude, request.Coordinates.Longitude)
	if err != nil {
		return geoservice.ConfirmPointRequest{}, err
	}
	userID, _ := userIDFromContext(context)
	return geoservice.ConfirmPointRequest{
		CityID:           request.CityID,
		Name:             request.Name,
		Address:          request.Address,
		Coordinates:      coordinates,
		Source:           request.Source,
		ExternalProvider: request.ExternalProvider,
		ExternalPlaceID:  request.ExternalPlaceID,
		Confidence:       request.Confidence,
		UserID:           userID,
		ActorRole:        context.GetString("user_role"),
		Comment:          request.Comment,
		IP:               context.ClientIP(),
		UserAgent:        context.Request.UserAgent(),
	}, nil
}

func (request AdminLocalPointRequest) toService() (geoservice.AdminLocalPointRequest, error) {
	coordinates, err := geodomain.NewCoordinates(request.Coordinates.Latitude, request.Coordinates.Longitude)
	if err != nil {
		return geoservice.AdminLocalPointRequest{}, err
	}
	return geoservice.AdminLocalPointRequest{
		CityID:      request.CityID,
		Name:        request.Name,
		Address:     request.Address,
		Coordinates: coordinates,
		TrustLevel:  request.TrustLevel,
	}, nil
}

func searchResultsToResponse(results []geodomain.SearchResult) []SearchResultResponse {
	responseItems := make([]SearchResultResponse, 0, len(results))
	for _, result := range results {
		responseItems = append(responseItems, SearchResultResponse{
			ID:              result.ID,
			LocalPointID:    result.LocalPointID,
			Provider:        result.Provider,
			Name:            result.Name,
			Address:         result.Address,
			CityID:          result.CityID,
			Coordinates:     CoordinatesResponse{Latitude: result.Coordinates.Latitude, Longitude: result.Coordinates.Longitude},
			Confidence:      result.Confidence,
			TrustLevel:      result.TrustLevel,
			ExternalPlaceID: result.ExternalPlaceID,
			ExpiresAt:       result.ExpiresAt,
		})
	}
	return responseItems
}

func localPointsToResponse(points []geodomain.LocalGeoPoint) []LocalPointResponse {
	items := make([]LocalPointResponse, 0, len(points))
	for _, point := range points {
		items = append(items, localPointToResponse(point))
	}
	return items
}

func localPointToResponse(point geodomain.LocalGeoPoint) LocalPointResponse {
	return LocalPointResponse{
		ID:                point.ID,
		CityID:            point.CityID,
		Name:              point.Name,
		NormalizedName:    point.NormalizedName,
		Address:           point.Address,
		Coordinates:       CoordinatesResponse{Latitude: point.Coordinates.Latitude, Longitude: point.Coordinates.Longitude},
		Source:            point.Source,
		ExternalProvider:  point.ExternalProvider,
		ExternalPlaceID:   point.ExternalPlaceID,
		Confidence:        point.Confidence,
		TrustLevel:        point.TrustLevel,
		ConfirmationCount: point.ConfirmationCount,
		RejectCount:       point.RejectCount,
		CreatedAt:         point.CreatedAt,
		UpdatedAt:         point.UpdatedAt,
	}
}

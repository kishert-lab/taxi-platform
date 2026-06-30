// Package service implements hybrid geocoder application use cases.
package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
)

type Repository interface {
	ResolveActorCity(ctx context.Context, actorUserID uuid.UUID, actorRole string) (CityContext, bool, error)
	ResolveCity(ctx context.Context, cityID uuid.UUID) (CityContext, bool, error)
	ResolveCityByCoordinates(ctx context.Context, coordinates geodomain.Coordinates) (CityContext, bool, error)
	SearchLocalPoints(ctx context.Context, request LocalSearchRequest) ([]geodomain.SearchResult, error)
	GetExternalCache(ctx context.Context, provider geodomain.Provider, normalizedQuery string, cityID *uuid.UUID, now time.Time) ([]geodomain.SearchResult, bool, error)
	SaveExternalCache(ctx context.Context, cache ExternalCacheRecord) error
	ConfirmPoint(ctx context.Context, request ConfirmPointRequest) (geodomain.LocalGeoPoint, error)
	CreateLocalPoint(ctx context.Context, request AdminLocalPointRequest) (geodomain.LocalGeoPoint, error)
	ListLocalPoints(ctx context.Context, filter LocalPointFilter) ([]geodomain.LocalGeoPoint, error)
	ApproveLocalPoint(ctx context.Context, id uuid.UUID, adminUserID *uuid.UUID) (geodomain.LocalGeoPoint, error)
	RejectLocalPoint(ctx context.Context, id uuid.UUID, adminUserID *uuid.UUID) (geodomain.LocalGeoPoint, error)
	ExportTrustedLocalPoints(ctx context.Context) ([]geodomain.LocalGeoPoint, error)
}

type CityContext struct {
	CityID uuid.UUID
	Name   string
	Center geodomain.Coordinates
}

type PeliasClient interface {
	Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error)
}

type YandexClient interface {
	Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error)
}

type DaDataClient interface {
	Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error)
}

type LocalSearchRequest struct {
	NormalizedQuery string
	CityID          *uuid.UUID
	Focus           *geodomain.Coordinates
	Limit           int
}

type ExternalCacheRecord struct {
	Provider        geodomain.Provider
	NormalizedQuery string
	CityID          *uuid.UUID
	RequestParams   []byte
	Response        []byte
	Results         []geodomain.SearchResult
	ExpiresAt       time.Time
}

type ConfirmPointRequest struct {
	CityID           uuid.UUID
	Name             string
	Address          string
	Coordinates      geodomain.Coordinates
	Source           geodomain.PointSource
	ExternalProvider string
	ExternalPlaceID  string
	Confidence       float64
	UserID           *uuid.UUID
	ActorRole        string
	Comment          string
	IP               string
	UserAgent        string
}

type AdminLocalPointRequest struct {
	CityID      uuid.UUID
	Name        string
	Address     string
	Coordinates geodomain.Coordinates
	TrustLevel  geodomain.TrustLevel
}

type LocalPointFilter struct {
	CityID     *uuid.UUID
	TrustLevel geodomain.TrustLevel
	Limit      int
}

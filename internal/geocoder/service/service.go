package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
)

type Config struct {
	YandexEnabled             bool
	DaDataEnabled             bool
	ExternalCacheTTL          time.Duration
	PeliasConfidenceThreshold float64
	DefaultLimit              int
}

type Service struct {
	repository   Repository
	peliasClient PeliasClient
	yandexClient YandexClient
	dadataClient DaDataClient
	logger       *zap.Logger
	config       Config
}

func New(repository Repository, peliasClient PeliasClient, yandexClient YandexClient, dadataClient DaDataClient, logger *zap.Logger, config Config) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	if config.ExternalCacheTTL <= 0 {
		config.ExternalCacheTTL = 30 * 24 * time.Hour
	}
	if config.ExternalCacheTTL > 30*24*time.Hour {
		config.ExternalCacheTTL = 30 * 24 * time.Hour
	}
	if config.PeliasConfidenceThreshold <= 0 {
		config.PeliasConfidenceThreshold = 0.75
	}
	if config.DefaultLimit <= 0 {
		config.DefaultLimit = 10
	}
	return &Service{
		repository:   repository,
		peliasClient: peliasClient,
		yandexClient: yandexClient,
		dadataClient: dadataClient,
		logger:       logger,
		config:       config,
	}
}

func (service *Service) Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	normalizedQuery := geodomain.NormalizeQuery(request.Query)
	if normalizedQuery == "" {
		return nil, geodomain.ErrInvalidQuery
	}
	if request.Limit <= 0 {
		request.Limit = service.config.DefaultLimit
	}
	if request.RequestedAt.IsZero() {
		request.RequestedAt = time.Now().UTC()
	}
	cityName := ""
	if request.CityID == nil && request.ActorUserID != nil {
		cityContext, found, err := service.repository.ResolveActorCity(ctx, *request.ActorUserID, request.ActorRole)
		if err != nil {
			return nil, fmt.Errorf("resolve actor geocoder city: %w", err)
		}
		if found {
			request.CityID = &cityContext.CityID
			cityName = cityContext.Name
			if request.Focus == nil {
				request.Focus = &cityContext.Center
			}
		}
	}
	if request.CityID != nil && (request.Focus == nil || cityName == "") {
		cityContext, found, err := service.repository.ResolveCity(ctx, *request.CityID)
		if err != nil {
			return nil, fmt.Errorf("resolve geocoder city: %w", err)
		}
		if found {
			cityName = cityContext.Name
			if request.Focus == nil {
				request.Focus = &cityContext.Center
			}
		}
	}

	localResults, err := service.repository.SearchLocalPoints(ctx, LocalSearchRequest{
		NormalizedQuery: normalizedQuery,
		CityID:          request.CityID,
		Focus:           request.Focus,
		Limit:           request.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("search local geo points: %w", err)
	}
	if len(localResults) > 0 {
		service.logger.Info("geocoder local hit", zap.String("query", normalizedQuery), zap.Int("results", len(localResults)))
		return limitResults(localResults, request.Limit), nil
	}

	externalRequest := request
	if cityName != "" && !strings.Contains(strings.ToLower(externalRequest.Query), strings.ToLower(cityName)) {
		externalRequest.Query = strings.TrimSpace(cityName + ", " + externalRequest.Query)
	}

	peliasResults, err := service.peliasClient.Search(ctx, externalRequest)
	if err != nil {
		service.logger.Warn("pelias geocoder failed", zap.Error(err), zap.String("query", normalizedQuery))
	}
	if bestConfidence(peliasResults) >= service.config.PeliasConfidenceThreshold {
		service.logger.Info("geocoder pelias hit", zap.String("query", normalizedQuery), zap.Float64("confidence", bestConfidence(peliasResults)))
		return limitResults(peliasResults, request.Limit), nil
	}

	yandexResults, yandexErr := service.searchExternalProvider(ctx, geodomain.ProviderYandex, normalizedQuery, request, externalRequest, request.RequestedAt)
	if yandexErr != nil {
		service.logger.Warn("yandex geocoder failed", zap.Error(yandexErr), zap.String("query", normalizedQuery))
	}
	if len(yandexResults) > 0 {
		return limitResults(yandexResults, request.Limit), nil
	}

	dadataResults, dadataErr := service.searchExternalProvider(ctx, geodomain.ProviderDaData, normalizedQuery, request, externalRequest, request.RequestedAt)
	if dadataErr != nil {
		service.logger.Warn("dadata geocoder failed", zap.Error(dadataErr), zap.String("query", normalizedQuery))
	}
	if len(dadataResults) > 0 {
		return limitResults(dadataResults, request.Limit), nil
	}

	if len(peliasResults) > 0 {
		return limitResults(peliasResults, request.Limit), nil
	}
	if yandexErr != nil || dadataErr != nil {
		return nil, geodomain.ErrExternalUnavailable
	}
	return []geodomain.SearchResult{}, nil
}

func (service *Service) searchExternalProvider(ctx context.Context, provider geodomain.Provider, normalizedQuery string, request geodomain.SearchRequest, externalRequest geodomain.SearchRequest, requestedAt time.Time) ([]geodomain.SearchResult, error) {
	cachedResults, found, err := service.repository.GetExternalCache(ctx, provider, normalizedQuery, request.CityID, requestedAt)
	if err != nil {
		return nil, fmt.Errorf("get %s geocoder cache: %w", provider, err)
	}
	if found {
		service.logger.Info("geocoder external cache hit", zap.String("provider", string(provider)), zap.String("query", normalizedQuery), zap.Int("results", len(cachedResults)))
		return cachedResults, nil
	}

	var results []geodomain.SearchResult
	switch provider {
	case geodomain.ProviderYandex:
		if !service.config.YandexEnabled || service.yandexClient == nil {
			return nil, nil
		}
		results, err = service.yandexClient.Search(ctx, externalRequest)
	case geodomain.ProviderDaData:
		if !service.config.DaDataEnabled || service.dadataClient == nil {
			return nil, nil
		}
		results, err = service.dadataClient.Search(ctx, externalRequest)
	default:
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}

	expiresAt := requestedAt.Add(service.config.ExternalCacheTTL)
	for index := range results {
		results[index].ExpiresAt = &expiresAt
	}
	requestParams, _ := json.Marshal(map[string]any{
		"query":          request.Query,
		"external_query": externalRequest.Query,
		"city_id":        request.CityID,
		"focus":          request.Focus,
		"limit":          request.Limit,
	})
	if err := service.repository.SaveExternalCache(ctx, ExternalCacheRecord{
		Provider:        provider,
		NormalizedQuery: normalizedQuery,
		CityID:          request.CityID,
		RequestParams:   requestParams,
		Response:        []byte("{}"),
		Results:         results,
		ExpiresAt:       expiresAt,
	}); err != nil {
		return nil, fmt.Errorf("save %s geocoder cache: %w", provider, err)
	}
	service.logger.Info("geocoder external fallback hit", zap.String("provider", string(provider)), zap.String("query", normalizedQuery), zap.Int("results", len(results)))
	return results, nil
}

func (service *Service) ConfirmPoint(ctx context.Context, request ConfirmPointRequest) (geodomain.LocalGeoPoint, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.Address = strings.TrimSpace(request.Address)
	if request.Name == "" {
		request.Name = request.Address
	}
	if request.Address == "" {
		return geodomain.LocalGeoPoint{}, geodomain.ErrInvalidQuery
	}
	if request.Source == "" {
		request.Source = geodomain.PointSourceUserConfirmed
	}
	if request.Confidence == 0 {
		request.Confidence = 1
	}
	if err := geodomain.ValidateConfidence(request.Confidence); err != nil {
		return geodomain.LocalGeoPoint{}, err
	}
	point, err := service.repository.ConfirmPoint(ctx, request)
	if err != nil {
		return geodomain.LocalGeoPoint{}, fmt.Errorf("confirm local geo point: %w", err)
	}
	return point, nil
}

func (service *Service) CreateLocalPoint(ctx context.Context, request AdminLocalPointRequest) (geodomain.LocalGeoPoint, error) {
	request.Name = strings.TrimSpace(request.Name)
	request.Address = strings.TrimSpace(request.Address)
	if request.Name == "" || request.Address == "" {
		return geodomain.LocalGeoPoint{}, geodomain.ErrInvalidQuery
	}
	if request.TrustLevel == "" {
		request.TrustLevel = geodomain.TrustLevelTrusted
	}
	return service.repository.CreateLocalPoint(ctx, request)
}

func (service *Service) ListLocalPoints(ctx context.Context, filter LocalPointFilter) ([]geodomain.LocalGeoPoint, error) {
	if filter.Limit <= 0 || filter.Limit > 200 {
		filter.Limit = 100
	}
	return service.repository.ListLocalPoints(ctx, filter)
}

func (service *Service) ApproveLocalPoint(ctx context.Context, id uuid.UUID, adminUserID *uuid.UUID) (geodomain.LocalGeoPoint, error) {
	return service.repository.ApproveLocalPoint(ctx, id, adminUserID)
}

func (service *Service) RejectLocalPoint(ctx context.Context, id uuid.UUID, adminUserID *uuid.UUID) (geodomain.LocalGeoPoint, error) {
	return service.repository.RejectLocalPoint(ctx, id, adminUserID)
}

func (service *Service) ExportTrustedLocalPoints(ctx context.Context) ([]geodomain.LocalGeoPoint, error) {
	points, err := service.repository.ExportTrustedLocalPoints(ctx)
	if err != nil {
		return nil, fmt.Errorf("export trusted local geo points: %w", err)
	}
	return points, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, geodomain.ErrPointNotFound)
}

func bestConfidence(results []geodomain.SearchResult) float64 {
	best := 0.0
	for _, result := range results {
		if result.Confidence > best {
			best = result.Confidence
		}
	}
	return best
}

func limitResults(results []geodomain.SearchResult, limit int) []geodomain.SearchResult {
	if limit <= 0 || len(results) <= limit {
		return results
	}
	return results[:limit]
}

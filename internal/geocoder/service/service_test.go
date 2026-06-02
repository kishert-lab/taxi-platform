package service

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/geocoder/exporter"
)

func TestSearchReturnsPeliasResultWhenConfidenceIsEnough(t *testing.T) {
	repository := &fakeRepository{}
	pelias := fakeClient{results: []geodomain.SearchResult{testResult(geodomain.ProviderPelias, 0.90)}}
	yandex := fakeClient{results: []geodomain.SearchResult{testResult(geodomain.ProviderYandex, 0.90)}}
	service := newTestService(repository, pelias, yandex)

	results, err := service.Search(context.Background(), geodomain.SearchRequest{Query: "Мира 8"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].Provider != geodomain.ProviderPelias {
		t.Fatalf("expected pelias result, got %#v", results)
	}
	if repository.cacheSaved {
		t.Fatal("pelias hit must not save yandex cache")
	}
}

func TestSearchFallsBackToYandexWhenPeliasConfidenceIsLow(t *testing.T) {
	repository := &fakeRepository{}
	pelias := fakeClient{results: []geodomain.SearchResult{testResult(geodomain.ProviderPelias, 0.40)}}
	yandex := fakeClient{results: []geodomain.SearchResult{testResult(geodomain.ProviderYandex, 0.90)}}
	service := newTestService(repository, pelias, yandex)

	results, err := service.Search(context.Background(), geodomain.SearchRequest{Query: "Мира 8"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].Provider != geodomain.ProviderYandex {
		t.Fatalf("expected yandex result, got %#v", results)
	}
	if !repository.cacheSaved {
		t.Fatal("expected yandex result to be cached")
	}
}

func TestSearchReturnsYandexCacheBeforeCallingYandex(t *testing.T) {
	repository := &fakeRepository{cached: []geodomain.SearchResult{testResult(geodomain.ProviderYandex, 0.88)}, cacheFound: true}
	pelias := fakeClient{results: []geodomain.SearchResult{}}
	yandex := fakeClient{results: []geodomain.SearchResult{testResult(geodomain.ProviderYandex, 0.90)}}
	service := newTestService(repository, pelias, yandex)

	results, err := service.Search(context.Background(), geodomain.SearchRequest{Query: "Мира 8"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].Provider != geodomain.ProviderYandex {
		t.Fatalf("expected cached yandex result, got %#v", results)
	}
	if yandex.calls != 0 {
		t.Fatalf("yandex should not be called when cache hit, calls=%d", yandex.calls)
	}
}

func TestSearchReturnsEmptyResultsWhenPeliasIsEmptyAndYandexIsDisabled(t *testing.T) {
	repository := &fakeRepository{}
	pelias := fakeClient{results: []geodomain.SearchResult{}}
	service := New(repository, &pelias, nil, nil, zap.NewNop(), Config{
		YandexEnabled:             false,
		ExternalCacheTTL:          30 * 24 * time.Hour,
		PeliasConfidenceThreshold: 0.75,
		DefaultLimit:              10,
	})

	results, err := service.Search(context.Background(), geodomain.SearchRequest{Query: "РњРёСЂ"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %#v", results)
	}
}

func TestSearchReturnsEmptyResultsWhenYandexFailsAndPeliasIsEmpty(t *testing.T) {
	repository := &fakeRepository{}
	pelias := fakeClient{results: []geodomain.SearchResult{}}
	yandex := fakeClient{err: errors.New("forbidden")}
	service := newTestService(repository, pelias, yandex)

	_, err := service.Search(context.Background(), geodomain.SearchRequest{Query: "РњРёСЂ"})
	if !errors.Is(err, geodomain.ErrExternalUnavailable) {
		t.Fatalf("expected external unavailable, got %v", err)
	}
}

func TestSearchFallsBackToDaDataWhenYandexIsEmpty(t *testing.T) {
	repository := &fakeRepository{}
	pelias := fakeClient{results: []geodomain.SearchResult{}}
	yandex := fakeClient{results: []geodomain.SearchResult{}}
	dadata := fakeClient{results: []geodomain.SearchResult{testResult(geodomain.ProviderDaData, 0.95)}}
	service := New(repository, &pelias, &yandex, &dadata, zap.NewNop(), Config{
		YandexEnabled:             true,
		DaDataEnabled:             true,
		ExternalCacheTTL:          30 * 24 * time.Hour,
		PeliasConfidenceThreshold: 0.75,
		DefaultLimit:              10,
	})

	results, err := service.Search(context.Background(), geodomain.SearchRequest{Query: "РњРёСЂР° 10"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(results) != 1 || results[0].Provider != geodomain.ProviderDaData {
		t.Fatalf("expected dadata result, got %#v", results)
	}
	if !repository.cacheSaved {
		t.Fatal("expected dadata result to be cached")
	}
}

func TestSearchUsesTaxiParkCityWhenCityIDIsMissing(t *testing.T) {
	cityID := uuid.New()
	center, _ := geodomain.NewCoordinates(58.01, 56.22)
	repository := &fakeRepository{actorCityFound: true, actorCity: CityContext{CityID: cityID, Name: "РџРµСЂРјСЊ", Center: center}}
	pelias := fakeClient{results: []geodomain.SearchResult{testResult(geodomain.ProviderPelias, 0.90)}}
	service := newTestService(repository, pelias, fakeClient{})
	actorUserID := uuid.New()

	if _, err := service.Search(context.Background(), geodomain.SearchRequest{
		Query:       "Мира 8",
		ActorUserID: &actorUserID,
		ActorRole:   "taxi_park",
	}); err != nil {
		t.Fatalf("search: %v", err)
	}
	if repository.lastLocalSearch.CityID == nil || *repository.lastLocalSearch.CityID != cityID {
		t.Fatalf("expected taxi park city filter, got %#v", repository.lastLocalSearch.CityID)
	}
	if repository.lastLocalSearch.Focus == nil || repository.lastLocalSearch.Focus.Latitude != center.Latitude {
		t.Fatalf("expected taxi park city focus, got %#v", repository.lastLocalSearch.Focus)
	}
}

func TestSearchUsesExplicitCityContextForExternalGeocoder(t *testing.T) {
	cityID := uuid.New()
	center, _ := geodomain.NewCoordinates(58.01, 56.22)
	repository := &fakeRepository{cityFound: true, city: CityContext{CityID: cityID, Name: "РџРµСЂРјСЊ", Center: center}}
	pelias := fakeClient{results: []geodomain.SearchResult{testResult(geodomain.ProviderPelias, 0.90)}}
	service := New(repository, &pelias, nil, nil, zap.NewNop(), Config{
		YandexEnabled:             true,
		ExternalCacheTTL:          30 * 24 * time.Hour,
		PeliasConfidenceThreshold: 0.75,
		DefaultLimit:              10,
	})

	if _, err := service.Search(context.Background(), geodomain.SearchRequest{
		Query:  "РњРёСЂР°",
		CityID: &cityID,
	}); err != nil {
		t.Fatalf("search: %v", err)
	}
	if pelias.lastRequest.Query != "РџРµСЂРјСЊ, РњРёСЂР°" {
		t.Fatalf("expected city-qualified external query, got %q", pelias.lastRequest.Query)
	}
	if pelias.lastRequest.Focus == nil || pelias.lastRequest.Focus.Latitude != center.Latitude {
		t.Fatalf("expected city focus, got %#v", pelias.lastRequest.Focus)
	}
}

func TestConfirmPointDelegatesToRepository(t *testing.T) {
	repository := &fakeRepository{}
	service := newTestService(repository, fakeClient{}, fakeClient{})
	coordinates, _ := geodomain.NewCoordinates(58.01, 56.22)

	point, err := service.ConfirmPoint(context.Background(), ConfirmPointRequest{
		CityID:      uuid.New(),
		Address:     "Пермь, Мира 8",
		Coordinates: coordinates,
	})
	if err != nil {
		t.Fatalf("confirm point: %v", err)
	}
	if !repository.confirmCalled {
		t.Fatal("expected repository confirmation")
	}
	if point.ConfirmationCount != 1 {
		t.Fatalf("unexpected confirmation count: %d", point.ConfirmationCount)
	}
}

func TestConfirmPointBecomesTrustedAfterThreeConfirmations(t *testing.T) {
	repository := &fakeRepository{confirmationCount: 2}
	service := newTestService(repository, fakeClient{}, fakeClient{})
	coordinates, _ := geodomain.NewCoordinates(58.01, 56.22)

	point, err := service.ConfirmPoint(context.Background(), ConfirmPointRequest{
		CityID:      uuid.New(),
		Address:     "Пермь, Мира 8",
		Coordinates: coordinates,
	})
	if err != nil {
		t.Fatalf("confirm point: %v", err)
	}
	if point.TrustLevel != geodomain.TrustLevelTrusted {
		t.Fatalf("expected trusted point, got %s", point.TrustLevel)
	}
}

func TestWritePeliasCSVExportsTrustedPoints(t *testing.T) {
	coordinates, _ := geodomain.NewCoordinates(58.01, 56.22)
	point := geodomain.LocalGeoPoint{
		ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		CityID:      uuid.MustParse("22222222-2222-2222-2222-222222222222"),
		Name:        "Мира 8",
		Address:     "Пермь, Мира 8",
		Coordinates: coordinates,
		Confidence:  1,
		TrustLevel:  geodomain.TrustLevelTrusted,
	}
	var buffer bytes.Buffer
	if err := exporter.WritePeliasCSV(&buffer, []geodomain.LocalGeoPoint{point}); err != nil {
		t.Fatalf("write csv: %v", err)
	}
	if !bytes.Contains(buffer.Bytes(), []byte("taxi_platform")) || !bytes.Contains(buffer.Bytes(), []byte("Мира 8")) {
		t.Fatalf("unexpected csv: %s", buffer.String())
	}
}

func newTestService(repository *fakeRepository, pelias fakeClient, yandex fakeClient) *Service {
	return New(repository, &pelias, &yandex, nil, zap.NewNop(), Config{
		YandexEnabled:             true,
		ExternalCacheTTL:          30 * 24 * time.Hour,
		PeliasConfidenceThreshold: 0.75,
		DefaultLimit:              10,
	})
}

func testResult(provider geodomain.Provider, confidence float64) geodomain.SearchResult {
	coordinates, _ := geodomain.NewCoordinates(58.01, 56.22)
	return geodomain.SearchResult{
		ID:          string(provider) + "-1",
		Provider:    provider,
		Name:        "Мира 8",
		Address:     "Пермь, Мира 8",
		Coordinates: coordinates,
		Confidence:  confidence,
	}
}

type fakeClient struct {
	results     []geodomain.SearchResult
	err         error
	calls       int
	lastRequest geodomain.SearchRequest
}

func (client *fakeClient) Search(_ context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	client.calls++
	client.lastRequest = request
	if client.err != nil {
		return nil, client.err
	}
	return client.results, nil
}

type fakeRepository struct {
	cached            []geodomain.SearchResult
	cacheFound        bool
	cacheSaved        bool
	confirmCalled     bool
	confirmationCount int
	actorCityFound    bool
	actorCity         CityContext
	cityFound         bool
	city              CityContext
	lastLocalSearch   LocalSearchRequest
}

func (repository *fakeRepository) ResolveActorCity(context.Context, uuid.UUID, string) (CityContext, bool, error) {
	return repository.actorCity, repository.actorCityFound, nil
}

func (repository *fakeRepository) ResolveCity(context.Context, uuid.UUID) (CityContext, bool, error) {
	return repository.city, repository.cityFound, nil
}

func (repository *fakeRepository) SearchLocalPoints(_ context.Context, request LocalSearchRequest) ([]geodomain.SearchResult, error) {
	repository.lastLocalSearch = request
	return nil, nil
}

func (repository *fakeRepository) GetExternalCache(context.Context, geodomain.Provider, string, *uuid.UUID, time.Time) ([]geodomain.SearchResult, bool, error) {
	return repository.cached, repository.cacheFound, nil
}

func (repository *fakeRepository) SaveExternalCache(context.Context, ExternalCacheRecord) error {
	repository.cacheSaved = true
	return nil
}

func (repository *fakeRepository) ConfirmPoint(_ context.Context, request ConfirmPointRequest) (geodomain.LocalGeoPoint, error) {
	repository.confirmCalled = true
	count := repository.confirmationCount + 1
	trustLevel := geodomain.TrustLevelConfirmed
	if count >= 3 {
		trustLevel = geodomain.TrustLevelTrusted
	}
	return geodomain.LocalGeoPoint{
		ID:                uuid.New(),
		CityID:            request.CityID,
		Name:              request.Address,
		Address:           request.Address,
		Coordinates:       request.Coordinates,
		TrustLevel:        trustLevel,
		ConfirmationCount: count,
	}, nil
}

func (repository *fakeRepository) CreateLocalPoint(context.Context, AdminLocalPointRequest) (geodomain.LocalGeoPoint, error) {
	return geodomain.LocalGeoPoint{}, nil
}

func (repository *fakeRepository) ListLocalPoints(context.Context, LocalPointFilter) ([]geodomain.LocalGeoPoint, error) {
	return nil, nil
}

func (repository *fakeRepository) ApproveLocalPoint(context.Context, uuid.UUID, *uuid.UUID) (geodomain.LocalGeoPoint, error) {
	return geodomain.LocalGeoPoint{}, nil
}

func (repository *fakeRepository) RejectLocalPoint(context.Context, uuid.UUID, *uuid.UUID) (geodomain.LocalGeoPoint, error) {
	return geodomain.LocalGeoPoint{}, nil
}

func (repository *fakeRepository) ExportTrustedLocalPoints(context.Context) ([]geodomain.LocalGeoPoint, error) {
	return nil, nil
}

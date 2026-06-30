package passenger

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
)

func TestSearchPassengerAddressesResolvesCityIDFromCoordinates(t *testing.T) {
	passengerID := uuid.New()
	cityID := uuid.New()
	latitude := 58.010455
	longitude := 56.229443
	searcher := &fakeAddressSearcher{
		cityFound: true,
		cityContext: geoservice.CityContext{
			CityID: cityID,
			Name:   "Perm",
			Center: geodomain.Coordinates{Latitude: latitude, Longitude: longitude},
		},
		results: []geodomain.SearchResult{{ID: "pelias:1", Provider: geodomain.ProviderPelias}},
	}
	service := NewAddressSearchService(searcher)

	results, err := service.SearchPassengerAddresses(
		context.Background(),
		passengerID,
		"Mira",
		nil,
		&latitude,
		&longitude,
		5,
	)
	if err != nil {
		t.Fatalf("SearchPassengerAddresses returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if searcher.lastRequest.CityID == nil || *searcher.lastRequest.CityID != cityID {
		t.Fatalf("expected resolved city id %s, got %#v", cityID, searcher.lastRequest.CityID)
	}
	if searcher.lastRequest.ActorRole != string(domain.UserRolePassenger) {
		t.Fatalf("expected passenger actor role, got %s", searcher.lastRequest.ActorRole)
	}
}

func TestSearchPassengerAddressesFallsBackWithoutResolvedCityIDWhenEmpty(t *testing.T) {
	passengerID := uuid.New()
	cityID := uuid.New()
	latitude := 56.8389
	longitude := 60.6057
	searcher := &fakeAddressSearcher{
		cityFound: true,
		cityContext: geoservice.CityContext{
			CityID: cityID,
			Name:   "Yekaterinburg",
			Center: geodomain.Coordinates{Latitude: latitude, Longitude: longitude},
		},
		searchResultsQueue: [][]geodomain.SearchResult{
			{},
			{{ID: "pelias:focus", Provider: geodomain.ProviderPelias}},
		},
	}
	service := NewAddressSearchService(searcher)

	results, err := service.SearchPassengerAddresses(
		context.Background(),
		passengerID,
		"lenina",
		nil,
		&latitude,
		&longitude,
		5,
	)
	if err != nil {
		t.Fatalf("SearchPassengerAddresses returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result after fallback, got %d", len(results))
	}
	if len(searcher.requests) != 2 {
		t.Fatalf("expected two search attempts, got %d", len(searcher.requests))
	}
	if searcher.requests[0].CityID == nil || *searcher.requests[0].CityID != cityID {
		t.Fatalf("expected first request to use resolved city id %s, got %#v", cityID, searcher.requests[0].CityID)
	}
	if searcher.requests[1].CityID != nil {
		t.Fatalf("expected second request without city id, got %#v", searcher.requests[1].CityID)
	}
	if searcher.requests[1].Focus == nil {
		t.Fatal("expected second request to preserve focus coordinates")
	}
}

type fakeAddressSearcher struct {
	cityContext        geoservice.CityContext
	cityFound          bool
	results            []geodomain.SearchResult
	searchResultsQueue [][]geodomain.SearchResult
	lastRequest        geodomain.SearchRequest
	requests           []geodomain.SearchRequest
}

func (searcher *fakeAddressSearcher) ResolveCityByCoordinates(_ context.Context, _ geodomain.Coordinates) (geoservice.CityContext, bool, error) {
	return searcher.cityContext, searcher.cityFound, nil
}

func (searcher *fakeAddressSearcher) Search(_ context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	searcher.lastRequest = request
	searcher.requests = append(searcher.requests, request)
	if len(searcher.searchResultsQueue) > 0 {
		results := searcher.searchResultsQueue[0]
		searcher.searchResultsQueue = searcher.searchResultsQueue[1:]
		return results, nil
	}
	return searcher.results, nil
}

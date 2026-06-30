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

type fakeAddressSearcher struct {
	cityContext geoservice.CityContext
	cityFound   bool
	results     []geodomain.SearchResult
	lastRequest geodomain.SearchRequest
}

func (searcher *fakeAddressSearcher) ResolveCityByCoordinates(_ context.Context, _ geodomain.Coordinates) (geoservice.CityContext, bool, error) {
	return searcher.cityContext, searcher.cityFound, nil
}

func (searcher *fakeAddressSearcher) Search(_ context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	searcher.lastRequest = request
	return searcher.results, nil
}

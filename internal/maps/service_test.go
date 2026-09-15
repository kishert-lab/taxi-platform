package maps

import (
	"context"
	"github.com/kishert-lab/taxi-platform/configs"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/routing"
	"testing"
)

type reverseStub struct{ results []geodomain.SearchResult }

func (stub reverseStub) Reverse(context.Context, geodomain.Coordinates) ([]geodomain.SearchResult, error) {
	return stub.results, nil
}
func TestReversePreservesPickupAndEmptyResult(t *testing.T) {
	service := New(configs.MapsConfig{}, routing.UnavailableService{}, true, false)
	point := geodomain.Coordinates{Latitude: 58, Longitude: 56}
	house := geodomain.Coordinates{Latitude: 58.001, Longitude: 56.001}
	service.WithReverseGeocoder(reverseStub{[]geodomain.SearchResult{{Address: "House", Coordinates: house}}})
	result, err := service.Reverse(context.Background(), point)
	if err != nil {
		t.Fatal(err)
	}
	if result.Point != point || !result.Found || *result.AddressLocation != house {
		t.Fatalf("pickup overwritten: %+v", result)
	}
	service.WithReverseGeocoder(reverseStub{})
	result, err = service.Reverse(context.Background(), point)
	if err != nil {
		t.Fatal(err)
	}
	if result.Found || result.Point != point || result.AddressLocation != nil {
		t.Fatalf("bad empty response: %+v", result)
	}
}
func TestConfigurationOnlyExposesVersionedHTTPSResources(t *testing.T) {
	service := New(configs.MapsConfig{PublicURL: "https://maps.example.test", DataVersion: "202609", Bounds: []float64{19, 41, -169, 82}, MaxZoom: 22}, routing.UnavailableService{}, true, false)
	result, err := service.Configuration(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.StyleURL != "https://maps.example.test/releases/202609/style.json" || result.RoutingAvailable {
		t.Fatalf("bad config: %+v", result)
	}
	service.config.PublicURL = "http://osrm:5000"
	if _, err := service.Configuration(context.Background()); err == nil {
		t.Fatal("internal HTTP origin exposed")
	}
}

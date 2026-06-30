package dadata

import (
	"testing"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
)

func TestPrioritizeByFocusPrefersNearbyResults(t *testing.T) {
	focus, _ := geodomain.NewCoordinates(56.8389, 60.6057)
	near, _ := geodomain.NewCoordinates(56.84, 60.61)
	far, _ := geodomain.NewCoordinates(52.6516303, 90.0885949)

	results := prioritizeByFocus([]geodomain.SearchResult{
		{ID: "far", Coordinates: far},
		{ID: "near", Coordinates: near},
	}, &focus)

	if len(results) != 1 {
		t.Fatalf("expected one nearby result after focus filtering, got %d", len(results))
	}
	if results[0].ID != "near" {
		t.Fatalf("expected nearby result first, got %s", results[0].ID)
	}
}

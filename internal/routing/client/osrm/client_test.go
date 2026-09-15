package osrm

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/routing"
)

func TestClientRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/route/v1/driving/60.600000,56.800000;60.700000,56.900000" {
			t.Fatalf("unexpected path: %s", request.URL.Path)
		}
		if request.URL.Query().Get("overview") != "full" || request.URL.Query().Get("geometries") != "geojson" {
			t.Fatalf("unexpected query: %s", request.URL.RawQuery)
		}
		_, _ = writer.Write([]byte(`{"code":"Ok","routes":[{"distance":1234.4,"duration":321.6,"geometry":{"type":"LineString","coordinates":[[60.6,56.8],[60.7,56.9]]}}],"waypoints":[{"location":[60.6,56.8],"distance":0},{"location":[60.7,56.9],"distance":0}]}`))
	}))
	defer server.Close()

	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	route, err := client.Route(context.Background(), coordinates(t, 56.8, 60.6), coordinates(t, 56.9, 60.7))
	if err != nil {
		t.Fatalf("Route() error = %v", err)
	}
	if route.DistanceMeters != 1234 || route.DurationSeconds != 322 || route.Source != "osrm" {
		t.Fatalf("unexpected route: %+v", route)
	}
}

func TestClientRouteNoRoute(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"code":"NoRoute","routes":[]}`))
	}))
	defer server.Close()
	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.Route(context.Background(), coordinates(t, 56.8, 60.6), coordinates(t, 56.9, 60.7))
	if !errors.Is(err, routing.ErrRouteNotFound) {
		t.Fatalf("Route() error = %v, want ErrRouteNotFound", err)
	}
}

func TestClientRouteUnavailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client, err := New(server.URL, server.Client())
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	_, err = client.Route(context.Background(), coordinates(t, 56.8, 60.6), coordinates(t, 56.9, 60.7))
	if !errors.Is(err, routing.ErrUnavailable) {
		t.Fatalf("Route() error = %v, want ErrUnavailable", err)
	}
}

func coordinates(t *testing.T, latitude float64, longitude float64) geodomain.Coordinates {
	t.Helper()
	coordinates, err := geodomain.NewCoordinates(latitude, longitude)
	if err != nil {
		t.Fatalf("NewCoordinates() error = %v", err)
	}
	return coordinates
}

func TestInvalidAndUnavailableResponses(t *testing.T) {
	for _, test := range []struct {
		name     string
		status   int
		body     string
		expected error
	}{
		{"missing_geometry", 200, `{"code":"Ok","routes":[{"distance":10,"duration":3}]}`, routing.ErrInvalidResponse},
		{"empty_body", 200, ``, routing.ErrInvalidResponse},
		{"missing_duration", 200, `{"code":"Ok","routes":[{"distance":10,"geometry":{"type":"LineString","coordinates":[[60,56],[61,57]]}}],"waypoints":[{},{}]}`, routing.ErrInvalidResponse},
		{"outside_graph", 400, `{"code":"NoSegment"}`, routing.ErrOutsideCoverage},
		{"bad_gateway", 502, `<html>bad gateway</html>`, routing.ErrUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			client, err := New(server.URL, server.Client())
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Route(context.Background(), coordinates(t, 56, 60), coordinates(t, 57, 61))
			if !errors.Is(err, test.expected) {
				t.Fatalf("got %v, want %v", err, test.expected)
			}
		})
	}
}
func TestValidationBeforeHTTP(t *testing.T) {
	client, err := NewWithOptions("http://127.0.0.1:1", nil, Options{DataVersion: "test", MaxSnapMeters: 100, Bounds: [4]float64{50, 55, 65, 60}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Route(context.Background(), geodomain.Coordinates{Latitude: 100, Longitude: 60}, coordinates(t, 57, 61))
	if !errors.Is(err, routing.ErrInvalidRequest) {
		t.Fatal(err)
	}
	// A latitude/longitude swap remains numerically valid but falls outside coverage.
	_, err = client.Route(context.Background(), coordinates(t, 60, 56), coordinates(t, 61, 57))
	if !errors.Is(err, routing.ErrOutsideCoverage) {
		t.Fatal(err)
	}
}
func TestContextCancellation(t *testing.T) {
	client, err := New("http://127.0.0.1:1", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = client.Route(ctx, coordinates(t, 56, 60), coordinates(t, 57, 61))
	if !errors.Is(err, context.Canceled) || !errors.Is(err, routing.ErrUnavailable) {
		t.Fatal(err)
	}
}

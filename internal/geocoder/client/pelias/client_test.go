package pelias

import (
	"context"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReverseUsesNamedCoordinates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/reverse" || request.URL.Query().Get("point.lat") != "58" || request.URL.Query().Get("point.lon") != "56" || request.URL.Query().Get("text") != "" {
			t.Error(request.URL.String())
		}
		_, _ = writer.Write([]byte(`{"features":[{"geometry":{"coordinates":[56.01,58.01]},"properties":{"label":"Address","confidence":1}}]}`))
	}))
	defer server.Close()
	results, err := New(server.URL, server.Client()).Reverse(context.Background(), geodomain.Coordinates{Latitude: 58, Longitude: 56})
	if err != nil || len(results) != 1 || results[0].Coordinates.Latitude != 58.01 {
		t.Fatalf("%+v %v", results, err)
	}
}

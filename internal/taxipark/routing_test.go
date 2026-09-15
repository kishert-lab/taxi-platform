package taxipark

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/routing"
	"github.com/kishert-lab/taxi-platform/internal/routing/client/osrm"
	"net/http"
	"net/http/httptest"
	"testing"
)

type roadTariffReader struct{ tariff domain.TaxiParkTariff }

func (reader roadTariffReader) GetOrderTariff(context.Context, uuid.UUID, uuid.UUID) (domain.TaxiParkTariff, error) {
	return reader.tariff, nil
}
func TestDispatcherEstimateUsesOSRMAndSnapshotsRates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/route/v1/driving/60.000000,56.000000;61.000000,57.000000" {
			t.Error(request.URL.Path)
		}
		_, _ = writer.Write([]byte(`{"code":"Ok","routes":[{"distance":1250,"duration":61,"geometry":{"type":"LineString","coordinates":[[60,56],[61,57]]}}],"waypoints":[{"location":[60,56],"distance":0},{"location":[61,57],"distance":0}]}`))
	}))
	defer server.Close()
	client, err := osrm.NewWithOptions(server.URL, server.Client(), osrm.Options{DataVersion: "release-test", MaxSnapMeters: 500, Bounds: [4]float64{-180, -90, 180, 90}})
	if err != nil {
		t.Fatal(err)
	}
	tariff := domain.TaxiParkTariff{ID: uuid.New(), PricingMode: domain.PricingModeDistanceTime, BasePrice: domain.Money{Amount: 10000, Currency: "RUB"}, PricePerKM: domain.Money{Amount: 2500}, PricePerMinute: domain.Money{Amount: 500}}
	service := NewService(&fakeRepository{}, fakePasswordHasher{}).WithRoutingService(client, roadTariffReader{tariff})
	destination := domain.Coordinates{Latitude: 57, Longitude: 61}
	snapshot, err := service.estimateRoadPrice(context.Background(), uuid.New(), tariff.ID, domain.Coordinates{Latitude: 56, Longitude: 60}, &destination)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.EstimatedPrice.Amount != 14125 || *snapshot.RouteDurationSeconds != 61 || *snapshot.RouteDistanceMeters != 1250 || snapshot.RouteDataVersion != "release-test" || len(snapshot.TariffRates) != 1 || snapshot.IsFinal {
		t.Fatalf("incorrect road price snapshot: %+v", snapshot)
	}
	service.WithRoutingService(routing.UnavailableService{}, roadTariffReader{tariff})
	if _, err := service.estimateRoadPrice(context.Background(), uuid.New(), tariff.ID, domain.Coordinates{Latitude: 56, Longitude: 60}, &destination); !errors.Is(err, routing.ErrUnavailable) {
		t.Fatalf("expected explicit unavailability, got %v", err)
	}
}

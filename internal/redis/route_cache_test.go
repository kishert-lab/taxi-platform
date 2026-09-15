package redis

import (
	"context"
	"github.com/alicebob/miniredis/v2"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/routing"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"testing"
	"time"
)

type countingProvider struct{ calls int }

func (provider *countingProvider) Route(context.Context, geodomain.Coordinates, geodomain.Coordinates) (routing.Route, error) {
	provider.calls++
	return routing.Route{Source: "osrm", DistanceMeters: 100, DurationSeconds: 20}, nil
}
func TestCacheSeparatesPointOrderAndRelease(t *testing.T) {
	server := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: server.Addr()})
	defer client.Close()
	provider := &countingProvider{}
	newCache := func(version string) *CachedService {
		service, err := NewCachedService(provider, client, version, time.Minute, zap.NewNop(), prometheus.NewRegistry())
		if err != nil {
			t.Fatal(err)
		}
		return service
	}
	first := newCache("graph-1|driving|500")
	a, b := geodomain.Coordinates{Latitude: 56, Longitude: 60}, geodomain.Coordinates{Latitude: 57, Longitude: 61}
	for i := 0; i < 2; i++ {
		if _, err := first.Route(context.Background(), a, b); err != nil {
			t.Fatal(err)
		}
	}
	if provider.calls != 1 {
		t.Fatalf("expected cache hit; calls=%d", provider.calls)
	}
	if _, err := first.Route(context.Background(), b, a); err != nil {
		t.Fatal(err)
	}
	if _, err := newCache("graph-2|driving|500").Route(context.Background(), a, b); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 3 {
		t.Fatalf("cache mixed point order or graph version: %d", provider.calls)
	}
	server.FastForward(2 * time.Minute)
	if _, err := first.Route(context.Background(), a, b); err != nil {
		t.Fatal(err)
	}
	if provider.calls != 4 {
		t.Fatalf("cache TTL not honored: %d", provider.calls)
	}
}

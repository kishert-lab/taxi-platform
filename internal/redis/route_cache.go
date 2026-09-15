package redis

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/kishert-lab/taxi-platform/internal/routing"
	"time"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// CachedService stores only successful routes. Namespace includes graph version,
// provider URL, profile, bounds, geometry options and snapping radius.
type CachedService struct {
	provider  routing.Service
	redis     *redis.Client
	namespace string
	ttl       time.Duration
	logger    *zap.Logger
	latency   *prometheus.HistogramVec
}

func NewCachedService(provider routing.Service, client *redis.Client, namespace string, ttl time.Duration, logger *zap.Logger, registry prometheus.Registerer) (*CachedService, error) {
	histogram := prometheus.NewHistogramVec(prometheus.HistogramOpts{Name: "taxi_routing_duration_seconds", Help: "Routing latency including cache access; status distinguishes errors.", Buckets: prometheus.DefBuckets}, []string{"status"})
	if err := registry.Register(histogram); err != nil {
		return nil, fmt.Errorf("register routing metrics: %w", err)
	}
	return &CachedService{provider: provider, redis: client, namespace: namespace, ttl: ttl, logger: logger, latency: histogram}, nil
}

func (service *CachedService) Route(ctx context.Context, origin, destination geodomain.Coordinates) (result routing.Route, routeError error) {
	started := time.Now()
	status := "error"
	defer func() { service.latency.WithLabelValues(status).Observe(time.Since(started).Seconds()) }()
	encoded, err := json.Marshal(struct {
		Namespace string
		Points    [2]geodomain.Coordinates
	}{service.namespace, [2]geodomain.Coordinates{origin, destination}})
	if err != nil {
		return routing.Route{}, fmt.Errorf("encode route cache key: %w: %w", routing.ErrInvalidRequest, err)
	}
	key := fmt.Sprintf("routing:v1:%x", sha256.Sum256(encoded))
	cached, err := service.redis.Get(ctx, key).Bytes()
	if err == nil {
		if decodeError := json.Unmarshal(cached, &result); decodeError == nil {
			status = "cache_hit"
			return result, nil
		} else {
			service.logger.Warn("decode route cache", zap.String("operation", "routing.cache.read"), zap.Error(decodeError))
		}
	} else if !errors.Is(err, redis.Nil) {
		service.logger.Warn("read route cache", zap.String("operation", "routing.cache.read"), zap.Error(err))
	}
	result, routeError = service.provider.Route(ctx, origin, destination)
	if routeError != nil {
		service.logger.Warn("calculate road route", zap.String("operation", "routing.route"), zap.Error(routeError))
		return routing.Route{}, routeError
	}
	status = "success"
	encoded, err = json.Marshal(result)
	if err != nil {
		return routing.Route{}, fmt.Errorf("encode route cache value: %w", err)
	}
	if err := service.redis.Set(ctx, key, encoded, service.ttl).Err(); err != nil {
		service.logger.Warn("write route cache", zap.String("operation", "routing.cache.write"), zap.Error(err))
	}
	return result, nil
}

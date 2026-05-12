package geo

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

func TestLocationServiceThrottlesFrequentUpdates(t *testing.T) {
	t.Parallel()

	repository := &fakeLocationRepository{}
	throttle := &fakeLocationThrottle{allowed: false}
	service := NewLocationService(repository, throttle)

	err := service.UpdateLocation(context.Background(), DriverLocationUpdate{
		DriverID: uuid.New(),
		CityID:   uuid.New(),
		Location: domain.Coordinates{Latitude: 56.8, Longitude: 60.5},
	})
	if !errors.Is(err, ErrLocationUpdateThrottled) {
		t.Fatalf("expected throttled error, got %v", err)
	}
	if repository.updated {
		t.Fatal("expected repository not to be called")
	}
}

type fakeLocationRepository struct {
	updated bool
}

func (repository *fakeLocationRepository) UpdateDriverLocation(_ context.Context, _ DriverLocationUpdate) error {
	repository.updated = true
	return nil
}

func (repository *fakeLocationRepository) MarkStaleDriversOffline(_ context.Context, _ time.Time, _ int) (int, error) {
	return 0, nil
}

type fakeLocationThrottle struct {
	allowed bool
}

func (throttle *fakeLocationThrottle) AllowLocationUpdate(_ context.Context, _ uuid.UUID, _ time.Duration) (bool, error) {
	return throttle.allowed, nil
}

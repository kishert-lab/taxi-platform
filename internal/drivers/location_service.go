package drivers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type LocationRepository interface {
	UpdateDriverLocation(ctx context.Context, update LocationUpdate) error
	MarkStaleDriversOffline(ctx context.Context, staleBefore time.Time, limit int) (int, error)
}

type LocationThrottle interface {
	AllowLocationUpdate(ctx context.Context, driverID uuid.UUID, interval time.Duration) (bool, error)
}

type LocationService struct {
	locationRepository LocationRepository
	locationThrottle   LocationThrottle
	minUpdateInterval  time.Duration
	staleAfter         time.Duration
}

type LocationUpdate struct {
	DriverID       uuid.UUID
	CityID         uuid.UUID
	Location       domain.Coordinates
	Heading        *int16
	SpeedMPS       *float64
	AccuracyMeters *float64
	RecordedAt     time.Time
}

var ErrLocationUpdateThrottled = errors.New("driver location update throttled")

func NewLocationService(locationRepository LocationRepository, locationThrottle LocationThrottle) *LocationService {
	return &LocationService{
		locationRepository: locationRepository,
		locationThrottle:   locationThrottle,
		minUpdateInterval:  2 * time.Second,
		staleAfter:         30 * time.Second,
	}
}

func (service *LocationService) UpdateLocation(ctx context.Context, update LocationUpdate) error {
	if update.RecordedAt.IsZero() {
		update.RecordedAt = time.Now().UTC()
	}

	allowed, err := service.locationThrottle.AllowLocationUpdate(ctx, update.DriverID, service.minUpdateInterval)
	if err != nil {
		return fmt.Errorf("check driver location throttle: %w", err)
	}
	if !allowed {
		return ErrLocationUpdateThrottled
	}

	if err := service.locationRepository.UpdateDriverLocation(ctx, update); err != nil {
		return fmt.Errorf("update driver location: %w", err)
	}
	return nil
}

func (service *LocationService) UpdateLocationBatch(ctx context.Context, updates []LocationUpdate) error {
	for _, update := range updates {
		if err := service.UpdateLocation(ctx, update); err != nil && !errors.Is(err, ErrLocationUpdateThrottled) {
			return err
		}
	}
	return nil
}

func (service *LocationService) MarkStaleDriversOffline(ctx context.Context, limit int) (int, error) {
	count, err := service.locationRepository.MarkStaleDriversOffline(ctx, time.Now().UTC().Add(-service.staleAfter), limit)
	if err != nil {
		return 0, fmt.Errorf("mark stale drivers offline: %w", err)
	}
	return count, nil
}

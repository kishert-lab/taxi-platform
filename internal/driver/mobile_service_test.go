package driver

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geo"
)

func TestAppendOrderRoutePointsSuccessAndDuplicateRetry(t *testing.T) {
	driverUserID := uuid.New()
	driverID := uuid.New()
	orderID := uuid.New()

	repository := &fakeMobileRepository{
		routeAccess: OrderRouteUploadAccess{
			OrderID:      orderID,
			DriverID:     driverID,
			DriverUserID: driverUserID,
			Status:       domain.OrderStatusInProgress,
		},
		routePointKeys: make(map[string]struct{}),
	}
	service := NewMobileService(repository, nil, nil, zap.NewNop())
	request := dto.DriverOrderRouteBatchRequest{
		Points: []dto.DriverOrderRouteBatchPointRequest{
			{
				Location:   dto.CoordinatesRequest{Latitude: 56.8, Longitude: 60.6},
				RecordedAt: time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
			},
			{
				Location:   dto.CoordinatesRequest{Latitude: 56.81, Longitude: 60.61},
				RecordedAt: time.Date(2026, 6, 28, 10, 0, 5, 0, time.UTC),
			},
		},
	}

	firstResult, err := service.AppendOrderRoutePoints(context.Background(), driverUserID, orderID, request)
	if err != nil {
		t.Fatalf("first append route points: %v", err)
	}
	if firstResult.AcceptedPoints != 2 || firstResult.IgnoredPoints != 0 {
		t.Fatalf("unexpected first result: %+v", firstResult)
	}

	secondResult, err := service.AppendOrderRoutePoints(context.Background(), driverUserID, orderID, request)
	if err != nil {
		t.Fatalf("second append route points: %v", err)
	}
	if secondResult.AcceptedPoints != 0 || secondResult.IgnoredPoints != 2 {
		t.Fatalf("unexpected second result: %+v", secondResult)
	}
}

func TestAppendOrderRoutePointsForbiddenForForeignOrder(t *testing.T) {
	service := NewMobileService(&fakeMobileRepository{
		routeAccess: OrderRouteUploadAccess{
			OrderID:      uuid.New(),
			DriverID:     uuid.New(),
			DriverUserID: uuid.New(),
			Status:       domain.OrderStatusInProgress,
		},
		routePointKeys: make(map[string]struct{}),
	}, nil, nil, zap.NewNop())

	_, err := service.AppendOrderRoutePoints(context.Background(), uuid.New(), uuid.New(), dto.DriverOrderRouteBatchRequest{
		Points: []dto.DriverOrderRouteBatchPointRequest{{
			Location:   dto.CoordinatesRequest{Latitude: 56.8, Longitude: 60.6},
			RecordedAt: time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		}},
	})
	if err == nil || err != ErrOrderAccessDenied {
		t.Fatalf("expected ErrOrderAccessDenied, got %v", err)
	}
}

func TestAppendOrderRoutePointsNotFound(t *testing.T) {
	service := NewMobileService(&fakeMobileRepository{
		routeAccessErr: ErrCurrentOrderNotFound,
		routePointKeys: make(map[string]struct{}),
	}, nil, nil, zap.NewNop())

	_, err := service.AppendOrderRoutePoints(context.Background(), uuid.New(), uuid.New(), dto.DriverOrderRouteBatchRequest{
		Points: []dto.DriverOrderRouteBatchPointRequest{{
			Location:   dto.CoordinatesRequest{Latitude: 56.8, Longitude: 60.6},
			RecordedAt: time.Date(2026, 6, 28, 10, 0, 0, 0, time.UTC),
		}},
	})
	if err == nil || err != ErrCurrentOrderNotFound {
		t.Fatalf("expected ErrCurrentOrderNotFound, got %v", err)
	}
}

type fakeMobileRepository struct {
	routeAccess    OrderRouteUploadAccess
	routeAccessErr error
	routePointKeys map[string]struct{}
}

func (repository *fakeMobileRepository) GetProfileByUserID(context.Context, uuid.UUID) (Profile, error) {
	return Profile{}, nil
}

func (repository *fakeMobileRepository) UpdateProfileByUserID(context.Context, uuid.UUID, ProfilePatch) (Profile, error) {
	return Profile{}, nil
}

func (repository *fakeMobileRepository) SetStatusByUserID(context.Context, uuid.UUID, domain.DriverStatus) (Profile, error) {
	return Profile{}, nil
}

func (repository *fakeMobileRepository) ListCarsByUserID(context.Context, uuid.UUID) ([]domain.Car, error) {
	return nil, nil
}

func (repository *fakeMobileRepository) GetCurrentOrderByUserID(context.Context, uuid.UUID) (CurrentOrder, error) {
	return CurrentOrder{}, ErrCurrentOrderNotFound
}

func (repository *fakeMobileRepository) GetOrderByUserID(context.Context, uuid.UUID, uuid.UUID) (CurrentOrder, error) {
	return CurrentOrder{}, ErrCurrentOrderNotFound
}

func (repository *fakeMobileRepository) ListOrderHistoryByUserID(context.Context, uuid.UUID, int) ([]CurrentOrder, error) {
	return nil, nil
}

func (repository *fakeMobileRepository) ListRoutePointsByUserID(context.Context, uuid.UUID, uuid.UUID) ([]RoutePoint, error) {
	return nil, nil
}

func (repository *fakeMobileRepository) GetOrderRouteUploadAccess(context.Context, uuid.UUID) (OrderRouteUploadAccess, error) {
	if repository.routeAccessErr != nil {
		return OrderRouteUploadAccess{}, repository.routeAccessErr
	}
	return repository.routeAccess, nil
}

func (repository *fakeMobileRepository) TransitionOrderByUserID(context.Context, uuid.UUID, uuid.UUID, domain.OrderStatus, string, *int64) (CurrentOrder, error) {
	return CurrentOrder{}, nil
}

func (repository *fakeMobileRepository) AppendRoutePointByUserID(context.Context, uuid.UUID, geoservice.DriverLocationUpdate) error {
	return nil
}

func (repository *fakeMobileRepository) AppendOrderRoutePoints(_ context.Context, orderID uuid.UUID, _ uuid.UUID, points []OrderRouteAppendPoint) (AppendOrderRoutePointsResult, error) {
	acceptedPoints := 0
	for _, point := range points {
		key := orderID.String() + "|" + point.RecordedAt.UTC().Format(time.RFC3339Nano) + "|" + formatCoordinate(point.Location.Latitude) + "|" + formatCoordinate(point.Location.Longitude)
		if _, exists := repository.routePointKeys[key]; exists {
			continue
		}
		repository.routePointKeys[key] = struct{}{}
		acceptedPoints++
	}
	return AppendOrderRoutePointsResult{
		OrderID:        orderID,
		AcceptedPoints: acceptedPoints,
		IgnoredPoints:  len(points) - acceptedPoints,
	}, nil
}

func formatCoordinate(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

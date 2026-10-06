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
	wsmsg "github.com/kishert-lab/taxi-platform/internal/ws"
)

func TestDriverAssignedActionsFollowPriceConfirmation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		confirmation  string
		expectedFirst string
	}{
		{name: "pending", confirmation: "pending", expectedFirst: "cancel"},
		{name: "confirmed", confirmation: "confirmed", expectedFirst: "arriving"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			actions := driverAllowedActionsForOrder(CurrentOrder{
				Status:                 domain.OrderStatusDriverAssigned,
				PriceConfirmationState: testCase.confirmation,
			})
			if len(actions) == 0 || actions[0] != testCase.expectedFirst {
				t.Fatalf("actions=%v, want first action %q", actions, testCase.expectedFirst)
			}
		})
	}
}

func TestMarkDriverStatusPublishesTaxiParkPresence(t *testing.T) {
	for _, testCase := range []struct {
		name   string
		status domain.DriverStatus
		mark   func(*MobileService, context.Context, uuid.UUID) (dto.DriverProfileResponse, error)
	}{
		{name: "online", status: domain.DriverStatusOnline, mark: (*MobileService).MarkDriverOnline},
		{name: "offline", status: domain.DriverStatusOffline, mark: (*MobileService).MarkDriverOffline},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			driverID := uuid.New()
			userID := uuid.New()
			taxiParkID := uuid.New()
			repository := &fakeMobileRepository{statusProfile: Profile{
				DriverID:   driverID,
				UserID:     userID,
				CityID:     uuid.New(),
				TaxiParkID: &taxiParkID,
				FirstName:  "Ivan",
				LastName:   "Ivanov",
			}}
			presence := &fakePresenceStore{}
			gateway := &fakeRealtimeGateway{}
			service := NewMobileServiceWithDispatch(repository, presence, nil, nil, zap.NewNop(), gateway)

			if _, err := testCase.mark(service, context.Background(), userID); err != nil {
				t.Fatalf("mark driver %s: %v", testCase.status, err)
			}
			if gateway.presenceDriverID != driverID {
				t.Fatalf("presence driver id = %s, want %s", gateway.presenceDriverID, driverID)
			}
			payload, ok := gateway.presencePayload.(map[string]any)
			if !ok {
				t.Fatalf("unexpected presence payload: %#v", gateway.presencePayload)
			}
			if payload["user_id"] != userID || payload["status"] != testCase.status {
				t.Fatalf("unexpected presence payload: %#v", payload)
			}
			if _, ok := payload["changed_at"].(time.Time); !ok {
				t.Fatalf("changed_at is missing from presence payload: %#v", payload)
			}
		})
	}
}

func TestDriverOrderStatusPipelineActions(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		status domain.OrderStatus
		action string
	}{
		{status: domain.OrderStatusDriverAssigned, action: "arriving"},
		{status: domain.OrderStatusDriverArriving, action: "arrived"},
		{status: domain.OrderStatusDriverWaiting, action: "start"},
		{status: domain.OrderStatusInProgress, action: "complete"},
	}

	for _, testCase := range testCases {
		actions := driverAllowedActions(testCase.status)
		if len(actions) == 0 || actions[0] != testCase.action {
			t.Fatalf("status=%s actions=%v, want first action %q", testCase.status, actions, testCase.action)
		}
	}
}

func TestCancelDriverOrderNotifiesPassengerWithDriverCancellationPayload(t *testing.T) {
	driverUserID := uuid.New()
	driverID := uuid.New()
	passengerID := uuid.New()
	orderID := uuid.New()

	repository := &fakeMobileRepository{
		profile: Profile{
			DriverID: driverID,
			UserID:   driverUserID,
			CityID:   uuid.New(),
		},
		transitionOrderResult: CurrentOrder{
			OrderID:     orderID,
			DriverID:    driverID,
			PassengerID: passengerID,
			Status:      domain.OrderStatusCancelled,
			Version:     3,
		},
	}
	presence := &fakePresenceStore{}
	realtime := &fakeRealtimeGateway{}
	notifier := &fakePassengerNotifier{}

	service := NewMobileServiceWithDispatch(repository, presence, nil, nil, zap.NewNop(), realtime).
		WithPassengerNotifier(notifier)

	_, err := service.CancelDriverOrder(context.Background(), driverUserID, orderID, "driver unavailable")
	if err != nil {
		t.Fatalf("CancelDriverOrder returned error: %v", err)
	}

	if realtime.passengerEvent != "order.cancelled" {
		t.Fatalf("expected passenger event order.cancelled, got %s", realtime.passengerEvent)
	}
	payload, ok := realtime.passengerPayload.(wsmsg.PassengerOrderStatePayload)
	if !ok {
		t.Fatalf("unexpected passenger payload type: %#v", realtime.passengerPayload)
	}
	if payload.CancelledBy != "driver" {
		t.Fatalf("expected cancelled_by=driver, got %s", payload.CancelledBy)
	}
	if payload.CancellationReason != "driver unavailable" {
		t.Fatalf("expected cancellation reason, got %s", payload.CancellationReason)
	}
	if !presence.markedOnline {
		t.Fatal("expected driver to be marked online after cancellation")
	}
	if len(notifier.notifications) != 1 {
		t.Fatalf("expected one passenger push notification, got %d", len(notifier.notifications))
	}
}

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

func TestCompleteDriverTripIgnoresClientFinalPrice(t *testing.T) {
	driverUserID := uuid.New()
	orderID := uuid.New()
	repository := &fakeMobileRepository{
		profile: Profile{DriverID: uuid.New(), UserID: driverUserID, CityID: uuid.New()},
		completionOrderResult: CurrentOrder{
			OrderID:  orderID,
			DriverID: uuid.New(),
			Status:   domain.OrderStatusCompleted,
		},
	}
	service := NewMobileServiceWithDispatch(repository, &fakePresenceStore{}, nil, nil, zap.NewNop(), nil)
	clientPrice := int64(99999999)

	if _, err := service.CompleteDriverTrip(context.Background(), driverUserID, orderID, dto.CompleteOrderRequest{FinalPrice: &clientPrice}); err != nil {
		t.Fatalf("complete driver trip: %v", err)
	}
	if repository.completedOrderID != orderID {
		t.Fatalf("expected server completion for order %s, got %s", orderID, repository.completedOrderID)
	}
}

type fakeMobileRepository struct {
	profile               Profile
	statusProfile         Profile
	currentOrder          CurrentOrder
	transitionOrderResult CurrentOrder
	completionOrderResult CurrentOrder
	completedOrderID      uuid.UUID
	routeAccess           OrderRouteUploadAccess
	routeAccessErr        error
	routePointKeys        map[string]struct{}
}

func (repository *fakeMobileRepository) GetProfileByUserID(context.Context, uuid.UUID) (Profile, error) {
	return repository.profile, nil
}

func (repository *fakeMobileRepository) UpdateProfileByUserID(context.Context, uuid.UUID, ProfilePatch) (Profile, error) {
	return Profile{}, nil
}

func (repository *fakeMobileRepository) SetStatusByUserID(_ context.Context, _ uuid.UUID, status domain.DriverStatus) (Profile, error) {
	profile := repository.statusProfile
	profile.Status = status
	return profile, nil
}

func (repository *fakeMobileRepository) ListCarsByUserID(context.Context, uuid.UUID) ([]domain.Car, error) {
	return nil, nil
}

func (repository *fakeMobileRepository) GetCurrentOrderByUserID(context.Context, uuid.UUID) (CurrentOrder, error) {
	if repository.currentOrder.OrderID == uuid.Nil {
		return CurrentOrder{}, ErrCurrentOrderNotFound
	}
	return repository.currentOrder, nil
}

func (repository *fakeMobileRepository) GetOrderByUserID(context.Context, uuid.UUID, uuid.UUID) (CurrentOrder, error) {
	return CurrentOrder{}, ErrCurrentOrderNotFound
}

func (repository *fakeMobileRepository) ListOrderHistoryByUserID(context.Context, uuid.UUID, int) ([]CurrentOrder, error) {
	return nil, nil
}

func (repository *fakeMobileRepository) CompleteOrderByUserID(_ context.Context, _ uuid.UUID, orderID uuid.UUID) (CurrentOrder, error) {
	repository.completedOrderID = orderID
	return repository.completionOrderResult, nil
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
	return repository.transitionOrderResult, nil
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

type fakePresenceStore struct {
	markedOnline  bool
	markedOffline bool
}

func (store *fakePresenceStore) MarkOnline(context.Context, uuid.UUID, uuid.UUID, time.Duration) error {
	store.markedOnline = true
	return nil
}

func (store *fakePresenceStore) MarkOffline(context.Context, uuid.UUID, uuid.UUID) error {
	store.markedOffline = true
	return nil
}

type fakeRealtimeGateway struct {
	presenceDriverID uuid.UUID
	presencePayload  any
	passengerEvent   string
	passengerPayload any
}

type fakePassengerNotifier struct {
	notifications []PassengerNotification
}

func (notifier *fakePassengerNotifier) NotifyPassenger(_ context.Context, _ uuid.UUID, notification PassengerNotification) error {
	notifier.notifications = append(notifier.notifications, notification)
	return nil
}

func (gateway *fakeRealtimeGateway) SendDriverPresenceToTaxiPark(_ context.Context, driverID uuid.UUID, payload any) error {
	gateway.presenceDriverID = driverID
	gateway.presencePayload = payload
	return nil
}

func (gateway *fakeRealtimeGateway) SendDriverLocationToTaxiPark(context.Context, uuid.UUID, any) error {
	return nil
}

func (gateway *fakeRealtimeGateway) SendToDriver(context.Context, uuid.UUID, string, any) error {
	return nil
}

func (gateway *fakeRealtimeGateway) SendToPassenger(_ context.Context, _ uuid.UUID, eventName string, payload any) error {
	gateway.passengerEvent = eventName
	gateway.passengerPayload = payload
	return nil
}

func (gateway *fakeRealtimeGateway) SendToTaxiParkByOrder(context.Context, uuid.UUID, string, any) error {
	return nil
}

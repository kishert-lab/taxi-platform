package order

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/develoop/taxi-platform/internal/domain"
)

func TestServiceRejectsInvalidTransition(t *testing.T) {
	t.Parallel()

	order := testOrder(domain.OrderStatusDriverAssigned)
	repository := &fakeRepository{order: order}
	service := NewService(NewServiceParams{Repository: repository})

	_, err := service.Transition(context.Background(), TransitionCommand{OrderID: order.ID, ToStatus: domain.OrderStatusCompleted})
	if !errors.Is(err, domain.ErrInvalidOrderStatusTransition) {
		t.Fatalf("expected invalid transition, got %v", err)
	}
}

func TestServicePersistsEventAndPublishesConsistentState(t *testing.T) {
	t.Parallel()

	order := testOrder(domain.OrderStatusDriverArriving)
	driverID := uuid.New()
	order.DriverID = &driverID
	repository := &fakeRepository{order: order}
	realtime := &fakeRealtimePublisher{}
	service := NewService(NewServiceParams{Repository: repository, RealtimePublisher: realtime})

	updatedOrder, err := service.Transition(context.Background(), TransitionCommand{OrderID: order.ID, ToStatus: domain.OrderStatusDriverWaiting})
	if err != nil {
		t.Fatalf("transition: %v", err)
	}

	if updatedOrder.Status != domain.OrderStatusDriverWaiting {
		t.Fatalf("expected driver_waiting, got %s", updatedOrder.Status)
	}
	if len(repository.events) != 1 {
		t.Fatalf("expected order event to be persisted")
	}
	if !realtime.sentToPassenger || !realtime.sentToDriver {
		t.Fatal("expected state to be published to passenger and driver")
	}
}

func TestServiceReportsConcurrentUpdate(t *testing.T) {
	t.Parallel()

	order := testOrder(domain.OrderStatusSearching)
	repository := &fakeRepository{order: order, forceConflict: true}
	service := NewService(NewServiceParams{Repository: repository})

	_, err := service.Transition(context.Background(), TransitionCommand{OrderID: order.ID, ToStatus: domain.OrderStatusFailed})
	if !errors.Is(err, ErrOrderConcurrentUpdate) {
		t.Fatalf("expected concurrent update error, got %v", err)
	}
}

func TestPassengerCancellationStopsDispatch(t *testing.T) {
	t.Parallel()

	order := testOrder(domain.OrderStatusSearching)
	repository := &fakeRepository{order: order}
	dispatchController := &fakeDispatchController{}
	service := NewService(NewServiceParams{Repository: repository, DispatchController: dispatchController})

	_, err := service.Transition(context.Background(), TransitionCommand{OrderID: order.ID, ToStatus: domain.OrderStatusCancelled, Reason: "passenger cancelled"})
	if err != nil {
		t.Fatalf("cancel order: %v", err)
	}
	if !dispatchController.stopped {
		t.Fatal("expected dispatch to be stopped on cancellation")
	}
}

func TestReconnectCurrentOrderRestoresState(t *testing.T) {
	t.Parallel()

	passengerID := uuid.New()
	order := testOrder(domain.OrderStatusDriverWaiting)
	order.PassengerID = passengerID
	repository := &fakeRepository{order: order}
	service := NewService(NewServiceParams{Repository: repository})

	currentOrder, err := service.CurrentForPassenger(context.Background(), passengerID)
	if err != nil {
		t.Fatalf("current order: %v", err)
	}
	if currentOrder.Status != domain.OrderStatusDriverWaiting {
		t.Fatalf("expected restored state driver_waiting, got %s", currentOrder.Status)
	}
}

func testOrder(status domain.OrderStatus) domain.Order {
	return domain.Order{ID: uuid.New(), PassengerID: uuid.New(), Status: status, Version: 1}
}

type fakeRepository struct {
	order         domain.Order
	events        []OrderEvent
	forceConflict bool
}

func (repository *fakeRepository) GetOrderByID(_ context.Context, _ uuid.UUID) (domain.Order, error) {
	return repository.order, nil
}

func (repository *fakeRepository) GetCurrentOrderByPassengerID(_ context.Context, _ uuid.UUID) (domain.Order, error) {
	return repository.order, nil
}

func (repository *fakeRepository) GetCurrentOrderByDriverID(_ context.Context, _ uuid.UUID) (domain.Order, error) {
	return repository.order, nil
}

func (repository *fakeRepository) TransitionOrderStatus(_ context.Context, transition domain.OrderTransition) (domain.Order, bool, error) {
	if repository.forceConflict {
		return domain.Order{}, false, nil
	}
	repository.order.Status = transition.ToStatus
	repository.order.Version++
	return repository.order, true, nil
}

func (repository *fakeRepository) AddStateEvent(_ context.Context, event OrderEvent) error {
	repository.events = append(repository.events, event)
	return nil
}

type fakeDispatchController struct {
	stopped bool
}

func (controller *fakeDispatchController) StopDispatch(_ context.Context, _ uuid.UUID) error {
	controller.stopped = true
	return nil
}

type fakeRealtimePublisher struct {
	sentToPassenger bool
	sentToDriver    bool
}

func (publisher *fakeRealtimePublisher) SendToDriver(_ context.Context, _ uuid.UUID, _ string, _ any) error {
	publisher.sentToDriver = true
	return nil
}

func (publisher *fakeRealtimePublisher) SendToPassenger(_ context.Context, _ uuid.UUID, _ string, _ any) error {
	publisher.sentToPassenger = true
	return nil
}

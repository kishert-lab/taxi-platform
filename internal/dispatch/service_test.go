package dispatch

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/develoop/taxi-platform/internal/domain"
)

func TestProcessTaskOffersInitialRadiusToMaxFiveNearestDrivers(t *testing.T) {
	t.Parallel()

	order := testOrder()
	order.Status = domain.OrderStatusSearching
	driverIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	orderRepository := &fakeOrderRepository{order: order}
	driverSearchRepository := &fakeDriverSearchRepository{candidates: candidatesFromIDs(driverIDs)}
	offerStore := newFakeOfferStore()
	realtimeGateway := &fakeRealtimeGateway{}
	timeoutQueue := &fakeTimeoutQueue{}
	service := newTestService(orderRepository, driverSearchRepository, offerStore, &fakeTaskQueue{}, timeoutQueue, realtimeGateway)

	result, err := service.ProcessTask(context.Background(), DispatchTask{OrderID: order.ID, Attempt: 0, QueuedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("process dispatch task: %v", err)
	}

	if result.RadiusMeters != 1000 {
		t.Fatalf("expected initial radius 1000, got %d", result.RadiusMeters)
	}
	if len(result.OfferedDriverIDs) != 5 {
		t.Fatalf("expected 5 offered drivers, got %d", len(result.OfferedDriverIDs))
	}
	if driverSearchRepository.lastQuery.RadiusMeters != 1000 {
		t.Fatalf("expected search radius 1000, got %d", driverSearchRepository.lastQuery.RadiusMeters)
	}
	if driverSearchRepository.lastQuery.Limit != 5 {
		t.Fatalf("expected search limit 5, got %d", driverSearchRepository.lastQuery.Limit)
	}
	if driverSearchRepository.lastQuery.LocationMaxAge != 30*time.Second {
		t.Fatalf("expected stale location cutoff 30s, got %s", driverSearchRepository.lastQuery.LocationMaxAge)
	}
	if len(realtimeGateway.driverEvents) != 5 {
		t.Fatalf("expected 5 realtime driver offers, got %d", len(realtimeGateway.driverEvents))
	}
	if len(timeoutQueue.scheduled) != 1 {
		t.Fatalf("expected offer timeout to be scheduled")
	}
}

func TestHandleOfferTimeoutPublishesNextRadiusAttempt(t *testing.T) {
	t.Parallel()

	order := testOrder()
	order.Status = domain.OrderStatusSearching
	previousDriverID := uuid.New()
	orderRepository := &fakeOrderRepository{order: order}
	offerStore := newFakeOfferStore()
	offerStore.driverIDs[order.ID] = []uuid.UUID{previousDriverID}
	taskQueue := &fakeTaskQueue{}
	realtimeGateway := &fakeRealtimeGateway{}
	service := newTestService(orderRepository, &fakeDriverSearchRepository{}, offerStore, taskQueue, &fakeTimeoutQueue{}, realtimeGateway)

	err := service.HandleOfferTimeout(context.Background(), DispatchTask{OrderID: order.ID, Attempt: 0, QueuedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("handle offer timeout: %v", err)
	}

	if len(taskQueue.published) != 1 {
		t.Fatalf("expected next dispatch attempt to be published")
	}
	if taskQueue.published[0].Attempt != 1 {
		t.Fatalf("expected next attempt 1, got %d", taskQueue.published[0].Attempt)
	}
	if !offerStore.removed {
		t.Fatalf("expected timed out offers to be removed")
	}
	if !realtimeGateway.hasDriverEvent(previousDriverID, EventOrderExpired) {
		t.Fatalf("expected driver offer expiration event")
	}
}

func TestProcessTaskUsesConfiguredRadiusAttempts(t *testing.T) {
	t.Parallel()

	order := testOrder()
	order.Status = domain.OrderStatusSearching
	orderRepository := &fakeOrderRepository{order: order}
	driverSearchRepository := &fakeDriverSearchRepository{candidates: candidatesFromIDs([]uuid.UUID{uuid.New()})}
	service := newTestService(orderRepository, driverSearchRepository, newFakeOfferStore(), &fakeTaskQueue{}, &fakeTimeoutQueue{}, &fakeRealtimeGateway{})

	result, err := service.ProcessTask(context.Background(), DispatchTask{OrderID: order.ID, Attempt: 2, QueuedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("process dispatch attempt: %v", err)
	}
	if result.RadiusMeters != 5000 {
		t.Fatalf("expected attempt 2 radius 5000, got %d", result.RadiusMeters)
	}
}

func TestProcessTaskDoesNotDuplicateActiveOffers(t *testing.T) {
	t.Parallel()

	order := testOrder()
	order.Status = domain.OrderStatusSearching
	driverID := uuid.New()
	orderRepository := &fakeOrderRepository{order: order}
	driverSearchRepository := &fakeDriverSearchRepository{candidates: candidatesFromIDs([]uuid.UUID{driverID})}
	offerStore := newFakeOfferStore()
	offerStore.saveTestOffer(order.ID, driverID)
	realtimeGateway := &fakeRealtimeGateway{}
	service := newTestService(orderRepository, driverSearchRepository, offerStore, &fakeTaskQueue{}, &fakeTimeoutQueue{}, realtimeGateway)

	result, err := service.ProcessTask(context.Background(), DispatchTask{OrderID: order.ID, Attempt: 0, QueuedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("process duplicate dispatch task: %v", err)
	}

	if len(result.OfferedDriverIDs) != 1 {
		t.Fatalf("expected existing offer to be reported")
	}
	if len(realtimeGateway.driverEvents) != 0 {
		t.Fatalf("expected no duplicate realtime offer, got %d events", len(realtimeGateway.driverEvents))
	}
}

func TestAcceptOfferOnlyOneDriverCanAssignOrder(t *testing.T) {
	t.Parallel()

	order := testOrder()
	order.Status = domain.OrderStatusSearching
	firstDriverID := uuid.New()
	secondDriverID := uuid.New()
	orderRepository := &fakeOrderRepository{order: order}
	offerStore := newFakeOfferStore()
	offerStore.saveTestOffer(order.ID, firstDriverID)
	offerStore.saveTestOffer(order.ID, secondDriverID)
	lockManager := &fakeLockManager{}
	service := newTestService(orderRepository, &fakeDriverSearchRepository{}, offerStore, &fakeTaskQueue{}, &fakeTimeoutQueue{}, &fakeRealtimeGateway{})
	service.lockManager = lockManager

	if err := service.AcceptOffer(context.Background(), order.ID, firstDriverID); err != nil {
		t.Fatalf("first accept offer: %v", err)
	}
	err := service.AcceptOffer(context.Background(), order.ID, secondDriverID)
	if !errors.Is(err, ErrOrderAlreadyAssigned) {
		t.Fatalf("expected order already assigned, got %v", err)
	}
}

func TestAcceptOfferAssignsDriverAndCancelsRemainingOffers(t *testing.T) {
	t.Parallel()

	order := testOrder()
	order.Status = domain.OrderStatusSearching
	acceptedDriverID := uuid.New()
	otherDriverID := uuid.New()
	orderRepository := &fakeOrderRepository{order: order}
	driverStateRepository := &fakeDriverStateRepository{}
	offerStore := newFakeOfferStore()
	offerStore.saveTestOffer(order.ID, acceptedDriverID)
	offerStore.saveTestOffer(order.ID, otherDriverID)
	realtimeGateway := &fakeRealtimeGateway{}
	service := newTestService(orderRepository, &fakeDriverSearchRepository{}, offerStore, &fakeTaskQueue{}, &fakeTimeoutQueue{}, realtimeGateway)
	service.driverStateRepository = driverStateRepository

	if err := service.AcceptOffer(context.Background(), order.ID, acceptedDriverID); err != nil {
		t.Fatalf("accept offer: %v", err)
	}

	if orderRepository.order.DriverID == nil || *orderRepository.order.DriverID != acceptedDriverID {
		t.Fatalf("expected accepted driver to be assigned")
	}
	if driverStateRepository.busyDriverID != acceptedDriverID {
		t.Fatalf("expected driver to be marked busy")
	}
	if !offerStore.removed {
		t.Fatalf("expected offers to be removed")
	}
	if !realtimeGateway.hasDriverEvent(otherDriverID, EventOrderCancelled) {
		t.Fatalf("expected remaining driver offer to be cancelled")
	}
	if !realtimeGateway.hasPassengerEvent(order.PassengerID, EventPassengerDriverAssigned) {
		t.Fatalf("expected passenger to receive driver assigned event")
	}
}

func newTestService(
	orderRepository *fakeOrderRepository,
	driverSearchRepository *fakeDriverSearchRepository,
	offerStore *fakeOfferStore,
	taskQueue *fakeTaskQueue,
	timeoutQueue *fakeTimeoutQueue,
	realtimeGateway *fakeRealtimeGateway,
) *Service {
	return NewService(NewServiceParams{
		OrderRepository:        orderRepository,
		DriverSearchRepository: driverSearchRepository,
		DriverStateRepository:  &fakeDriverStateRepository{},
		OfferStore:             offerStore,
		TaskQueue:              taskQueue,
		TimeoutQueue:           timeoutQueue,
		LockManager:            &fakeLockManager{},
		RealtimeGateway:        realtimeGateway,
		Logger:                 zap.NewNop(),
		Config: Config{
			RadiusAttemptsMeters: []int{1000, 3000, 5000, 10000},
			MaxDriversPerOffer:   5,
			DriverLocationMaxAge: 30 * time.Second,
			OfferTTL:             15 * time.Second,
			AcceptLockTTL:        30 * time.Second,
			WorkerPollTimeout:    time.Second,
			RecoveryInterval:     time.Second,
		},
	})
}

func testOrder() domain.Order {
	return domain.Order{
		ID:             uuid.New(),
		PassengerID:    uuid.New(),
		CityID:         uuid.New(),
		Status:         domain.OrderStatusCreated,
		PickupAddress:  "Lenina 1",
		PickupLocation: domain.Coordinates{Latitude: 56.838011, Longitude: 60.597465},
	}
}

func candidatesFromIDs(driverIDs []uuid.UUID) []DriverCandidate {
	candidates := make([]DriverCandidate, 0, len(driverIDs))
	for index, driverID := range driverIDs {
		candidates = append(candidates, DriverCandidate{
			DriverID:       driverID,
			DistanceMeters: float64(index+1) * 100,
			Location:       domain.Coordinates{Latitude: 56.838011, Longitude: 60.597465},
		})
	}
	return candidates
}

type fakeOrderRepository struct {
	order domain.Order
}

func (repository *fakeOrderRepository) GetOrderByID(_ context.Context, _ uuid.UUID) (domain.Order, error) {
	return repository.order, nil
}

func (repository *fakeOrderRepository) MarkOrderSearching(_ context.Context, _ uuid.UUID) error {
	repository.order.Status = domain.OrderStatusSearching
	return nil
}

func (repository *fakeOrderRepository) AssignDriver(_ context.Context, _ uuid.UUID, driverID uuid.UUID, acceptedAt time.Time) (bool, error) {
	if repository.order.DriverID != nil {
		return false, nil
	}
	repository.order.DriverID = &driverID
	repository.order.Status = domain.OrderStatusDriverAssigned
	repository.order.AcceptedAt = &acceptedAt
	return true, nil
}

func (repository *fakeOrderRepository) IncrementDispatchAttempt(_ context.Context, _ uuid.UUID) error {
	repository.order.DispatchAttempt++
	return nil
}

func (repository *fakeOrderRepository) FailOrder(_ context.Context, _ uuid.UUID, _ string) error {
	repository.order.Status = domain.OrderStatusFailed
	return nil
}

func (repository *fakeOrderRepository) AddOrderEvent(_ context.Context, _ OrderEvent) error {
	return nil
}

type fakeDriverSearchRepository struct {
	candidates []DriverCandidate
	lastQuery  NearestDriversQuery
}

func (repository *fakeDriverSearchRepository) FindNearestOnlineDrivers(_ context.Context, query NearestDriversQuery) ([]DriverCandidate, error) {
	repository.lastQuery = query
	return repository.candidates, nil
}

type fakeDriverStateRepository struct {
	busyDriverID uuid.UUID
}

func (repository *fakeDriverStateRepository) MarkDriverBusy(_ context.Context, driverID uuid.UUID) error {
	repository.busyDriverID = driverID
	return nil
}

type fakeOfferStore struct {
	driverIDs map[uuid.UUID][]uuid.UUID
	offers    map[string]OrderOffer
	removed   bool
}

func newFakeOfferStore() *fakeOfferStore {
	return &fakeOfferStore{driverIDs: make(map[uuid.UUID][]uuid.UUID), offers: make(map[string]OrderOffer)}
}

func (store *fakeOfferStore) SaveOffer(_ context.Context, offer OrderOffer, _ time.Duration) error {
	store.driverIDs[offer.OrderID] = append(store.driverIDs[offer.OrderID], offer.DriverID)
	store.offers[offerKey(offer.OrderID, offer.DriverID)] = offer
	return nil
}

func (store *fakeOfferStore) GetOffer(_ context.Context, orderID uuid.UUID, driverID uuid.UUID) (OrderOffer, bool, error) {
	offer, ok := store.offers[offerKey(orderID, driverID)]
	return offer, ok, nil
}

func (store *fakeOfferStore) ListOfferedDriverIDs(_ context.Context, orderID uuid.UUID) ([]uuid.UUID, error) {
	return append([]uuid.UUID(nil), store.driverIDs[orderID]...), nil
}

func (store *fakeOfferStore) RemoveOffers(_ context.Context, orderID uuid.UUID) error {
	delete(store.driverIDs, orderID)
	for key := range store.offers {
		delete(store.offers, key)
	}
	store.removed = true
	return nil
}

func (store *fakeOfferStore) saveTestOffer(orderID uuid.UUID, driverID uuid.UUID) {
	_ = store.SaveOffer(context.Background(), OrderOffer{OrderID: orderID, DriverID: driverID, ExpiresAt: time.Now().Add(time.Minute), CreatedAt: time.Now()}, time.Minute)
}

func offerKey(orderID uuid.UUID, driverID uuid.UUID) string {
	return orderID.String() + ":" + driverID.String()
}

type fakeTaskQueue struct {
	published []DispatchTask
}

func (queue *fakeTaskQueue) Publish(_ context.Context, task DispatchTask) error {
	queue.published = append(queue.published, task)
	return nil
}

func (queue *fakeTaskQueue) Consume(_ context.Context, _ time.Duration) (DispatchTask, bool, error) {
	if len(queue.published) == 0 {
		return DispatchTask{}, false, nil
	}
	task := queue.published[0]
	queue.published = queue.published[1:]
	return task, true, nil
}

type fakeTimeoutQueue struct {
	scheduled []DispatchTask
}

func (queue *fakeTimeoutQueue) Schedule(_ context.Context, task DispatchTask, _ time.Time) error {
	queue.scheduled = append(queue.scheduled, task)
	return nil
}

func (queue *fakeTimeoutQueue) Due(_ context.Context, _ time.Time, _ int) ([]DispatchTask, error) {
	return append([]DispatchTask(nil), queue.scheduled...), nil
}

func (queue *fakeTimeoutQueue) Remove(_ context.Context, _ DispatchTask) error {
	queue.scheduled = nil
	return nil
}

type fakeLockManager struct{}

func (manager *fakeLockManager) Acquire(_ context.Context, _ string, _ time.Duration) (Lock, bool, error) {
	return fakeLock{}, true, nil
}

type fakeLock struct{}

func (lock fakeLock) Release(_ context.Context) error {
	return nil
}

type fakeRealtimeGateway struct {
	driverEvents    []fakeRealtimeEvent
	passengerEvents []fakeRealtimeEvent
}

type fakeRealtimeEvent struct {
	targetID uuid.UUID
	name     string
}

func (gateway *fakeRealtimeGateway) SendToDriver(_ context.Context, driverID uuid.UUID, eventName string, _ any) error {
	gateway.driverEvents = append(gateway.driverEvents, fakeRealtimeEvent{targetID: driverID, name: eventName})
	return nil
}

func (gateway *fakeRealtimeGateway) SendToPassenger(_ context.Context, passengerID uuid.UUID, eventName string, _ any) error {
	gateway.passengerEvents = append(gateway.passengerEvents, fakeRealtimeEvent{targetID: passengerID, name: eventName})
	return nil
}

func (gateway *fakeRealtimeGateway) hasDriverEvent(driverID uuid.UUID, eventName string) bool {
	for _, event := range gateway.driverEvents {
		if event.targetID == driverID && event.name == eventName {
			return true
		}
	}
	return false
}

func (gateway *fakeRealtimeGateway) hasPassengerEvent(passengerID uuid.UUID, eventName string) bool {
	for _, event := range gateway.passengerEvents {
		if event.targetID == passengerID && event.name == eventName {
			return true
		}
	}
	return false
}

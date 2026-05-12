package dispatch

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/develoop/taxi-platform/internal/domain"
)

type Service struct {
	orderRepository        OrderRepository
	driverSearchRepository DriverSearchRepository
	driverStateRepository  DriverStateRepository
	offerStore             OfferStore
	taskQueue              TaskQueue
	timeoutQueue           TimeoutQueue
	lockManager            LockManager
	realtimeGateway        RealtimeGateway
	metrics                Metrics
	logger                 *zap.Logger
	config                 Config
}

type NewServiceParams struct {
	OrderRepository        OrderRepository
	DriverSearchRepository DriverSearchRepository
	DriverStateRepository  DriverStateRepository
	OfferStore             OfferStore
	TaskQueue              TaskQueue
	TimeoutQueue           TimeoutQueue
	LockManager            LockManager
	RealtimeGateway        RealtimeGateway
	Metrics                Metrics
	Logger                 *zap.Logger
	Config                 Config
}

var (
	ErrDispatchAlreadyFinished = errors.New("dispatch already finished")
	ErrOfferNotAccepted        = errors.New("offer was not accepted")
	ErrOrderAlreadyAssigned    = errors.New("order already assigned")
)

func NewService(params NewServiceParams) *Service {
	return &Service{
		orderRepository:        params.OrderRepository,
		driverSearchRepository: params.DriverSearchRepository,
		driverStateRepository:  params.DriverStateRepository,
		offerStore:             params.OfferStore,
		taskQueue:              params.TaskQueue,
		timeoutQueue:           params.TimeoutQueue,
		lockManager:            params.LockManager,
		realtimeGateway:        params.RealtimeGateway,
		metrics:                params.Metrics,
		logger:                 loggerOrNop(params.Logger),
		config:                 normalizeConfig(params.Config),
	}
}

func (service *Service) EnqueueOrder(ctx context.Context, orderID uuid.UUID) error {
	if err := service.orderRepository.MarkOrderSearching(ctx, orderID); err != nil {
		return fmt.Errorf("mark order searching before enqueue dispatch: %w", err)
	}
	task := DispatchTask{OrderID: orderID, Attempt: 0, QueuedAt: time.Now().UTC()}
	if err := service.taskQueue.Publish(ctx, task); err != nil {
		return fmt.Errorf("publish dispatch task: %w", err)
	}

	service.logger.Info("dispatch start queued", zap.String("order_id", orderID.String()), zap.Int("attempt", task.Attempt))
	return nil
}

func (service *Service) ProcessTask(ctx context.Context, task DispatchTask) (DispatchResult, error) {
	startedAt := time.Now()
	defer func() {
		if service.metrics != nil {
			service.metrics.ObserveDispatchDuration(time.Since(startedAt))
		}
	}()

	order, err := service.orderRepository.GetOrderByID(ctx, task.OrderID)
	if err != nil {
		return DispatchResult{}, fmt.Errorf("get order for dispatch: %w", err)
	}
	if order.Status.IsTerminal() || order.Status == domain.OrderStatusDriverAssigned {
		return DispatchResult{}, fmt.Errorf("%w: order status %s", ErrDispatchAlreadyFinished, order.Status)
	}
	if order.Status != domain.OrderStatusSearching {
		if err := service.orderRepository.MarkOrderSearching(ctx, task.OrderID); err != nil {
			return DispatchResult{}, fmt.Errorf("mark order searching: %w", err)
		}
		order.Status = domain.OrderStatusSearching
	}

	radiusMeters, ok := service.radiusForAttempt(task.Attempt)
	if !ok {
		return service.failNoDriversFound(ctx, order, radiusMeters)
	}
	if service.metrics != nil {
		service.metrics.ObserveDispatchRadiusAttempt(radiusMeters)
	}

	candidates, err := service.driverSearchRepository.FindNearestOnlineDrivers(ctx, NearestDriversQuery{
		CityID:         order.CityID,
		Pickup:         order.PickupLocation,
		RadiusMeters:   radiusMeters,
		Limit:          service.config.MaxDriversPerOffer,
		ExcludeIDs:     nil,
		LocationMaxAge: service.config.DriverLocationMaxAge,
	})
	if err != nil {
		return DispatchResult{}, fmt.Errorf("find nearest online drivers: %w", err)
	}
	if len(candidates) == 0 {
		if err := service.scheduleNextAttempt(ctx, task); err != nil {
			return DispatchResult{}, err
		}
		service.logger.Info("no drivers in dispatch radius", zap.String("order_id", task.OrderID.String()), zap.Int("radius_meters", radiusMeters), zap.Int("attempt", task.Attempt))
		return DispatchResult{OrderID: task.OrderID, Status: domain.OrderStatusSearching, Attempt: task.Attempt, RadiusMeters: radiusMeters, NextRetryAfter: service.config.OfferTTL}, nil
	}

	offeredDriverIDs, err := service.offerDrivers(ctx, order, task.Attempt, candidates, radiusMeters)
	if err != nil {
		return DispatchResult{}, err
	}
	if err := service.timeoutQueue.Schedule(ctx, task, time.Now().UTC().Add(service.config.OfferTTL)); err != nil {
		return DispatchResult{}, fmt.Errorf("schedule offer timeout: %w", err)
	}

	return DispatchResult{
		OrderID:          task.OrderID,
		Status:           domain.OrderStatusSearching,
		Attempt:          task.Attempt,
		RadiusMeters:     radiusMeters,
		OfferedDriverIDs: offeredDriverIDs,
		NextRetryAfter:   service.config.OfferTTL,
	}, nil
}

func (service *Service) HandleOfferTimeout(ctx context.Context, task DispatchTask) error {
	order, err := service.orderRepository.GetOrderByID(ctx, task.OrderID)
	if err != nil {
		return fmt.Errorf("get order for offer timeout: %w", err)
	}
	if order.Status != domain.OrderStatusSearching {
		if err := service.timeoutQueue.Remove(ctx, task); err != nil {
			return fmt.Errorf("remove stale timeout task: %w", err)
		}
		return nil
	}

	offeredDriverIDs, err := service.offerStore.ListOfferedDriverIDs(ctx, task.OrderID)
	if err != nil {
		return fmt.Errorf("list timed out offers: %w", err)
	}
	for _, driverID := range offeredDriverIDs {
		if err := service.realtimeGateway.SendToDriver(ctx, driverID, EventOrderExpired, map[string]any{"order_id": task.OrderID}); err != nil {
			return fmt.Errorf("send offer expired event: %w", err)
		}
	}
	if err := service.offerStore.RemoveOffers(ctx, task.OrderID); err != nil {
		return fmt.Errorf("remove timed out offers: %w", err)
	}
	if err := service.timeoutQueue.Remove(ctx, task); err != nil {
		return fmt.Errorf("remove timeout task: %w", err)
	}

	service.logger.Info("offer timeout", zap.String("order_id", task.OrderID.String()), zap.Int("attempt", task.Attempt))
	return service.scheduleNextAttempt(ctx, task)
}

func (service *Service) AcceptOffer(ctx context.Context, orderID uuid.UUID, driverID uuid.UUID) error {
	offer, exists, err := service.offerStore.GetOffer(ctx, orderID, driverID)
	if err != nil {
		return fmt.Errorf("get active order offer: %w", err)
	}
	if !exists || time.Now().UTC().After(offer.ExpiresAt) {
		order, orderErr := service.orderRepository.GetOrderByID(ctx, orderID)
		if orderErr != nil {
			return fmt.Errorf("get order after missing active offer: %w", orderErr)
		}
		if order.Status == domain.OrderStatusDriverAssigned || order.DriverID != nil {
			return fmt.Errorf("%w: %w", ErrOfferNotAccepted, ErrOrderAlreadyAssigned)
		}
		return fmt.Errorf("%w: active offer not found", ErrOfferNotAccepted)
	}

	lock, acquired, err := service.lockManager.Acquire(ctx, fmt.Sprintf("order:%s:accept_lock", orderID), service.config.AcceptLockTTL)
	if err != nil {
		return fmt.Errorf("acquire order accept lock: %w", err)
	}
	if !acquired {
		return fmt.Errorf("%w: %w", ErrOfferNotAccepted, ErrOrderAlreadyAssigned)
	}
	defer func() {
		if releaseErr := lock.Release(ctx); releaseErr != nil {
			service.logger.Error("release order accept lock", zap.Error(releaseErr), zap.String("order_id", orderID.String()))
		}
	}()

	acceptedAt := time.Now().UTC()
	accepted, err := service.orderRepository.AssignDriver(ctx, orderID, driverID, acceptedAt)
	if err != nil {
		return fmt.Errorf("assign driver to order: %w", err)
	}
	if !accepted {
		return fmt.Errorf("%w: %w", ErrOfferNotAccepted, ErrOrderAlreadyAssigned)
	}

	if service.metrics != nil {
		service.metrics.ObserveDriverAcceptTime(acceptedAt.Sub(offer.CreatedAt))
	}
	if err := service.driverStateRepository.MarkDriverBusy(ctx, driverID); err != nil {
		return fmt.Errorf("mark driver busy after accept: %w", err)
	}
	if err := service.cancelRemainingOffers(ctx, orderID, driverID); err != nil {
		return err
	}

	order, err := service.orderRepository.GetOrderByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("get accepted order: %w", err)
	}
	if err := service.realtimeGateway.SendToDriver(ctx, driverID, EventOrderAssigned, map[string]any{"order_id": orderID}); err != nil {
		return fmt.Errorf("send order assigned event to driver: %w", err)
	}
	if err := service.realtimeGateway.SendToPassenger(ctx, order.PassengerID, EventPassengerDriverAssigned, map[string]any{"order_id": orderID, "driver_id": driverID}); err != nil {
		return fmt.Errorf("send driver assigned event to passenger: %w", err)
	}

	service.logger.Info("offer accepted", zap.String("order_id", orderID.String()), zap.String("driver_id", driverID.String()))
	return nil
}

func (service *Service) offerDrivers(ctx context.Context, order domain.Order, attempt int, candidates []DriverCandidate, radiusMeters int) ([]uuid.UUID, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(service.config.OfferTTL)
	if len(candidates) > service.config.MaxDriversPerOffer {
		candidates = candidates[:service.config.MaxDriversPerOffer]
	}
	offeredDriverIDs := make([]uuid.UUID, 0, len(candidates))

	for _, candidate := range candidates {
		existingOffer, exists, err := service.offerStore.GetOffer(ctx, order.ID, candidate.DriverID)
		if err != nil {
			return nil, fmt.Errorf("check existing order offer: %w", err)
		}
		if exists && time.Now().UTC().Before(existingOffer.ExpiresAt) {
			offeredDriverIDs = append(offeredDriverIDs, candidate.DriverID)
			continue
		}

		offer := OrderOffer{
			OrderID:        order.ID,
			DriverID:       candidate.DriverID,
			Attempt:        attempt,
			RadiusMeters:   radiusMeters,
			DistanceMeters: candidate.DistanceMeters,
			ExpiresAt:      expiresAt,
			CreatedAt:      now,
		}
		if err := service.offerStore.SaveOffer(ctx, offer, service.config.OfferTTL); err != nil {
			return nil, fmt.Errorf("save order offer: %w", err)
		}
		if err := service.realtimeGateway.SendToDriver(ctx, candidate.DriverID, EventOrderOffer, offerPayload(order, offer)); err != nil {
			return nil, fmt.Errorf("send order offer to driver: %w", err)
		}
		offeredDriverIDs = append(offeredDriverIDs, candidate.DriverID)
		service.logger.Info("offer sent", zap.String("order_id", order.ID.String()), zap.String("driver_id", candidate.DriverID.String()), zap.Int("radius_meters", radiusMeters))
	}

	if err := service.orderRepository.AddOrderEvent(ctx, OrderEvent{
		OrderID:   order.ID,
		EventType: domain.OrderEventOffer,
		Payload: map[string]any{
			"attempt":               attempt,
			"radius_meters":         radiusMeters,
			"offered_driver_ids":    offeredDriverIDs,
			"expires_at":            expiresAt,
			"max_drivers_per_offer": service.config.MaxDriversPerOffer,
		},
		CreatedAt: now,
	}); err != nil {
		return nil, fmt.Errorf("add order offer event: %w", err)
	}

	return offeredDriverIDs, nil
}

func (service *Service) cancelRemainingOffers(ctx context.Context, orderID uuid.UUID, acceptedDriverID uuid.UUID) error {
	offeredDriverIDs, err := service.offerStore.ListOfferedDriverIDs(ctx, orderID)
	if err != nil {
		return fmt.Errorf("list offered drivers for cancellation: %w", err)
	}
	for _, offeredDriverID := range offeredDriverIDs {
		if offeredDriverID == acceptedDriverID {
			continue
		}
		if err := service.realtimeGateway.SendToDriver(ctx, offeredDriverID, EventOrderCancelled, map[string]any{"order_id": orderID}); err != nil {
			return fmt.Errorf("send offer cancelled event: %w", err)
		}
	}
	if err := service.offerStore.RemoveOffers(ctx, orderID); err != nil {
		return fmt.Errorf("remove accepted order offers: %w", err)
	}

	return nil
}

func (service *Service) scheduleNextAttempt(ctx context.Context, task DispatchTask) error {
	nextTask := DispatchTask{OrderID: task.OrderID, Attempt: task.Attempt + 1, QueuedAt: time.Now().UTC()}
	if _, ok := service.radiusForAttempt(nextTask.Attempt); !ok {
		order, err := service.orderRepository.GetOrderByID(ctx, task.OrderID)
		if err != nil {
			return fmt.Errorf("get order before no drivers found: %w", err)
		}
		_, err = service.failNoDriversFound(ctx, order, service.lastRadius())
		return err
	}
	if err := service.taskQueue.Publish(ctx, nextTask); err != nil {
		return fmt.Errorf("publish next dispatch attempt: %w", err)
	}
	service.logger.Info("radius expansion queued", zap.String("order_id", task.OrderID.String()), zap.Int("attempt", nextTask.Attempt))
	return nil
}

func (service *Service) failNoDriversFound(ctx context.Context, order domain.Order, radiusMeters int) (DispatchResult, error) {
	if err := service.orderRepository.FailOrder(ctx, order.ID, "no online drivers found in dispatch radius"); err != nil {
		return DispatchResult{}, fmt.Errorf("fail order after dispatch exhaustion: %w", err)
	}
	if err := service.realtimeGateway.SendToPassenger(ctx, order.PassengerID, EventOrderNoDriversFound, map[string]any{"order_id": order.ID}); err != nil {
		return DispatchResult{}, fmt.Errorf("send no drivers found event to passenger: %w", err)
	}
	if service.metrics != nil {
		service.metrics.IncrementFailedDispatches()
	}
	service.logger.Info("no drivers found", zap.String("order_id", order.ID.String()), zap.Int("radius_meters", radiusMeters))
	return DispatchResult{OrderID: order.ID, Status: domain.OrderStatusFailed, RadiusMeters: radiusMeters, NoDriversAvailable: true}, nil
}

func (service *Service) radiusForAttempt(attempt int) (int, bool) {
	if attempt < 0 || attempt >= len(service.config.RadiusAttemptsMeters) {
		return service.lastRadius(), false
	}
	return service.config.RadiusAttemptsMeters[attempt], true
}

func (service *Service) lastRadius() int {
	if len(service.config.RadiusAttemptsMeters) == 0 {
		return service.config.MaxRadiusMeters
	}
	return service.config.RadiusAttemptsMeters[len(service.config.RadiusAttemptsMeters)-1]
}

func offerPayload(order domain.Order, offer OrderOffer) map[string]any {
	return map[string]any{
		"order_id":            order.ID,
		"pickup_address":      order.PickupAddress,
		"pickup_location":     order.PickupLocation,
		"destination_address": order.DestinationAddress,
		"estimated_price":     order.EstimatedPrice,
		"attempt":             offer.Attempt,
		"radius_meters":       offer.RadiusMeters,
		"distance_meters":     offer.DistanceMeters,
		"expires_at":          offer.ExpiresAt,
	}
}

func normalizeConfig(config Config) Config {
	if len(config.RadiusAttemptsMeters) == 0 {
		config.RadiusAttemptsMeters = []int{1000, 3000, 5000, 10000}
	}
	if config.InitialRadiusMeters <= 0 {
		config.InitialRadiusMeters = config.RadiusAttemptsMeters[0]
	}
	if config.MaxRadiusMeters < config.InitialRadiusMeters {
		config.MaxRadiusMeters = config.RadiusAttemptsMeters[len(config.RadiusAttemptsMeters)-1]
	}
	if config.RadiusStepMeters <= 0 {
		config.RadiusStepMeters = 1000
	}
	if config.MaxDriversPerOffer <= 0 {
		config.MaxDriversPerOffer = 5
	}
	if config.DriverLocationMaxAge <= 0 {
		config.DriverLocationMaxAge = 30 * time.Second
	}
	if config.OfferTTL <= 0 {
		config.OfferTTL = 15 * time.Second
	}
	if config.AcceptLockTTL <= 0 {
		config.AcceptLockTTL = 30 * time.Second
	}
	if config.WorkerPollTimeout <= 0 {
		config.WorkerPollTimeout = 5 * time.Second
	}
	if config.RecoveryInterval <= 0 {
		config.RecoveryInterval = 30 * time.Second
	}

	return config
}

func loggerOrNop(logger *zap.Logger) *zap.Logger {
	if logger == nil {
		return zap.NewNop()
	}
	return logger
}

package dispatch

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type Worker struct {
	service            *Service
	taskQueue          TaskQueue
	timeoutQueue       TimeoutQueue
	recoveryRepository RecoveryRepository
	metrics            Metrics
	logger             *zap.Logger
	config             Config
}

type NewWorkerParams struct {
	Service            *Service
	TaskQueue          TaskQueue
	TimeoutQueue       TimeoutQueue
	RecoveryRepository RecoveryRepository
	Metrics            Metrics
	Logger             *zap.Logger
	Config             Config
}

func NewWorker(params NewWorkerParams) *Worker {
	return &Worker{
		service:            params.Service,
		taskQueue:          params.TaskQueue,
		timeoutQueue:       params.TimeoutQueue,
		recoveryRepository: params.RecoveryRepository,
		metrics:            params.Metrics,
		logger:             loggerOrNop(params.Logger),
		config:             normalizeConfig(params.Config),
	}
}

func (worker *Worker) Run(ctx context.Context) error {
	errorChannel := make(chan error, 3)

	go func() {
		errorChannel <- worker.runDispatchWorker(ctx)
	}()
	go func() {
		errorChannel <- worker.runOfferTimeoutWorker(ctx)
	}()
	go func() {
		errorChannel <- worker.runStaleOrderRecoveryWorker(ctx)
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errorChannel:
		if err != nil {
			return err
		}
		return nil
	}
}

func (worker *Worker) runDispatchWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		task, ok, err := worker.taskQueue.Consume(ctx, worker.config.WorkerPollTimeout)
		if err != nil {
			return fmt.Errorf("consume dispatch task: %w", err)
		}
		if !ok {
			continue
		}
		if _, err := worker.service.ProcessTask(ctx, task); err != nil {
			worker.logger.Error("process dispatch task", zap.Error(err), zap.String("order_id", task.OrderID.String()), zap.Int("attempt", task.Attempt))
		}
	}
}

func (worker *Worker) runOfferTimeoutWorker(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			tasks, err := worker.timeoutQueue.Due(ctx, now.UTC(), 100)
			if err != nil {
				return fmt.Errorf("load due offer timeout tasks: %w", err)
			}
			for _, task := range tasks {
				if err := worker.service.HandleOfferTimeout(ctx, task); err != nil {
					worker.logger.Error("handle offer timeout", zap.Error(err), zap.String("order_id", task.OrderID.String()), zap.Int("attempt", task.Attempt))
				}
			}
		}
	}
}

func (worker *Worker) runStaleOrderRecoveryWorker(ctx context.Context) error {
	ticker := time.NewTicker(worker.config.RecoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := worker.recoverSearchingOrders(ctx); err != nil {
				return err
			}
		}
	}
}

func (worker *Worker) recoverSearchingOrders(ctx context.Context) error {
	if worker.recoveryRepository == nil {
		return nil
	}

	orderIDs, err := worker.recoveryRepository.ListSearchingOrders(ctx, 500)
	if err != nil {
		return fmt.Errorf("list searching orders for recovery: %w", err)
	}
	if worker.metrics != nil {
		worker.metrics.SetActiveSearches(len(orderIDs))
	}
	for _, orderID := range orderIDs {
		task := DispatchTask{OrderID: orderID, Attempt: 0, QueuedAt: time.Now().UTC()}
		if err := worker.taskQueue.Publish(ctx, task); err != nil {
			return fmt.Errorf("publish recovered dispatch task: %w", err)
		}
	}

	return nil
}

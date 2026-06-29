package scheduled

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
)

type WorkerConfig struct {
	Enabled     bool
	TickSeconds int
}

type Worker struct {
	service *Service
	logger  *zap.Logger
	config  WorkerConfig
}

func NewWorker(service *Service, logger *zap.Logger, config WorkerConfig) *Worker {
	if logger == nil {
		logger = zap.NewNop()
	}
	if config.TickSeconds <= 0 {
		config.TickSeconds = 30
	}
	return &Worker{service: service, logger: logger, config: config}
}

func (worker *Worker) Run(ctx context.Context) error {
	if worker == nil || worker.service == nil || !worker.config.Enabled {
		return nil
	}
	ticker := time.NewTicker(time.Duration(worker.config.TickSeconds) * time.Second)
	defer ticker.Stop()

	for {
		if err := worker.service.ActivateDueOrders(ctx); err != nil && !errors.Is(err, context.Canceled) {
			worker.logger.Error("activate due scheduled orders", zap.Error(err))
		}
		if err := worker.service.ExpirePendingOrders(ctx); err != nil && !errors.Is(err, context.Canceled) {
			worker.logger.Error("expire scheduled orders", zap.Error(err))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

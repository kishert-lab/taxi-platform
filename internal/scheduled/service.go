package scheduled

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	dispatchapp "github.com/kishert-lab/taxi-platform/internal/dispatch"
	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type Repository interface {
	ListDueOrderIDs(ctx context.Context, limit int) ([]uuid.UUID, error)
	ActivateOrder(ctx context.Context, orderID uuid.UUID) (domain.Order, bool, error)
	ExpirePendingOrders(ctx context.Context, limit int) ([]domain.Order, error)
	GetTaxiParkSettingsByOrderID(ctx context.Context, orderID uuid.UUID) (domain.TaxiParkSettings, error)
}

type DispatchController interface {
	EnqueueOrderWithConfig(ctx context.Context, orderID uuid.UUID, config dispatchapp.Config) error
}

type RealtimeGateway interface {
	SendToDriver(ctx context.Context, driverID uuid.UUID, eventName string, payload any) error
	SendToPassenger(ctx context.Context, passengerID uuid.UUID, eventName string, payload any) error
	SendToTaxiParkByOrder(ctx context.Context, orderID uuid.UUID, eventName string, payload any) error
}

type Config struct {
	BatchSize int
}

type Service struct {
	repository         Repository
	dispatchController DispatchController
	realtimeGateway    RealtimeGateway
	logger             *zap.Logger
	config             Config
}

func NewService(repository Repository, dispatchController DispatchController, realtimeGateway RealtimeGateway, logger *zap.Logger, config Config) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 100
	}
	return &Service{
		repository:         repository,
		dispatchController: dispatchController,
		realtimeGateway:    realtimeGateway,
		logger:             logger,
		config:             config,
	}
}

func (service *Service) ActivateDueOrders(ctx context.Context) error {
	orderIDs, err := service.repository.ListDueOrderIDs(ctx, service.config.BatchSize)
	if err != nil {
		return fmt.Errorf("list due scheduled orders: %w", err)
	}
	for _, orderID := range orderIDs {
		order, changed, activateErr := service.repository.ActivateOrder(ctx, orderID)
		if activateErr != nil {
			return fmt.Errorf("activate scheduled order %s: %w", orderID, activateErr)
		}
		if !changed {
			continue
		}
		if order.PreassignedDriverID == nil {
			settings, settingsErr := service.repository.GetTaxiParkSettingsByOrderID(ctx, order.ID)
			if settingsErr != nil {
				return fmt.Errorf("get taxi park settings for scheduled activation %s: %w", order.ID, settingsErr)
			}
			if err := service.dispatchController.EnqueueOrderWithConfig(ctx, order.ID, dispatchConfigFromTaxiParkSettings(settings)); err != nil {
				return fmt.Errorf("enqueue activated scheduled order %s: %w", order.ID, err)
			}
		}
		if err := service.publishScheduledEvent(ctx, order, "scheduled_order_activated"); err != nil {
			return err
		}
		service.logger.Info("scheduled order activated", zap.String("order_id", order.ID.String()))
	}
	return nil
}

func (service *Service) ExpirePendingOrders(ctx context.Context) error {
	orders, err := service.repository.ExpirePendingOrders(ctx, service.config.BatchSize)
	if err != nil {
		return fmt.Errorf("expire scheduled orders: %w", err)
	}
	for _, order := range orders {
		if err := service.publishScheduledEvent(ctx, order, "scheduled_order_expired"); err != nil {
			return err
		}
		service.logger.Info("scheduled order expired", zap.String("order_id", order.ID.String()))
	}
	return nil
}

func (service *Service) publishScheduledEvent(ctx context.Context, order domain.Order, eventName string) error {
	if service.realtimeGateway == nil {
		return nil
	}
	payload := map[string]any{
		"order_id":            order.ID,
		"scheduled_status":    order.ScheduledStatus,
		"scheduled_at":        order.ScheduledAt,
		"activation_at":       order.ActivationAt,
		"pickup_address":      order.PickupAddress,
		"destination_address": order.DestinationAddress,
		"driver_id":           order.PreassignedDriverID,
		"passenger_id":        order.PassengerID,
		"comment":             order.PassengerComment,
	}
	if err := service.realtimeGateway.SendToPassenger(ctx, order.PassengerID, eventName, payload); err != nil {
		return fmt.Errorf("publish scheduled event to passenger: %w", err)
	}
	if err := service.realtimeGateway.SendToTaxiParkByOrder(ctx, order.ID, eventName, payload); err != nil {
		return fmt.Errorf("publish scheduled event to taxi park: %w", err)
	}
	if order.PreassignedDriverID != nil {
		if err := service.realtimeGateway.SendToDriver(ctx, *order.PreassignedDriverID, eventName, payload); err != nil {
			return fmt.Errorf("publish scheduled event to driver: %w", err)
		}
	}
	return nil
}

func dispatchConfigFromTaxiParkSettings(settings domain.TaxiParkSettings) dispatchapp.Config {
	return dispatchapp.Config{
		InitialRadiusMeters:  settings.DispatchInitialRadiusMeters,
		MaxRadiusMeters:      settings.DispatchMaxRadiusMeters,
		RadiusStepMeters:     settings.DispatchRadiusStepMeters,
		RadiusAttemptsMeters: settings.DispatchRadiusAttemptsMeters,
		MaxDriversPerOffer:   settings.DispatchMaxDriversPerOffer,
		DriverLocationMaxAge: time.Duration(settings.DispatchDriverLocationMaxAgeSec) * time.Second,
		OfferTTL:             time.Duration(settings.DispatchOfferTTLSec) * time.Second,
		AcceptLockTTL:        time.Duration(settings.DispatchAcceptLockTTLSec) * time.Second,
		WorkerPollTimeout:    time.Duration(settings.DispatchWorkerPollTimeoutSec) * time.Second,
		RecoveryInterval:     time.Duration(settings.DispatchRecoveryIntervalSec) * time.Second,
	}
}

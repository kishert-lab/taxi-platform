package finance

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

var (
	ErrOrderWithoutFinalPrice       = errors.New("completed order has no final price")
	ErrOrderWithoutAssignedDriver   = errors.New("completed order has no assigned driver")
	ErrFinancialSettlementDuplicate = errors.New("financial settlement already exists")
	ErrDriverFinanceAccountNotFound = errors.New("driver finance account not found")
)

type Service struct {
	repository Repository
	metrics    *Metrics
	logger     *zap.Logger
}

func NewService(repository Repository, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{repository: repository, logger: logger}
}

func NewServiceWithMetrics(repository Repository, metrics *Metrics, logger *zap.Logger) *Service {
	service := NewService(repository, logger)
	service.metrics = metrics
	return service
}

func (service *Service) SettleCompletedOrder(ctx context.Context, orderID uuid.UUID) (domain.OrderSettlement, error) {
	snapshot, err := service.repository.GetOrderSnapshot(ctx, orderID)
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("get order finance snapshot: %w", err)
	}
	if snapshot.Status != domain.OrderStatusCompleted {
		return domain.OrderSettlement{}, domain.ErrFinancialOrderIgnored
	}
	if snapshot.FinalPrice == nil {
		return domain.OrderSettlement{}, ErrOrderWithoutFinalPrice
	}
	if snapshot.DriverID == nil {
		return domain.OrderSettlement{}, ErrOrderWithoutAssignedDriver
	}

	rate, err := domain.ResolveCommissionRate(domain.CommissionContext{
		PlatformDefaultBasisPoints: snapshot.PlatformDefaultBPS,
		CityBasisPoints:            snapshot.CityCommissionBPS,
		TariffBasisPoints:          snapshot.TariffCommissionBPS,
		TaxiParkBasisPoints:        snapshot.TaxiParkCommissionBPS,
		DriverBasisPoints:          snapshot.DriverCommissionBPS,
	})
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("resolve commission: %w", err)
	}

	commission, net, err := domain.CalculateCommission(*snapshot.FinalPrice, rate)
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("calculate commission: %w", err)
	}

	settlement := domain.OrderSettlement{
		OrderID:          snapshot.OrderID,
		DriverID:         *snapshot.DriverID,
		TaxiParkID:       snapshot.TaxiParkID,
		GrossAmount:      *snapshot.FinalPrice,
		CommissionRate:   rate,
		CommissionAmount: commission,
		NetAmount:        net,
		CreatedAt:        time.Now().UTC(),
	}

	createdSettlement, err := service.repository.CreateOrderSettlement(ctx, settlement)
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("create order financial settlement: %w", err)
	}

	service.logger.Info(
		"order financial settlement created",
		zap.String("order_id", orderID.String()),
		zap.Int64("gross_amount_cents", createdSettlement.GrossAmount.Amount),
		zap.Int32("commission_bps", createdSettlement.CommissionRate.BasisPoints),
		zap.Int64("commission_amount_cents", createdSettlement.CommissionAmount.Amount),
		zap.Int64("net_amount_cents", createdSettlement.NetAmount.Amount),
	)
	service.metrics.ObserveSettlement(
		createdSettlement.GrossAmount.Amount,
		createdSettlement.CommissionAmount.Amount,
		createdSettlement.NetAmount.Amount,
		createdSettlement.TaxiParkID != nil,
	)

	return createdSettlement, nil
}

func (service *Service) GetDriverBalance(ctx context.Context, userID uuid.UUID) (domain.DriverBalance, error) {
	driverID, err := service.repository.GetDriverIDByUserID(ctx, userID)
	if err != nil {
		return domain.DriverBalance{}, fmt.Errorf("resolve driver balance owner: %w", err)
	}
	return service.repository.GetDriverBalance(ctx, driverID)
}

func (service *Service) ListDriverTransactions(ctx context.Context, userID uuid.UUID, limit int) ([]domain.FinancialTransaction, error) {
	driverID, err := service.repository.GetDriverIDByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("resolve driver transaction owner: %w", err)
	}
	return service.repository.ListDriverTransactions(ctx, driverID, normalizeLimit(limit))
}

func (service *Service) GetTaxiParkBalance(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkBalance, error) {
	return service.repository.GetTaxiParkBalance(ctx, ownerUserID)
}

func (service *Service) ListTaxiParkDrivers(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]TaxiParkDriver, error) {
	return service.repository.ListTaxiParkDrivers(ctx, ownerUserID, normalizeLimit(limit))
}

func (service *Service) ListTaxiParkOrders(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]TaxiParkOrder, error) {
	return service.repository.ListTaxiParkOrders(ctx, ownerUserID, normalizeLimit(limit))
}

func (service *Service) ListTaxiParkTransactions(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]domain.FinancialTransaction, error) {
	return service.repository.ListTaxiParkTransactions(ctx, ownerUserID, normalizeLimit(limit))
}

func (service *Service) GetAdminOverview(ctx context.Context, periodFrom time.Time, periodTo time.Time) (AdminOverview, error) {
	if periodTo.IsZero() {
		periodTo = time.Now().UTC()
	}
	if periodFrom.IsZero() {
		periodFrom = periodTo.AddDate(0, 0, -30)
	}

	return service.repository.GetAdminOverview(ctx, periodFrom, periodTo)
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}

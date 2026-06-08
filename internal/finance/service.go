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
	ErrInsufficientDriverBalance    = errors.New("insufficient driver balance")
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

	driverCommissionBasisPoints := int32(0)
	if snapshot.DriverCommissionBPS != nil {
		driverCommissionBasisPoints = *snapshot.DriverCommissionBPS
	}

	rate, err := domain.ResolveCommissionRate(domain.CommissionContext{
		PlatformDefaultBasisPoints: driverCommissionBasisPoints,
	})
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("resolve commission: %w", err)
	}

	commission, driverIncome, err := domain.CalculateCommission(*snapshot.FinalPrice, rate)
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("calculate commission: %w", err)
	}
	platformFeeRate, err := domain.ResolveCommissionRate(domain.CommissionContext{
		PlatformDefaultBasisPoints: snapshot.PlatformFeeBPS,
	})
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("resolve platform fee rate: %w", err)
	}
	platformFeeAmount, err := domain.CalculatePlatformFee(commission, platformFeeRate)
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("calculate platform fee: %w", err)
	}

	settlement := domain.OrderSettlement{
		OrderID:              snapshot.OrderID,
		DriverID:             *snapshot.DriverID,
		TaxiParkID:           snapshot.TaxiParkID,
		GrossAmount:          *snapshot.FinalPrice,
		CommissionRate:       rate,
		CommissionAmount:     commission,
		NetAmount:            driverIncome,
		PlatformFeeRate:      platformFeeRate,
		PlatformFeeAmount:    platformFeeAmount,
		TaxiParkIncomeAmount: commission,
		CreatedAt:            time.Now().UTC(),
	}

	createdSettlement, err := service.repository.CreateOrderSettlement(ctx, settlement)
	if err != nil {
		return domain.OrderSettlement{}, fmt.Errorf("create order financial settlement: %w", err)
	}

	service.logger.Info(
		"order financial settlement created",
		zap.String("order_id", orderID.String()),
		zap.Int64("gross_amount_cents", createdSettlement.GrossAmount.Amount),
		zap.Int32("driver_commission_bps", createdSettlement.CommissionRate.BasisPoints),
		zap.Int64("taxi_park_commission_amount_cents", createdSettlement.CommissionAmount.Amount),
		zap.Int64("driver_income_amount_cents", createdSettlement.NetAmount.Amount),
		zap.Int32("platform_fee_bps", createdSettlement.PlatformFeeRate.BasisPoints),
		zap.Int64("platform_fee_amount_cents", createdSettlement.PlatformFeeAmount.Amount),
	)
	service.metrics.ObserveSettlement(
		createdSettlement.GrossAmount.Amount,
		createdSettlement.CommissionAmount.Amount,
		createdSettlement.NetAmount.Amount,
		createdSettlement.TaxiParkID != nil,
	)

	return createdSettlement, nil
}

func (service *Service) GetTaxiParkFinanceSettings(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkFinanceSettings, error) {
	return service.repository.GetTaxiParkFinanceSettings(ctx, ownerUserID)
}

func (service *Service) UpdateTaxiParkDriverCommission(ctx context.Context, ownerUserID uuid.UUID, driverCommissionBasisPoints int32) (domain.TaxiParkFinanceSettings, error) {
	if driverCommissionBasisPoints < 0 || driverCommissionBasisPoints > 10000 {
		return domain.TaxiParkFinanceSettings{}, domain.ErrInvalidCommissionRate
	}
	return service.repository.UpdateTaxiParkDriverCommission(ctx, ownerUserID, driverCommissionBasisPoints)
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

func (service *Service) ListDriverOrderFinances(ctx context.Context, userID uuid.UUID, limit int) ([]OrderFinance, error) {
	return service.repository.ListDriverOrderFinances(ctx, userID, normalizeLimit(limit))
}

func (service *Service) ListDriverPayouts(ctx context.Context, userID uuid.UUID, limit int) ([]DriverPayout, error) {
	return service.repository.ListDriverPayouts(ctx, userID, normalizeLimit(limit))
}

func (service *Service) ListDriverFinanceDocuments(ctx context.Context, userID uuid.UUID, limit int) ([]FinanceDocument, error) {
	return service.repository.ListDriverFinanceDocuments(ctx, userID, normalizeLimit(limit))
}

func (service *Service) GetTaxiParkBalance(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkBalance, error) {
	return service.repository.GetTaxiParkBalance(ctx, ownerUserID)
}

func (service *Service) GetTaxiParkOverview(ctx context.Context, ownerUserID uuid.UUID, periodFrom time.Time, periodTo time.Time) (TaxiParkFinanceOverview, error) {
	periodFrom, periodTo = normalizePeriod(periodFrom, periodTo)
	return service.repository.GetTaxiParkOverview(ctx, ownerUserID, periodFrom, periodTo)
}

func (service *Service) ListTaxiParkOrderFinances(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]OrderFinance, error) {
	return service.repository.ListTaxiParkOrderFinances(ctx, ownerUserID, normalizeLimit(limit))
}

func (service *Service) GetTaxiParkDriverBalance(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) (domain.DriverBalance, error) {
	return service.repository.GetTaxiParkDriverBalance(ctx, ownerUserID, driverID)
}

func (service *Service) ListTaxiParkDriverPayouts(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, limit int) ([]DriverPayout, error) {
	return service.repository.ListTaxiParkDriverPayouts(ctx, ownerUserID, driverID, normalizeLimit(limit))
}

func (service *Service) CreateDriverPayout(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, input CreateDriverPayoutInput) (DriverPayout, error) {
	if input.AmountCents <= 0 {
		return DriverPayout{}, domain.ErrInvalidMoney
	}
	payout, err := service.repository.CreateDriverPayout(ctx, ownerUserID, driverID, input)
	if err != nil {
		return DriverPayout{}, err
	}
	if _, err := service.repository.GenerateDriverPayoutAct(ctx, payout.ID); err != nil {
		service.logger.Warn("generate driver payout act failed", zap.String("payout_id", payout.ID.String()), zap.Error(err))
	}
	return payout, nil
}

func (service *Service) ApproveDriverPayout(ctx context.Context, ownerUserID uuid.UUID, payoutID uuid.UUID) (DriverPayout, error) {
	return service.repository.ApproveDriverPayout(ctx, ownerUserID, payoutID)
}

func (service *Service) MarkDriverPayoutPaid(ctx context.Context, ownerUserID uuid.UUID, payoutID uuid.UUID) (DriverPayout, error) {
	return service.repository.MarkDriverPayoutPaid(ctx, ownerUserID, payoutID)
}

func (service *Service) GetTaxiParkPlatformFeeDebt(ctx context.Context, ownerUserID uuid.UUID) (domain.Money, error) {
	return service.repository.GetTaxiParkPlatformFeeDebt(ctx, ownerUserID)
}

func (service *Service) ListTaxiParkPlatformFeeAccruals(ctx context.Context, ownerUserID uuid.UUID, periodFrom time.Time, periodTo time.Time, limit int) ([]OrderFinance, error) {
	periodFrom, periodTo = normalizePeriod(periodFrom, periodTo)
	return service.repository.ListTaxiParkPlatformFeeAccruals(ctx, ownerUserID, periodFrom, periodTo, normalizeLimit(limit))
}

func (service *Service) ListTaxiParkPlatformInvoices(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]PlatformInvoice, error) {
	return service.repository.ListTaxiParkPlatformInvoices(ctx, ownerUserID, normalizeLimit(limit))
}

func (service *Service) CreateTaxiParkPlatformInvoice(ctx context.Context, ownerUserID uuid.UUID, input CreatePlatformInvoiceInput) (PlatformInvoice, error) {
	invoice, err := service.repository.CreateTaxiParkPlatformInvoice(ctx, ownerUserID, input)
	if err != nil {
		return PlatformInvoice{}, err
	}
	if _, err := service.repository.GeneratePlatformInvoice(ctx, invoice.ID); err != nil {
		service.logger.Warn("generate platform invoice document failed", zap.String("invoice_id", invoice.ID.String()), zap.Error(err))
	}
	return invoice, nil
}

func (service *Service) ListTaxiParkDocuments(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]FinanceDocument, error) {
	return service.repository.ListTaxiParkDocuments(ctx, ownerUserID, normalizeLimit(limit))
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
	periodFrom, periodTo = normalizePeriod(periodFrom, periodTo)
	return service.repository.GetAdminOverview(ctx, periodFrom, periodTo)
}

func (service *Service) GetAdminTaxiParkOverview(ctx context.Context, taxiParkID uuid.UUID, periodFrom time.Time, periodTo time.Time) (TaxiParkFinanceOverview, error) {
	periodFrom, periodTo = normalizePeriod(periodFrom, periodTo)
	return service.repository.GetAdminTaxiParkOverview(ctx, taxiParkID, periodFrom, periodTo)
}

func (service *Service) GetAdminTaxiParkPlatformFeeDebt(ctx context.Context, taxiParkID uuid.UUID) (domain.Money, error) {
	return service.repository.GetAdminTaxiParkPlatformFeeDebt(ctx, taxiParkID)
}

func (service *Service) ListAdminPlatformInvoices(ctx context.Context, limit int) ([]PlatformInvoice, error) {
	return service.repository.ListAdminPlatformInvoices(ctx, normalizeLimit(limit))
}

func (service *Service) MarkAdminPlatformInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (PlatformInvoice, error) {
	invoice, err := service.repository.MarkAdminPlatformInvoicePaid(ctx, invoiceID)
	if err != nil {
		return PlatformInvoice{}, err
	}
	if _, err := service.repository.GeneratePlatformAct(ctx, invoice.ID); err != nil {
		service.logger.Warn("generate platform act failed", zap.String("invoice_id", invoice.ID.String()), zap.Error(err))
	}
	return invoice, nil
}

func (service *Service) ListAdminDocuments(ctx context.Context, limit int) ([]FinanceDocument, error) {
	return service.repository.ListAdminDocuments(ctx, normalizeLimit(limit))
}

func normalizeLimit(limit int) int {
	if limit <= 0 || limit > 100 {
		return 50
	}
	return limit
}

func normalizePeriod(periodFrom time.Time, periodTo time.Time) (time.Time, time.Time) {
	if periodTo.IsZero() {
		periodTo = time.Now().UTC()
	}
	if periodFrom.IsZero() {
		periodFrom = periodTo.AddDate(0, 0, -30)
	}
	return periodFrom, periodTo
}

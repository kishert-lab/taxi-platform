package finance

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

func TestSettleCompletedOrderCreatesTransaction(t *testing.T) {
	orderID := uuid.New()
	driverID := uuid.New()
	finalPrice := domain.Money{Amount: 50000, Currency: "RUB"}
	repository := &fakeFinanceRepository{
		snapshot: OrderSnapshot{
			OrderID:            orderID,
			DriverID:           &driverID,
			Status:             domain.OrderStatusCompleted,
			FinalPrice:         &finalPrice,
			PlatformDefaultBPS: domain.DefaultPlatformCommissionBasisPoints,
		},
	}
	service := NewService(repository, nil)

	settlement, err := service.SettleCompletedOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("settle completed order: %v", err)
	}

	if !repository.created {
		t.Fatal("expected settlement to be persisted")
	}
	if settlement.CommissionAmount.Amount != 500 || settlement.NetAmount.Amount != 49500 {
		t.Fatalf("unexpected settlement: %#v", settlement)
	}
}

func TestCancelledOrderCreatesNoCommission(t *testing.T) {
	orderID := uuid.New()
	driverID := uuid.New()
	finalPrice := domain.Money{Amount: 50000, Currency: "RUB"}
	repository := &fakeFinanceRepository{
		snapshot: OrderSnapshot{
			OrderID:    orderID,
			DriverID:   &driverID,
			Status:     domain.OrderStatusCancelled,
			FinalPrice: &finalPrice,
		},
	}
	service := NewService(repository, nil)

	_, err := service.SettleCompletedOrder(context.Background(), orderID)
	if !errors.Is(err, domain.ErrFinancialOrderIgnored) {
		t.Fatalf("expected ignored financial order, got %v", err)
	}
	if repository.created {
		t.Fatal("cancelled order must not create settlement")
	}
}

func TestDuplicateCompletionPrevented(t *testing.T) {
	orderID := uuid.New()
	driverID := uuid.New()
	finalPrice := domain.Money{Amount: 50000, Currency: "RUB"}
	repository := &fakeFinanceRepository{
		snapshot: OrderSnapshot{
			OrderID:            orderID,
			DriverID:           &driverID,
			Status:             domain.OrderStatusCompleted,
			FinalPrice:         &finalPrice,
			PlatformDefaultBPS: domain.DefaultPlatformCommissionBasisPoints,
		},
		createErr: ErrFinancialSettlementDuplicate,
	}
	service := NewService(repository, nil)

	_, err := service.SettleCompletedOrder(context.Background(), orderID)
	if !errors.Is(err, ErrFinancialSettlementDuplicate) {
		t.Fatalf("expected duplicate settlement error, got %v", err)
	}
}

type fakeFinanceRepository struct {
	snapshot  OrderSnapshot
	created   bool
	createErr error
}

func (repository *fakeFinanceRepository) GetOrderSnapshot(_ context.Context, _ uuid.UUID) (OrderSnapshot, error) {
	return repository.snapshot, nil
}

func (repository *fakeFinanceRepository) CreateOrderSettlement(_ context.Context, settlement domain.OrderSettlement) (domain.OrderSettlement, error) {
	if repository.createErr != nil {
		return domain.OrderSettlement{}, repository.createErr
	}
	repository.created = true
	settlement.CommissionTransactionID = uuid.New()
	settlement.IncomeTransactionID = uuid.New()
	return settlement, nil
}

func (repository *fakeFinanceRepository) GetDriverIDByUserID(_ context.Context, userID uuid.UUID) (uuid.UUID, error) {
	return userID, nil
}

func (repository *fakeFinanceRepository) GetDriverBalance(_ context.Context, driverID uuid.UUID) (domain.DriverBalance, error) {
	return domain.DriverBalance{DriverID: driverID}, nil
}

func (repository *fakeFinanceRepository) ListDriverTransactions(_ context.Context, _ uuid.UUID, _ int) ([]domain.FinancialTransaction, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) GetTaxiParkBalance(_ context.Context, ownerUserID uuid.UUID) (domain.TaxiParkBalance, error) {
	return domain.TaxiParkBalance{TaxiParkID: ownerUserID}, nil
}

func (repository *fakeFinanceRepository) ListTaxiParkDrivers(_ context.Context, _ uuid.UUID, _ int) ([]TaxiParkDriver, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) ListTaxiParkOrders(_ context.Context, _ uuid.UUID, _ int) ([]TaxiParkOrder, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) ListTaxiParkTransactions(_ context.Context, _ uuid.UUID, _ int) ([]domain.FinancialTransaction, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) GetAdminOverview(_ context.Context, periodFrom time.Time, periodTo time.Time) (AdminOverview, error) {
	return AdminOverview{PeriodFrom: periodFrom, PeriodTo: periodTo}, nil
}

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
	taxiParkID := uuid.New()
	finalPrice := domain.Money{Amount: 50000, Currency: "RUB"}
	repository := &fakeFinanceRepository{
		snapshot: OrderSnapshot{
			OrderID:             orderID,
			DriverID:            &driverID,
			TaxiParkID:          &taxiParkID,
			Status:              domain.OrderStatusCompleted,
			FinalPrice:          &finalPrice,
			DriverCommissionBPS: int32Ptr(2000),
			PlatformFeeBPS:      100,
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
	if settlement.CommissionAmount.Amount != 10000 || settlement.NetAmount.Amount != 40000 {
		t.Fatalf("unexpected settlement: %#v", settlement)
	}
	if settlement.PlatformFeeAmount.Amount != 100 {
		t.Fatalf("expected platform fee 100 cents, got %d", settlement.PlatformFeeAmount.Amount)
	}
	if settlement.TaxiParkIncomeAmount.Amount != 10000 {
		t.Fatalf("expected taxi park income 10000 cents, got %d", settlement.TaxiParkIncomeAmount.Amount)
	}
}

func TestCancelledOrderCreatesNoCommission(t *testing.T) {
	orderID := uuid.New()
	driverID := uuid.New()
	finalPrice := domain.Money{Amount: 50000, Currency: "RUB"}
	repository := &fakeFinanceRepository{
		snapshot: OrderSnapshot{
			OrderID:        orderID,
			DriverID:       &driverID,
			Status:         domain.OrderStatusCancelled,
			FinalPrice:     &finalPrice,
			PlatformFeeBPS: 100,
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

func TestDuplicateCompletionReturnsExistingSettlement(t *testing.T) {
	orderID := uuid.New()
	driverID := uuid.New()
	finalPrice := domain.Money{Amount: 50000, Currency: "RUB"}
	existingSettlement := domain.OrderSettlement{
		OrderID:              orderID,
		DriverID:             driverID,
		GrossAmount:          finalPrice,
		CommissionRate:       domain.CommissionRate{BasisPoints: 2000, Source: "taxi_park"},
		CommissionAmount:     domain.Money{Amount: 10000, Currency: "RUB"},
		NetAmount:            domain.Money{Amount: 40000, Currency: "RUB"},
		PlatformFeeRate:      domain.CommissionRate{BasisPoints: 100, Source: "platform"},
		PlatformFeeAmount:    domain.Money{Amount: 100, Currency: "RUB"},
		TaxiParkIncomeAmount: domain.Money{Amount: 10000, Currency: "RUB"},
	}
	repository := &fakeFinanceRepository{
		snapshot: OrderSnapshot{
			OrderID:             orderID,
			DriverID:            &driverID,
			Status:              domain.OrderStatusCompleted,
			FinalPrice:          &finalPrice,
			DriverCommissionBPS: int32Ptr(2000),
			PlatformFeeBPS:      100,
		},
		existingSettlement: existingSettlement,
	}
	service := NewService(repository, nil)

	settlement, err := service.SettleCompletedOrder(context.Background(), orderID)
	if err != nil {
		t.Fatalf("expected existing settlement without error, got %v", err)
	}
	if settlement.PlatformFeeAmount.Amount != existingSettlement.PlatformFeeAmount.Amount {
		t.Fatalf("expected existing settlement to be returned, got %#v", settlement)
	}
}

type fakeFinanceRepository struct {
	snapshot           OrderSnapshot
	created            bool
	existingSettlement domain.OrderSettlement
}

func (repository *fakeFinanceRepository) GetOrderSnapshot(_ context.Context, _ uuid.UUID) (OrderSnapshot, error) {
	return repository.snapshot, nil
}

func (repository *fakeFinanceRepository) CreateOrderSettlement(_ context.Context, settlement domain.OrderSettlement) (domain.OrderSettlement, error) {
	if repository.existingSettlement.OrderID != uuid.Nil {
		return repository.existingSettlement, nil
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

func (repository *fakeFinanceRepository) ListDriverOrderFinances(_ context.Context, _ uuid.UUID, _ int) ([]OrderFinance, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) ListDriverPayouts(_ context.Context, _ uuid.UUID, _ int) ([]DriverPayout, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) ListDriverFinanceDocuments(_ context.Context, _ uuid.UUID, _ int) ([]FinanceDocument, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) GetTaxiParkBalance(_ context.Context, ownerUserID uuid.UUID) (domain.TaxiParkBalance, error) {
	return domain.TaxiParkBalance{TaxiParkID: ownerUserID}, nil
}

func (repository *fakeFinanceRepository) GetTaxiParkFinanceSettings(_ context.Context, ownerUserID uuid.UUID) (domain.TaxiParkFinanceSettings, error) {
	return domain.TaxiParkFinanceSettings{TaxiParkID: ownerUserID}, nil
}

func (repository *fakeFinanceRepository) UpdateTaxiParkDriverCommission(_ context.Context, ownerUserID uuid.UUID, driverCommissionBasisPoints int32) (domain.TaxiParkFinanceSettings, error) {
	return domain.TaxiParkFinanceSettings{
		TaxiParkID:           ownerUserID,
		DriverCommissionRate: domain.CommissionRate{BasisPoints: driverCommissionBasisPoints, Source: "taxi_park"},
	}, nil
}

func (repository *fakeFinanceRepository) GetTaxiParkOverview(_ context.Context, ownerUserID uuid.UUID, periodFrom time.Time, periodTo time.Time) (TaxiParkFinanceOverview, error) {
	return TaxiParkFinanceOverview{TaxiParkID: ownerUserID, PeriodFrom: periodFrom, PeriodTo: periodTo}, nil
}

func (repository *fakeFinanceRepository) ListTaxiParkOrderFinances(_ context.Context, _ uuid.UUID, _ int) ([]OrderFinance, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) GetTaxiParkDriverBalance(_ context.Context, _ uuid.UUID, driverID uuid.UUID) (domain.DriverBalance, error) {
	return domain.DriverBalance{DriverID: driverID}, nil
}

func (repository *fakeFinanceRepository) ListTaxiParkDriverPayouts(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) ([]DriverPayout, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) CreateDriverPayout(_ context.Context, _ uuid.UUID, driverID uuid.UUID, input CreateDriverPayoutInput) (DriverPayout, error) {
	return DriverPayout{ID: uuid.New(), DriverID: driverID, Amount: domain.Money{Amount: input.AmountCents, Currency: "RUB"}}, nil
}

func (repository *fakeFinanceRepository) ApproveDriverPayout(_ context.Context, _ uuid.UUID, payoutID uuid.UUID) (DriverPayout, error) {
	return DriverPayout{ID: payoutID}, nil
}

func (repository *fakeFinanceRepository) MarkDriverPayoutPaid(_ context.Context, _ uuid.UUID, payoutID uuid.UUID) (DriverPayout, error) {
	return DriverPayout{ID: payoutID}, nil
}

func (repository *fakeFinanceRepository) GetTaxiParkPlatformFeeDebt(_ context.Context, _ uuid.UUID) (domain.Money, error) {
	return domain.Money{Currency: "RUB"}, nil
}

func (repository *fakeFinanceRepository) ListTaxiParkPlatformFeeAccruals(_ context.Context, _ uuid.UUID, _ time.Time, _ time.Time, _ int) ([]OrderFinance, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) ListTaxiParkPlatformInvoices(_ context.Context, _ uuid.UUID, _ int) ([]PlatformInvoice, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) CreateTaxiParkPlatformInvoice(_ context.Context, ownerUserID uuid.UUID, input CreatePlatformInvoiceInput) (PlatformInvoice, error) {
	return PlatformInvoice{ID: uuid.New(), TaxiParkID: ownerUserID, PeriodFrom: input.PeriodFrom, PeriodTo: input.PeriodTo}, nil
}

func (repository *fakeFinanceRepository) ListTaxiParkDocuments(_ context.Context, _ uuid.UUID, _ int) ([]FinanceDocument, error) {
	return nil, nil
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

func (repository *fakeFinanceRepository) GetAdminTaxiParkOverview(_ context.Context, taxiParkID uuid.UUID, periodFrom time.Time, periodTo time.Time) (TaxiParkFinanceOverview, error) {
	return TaxiParkFinanceOverview{TaxiParkID: taxiParkID, PeriodFrom: periodFrom, PeriodTo: periodTo}, nil
}

func (repository *fakeFinanceRepository) GetAdminTaxiParkPlatformFeeDebt(_ context.Context, _ uuid.UUID) (domain.Money, error) {
	return domain.Money{Currency: "RUB"}, nil
}

func (repository *fakeFinanceRepository) ListAdminPlatformInvoices(_ context.Context, _ int) ([]PlatformInvoice, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) MarkAdminPlatformInvoicePaid(_ context.Context, invoiceID uuid.UUID) (PlatformInvoice, error) {
	return PlatformInvoice{ID: invoiceID}, nil
}

func (repository *fakeFinanceRepository) ListAdminDocuments(_ context.Context, _ int) ([]FinanceDocument, error) {
	return nil, nil
}

func (repository *fakeFinanceRepository) GenerateDriverPayoutAct(_ context.Context, payoutID uuid.UUID) (FinanceDocument, error) {
	return FinanceDocument{PayoutID: &payoutID}, nil
}

func (repository *fakeFinanceRepository) GeneratePlatformInvoice(_ context.Context, invoiceID uuid.UUID) (FinanceDocument, error) {
	return FinanceDocument{InvoiceID: &invoiceID}, nil
}

func (repository *fakeFinanceRepository) GeneratePlatformAct(_ context.Context, invoiceID uuid.UUID) (FinanceDocument, error) {
	return FinanceDocument{InvoiceID: &invoiceID}, nil
}

func (repository *fakeFinanceRepository) GenerateOrderFinancialReport(_ context.Context, orderID uuid.UUID) (FinanceDocument, error) {
	return FinanceDocument{OrderID: &orderID}, nil
}

func (repository *fakeFinanceRepository) GenerateReconciliationAct(_ context.Context, taxiParkID uuid.UUID, _, _ time.Time) (FinanceDocument, error) {
	return FinanceDocument{TaxiParkID: &taxiParkID}, nil
}

func int32Ptr(value int32) *int32 {
	return &value
}

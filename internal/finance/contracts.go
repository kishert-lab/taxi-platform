package finance

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type Repository interface {
	GetOrderSnapshot(ctx context.Context, orderID uuid.UUID) (OrderSnapshot, error)
	CreateOrderSettlement(ctx context.Context, settlement domain.OrderSettlement) (domain.OrderSettlement, error)
	GetDriverBalance(ctx context.Context, driverID uuid.UUID) (domain.DriverBalance, error)
	ListDriverTransactions(ctx context.Context, driverID uuid.UUID, limit int) ([]domain.FinancialTransaction, error)
	GetTaxiParkBalance(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkBalance, error)
	ListTaxiParkDrivers(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]TaxiParkDriver, error)
	ListTaxiParkOrders(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]TaxiParkOrder, error)
	ListTaxiParkTransactions(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]domain.FinancialTransaction, error)
	GetAdminOverview(ctx context.Context, periodFrom time.Time, periodTo time.Time) (AdminOverview, error)
}

type OrderSnapshot struct {
	OrderID               uuid.UUID
	DriverID              *uuid.UUID
	TaxiParkID            *uuid.UUID
	CityID                uuid.UUID
	TariffID              *uuid.UUID
	Status                domain.OrderStatus
	FinalPrice            *domain.Money
	DriverCommissionBPS   *int32
	TaxiParkCommissionBPS *int32
	TariffCommissionBPS   *int32
	CityCommissionBPS     *int32
	PlatformDefaultBPS    int32
}

type TaxiParkDriver struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	FullName  string
	Status    domain.DriverStatus
	Rating    float64
	CreatedAt time.Time
}

type TaxiParkOrder struct {
	ID          uuid.UUID
	DriverID    uuid.UUID
	Status      domain.OrderStatus
	GrossAmount domain.Money
	CreatedAt   time.Time
	CompletedAt *time.Time
}

type AdminOverview struct {
	CompletedOrdersRevenue domain.Money
	TotalCommissions       domain.Money
	DriverPayouts          domain.Money
	TaxiParkRevenue        domain.Money
	AverageCommission      domain.Money
	CompletedOrdersCount   int64
	PeriodFrom             time.Time
	PeriodTo               time.Time
}

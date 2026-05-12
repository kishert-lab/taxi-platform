package dto

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type MoneyCentsResponse struct {
	AmountCents int64  `json:"amount_cents" example:"49500"`
	Currency    string `json:"currency" example:"RUB"`
}

type DriverBalanceResponse struct {
	DriverID         uuid.UUID          `json:"driver_id" example:"22222222-2222-2222-2222-222222222222"`
	AvailableBalance MoneyCentsResponse `json:"available_balance"`
	PendingBalance   MoneyCentsResponse `json:"pending_balance"`
	UpdatedAt        time.Time          `json:"updated_at" example:"2026-05-12T12:00:00Z"`
}

type TaxiParkBalanceResponse struct {
	TaxiParkID       uuid.UUID          `json:"taxi_park_id" example:"33333333-3333-3333-3333-333333333333"`
	AvailableBalance MoneyCentsResponse `json:"available_balance"`
	UpdatedAt        time.Time          `json:"updated_at" example:"2026-05-12T12:00:00Z"`
}

type FinancialTransactionResponse struct {
	ID                    uuid.UUID              `json:"id" example:"44444444-4444-4444-4444-444444444444"`
	OrderID               *uuid.UUID             `json:"order_id,omitempty" example:"55555555-5555-5555-5555-555555555555"`
	DriverID              *uuid.UUID             `json:"driver_id,omitempty" example:"22222222-2222-2222-2222-222222222222"`
	TaxiParkID            *uuid.UUID             `json:"taxi_park_id,omitempty" example:"33333333-3333-3333-3333-333333333333"`
	TransactionType       domain.TransactionType `json:"transaction_type" example:"driver_income"`
	GrossAmount           MoneyCentsResponse     `json:"gross_amount"`
	CommissionBasisPoints int32                  `json:"commission_basis_points" example:"100"`
	CommissionPercent     string                 `json:"commission_percent" example:"1.00"`
	CommissionAmount      MoneyCentsResponse     `json:"commission_amount"`
	NetAmount             MoneyCentsResponse     `json:"net_amount"`
	CreatedAt             time.Time              `json:"created_at" example:"2026-05-12T12:00:00Z"`
}

type FinancialTransactionsResponse struct {
	Transactions []FinancialTransactionResponse `json:"transactions"`
}

type TaxiParkDriverResponse struct {
	ID        uuid.UUID           `json:"id" example:"22222222-2222-2222-2222-222222222222"`
	UserID    uuid.UUID           `json:"user_id" example:"66666666-6666-6666-6666-666666666666"`
	FullName  string              `json:"full_name" example:"Ivan Petrov"`
	Status    domain.DriverStatus `json:"status" example:"online"`
	Rating    float64             `json:"rating" example:"4.95"`
	CreatedAt time.Time           `json:"created_at" example:"2026-05-12T12:00:00Z"`
}

type TaxiParkDriversResponse struct {
	Drivers []TaxiParkDriverResponse `json:"drivers"`
}

type TaxiParkOrderResponse struct {
	ID          uuid.UUID          `json:"id" example:"55555555-5555-5555-5555-555555555555"`
	DriverID    uuid.UUID          `json:"driver_id" example:"22222222-2222-2222-2222-222222222222"`
	Status      domain.OrderStatus `json:"status" example:"completed"`
	GrossAmount MoneyCentsResponse `json:"gross_amount"`
	CreatedAt   time.Time          `json:"created_at" example:"2026-05-12T12:00:00Z"`
	CompletedAt *time.Time         `json:"completed_at,omitempty" example:"2026-05-12T12:30:00Z"`
}

type TaxiParkOrdersResponse struct {
	Orders []TaxiParkOrderResponse `json:"orders"`
}

type AdminFinanceOverviewResponse struct {
	CompletedOrdersRevenue MoneyCentsResponse `json:"completed_orders_revenue"`
	TotalCommissions       MoneyCentsResponse `json:"total_commissions"`
	DriverPayouts          MoneyCentsResponse `json:"driver_payouts"`
	TaxiParkRevenue        MoneyCentsResponse `json:"taxi_park_revenue"`
	AverageCommission      MoneyCentsResponse `json:"average_commission_per_order"`
	CompletedOrdersCount   int64              `json:"completed_orders_count" example:"120"`
	PeriodFrom             time.Time          `json:"period_from" example:"2026-05-01T00:00:00Z"`
	PeriodTo               time.Time          `json:"period_to" example:"2026-06-01T00:00:00Z"`
}

func MoneyCentsFromDomain(money domain.Money) MoneyCentsResponse {
	return MoneyCentsResponse{AmountCents: money.Amount, Currency: money.Currency}
}

func DriverBalanceFromDomain(balance domain.DriverBalance) DriverBalanceResponse {
	return DriverBalanceResponse{
		DriverID:         balance.DriverID,
		AvailableBalance: MoneyCentsFromDomain(balance.AvailableBalance),
		PendingBalance:   MoneyCentsFromDomain(balance.PendingBalance),
		UpdatedAt:        balance.UpdatedAt,
	}
}

func TaxiParkBalanceFromDomain(balance domain.TaxiParkBalance) TaxiParkBalanceResponse {
	return TaxiParkBalanceResponse{
		TaxiParkID:       balance.TaxiParkID,
		AvailableBalance: MoneyCentsFromDomain(balance.AvailableBalance),
		UpdatedAt:        balance.UpdatedAt,
	}
}

func FinancialTransactionFromDomain(transaction domain.FinancialTransaction) FinancialTransactionResponse {
	return FinancialTransactionResponse{
		ID:                    transaction.ID,
		OrderID:               transaction.OrderID,
		DriverID:              transaction.DriverID,
		TaxiParkID:            transaction.TaxiParkID,
		TransactionType:       transaction.TransactionType,
		GrossAmount:           MoneyCentsFromDomain(transaction.GrossAmount),
		CommissionBasisPoints: transaction.CommissionBasisPoints,
		CommissionPercent:     FormatBasisPoints(transaction.CommissionBasisPoints),
		CommissionAmount:      MoneyCentsFromDomain(transaction.CommissionAmount),
		NetAmount:             MoneyCentsFromDomain(transaction.NetAmount),
		CreatedAt:             transaction.CreatedAt,
	}
}

func FinancialTransactionsFromDomain(transactions []domain.FinancialTransaction) FinancialTransactionsResponse {
	response := FinancialTransactionsResponse{Transactions: make([]FinancialTransactionResponse, 0, len(transactions))}
	for _, transaction := range transactions {
		response.Transactions = append(response.Transactions, FinancialTransactionFromDomain(transaction))
	}
	return response
}

func FormatBasisPoints(basisPoints int32) string {
	return formatCommissionPercent(basisPoints)
}

func formatCommissionPercent(basisPoints int32) string {
	whole := basisPoints / 100
	fraction := basisPoints % 100
	return twoDigitDecimal(whole, fraction)
}

func twoDigitDecimal(whole int32, fraction int32) string {
	if fraction < 10 {
		return stringInt32(whole) + ".0" + stringInt32(fraction)
	}
	return stringInt32(whole) + "." + stringInt32(fraction)
}

func stringInt32(value int32) string {
	return strconv.FormatInt(int64(value), 10)
}

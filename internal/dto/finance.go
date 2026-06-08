package dto

import (
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/finance"
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

type TaxiParkFinanceSettingsResponse struct {
	TaxiParkID                  uuid.UUID `json:"taxi_park_id" example:"33333333-3333-3333-3333-333333333333"`
	DriverCommissionBasisPoints int32     `json:"driver_commission_basis_points" example:"2000"`
	DriverCommissionPercent     string    `json:"driver_commission_percent" example:"20.00"`
	PlatformFeeBasisPoints      int32     `json:"platform_fee_basis_points" example:"100"`
	PlatformFeePercent          string    `json:"platform_fee_percent" example:"1.00"`
	IsActive                    bool      `json:"is_active" example:"true"`
	CreatedAt                   time.Time `json:"created_at" example:"2026-05-12T12:00:00Z"`
	UpdatedAt                   time.Time `json:"updated_at" example:"2026-05-12T12:00:00Z"`
}

type UpdateTaxiParkDriverCommissionRequest struct {
	DriverCommissionBasisPoints int32 `json:"driver_commission_basis_points" binding:"required,min=0,max=10000" example:"2000"`
}

type TaxiParkFinanceOverviewResponse struct {
	TaxiParkID               uuid.UUID          `json:"taxi_park_id"`
	OrdersCount              int64              `json:"orders_count"`
	OrderTotalAmount         MoneyCentsResponse `json:"order_total_amount"`
	DriverIncomeAmount       MoneyCentsResponse `json:"driver_income_amount"`
	TaxiParkCommissionAmount MoneyCentsResponse `json:"taxi_park_commission_amount"`
	// TaxiParkIncomeAmount is the full taxi park income from the order commission before any platform fee settlement.
	TaxiParkIncomeAmount MoneyCentsResponse `json:"taxi_park_income_amount"`
	// PlatformServiceFeeAmount is accrued separately and must not reduce taxi_park_income_amount inside order settlement.
	PlatformServiceFeeAmount MoneyCentsResponse `json:"platform_service_fee_amount"`
	// PlatformDebtAmount is a separate taxi park obligation to the platform; example: 1000 RUB order -> 200 RUB taxi park income and 2 RUB platform debt.
	PlatformDebtAmount MoneyCentsResponse `json:"platform_debt_amount"`
	PeriodFrom         time.Time          `json:"period_from"`
	PeriodTo           time.Time          `json:"period_to"`
}

type OrderFinanceResponse struct {
	ID                          uuid.UUID          `json:"id"`
	OrderID                     uuid.UUID          `json:"order_id"`
	TaxiParkID                  *uuid.UUID         `json:"taxi_park_id,omitempty"`
	DriverID                    uuid.UUID          `json:"driver_id"`
	PassengerID                 *uuid.UUID         `json:"passenger_id,omitempty"`
	OrderTotalAmount            MoneyCentsResponse `json:"order_total_amount"`
	DriverCommissionBasisPoints int32              `json:"driver_commission_basis_points"`
	DriverCommissionPercent     string             `json:"driver_commission_percent"`
	TaxiParkCommissionAmount    MoneyCentsResponse `json:"taxi_park_commission_amount"`
	DriverIncomeAmount          MoneyCentsResponse `json:"driver_income_amount"`
	PlatformFeeBasisPoints      int32              `json:"platform_fee_basis_points"`
	PlatformFeePercent          string             `json:"platform_fee_percent"`
	// PlatformFeeAmount is recorded as a separate service fee accrual and is not deducted from taxi park order income.
	PlatformFeeAmount MoneyCentsResponse `json:"platform_fee_amount"`
	// TaxiParkIncomeAmount equals the full taxi park commission amount for the order; for example 200 RUB, not 198 RUB.
	TaxiParkIncomeAmount MoneyCentsResponse `json:"taxi_park_income_amount"`
	Status               string             `json:"status"`
	CreatedAt            time.Time          `json:"created_at"`
	UpdatedAt            time.Time          `json:"updated_at"`
}

type OrderFinancesResponse struct {
	Orders []OrderFinanceResponse `json:"orders"`
}

type CreateDriverPayoutRequest struct {
	AmountCents int64      `json:"amount_cents" binding:"required,min=1"`
	PeriodFrom  *time.Time `json:"period_from,omitempty"`
	PeriodTo    *time.Time `json:"period_to,omitempty"`
	Comment     string     `json:"comment,omitempty"`
}

type DriverPayoutResponse struct {
	ID                    uuid.UUID          `json:"id"`
	DriverID              uuid.UUID          `json:"driver_id"`
	TaxiParkID            uuid.UUID          `json:"taxi_park_id"`
	Amount                MoneyCentsResponse `json:"amount"`
	Status                string             `json:"status"`
	PeriodFrom            *time.Time         `json:"period_from,omitempty"`
	PeriodTo              *time.Time         `json:"period_to,omitempty"`
	PaymentMethod         string             `json:"payment_method,omitempty"`
	PaymentDocumentNumber string             `json:"payment_document_number,omitempty"`
	Comment               string             `json:"comment,omitempty"`
	CreatedBy             *uuid.UUID         `json:"created_by,omitempty"`
	CreatedAt             time.Time          `json:"created_at"`
	PaidAt                *time.Time         `json:"paid_at,omitempty"`
	UpdatedAt             time.Time          `json:"updated_at"`
}

type DriverPayoutsResponse struct {
	Payouts []DriverPayoutResponse `json:"payouts"`
}

type PlatformFeeDebtResponse struct {
	// Amount is the separate accumulated taxi park debt to the platform and must not be mixed with taxi park order income.
	Amount MoneyCentsResponse `json:"amount"`
}

type CreatePlatformInvoiceRequest struct {
	// PeriodFrom is the inclusive start of the platform fee billing period.
	PeriodFrom time.Time `json:"period_from" binding:"required"`
	// PeriodTo is the exclusive end of the platform fee billing period.
	PeriodTo time.Time `json:"period_to" binding:"required"`
}

type PlatformInvoiceResponse struct {
	ID         uuid.UUID `json:"id"`
	TaxiParkID uuid.UUID `json:"taxi_park_id"`
	// Amount is the billed platform service fee amount derived from separate accruals for the selected period.
	Amount        MoneyCentsResponse `json:"amount"`
	PeriodFrom    time.Time          `json:"period_from"`
	PeriodTo      time.Time          `json:"period_to"`
	Status        string             `json:"status"`
	InvoiceNumber string             `json:"invoice_number"`
	DocumentURL   string             `json:"document_url,omitempty"`
	CreatedAt     time.Time          `json:"created_at"`
	PaidAt        *time.Time         `json:"paid_at,omitempty"`
	UpdatedAt     time.Time          `json:"updated_at"`
}

type PlatformInvoicesResponse struct {
	Invoices []PlatformInvoiceResponse `json:"invoices"`
}

type FinanceDocumentResponse struct {
	ID         uuid.UUID  `json:"id"`
	TaxiParkID *uuid.UUID `json:"taxi_park_id,omitempty"`
	DriverID   *uuid.UUID `json:"driver_id,omitempty"`
	OrderID    *uuid.UUID `json:"order_id,omitempty"`
	PayoutID   *uuid.UUID `json:"payout_id,omitempty"`
	InvoiceID  *uuid.UUID `json:"invoice_id,omitempty"`
	Type       string     `json:"type"`
	Number     string     `json:"number"`
	Status     string     `json:"status"`
	FileURL    string     `json:"file_url,omitempty"`
	// Payload contains the generated JSON finance document body until PDF rendering is implemented.
	Payload   string    `json:"payload"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type FinanceDocumentsResponse struct {
	Documents []FinanceDocumentResponse `json:"documents"`
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
	ID                            uuid.UUID                          `json:"id" example:"22222222-2222-2222-2222-222222222222"`
	UserID                        uuid.UUID                          `json:"user_id" example:"66666666-6666-6666-6666-666666666666"`
	Phone                         string                             `json:"phone" example:"+79990000001"`
	Email                         string                             `json:"email,omitempty" example:"driver@example.com"`
	FirstName                     string                             `json:"first_name,omitempty" example:"Ivan"`
	LastName                      string                             `json:"last_name,omitempty" example:"Petrov"`
	FullName                      string                             `json:"full_name" example:"Ivan Petrov"`
	Status                        domain.DriverStatus                `json:"status" example:"offline"`
	VerificationStatus            domain.VerificationLifecycleStatus `json:"verification_status" example:"pending_verification"`
	Rating                        float64                            `json:"rating" example:"4.95"`
	RatingsCount                  int                                `json:"ratings_count" example:"0"`
	BirthDate                     *time.Time                         `json:"birth_date,omitempty" example:"1990-01-31T00:00:00Z"`
	LicenseSeries                 string                             `json:"license_series,omitempty" example:"77 01"`
	LicenseNumber                 string                             `json:"license_number,omitempty" example:"7700000000"`
	LicenseCategory               string                             `json:"license_category,omitempty" example:"B"`
	LicenseIssuedAt               *time.Time                         `json:"license_issued_at,omitempty" example:"2020-01-31T00:00:00Z"`
	LicenseExpiresAt              *time.Time                         `json:"license_expires_at,omitempty" example:"2030-01-31T00:00:00Z"`
	DrivingExperienceFrom         *time.Time                         `json:"driving_experience_from,omitempty" example:"2015-01-31T00:00:00Z"`
	HasNoTaxiWorkRestrictions     bool                               `json:"has_no_taxi_work_restrictions" example:"true"`
	FederalLaw580Compliant        bool                               `json:"federal_law_580_compliant" example:"true"`
	RegionalRequirementsCompliant bool                               `json:"regional_requirements_compliant" example:"true"`
	MedicalCheckPassed            bool                               `json:"medical_check_passed" example:"true"`
	PretripControlRequired        bool                               `json:"pretrip_control_required" example:"true"`
	PretripControlPassed          bool                               `json:"pretrip_control_passed" example:"true"`
	NoTransportBan                bool                               `json:"no_transport_ban" example:"true"`
	VerificationCheckedAt         *time.Time                         `json:"verification_checked_at,omitempty" example:"2026-05-19T12:00:00Z"`
	VerificationCheckedBy         *uuid.UUID                         `json:"verification_checked_by,omitempty" example:"11111111-1111-1111-1111-111111111111"`
	IsVerified                    bool                               `json:"is_verified" example:"false"`
	BlockedReason                 string                             `json:"blocked_reason,omitempty" example:"Expired license"`
	TaxiParkComment               string                             `json:"taxi_park_comment,omitempty" example:"Documents checked"`
	CreatedAt                     time.Time                          `json:"created_at" example:"2026-05-12T12:00:00Z"`
	UpdatedAt                     time.Time                          `json:"updated_at" example:"2026-05-12T12:00:00Z"`
}

type TaxiParkDriversResponse struct {
	Drivers []TaxiParkDriverResponse `json:"drivers"`
}

type TaxiParkOrderResponse struct {
	ID          uuid.UUID          `json:"id" example:"55555555-5555-5555-5555-555555555555"`
	DriverID    *uuid.UUID         `json:"driver_id,omitempty" example:"22222222-2222-2222-2222-222222222222"`
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

func TaxiParkFinanceSettingsFromDomain(settings domain.TaxiParkFinanceSettings) TaxiParkFinanceSettingsResponse {
	return TaxiParkFinanceSettingsResponse{
		TaxiParkID:                  settings.TaxiParkID,
		DriverCommissionBasisPoints: settings.DriverCommissionRate.BasisPoints,
		DriverCommissionPercent:     FormatBasisPoints(settings.DriverCommissionRate.BasisPoints),
		PlatformFeeBasisPoints:      settings.PlatformFeeRate.BasisPoints,
		PlatformFeePercent:          FormatBasisPoints(settings.PlatformFeeRate.BasisPoints),
		IsActive:                    settings.IsActive,
		CreatedAt:                   settings.CreatedAt,
		UpdatedAt:                   settings.UpdatedAt,
	}
}

func TaxiParkFinanceOverviewFromFinance(overview finance.TaxiParkFinanceOverview) TaxiParkFinanceOverviewResponse {
	return TaxiParkFinanceOverviewResponse{
		TaxiParkID:               overview.TaxiParkID,
		OrdersCount:              overview.OrdersCount,
		OrderTotalAmount:         MoneyCentsFromDomain(overview.OrderTotalAmount),
		DriverIncomeAmount:       MoneyCentsFromDomain(overview.DriverIncomeAmount),
		TaxiParkCommissionAmount: MoneyCentsFromDomain(overview.TaxiParkCommissionAmount),
		TaxiParkIncomeAmount:     MoneyCentsFromDomain(overview.TaxiParkIncomeAmount),
		PlatformServiceFeeAmount: MoneyCentsFromDomain(overview.PlatformServiceFeeAmount),
		PlatformDebtAmount:       MoneyCentsFromDomain(overview.PlatformDebtAmount),
		PeriodFrom:               overview.PeriodFrom,
		PeriodTo:                 overview.PeriodTo,
	}
}

func OrderFinanceFromFinance(item finance.OrderFinance) OrderFinanceResponse {
	return OrderFinanceResponse{
		ID:                          item.ID,
		OrderID:                     item.OrderID,
		TaxiParkID:                  item.TaxiParkID,
		DriverID:                    item.DriverID,
		PassengerID:                 item.PassengerID,
		OrderTotalAmount:            MoneyCentsFromDomain(item.OrderTotalAmount),
		DriverCommissionBasisPoints: item.DriverCommissionBasisPoints,
		DriverCommissionPercent:     FormatBasisPoints(item.DriverCommissionBasisPoints),
		TaxiParkCommissionAmount:    MoneyCentsFromDomain(item.TaxiParkCommissionAmount),
		DriverIncomeAmount:          MoneyCentsFromDomain(item.DriverIncomeAmount),
		PlatformFeeBasisPoints:      item.PlatformFeeBasisPoints,
		PlatformFeePercent:          FormatBasisPoints(item.PlatformFeeBasisPoints),
		PlatformFeeAmount:           MoneyCentsFromDomain(item.PlatformFeeAmount),
		TaxiParkIncomeAmount:        MoneyCentsFromDomain(item.TaxiParkIncomeAmount),
		Status:                      item.Status,
		CreatedAt:                   item.CreatedAt,
		UpdatedAt:                   item.UpdatedAt,
	}
}

func OrderFinancesFromFinance(items []finance.OrderFinance) OrderFinancesResponse {
	response := OrderFinancesResponse{Orders: make([]OrderFinanceResponse, 0, len(items))}
	for _, item := range items {
		response.Orders = append(response.Orders, OrderFinanceFromFinance(item))
	}
	return response
}

func DriverPayoutFromFinance(item finance.DriverPayout) DriverPayoutResponse {
	return DriverPayoutResponse{
		ID:                    item.ID,
		DriverID:              item.DriverID,
		TaxiParkID:            item.TaxiParkID,
		Amount:                MoneyCentsFromDomain(item.Amount),
		Status:                item.Status,
		PeriodFrom:            item.PeriodFrom,
		PeriodTo:              item.PeriodTo,
		PaymentMethod:         item.PaymentMethod,
		PaymentDocumentNumber: item.PaymentDocumentNumber,
		Comment:               item.Comment,
		CreatedBy:             item.CreatedBy,
		CreatedAt:             item.CreatedAt,
		PaidAt:                item.PaidAt,
		UpdatedAt:             item.UpdatedAt,
	}
}

func DriverPayoutsFromFinance(items []finance.DriverPayout) DriverPayoutsResponse {
	response := DriverPayoutsResponse{Payouts: make([]DriverPayoutResponse, 0, len(items))}
	for _, item := range items {
		response.Payouts = append(response.Payouts, DriverPayoutFromFinance(item))
	}
	return response
}

func PlatformInvoiceFromFinance(item finance.PlatformInvoice) PlatformInvoiceResponse {
	return PlatformInvoiceResponse{
		ID:            item.ID,
		TaxiParkID:    item.TaxiParkID,
		Amount:        MoneyCentsFromDomain(item.Amount),
		PeriodFrom:    item.PeriodFrom,
		PeriodTo:      item.PeriodTo,
		Status:        item.Status,
		InvoiceNumber: item.InvoiceNumber,
		DocumentURL:   item.DocumentURL,
		CreatedAt:     item.CreatedAt,
		PaidAt:        item.PaidAt,
		UpdatedAt:     item.UpdatedAt,
	}
}

func PlatformInvoicesFromFinance(items []finance.PlatformInvoice) PlatformInvoicesResponse {
	response := PlatformInvoicesResponse{Invoices: make([]PlatformInvoiceResponse, 0, len(items))}
	for _, item := range items {
		response.Invoices = append(response.Invoices, PlatformInvoiceFromFinance(item))
	}
	return response
}

func FinanceDocumentsFromFinance(items []finance.FinanceDocument) FinanceDocumentsResponse {
	response := FinanceDocumentsResponse{Documents: make([]FinanceDocumentResponse, 0, len(items))}
	for _, item := range items {
		response.Documents = append(response.Documents, FinanceDocumentResponse{
			ID:         item.ID,
			TaxiParkID: item.TaxiParkID,
			DriverID:   item.DriverID,
			OrderID:    item.OrderID,
			PayoutID:   item.PayoutID,
			InvoiceID:  item.InvoiceID,
			Type:       item.Type,
			Number:     item.Number,
			Status:     item.Status,
			FileURL:    item.FileURL,
			Payload:    string(item.Payload),
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  item.UpdatedAt,
		})
	}
	return response
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

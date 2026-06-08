package finance

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TaxiParkFinanceOverview struct {
	TaxiParkID               uuid.UUID
	OrdersCount              int64
	OrderTotalAmount         domain.Money
	DriverIncomeAmount       domain.Money
	TaxiParkCommissionAmount domain.Money
	TaxiParkIncomeAmount     domain.Money
	PlatformServiceFeeAmount domain.Money
	PlatformDebtAmount       domain.Money
	PeriodFrom               time.Time
	PeriodTo                 time.Time
}

type OrderFinance struct {
	ID                          uuid.UUID
	OrderID                     uuid.UUID
	TaxiParkID                  *uuid.UUID
	DriverID                    uuid.UUID
	PassengerID                 *uuid.UUID
	OrderTotalAmount            domain.Money
	DriverCommissionBasisPoints int32
	TaxiParkCommissionAmount    domain.Money
	DriverIncomeAmount          domain.Money
	PlatformFeeBasisPoints      int32
	PlatformFeeAmount           domain.Money
	TaxiParkIncomeAmount        domain.Money
	Status                      string
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}

type DriverPayout struct {
	ID                    uuid.UUID
	DriverID              uuid.UUID
	TaxiParkID            uuid.UUID
	Amount                domain.Money
	Status                string
	PeriodFrom            *time.Time
	PeriodTo              *time.Time
	PaymentMethod         string
	PaymentDocumentNumber string
	Comment               string
	CreatedBy             *uuid.UUID
	CreatedAt             time.Time
	PaidAt                *time.Time
	UpdatedAt             time.Time
}

type PlatformInvoice struct {
	ID            uuid.UUID
	TaxiParkID    uuid.UUID
	Amount        domain.Money
	PeriodFrom    time.Time
	PeriodTo      time.Time
	Status        string
	InvoiceNumber string
	DocumentURL   string
	CreatedAt     time.Time
	PaidAt        *time.Time
	UpdatedAt     time.Time
}

type FinanceDocument struct {
	ID         uuid.UUID
	TaxiParkID *uuid.UUID
	DriverID   *uuid.UUID
	OrderID    *uuid.UUID
	PayoutID   *uuid.UUID
	InvoiceID  *uuid.UUID
	Type       string
	Number     string
	Status     string
	FileURL    string
	Payload    []byte
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateDriverPayoutInput struct {
	AmountCents int64
	PeriodFrom  *time.Time
	PeriodTo    *time.Time
	Comment     string
}

type CreatePlatformInvoiceInput struct {
	PeriodFrom time.Time
	PeriodTo   time.Time
}

package handler

import (
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type DriverBalanceSuccessResponse struct {
	Data dto.DriverBalanceResponse `json:"data"`
	Meta response.Meta             `json:"meta"`
}

type TaxiParkBalanceSuccessResponse struct {
	Data dto.TaxiParkBalanceResponse `json:"data"`
	Meta response.Meta               `json:"meta"`
}

type TaxiParkFinanceSettingsSuccessResponse struct {
	Data dto.TaxiParkFinanceSettingsResponse `json:"data"`
	Meta response.Meta                       `json:"meta"`
}

type FinancialTransactionsSuccessResponse struct {
	Data dto.FinancialTransactionsResponse `json:"data"`
	Meta response.Meta                     `json:"meta"`
}

type TaxiParkDriversSuccessResponse struct {
	Data dto.TaxiParkDriversResponse `json:"data"`
	Meta response.Meta               `json:"meta"`
}

type TaxiParkOrdersSuccessResponse struct {
	Data dto.TaxiParkOrdersResponse `json:"data"`
	Meta response.Meta              `json:"meta"`
}

type AdminFinanceOverviewSuccessResponse struct {
	Data dto.AdminFinanceOverviewResponse `json:"data"`
	Meta response.Meta                    `json:"meta"`
}

type TaxiParkFinanceOverviewSuccessResponse struct {
	Data dto.TaxiParkFinanceOverviewResponse `json:"data"`
	Meta response.Meta                       `json:"meta"`
}

type OrderFinancesSuccessResponse struct {
	Data dto.OrderFinancesResponse `json:"data"`
	Meta response.Meta             `json:"meta"`
}

type DriverPayoutsSuccessResponse struct {
	Data dto.DriverPayoutsResponse `json:"data"`
	Meta response.Meta             `json:"meta"`
}

type DriverPayoutSuccessResponse struct {
	Data dto.DriverPayoutResponse `json:"data"`
	Meta response.Meta            `json:"meta"`
}

type PlatformFeeDebtSuccessResponse struct {
	Data dto.PlatformFeeDebtResponse `json:"data"`
	Meta response.Meta               `json:"meta"`
}

type PlatformInvoicesSuccessResponse struct {
	Data dto.PlatformInvoicesResponse `json:"data"`
	Meta response.Meta                `json:"meta"`
}

type PlatformInvoiceSuccessResponse struct {
	Data dto.PlatformInvoiceResponse `json:"data"`
	Meta response.Meta               `json:"meta"`
}

type FinanceDocumentsSuccessResponse struct {
	Data dto.FinanceDocumentsResponse `json:"data"`
	Meta response.Meta                `json:"meta"`
}

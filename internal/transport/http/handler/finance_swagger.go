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

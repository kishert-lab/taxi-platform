package handler

import (
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type TaxiParkSettingsSuccessResponse struct {
	Data dto.TaxiParkSettingsResponse `json:"data"`
	Meta response.Meta                `json:"meta"`
}

type TaxiParkTariffSuccessResponse struct {
	Data dto.TaxiParkTariffResponse `json:"data"`
	Meta response.Meta              `json:"meta"`
}

type TaxiParkTariffsSuccessResponse struct {
	Data dto.TaxiParkTariffsResponse `json:"data"`
	Meta response.Meta               `json:"meta"`
}

type TaxiParkCreateDriverSuccessResponse struct {
	Data dto.TaxiParkCreateDriverResponse `json:"data"`
	Meta response.Meta                    `json:"meta"`
}

type TaxiParkDispatcherSuccessResponse struct {
	Data dto.TaxiParkDispatcherResponse `json:"data"`
	Meta response.Meta                  `json:"meta"`
}

type TaxiParkDispatchersSuccessResponse struct {
	Data dto.TaxiParkDispatchersResponse `json:"data"`
	Meta response.Meta                   `json:"meta"`
}

type ScheduledOrderSuccessResponse struct {
	Data dto.ScheduledOrderResponse `json:"data"`
	Meta response.Meta              `json:"meta"`
}

type ScheduledOrdersSuccessResponse struct {
	Data dto.ScheduledOrdersResponse `json:"data"`
	Meta response.Meta               `json:"meta"`
}

type TaxiParkDriverPasswordSuccessResponse struct {
	Data dto.TaxiParkDriverPasswordResponse `json:"data"`
	Meta response.Meta                      `json:"meta"`
}

type TaxiParkCarSuccessResponse struct {
	Data dto.TaxiParkCarResponse `json:"data"`
	Meta response.Meta           `json:"meta"`
}

type TaxiParkCarsSuccessResponse struct {
	Data dto.TaxiParkCarsResponse `json:"data"`
	Meta response.Meta            `json:"meta"`
}

type TaxiParkDriverLocationsSuccessResponse struct {
	Data dto.TaxiParkDriverLocationsResponse `json:"data"`
	Meta response.Meta                       `json:"meta"`
}

type TaxiParkDocumentsSuccessResponse struct {
	Data dto.TaxiParkDocumentsResponse `json:"data"`
	Meta response.Meta                 `json:"meta"`
}

type LegalDocumentSuccessResponse struct {
	Data dto.LegalDocumentResponse `json:"data"`
	Meta response.Meta             `json:"meta"`
}

type LegalDocumentsSuccessResponse struct {
	Data dto.LegalDocumentsResponse `json:"data"`
	Meta response.Meta              `json:"meta"`
}

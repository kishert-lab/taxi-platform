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

type LegalDocumentSuccessResponse struct {
	Data dto.LegalDocumentResponse `json:"data"`
	Meta response.Meta             `json:"meta"`
}

type LegalDocumentsSuccessResponse struct {
	Data dto.LegalDocumentsResponse `json:"data"`
	Meta response.Meta              `json:"meta"`
}

package handler

import (
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerProfileSuccessResponse struct {
	Data dto.PassengerProfileResponse `json:"data"`
	Meta response.Meta                `json:"meta"`
}

type DriverProfileSuccessResponse struct {
	Data dto.DriverProfileResponse `json:"data"`
	Meta response.Meta             `json:"meta"`
}

type ProfilePhotoUploadSuccessResponse struct {
	Data dto.ProfilePhotoUploadResponse `json:"data"`
	Meta response.Meta                  `json:"meta"`
}

type PassengerPushTokenSuccessResponse struct {
	Data dto.PassengerPushTokenResponse `json:"data"`
	Meta response.Meta                  `json:"meta"`
}

type OrderEstimateSuccessResponse struct {
	Data dto.OrderEstimateResponse `json:"data"`
	Meta response.Meta             `json:"meta"`
}

type OrderSuccessResponse struct {
	Data dto.OrderResponse `json:"data"`
	Meta response.Meta     `json:"meta"`
}

type PassengerOrderSuccessResponse struct {
	Data dto.PassengerOrderResponse `json:"data"`
	Meta response.Meta              `json:"meta"`
}

type PassengerOrderHistorySuccessResponse struct {
	Data dto.OrderHistoryResponse `json:"data"`
	Meta response.Meta            `json:"meta"`
}

type PassengerCarClassesSuccessResponse struct {
	Data dto.PassengerCarClassesResponse `json:"data"`
	Meta response.Meta                   `json:"meta"`
}

type DriverOrderSuccessResponse struct {
	Data dto.DriverOrderResponse `json:"data"`
	Meta response.Meta           `json:"meta"`
}

type DriverOrderHistorySuccessResponse struct {
	Data dto.DriverOrderHistoryResponse `json:"data"`
	Meta response.Meta                  `json:"meta"`
}

type DriverOrderOffersSuccessResponse struct {
	Data dto.DriverOrderOffersResponse `json:"data"`
	Meta response.Meta                 `json:"meta"`
}

type DriverOrderRouteSuccessResponse struct {
	Data dto.OrderRouteResponse `json:"data"`
	Meta response.Meta          `json:"meta"`
}

type DriverOrderRouteBatchSuccessResponse struct {
	Data dto.DriverOrderRouteBatchResponse `json:"data"`
	Meta response.Meta                     `json:"meta"`
}

type AuthCodeSentSuccessResponse struct {
	Data dto.AuthCodeSentResponse `json:"data"`
	Meta response.Meta            `json:"meta"`
}

type AuthTokenSuccessResponse struct {
	Data dto.AuthTokenResponse `json:"data"`
	Meta response.Meta         `json:"meta"`
}

type AcceptedSuccessResponse struct {
	Data acceptedResponse `json:"data"`
	Meta response.Meta    `json:"meta"`
}

type acceptedResponse struct {
	Accepted bool `json:"accepted" example:"true"`
}

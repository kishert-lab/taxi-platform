package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

// EstimateOrder godoc
// @Summary Estimate a dispatcher order using road distance and duration
// @Description Uses the selected authorized park tariff, preserving fixed/distance/time/combined modes. Snapshot amounts are kopecks. OSRM duration excludes live traffic. Does not create an order.
// @Tags taxi-park-orders
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body dto.TaxiParkCreateOrderRequest true "Order form"
// @Success 200 {object} TaxiParkEstimateResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 422 {object} response.Error
// @Failure 503 {object} response.Error
// @Router /taxi-park/orders/estimate [post]
func (handler *TaxiParkSettingsHandler) EstimateOrder(context *gin.Context) {
	actorID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var request dto.TaxiParkCreateOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid order estimate request")
		return
	}
	result, err := handler.useCase.EstimateOrder(context.Request.Context(), actorID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.PricingSnapshotResponse(result))
}

type TaxiParkEstimateResponse struct {
	Data dto.OrderPricingResponse `json:"data"`
	Meta response.Meta            `json:"meta"`
}

package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerAddressHandler struct {
	useCase PassengerAddressUseCase
}

func NewPassengerAddressHandler(useCase PassengerAddressUseCase) *PassengerAddressHandler {
	return &PassengerAddressHandler{useCase: useCase}
}

func (handler *PassengerAddressHandler) RegisterRoutes(router gin.IRouter, passengerAuthMiddleware gin.HandlerFunc) {
	protected := router.Group("/passenger", passengerAuthMiddleware)
	protected.GET("/address/search", handler.SearchAddresses)
}

// SearchAddresses godoc
// @Summary Search passenger addresses
// @Description Searches address suggestions for passenger mobile order form.
// @Tags passenger
// @Produce json
// @Param Authorization header string true "Bearer passenger access token"
// @Param q query string true "Address query"
// @Param city_id query string false "City UUID"
// @Param lat query number false "Focus latitude"
// @Param lon query number false "Focus longitude"
// @Param limit query int false "Result limit"
// @Success 200 {object} PassengerAddressSearchSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /passenger/address/search [get]
func (handler *PassengerAddressHandler) SearchAddresses(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	query := context.Query("q")
	cityID, focusLatitude, focusLongitude, limit, err := passengerAddressSearchQuery(context)
	if err != nil {
		failValidation(context, "Invalid address search request")
		return
	}

	results, err := handler.useCase.SearchPassengerAddresses(
		context.Request.Context(),
		passengerID,
		query,
		cityID,
		focusLatitude,
		focusLongitude,
		limit,
	)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, PassengerAddressSearchResponse{Results: passengerAddressSearchResults(results)})
}

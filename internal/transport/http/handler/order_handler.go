package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/develoop/taxi-platform/internal/domain"
	"github.com/develoop/taxi-platform/internal/dto"
)

type CurrentOrderUseCase interface {
	CurrentForPassenger(ctx context.Context, passengerID uuid.UUID) (domain.Order, error)
}

type OrderHandler struct {
	currentOrderUseCase CurrentOrderUseCase
}

func NewOrderHandler(currentOrderUseCase CurrentOrderUseCase) *OrderHandler {
	return &OrderHandler{currentOrderUseCase: currentOrderUseCase}
}

func (handler *OrderHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/orders/current", handler.Current)
}

// Current godoc
// @Summary Get current order
// @Description Returns current active trip state for reconnect recovery.
// @Tags orders
// @Produce json
// @Security BearerAuth
// @Success 200 {object} dto.CurrentOrderResponse
// @Failure 401 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /orders/current [get]
func (handler *OrderHandler) Current(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		context.JSON(http.StatusUnauthorized, errorResponse{Error: "user id is missing"})
		return
	}

	order, err := handler.currentOrderUseCase.CurrentForPassenger(context.Request.Context(), passengerID)
	if err != nil {
		context.JSON(http.StatusNotFound, errorResponse{Error: "current order not found"})
		return
	}

	context.JSON(http.StatusOK, dto.CurrentOrderResponse{Order: orderToResponse(order)})
}

func userIDFromContext(context *gin.Context) (uuid.UUID, bool) {
	value, exists := context.Get("user_id")
	if !exists {
		return uuid.Nil, false
	}

	switch typedValue := value.(type) {
	case uuid.UUID:
		return typedValue, true
	case string:
		parsedValue, err := uuid.Parse(typedValue)
		return parsedValue, err == nil
	default:
		return uuid.Nil, false
	}
}

func orderToResponse(order domain.Order) dto.OrderResponse {
	response := dto.OrderResponse{
		ID:                 order.ID,
		PassengerID:        order.PassengerID,
		DriverID:           order.DriverID,
		CityID:             order.CityID,
		TariffID:           order.TariffID,
		Status:             order.Status,
		Version:            order.Version,
		PickupAddress:      order.PickupAddress,
		PickupLocation:     dto.CoordinatesResponse{Latitude: order.PickupLocation.Latitude, Longitude: order.PickupLocation.Longitude},
		DestinationAddress: order.DestinationAddress,
		PaymentMethod:      order.PaymentMethod,
	}
	if order.EstimatedPrice != nil {
		response.EstimatedPrice = &dto.MoneyResponse{Amount: order.EstimatedPrice.Amount, Currency: order.EstimatedPrice.Currency}
	}
	if order.FinalPrice != nil {
		response.FinalPrice = &dto.MoneyResponse{Amount: order.FinalPrice.Amount, Currency: order.FinalPrice.Currency}
	}

	return response
}

package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerOrdersUseCase interface {
	EstimatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.OrderEstimateRequest) (dto.OrderEstimateResponse, error)
	CreatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error)
	GetCurrentPassengerOrder(ctx context.Context, passengerID uuid.UUID) (dto.PassengerOrderResponse, error)
	ListPassengerOrderHistory(ctx context.Context, passengerID uuid.UUID) (dto.OrderHistoryResponse, error)
	GetPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (dto.PassengerOrderResponse, error)
	CancelPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.CancelOrderRequest) (dto.PassengerOrderResponse, error)
	RatePassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.RateOrderRequest) (dto.PassengerOrderResponse, error)
}

type PassengerOrdersHandler struct {
	useCase PassengerOrdersUseCase
}

func NewPassengerOrdersHandler(useCase PassengerOrdersUseCase) *PassengerOrdersHandler {
	return &PassengerOrdersHandler{useCase: useCase}
}

func (handler *PassengerOrdersHandler) RegisterRoutes(router gin.IRouter, passengerAuthMiddleware gin.HandlerFunc) {
	protected := router.Group("/passenger", passengerAuthMiddleware)
	protected.POST("/orders/estimate", handler.EstimateOrder)
	protected.POST("/orders", handler.CreateOrder)
	protected.GET("/orders/current", handler.CurrentOrder)
	protected.GET("/orders/history", handler.OrderHistory)
	protected.GET("/orders/:id", handler.GetOrder)
	protected.POST("/orders/:id/cancel", handler.CancelOrder)
	protected.POST("/orders/:id/rate", handler.RateOrder)
}

// EstimateOrder godoc
// @Summary Estimate passenger order
// @Description Returns estimated distance, duration and tariff price for mobile order form.
// @Tags passenger-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.OrderEstimateRequest true "Order estimate request"
// @Success 200 {object} OrderEstimateSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/orders/estimate [post]
func (handler *PassengerOrdersHandler) EstimateOrder(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	var request dto.OrderEstimateRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid order estimate request")
		return
	}

	result, err := handler.useCase.EstimatePassengerOrder(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// CreateOrder godoc
// @Summary Create passenger order
// @Tags passenger-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.PassengerCreateOrderRequest true "Order request"
// @Success 201 {object} PassengerOrderSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /passenger/orders [post]
func (handler *PassengerOrdersHandler) CreateOrder(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	var request dto.PassengerCreateOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid order request")
		return
	}

	result, err := handler.useCase.CreatePassengerOrder(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.Created(context, result)
}

// CurrentOrder godoc
// @Summary Get current passenger order
// @Description Mobile reconnect sync endpoint. Response includes allowed_actions for passenger UI.
// @Tags passenger-orders
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PassengerOrderSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /passenger/orders/current [get]
func (handler *PassengerOrdersHandler) CurrentOrder(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	result, err := handler.useCase.GetCurrentPassengerOrder(context.Request.Context(), passengerID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// OrderHistory godoc
// @Summary Get passenger order history
// @Tags passenger-orders
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PassengerOrderHistorySuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/orders/history [get]
func (handler *PassengerOrdersHandler) OrderHistory(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	result, err := handler.useCase.ListPassengerOrderHistory(context.Request.Context(), passengerID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// GetOrder godoc
// @Summary Get passenger order by id
// @Tags passenger-orders
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} PassengerOrderSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /passenger/orders/{id} [get]
func (handler *PassengerOrdersHandler) GetOrder(context *gin.Context) {
	passengerID, orderID, ok := passengerOrderIDsFromPassengerContext(context)
	if !ok {
		return
	}

	result, err := handler.useCase.GetPassengerOrder(context.Request.Context(), passengerID, orderID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// CancelOrder godoc
// @Summary Cancel passenger order
// @Tags passenger-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.CancelOrderRequest true "Cancellation reason"
// @Success 200 {object} PassengerOrderSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /passenger/orders/{id}/cancel [post]
func (handler *PassengerOrdersHandler) CancelOrder(context *gin.Context) {
	passengerID, orderID, ok := passengerOrderIDsFromPassengerContext(context)
	if !ok {
		return
	}

	var request dto.CancelOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid cancellation request")
		return
	}

	result, err := handler.useCase.CancelPassengerOrder(context.Request.Context(), passengerID, orderID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// RateOrder godoc
// @Summary Rate completed passenger order
// @Tags passenger-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.RateOrderRequest true "Order rating"
// @Success 200 {object} PassengerOrderSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /passenger/orders/{id}/rate [post]
func (handler *PassengerOrdersHandler) RateOrder(context *gin.Context) {
	passengerID, orderID, ok := passengerOrderIDsFromPassengerContext(context)
	if !ok {
		return
	}

	var request dto.RateOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid rating request")
		return
	}

	result, err := handler.useCase.RatePassengerOrder(context.Request.Context(), passengerID, orderID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

func passengerOrderIDsFromPassengerContext(context *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return uuid.Nil, uuid.Nil, false
	}

	orderID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid order id")
		return uuid.Nil, uuid.Nil, false
	}

	return passengerID, orderID, true
}

package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerProfileUseCase interface {
	CreatePassengerProfile(ctx context.Context, passengerID uuid.UUID, request dto.PassengerProfileRequest) (dto.PassengerProfileResponse, error)
	GetPassengerProfile(ctx context.Context, passengerID uuid.UUID) (dto.PassengerProfileResponse, error)
	UpdatePassengerProfile(ctx context.Context, passengerID uuid.UUID, request dto.PassengerProfilePatchRequest) (dto.PassengerProfileResponse, error)
}

type PassengerOrderUseCase interface {
	EstimatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.OrderEstimateRequest) (dto.OrderEstimateResponse, error)
	CreatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error)
	GetCurrentPassengerOrder(ctx context.Context, passengerID uuid.UUID) (dto.PassengerOrderResponse, error)
	ListPassengerOrderHistory(ctx context.Context, passengerID uuid.UUID) (dto.OrderHistoryResponse, error)
	GetPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (dto.PassengerOrderResponse, error)
	CancelPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.CancelOrderRequest) (dto.PassengerOrderResponse, error)
	RatePassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.RateOrderRequest) (dto.PassengerOrderResponse, error)
}

type PassengerMobileHandler struct {
	profileUseCase PassengerProfileUseCase
	orderUseCase   PassengerOrderUseCase
}

func NewPassengerMobileHandler(profileUseCase PassengerProfileUseCase, orderUseCase PassengerOrderUseCase) *PassengerMobileHandler {
	return &PassengerMobileHandler{profileUseCase: profileUseCase, orderUseCase: orderUseCase}
}

func (handler *PassengerMobileHandler) RegisterRoutes(router gin.IRouter) {
	passenger := router.Group("/passenger", middleware.RequireRole(domain.UserRolePassenger))
	passenger.POST("/profile", handler.CreateProfile)
	passenger.GET("/profile", handler.GetProfile)
	passenger.PATCH("/profile", handler.UpdateProfile)
	passenger.POST("/orders/estimate", handler.EstimateOrder)
	passenger.POST("/orders", handler.CreateOrder)
	passenger.GET("/orders/current", handler.CurrentOrder)
	passenger.GET("/orders/history", handler.OrderHistory)
	passenger.GET("/orders/:id", handler.GetOrder)
	passenger.POST("/orders/:id/cancel", handler.CancelOrder)
	passenger.POST("/orders/:id/rate", handler.RateOrder)
}

// CreateProfile godoc
// @Summary Create passenger profile
// @Tags passenger
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.PassengerProfileRequest true "Passenger profile"
// @Success 201 {object} PassengerProfileSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/profile [post]
func (handler *PassengerMobileHandler) CreateProfile(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.PassengerProfileRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger profile request")
		return
	}

	result, err := handler.profileUseCase.CreatePassengerProfile(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.Created(context, result)
}

// GetProfile godoc
// @Summary Get passenger profile
// @Tags passenger
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PassengerProfileSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/profile [get]
func (handler *PassengerMobileHandler) GetProfile(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.profileUseCase.GetPassengerProfile(context.Request.Context(), passengerID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// UpdateProfile godoc
// @Summary Update passenger profile
// @Tags passenger
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.PassengerProfilePatchRequest true "Passenger profile patch"
// @Success 200 {object} PassengerProfileSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/profile [patch]
func (handler *PassengerMobileHandler) UpdateProfile(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.PassengerProfilePatchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger profile patch")
		return
	}

	result, err := handler.profileUseCase.UpdatePassengerProfile(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
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
func (handler *PassengerMobileHandler) EstimateOrder(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.OrderEstimateRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid order estimate request")
		return
	}

	result, err := handler.orderUseCase.EstimatePassengerOrder(context.Request.Context(), passengerID, request)
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
func (handler *PassengerMobileHandler) CreateOrder(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.PassengerCreateOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid order request")
		return
	}

	result, err := handler.orderUseCase.CreatePassengerOrder(context.Request.Context(), passengerID, request)
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
func (handler *PassengerMobileHandler) CurrentOrder(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.orderUseCase.GetCurrentPassengerOrder(context.Request.Context(), passengerID)
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
func (handler *PassengerMobileHandler) OrderHistory(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.orderUseCase.ListPassengerOrderHistory(context.Request.Context(), passengerID)
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
func (handler *PassengerMobileHandler) GetOrder(context *gin.Context) {
	passengerID, orderID, ok := passengerOrderIDs(context)
	if !ok {
		return
	}

	result, err := handler.orderUseCase.GetPassengerOrder(context.Request.Context(), passengerID, orderID)
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
func (handler *PassengerMobileHandler) CancelOrder(context *gin.Context) {
	passengerID, orderID, ok := passengerOrderIDs(context)
	if !ok {
		return
	}

	var request dto.CancelOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid cancellation request")
		return
	}

	result, err := handler.orderUseCase.CancelPassengerOrder(context.Request.Context(), passengerID, orderID, request)
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
func (handler *PassengerMobileHandler) RateOrder(context *gin.Context) {
	passengerID, orderID, ok := passengerOrderIDs(context)
	if !ok {
		return
	}

	var request dto.RateOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid rating request")
		return
	}

	result, err := handler.orderUseCase.RatePassengerOrder(context.Request.Context(), passengerID, orderID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

func passengerOrderIDs(context *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return uuid.Nil, uuid.Nil, false
	}

	orderID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid order id")
		return uuid.Nil, uuid.Nil, false
	}

	return passengerID, orderID, true
}

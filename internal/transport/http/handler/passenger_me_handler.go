package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerMeUseCase interface {
	GetMe(ctx context.Context, passengerID uuid.UUID) (dto.PassengerMeResponse, error)
	UpdateMe(ctx context.Context, passengerID uuid.UUID, request dto.PassengerMePatchRequest) (dto.PassengerMeResponse, error)
}

type PassengerMeHandler struct {
	useCase PassengerMeUseCase
}

func NewPassengerMeHandler(useCase PassengerMeUseCase) *PassengerMeHandler {
	return &PassengerMeHandler{useCase: useCase}
}

func (handler *PassengerMeHandler) RegisterRoutes(router gin.IRouter, passengerAuthMiddleware gin.HandlerFunc) {
	protected := router.Group("/passenger", passengerAuthMiddleware)
	protected.GET("/me", handler.GetMe)
	protected.PATCH("/me", handler.UpdateMe)
}

func (handler *PassengerMeHandler) GetMe(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	result, err := handler.useCase.GetMe(context.Request.Context(), passengerID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

func (handler *PassengerMeHandler) UpdateMe(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	var request dto.PassengerMePatchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger profile patch")
		return
	}

	result, err := handler.useCase.UpdateMe(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

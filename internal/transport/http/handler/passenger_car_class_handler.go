package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerCarClassUseCase interface {
	ListPassengerCarClasses(ctx context.Context, passengerID uuid.UUID) (dto.PassengerCarClassesResponse, error)
}

type PassengerCarClassHandler struct {
	useCase PassengerCarClassUseCase
}

func NewPassengerCarClassHandler(useCase PassengerCarClassUseCase) *PassengerCarClassHandler {
	return &PassengerCarClassHandler{useCase: useCase}
}

func (handler *PassengerCarClassHandler) RegisterRoutes(router gin.IRouter, passengerAuthMiddleware gin.HandlerFunc) {
	protected := router.Group("/passenger", passengerAuthMiddleware)
	protected.GET("/car-classes", handler.List)
}

// List godoc
// @Summary List active passenger car classes
// @Tags passenger-orders
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PassengerCarClassesSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/car-classes [get]
func (handler *PassengerCarClassHandler) List(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	result, err := handler.useCase.ListPassengerCarClasses(context.Request.Context(), passengerID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

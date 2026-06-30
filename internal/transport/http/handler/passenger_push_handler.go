package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerPushUseCase interface {
	RegisterToken(ctx context.Context, passengerID uuid.UUID, request dto.PassengerPushTokenRequest) (dto.PassengerPushTokenResponse, error)
}

type PassengerPushHandler struct {
	useCase PassengerPushUseCase
}

func NewPassengerPushHandler(useCase PassengerPushUseCase) *PassengerPushHandler {
	return &PassengerPushHandler{useCase: useCase}
}

func (handler *PassengerPushHandler) RegisterRoutes(router gin.IRouter, passengerAuthMiddleware gin.HandlerFunc) {
	protected := router.Group("/passenger", passengerAuthMiddleware)
	protected.POST("/push-tokens", handler.RegisterToken)
	protected.POST("/push/token", handler.RegisterToken)
}

// RegisterToken godoc
// @Summary Register passenger push token
// @Tags passenger
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer access token"
// @Param request body dto.PassengerPushTokenRequest true "Passenger push token registration payload"
// @Success 200 {object} PassengerPushTokenSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /passenger/push-tokens [post]
func (handler *PassengerPushHandler) RegisterToken(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}

	var request dto.PassengerPushTokenRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger push token request")
		return
	}

	result, err := handler.useCase.RegisterToken(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

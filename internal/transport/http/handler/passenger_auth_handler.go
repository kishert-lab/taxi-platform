package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerAuthUseCase interface {
	RequestCode(ctx context.Context, request dto.PassengerAuthRequestCodeRequest) (dto.PassengerAuthRequestCodeResponse, error)
	ConfirmCode(ctx context.Context, request dto.PassengerAuthConfirmCodeRequest) (dto.PassengerAuthTokenResponse, error)
	Refresh(ctx context.Context, request dto.RefreshTokenRequest) (dto.PassengerAuthRefreshResponse, error)
	Logout(ctx context.Context, request dto.LogoutRequest) error
}

type PassengerAuthHandler struct {
	useCase PassengerAuthUseCase
}

func NewPassengerAuthHandler(useCase PassengerAuthUseCase) *PassengerAuthHandler {
	return &PassengerAuthHandler{useCase: useCase}
}

func (handler *PassengerAuthHandler) RegisterRoutes(router gin.IRouter, passengerAuthMiddleware gin.HandlerFunc) {
	router.POST("/passenger/auth/request-code", handler.RequestCode)
	router.POST("/passenger/auth/confirm-code", handler.ConfirmCode)
	router.POST("/passenger/auth/refresh", handler.Refresh)

	protected := router.Group("/passenger/auth", passengerAuthMiddleware)
	protected.POST("/logout", handler.Logout)
}

func (handler *PassengerAuthHandler) RequestCode(context *gin.Context) {
	var request dto.PassengerAuthRequestCodeRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger auth code request")
		return
	}

	result, err := handler.useCase.RequestCode(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

func (handler *PassengerAuthHandler) ConfirmCode(context *gin.Context) {
	var request dto.PassengerAuthConfirmCodeRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger auth confirm request")
		return
	}

	result, err := handler.useCase.ConfirmCode(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

func (handler *PassengerAuthHandler) Refresh(context *gin.Context) {
	var request dto.RefreshTokenRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger refresh request")
		return
	}

	result, err := handler.useCase.Refresh(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

func (handler *PassengerAuthHandler) Logout(context *gin.Context) {
	var request dto.LogoutRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger logout request")
		return
	}

	if err := handler.useCase.Logout(context.Request.Context(), request); err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, gin.H{"logged_out": true})
}

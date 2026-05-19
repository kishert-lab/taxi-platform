package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type MobileAuthUseCase interface {
	StartLogin(ctx context.Context, request dto.AuthLoginRequest) (dto.AuthTokenResponse, error)
	SendEmailCode(ctx context.Context, request dto.AuthEmailCodeRequest) (dto.AuthCodeSentResponse, error)
	VerifyCode(ctx context.Context, request dto.AuthVerifyCodeRequest) (dto.AuthTokenResponse, error)
	Refresh(ctx context.Context, request dto.RefreshTokenRequest) (dto.AuthTokenResponse, error)
	Logout(ctx context.Context, request dto.LogoutRequest) error
}

type MobileAuthHandler struct {
	useCase MobileAuthUseCase
}

func NewMobileAuthHandler(useCase MobileAuthUseCase) *MobileAuthHandler {
	return &MobileAuthHandler{useCase: useCase}
}

func (handler *MobileAuthHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/auth/login", handler.Login)
	router.POST("/auth/email/send-code", handler.SendEmailCode)
	router.POST("/auth/email/verify", handler.VerifyEmailCode)
	router.POST("/auth/verify-code", handler.VerifyCode)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/logout", handler.Logout)
}

// Login godoc
// @Summary Login by phone and password
// @Description Authenticates by phone number and password. SMS codes are used only for registration phone confirmation.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthLoginRequest true "Login request"
// @Success 200 {object} AuthTokenSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Router /auth/login [post]
func (handler *MobileAuthHandler) Login(context *gin.Context) {
	var request dto.AuthLoginRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid login request")
		return
	}

	result, err := handler.useCase.StartLogin(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// SendEmailCode godoc
// @Summary Send email verification code
// @Description Sends a verification code for the notification email attached to an existing account.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthEmailCodeRequest true "Email code request"
// @Success 200 {object} AuthCodeSentSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 429 {object} response.Error
// @Router /auth/email/send-code [post]
func (handler *MobileAuthHandler) SendEmailCode(context *gin.Context) {
	var request dto.AuthEmailCodeRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid email code request")
		return
	}

	result, err := handler.useCase.SendEmailCode(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// VerifyEmailCode godoc
// @Summary Verify email code
// @Description Confirms the notification email and returns a fresh access/refresh token pair.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthEmailVerifyRequest true "Email verification request"
// @Success 200 {object} AuthTokenSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Router /auth/email/verify [post]
func (handler *MobileAuthHandler) VerifyEmailCode(context *gin.Context) {
	var request dto.AuthEmailVerifyRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid email verification request")
		return
	}

	result, err := handler.useCase.VerifyCode(context.Request.Context(), dto.AuthVerifyCodeRequest{
		Email: request.Email,
		Role:  request.Role,
		Code:  request.Code,
	})
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// VerifyCode godoc
// @Summary Verify login code
// @Description Verifies email confirmation code and returns access/refresh tokens. SMS code verification is used only by /auth/confirm-phone after registration.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthVerifyCodeRequest true "Verification request"
// @Success 200 {object} AuthTokenSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Router /auth/verify-code [post]
func (handler *MobileAuthHandler) VerifyCode(context *gin.Context) {
	var request dto.AuthVerifyCodeRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid verification request")
		return
	}
	if request.Email == "" {
		failValidation(context, "Email is required")
		return
	}

	result, err := handler.useCase.VerifyCode(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// Refresh godoc
// @Summary Rotate refresh token
// @Description Returns new access and refresh tokens. Old refresh token must be invalidated by service layer.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.RefreshTokenRequest true "Refresh request"
// @Success 200 {object} AuthTokenSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Router /auth/refresh [post]
func (handler *MobileAuthHandler) Refresh(context *gin.Context) {
	var request dto.RefreshTokenRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid refresh request")
		return
	}

	result, err := handler.useCase.Refresh(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// Logout godoc
// @Summary Logout mobile user
// @Description Revokes the provided refresh token.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.LogoutRequest true "Logout request"
// @Success 200 {object} AcceptedSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Router /auth/logout [post]
func (handler *MobileAuthHandler) Logout(context *gin.Context) {
	var request dto.LogoutRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid logout request")
		return
	}

	if err := handler.useCase.Logout(context.Request.Context(), request); err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, gin.H{"logged_out": true})
}

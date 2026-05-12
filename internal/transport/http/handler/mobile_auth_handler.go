package handler

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type MobileAuthUseCase interface {
	StartLogin(ctx context.Context, request dto.AuthLoginRequest) (dto.AuthCodeSentResponse, error)
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
	router.POST("/auth/verify-code", handler.VerifyCode)
	router.POST("/auth/refresh", handler.Refresh)
	router.POST("/auth/logout", handler.Logout)
}

// Login godoc
// @Summary Start phone or email login
// @Description Sends verification code to phone or email. At least one of phone or email must be provided.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.AuthLoginRequest true "Login request"
// @Success 200 {object} AuthCodeSentSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 429 {object} response.Error
// @Router /auth/login [post]
func (handler *MobileAuthHandler) Login(context *gin.Context) {
	var request dto.AuthLoginRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid login request")
		return
	}
	if request.Phone == "" && request.Email == "" {
		failValidation(context, "Phone or email is required")
		return
	}

	result, err := handler.useCase.StartLogin(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// VerifyCode godoc
// @Summary Verify login code
// @Description Verifies SMS or email code and returns access/refresh tokens.
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
	if request.Phone == "" && request.Email == "" {
		failValidation(context, "Phone or email is required")
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

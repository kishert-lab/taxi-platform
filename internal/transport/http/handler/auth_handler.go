package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/develoop/taxi-platform/internal/auth"
	"github.com/develoop/taxi-platform/internal/domain"
	"github.com/develoop/taxi-platform/internal/dto"
)

type RegistrationUseCase interface {
	StartRegistration(ctx context.Context, command auth.StartRegistrationCommand) (auth.StartRegistrationResult, error)
}

type AuthHandler struct {
	registrationUseCase RegistrationUseCase
}

func NewAuthHandler(registrationUseCase RegistrationUseCase) *AuthHandler {
	return &AuthHandler{registrationUseCase: registrationUseCase}
}

func (handler *AuthHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/auth/register", handler.Register)
}

// Register godoc
// @Summary Register user
// @Description Registers passenger, driver, or taxi park owner. Personal data consent and terms acceptance are mandatory.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.StartRegistrationRequest true "Registration request"
// @Success 201 {object} dto.StartRegistrationResponse
// @Failure 400 {object} errorResponse
// @Failure 500 {object} errorResponse
// @Router /auth/register [post]
func (handler *AuthHandler) Register(context *gin.Context) {
	var request dto.StartRegistrationRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		context.JSON(http.StatusBadRequest, errorResponse{Error: "invalid registration request"})
		return
	}

	if err := domain.ValidateRequiredRegistrationConsent(
		request.PersonalDataConsent,
		request.TermsAccepted,
		request.PrivacyPolicyVersion,
		request.TermsVersion,
	); err != nil {
		context.JSON(http.StatusBadRequest, errorResponse{Error: domain.ErrConsentRequired.Error()})
		return
	}

	command := auth.StartRegistrationCommand{
		Phone:                request.Phone,
		Email:                request.Email,
		Password:             request.Password,
		FirstName:            request.FirstName,
		LastName:             request.LastName,
		RegistrationType:     request.RegistrationType,
		CityID:               request.CityID,
		PersonalDataConsent:  request.PersonalDataConsent,
		TermsAccepted:        request.TermsAccepted,
		PrivacyPolicyVersion: request.PrivacyPolicyVersion,
		TermsVersion:         request.TermsVersion,
		ConsentIP:            context.ClientIP(),
		ConsentUserAgent:     context.Request.UserAgent(),
	}
	if request.TaxiPark != nil {
		command.TaxiParkName = request.TaxiPark.Name
		command.TaxiParkLegalName = request.TaxiPark.LegalName
		command.TaxiParkTaxID = request.TaxiPark.TaxID
	}

	result, err := handler.registrationUseCase.StartRegistration(context.Request.Context(), command)
	if err != nil {
		if errors.Is(err, domain.ErrConsentRequired) {
			context.JSON(http.StatusBadRequest, errorResponse{Error: domain.ErrConsentRequired.Error()})
			return
		}
		context.JSON(http.StatusInternalServerError, errorResponse{Error: "registration failed"})
		return
	}

	context.JSON(http.StatusCreated, dto.StartRegistrationResponse{
		UserID:           result.UserID,
		Role:             result.Role,
		RegistrationType: result.RegistrationType,
		PhoneMasked:      result.PhoneMasked,
		EmailMasked:      result.EmailMasked,
		Message:          "confirmation codes sent",
	})
}

type errorResponse struct {
	Error string `json:"error" example:"personal data consent required"`
}

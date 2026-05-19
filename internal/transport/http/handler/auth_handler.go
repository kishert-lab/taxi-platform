package handler

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type RegistrationUseCase interface {
	StartRegistration(ctx context.Context, command auth.StartRegistrationCommand) (auth.StartRegistrationResult, error)
	ConfirmPhone(ctx context.Context, command auth.ConfirmPhoneCommand) error
}

type AuthHandler struct {
	registrationUseCase RegistrationUseCase
}

func NewAuthHandler(registrationUseCase RegistrationUseCase) *AuthHandler {
	return &AuthHandler{registrationUseCase: registrationUseCase}
}

func (handler *AuthHandler) RegisterRoutes(router gin.IRouter) {
	router.POST("/auth/register", handler.Register)
	router.POST("/auth/confirm-phone", handler.ConfirmPhone)
}

// Register godoc
// @Summary Register user
// @Description Registers passenger, driver, or taxi park owner. Personal data consent and terms acceptance are mandatory.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.StartRegistrationRequest true "Registration request"
// @Success 201 {object} response.Success
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /auth/register [post]
func (handler *AuthHandler) Register(context *gin.Context) {
	var request dto.StartRegistrationRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid registration request")
		return
	}

	if err := domain.ValidateRequiredRegistrationConsent(
		request.PersonalDataConsent,
		request.TermsAccepted,
		request.PrivacyPolicyVersion,
		request.TermsVersion,
	); err != nil {
		response.Fail(context, 400, response.CodeConsentRequired, domain.ErrConsentRequired.Error(), nil)
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
			response.Fail(context, 400, response.CodeConsentRequired, domain.ErrConsentRequired.Error(), nil)
			return
		}
		response.Fail(context, 500, response.CodeInternalError, "Registration failed", nil)
		return
	}

	response.Created(context, dto.StartRegistrationResponse{
		UserID:           result.UserID,
		Role:             result.Role,
		RegistrationType: result.RegistrationType,
		PhoneMasked:      result.PhoneMasked,
		EmailMasked:      result.EmailMasked,
		Message:          "phone confirmation code sent",
		DebugCode:        result.DebugCode,
	})
}

// ConfirmPhone godoc
// @Summary Confirm phone after registration
// @Description Confirms the SMS code sent during registration.
// @Tags auth
// @Accept json
// @Produce json
// @Param request body dto.ConfirmPhoneRequest true "Phone confirmation request"
// @Success 200 {object} response.Success
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Router /auth/confirm-phone [post]
func (handler *AuthHandler) ConfirmPhone(context *gin.Context) {
	var request dto.ConfirmPhoneRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid phone confirmation request")
		return
	}

	err := handler.registrationUseCase.ConfirmPhone(context.Request.Context(), auth.ConfirmPhoneCommand{
		Phone:            request.Phone,
		RegistrationType: request.Type,
		Code:             request.Code,
	})
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, gin.H{"phone_confirmed": true})
}

type errorResponse struct {
	Error string `json:"error" example:"personal data consent required"`
}

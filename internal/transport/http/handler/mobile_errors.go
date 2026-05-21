package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/common"
	"github.com/kishert-lab/taxi-platform/internal/dispatch"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/finance"
	orderapp "github.com/kishert-lab/taxi-platform/internal/order"
	taxiparkapp "github.com/kishert-lab/taxi-platform/internal/taxipark"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

var (
	ErrMobileOrderNotFound     = errors.New("order not found")
	ErrMobileDriverUnavailable = errors.New("driver not available")
	ErrMobileDispatchActive    = errors.New("dispatch in progress")
)

func failValidation(context *gin.Context, message string) {
	response.Fail(context, http.StatusBadRequest, response.CodeValidationError, message, nil)
}

func failUnauthorized(context *gin.Context, message string) {
	response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, message, nil)
}

func failForbidden(context *gin.Context, message string) {
	response.Fail(context, http.StatusForbidden, response.CodeForbidden, message, nil)
}

func failByError(context *gin.Context, err error) {
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrInvalidCode), errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrInactiveUser):
		response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Unauthorized", nil)
	case errors.Is(err, auth.ErrDriverAccessDenied):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Driver access is blocked", nil)
	case errors.Is(err, domain.ErrInvalidPhone), errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrInvalidUserRole), errors.Is(err, domain.ErrInvalidVerificationStatus):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid auth request", nil)
	case errors.Is(err, ErrMobileOrderNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeOrderNotFound, "Order not found", nil)
	case errors.Is(err, domain.ErrInvalidOrderStatusTransition):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Order invalid state", nil)
	case errors.Is(err, dispatch.ErrOrderAlreadyAssigned):
		response.Fail(context, http.StatusConflict, response.CodeOrderAlreadyAssigned, "Order already assigned", nil)
	case errors.Is(err, ErrMobileDriverUnavailable):
		response.Fail(context, http.StatusConflict, response.CodeDriverNotAvailable, "Driver is not available", nil)
	case errors.Is(err, ErrMobileDispatchActive):
		response.Fail(context, http.StatusConflict, response.CodeDispatchInProgress, "Dispatch is already in progress", nil)
	case errors.Is(err, orderapp.ErrOrderConcurrentUpdate):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Order was changed concurrently", nil)
	case errors.Is(err, finance.ErrFinancialSettlementDuplicate):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Financial settlement already exists", nil)
	case errors.Is(err, taxiparkapp.ErrTaxiParkNotFound):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Taxi park account is not available", nil)
	case errors.Is(err, taxiparkapp.ErrInvalidDriverCreateFields):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid taxi park request", nil)
	case errors.Is(err, taxiparkapp.ErrDriverPhoneAlreadyExists):
		response.Fail(context, http.StatusConflict, response.CodeValidationError, "Driver with this phone already exists", nil)
	case errors.Is(err, common.ErrNotImplemented):
		response.Fail(context, http.StatusNotImplemented, response.CodeNotImplemented, "Endpoint is registered but service is not implemented", nil)
	default:
		response.Fail(context, http.StatusInternalServerError, response.CodeInternalError, "Internal error", nil)
	}
}

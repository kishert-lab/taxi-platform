package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/common"
	"github.com/kishert-lab/taxi-platform/internal/dispatch"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	driverapp "github.com/kishert-lab/taxi-platform/internal/driver"
	"github.com/kishert-lab/taxi-platform/internal/finance"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geo"
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

func failValidationWithDetails(context *gin.Context, message string, details map[string]any) {
	response.Fail(context, http.StatusBadRequest, response.CodeValidationError, message, details)
}

func failUnauthorized(context *gin.Context, message string) {
	response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, message, nil)
}

func failForbidden(context *gin.Context, message string) {
	response.Fail(context, http.StatusForbidden, response.CodeForbidden, message, nil)
}

func failByError(context *gin.Context, err error) {
	if err != nil {
		_ = context.Error(err)
	}
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrInvalidCode), errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrInactiveUser):
		response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Unauthorized", nil)
	case errors.Is(err, auth.ErrDriverAccessDenied):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Driver access is blocked", nil)
	case errors.Is(err, domain.ErrInvalidPhone), errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrInvalidUserRole), errors.Is(err, domain.ErrInvalidVerificationStatus), errors.Is(err, domain.ErrInvalidPaymentMethod):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid request", nil)
	case errors.Is(err, ErrMobileOrderNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeOrderNotFound, "Order not found", nil)
	case errors.Is(err, driverapp.ErrCurrentOrderNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeOrderNotFound, "Order not found", nil)
	case errors.Is(err, domain.ErrInvalidOrderStatusTransition):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Order invalid state", nil)
	case errors.Is(err, dispatch.ErrOrderAlreadyAssigned):
		response.Fail(context, http.StatusConflict, response.CodeOrderAlreadyAssigned, "Order already assigned", nil)
	case errors.Is(err, ErrMobileDriverUnavailable), errors.Is(err, driverapp.ErrDriverNotAvailable):
		response.Fail(context, http.StatusConflict, response.CodeDriverNotAvailable, "Driver is not available", nil)
	case errors.Is(err, driverapp.ErrDriverNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeNotFound, "Driver not found", nil)
	case errors.Is(err, geoservice.ErrLocationUpdateThrottled):
		response.Fail(context, http.StatusTooManyRequests, response.CodeRateLimited, "Driver location update is rate limited", nil)
	case errors.Is(err, ErrMobileDispatchActive):
		response.Fail(context, http.StatusConflict, response.CodeDispatchInProgress, "Dispatch is already in progress", nil)
	case errors.Is(err, orderapp.ErrOrderConcurrentUpdate):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Order was changed concurrently", nil)
	case errors.Is(err, finance.ErrFinancialSettlementDuplicate):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Financial settlement already exists", nil)
	case errors.Is(err, taxiparkapp.ErrTaxiParkNotFound):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Taxi park account is not available", nil)
	case errors.Is(err, taxiparkapp.ErrTaxiParkResourceNotFound), errors.Is(err, pgx.ErrNoRows):
		response.Fail(context, http.StatusNotFound, response.CodeNotFound, "Taxi park resource not found", nil)
	case errors.Is(err, taxiparkapp.ErrTaxiParkResourceForbidden):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Taxi park resource is forbidden", nil)
	case errors.Is(err, taxiparkapp.ErrInvalidDriverCreateFields):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid taxi park request", nil)
	case errors.Is(err, taxiparkapp.ErrInvalidOrderFields):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid taxi park order request", map[string]any{
			"reason": err.Error(),
		})
	case errors.Is(err, taxiparkapp.ErrOrderTariffNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeNotFound, "Taxi park order tariff not found", map[string]any{
			"field":  "tariff_id",
			"reason": "Tariff must be active and belong to the current taxi park or taxi park city",
		})
	case errors.Is(err, taxiparkapp.ErrInvalidDriverPassword):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Driver password must contain at least 8 characters", nil)
	case errors.Is(err, taxiparkapp.ErrDriverPhoneAlreadyExists):
		response.Fail(context, http.StatusConflict, response.CodeValidationError, "Driver with this phone already exists", nil)
	case errors.Is(err, taxiparkapp.ErrCarAlreadyExists):
		response.Fail(context, http.StatusConflict, response.CodeValidationError, "Car with this plate or VIN already exists", nil)
	case errors.Is(err, common.ErrNotImplemented):
		response.Fail(context, http.StatusNotImplemented, response.CodeNotImplemented, "Endpoint is registered but service is not implemented", nil)
	default:
		response.Fail(context, http.StatusInternalServerError, response.CodeInternalError, "Internal error", nil)
	}
}

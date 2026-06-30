package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/kishert-lab/taxi-platform/internal/auth"
	chatapp "github.com/kishert-lab/taxi-platform/internal/chat"
	"github.com/kishert-lab/taxi-platform/internal/common"
	"github.com/kishert-lab/taxi-platform/internal/dispatch"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	driverapp "github.com/kishert-lab/taxi-platform/internal/driver"
	"github.com/kishert-lab/taxi-platform/internal/finance"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geo"
	orderapp "github.com/kishert-lab/taxi-platform/internal/order"
	passengerapp "github.com/kishert-lab/taxi-platform/internal/passenger"
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

func failByError(context *gin.Context, err error) {
	if err != nil {
		_ = context.Error(err)
	}
	switch {
	case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrInvalidCode), errors.Is(err, auth.ErrInvalidToken), errors.Is(err, auth.ErrInactiveUser):
		response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Unauthorized", nil)
	case errors.Is(err, passengerapp.ErrInvalidToken), errors.Is(err, passengerapp.ErrInvalidRefreshToken):
		response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Unauthorized", nil)
	case errors.Is(err, passengerapp.ErrPassengerBlocked):
		response.Fail(context, http.StatusForbidden, response.CodePassengerBlocked, "Passenger is blocked", nil)
	case errors.Is(err, passengerapp.ErrCodeExpired):
		response.Fail(context, http.StatusUnauthorized, response.CodeCodeExpired, "Code expired", nil)
	case errors.Is(err, passengerapp.ErrCodeAlreadyUsed), errors.Is(err, passengerapp.ErrInvalidCode):
		response.Fail(context, http.StatusUnauthorized, response.CodeInvalidCode, "Invalid confirmation code", nil)
	case errors.Is(err, passengerapp.ErrTooManyAttempts):
		response.Fail(context, http.StatusTooManyRequests, response.CodeTooManyAttempts, "Too many confirmation attempts", nil)
	case errors.Is(err, auth.ErrDriverAccessDenied):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Driver access is blocked", nil)
	case errors.Is(err, domain.ErrInvalidPhone):
		response.Fail(context, http.StatusBadRequest, response.CodeInvalidPhone, "Invalid phone", nil)
	case errors.Is(err, domain.ErrInvalidEmail), errors.Is(err, domain.ErrInvalidUserRole), errors.Is(err, domain.ErrInvalidVerificationStatus), errors.Is(err, domain.ErrInvalidPaymentMethod), errors.Is(err, domain.ErrInvalidChatType), errors.Is(err, domain.ErrInvalidChatMessage), errors.Is(err, domain.ErrInvalidPushToken), errors.Is(err, domain.ErrInvalidPushPlatform):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid request", nil)
	case errors.Is(err, ErrMobileOrderNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeOrderNotFound, "Order not found", nil)
	case errors.Is(err, driverapp.ErrCurrentOrderNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeOrderNotFound, "Order not found", nil)
	case errors.Is(err, driverapp.ErrOrderAccessDenied):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Order is forbidden", nil)
	case errors.Is(err, driverapp.ErrOrderRouteForbidden):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Order route upload is forbidden for current order state", nil)
	case errors.Is(err, domain.ErrInvalidOrderStatusTransition):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Order invalid state", nil)
	case errors.Is(err, dispatch.ErrOrderAlreadyAssigned):
		response.Fail(context, http.StatusConflict, response.CodeOrderAlreadyAssigned, "Order already assigned", nil)
	case errors.Is(err, dispatch.ErrOfferNotAccepted):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Order offer is not active", nil)
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
	case errors.Is(err, chatapp.ErrChatForbidden):
		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Chat is forbidden", nil)
	case errors.Is(err, chatapp.ErrChatUnavailable):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Chat is not available for this order state", nil)
	case errors.Is(err, finance.ErrFinancialSettlementDuplicate):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Financial settlement already exists", nil)
	case errors.Is(err, finance.ErrDriverFinanceAccountNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeNotFound, "Driver finance account not found", nil)
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
	case errors.Is(err, taxiparkapp.ErrInvalidScheduledOrder):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid scheduled order request", map[string]any{
			"reason": err.Error(),
		})
	case errors.Is(err, taxiparkapp.ErrScheduledOrdersDisabled):
		response.Fail(context, http.StatusConflict, response.CodeOrderInvalidState, "Scheduled orders are disabled for taxi park", nil)
	case errors.Is(err, taxiparkapp.ErrOrderTariffNotFound):
		response.Fail(context, http.StatusNotFound, response.CodeNotFound, "Taxi park order tariff not found", map[string]any{
			"field":  "tariff_id",
			"reason": "Tariff must be active and belong to the current taxi park or taxi park city",
		})
	case errors.Is(err, taxiparkapp.ErrInvalidDispatcherPassword):
		response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Dispatcher password must contain at least 8 characters", nil)
	case errors.Is(err, taxiparkapp.ErrDispatcherAlreadyExists):
		response.Fail(context, http.StatusConflict, response.CodeValidationError, "Dispatcher with this phone or email already exists", nil)
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

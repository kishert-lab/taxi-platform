package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorCode string

const (
	CodeValidationError      ErrorCode = "VALIDATION_ERROR"
	CodeUnauthorized         ErrorCode = "UNAUTHORIZED"
	CodeForbidden            ErrorCode = "FORBIDDEN"
	CodeOrderNotFound        ErrorCode = "ORDER_NOT_FOUND"
	CodeOrderInvalidState    ErrorCode = "ORDER_INVALID_STATE"
	CodeDriverNotAvailable   ErrorCode = "DRIVER_NOT_AVAILABLE"
	CodeOrderAlreadyAssigned ErrorCode = "ORDER_ALREADY_ASSIGNED"
	CodeDispatchInProgress   ErrorCode = "DISPATCH_IN_PROGRESS"
	CodeRateLimited          ErrorCode = "RATE_LIMITED"
	CodeConsentRequired      ErrorCode = "CONSENT_REQUIRED"
	CodeNotImplemented       ErrorCode = "NOT_IMPLEMENTED"
	CodeInternalError        ErrorCode = "INTERNAL_ERROR"
)

const RequestIDContextKey = "request_id"

type Meta struct {
	RequestID string `json:"request_id" example:"11111111-1111-1111-1111-111111111111"`
}

type Success struct {
	Data any  `json:"data"`
	Meta Meta `json:"meta"`
}

type ErrorBody struct {
	Code    ErrorCode      `json:"code" example:"ORDER_NOT_FOUND"`
	Message string         `json:"message" example:"Order not found"`
	Details map[string]any `json:"details,omitempty"`
}

type Error struct {
	Error ErrorBody `json:"error"`
	Meta  Meta      `json:"meta"`
}

func OK(context *gin.Context, data any) {
	context.JSON(http.StatusOK, Success{Data: data, Meta: meta(context)})
}

func Created(context *gin.Context, data any) {
	context.JSON(http.StatusCreated, Success{Data: data, Meta: meta(context)})
}

func Fail(context *gin.Context, status int, code ErrorCode, message string, details map[string]any) {
	context.JSON(status, Error{
		Error: ErrorBody{Code: code, Message: message, Details: details},
		Meta:  meta(context),
	})
}

func meta(context *gin.Context) Meta {
	requestID, _ := context.Get(RequestIDContextKey)
	if value, ok := requestID.(string); ok {
		return Meta{RequestID: value}
	}
	return Meta{}
}

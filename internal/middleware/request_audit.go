package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	auditapp "github.com/kishert-lab/taxi-platform/internal/audit"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type HTTPRequestAuditLogger interface {
	LogHTTP(ctx context.Context, command auditapp.HTTPRequestLogCommand)
}

func RequestAudit(logger HTTPRequestAuditLogger) gin.HandlerFunc {
	return func(context *gin.Context) {
		if logger == nil {
			context.Next()
			return
		}

		startedAt := time.Now()
		requestBody := readAuditRequestBody(context.Request)

		context.Next()

		logger.LogHTTP(context.Request.Context(), auditapp.HTTPRequestLogCommand{
			RequestID:    context.GetString(response.RequestIDContextKey),
			Method:       context.Request.Method,
			Route:        context.FullPath(),
			Path:         context.Request.URL.Path,
			RawQuery:     context.Request.URL.RawQuery,
			StatusCode:   context.Writer.Status(),
			Duration:     time.Since(startedAt),
			ClientIP:     context.ClientIP(),
			UserAgent:    context.Request.UserAgent(),
			ActorUserID:  auditUserIDFromContext(context),
			ActorRole:    auditUserRoleFromContext(context),
			ErrorMessage: context.Errors.String(),
			ContentType:  context.ContentType(),
			RequestBody:  requestBody,
		})
	}
}

func readAuditRequestBody(request *http.Request) string {
	if request == nil || request.Body == nil || !shouldLogRequestBody(request.Header.Get("Content-Type")) {
		return ""
	}
	body, err := io.ReadAll(io.LimitReader(request.Body, 4096))
	if err != nil {
		return ""
	}

	remainingBody, err := io.ReadAll(request.Body)
	if err != nil {
		request.Body = io.NopCloser(bytes.NewReader(body))
		return string(body)
	}

	request.Body = io.NopCloser(bytes.NewReader(append(body, remainingBody...)))
	return string(body)
}

func auditUserIDFromContext(context *gin.Context) uuid.UUID {
	value, exists := context.Get(UserIDContextKey)
	if !exists {
		return uuid.Nil
	}

	switch typedValue := value.(type) {
	case uuid.UUID:
		return typedValue
	case string:
		parsedID, err := uuid.Parse(typedValue)
		if err == nil {
			return parsedID
		}
	}

	return uuid.Nil
}

func auditUserRoleFromContext(context *gin.Context) domain.UserRole {
	value, exists := context.Get(UserRoleContextKey)
	if !exists {
		return ""
	}

	switch typedValue := value.(type) {
	case domain.UserRole:
		return typedValue
	case string:
		return domain.UserRole(typedValue)
	default:
		return ""
	}
}

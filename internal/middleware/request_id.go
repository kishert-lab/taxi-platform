package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/pkg/response"
)

func RequestID() gin.HandlerFunc {
	return func(context *gin.Context) {
		requestID := context.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		context.Set(response.RequestIDContextKey, requestID)
		context.Header("X-Request-ID", requestID)
		context.Next()
	}
}

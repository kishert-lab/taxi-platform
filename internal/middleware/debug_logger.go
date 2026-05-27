package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type debugResponseWriter struct {
	gin.ResponseWriter
	body bytes.Buffer
}

func (writer *debugResponseWriter) Write(data []byte) (int, error) {
	writer.body.Write(data)
	return writer.ResponseWriter.Write(data)
}

func (writer *debugResponseWriter) WriteString(data string) (int, error) {
	writer.body.WriteString(data)
	return writer.ResponseWriter.WriteString(data)
}

func DebugRequestLogger(logger *zap.Logger, enabled bool) gin.HandlerFunc {
	if logger == nil {
		logger = zap.NewNop()
	}
	return func(context *gin.Context) {
		if !enabled {
			context.Next()
			return
		}

		startedAt := time.Now()
		requestBody := readDebugRequestBody(context.Request)
		responseWriter := &debugResponseWriter{ResponseWriter: context.Writer}
		context.Writer = responseWriter

		logger.Debug("http request started",
			zap.String("request_id", context.GetString(response.RequestIDContextKey)),
			zap.String("method", context.Request.Method),
			zap.String("path", context.Request.URL.Path),
			zap.String("raw_query", context.Request.URL.RawQuery),
			zap.String("client_ip", context.ClientIP()),
			zap.String("user_agent", context.Request.UserAgent()),
			zap.String("content_type", context.ContentType()),
			zap.String("request_body", requestBody),
		)

		context.Next()

		fields := []zap.Field{
			zap.String("request_id", context.GetString(response.RequestIDContextKey)),
			zap.String("method", context.Request.Method),
			zap.String("path", context.FullPath()),
			zap.String("raw_path", context.Request.URL.Path),
			zap.Int("status", context.Writer.Status()),
			zap.Int("bytes", context.Writer.Size()),
			zap.Duration("duration", time.Since(startedAt)),
			zap.String("client_ip", context.ClientIP()),
			zap.String("user_id", debugContextValue(context, UserIDContextKey)),
			zap.String("user_role", debugContextValue(context, UserRoleContextKey)),
			zap.String("response_body", responseWriter.body.String()),
		}
		if len(context.Errors) > 0 {
			fields = append(fields, zap.String("errors", context.Errors.String()))
			fields = append(fields, zap.Error(context.Errors.Last().Err))
		}

		logger.Debug("http request completed", fields...)
	}
}

func readDebugRequestBody(request *http.Request) string {
	if request == nil || request.Body == nil || !shouldLogRequestBody(request.Header.Get("Content-Type")) {
		return ""
	}
	body, err := io.ReadAll(request.Body)
	if err != nil {
		return "<read error: " + err.Error() + ">"
	}
	request.Body = io.NopCloser(bytes.NewReader(body))
	return string(body)
}

func shouldLogRequestBody(contentType string) bool {
	contentType = strings.ToLower(contentType)
	if strings.Contains(contentType, "multipart/form-data") || strings.Contains(contentType, "application/octet-stream") {
		return false
	}
	return true
}

func debugContextValue(context *gin.Context, key string) string {
	value, exists := context.Get(key)
	if !exists || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

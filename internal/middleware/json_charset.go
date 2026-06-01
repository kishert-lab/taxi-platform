package middleware

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/encoding/charmap"

	"github.com/kishert-lab/taxi-platform/pkg/response"
)

// JSONCharset validates JSON request bodies as UTF-8 and converts explicitly
// declared Windows-1251 JSON bodies to UTF-8 before handlers bind DTOs.
func JSONCharset() gin.HandlerFunc {
	return func(context *gin.Context) {
		if context.Request.Body == nil || !mayHaveRequestBody(context.Request.Method) {
			context.Next()
			return
		}

		mediaType, params, err := mime.ParseMediaType(context.GetHeader("Content-Type"))
		if err != nil || !strings.EqualFold(mediaType, "application/json") {
			context.Next()
			return
		}

		body, err := io.ReadAll(context.Request.Body)
		if err != nil {
			_ = context.Error(fmt.Errorf("read json request body: %w", err))
			response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid request body", nil)
			context.Abort()
			return
		}
		_ = context.Request.Body.Close()

		charset := strings.ToLower(strings.TrimSpace(params["charset"]))
		switch charset {
		case "", "utf-8", "utf8":
			if !utf8.Valid(body) {
				response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "JSON body must be UTF-8 encoded", map[string]any{
					"charset": charset,
				})
				context.Abort()
				return
			}
		case "windows-1251", "cp1251":
			decoded, err := charmap.Windows1251.NewDecoder().Bytes(body)
			if err != nil {
				_ = context.Error(fmt.Errorf("decode windows-1251 json request body: %w", err))
				response.Fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid request body encoding", map[string]any{
					"charset": charset,
				})
				context.Abort()
				return
			}
			body = decoded
			context.Request.Header.Set("Content-Type", "application/json; charset=utf-8")
		default:
			response.Fail(context, http.StatusUnsupportedMediaType, response.CodeValidationError, "Unsupported JSON charset", map[string]any{
				"charset": charset,
			})
			context.Abort()
			return
		}

		context.Request.Body = io.NopCloser(bytes.NewReader(body))
		context.Request.ContentLength = int64(len(body))
		context.Next()
	}
}

func mayHaveRequestBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

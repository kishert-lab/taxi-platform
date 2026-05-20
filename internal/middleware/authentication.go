package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authapp "github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

const UserIDContextKey = "user_id"

type AccessTokenParser interface {
	ParseAccessToken(token string) (authapp.TokenClaims, error)
}

func AuthenticateAccessToken(parser AccessTokenParser, publicPathPrefixes ...string) gin.HandlerFunc {
	return func(context *gin.Context) {
		if isPublicPath(context.Request.URL.Path, publicPathPrefixes) {
			context.Next()
			return
		}

		token := bearerToken(context.GetHeader("Authorization"))
		if token == "" {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Authorization token is missing", nil)
			context.Abort()
			return
		}

		claims, err := parser.ParseAccessToken(token)
		if err != nil {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Authorization token is invalid", nil)
			context.Abort()
			return
		}
		if claims.Subject == uuid.Nil || claims.Role.Validate() != nil {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "Authorization token claims are invalid", nil)
			context.Abort()
			return
		}

		context.Set(UserIDContextKey, claims.Subject)
		context.Set(UserRoleContextKey, domain.UserRole(claims.Role))
		context.Next()
	}
}

func bearerToken(header string) string {
	fields := strings.Fields(header)
	if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(fields[1])
}

func isPublicPath(path string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}

	return false
}

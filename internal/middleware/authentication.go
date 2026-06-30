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

type publicPathMatcher interface {
	Matches(path string) bool
}

type exactPublicPath string

func ExactPublicPath(path string) publicPathMatcher {
	return exactPublicPath(path)
}

func (path exactPublicPath) Matches(candidate string) bool {
	return candidate == string(path)
}

type prefixPublicPath string

func PrefixPublicPath(path string) publicPathMatcher {
	return prefixPublicPath(path)
}

func (path prefixPublicPath) Matches(candidate string) bool {
	return strings.HasPrefix(candidate, string(path))
}

func AuthenticateAccessToken(parser AccessTokenParser, publicPathPrefixes ...string) gin.HandlerFunc {
	matchers := make([]publicPathMatcher, 0, len(publicPathPrefixes))
	for _, prefix := range publicPathPrefixes {
		matchers = append(matchers, PrefixPublicPath(prefix))
	}

	return AuthenticateAccessTokenWithMatchers(parser, matchers...)
}

func AuthenticateAccessTokenWithMatchers(parser AccessTokenParser, matchers ...publicPathMatcher) gin.HandlerFunc {
	return func(context *gin.Context) {
		if isPublicPath(context.Request.URL.Path, matchers) {
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

func isPublicPath(path string, matchers []publicPathMatcher) bool {
	for _, matcher := range matchers {
		if matcher.Matches(path) {
			return true
		}
	}

	return false
}

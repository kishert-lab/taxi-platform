package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

const UserRoleContextKey = "user_role"

func RequireAuthenticated() gin.HandlerFunc {
	return func(context *gin.Context) {
		if _, ok := roleFromContext(context); !ok {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "User role is missing", nil)
			context.Abort()
			return
		}
		if _, exists := context.Get(UserIDContextKey); !exists {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "User id is missing", nil)
			context.Abort()
			return
		}

		context.Next()
	}
}

func RequireRole(allowedRoles ...domain.UserRole) gin.HandlerFunc {
	return func(context *gin.Context) {
		role, ok := roleFromContext(context)
		if !ok {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "User role is missing", nil)
			context.Abort()
			return
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				context.Next()
				return
			}
		}

		response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Role is not allowed", nil)
		context.Abort()
	}
}

func RequirePermission(requiredPermission domain.Permission) gin.HandlerFunc {
	return func(context *gin.Context) {
		role, ok := roleFromContext(context)
		if !ok {
			response.Fail(context, http.StatusUnauthorized, response.CodeUnauthorized, "User role is missing", nil)
			context.Abort()
			return
		}

		if !domain.RoleHasPermission(role, requiredPermission) {
			response.Fail(context, http.StatusForbidden, response.CodeForbidden, "Permission is not allowed", nil)
			context.Abort()
			return
		}

		context.Next()
	}
}

func roleFromContext(context *gin.Context) (domain.UserRole, bool) {
	value, exists := context.Get(UserRoleContextKey)
	if !exists {
		return "", false
	}

	switch role := value.(type) {
	case domain.UserRole:
		return role, role.Validate() == nil
	case string:
		userRole := domain.UserRole(role)
		return userRole, userRole.Validate() == nil
	default:
		return "", false
	}
}

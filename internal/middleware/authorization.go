package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

const UserRoleContextKey = "user_role"

func RequireRole(allowedRoles ...domain.UserRole) gin.HandlerFunc {
	return func(context *gin.Context) {
		role, ok := roleFromContext(context)
		if !ok {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user role is missing"})
			return
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				context.Next()
				return
			}
		}

		context.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "role is not allowed"})
	}
}

func RequirePermission(requiredPermission domain.Permission) gin.HandlerFunc {
	return func(context *gin.Context) {
		role, ok := roleFromContext(context)
		if !ok {
			context.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user role is missing"})
			return
		}

		if !domain.RoleHasPermission(role, requiredPermission) {
			context.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "permission is not allowed"})
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

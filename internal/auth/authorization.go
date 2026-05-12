package auth

import (
	"fmt"

	"github.com/develoop/taxi-platform/internal/domain"
)

type Authorizer struct{}

func NewAuthorizer() *Authorizer {
	return &Authorizer{}
}

func (authorizer *Authorizer) RequireRole(actualRole domain.UserRole, allowedRoles ...domain.UserRole) error {
	if err := actualRole.Validate(); err != nil {
		return fmt.Errorf("validate actual role: %w", err)
	}

	for _, allowedRole := range allowedRoles {
		if actualRole == allowedRole {
			return nil
		}
	}

	return fmt.Errorf("role %q is not allowed", actualRole)
}

func (authorizer *Authorizer) RequirePermission(role domain.UserRole, requiredPermission domain.Permission) error {
	if err := role.Validate(); err != nil {
		return fmt.Errorf("validate role for permission check: %w", err)
	}

	if !domain.RoleHasPermission(role, requiredPermission) {
		return fmt.Errorf("role %q does not have permission %q", role, requiredPermission)
	}

	return nil
}

package dto

import "github.com/kishert-lab/taxi-platform/internal/domain"

type RoleResponse struct {
	Role        domain.UserRole     `json:"role" example:"passenger"`
	Permissions []domain.Permission `json:"permissions"`
}

type RolesResponse struct {
	Roles []RoleResponse `json:"roles"`
}

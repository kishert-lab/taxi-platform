package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TaxiParkCreateDispatcherRequest struct {
	Phone     string `json:"phone" binding:"required" example:"+79990000000"`
	Email     string `json:"email,omitempty" binding:"omitempty,email" example:"dispatcher@example.com"`
	Password  string `json:"password" binding:"required" example:"strong-password"`
	FirstName string `json:"first_name,omitempty" example:"Ivan"`
	LastName  string `json:"last_name,omitempty" example:"Petrov"`
}

type TaxiParkUpdateDispatcherRequest struct {
	Email     *string `json:"email,omitempty" binding:"omitempty,email" example:"dispatcher@example.com"`
	FirstName *string `json:"first_name,omitempty" example:"Ivan"`
	LastName  *string `json:"last_name,omitempty" example:"Petrov"`
}

type TaxiParkDispatcherResponse struct {
	DispatcherID uuid.UUID       `json:"dispatcher_id" example:"11111111-1111-1111-1111-111111111111"`
	UserID       uuid.UUID       `json:"user_id" example:"22222222-2222-2222-2222-222222222222"`
	TaxiParkID   uuid.UUID       `json:"taxi_park_id" example:"33333333-3333-3333-3333-333333333333"`
	Phone        string          `json:"phone" example:"+79990000000"`
	Email        string          `json:"email,omitempty" example:"dispatcher@example.com"`
	FirstName    string          `json:"first_name,omitempty" example:"Ivan"`
	LastName     string          `json:"last_name,omitempty" example:"Petrov"`
	Name         string          `json:"name,omitempty" example:"Ivan Petrov"`
	Role         domain.UserRole `json:"role" example:"dispatcher"`
	IsActive     bool            `json:"is_active" example:"true"`
	CreatedAt    time.Time       `json:"created_at" example:"2026-06-27T12:00:00Z"`
	UpdatedAt    time.Time       `json:"updated_at" example:"2026-06-27T12:00:00Z"`
}

type TaxiParkDispatchersResponse struct {
	Dispatchers []TaxiParkDispatcherResponse `json:"dispatchers"`
}

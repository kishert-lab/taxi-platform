// Package admin contains application services for privileged console operations.
package admin

import (
	"context"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type Repository interface {
	CreateTaxiParkOwner(ctx context.Context, record CreateTaxiParkOwnerRecord) (CreateTaxiParkOwnerResult, error)
	ResetPasswordByPhone(ctx context.Context, record ResetPasswordRecord) (ResetPasswordResult, error)
}

type CreateTaxiParkCommand struct {
	Phone                string
	Email                string
	Password             string
	FirstName            string
	LastName             string
	CityID               uuid.UUID
	Name                 string
	LegalName            string
	TaxID                string
	CommissionPercent    string
	Verified             bool
	AcceptDocuments      bool
	PrivacyPolicyVersion string
	TermsVersion         string
	ConsentIP            string
	ConsentUserAgent     string
}

type CreateTaxiParkOwnerRecord struct {
	Phone                string
	Email                string
	PasswordHash         string
	FirstName            string
	LastName             string
	CityID               uuid.UUID
	Name                 string
	LegalName            string
	TaxID                string
	CommissionPercent    *string
	Verified             bool
	PrivacyPolicyVersion string
	TermsVersion         string
	ConsentIP            string
	ConsentUserAgent     string
}

type CreateTaxiParkOwnerResult struct {
	UserID     uuid.UUID `json:"user_id"`
	TaxiParkID uuid.UUID `json:"taxi_park_id"`
	Phone      string    `json:"phone"`
	Email      string    `json:"email"`
}

type CreateTaxiParkResult struct {
	CreateTaxiParkOwnerResult
	GeneratedPassword string `json:"generated_password,omitempty"`
	PasswordGenerated bool   `json:"password_generated"`
}

type ResetPasswordCommand struct {
	Phone    string
	Role     domain.UserRole
	Password string
}

type ResetPasswordRecord struct {
	Phone        string
	Role         domain.UserRole
	PasswordHash string
}

type ResetPasswordResult struct {
	UserID            uuid.UUID       `json:"user_id"`
	Phone             string          `json:"phone"`
	Role              domain.UserRole `json:"role"`
	RevokedTokenCount int64           `json:"revoked_token_count"`
}

type ResetPasswordCommandResult struct {
	ResetPasswordResult
	GeneratedPassword string `json:"generated_password,omitempty"`
	PasswordGenerated bool   `json:"password_generated"`
}

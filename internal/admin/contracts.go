// Package admin contains application services for privileged console operations.
package admin

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type PasswordHasher interface {
	HashPassword(password string) (string, error)
}

type Repository interface {
	CreateTaxiParkOwner(ctx context.Context, record CreateTaxiParkOwnerRecord) (CreateTaxiParkOwnerResult, error)
	ResetPasswordByPhone(ctx context.Context, record ResetPasswordRecord) (ResetPasswordResult, error)
	ListTaxiParkAccounts(ctx context.Context, filter ListTaxiParkAccountsFilter) ([]TaxiParkAccount, error)
	ListCities(ctx context.Context) ([]CityRecord, error)
	GetMonitorDatabaseSnapshot(ctx context.Context) (MonitorDatabaseSnapshot, error)
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

type ListTaxiParkAccountsCommand struct {
	Limit          int
	Search         string
	IncludeDeleted bool
}

type ListTaxiParkAccountsFilter struct {
	Limit          int
	Search         string
	IncludeDeleted bool
}

type TaxiParkAccount struct {
	TaxiParkID         uuid.UUID  `json:"taxi_park_id"`
	OwnerUserID        uuid.UUID  `json:"owner_user_id"`
	CityID             uuid.UUID  `json:"city_id"`
	CityName           string     `json:"city_name"`
	Name               string     `json:"name"`
	LegalName          string     `json:"legal_name,omitempty"`
	TaxID              string     `json:"tax_id,omitempty"`
	ContactPhone       string     `json:"contact_phone"`
	ContactEmail       string     `json:"contact_email"`
	OwnerPhone         string     `json:"owner_phone"`
	OwnerEmail         string     `json:"owner_email,omitempty"`
	IsVerified         bool       `json:"is_verified"`
	VerificationStatus string     `json:"verification_status"`
	CommissionPercent  string     `json:"commission_percent,omitempty"`
	BalanceCents       int64      `json:"balance_cents"`
	IsOwnerActive      bool       `json:"is_owner_active"`
	CreatedAt          time.Time  `json:"created_at"`
	DeletedAt          *time.Time `json:"deleted_at,omitempty"`
}

type CityRecord struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Region      string    `json:"region"`
	CountryCode string    `json:"country_code"`
	Timezone    string    `json:"timezone"`
	Latitude    float64   `json:"latitude"`
	Longitude   float64   `json:"longitude"`
	IsActive    bool      `json:"is_active"`
}

type MonitorDatabaseSnapshot struct {
	CollectedAt          time.Time `json:"collected_at"`
	TotalUsers           int64     `json:"total_users"`
	ActiveUsers          int64     `json:"active_users"`
	RecentlyActiveUsers  int64     `json:"recently_active_users"`
	TotalTaxiParks       int64     `json:"total_taxi_parks"`
	ActiveTaxiParks      int64     `json:"active_taxi_parks"`
	TotalDrivers         int64     `json:"total_drivers"`
	OnlineDrivers        int64     `json:"online_drivers"`
	BusyDrivers          int64     `json:"busy_drivers"`
	BlockedDrivers       int64     `json:"blocked_drivers"`
	TotalOrders          int64     `json:"total_orders"`
	ActiveOrders         int64     `json:"active_orders"`
	SearchingOrders      int64     `json:"searching_orders"`
	AssignedOrders       int64     `json:"assigned_orders"`
	InProgressOrders     int64     `json:"in_progress_orders"`
	CompletedOrdersToday int64     `json:"completed_orders_today"`
	CancelledOrdersToday int64     `json:"cancelled_orders_today"`
	FailedOrdersToday    int64     `json:"failed_orders_today"`
}

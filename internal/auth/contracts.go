package auth

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type UserRepository interface {
	CreateUser(ctx context.Context, user domain.User) (domain.User, error)
	GetUserByID(ctx context.Context, userID uuid.UUID) (domain.User, error)
	GetUserByPhoneAndRole(ctx context.Context, phone string, role domain.UserRole) (domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (domain.User, error)
	GetDriverLoginAccess(ctx context.Context, userID uuid.UUID) (DriverLoginAccess, error)
	MarkPhoneConfirmed(ctx context.Context, userID uuid.UUID, confirmedAt time.Time) error
	MarkEmailConfirmed(ctx context.Context, userID uuid.UUID, confirmedAt time.Time) error
}

type DriverLoginAccess struct {
	DriverID           uuid.UUID
	TaxiParkID         *uuid.UUID
	VerificationStatus domain.VerificationLifecycleStatus
	TaxiParkActive     bool
}

type TaxiParkRepository interface {
	CreateTaxiPark(ctx context.Context, taxiPark domain.TaxiPark) (domain.TaxiPark, error)
	GetTaxiParkByOwnerUserID(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiPark, error)
}

type VerificationCodeRepository interface {
	CreateVerificationCode(ctx context.Context, code domain.VerificationCode) (domain.VerificationCode, error)
	GetLatestActiveCode(ctx context.Context, target string, channel domain.VerificationChannel, purpose domain.VerificationPurpose) (domain.VerificationCode, error)
	IncrementAttempts(ctx context.Context, codeID uuid.UUID) error
	ConsumeCode(ctx context.Context, codeID uuid.UUID, consumedAt time.Time) error
}

type RefreshTokenRepository interface {
	StoreRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	RotateRefreshToken(ctx context.Context, oldTokenHash string, userID uuid.UUID, newTokenHash string, newExpiresAt time.Time) error
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
}

type UserConsentEventRepository interface {
	CreateUserConsentEvent(ctx context.Context, event domain.UserConsentEvent) error
}

type SMSProvider interface {
	SendVerificationCode(ctx context.Context, phone string, code string) error
}

type EmailProvider interface {
	SendEmailConfirmationCode(ctx context.Context, email string, code string) error
}

type PasswordHasher interface {
	HashPassword(password string) (string, error)
	ComparePasswordAndHash(password string, hash string) error
}

type CodeHasher interface {
	HashCode(code string) (string, error)
	CompareCodeAndHash(code string, hash string) error
}

type CodeGenerator interface {
	GenerateNumericCode(length int) (string, error)
}

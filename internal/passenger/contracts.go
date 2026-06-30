package passenger

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type Repository interface {
	Create(ctx context.Context, passenger domain.Passenger) (domain.Passenger, error)
	GetByID(ctx context.Context, passengerID uuid.UUID) (domain.Passenger, error)
	GetByPhone(ctx context.Context, phone string) (domain.Passenger, error)
	UpdateProfile(ctx context.Context, passengerID uuid.UUID, name string, email string, avatarURL string) (domain.Passenger, error)
	MarkAuthenticated(ctx context.Context, passengerID uuid.UUID, phoneVerifiedAt *time.Time, lastLoginAt time.Time) (domain.Passenger, error)
}

type AuthCodeRepository interface {
	Create(ctx context.Context, code domain.PassengerAuthCode) (domain.PassengerAuthCode, error)
	InvalidateActiveByPhone(ctx context.Context, phone string, invalidatedAt time.Time) error
	GetLatestActiveByPhone(ctx context.Context, phone string) (domain.PassengerAuthCode, error)
	IncrementAttempts(ctx context.Context, codeID uuid.UUID) error
	MarkUsed(ctx context.Context, codeID uuid.UUID, usedAt time.Time) error
}

type RefreshTokenRepository interface {
	Store(ctx context.Context, passengerID uuid.UUID, tokenHash string, expiresAt time.Time) error
	Rotate(ctx context.Context, oldTokenHash string, passengerID uuid.UUID, newTokenHash string, newExpiresAt time.Time) error
	Revoke(ctx context.Context, tokenHash string) error
}

type SMSService interface {
	SendCode(ctx context.Context, phone string, code string) error
}

type CodeGenerator interface {
	GenerateNumericCode(length int) (string, error)
}

type CodeHasher interface {
	HashCode(code string) (string, error)
	CompareCodeAndHash(code string, hash string) error
}

type AuthUseCase interface {
	RequestCode(ctx context.Context, request dto.PassengerAuthRequestCodeRequest) (dto.PassengerAuthRequestCodeResponse, error)
	ConfirmCode(ctx context.Context, request dto.PassengerAuthConfirmCodeRequest) (dto.PassengerAuthTokenResponse, error)
	Refresh(ctx context.Context, request dto.RefreshTokenRequest) (dto.PassengerAuthRefreshResponse, error)
	Logout(ctx context.Context, request dto.LogoutRequest) error
}

type ProfileUseCase interface {
	GetMe(ctx context.Context, passengerID uuid.UUID) (dto.PassengerMeResponse, error)
	UpdateMe(ctx context.Context, passengerID uuid.UUID, request dto.PassengerMePatchRequest) (dto.PassengerMeResponse, error)
}

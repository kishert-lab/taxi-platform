package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type MobileService struct {
	userRepository             UserRepository
	verificationCodeRepository VerificationCodeRepository
	refreshTokenRepository     RefreshTokenRepository
	codeGenerator              CodeGenerator
	codeHasher                 CodeHasher
	tokenManager               *TokenManager
	logger                     *zap.Logger
	codeLength                 int
	phoneCodeTTL               time.Duration
	emailCodeTTL               time.Duration
	maxCodeAttempts            int
}

type NewMobileServiceParams struct {
	UserRepository             UserRepository
	VerificationCodeRepository VerificationCodeRepository
	RefreshTokenRepository     RefreshTokenRepository
	CodeGenerator              CodeGenerator
	CodeHasher                 CodeHasher
	TokenManager               *TokenManager
	Logger                     *zap.Logger
	CodeLength                 int
	PhoneCodeTTL               time.Duration
	EmailCodeTTL               time.Duration
	MaxCodeAttempts            int
}

func NewMobileService(params NewMobileServiceParams) *MobileService {
	return &MobileService{
		userRepository:             params.UserRepository,
		verificationCodeRepository: params.VerificationCodeRepository,
		refreshTokenRepository:     params.RefreshTokenRepository,
		codeGenerator:              params.CodeGenerator,
		codeHasher:                 params.CodeHasher,
		tokenManager:               params.TokenManager,
		logger:                     params.Logger,
		codeLength:                 params.CodeLength,
		phoneCodeTTL:               params.PhoneCodeTTL,
		emailCodeTTL:               params.EmailCodeTTL,
		maxCodeAttempts:            params.MaxCodeAttempts,
	}
}

func (service *MobileService) StartLogin(ctx context.Context, request dto.AuthLoginRequest) (dto.AuthCodeSentResponse, error) {
	if err := request.Role.Validate(); err != nil {
		return dto.AuthCodeSentResponse{}, fmt.Errorf("validate login role: %w", err)
	}

	user, target, channel, ttl, err := service.resolveLoginTarget(ctx, request)
	if err != nil {
		return dto.AuthCodeSentResponse{}, err
	}
	if !user.IsActive {
		return dto.AuthCodeSentResponse{}, ErrInactiveUser
	}

	code, err := service.codeGenerator.GenerateNumericCode(service.codeLength)
	if err != nil {
		return dto.AuthCodeSentResponse{}, fmt.Errorf("generate login code: %w", err)
	}
	codeHash, err := service.codeHasher.HashCode(code)
	if err != nil {
		return dto.AuthCodeSentResponse{}, fmt.Errorf("hash login code: %w", err)
	}

	now := time.Now().UTC()
	verificationCode := domain.VerificationCode{
		UserID:      user.ID,
		Target:      target,
		Channel:     channel,
		Purpose:     domain.VerificationPurposeLogin,
		CodeHash:    codeHash,
		MaxAttempts: service.maxCodeAttempts,
		ExpiresAt:   now.Add(ttl),
		LastSentAt:  now,
	}
	if _, err := service.verificationCodeRepository.CreateVerificationCode(ctx, verificationCode); err != nil {
		return dto.AuthCodeSentResponse{}, fmt.Errorf("store login verification code: %w", err)
	}

	service.logger.Info(
		"auth login verification code issued",
		zap.String("user_id", user.ID.String()),
		zap.String("role", string(user.Role)),
		zap.String("channel", string(channel)),
		zap.String("target", target),
		zap.String("verification_code", code),
	)

	return dto.AuthCodeSentResponse{
		DeliveryChannel: string(channel),
		Message:         "verification code sent",
		DebugCode:       code,
	}, nil
}

func (service *MobileService) VerifyCode(ctx context.Context, request dto.AuthVerifyCodeRequest) (dto.AuthTokenResponse, error) {
	if err := request.Role.Validate(); err != nil {
		return dto.AuthTokenResponse{}, fmt.Errorf("validate verification role: %w", err)
	}

	user, target, channel, _, err := service.resolveLoginTarget(ctx, dto.AuthLoginRequest{
		Phone: request.Phone,
		Email: request.Email,
		Role:  request.Role,
	})
	if err != nil {
		return dto.AuthTokenResponse{}, err
	}
	if !user.IsActive {
		return dto.AuthTokenResponse{}, ErrInactiveUser
	}

	verificationCode, err := service.verificationCodeRepository.GetLatestActiveCode(ctx, target, channel, domain.VerificationPurposeLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.AuthTokenResponse{}, ErrInvalidCode
		}
		return dto.AuthTokenResponse{}, fmt.Errorf("get login verification code: %w", err)
	}
	if time.Now().UTC().After(verificationCode.ExpiresAt) {
		return dto.AuthTokenResponse{}, ErrInvalidCode
	}
	if err := service.codeHasher.CompareCodeAndHash(request.Code, verificationCode.CodeHash); err != nil {
		if incrementErr := service.verificationCodeRepository.IncrementAttempts(ctx, verificationCode.ID); incrementErr != nil {
			return dto.AuthTokenResponse{}, fmt.Errorf("increment login verification attempts: %w", incrementErr)
		}
		return dto.AuthTokenResponse{}, ErrInvalidCode
	}

	now := time.Now().UTC()
	if err := service.verificationCodeRepository.ConsumeCode(ctx, verificationCode.ID, now); err != nil {
		return dto.AuthTokenResponse{}, fmt.Errorf("consume login verification code: %w", err)
	}

	return service.issueAndStoreTokenPair(ctx, user.ID, user.Role, now)
}

func (service *MobileService) Refresh(ctx context.Context, request dto.RefreshTokenRequest) (dto.AuthTokenResponse, error) {
	claims, err := service.tokenManager.ParseRefreshToken(request.RefreshToken)
	if err != nil {
		return dto.AuthTokenResponse{}, ErrInvalidToken
	}

	user, err := service.userRepository.GetUserByID(ctx, claims.Subject)
	if err != nil {
		return dto.AuthTokenResponse{}, ErrInvalidToken
	}
	if !user.IsActive || user.Role != claims.Role {
		return dto.AuthTokenResponse{}, ErrInvalidToken
	}

	now := time.Now().UTC()
	accessToken, refreshToken, refreshExpiresAt, err := service.tokenManager.IssueTokenPair(user.ID, user.Role, now)
	if err != nil {
		return dto.AuthTokenResponse{}, fmt.Errorf("issue refreshed token pair: %w", err)
	}
	if err := service.refreshTokenRepository.RotateRefreshToken(
		ctx,
		HashToken(request.RefreshToken),
		user.ID,
		HashToken(refreshToken),
		refreshExpiresAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.AuthTokenResponse{}, ErrInvalidToken
		}
		return dto.AuthTokenResponse{}, fmt.Errorf("rotate refresh token: %w", err)
	}

	return dto.AuthTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    service.tokenManager.AccessTTLSeconds(),
	}, nil
}

func (service *MobileService) Logout(ctx context.Context, request dto.LogoutRequest) error {
	if request.RefreshToken == "" {
		return ErrInvalidToken
	}
	if err := service.refreshTokenRepository.RevokeRefreshToken(ctx, HashToken(request.RefreshToken)); err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}

	return nil
}

func (service *MobileService) AuthenticateWebSocket(_ context.Context, token string) (uuid.UUID, domain.UserRole, error) {
	claims, err := service.tokenManager.ParseAccessToken(token)
	if err != nil {
		return uuid.Nil, "", ErrInvalidToken
	}

	return claims.Subject, claims.Role, nil
}

func (service *MobileService) issueAndStoreTokenPair(ctx context.Context, userID uuid.UUID, role domain.UserRole, now time.Time) (dto.AuthTokenResponse, error) {
	accessToken, refreshToken, refreshExpiresAt, err := service.tokenManager.IssueTokenPair(userID, role, now)
	if err != nil {
		return dto.AuthTokenResponse{}, fmt.Errorf("issue token pair: %w", err)
	}
	if err := service.refreshTokenRepository.StoreRefreshToken(ctx, userID, HashToken(refreshToken), refreshExpiresAt); err != nil {
		return dto.AuthTokenResponse{}, fmt.Errorf("store refresh token: %w", err)
	}

	return dto.AuthTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    service.tokenManager.AccessTTLSeconds(),
	}, nil
}

func (service *MobileService) resolveLoginTarget(ctx context.Context, request dto.AuthLoginRequest) (domain.User, string, domain.VerificationChannel, time.Duration, error) {
	if request.Phone != "" {
		phone, err := domain.NormalizePhone(request.Phone)
		if err != nil {
			return domain.User{}, "", "", 0, fmt.Errorf("normalize login phone: %w", err)
		}
		user, err := service.userRepository.GetUserByPhoneAndRole(ctx, phone, request.Role)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return domain.User{}, "", "", 0, ErrInvalidCredentials
			}
			return domain.User{}, "", "", 0, fmt.Errorf("get login user by phone: %w", err)
		}
		return user, phone, domain.VerificationChannelSMS, service.phoneCodeTTL, nil
	}

	email, err := domain.NormalizeEmail(request.Email)
	if err != nil {
		return domain.User{}, "", "", 0, fmt.Errorf("normalize login email: %w", err)
	}
	user, err := service.userRepository.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, "", "", 0, ErrInvalidCredentials
		}
		return domain.User{}, "", "", 0, fmt.Errorf("get login user by email: %w", err)
	}
	if user.Role != request.Role {
		return domain.User{}, "", "", 0, ErrInvalidCredentials
	}

	return user, email, domain.VerificationChannelEmail, service.emailCodeTTL, nil
}

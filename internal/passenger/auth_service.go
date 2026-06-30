package passenger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type AuthService struct {
	repository             Repository
	authCodeRepository     AuthCodeRepository
	refreshTokenRepository RefreshTokenRepository
	smsService             SMSService
	codeGenerator          CodeGenerator
	codeHasher             CodeHasher
	tokenManager           *TokenManager
	logger                 *zap.Logger
	codeLength             int
	codeTTL                time.Duration
	maxCodeAttempts        int
	devCode                string
}

type AuthServiceParams struct {
	Repository             Repository
	AuthCodeRepository     AuthCodeRepository
	RefreshTokenRepository RefreshTokenRepository
	SMSService             SMSService
	CodeGenerator          CodeGenerator
	CodeHasher             CodeHasher
	TokenManager           *TokenManager
	Logger                 *zap.Logger
	CodeLength             int
	CodeTTL                time.Duration
	MaxCodeAttempts        int
	DevCode                string
}

func NewAuthService(params AuthServiceParams) *AuthService {
	return &AuthService{
		repository:             params.Repository,
		authCodeRepository:     params.AuthCodeRepository,
		refreshTokenRepository: params.RefreshTokenRepository,
		smsService:             params.SMSService,
		codeGenerator:          params.CodeGenerator,
		codeHasher:             params.CodeHasher,
		tokenManager:           params.TokenManager,
		logger:                 params.Logger,
		codeLength:             params.CodeLength,
		codeTTL:                params.CodeTTL,
		maxCodeAttempts:        params.MaxCodeAttempts,
		devCode:                strings.TrimSpace(params.DevCode),
	}
}

func (service *AuthService) RequestCode(ctx context.Context, request dto.PassengerAuthRequestCodeRequest) (dto.PassengerAuthRequestCodeResponse, error) {
	phone, err := domain.NormalizePhone(request.Phone)
	if err != nil {
		return dto.PassengerAuthRequestCodeResponse{}, fmt.Errorf("normalize passenger phone: %w", err)
	}

	codeValue, err := service.generateCode()
	if err != nil {
		return dto.PassengerAuthRequestCodeResponse{}, err
	}
	codeHash, err := service.codeHasher.HashCode(codeValue)
	if err != nil {
		return dto.PassengerAuthRequestCodeResponse{}, fmt.Errorf("hash passenger code: %w", err)
	}

	now := time.Now().UTC()
	if err := service.authCodeRepository.InvalidateActiveByPhone(ctx, phone, now); err != nil {
		return dto.PassengerAuthRequestCodeResponse{}, fmt.Errorf("invalidate active passenger codes: %w", err)
	}

	_, err = service.authCodeRepository.Create(ctx, domain.PassengerAuthCode{
		Phone:       phone,
		CodeHash:    codeHash,
		MaxAttempts: service.maxCodeAttempts,
		ExpiresAt:   now.Add(service.codeTTL),
	})
	if err != nil {
		return dto.PassengerAuthRequestCodeResponse{}, fmt.Errorf("create passenger auth code: %w", err)
	}

	if err := service.smsService.SendCode(ctx, phone, codeValue); err != nil {
		return dto.PassengerAuthRequestCodeResponse{}, fmt.Errorf("send passenger sms code: %w", err)
	}

	service.logger.Info("passenger auth code issued", zap.String("phone", phone))

	return dto.PassengerAuthRequestCodeResponse{
		Success: true,
		Message: "Код подтверждения отправлен",
	}, nil
}

func (service *AuthService) ConfirmCode(ctx context.Context, request dto.PassengerAuthConfirmCodeRequest) (dto.PassengerAuthTokenResponse, error) {
	phone, err := domain.NormalizePhone(request.Phone)
	if err != nil {
		return dto.PassengerAuthTokenResponse{}, fmt.Errorf("normalize passenger phone: %w", err)
	}

	codeRecord, err := service.authCodeRepository.GetLatestActiveByPhone(ctx, phone)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.PassengerAuthTokenResponse{}, ErrInvalidCode
		}
		return dto.PassengerAuthTokenResponse{}, fmt.Errorf("get passenger auth code: %w", err)
	}

	now := time.Now().UTC()
	if codeRecord.UsedAt != nil {
		return dto.PassengerAuthTokenResponse{}, ErrCodeAlreadyUsed
	}
	if now.After(codeRecord.ExpiresAt) {
		return dto.PassengerAuthTokenResponse{}, ErrCodeExpired
	}
	if codeRecord.Attempts >= codeRecord.MaxAttempts {
		return dto.PassengerAuthTokenResponse{}, ErrTooManyAttempts
	}
	if err := service.codeHasher.CompareCodeAndHash(strings.TrimSpace(request.Code), codeRecord.CodeHash); err != nil {
		if incrementErr := service.authCodeRepository.IncrementAttempts(ctx, codeRecord.ID); incrementErr != nil {
			return dto.PassengerAuthTokenResponse{}, fmt.Errorf("increment passenger auth attempts: %w", incrementErr)
		}
		return dto.PassengerAuthTokenResponse{}, ErrInvalidCode
	}

	passengerRecord, err := service.repository.GetByPhone(ctx, phone)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return dto.PassengerAuthTokenResponse{}, fmt.Errorf("get passenger by phone: %w", err)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		passengerRecord, err = service.repository.Create(ctx, domain.Passenger{
			Phone:    phone,
			Name:     domain.NormalizePassengerName(request.Name),
			IsActive: true,
		})
		if err != nil {
			return dto.PassengerAuthTokenResponse{}, fmt.Errorf("create passenger: %w", err)
		}
	}
	if !passengerRecord.IsActive {
		return dto.PassengerAuthTokenResponse{}, ErrPassengerBlocked
	}

	var phoneVerifiedAt *time.Time
	if passengerRecord.PhoneVerifiedAt == nil {
		phoneVerifiedAt = &now
	}
	passengerRecord, err = service.repository.MarkAuthenticated(ctx, passengerRecord.ID, phoneVerifiedAt, now)
	if err != nil {
		return dto.PassengerAuthTokenResponse{}, fmt.Errorf("mark passenger authenticated: %w", err)
	}
	if err := service.authCodeRepository.MarkUsed(ctx, codeRecord.ID, now); err != nil {
		return dto.PassengerAuthTokenResponse{}, fmt.Errorf("mark passenger auth code used: %w", err)
	}

	accessToken, refreshToken, refreshExpiresAt, err := service.tokenManager.GenerateTokenPair(passengerRecord.ID, now)
	if err != nil {
		return dto.PassengerAuthTokenResponse{}, fmt.Errorf("issue passenger token pair: %w", err)
	}
	if err := service.refreshTokenRepository.Store(ctx, passengerRecord.ID, HashToken(refreshToken), refreshExpiresAt); err != nil {
		return dto.PassengerAuthTokenResponse{}, fmt.Errorf("store passenger refresh token: %w", err)
	}

	service.logger.Info("passenger authenticated", zap.String("passenger_id", passengerRecord.ID.String()))

	return dto.PassengerAuthTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    service.tokenManager.AccessTTLSeconds(),
		Passenger:    passengerDTO(passengerRecord),
	}, nil
}

func (service *AuthService) Refresh(ctx context.Context, request dto.RefreshTokenRequest) (dto.PassengerAuthRefreshResponse, error) {
	claims, err := service.tokenManager.ParsePassengerRefreshToken(request.RefreshToken)
	if err != nil {
		return dto.PassengerAuthRefreshResponse{}, ErrInvalidRefreshToken
	}

	passengerRecord, err := service.repository.GetByID(ctx, claims.Subject)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.PassengerAuthRefreshResponse{}, ErrInvalidRefreshToken
		}
		return dto.PassengerAuthRefreshResponse{}, fmt.Errorf("get passenger by refresh token: %w", err)
	}
	if !passengerRecord.IsActive {
		return dto.PassengerAuthRefreshResponse{}, ErrPassengerBlocked
	}

	now := time.Now().UTC()
	accessToken, refreshToken, refreshExpiresAt, err := service.tokenManager.GenerateTokenPair(passengerRecord.ID, now)
	if err != nil {
		return dto.PassengerAuthRefreshResponse{}, fmt.Errorf("issue passenger refreshed tokens: %w", err)
	}
	if err := service.refreshTokenRepository.Rotate(ctx, HashToken(request.RefreshToken), passengerRecord.ID, HashToken(refreshToken), refreshExpiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dto.PassengerAuthRefreshResponse{}, ErrInvalidRefreshToken
		}
		return dto.PassengerAuthRefreshResponse{}, fmt.Errorf("rotate passenger refresh token: %w", err)
	}

	return dto.PassengerAuthRefreshResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    service.tokenManager.AccessTTLSeconds(),
	}, nil
}

func (service *AuthService) Logout(ctx context.Context, request dto.LogoutRequest) error {
	if strings.TrimSpace(request.RefreshToken) == "" {
		return ErrInvalidRefreshToken
	}
	if err := service.refreshTokenRepository.Revoke(ctx, HashToken(request.RefreshToken)); err != nil {
		return fmt.Errorf("revoke passenger refresh token: %w", err)
	}

	return nil
}

func (service *AuthService) generateCode() (string, error) {
	if service.devCode != "" {
		return service.devCode, nil
	}

	code, err := service.codeGenerator.GenerateNumericCode(service.codeLength)
	if err != nil {
		return "", fmt.Errorf("generate passenger auth code: %w", err)
	}

	return code, nil
}

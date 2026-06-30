package passenger

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
)

type PushTokenRepository interface {
	Upsert(ctx context.Context, token domain.PassengerPushToken) (domain.PassengerPushToken, error)
	ListActiveTokens(ctx context.Context, passengerID uuid.UUID) ([]domain.PassengerPushToken, error)
}

type PushTokenUseCase interface {
	RegisterToken(ctx context.Context, passengerID uuid.UUID, request dto.PassengerPushTokenRequest) (dto.PassengerPushTokenResponse, error)
}

type PushTokenService struct {
	repository PushTokenRepository
}

func NewPushTokenService(repository PushTokenRepository) *PushTokenService {
	return &PushTokenService{repository: repository}
}

func (service *PushTokenService) RegisterToken(ctx context.Context, passengerID uuid.UUID, request dto.PassengerPushTokenRequest) (dto.PassengerPushTokenResponse, error) {
	token := strings.TrimSpace(request.Token)
	platform := domain.NormalizePushPlatform(request.Platform)
	if token == "" {
		return dto.PassengerPushTokenResponse{}, fmt.Errorf("normalize push token request: %w", domain.ErrInvalidPushToken)
	}
	if platform == "" {
		return dto.PassengerPushTokenResponse{}, fmt.Errorf("normalize push token request: %w", domain.ErrInvalidPushPlatform)
	}

	record, err := service.repository.Upsert(ctx, domain.PassengerPushToken{
		PassengerID: passengerID,
		Token:       token,
		Platform:    platform,
		DeviceID:    strings.TrimSpace(request.DeviceID),
		IsActive:    true,
	})
	if err != nil {
		return dto.PassengerPushTokenResponse{}, fmt.Errorf("upsert passenger push token: %w", err)
	}

	return dto.PassengerPushTokenResponse{
		Token:    record.Token,
		Platform: record.Platform,
		DeviceID: record.DeviceID,
	}, nil
}

package push

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	provider   Provider
	repository PassengerTokenRepository
	logger     *zap.Logger
	enabled    bool
}

type ServiceParams struct {
	Provider   Provider
	Repository PassengerTokenRepository
	Logger     *zap.Logger
	Enabled    bool
}

func NewService(params ServiceParams) *Service {
	logger := params.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		provider:   params.Provider,
		repository: params.Repository,
		logger:     logger,
		enabled:    params.Enabled,
	}
}

func (service *Service) NotifyPassenger(ctx context.Context, passengerID uuid.UUID, notification Notification) error {
	if !service.enabled || service.provider == nil || service.repository == nil || passengerID == uuid.Nil {
		return nil
	}

	tokens, err := service.repository.ListActiveTokens(ctx, passengerID)
	if err != nil {
		return fmt.Errorf("list active passenger push tokens: %w", err)
	}
	if len(tokens) == 0 {
		return nil
	}

	tokenValues := make([]string, 0, len(tokens))
	for _, token := range tokens {
		tokenValues = append(tokenValues, token.Token)
	}

	if err := service.provider.SendToTokens(ctx, tokenValues, notification); err != nil {
		return fmt.Errorf("send passenger push notification: %w", err)
	}

	service.logger.Info(
		"passenger push notification sent",
		zap.String("passenger_id", passengerID.String()),
		zap.Int("tokens", len(tokenValues)),
		zap.String("title", notification.Title),
	)
	return nil
}

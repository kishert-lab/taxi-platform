package push

import (
	"context"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type Provider interface {
	SendToTokens(ctx context.Context, tokens []string, notification Notification) error
}

type PassengerTokenRepository interface {
	ListActiveTokens(ctx context.Context, passengerID uuid.UUID) ([]domain.PassengerPushToken, error)
}

type Notification struct {
	Title string
	Body  string
	Data  map[string]string
}

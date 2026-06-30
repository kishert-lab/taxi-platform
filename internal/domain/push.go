package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidPushToken    = errors.New("invalid push token")
	ErrInvalidPushPlatform = errors.New("invalid push platform")
)

type PassengerPushToken struct {
	ID         uuid.UUID
	PassengerID uuid.UUID
	Token      string
	Platform   string
	DeviceID   string
	IsActive   bool
	LastSeenAt time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  *time.Time
}

func NormalizePushPlatform(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

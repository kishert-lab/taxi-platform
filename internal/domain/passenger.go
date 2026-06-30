package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Passenger struct {
	ID              uuid.UUID
	Phone           string
	Name            string
	Email           string
	AvatarURL       string
	IsActive        bool
	PhoneVerifiedAt *time.Time
	LastLoginAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       *time.Time
}

type PassengerAuthCode struct {
	ID          uuid.UUID
	Phone       string
	CodeHash    string
	Attempts    int
	MaxAttempts int
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

var (
	ErrPassengerBlocked       = errors.New("passenger blocked")
	ErrPassengerCodeExpired   = errors.New("passenger code expired")
	ErrPassengerCodeUsed      = errors.New("passenger code already used")
	ErrPassengerCodeAttempts  = errors.New("passenger code attempts exceeded")
	ErrPassengerCodeInvalid   = errors.New("passenger code invalid")
	ErrPassengerTokenInvalid  = errors.New("passenger token invalid")
	ErrPassengerRefreshDenied = errors.New("passenger refresh denied")
)

func NormalizePassengerName(name string) string {
	return strings.TrimSpace(name)
}

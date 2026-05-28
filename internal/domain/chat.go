package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ChatType string

const (
	ChatTypeDispatcherDriver ChatType = "dispatcher_driver"
	ChatTypeDriverPassenger  ChatType = "driver_passenger"
	ChatTypePassengerSupport ChatType = "passenger_support"
)

type ChatStatus string

const (
	ChatStatusOpen     ChatStatus = "open"
	ChatStatusClosed   ChatStatus = "closed"
	ChatStatusArchived ChatStatus = "archived"
)

type ChatThread struct {
	ID          uuid.UUID
	Type        ChatType
	OrderID     *uuid.UUID
	TaxiParkID  *uuid.UUID
	PassengerID *uuid.UUID
	DriverID    *uuid.UUID
	Status      ChatStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ClosedAt    *time.Time
}

type ChatMessage struct {
	ID           uuid.UUID
	ThreadID     uuid.UUID
	OrderID      *uuid.UUID
	SenderUserID uuid.UUID
	SenderRole   UserRole
	Body         string
	CreatedAt    time.Time
	EditedAt     *time.Time
}

var (
	ErrInvalidChatType    = errors.New("invalid chat type")
	ErrInvalidChatStatus  = errors.New("invalid chat status")
	ErrInvalidChatMessage = errors.New("invalid chat message")
)

func (chatType ChatType) Validate() error {
	switch chatType {
	case ChatTypeDispatcherDriver, ChatTypeDriverPassenger, ChatTypePassengerSupport:
		return nil
	default:
		return ErrInvalidChatType
	}
}

func (status ChatStatus) Validate() error {
	switch status {
	case ChatStatusOpen, ChatStatusClosed, ChatStatusArchived:
		return nil
	default:
		return ErrInvalidChatStatus
	}
}

func NormalizeChatMessageBody(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || len([]rune(trimmed)) > 2000 {
		return "", ErrInvalidChatMessage
	}
	return trimmed, nil
}

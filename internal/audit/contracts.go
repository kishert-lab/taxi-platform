package audit

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type TransportRequestLogRepository interface {
	CreateTransportRequestLog(ctx context.Context, logEntry TransportRequestLog) error
}

type TransportRequestLog struct {
	Protocol     string
	EventType    string
	RequestID    string
	Method       string
	Route        string
	Path         string
	RawQuery     string
	StatusCode   int
	Duration     time.Duration
	ClientIP     string
	UserAgent    string
	ActorUserID  uuid.UUID
	ActorRole    domain.UserRole
	ErrorMessage string
	Metadata     map[string]any
}

type HTTPRequestLogCommand struct {
	RequestID    string
	Method       string
	Route        string
	Path         string
	RawQuery     string
	StatusCode   int
	Duration     time.Duration
	ClientIP     string
	UserAgent    string
	ActorUserID  uuid.UUID
	ActorRole    domain.UserRole
	ErrorMessage string
	ContentType  string
	RequestBody  string
}

type WebSocketRequestLogCommand struct {
	RequestID    string
	EventType    string
	Path         string
	RawQuery     string
	StatusCode   int
	Duration     time.Duration
	ClientIP     string
	UserAgent    string
	ActorUserID  uuid.UUID
	ActorRole    domain.UserRole
	ErrorMessage string
	CloseReason  string
}

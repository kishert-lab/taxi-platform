package audit

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

const maxLoggedBodyLength = 4096

type Service struct {
	repository   TransportRequestLogRepository
	logger       *zap.Logger
	writeTimeout time.Duration
}

func NewService(repository TransportRequestLogRepository, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{
		repository:   repository,
		logger:       logger,
		writeTimeout: 2 * time.Second,
	}
}

func (service *Service) LogHTTP(ctx context.Context, command HTTPRequestLogCommand) {
	if service == nil || service.repository == nil {
		return
	}

	metadata := map[string]any{
		"content_type": command.ContentType,
	}
	if command.RequestBody != "" {
		metadata["request_body"] = truncate(command.RequestBody, maxLoggedBodyLength)
	}

	service.persist(ctx, TransportRequestLog{
		Protocol:     "http",
		EventType:    "request.completed",
		RequestID:    command.RequestID,
		Method:       command.Method,
		Route:        command.Route,
		Path:         command.Path,
		RawQuery:     command.RawQuery,
		StatusCode:   command.StatusCode,
		Duration:     command.Duration,
		ClientIP:     command.ClientIP,
		UserAgent:    command.UserAgent,
		ActorUserID:  command.ActorUserID,
		ActorRole:    command.ActorRole,
		ErrorMessage: command.ErrorMessage,
		Metadata:     metadata,
	})
}

func (service *Service) LogWebSocket(ctx context.Context, command WebSocketRequestLogCommand) {
	if service == nil || service.repository == nil {
		return
	}

	metadata := map[string]any{}
	if command.CloseReason != "" {
		metadata["close_reason"] = truncate(command.CloseReason, maxLoggedBodyLength)
	}

	service.persist(ctx, TransportRequestLog{
		Protocol:     "ws",
		EventType:    command.EventType,
		RequestID:    command.RequestID,
		Method:       "GET",
		Path:         command.Path,
		RawQuery:     command.RawQuery,
		StatusCode:   command.StatusCode,
		Duration:     command.Duration,
		ClientIP:     command.ClientIP,
		UserAgent:    command.UserAgent,
		ActorUserID:  command.ActorUserID,
		ActorRole:    command.ActorRole,
		ErrorMessage: command.ErrorMessage,
		Metadata:     metadata,
	})
}

func (service *Service) persist(ctx context.Context, logEntry TransportRequestLog) {
	writeContext, cancel := context.WithTimeout(context.Background(), service.writeTimeout)
	defer cancel()

	if ctx != nil {
		if deadline, ok := ctx.Deadline(); ok {
			if untilDeadline := time.Until(deadline); untilDeadline > 0 && untilDeadline < service.writeTimeout {
				var deadlineCancel context.CancelFunc
				writeContext, deadlineCancel = context.WithTimeout(context.Background(), untilDeadline)
				defer deadlineCancel()
			}
		}
	}

	if err := service.repository.CreateTransportRequestLog(writeContext, logEntry); err != nil {
		service.logger.Error("persist transport request log", zap.Error(fmt.Errorf("create transport request log: %w", err)))
	}
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}

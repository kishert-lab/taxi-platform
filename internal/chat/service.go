package chat

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	wsmsg "github.com/kishert-lab/taxi-platform/internal/ws"
)

const EventChatMessage = "chat.message"

var (
	ErrChatForbidden   = errors.New("chat forbidden")
	ErrChatUnavailable = errors.New("chat unavailable")
)

type Service struct {
	repository      Repository
	realtimeGateway RealtimeGateway
	logger          *zap.Logger
}

func NewService(repository Repository, realtimeGateway RealtimeGateway, logger *zap.Logger) *Service {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Service{repository: repository, realtimeGateway: realtimeGateway, logger: logger}
}

func (service *Service) ListOrderMessages(ctx context.Context, actorUserID uuid.UUID, actorRole domain.UserRole, orderID uuid.UUID, chatType domain.ChatType, limit int) (dto.ChatMessagesResponse, error) {
	thread, err := service.authorizedOrderThread(ctx, actorUserID, actorRole, orderID, chatType)
	if err != nil {
		return dto.ChatMessagesResponse{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	messages, err := service.repository.ListMessages(ctx, thread, limit)
	if err != nil {
		return dto.ChatMessagesResponse{}, err
	}
	return chatMessagesResponse(thread, messages), nil
}

func (service *Service) SendOrderMessage(ctx context.Context, actorUserID uuid.UUID, actorRole domain.UserRole, orderID uuid.UUID, chatType domain.ChatType, request dto.ChatSendMessageRequest) (dto.ChatMessageResponse, error) {
	body, err := domain.NormalizeChatMessageBody(request.Body)
	if err != nil {
		return dto.ChatMessageResponse{}, err
	}
	thread, err := service.authorizedOrderThread(ctx, actorUserID, actorRole, orderID, chatType)
	if err != nil {
		return dto.ChatMessageResponse{}, err
	}
	message, err := service.repository.CreateMessage(ctx, thread, actorUserID, actorRole, body)
	if err != nil {
		return dto.ChatMessageResponse{}, err
	}
	response := chatMessageResponse(thread, message)
	if err := service.publishOrderMessage(ctx, thread, response); err != nil {
		return dto.ChatMessageResponse{}, err
	}
	service.logger.Info("chat message created", zap.String("thread_id", thread.ID.String()), zap.String("chat_type", string(thread.Type)))
	return response, nil
}

func (service *Service) ListPassengerSupportMessages(ctx context.Context, passengerID uuid.UUID, limit int) (dto.ChatMessagesResponse, error) {
	thread, err := service.repository.EnsurePassengerSupportThread(ctx, passengerID)
	if err != nil {
		return dto.ChatMessagesResponse{}, err
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	messages, err := service.repository.ListMessages(ctx, thread, limit)
	if err != nil {
		return dto.ChatMessagesResponse{}, err
	}
	return chatMessagesResponse(thread, messages), nil
}

func (service *Service) SendPassengerSupportMessage(ctx context.Context, passengerID uuid.UUID, request dto.ChatSendMessageRequest) (dto.ChatMessageResponse, error) {
	body, err := domain.NormalizeChatMessageBody(request.Body)
	if err != nil {
		return dto.ChatMessageResponse{}, err
	}
	thread, err := service.repository.EnsurePassengerSupportThread(ctx, passengerID)
	if err != nil {
		return dto.ChatMessageResponse{}, err
	}
	message, err := service.repository.CreateMessage(ctx, thread, passengerID, domain.UserRolePassenger, body)
	if err != nil {
		return dto.ChatMessageResponse{}, err
	}
	response := chatMessageResponse(thread, message)
	if service.realtimeGateway != nil {
		if err := service.realtimeGateway.SendToPassenger(ctx, passengerID, EventChatMessage, wsmsg.PassengerChatMessagePayload{Message: response}); err != nil {
			return dto.ChatMessageResponse{}, fmt.Errorf("publish passenger support chat message: %w", err)
		}
	}
	return response, nil
}

func (service *Service) authorizedOrderThread(ctx context.Context, actorUserID uuid.UUID, actorRole domain.UserRole, orderID uuid.UUID, chatType domain.ChatType) (domain.ChatThread, error) {
	if err := chatType.Validate(); err != nil {
		return domain.ChatThread{}, err
	}
	orderContext, err := service.repository.GetOrderChatContext(ctx, orderID)
	if err != nil {
		return domain.ChatThread{}, err
	}
	if !orderContext.AllowsChat(chatType) {
		return domain.ChatThread{}, ErrChatUnavailable
	}
	allowed, err := service.actorCanUseOrderChat(ctx, actorUserID, actorRole, chatType, orderContext)
	if err != nil {
		return domain.ChatThread{}, err
	}
	if !allowed {
		return domain.ChatThread{}, ErrChatForbidden
	}
	return service.repository.EnsureOrderThread(ctx, orderID, chatType)
}

func (service *Service) actorCanUseOrderChat(ctx context.Context, actorUserID uuid.UUID, actorRole domain.UserRole, chatType domain.ChatType, orderContext OrderChatContext) (bool, error) {
	switch chatType {
	case domain.ChatTypeDispatcherDriver:
		if actorRole == domain.UserRoleDriver {
			return orderContext.DriverUserID != nil && *orderContext.DriverUserID == actorUserID, nil
		}
		if actorRole == domain.UserRoleTaxiPark || actorRole == domain.UserRoleDispatcher {
			if orderContext.TaxiParkID == nil {
				return false, nil
			}
			return service.repository.IsTaxiParkActor(ctx, *orderContext.TaxiParkID, actorUserID)
		}
	case domain.ChatTypeDriverPassenger:
		if actorRole == domain.UserRoleDriver {
			return orderContext.DriverUserID != nil && *orderContext.DriverUserID == actorUserID, nil
		}
		if actorRole == domain.UserRolePassenger {
			return orderContext.PassengerID == actorUserID, nil
		}
	}
	return false, nil
}

func (context OrderChatContext) AllowsChat(chatType domain.ChatType) bool {
	switch chatType {
	case domain.ChatTypeDispatcherDriver:
		return context.DriverID != nil && context.TaxiParkID != nil && !context.Status.IsTerminal()
	case domain.ChatTypeDriverPassenger:
		return context.DriverID != nil &&
			(context.Status == domain.OrderStatusDriverAssigned ||
				context.Status == domain.OrderStatusDriverArriving ||
				context.Status == domain.OrderStatusDriverWaiting ||
				context.Status == domain.OrderStatusInProgress)
	default:
		return false
	}
}

func (service *Service) publishOrderMessage(ctx context.Context, thread domain.ChatThread, message dto.ChatMessageResponse) error {
	if service.realtimeGateway == nil || thread.OrderID == nil {
		return nil
	}
	payload := wsmsg.PassengerChatMessagePayload{Message: message}
	switch thread.Type {
	case domain.ChatTypeDispatcherDriver:
		if thread.DriverID != nil {
			if err := service.realtimeGateway.SendToDriver(ctx, *thread.DriverID, EventChatMessage, payload); err != nil {
				return fmt.Errorf("publish chat message to driver: %w", err)
			}
		}
		return service.realtimeGateway.SendToTaxiParkByOrder(ctx, *thread.OrderID, EventChatMessage, payload)
	case domain.ChatTypeDriverPassenger:
		if thread.DriverID != nil {
			if err := service.realtimeGateway.SendToDriver(ctx, *thread.DriverID, EventChatMessage, payload); err != nil {
				return fmt.Errorf("publish chat message to driver: %w", err)
			}
		}
		if thread.PassengerID != nil {
			return service.realtimeGateway.SendToPassenger(ctx, *thread.PassengerID, EventChatMessage, payload)
		}
	}
	return nil
}

func chatMessagesResponse(thread domain.ChatThread, messages []domain.ChatMessage) dto.ChatMessagesResponse {
	response := dto.ChatMessagesResponse{ThreadID: thread.ID, ChatType: thread.Type, Messages: make([]dto.ChatMessageResponse, 0, len(messages))}
	for _, message := range messages {
		response.Messages = append(response.Messages, chatMessageResponse(thread, message))
	}
	return response
}

func chatMessageResponse(thread domain.ChatThread, message domain.ChatMessage) dto.ChatMessageResponse {
	return dto.ChatMessageResponse{
		ID:           message.ID,
		ThreadID:     message.ThreadID,
		OrderID:      message.OrderID,
		ChatType:     thread.Type,
		SenderUserID: message.SenderUserID,
		SenderRole:   message.SenderRole,
		Body:         message.Body,
		CreatedAt:    message.CreatedAt,
	}
}

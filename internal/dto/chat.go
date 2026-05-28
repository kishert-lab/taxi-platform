package dto

import (
	"time"

	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
)

type ChatSendMessageRequest struct {
	Body string `json:"body" binding:"required" example:"Arriving at pickup point"`
}

type ChatMessageResponse struct {
	ID           uuid.UUID       `json:"id" example:"11111111-1111-1111-1111-111111111111"`
	ThreadID     uuid.UUID       `json:"thread_id" example:"22222222-2222-2222-2222-222222222222"`
	OrderID      *uuid.UUID      `json:"order_id,omitempty" example:"33333333-3333-3333-3333-333333333333"`
	ChatType     domain.ChatType `json:"chat_type" example:"driver_passenger"`
	SenderUserID uuid.UUID       `json:"sender_user_id" example:"44444444-4444-4444-4444-444444444444"`
	SenderRole   domain.UserRole `json:"sender_role" example:"driver"`
	Body         string          `json:"body" example:"Arriving"`
	CreatedAt    time.Time       `json:"created_at" example:"2026-05-28T09:00:00Z"`
}

type ChatMessagesResponse struct {
	ThreadID uuid.UUID             `json:"thread_id" example:"22222222-2222-2222-2222-222222222222"`
	ChatType domain.ChatType       `json:"chat_type" example:"driver_passenger"`
	Messages []ChatMessageResponse `json:"messages"`
}

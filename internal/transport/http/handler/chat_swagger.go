package handler

import (
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type ChatMessageSuccessResponse struct {
	Data dto.ChatMessageResponse `json:"data"`
	Meta response.Meta           `json:"meta"`
}

type ChatMessagesSuccessResponse struct {
	Data dto.ChatMessagesResponse `json:"data"`
	Meta response.Meta            `json:"meta"`
}

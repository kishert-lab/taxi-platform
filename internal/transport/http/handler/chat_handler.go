package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type ChatUseCase interface {
	ListOrderMessages(ctx context.Context, actorUserID uuid.UUID, actorRole domain.UserRole, orderID uuid.UUID, chatType domain.ChatType, limit int) (dto.ChatMessagesResponse, error)
	SendOrderMessage(ctx context.Context, actorUserID uuid.UUID, actorRole domain.UserRole, orderID uuid.UUID, chatType domain.ChatType, request dto.ChatSendMessageRequest) (dto.ChatMessageResponse, error)
	ListPassengerSupportMessages(ctx context.Context, passengerID uuid.UUID, limit int) (dto.ChatMessagesResponse, error)
	SendPassengerSupportMessage(ctx context.Context, passengerID uuid.UUID, request dto.ChatSendMessageRequest) (dto.ChatMessageResponse, error)
}

type ChatHandler struct {
	useCase ChatUseCase
}

func NewChatHandler(useCase ChatUseCase) *ChatHandler {
	return &ChatHandler{useCase: useCase}
}

func (handler *ChatHandler) RegisterRoutes(router gin.IRouter, passengerAuthMiddleware gin.HandlerFunc) {
	taxiPark := router.Group("/taxi-park", middleware.RequireRole(domain.UserRoleTaxiPark, domain.UserRoleDispatcher))
	taxiPark.GET("/orders/:id/chat/driver/messages", handler.ListDispatcherDriverMessages)
	taxiPark.POST("/orders/:id/chat/driver/messages", handler.SendDispatcherDriverMessage)

	driver := router.Group("/driver", middleware.RequireRole(domain.UserRoleDriver))
	driver.GET("/orders/:id/chat/dispatcher/messages", handler.ListDriverDispatcherMessages)
	driver.POST("/orders/:id/chat/dispatcher/messages", handler.SendDriverDispatcherMessage)
	driver.GET("/orders/:id/chat/passenger/messages", handler.ListDriverPassengerMessages)
	driver.POST("/orders/:id/chat/passenger/messages", handler.SendDriverPassengerMessage)

	passenger := router.Group("/passenger", passengerAuthMiddleware)
	passenger.GET("/orders/:id/chat/driver/messages", handler.ListPassengerDriverMessages)
	passenger.POST("/orders/:id/chat/driver/messages", handler.SendPassengerDriverMessage)
	passenger.GET("/support/chat/messages", handler.ListPassengerSupportMessages)
	passenger.POST("/support/chat/messages", handler.SendPassengerSupportMessage)
}

// ListDispatcherDriverMessages godoc
// @Summary List dispatcher-driver chat messages
// @Tags chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param limit query int false "Messages limit, default 50, max 100"
// @Success 200 {object} ChatMessagesSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /taxi-park/orders/{id}/chat/driver/messages [get]
func (handler *ChatHandler) ListDispatcherDriverMessages(context *gin.Context) {
	handler.listOrderMessages(context, domain.ChatTypeDispatcherDriver)
}

// SendDispatcherDriverMessage godoc
// @Summary Send dispatcher-driver chat message
// @Tags chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.ChatSendMessageRequest true "Chat message"
// @Success 200 {object} ChatMessageSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /taxi-park/orders/{id}/chat/driver/messages [post]
func (handler *ChatHandler) SendDispatcherDriverMessage(context *gin.Context) {
	handler.sendOrderMessage(context, domain.ChatTypeDispatcherDriver)
}

// ListDriverDispatcherMessages godoc
// @Summary List driver-dispatcher chat messages
// @Tags chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param limit query int false "Messages limit, default 50, max 100"
// @Success 200 {object} ChatMessagesSuccessResponse
// @Router /driver/orders/{id}/chat/dispatcher/messages [get]
func (handler *ChatHandler) ListDriverDispatcherMessages(context *gin.Context) {
	handler.listOrderMessages(context, domain.ChatTypeDispatcherDriver)
}

// SendDriverDispatcherMessage godoc
// @Summary Send driver-dispatcher chat message
// @Tags chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.ChatSendMessageRequest true "Chat message"
// @Success 200 {object} ChatMessageSuccessResponse
// @Router /driver/orders/{id}/chat/dispatcher/messages [post]
func (handler *ChatHandler) SendDriverDispatcherMessage(context *gin.Context) {
	handler.sendOrderMessage(context, domain.ChatTypeDispatcherDriver)
}

// ListDriverPassengerMessages godoc
// @Summary List driver-passenger chat messages
// @Tags chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param limit query int false "Messages limit, default 50, max 100"
// @Success 200 {object} ChatMessagesSuccessResponse
// @Router /driver/orders/{id}/chat/passenger/messages [get]
func (handler *ChatHandler) ListDriverPassengerMessages(context *gin.Context) {
	handler.listOrderMessages(context, domain.ChatTypeDriverPassenger)
}

// SendDriverPassengerMessage godoc
// @Summary Send driver-passenger chat message
// @Tags chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.ChatSendMessageRequest true "Chat message"
// @Success 200 {object} ChatMessageSuccessResponse
// @Router /driver/orders/{id}/chat/passenger/messages [post]
func (handler *ChatHandler) SendDriverPassengerMessage(context *gin.Context) {
	handler.sendOrderMessage(context, domain.ChatTypeDriverPassenger)
}

// ListPassengerDriverMessages godoc
// @Summary List passenger-driver chat messages
// @Tags chat
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param limit query int false "Messages limit, default 50, max 100"
// @Success 200 {object} ChatMessagesSuccessResponse
// @Router /passenger/orders/{id}/chat/driver/messages [get]
func (handler *ChatHandler) ListPassengerDriverMessages(context *gin.Context) {
	handler.listPassengerOrderMessages(context, domain.ChatTypeDriverPassenger)
}

// SendPassengerDriverMessage godoc
// @Summary Send passenger-driver chat message
// @Tags chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.ChatSendMessageRequest true "Chat message"
// @Success 200 {object} ChatMessageSuccessResponse
// @Router /passenger/orders/{id}/chat/driver/messages [post]
func (handler *ChatHandler) SendPassengerDriverMessage(context *gin.Context) {
	handler.sendPassengerOrderMessage(context, domain.ChatTypeDriverPassenger)
}

// ListPassengerSupportMessages godoc
// @Summary List passenger support chat messages
// @Tags chat
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Messages limit, default 50, max 100"
// @Success 200 {object} ChatMessagesSuccessResponse
// @Router /passenger/support/chat/messages [get]
func (handler *ChatHandler) ListPassengerSupportMessages(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}
	result, err := handler.useCase.ListPassengerSupportMessages(context.Request.Context(), passengerID, chatLimitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, result)
}

// SendPassengerSupportMessage godoc
// @Summary Send passenger support chat message
// @Tags chat
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.ChatSendMessageRequest true "Chat message"
// @Success 200 {object} ChatMessageSuccessResponse
// @Router /passenger/support/chat/messages [post]
func (handler *ChatHandler) SendPassengerSupportMessage(context *gin.Context) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return
	}
	var request dto.ChatSendMessageRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid chat message request")
		return
	}
	result, err := handler.useCase.SendPassengerSupportMessage(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, result)
}

func (handler *ChatHandler) listOrderMessages(context *gin.Context, chatType domain.ChatType) {
	actorUserID, actorRole, orderID, ok := chatActorOrderContext(context)
	if !ok {
		return
	}
	result, err := handler.useCase.ListOrderMessages(context.Request.Context(), actorUserID, actorRole, orderID, chatType, chatLimitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, result)
}

func (handler *ChatHandler) sendOrderMessage(context *gin.Context, chatType domain.ChatType) {
	actorUserID, actorRole, orderID, ok := chatActorOrderContext(context)
	if !ok {
		return
	}
	var request dto.ChatSendMessageRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid chat message request")
		return
	}
	result, err := handler.useCase.SendOrderMessage(context.Request.Context(), actorUserID, actorRole, orderID, chatType, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, result)
}

func (handler *ChatHandler) listPassengerOrderMessages(context *gin.Context, chatType domain.ChatType) {
	passengerID, orderID, ok := passengerChatActorOrderContext(context)
	if !ok {
		return
	}
	result, err := handler.useCase.ListOrderMessages(context.Request.Context(), passengerID, domain.UserRolePassenger, orderID, chatType, chatLimitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, result)
}

func (handler *ChatHandler) sendPassengerOrderMessage(context *gin.Context, chatType domain.ChatType) {
	passengerID, orderID, ok := passengerChatActorOrderContext(context)
	if !ok {
		return
	}
	var request dto.ChatSendMessageRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid chat message request")
		return
	}
	result, err := handler.useCase.SendOrderMessage(context.Request.Context(), passengerID, domain.UserRolePassenger, orderID, chatType, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, result)
}

func chatActorOrderContext(context *gin.Context) (uuid.UUID, domain.UserRole, uuid.UUID, bool) {
	userID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return uuid.Nil, "", uuid.Nil, false
	}
	role, ok := userRoleFromContext(context)
	if !ok {
		failUnauthorized(context, "User role is missing")
		return uuid.Nil, "", uuid.Nil, false
	}
	orderID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid order id")
		return uuid.Nil, "", uuid.Nil, false
	}
	return userID, role, orderID, true
}

func passengerChatActorOrderContext(context *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	passengerID, ok := middleware.PassengerIDFromContext(context)
	if !ok {
		failUnauthorized(context, "Passenger id is missing")
		return uuid.Nil, uuid.Nil, false
	}
	orderID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid order id")
		return uuid.Nil, uuid.Nil, false
	}
	return passengerID, orderID, true
}

func userRoleFromContext(context *gin.Context) (domain.UserRole, bool) {
	value, exists := context.Get(middleware.UserRoleContextKey)
	if !exists {
		return "", false
	}
	switch role := value.(type) {
	case domain.UserRole:
		return role, role.Validate() == nil
	case string:
		userRole := domain.UserRole(role)
		return userRole, userRole.Validate() == nil
	default:
		return "", false
	}
}

func chatLimitFromQuery(context *gin.Context) int {
	value := context.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 50
	}
	if limit > 100 {
		return 100
	}
	return limit
}

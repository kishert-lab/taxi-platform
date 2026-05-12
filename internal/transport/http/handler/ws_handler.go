package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	wsmsg "github.com/kishert-lab/taxi-platform/internal/ws"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type WebSocketAuthUseCase interface {
	AuthenticateWebSocket(ctx context.Context, token string) (uuid.UUID, domain.UserRole, error)
}

type WebSocketHandler struct {
	authUseCase WebSocketAuthUseCase
	upgrader    websocket.Upgrader
}

func NewWebSocketHandler(authUseCase WebSocketAuthUseCase) *WebSocketHandler {
	return &WebSocketHandler{
		authUseCase: authUseCase,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}
}

func (handler *WebSocketHandler) RegisterRoutes(router gin.IRouter) {
	router.GET("/ws", handler.Connect)
}

// Connect godoc
// @Summary Connect mobile WebSocket
// @Description Mobile realtime endpoint. JWT can be passed in Authorization header or token query parameter. After reconnect the server emits sync.required and the client must call the current order REST endpoint.
// @Tags websocket
// @Produce json
// @Security BearerAuth
// @Param token query string false "JWT token fallback for mobile clients"
// @Success 101 {string} string "Switching Protocols"
// @Failure 401 {object} response.Error
// @Router /ws [get]
func (handler *WebSocketHandler) Connect(context *gin.Context) {
	token := websocketToken(context)
	if token == "" {
		failUnauthorized(context, "WebSocket token is missing")
		return
	}

	var userID uuid.UUID
	var role domain.UserRole
	if handler.authUseCase != nil {
		authenticatedUserID, authenticatedRole, err := handler.authUseCase.AuthenticateWebSocket(context.Request.Context(), token)
		if err != nil {
			failUnauthorized(context, "WebSocket token is invalid")
			return
		}
		userID = authenticatedUserID
		role = authenticatedRole
	}

	connection, err := handler.upgrader.Upgrade(context.Writer, context.Request, nil)
	if err != nil {
		response.Fail(context, http.StatusInternalServerError, response.CodeInternalError, "WebSocket upgrade failed", nil)
		return
	}
	defer connection.Close()

	requestID, _ := uuid.Parse(context.GetString(response.RequestIDContextKey))
	if requestID == uuid.Nil {
		requestID = uuid.New()
	}

	message := wsmsg.NewMessage(wsmsg.EventSyncRequired, requestID, map[string]any{
		"user_id": userID,
		"role":    role,
		"reason":  "reconnect",
	})
	_ = connection.WriteJSON(message)
}

func websocketToken(context *gin.Context) string {
	authorizationHeader := context.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(authorizationHeader), "bearer ") {
		return strings.TrimSpace(authorizationHeader[len("Bearer "):])
	}

	return strings.TrimSpace(context.Query("token"))
}

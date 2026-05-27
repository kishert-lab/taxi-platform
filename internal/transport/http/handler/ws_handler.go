package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	wsmsg "github.com/kishert-lab/taxi-platform/internal/ws"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

const (
	websocketPongWait     = 60 * time.Second
	websocketPingPeriod   = 45 * time.Second
	websocketWriteTimeout = 10 * time.Second
	websocketReadLimit    = 4096
)

type WebSocketAuthUseCase interface {
	AuthenticateWebSocket(ctx context.Context, token string) (uuid.UUID, domain.UserRole, error)
}

type WebSocketHandler struct {
	authUseCase WebSocketAuthUseCase
	upgrader    websocket.Upgrader
}

func NewWebSocketHandler(authUseCase WebSocketAuthUseCase, allowedOrigins []string) *WebSocketHandler {
	return &WebSocketHandler{
		authUseCase: authUseCase,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     websocketOriginChecker(allowedOrigins),
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
		return
	}
	defer connection.Close()
	connection.SetReadLimit(websocketReadLimit)
	_ = connection.SetReadDeadline(time.Now().Add(websocketPongWait))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(websocketPongWait))
	})

	requestID, _ := uuid.Parse(context.GetString(response.RequestIDContextKey))
	if requestID == uuid.Nil {
		requestID = uuid.New()
	}

	message := wsmsg.NewMessage(wsmsg.EventSyncRequired, requestID, map[string]any{
		"user_id": userID,
		"role":    role,
		"reason":  "reconnect",
	})
	if err := writeWebSocketJSON(connection, message); err != nil {
		return
	}

	handler.keepConnectionAlive(context.Request.Context(), connection)
}

func (handler *WebSocketHandler) keepConnectionAlive(ctx context.Context, connection *websocket.Conn) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := connection.NextReader(); err != nil {
				return
			}
		}
	}()

	ticker := time.NewTicker(websocketPingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := writeWebSocketPing(connection); err != nil {
				return
			}
		}
	}
}

func writeWebSocketJSON(connection *websocket.Conn, message any) error {
	if err := connection.SetWriteDeadline(time.Now().Add(websocketWriteTimeout)); err != nil {
		return err
	}
	return connection.WriteJSON(message)
}

func writeWebSocketPing(connection *websocket.Conn) error {
	if err := connection.SetWriteDeadline(time.Now().Add(websocketWriteTimeout)); err != nil {
		return err
	}
	return connection.WriteMessage(websocket.PingMessage, nil)
}

func websocketToken(context *gin.Context) string {
	authorizationHeader := context.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(authorizationHeader), "bearer ") {
		return strings.TrimSpace(authorizationHeader[len("Bearer "):])
	}

	return strings.TrimSpace(context.Query("token"))
}

func websocketOriginChecker(allowedOrigins []string) func(request *http.Request) bool {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	allowAny := false
	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin == "" {
			continue
		}
		if origin == "*" {
			allowAny = true
			continue
		}
		allowed[origin] = struct{}{}
	}

	return func(request *http.Request) bool {
		origin := strings.TrimSpace(request.Header.Get("Origin"))
		if origin == "" || allowAny {
			return true
		}
		_, ok := allowed[origin]
		return ok
	}
}

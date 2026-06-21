package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	goredis "github.com/redis/go-redis/v9"

	auditapp "github.com/kishert-lab/taxi-platform/internal/audit"
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
	auditLogger WebSocketAuditLogger
	redisClient *goredis.Client
	upgrader    websocket.Upgrader
}

type WebSocketAuditLogger interface {
	LogWebSocket(ctx context.Context, command auditapp.WebSocketRequestLogCommand)
}

func NewWebSocketHandler(authUseCase WebSocketAuthUseCase, auditLogger WebSocketAuditLogger, allowedOrigins []string, redisClients ...*goredis.Client) *WebSocketHandler {
	var redisClient *goredis.Client
	if len(redisClients) > 0 {
		redisClient = redisClients[0]
	}
	return &WebSocketHandler{
		authUseCase: authUseCase,
		auditLogger: auditLogger,
		redisClient: redisClient,
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
	startedAt := time.Now()
	requestID := context.GetString(response.RequestIDContextKey)
	path := context.Request.URL.Path
	rawQuery := context.Request.URL.RawQuery
	clientIP := context.ClientIP()
	userAgent := context.Request.UserAgent()

	token := websocketToken(context)
	if token == "" {
		handler.logWebSocketEvent(context.Request.Context(), auditapp.WebSocketRequestLogCommand{
			RequestID:    requestID,
			EventType:    "connection.rejected",
			Path:         path,
			RawQuery:     rawQuery,
			StatusCode:   http.StatusUnauthorized,
			Duration:     time.Since(startedAt),
			ClientIP:     clientIP,
			UserAgent:    userAgent,
			ErrorMessage: "websocket token is missing",
		})
		failUnauthorized(context, "WebSocket token is missing")
		return
	}

	var userID uuid.UUID
	var role domain.UserRole
	if handler.authUseCase != nil {
		authenticatedUserID, authenticatedRole, err := handler.authUseCase.AuthenticateWebSocket(context.Request.Context(), token)
		if err != nil {
			handler.logWebSocketEvent(context.Request.Context(), auditapp.WebSocketRequestLogCommand{
				RequestID:    requestID,
				EventType:    "connection.rejected",
				Path:         path,
				RawQuery:     rawQuery,
				StatusCode:   http.StatusUnauthorized,
				Duration:     time.Since(startedAt),
				ClientIP:     clientIP,
				UserAgent:    userAgent,
				ErrorMessage: "websocket token is invalid",
			})
			failUnauthorized(context, "WebSocket token is invalid")
			return
		}
		userID = authenticatedUserID
		role = authenticatedRole
	}

	connection, err := handler.upgrader.Upgrade(context.Writer, context.Request, nil)
	if err != nil {
		handler.logWebSocketEvent(context.Request.Context(), auditapp.WebSocketRequestLogCommand{
			RequestID:    requestID,
			EventType:    "connection.upgrade_failed",
			Path:         path,
			RawQuery:     rawQuery,
			StatusCode:   http.StatusBadRequest,
			Duration:     time.Since(startedAt),
			ClientIP:     clientIP,
			UserAgent:    userAgent,
			ActorUserID:  userID,
			ActorRole:    role,
			ErrorMessage: err.Error(),
		})
		return
	}
	defer connection.Close()
	connection.SetReadLimit(websocketReadLimit)
	_ = connection.SetReadDeadline(time.Now().Add(websocketPongWait))
	connection.SetPongHandler(func(string) error {
		return connection.SetReadDeadline(time.Now().Add(websocketPongWait))
	})

	messageRequestID, _ := uuid.Parse(requestID)
	if messageRequestID == uuid.Nil {
		messageRequestID = uuid.New()
	}

	message := wsmsg.NewMessage(wsmsg.EventSyncRequired, messageRequestID, map[string]any{
		"user_id": userID,
		"role":    role,
		"reason":  "reconnect",
	})
	if err := writeWebSocketJSON(connection, message); err != nil {
		handler.logWebSocketEvent(context.Request.Context(), auditapp.WebSocketRequestLogCommand{
			RequestID:    requestID,
			EventType:    "connection.bootstrap_failed",
			Path:         path,
			RawQuery:     rawQuery,
			StatusCode:   http.StatusSwitchingProtocols,
			Duration:     time.Since(startedAt),
			ClientIP:     clientIP,
			UserAgent:    userAgent,
			ActorUserID:  userID,
			ActorRole:    role,
			ErrorMessage: err.Error(),
		})
		return
	}

	handler.logWebSocketEvent(context.Request.Context(), auditapp.WebSocketRequestLogCommand{
		RequestID:   requestID,
		EventType:   "connection.opened",
		Path:        path,
		RawQuery:    rawQuery,
		StatusCode:  http.StatusSwitchingProtocols,
		Duration:    time.Since(startedAt),
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		ActorUserID: userID,
		ActorRole:   role,
	})

	closeReason := handler.keepConnectionAlive(context.Request.Context(), connection, userID)
	handler.logWebSocketEvent(context.Request.Context(), auditapp.WebSocketRequestLogCommand{
		RequestID:   requestID,
		EventType:   "connection.closed",
		Path:        path,
		RawQuery:    rawQuery,
		StatusCode:  http.StatusSwitchingProtocols,
		Duration:    time.Since(startedAt),
		ClientIP:    clientIP,
		UserAgent:   userAgent,
		ActorUserID: userID,
		ActorRole:   role,
		CloseReason: closeReason,
	})
}

func (handler *WebSocketHandler) keepConnectionAlive(ctx context.Context, connection *websocket.Conn, userID uuid.UUID) string {
	done := make(chan string, 1)
	go func() {
		for {
			if _, _, err := connection.NextReader(); err != nil {
				done <- err.Error()
				return
			}
		}
	}()

	ticker := time.NewTicker(websocketPingPeriod)
	defer ticker.Stop()

	var messages <-chan *goredis.Message
	var subscription *goredis.PubSub
	if handler.redisClient != nil && userID != uuid.Nil {
		subscription = handler.redisClient.Subscribe(ctx, webSocketUserChannel(userID))
		defer subscription.Close()
		messages = subscription.Channel()
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err().Error()
		case reason := <-done:
			return reason
		case message, ok := <-messages:
			if !ok || message == nil {
				return "redis subscription closed"
			}
			if err := writeWebSocketText(connection, message.Payload); err != nil {
				return err.Error()
			}
		case <-ticker.C:
			if err := writeWebSocketPing(connection); err != nil {
				return err.Error()
			}
		}
	}
}

func (handler *WebSocketHandler) logWebSocketEvent(ctx context.Context, command auditapp.WebSocketRequestLogCommand) {
	if handler.auditLogger == nil {
		return
	}
	handler.auditLogger.LogWebSocket(ctx, command)
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

func writeWebSocketText(connection *websocket.Conn, payload string) error {
	if err := connection.SetWriteDeadline(time.Now().Add(websocketWriteTimeout)); err != nil {
		return err
	}
	return connection.WriteMessage(websocket.TextMessage, []byte(payload))
}

func websocketToken(context *gin.Context) string {
	authorizationHeader := context.GetHeader("Authorization")
	if strings.HasPrefix(strings.ToLower(authorizationHeader), "bearer ") {
		return strings.TrimSpace(authorizationHeader[len("Bearer "):])
	}

	return strings.TrimSpace(context.Query("token"))
}

func webSocketUserChannel(userID uuid.UUID) string {
	return fmt.Sprintf("ws:user:%s", userID)
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

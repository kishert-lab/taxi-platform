package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	wsmsg "github.com/kishert-lab/taxi-platform/internal/ws"
)

func TestWebSocketAcceptsQueryTokenFromAllowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api/v1")
	NewWebSocketHandler(
		fakeWebSocketAuth{userID: uuid.New(), role: domain.UserRoleTaxiPark},
		nil,
		[]string{"http://localhost:5174"},
	).RegisterRoutes(api)

	server := httptest.NewServer(router)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=valid-token"
	headers := http.Header{}
	headers.Set("Origin", "http://localhost:5174")

	connection, response, err := websocket.DefaultDialer.Dial(url, headers)
	if err != nil {
		if response != nil {
			t.Fatalf("dial websocket: status=%d err=%v", response.StatusCode, err)
		}
		t.Fatalf("dial websocket: %v", err)
	}
	defer connection.Close()

	var message wsmsg.Message
	if err := connection.ReadJSON(&message); err != nil {
		t.Fatalf("read websocket sync message: %v", err)
	}
	if message.Event != wsmsg.EventSyncRequired {
		t.Fatalf("expected sync.required event, got %s", message.Event)
	}
}

func TestWebSocketRejectsDisallowedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	api := router.Group("/api/v1")
	NewWebSocketHandler(
		fakeWebSocketAuth{userID: uuid.New(), role: domain.UserRoleTaxiPark},
		nil,
		[]string{"http://localhost:5174"},
	).RegisterRoutes(api)

	server := httptest.NewServer(router)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http") + "/api/v1/ws?token=valid-token"
	headers := http.Header{}
	headers.Set("Origin", "http://evil.example")

	connection, response, err := websocket.DefaultDialer.Dial(url, headers)
	if err == nil {
		connection.Close()
		t.Fatal("expected websocket dial to fail for disallowed origin")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403 for disallowed origin, response=%v err=%v", response, err)
	}
}

type fakeWebSocketAuth struct {
	userID uuid.UUID
	role   domain.UserRole
}

func (auth fakeWebSocketAuth) AuthenticateWebSocket(context.Context, string) (uuid.UUID, domain.UserRole, error) {
	return auth.userID, auth.role, nil
}

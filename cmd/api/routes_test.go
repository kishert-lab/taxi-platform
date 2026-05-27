package main

import (
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/configs"
	"github.com/kishert-lab/taxi-platform/internal/service"
	"github.com/kishert-lab/taxi-platform/internal/transport/http/handler"
)

func TestMobileAndFinanceRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	unavailableUseCase := service.NewUnavailableUseCase()
	routes := applicationRoutes{
		auth:       handler.NewAuthHandler(unavailableUseCase),
		mobileAuth: handler.NewMobileAuthHandler(unavailableUseCase),
		order:      handler.NewOrderHandler(unavailableUseCase),
		passenger:  handler.NewPassengerMobileHandler(unavailableUseCase, unavailableUseCase),
		driver:     handler.NewDriverMobileHandler(unavailableUseCase),
		finance:    handler.NewFinanceHandler(unavailableUseCase),
		taxiPark:   handler.NewTaxiParkSettingsHandler(unavailableUseCase),
		legal:      handler.NewLegalHandler(unavailableUseCase),
		websocket:  handler.NewWebSocketHandler(unavailableUseCase, []string{"*"}),
	}
	router := buildRouter(testConfig(), zap.NewNop(), routes)

	registered := make(map[string]struct{})
	for _, route := range router.Routes() {
		registered[route.Method+" "+route.Path] = struct{}{}
	}

	expectedRoutes := []string{
		http.MethodPost + " /api/v1/auth/register",
		http.MethodPost + " /api/v1/auth/confirm-phone",
		http.MethodPost + " /api/v1/auth/login",
		http.MethodPost + " /api/v1/auth/email/send-code",
		http.MethodPost + " /api/v1/auth/email/verify",
		http.MethodPost + " /api/v1/auth/verify-code",
		http.MethodPost + " /api/v1/auth/refresh",
		http.MethodPost + " /api/v1/auth/logout",
		http.MethodPost + " /api/v1/passenger/profile",
		http.MethodGet + " /api/v1/passenger/profile",
		http.MethodPatch + " /api/v1/passenger/profile",
		http.MethodPost + " /api/v1/passenger/profile/photo",
		http.MethodPost + " /api/v1/passenger/orders/estimate",
		http.MethodPost + " /api/v1/passenger/orders",
		http.MethodGet + " /api/v1/passenger/orders/current",
		http.MethodGet + " /api/v1/passenger/orders/history",
		http.MethodGet + " /api/v1/passenger/orders/:id",
		http.MethodPost + " /api/v1/passenger/orders/:id/cancel",
		http.MethodPost + " /api/v1/passenger/orders/:id/rate",
		http.MethodGet + " /api/v1/driver/profile",
		http.MethodPatch + " /api/v1/driver/profile",
		http.MethodPost + " /api/v1/driver/profile/photo",
		http.MethodPost + " /api/v1/driver/online",
		http.MethodPost + " /api/v1/driver/offline",
		http.MethodPost + " /api/v1/driver/location",
		http.MethodPost + " /api/v1/driver/location/batch",
		http.MethodGet + " /api/v1/driver/orders/current",
		http.MethodGet + " /api/v1/driver/orders/history",
		http.MethodPost + " /api/v1/driver/orders/:id/accept",
		http.MethodPost + " /api/v1/driver/orders/:id/reject",
		http.MethodPost + " /api/v1/driver/orders/:id/arrived",
		http.MethodPost + " /api/v1/driver/orders/:id/start",
		http.MethodPost + " /api/v1/driver/orders/:id/complete",
		http.MethodPost + " /api/v1/driver/orders/:id/rate-passenger",
		http.MethodGet + " /api/v1/driver/balance",
		http.MethodGet + " /api/v1/driver/transactions",
		http.MethodGet + " /api/v1/taxi-park/balance",
		http.MethodGet + " /api/v1/taxi-park/drivers",
		http.MethodPost + " /api/v1/taxi-park/drivers",
		http.MethodPatch + " /api/v1/taxi-park/drivers/:id",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/password",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/block",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/unblock",
		http.MethodGet + " /api/v1/taxi-park/drivers/:id/documents",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/cars/:car_id",
		http.MethodDelete + " /api/v1/taxi-park/drivers/:id/cars/:car_id",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/cars/:car_id/attach",
		http.MethodDelete + " /api/v1/taxi-park/drivers/:id/cars/:car_id/detach",
		http.MethodDelete + " /api/v1/taxi-park/drivers/:id",
		http.MethodGet + " /api/v1/taxi-park/cars",
		http.MethodPost + " /api/v1/taxi-park/cars",
		http.MethodPatch + " /api/v1/taxi-park/cars/:id",
		http.MethodDelete + " /api/v1/taxi-park/cars/:id",
		http.MethodPost + " /api/v1/taxi-park/cars/:id/verify",
		http.MethodGet + " /api/v1/taxi-park/cars/:id/documents",
		http.MethodGet + " /api/v1/taxi-park/orders",
		http.MethodPost + " /api/v1/taxi-park/orders",
		http.MethodGet + " /api/v1/taxi-park/transactions",
		http.MethodGet + " /api/v1/taxi-park/settings",
		http.MethodPatch + " /api/v1/taxi-park/settings",
		http.MethodGet + " /api/v1/taxi-park/tariffs",
		http.MethodPost + " /api/v1/taxi-park/tariffs",
		http.MethodPatch + " /api/v1/taxi-park/tariffs/:id",
		http.MethodGet + " /api/v1/admin/finance/overview",
		http.MethodGet + " /api/v1/public/legal/privacy-policy",
		http.MethodGet + " /api/v1/public/legal/terms",
		http.MethodGet + " /api/v1/public/legal/consent",
		http.MethodGet + " /api/v1/admin/legal/documents",
		http.MethodPost + " /api/v1/admin/legal/documents",
		http.MethodPost + " /api/v1/admin/legal/documents/:id/activate",
		http.MethodPost + " /api/v1/admin/legal/documents/:id/deactivate",
		http.MethodGet + " /api/v1/ws",
		http.MethodGet + " /api/v1/orders/current",
	}

	for _, expectedRoute := range expectedRoutes {
		if _, exists := registered[expectedRoute]; !exists {
			t.Fatalf("expected route to be registered: %s", expectedRoute)
		}
	}
}

func testConfig() *configs.Config {
	return &configs.Config{
		App: configs.AppConfig{Name: "taxi-platform"},
		Server: configs.ServerConfig{
			Port: 8080,
		},
		HTTP: configs.HTTPConfig{
			CORS: configs.CORSConfig{AllowedOrigins: []string{"*"}},
		},
	}
}

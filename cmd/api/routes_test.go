package main

import (
	"context"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/configs"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/service"
	"github.com/kishert-lab/taxi-platform/internal/transport/http/handler"
)

func TestMobileAndFinanceRoutesAreRegistered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	unavailableUseCase := service.NewUnavailableUseCase()
	routes := applicationRoutes{
		auth:             handler.NewAuthHandler(unavailableUseCase),
		mobileAuth:       handler.NewMobileAuthHandler(unavailableUseCase),
		passengerAuth:    handler.NewPassengerAuthHandler(fakePassengerAuthUseCase{}),
		passengerMe:      handler.NewPassengerMeHandler(fakePassengerMeUseCase{}),
		passengerAddress: handler.NewPassengerAddressHandler(fakePassengerAddressUseCase{}),
		passengerPush:    handler.NewPassengerPushHandler(fakePassengerPushUseCase{}),
		passengerAuthMiddleware: func(context *gin.Context) {
			context.Next()
		},
		order:     handler.NewOrderHandler(unavailableUseCase),
		passenger: handler.NewPassengerMobileHandler(unavailableUseCase, unavailableUseCase),
		driver:    handler.NewDriverMobileHandler(unavailableUseCase),
		finance:   handler.NewFinanceHandler(unavailableUseCase),
		taxiPark:  handler.NewTaxiParkSettingsHandler(unavailableUseCase),
		legal:     handler.NewLegalHandler(unavailableUseCase),
		chat:      handler.NewChatHandler(unavailableUseCase),
		websocket: handler.NewWebSocketHandler(unavailableUseCase, nil, []string{"*"}),
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
		http.MethodPost + " /api/v1/passenger/auth/request-code",
		http.MethodPost + " /api/v1/passenger/auth/confirm-code",
		http.MethodPost + " /api/v1/passenger/auth/refresh",
		http.MethodPost + " /api/v1/passenger/auth/logout",
		http.MethodGet + " /api/v1/passenger/me",
		http.MethodPatch + " /api/v1/passenger/me",
		http.MethodPost + " /api/v1/passenger/push-tokens",
		http.MethodPost + " /api/v1/passenger/push/token",
		http.MethodPost + " /api/v1/passenger/profile",
		http.MethodGet + " /api/v1/passenger/profile",
		http.MethodPatch + " /api/v1/passenger/profile",
		http.MethodPost + " /api/v1/passenger/profile/photo",
		http.MethodGet + " /api/v1/passenger/address/search",
		http.MethodPost + " /api/v1/passenger/orders/estimate",
		http.MethodPost + " /api/v1/passenger/orders",
		http.MethodGet + " /api/v1/passenger/orders/current",
		http.MethodGet + " /api/v1/passenger/orders/history",
		http.MethodGet + " /api/v1/passenger/orders/:id",
		http.MethodPost + " /api/v1/passenger/orders/:id/cancel",
		http.MethodPost + " /api/v1/passenger/orders/:id/rate",
		http.MethodGet + " /api/v1/passenger/orders/:id/chat/driver/messages",
		http.MethodPost + " /api/v1/passenger/orders/:id/chat/driver/messages",
		http.MethodGet + " /api/v1/passenger/support/chat/messages",
		http.MethodPost + " /api/v1/passenger/support/chat/messages",
		http.MethodGet + " /api/v1/driver/profile",
		http.MethodGet + " /api/v1/driver/cars",
		http.MethodPatch + " /api/v1/driver/profile",
		http.MethodPost + " /api/v1/driver/profile/photo",
		http.MethodPost + " /api/v1/driver/online",
		http.MethodPost + " /api/v1/driver/offline",
		http.MethodPost + " /api/v1/driver/location",
		http.MethodPost + " /api/v1/driver/location/batch",
		http.MethodGet + " /api/v1/driver/orders/current",
		http.MethodGet + " /api/v1/driver/orders/history",
		http.MethodGet + " /api/v1/driver/orders/:id",
		http.MethodPost + " /api/v1/driver/orders/:id/route/batch",
		http.MethodPost + " /api/v1/driver/orders/:id/accept",
		http.MethodPost + " /api/v1/driver/orders/:id/reject",
		http.MethodPost + " /api/v1/driver/orders/:id/arrived",
		http.MethodPost + " /api/v1/driver/orders/:id/start",
		http.MethodPost + " /api/v1/driver/orders/:id/complete",
		http.MethodPost + " /api/v1/driver/orders/:id/rate-passenger",
		http.MethodGet + " /api/v1/driver/orders/:id/chat/dispatcher/messages",
		http.MethodPost + " /api/v1/driver/orders/:id/chat/dispatcher/messages",
		http.MethodGet + " /api/v1/driver/orders/:id/chat/passenger/messages",
		http.MethodPost + " /api/v1/driver/orders/:id/chat/passenger/messages",
		http.MethodGet + " /api/v1/driver/balance",
		http.MethodGet + " /api/v1/driver/transactions",
		http.MethodGet + " /api/v1/taxi-park/balance",
		http.MethodGet + " /api/v1/taxi-park/drivers",
		http.MethodPost + " /api/v1/taxi-park/drivers",
		http.MethodGet + " /api/v1/taxi-park/drivers/locations",
		http.MethodPatch + " /api/v1/taxi-park/drivers/:id",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/password",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/block",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/unblock",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/archive",
		http.MethodGet + " /api/v1/taxi-park/drivers/:id/documents",
		http.MethodGet + " /api/v1/taxi-park/drivers/:id/cars",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/cars/:car_id",
		http.MethodPost + " /api/v1/taxi-park/drivers/:id/cars/:car_id/assign",
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
		http.MethodGet + " /api/v1/taxi-park/orders/scheduled",
		http.MethodPost + " /api/v1/taxi-park/orders/scheduled",
		http.MethodGet + " /api/v1/taxi-park/orders/scheduled/:id",
		http.MethodPatch + " /api/v1/taxi-park/orders/scheduled/:id",
		http.MethodPost + " /api/v1/taxi-park/orders/scheduled/:id/cancel",
		http.MethodPost + " /api/v1/taxi-park/orders/scheduled/:id/assign-driver",
		http.MethodGet + " /api/v1/taxi-park/orders/:id/chat/driver/messages",
		http.MethodPost + " /api/v1/taxi-park/orders/:id/chat/driver/messages",
		http.MethodGet + " /api/v1/taxi-park/transactions",
		http.MethodGet + " /api/v1/taxi-park/settings",
		http.MethodPatch + " /api/v1/taxi-park/settings",
		http.MethodGet + " /api/v1/taxi-park/dispatchers",
		http.MethodPost + " /api/v1/taxi-park/dispatchers",
		http.MethodPatch + " /api/v1/taxi-park/dispatchers/:id",
		http.MethodPost + " /api/v1/taxi-park/dispatchers/:id/block",
		http.MethodPost + " /api/v1/taxi-park/dispatchers/:id/unblock",
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

type fakePassengerAuthUseCase struct{}

func (fakePassengerAuthUseCase) RequestCode(context.Context, dto.PassengerAuthRequestCodeRequest) (dto.PassengerAuthRequestCodeResponse, error) {
	return dto.PassengerAuthRequestCodeResponse{}, nil
}

func (fakePassengerAuthUseCase) ConfirmCode(context.Context, dto.PassengerAuthConfirmCodeRequest) (dto.PassengerAuthTokenResponse, error) {
	return dto.PassengerAuthTokenResponse{}, nil
}

func (fakePassengerAuthUseCase) Refresh(context.Context, dto.RefreshTokenRequest) (dto.PassengerAuthRefreshResponse, error) {
	return dto.PassengerAuthRefreshResponse{}, nil
}

func (fakePassengerAuthUseCase) Logout(context.Context, dto.LogoutRequest) error {
	return nil
}

type fakePassengerMeUseCase struct{}

func (fakePassengerMeUseCase) GetMe(context.Context, uuid.UUID) (dto.PassengerMeResponse, error) {
	return dto.PassengerMeResponse{}, nil
}

func (fakePassengerMeUseCase) UpdateMe(context.Context, uuid.UUID, dto.PassengerMePatchRequest) (dto.PassengerMeResponse, error) {
	return dto.PassengerMeResponse{}, nil
}

type fakePassengerPushUseCase struct{}

func (fakePassengerPushUseCase) RegisterToken(context.Context, uuid.UUID, dto.PassengerPushTokenRequest) (dto.PassengerPushTokenResponse, error) {
	return dto.PassengerPushTokenResponse{}, nil
}

type fakePassengerAddressUseCase struct{}

func (fakePassengerAddressUseCase) SearchPassengerAddresses(context.Context, uuid.UUID, string, *uuid.UUID, *float64, *float64, int) ([]geodomain.SearchResult, error) {
	return nil, nil
}

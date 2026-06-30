package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/configs"
	authapp "github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	passengerapp "github.com/kishert-lab/taxi-platform/internal/passenger"
	"github.com/kishert-lab/taxi-platform/internal/service"
	"github.com/kishert-lab/taxi-platform/internal/transport/http/handler"
)

func TestPassengerOrdersCurrentAcceptsPassengerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	passengerID := uuid.New()
	router, passengerTokenManager, _ := passengerOrdersAuthRouter(t, fakePassengerOrdersUseCase{
		currentResponse: dto.PassengerOrderResponse{
			OrderID: uuid.New(),
			Status:  domain.OrderStatusCreated,
		},
	}, fakePassengerLookupRepositoryForOrders{
		passenger: domain.Passenger{ID: passengerID, IsActive: true},
	})

	accessToken, err := passengerTokenManager.GeneratePassengerAccessToken(passengerID, time.Now().UTC())
	if err != nil {
		t.Fatalf("GeneratePassengerAccessToken returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/passenger/orders/current", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPassengerOrdersHistoryRejectsGenericUserToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, _, userTokenManager := passengerOrdersAuthRouter(t, fakePassengerOrdersUseCase{}, fakePassengerLookupRepositoryForOrders{
		passenger: domain.Passenger{ID: uuid.New(), IsActive: true},
	})

	userAccessToken, _, _, err := userTokenManager.IssueTokenPair(uuid.New(), domain.UserRoleDriver, time.Now().UTC())
	if err != nil {
		t.Fatalf("IssueTokenPair returned error: %v", err)
	}

	request := httptest.NewRequest(http.MethodGet, "/api/v1/passenger/orders/history", nil)
	request.Header.Set("Authorization", "Bearer "+userAccessToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPassengerOrdersCurrentRejectsInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router, _, _ := passengerOrdersAuthRouter(t, fakePassengerOrdersUseCase{}, fakePassengerLookupRepositoryForOrders{
		passenger: domain.Passenger{ID: uuid.New(), IsActive: true},
	})

	request := httptest.NewRequest(http.MethodGet, "/api/v1/passenger/orders/current", nil)
	request.Header.Set("Authorization", "Bearer invalid-token")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func passengerOrdersAuthRouter(
	t *testing.T,
	orderUseCase fakePassengerOrdersUseCase,
	passengerRepository fakePassengerLookupRepositoryForOrders,
) (*gin.Engine, *passengerapp.TokenManager, *authapp.TokenManager) {
	t.Helper()

	unavailableUseCase := service.NewUnavailableUseCase()
	config := testAuthConfig()
	passengerTokenManager := passengerapp.NewTokenManager(passengerapp.TokenManagerConfig{
		AccessSecret:  config.JWT.AccessSecret,
		RefreshSecret: config.JWT.RefreshSecret,
		Issuer:        config.JWT.Issuer,
		AccessTTL:     config.JWT.AccessTTL,
		RefreshTTL:    config.JWT.RefreshTTL,
	})
	userTokenManager := authapp.NewTokenManager(authapp.TokenManagerConfig{
		AccessSecret:  config.JWT.AccessSecret,
		RefreshSecret: config.JWT.RefreshSecret,
		Issuer:        config.JWT.Issuer,
		AccessTTL:     config.JWT.AccessTTL,
		RefreshTTL:    config.JWT.RefreshTTL,
	})

	routes := applicationRoutes{
		auth:             handler.NewAuthHandler(unavailableUseCase),
		mobileAuth:       handler.NewMobileAuthHandler(unavailableUseCase),
		passengerAuth:    handler.NewPassengerAuthHandler(fakePassengerAuthUseCase{}),
		passengerMe:      handler.NewPassengerMeHandler(fakePassengerMeUseCase{}),
		passengerAddress: handler.NewPassengerAddressHandler(fakePassengerAddressUseCase{}),
		passengerOrders:  handler.NewPassengerOrdersHandler(orderUseCase),
		passengerPush:    handler.NewPassengerPushHandler(fakePassengerPushUseCase{}),
		passengerAuthMiddleware: middleware.AuthenticatePassengerAccessToken(
			passengerTokenManager,
			passengerRepository,
		),
		order:     handler.NewOrderHandler(unavailableUseCase),
		passenger: handler.NewPassengerMobileHandler(unavailableUseCase, unavailableUseCase),
		driver:    handler.NewDriverMobileHandler(unavailableUseCase),
		finance:   handler.NewFinanceHandler(unavailableUseCase),
		taxiPark:  handler.NewTaxiParkSettingsHandler(unavailableUseCase),
		legal:     handler.NewLegalHandler(unavailableUseCase),
		chat:      handler.NewChatHandler(unavailableUseCase),
		websocket: handler.NewWebSocketHandler(unavailableUseCase, nil, []string{"*"}),
	}

	return buildRouter(config, zap.NewNop(), routes), passengerTokenManager, userTokenManager
}

func testAuthConfig() *configs.Config {
	config := testConfig()
	config.JWT.AccessSecret = "test-access-secret-with-more-than-32-chars"
	config.JWT.RefreshSecret = "test-refresh-secret-with-more-than-32-chars"
	config.JWT.Issuer = "taxi-platform-test"
	config.JWT.AccessTTL = time.Hour
	config.JWT.RefreshTTL = time.Hour
	return config
}

type fakePassengerOrdersUseCase struct {
	currentResponse dto.PassengerOrderResponse
	historyResponse dto.OrderHistoryResponse
}

func (useCase fakePassengerOrdersUseCase) EstimatePassengerOrder(context.Context, uuid.UUID, dto.OrderEstimateRequest) (dto.OrderEstimateResponse, error) {
	return dto.OrderEstimateResponse{}, nil
}

func (useCase fakePassengerOrdersUseCase) CreatePassengerOrder(context.Context, uuid.UUID, dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, nil
}

func (useCase fakePassengerOrdersUseCase) GetCurrentPassengerOrder(context.Context, uuid.UUID) (dto.PassengerOrderResponse, error) {
	return useCase.currentResponse, nil
}

func (useCase fakePassengerOrdersUseCase) ListPassengerOrderHistory(context.Context, uuid.UUID) (dto.OrderHistoryResponse, error) {
	return useCase.historyResponse, nil
}

func (useCase fakePassengerOrdersUseCase) GetPassengerOrder(context.Context, uuid.UUID, uuid.UUID) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, nil
}

func (useCase fakePassengerOrdersUseCase) CancelPassengerOrder(context.Context, uuid.UUID, uuid.UUID, dto.CancelOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, nil
}

func (useCase fakePassengerOrdersUseCase) RatePassengerOrder(context.Context, uuid.UUID, uuid.UUID, dto.RateOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, nil
}

type fakePassengerLookupRepositoryForOrders struct {
	passenger domain.Passenger
}

func (repository fakePassengerLookupRepositoryForOrders) GetByID(context.Context, uuid.UUID) (domain.Passenger, error) {
	return repository.passenger, nil
}

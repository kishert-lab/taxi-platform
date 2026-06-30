package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	driverapp "github.com/kishert-lab/taxi-platform/internal/driver"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

func TestPassengerCanCreateOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	passengerID := uuid.New()
	orderID := uuid.New()
	orderUseCase := &fakePassengerOrderUseCase{
		createResult: dto.PassengerOrderResponse{
			OrderID:        orderID,
			Status:         domain.OrderStatusSearching,
			AllowedActions: []string{"cancel"},
		},
	}
	router := passengerRouter(passengerID, domain.UserRolePassenger, orderUseCase, &fakePassengerAddressUseCase{})

	requestBody := `{
		"city_id":"11111111-1111-1111-1111-111111111111",
		"pickup_location":{"latitude":56.838011,"longitude":60.597465},
		"pickup_address":"Lenina 1",
		"destination_location":{"latitude":56.848011,"longitude":60.607465},
		"destination_address":"Mira 10",
		"tariff_id":"22222222-2222-2222-2222-222222222222",
		"payment_type":"cash"
	}`
	responseRecorder := performJSON(router, http.MethodPost, "/api/v1/passenger/orders", requestBody)

	if responseRecorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	if orderUseCase.createPassengerID != passengerID {
		t.Fatalf("expected passenger id %s, got %s", passengerID, orderUseCase.createPassengerID)
	}

	var responseBody response.Success
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Meta.RequestID == "" {
		t.Fatal("expected request id in meta")
	}
}

func TestDriverCanAcceptOrder(t *testing.T) {
	gin.SetMode(gin.TestMode)

	driverID := uuid.New()
	orderID := uuid.New()
	driverUseCase := &fakeDriverMobileUseCase{
		acceptResult: dto.DriverOrderResponse{
			OrderID:        orderID,
			Status:         domain.OrderStatusDriverAssigned,
			AllowedActions: []string{"arrived", "call_passenger"},
		},
	}
	router := driverRouter(driverID, domain.UserRoleDriver, driverUseCase)

	responseRecorder := performJSON(router, http.MethodPost, "/api/v1/driver/orders/"+orderID.String()+"/accept", `{}`)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	if driverUseCase.acceptDriverID != driverID {
		t.Fatalf("expected driver id %s, got %s", driverID, driverUseCase.acceptDriverID)
	}
	if driverUseCase.acceptOrderID != orderID {
		t.Fatalf("expected order id %s, got %s", orderID, driverUseCase.acceptOrderID)
	}
}

func TestDriverCanUploadBufferedRouteBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	driverID := uuid.New()
	orderID := uuid.New()
	driverUseCase := &fakeDriverMobileUseCase{
		routeBatchResult: dto.DriverOrderRouteBatchResponse{
			OrderID:        orderID,
			AcceptedPoints: 2,
			IgnoredPoints:  1,
		},
	}
	router := driverRouter(driverID, domain.UserRoleDriver, driverUseCase)

	responseRecorder := performJSON(router, http.MethodPost, "/api/v1/driver/orders/"+orderID.String()+"/route/batch", `{
		"points": [
			{
				"location": {"latitude": 56.8, "longitude": 60.6},
				"heading": 180,
				"speed_mps": 7.5,
				"accuracy_meters": 8.0,
				"recorded_at": "2026-06-28T10:00:00Z"
			}
		]
	}`)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	if driverUseCase.routeBatchDriverID != driverID {
		t.Fatalf("expected driver id %s, got %s", driverID, driverUseCase.routeBatchDriverID)
	}
	if driverUseCase.routeBatchOrderID != orderID {
		t.Fatalf("expected order id %s, got %s", orderID, driverUseCase.routeBatchOrderID)
	}
	if len(driverUseCase.routeBatchRequest.Points) != 1 {
		t.Fatalf("expected one route point, got %d", len(driverUseCase.routeBatchRequest.Points))
	}
}

func TestDriverBufferedRouteBatchValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := driverRouter(uuid.New(), domain.UserRoleDriver, &fakeDriverMobileUseCase{})

	emptyBatch := performJSON(router, http.MethodPost, "/api/v1/driver/orders/"+uuid.New().String()+"/route/batch", `{"points":[]}`)
	if emptyBatch.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty batch, got %d: %s", emptyBatch.Code, emptyBatch.Body.String())
	}
	assertErrorCode(t, emptyBatch, response.CodeValidationError)

	tooLargePoints := make([]map[string]any, 0, 501)
	for index := 0; index < 501; index++ {
		tooLargePoints = append(tooLargePoints, map[string]any{
			"location": map[string]float64{
				"latitude":  56.8,
				"longitude": 60.6,
			},
			"recorded_at": "2026-06-28T10:00:00Z",
		})
	}
	requestBody, err := json.Marshal(map[string]any{"points": tooLargePoints})
	if err != nil {
		t.Fatalf("marshal oversized batch: %v", err)
	}
	tooLargeBatch := performJSON(router, http.MethodPost, "/api/v1/driver/orders/"+uuid.New().String()+"/route/batch", string(requestBody))
	if tooLargeBatch.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized batch, got %d: %s", tooLargeBatch.Code, tooLargeBatch.Body.String())
	}
	assertErrorCode(t, tooLargeBatch, response.CodeValidationError)
}

func TestDriverBufferedRouteBatchErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	driverID := uuid.New()
	orderID := uuid.New()

	forbiddenRouter := driverRouter(driverID, domain.UserRoleDriver, &fakeDriverMobileUseCase{
		routeBatchErr: errors.Join(driverapp.ErrOrderAccessDenied),
	})
	forbiddenResponse := performJSON(forbiddenRouter, http.MethodPost, "/api/v1/driver/orders/"+orderID.String()+"/route/batch", `{
		"points": [{"location":{"latitude":56.8,"longitude":60.6},"recorded_at":"2026-06-28T10:00:00Z"}]
	}`)
	if forbiddenResponse.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", forbiddenResponse.Code, forbiddenResponse.Body.String())
	}
	assertErrorCode(t, forbiddenResponse, response.CodeForbidden)

	notFoundRouter := driverRouter(driverID, domain.UserRoleDriver, &fakeDriverMobileUseCase{
		routeBatchErr: driverapp.ErrCurrentOrderNotFound,
	})
	notFoundResponse := performJSON(notFoundRouter, http.MethodPost, "/api/v1/driver/orders/"+orderID.String()+"/route/batch", `{
		"points": [{"location":{"latitude":56.8,"longitude":60.6},"recorded_at":"2026-06-28T10:00:00Z"}]
	}`)
	if notFoundResponse.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", notFoundResponse.Code, notFoundResponse.Body.String())
	}
	assertErrorCode(t, notFoundResponse, response.CodeOrderNotFound)
}

func TestInvalidTransitionReturnsOrderInvalidState(t *testing.T) {
	gin.SetMode(gin.TestMode)

	driverID := uuid.New()
	orderID := uuid.New()
	driverUseCase := &fakeDriverMobileUseCase{
		acceptErr: domain.ErrInvalidOrderStatusTransition,
	}
	router := driverRouter(driverID, domain.UserRoleDriver, driverUseCase)

	responseRecorder := performJSON(router, http.MethodPost, "/api/v1/driver/orders/"+orderID.String()+"/accept", `{}`)

	if responseRecorder.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	assertErrorCode(t, responseRecorder, response.CodeOrderInvalidState)
}

func TestCurrentOrderReturnsAllowedActions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	passengerID := uuid.New()
	orderUseCase := &fakePassengerOrderUseCase{
		currentResult: dto.PassengerOrderResponse{
			OrderID:        uuid.New(),
			Status:         domain.OrderStatusDriverArriving,
			AllowedActions: []string{"cancel", "call_driver"},
		},
	}
	router := passengerRouter(passengerID, domain.UserRolePassenger, orderUseCase, &fakePassengerAddressUseCase{})

	responseRecorder := performJSON(router, http.MethodGet, "/api/v1/passenger/orders/current", "")

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	var responseBody struct {
		Data dto.PassengerOrderResponse `json:"data"`
		Meta response.Meta              `json:"meta"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(responseBody.Data.AllowedActions) != 2 || responseBody.Data.AllowedActions[1] != "call_driver" {
		t.Fatalf("unexpected allowed actions: %#v", responseBody.Data.AllowedActions)
	}
}

func TestEstimateReturnsTariffPrice(t *testing.T) {
	gin.SetMode(gin.TestMode)

	expectedTariffID := uuid.New()
	orderUseCase := &fakePassengerOrderUseCase{
		estimateResult: dto.OrderEstimateResponse{
			TariffID:    expectedTariffID,
			TariffName:  "Economy",
			DistanceKM:  4.2,
			DurationMin: 11,
			Price:       250,
			Currency:    "RUB",
			PriceType:   "estimated",
		},
	}
	router := passengerRouter(uuid.New(), domain.UserRolePassenger, orderUseCase, &fakePassengerAddressUseCase{})

	requestBody := `{
		"city_id":"11111111-1111-1111-1111-111111111111",
		"tariff_id":"22222222-2222-2222-2222-222222222222",
		"pickup_location":{"latitude":56.838011,"longitude":60.597465},
		"destination_location":{"latitude":56.848011,"longitude":60.607465}
	}`
	responseRecorder := performJSON(router, http.MethodPost, "/api/v1/passenger/orders/estimate", requestBody)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	var responseBody struct {
		Data dto.OrderEstimateResponse `json:"data"`
		Meta response.Meta             `json:"meta"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Data.TariffID != expectedTariffID || responseBody.Data.Price != 250 {
		t.Fatalf("unexpected estimate: %#v", responseBody.Data)
	}
}

func TestUnauthorizedRequestRejected(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := passengerRouter(uuid.Nil, "", &fakePassengerOrderUseCase{}, &fakePassengerAddressUseCase{})
	responseRecorder := performJSON(router, http.MethodGet, "/api/v1/passenger/orders/current", "")

	if responseRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	assertErrorCode(t, responseRecorder, response.CodeUnauthorized)
}

func TestAuthenticatedNonPassengerCanUsePassengerRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	orderUseCase := &fakePassengerOrderUseCase{
		currentResult: dto.PassengerOrderResponse{
			OrderID: uuid.New(),
			Status:  domain.OrderStatusCreated,
		},
	}
	router := passengerRouter(uuid.New(), domain.UserRoleDriver, orderUseCase, &fakePassengerAddressUseCase{})
	responseRecorder := performJSON(router, http.MethodGet, "/api/v1/passenger/orders/current", "")

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
}

func TestPassengerCanSearchAddresses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	passengerID := uuid.New()
	addressUseCase := &fakePassengerAddressUseCase{
		results: []geodomain.SearchResult{
			{
				ID:          "pelias:address:1",
				Provider:    geodomain.ProviderPelias,
				Name:        "Мира 8",
				Address:     "Пермь, улица Мира, 8",
				Coordinates: geodomain.Coordinates{Latitude: 58.010455, Longitude: 56.229443},
				Confidence:  0.91,
			},
		},
	}

	router := passengerRouter(passengerID, domain.UserRolePassenger, &fakePassengerOrderUseCase{}, addressUseCase)
	responseRecorder := performJSON(router, http.MethodGet, "/api/v1/passenger/address/search?q=%D0%9C%D0%B8%D1%80%D0%B0&limit=5", "")

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	if addressUseCase.passengerID != passengerID {
		t.Fatalf("expected passenger id %s, got %s", passengerID, addressUseCase.passengerID)
	}
	if addressUseCase.query != "Мира" || addressUseCase.limit != 5 {
		t.Fatalf("unexpected address search request: query=%q limit=%d", addressUseCase.query, addressUseCase.limit)
	}
}

func passengerRouter(userID uuid.UUID, role domain.UserRole, orderUseCase PassengerOrderUseCase, addressUseCase PassengerAddressUseCase) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(testAuthContext(userID, role))
	api := router.Group("/api/v1")
	NewPassengerMobileHandler(&fakePassengerProfileUseCase{}, orderUseCase).RegisterRoutes(api)
	NewPassengerAddressHandler(addressUseCase).RegisterRoutes(api, func(context *gin.Context) {
		context.Next()
	})
	return router
}

func driverRouter(userID uuid.UUID, role domain.UserRole, useCase DriverMobileUseCase) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(testAuthContext(userID, role))
	api := router.Group("/api/v1")
	NewDriverMobileHandler(useCase).RegisterRoutes(api)
	return router
}

func testAuthContext(userID uuid.UUID, role domain.UserRole) gin.HandlerFunc {
	return func(context *gin.Context) {
		if userID != uuid.Nil {
			context.Set("user_id", userID)
			if role == domain.UserRolePassenger {
				context.Set(middleware.PassengerIDContextKey, userID)
			}
		}
		if role != "" {
			context.Set(middleware.UserRoleContextKey, role)
		}
		context.Next()
	}
}

func performJSON(router http.Handler, method string, path string, body string) *httptest.ResponseRecorder {
	var requestBody *bytes.Reader
	if body == "" {
		requestBody = bytes.NewReader(nil)
	} else {
		requestBody = bytes.NewReader([]byte(body))
	}

	request := httptest.NewRequest(method, path, requestBody)
	request.Header.Set("Content-Type", "application/json")
	responseRecorder := httptest.NewRecorder()
	router.ServeHTTP(responseRecorder, request)
	return responseRecorder
}

func assertErrorCode(t *testing.T, responseRecorder *httptest.ResponseRecorder, expected response.ErrorCode) {
	t.Helper()

	var responseBody response.Error
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if responseBody.Error.Code != expected {
		t.Fatalf("expected error code %s, got %s", expected, responseBody.Error.Code)
	}
}

type fakePassengerProfileUseCase struct{}

func (useCase *fakePassengerProfileUseCase) CreatePassengerProfile(_ context.Context, passengerID uuid.UUID, request dto.PassengerProfileRequest) (dto.PassengerProfileResponse, error) {
	return dto.PassengerProfileResponse{ID: passengerID, Name: request.Name, Email: request.Email}, nil
}

func (useCase *fakePassengerProfileUseCase) GetPassengerProfile(_ context.Context, passengerID uuid.UUID) (dto.PassengerProfileResponse, error) {
	return dto.PassengerProfileResponse{ID: passengerID, Name: "Irina"}, nil
}

func (useCase *fakePassengerProfileUseCase) UpdatePassengerProfile(_ context.Context, passengerID uuid.UUID, request dto.PassengerProfilePatchRequest) (dto.PassengerProfileResponse, error) {
	responseBody := dto.PassengerProfileResponse{ID: passengerID}
	if request.Name != nil {
		responseBody.Name = *request.Name
	}
	if request.Email != nil {
		responseBody.Email = *request.Email
	}
	return responseBody, nil
}

func (useCase *fakePassengerProfileUseCase) UploadPassengerProfilePhoto(_ context.Context, _ uuid.UUID, _ dto.ProfilePhotoUploadRequest) (dto.ProfilePhotoUploadResponse, error) {
	return dto.ProfilePhotoUploadResponse{PhotoURL: "https://cdn.example.com/passengers/photo.jpg"}, nil
}

type fakePassengerOrderUseCase struct {
	estimateResult    dto.OrderEstimateResponse
	createResult      dto.PassengerOrderResponse
	currentResult     dto.PassengerOrderResponse
	createPassengerID uuid.UUID
	err               error
}

func (useCase *fakePassengerOrderUseCase) EstimatePassengerOrder(_ context.Context, _ uuid.UUID, _ dto.OrderEstimateRequest) (dto.OrderEstimateResponse, error) {
	return useCase.estimateResult, useCase.err
}

func (useCase *fakePassengerOrderUseCase) CreatePassengerOrder(_ context.Context, passengerID uuid.UUID, _ dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error) {
	useCase.createPassengerID = passengerID
	return useCase.createResult, useCase.err
}

func (useCase *fakePassengerOrderUseCase) GetCurrentPassengerOrder(_ context.Context, _ uuid.UUID) (dto.PassengerOrderResponse, error) {
	if useCase.currentResult.OrderID == uuid.Nil && useCase.err == nil {
		return dto.PassengerOrderResponse{}, ErrMobileOrderNotFound
	}
	return useCase.currentResult, useCase.err
}

func (useCase *fakePassengerOrderUseCase) ListPassengerOrderHistory(_ context.Context, _ uuid.UUID) (dto.OrderHistoryResponse, error) {
	return dto.OrderHistoryResponse{}, useCase.err
}

func (useCase *fakePassengerOrderUseCase) GetPassengerOrder(_ context.Context, _ uuid.UUID, _ uuid.UUID) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, useCase.err
}

func (useCase *fakePassengerOrderUseCase) CancelPassengerOrder(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ dto.CancelOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, useCase.err
}

func (useCase *fakePassengerOrderUseCase) RatePassengerOrder(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ dto.RateOrderRequest) (dto.PassengerOrderResponse, error) {
	return dto.PassengerOrderResponse{}, useCase.err
}

type fakePassengerAddressUseCase struct {
	passengerID uuid.UUID
	query       string
	cityID      *uuid.UUID
	latitude    *float64
	longitude   *float64
	limit       int
	results     []geodomain.SearchResult
	err         error
}

func (useCase *fakePassengerAddressUseCase) SearchPassengerAddresses(_ context.Context, passengerID uuid.UUID, query string, cityID *uuid.UUID, focusLatitude *float64, focusLongitude *float64, limit int) ([]geodomain.SearchResult, error) {
	useCase.passengerID = passengerID
	useCase.query = query
	useCase.cityID = cityID
	useCase.latitude = focusLatitude
	useCase.longitude = focusLongitude
	useCase.limit = limit
	return useCase.results, useCase.err
}

type fakeDriverMobileUseCase struct {
	acceptResult       dto.DriverOrderResponse
	acceptErr          error
	acceptDriverID     uuid.UUID
	acceptOrderID      uuid.UUID
	routeBatchResult   dto.DriverOrderRouteBatchResponse
	routeBatchErr      error
	routeBatchDriverID uuid.UUID
	routeBatchOrderID  uuid.UUID
	routeBatchRequest  dto.DriverOrderRouteBatchRequest
}

func (useCase *fakeDriverMobileUseCase) GetDriverProfile(_ context.Context, driverID uuid.UUID) (dto.DriverProfileResponse, error) {
	return dto.DriverProfileResponse{ID: driverID, Status: domain.DriverStatusOnline}, nil
}

func (useCase *fakeDriverMobileUseCase) ListDriverCars(context.Context, uuid.UUID) (dto.TaxiParkCarsResponse, error) {
	return dto.TaxiParkCarsResponse{}, nil
}

func (useCase *fakeDriverMobileUseCase) UpdateDriverProfile(_ context.Context, driverID uuid.UUID, _ dto.DriverProfilePatchRequest) (dto.DriverProfileResponse, error) {
	return dto.DriverProfileResponse{ID: driverID, Status: domain.DriverStatusOnline}, nil
}

func (useCase *fakeDriverMobileUseCase) UploadDriverProfilePhoto(_ context.Context, _ uuid.UUID, _ dto.ProfilePhotoUploadRequest) (dto.ProfilePhotoUploadResponse, error) {
	return dto.ProfilePhotoUploadResponse{PhotoURL: "https://cdn.example.com/drivers/photo.jpg"}, nil
}

func (useCase *fakeDriverMobileUseCase) MarkDriverOnline(_ context.Context, driverID uuid.UUID) (dto.DriverProfileResponse, error) {
	return dto.DriverProfileResponse{ID: driverID, Status: domain.DriverStatusOnline}, nil
}

func (useCase *fakeDriverMobileUseCase) MarkDriverOffline(_ context.Context, driverID uuid.UUID) (dto.DriverProfileResponse, error) {
	return dto.DriverProfileResponse{ID: driverID, Status: domain.DriverStatusOffline}, nil
}

func (useCase *fakeDriverMobileUseCase) UpdateDriverLocation(_ context.Context, _ uuid.UUID, _ dto.DriverLocationRequest) error {
	return nil
}

func (useCase *fakeDriverMobileUseCase) UpdateDriverLocationBatch(_ context.Context, _ uuid.UUID, _ dto.DriverLocationBatchRequest) error {
	return nil
}

func (useCase *fakeDriverMobileUseCase) GetCurrentDriverOrder(_ context.Context, _ uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, ErrMobileOrderNotFound
}

func (useCase *fakeDriverMobileUseCase) GetDriverOrder(_ context.Context, _ uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{OrderID: orderID}, nil
}

func (useCase *fakeDriverMobileUseCase) ListDriverOrderHistory(_ context.Context, _ uuid.UUID) (dto.DriverOrderHistoryResponse, error) {
	return dto.DriverOrderHistoryResponse{}, nil
}

func (useCase *fakeDriverMobileUseCase) ListDriverOrderOffers(_ context.Context, _ uuid.UUID) (dto.DriverOrderOffersResponse, error) {
	return dto.DriverOrderOffersResponse{}, nil
}

func (useCase *fakeDriverMobileUseCase) GetDriverOrderRoute(_ context.Context, _ uuid.UUID, orderID uuid.UUID) (dto.OrderRouteResponse, error) {
	return dto.OrderRouteResponse{OrderID: orderID, Points: []dto.OrderRoutePointResponse{}}, nil
}

func (useCase *fakeDriverMobileUseCase) AppendOrderRoutePoints(_ context.Context, driverID uuid.UUID, orderID uuid.UUID, request dto.DriverOrderRouteBatchRequest) (dto.DriverOrderRouteBatchResponse, error) {
	useCase.routeBatchDriverID = driverID
	useCase.routeBatchOrderID = orderID
	useCase.routeBatchRequest = request
	if useCase.routeBatchErr != nil {
		return dto.DriverOrderRouteBatchResponse{}, useCase.routeBatchErr
	}
	return useCase.routeBatchResult, nil
}

func (useCase *fakeDriverMobileUseCase) AcceptDriverOrder(_ context.Context, driverID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error) {
	useCase.acceptDriverID = driverID
	useCase.acceptOrderID = orderID
	if useCase.acceptErr != nil {
		return dto.DriverOrderResponse{}, errors.Join(useCase.acceptErr)
	}
	return useCase.acceptResult, nil
}

func (useCase *fakeDriverMobileUseCase) RejectDriverOrder(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ dto.RejectOrderRequest) error {
	return nil
}

func (useCase *fakeDriverMobileUseCase) MarkDriverArriving(_ context.Context, _ uuid.UUID, _ uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, nil
}

func (useCase *fakeDriverMobileUseCase) MarkDriverArrived(_ context.Context, _ uuid.UUID, _ uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, nil
}

func (useCase *fakeDriverMobileUseCase) CancelDriverOrder(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ string) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, nil
}

func (useCase *fakeDriverMobileUseCase) StartDriverTrip(_ context.Context, _ uuid.UUID, _ uuid.UUID) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, nil
}

func (useCase *fakeDriverMobileUseCase) CompleteDriverTrip(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ dto.CompleteOrderRequest) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, nil
}

func (useCase *fakeDriverMobileUseCase) RatePassenger(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ dto.RateOrderRequest) (dto.DriverOrderResponse, error) {
	return dto.DriverOrderResponse{}, nil
}

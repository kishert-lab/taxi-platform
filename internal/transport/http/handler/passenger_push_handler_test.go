package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
)

func TestPassengerCanRegisterPushToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	passengerID := uuid.New()
	useCase := &fakePassengerPushHandlerUseCase{
		response: dto.PassengerPushTokenResponse{
			Token:    "fcm-token",
			Platform: "android",
			DeviceID: "pixel-8",
		},
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(func(context *gin.Context) {
		context.Set(middleware.PassengerIDContextKey, passengerID)
		context.Next()
	})
	api := router.Group("/api/v1")
	NewPassengerPushHandler(useCase).RegisterRoutes(api, func(context *gin.Context) {
		context.Next()
	})

	responseRecorder := performJSON(router, http.MethodPost, "/api/v1/passenger/push-tokens", `{
		"token":"fcm-token",
		"platform":"android",
		"device_id":"pixel-8"
	}`)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
	if useCase.passengerID != passengerID {
		t.Fatalf("expected passenger id %s, got %s", passengerID, useCase.passengerID)
	}
	if useCase.request.Token != "fcm-token" || useCase.request.Platform != "android" {
		t.Fatalf("unexpected request: %+v", useCase.request)
	}

	var responseBody struct {
		Data dto.PassengerPushTokenResponse `json:"data"`
	}
	if err := json.Unmarshal(responseRecorder.Body.Bytes(), &responseBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if responseBody.Data.Token != "fcm-token" {
		t.Fatalf("unexpected response body: %+v", responseBody.Data)
	}
}

func TestPassengerCanRegisterPushTokenWithLegacyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	passengerID := uuid.New()
	useCase := &fakePassengerPushHandlerUseCase{
		response: dto.PassengerPushTokenResponse{
			Token:    "fcm-token",
			Platform: "android",
		},
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(func(context *gin.Context) {
		context.Set(middleware.PassengerIDContextKey, passengerID)
		context.Next()
	})
	api := router.Group("/api/v1")
	NewPassengerPushHandler(useCase).RegisterRoutes(api, func(context *gin.Context) {
		context.Next()
	})

	responseRecorder := performJSON(router, http.MethodPost, "/api/v1/passenger/push/token", `{
		"token":"fcm-token",
		"platform":"android"
	}`)

	if responseRecorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", responseRecorder.Code, responseRecorder.Body.String())
	}
}

type fakePassengerPushHandlerUseCase struct {
	passengerID uuid.UUID
	request     dto.PassengerPushTokenRequest
	response    dto.PassengerPushTokenResponse
}

func (useCase *fakePassengerPushHandlerUseCase) RegisterToken(_ context.Context, passengerID uuid.UUID, request dto.PassengerPushTokenRequest) (dto.PassengerPushTokenResponse, error) {
	useCase.passengerID = passengerID
	useCase.request = request
	return useCase.response, nil
}

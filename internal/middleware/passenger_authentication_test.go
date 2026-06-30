package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	passengerapp "github.com/kishert-lab/taxi-platform/internal/passenger"
)

func TestAuthenticatePassengerAccessTokenSetsPassengerContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	passengerID := uuid.New()
	tokenManager := passengerapp.NewTokenManager(passengerapp.TokenManagerConfig{
		AccessSecret:  "test-passenger-access-secret-32-characters-min",
		RefreshSecret: "test-passenger-refresh-secret-32-characters-min",
		Issuer:        "taxi-platform-test",
		AccessTTL:     time.Hour,
		RefreshTTL:    time.Hour,
	})
	accessToken, err := tokenManager.GeneratePassengerAccessToken(passengerID, time.Now().UTC())
	if err != nil {
		t.Fatalf("GeneratePassengerAccessToken returned error: %v", err)
	}

	router := gin.New()
	router.Use(AuthenticatePassengerAccessToken(tokenManager, fakePassengerLookupRepository{
		passenger: domain.Passenger{ID: passengerID, IsActive: true},
	}))
	router.GET("/passenger/me", func(context *gin.Context) {
		actualPassengerID, exists := context.Get(PassengerIDContextKey)
		if !exists || actualPassengerID != passengerID {
			t.Fatalf("expected passenger_id %s, got %#v", passengerID, actualPassengerID)
		}
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/passenger/me", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestAuthenticatePassengerAccessTokenRejectsUserToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	userTokenManager := auth.NewTokenManager(auth.TokenManagerConfig{
		AccessSecret:  "test-access-secret-with-more-than-32-chars",
		RefreshSecret: "test-refresh-secret-with-more-than-32-chars",
		Issuer:        "taxi-platform-test",
		AccessTTL:     time.Hour,
		RefreshTTL:    time.Hour,
	})
	userAccessToken, _, _, err := userTokenManager.IssueTokenPair(uuid.New(), domain.UserRoleDriver, time.Now().UTC())
	if err != nil {
		t.Fatalf("IssueTokenPair returned error: %v", err)
	}

	router := gin.New()
	router.Use(AuthenticatePassengerAccessToken(passengerapp.NewTokenManager(passengerapp.TokenManagerConfig{
		AccessSecret:  "test-passenger-access-secret-32-characters-min",
		RefreshSecret: "test-passenger-refresh-secret-32-characters-min",
		Issuer:        "taxi-platform-test",
		AccessTTL:     time.Hour,
		RefreshTTL:    time.Hour,
	}), fakePassengerLookupRepository{}))
	router.GET("/passenger/me", func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/passenger/me", nil)
	request.Header.Set("Authorization", "Bearer "+userAccessToken)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

type fakePassengerLookupRepository struct {
	passenger domain.Passenger
}

func (repository fakePassengerLookupRepository) GetByID(context.Context, uuid.UUID) (domain.Passenger, error) {
	return repository.passenger, nil
}

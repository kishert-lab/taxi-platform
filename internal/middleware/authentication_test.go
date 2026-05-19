package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	authapp "github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/domain"
)

func TestAuthenticateAccessTokenSetsUserContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tokenManager := authapp.NewTokenManager(authapp.TokenManagerConfig{
		AccessSecret:  "test-access-secret-with-more-than-32-chars",
		RefreshSecret: "test-refresh-secret-with-more-than-32-chars",
		Issuer:        "taxi-platform-test",
		AccessTTL:     time.Hour,
		RefreshTTL:    time.Hour,
	})
	userID := uuid.New()
	accessToken, _, _, err := tokenManager.IssueTokenPair(userID, domain.UserRoleTaxiPark, time.Now().UTC())
	if err != nil {
		t.Fatalf("issue token pair: %v", err)
	}

	router := gin.New()
	router.Use(AuthenticateAccessToken(tokenManager))
	router.GET("/protected", func(context *gin.Context) {
		actualUserID, exists := context.Get(UserIDContextKey)
		if !exists || actualUserID != userID {
			t.Fatalf("expected user_id %s, got %#v", userID, actualUserID)
		}
		actualRole, exists := context.Get(UserRoleContextKey)
		if !exists || actualRole != domain.UserRoleTaxiPark {
			t.Fatalf("expected role %s, got %#v", domain.UserRoleTaxiPark, actualRole)
		}
		context.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer "+accessToken)
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

func TestAuthenticateAccessTokenRejectsMissingBearerToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthenticateAccessToken(authapp.NewTokenManager(authapp.TokenManagerConfig{})))
	router.GET("/protected", func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/protected", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
}

func TestAuthenticateAccessTokenSkipsPublicPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(AuthenticateAccessToken(authapp.NewTokenManager(authapp.TokenManagerConfig{}), "/api/v1/auth"))
	router.GET("/api/v1/auth/login", func(context *gin.Context) {
		context.Status(http.StatusNoContent)
	})

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil))

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected %d, got %d", http.StatusNoContent, recorder.Code)
	}
}

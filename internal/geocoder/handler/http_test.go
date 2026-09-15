package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
)

func TestAdminGeocoderRoutesRequireAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set(middleware.UserRoleContextKey, context.GetHeader("X-Test-Role"))
	})
	New(geocoderUseCaseStub{}).RegisterRoutes(router)

	for _, testCase := range []struct {
		name string
		role string
		want int
	}{
		{name: "driver is forbidden", role: string(domain.UserRoleDriver), want: http.StatusForbidden},
		{name: "admin is allowed", role: string(domain.UserRoleAdmin), want: http.StatusOK},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/admin/geocoder/local-points", nil)
			request.Header.Set("X-Test-Role", testCase.role)
			responseRecorder := httptest.NewRecorder()

			router.ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != testCase.want {
				t.Fatalf("expected status %d, got %d", testCase.want, responseRecorder.Code)
			}
		})
	}
}

type geocoderUseCaseStub struct{}

func (geocoderUseCaseStub) Search(context.Context, geodomain.SearchRequest) ([]geodomain.SearchResult, error) {
	return nil, nil
}

func (geocoderUseCaseStub) ConfirmPoint(context.Context, geoservice.ConfirmPointRequest) (geodomain.LocalGeoPoint, error) {
	return geodomain.LocalGeoPoint{}, nil
}

func (geocoderUseCaseStub) CreateLocalPoint(context.Context, geoservice.AdminLocalPointRequest) (geodomain.LocalGeoPoint, error) {
	return geodomain.LocalGeoPoint{}, nil
}

func (geocoderUseCaseStub) ListLocalPoints(context.Context, geoservice.LocalPointFilter) ([]geodomain.LocalGeoPoint, error) {
	return nil, nil
}

func (geocoderUseCaseStub) ApproveLocalPoint(context.Context, uuid.UUID, *uuid.UUID) (geodomain.LocalGeoPoint, error) {
	return geodomain.LocalGeoPoint{}, nil
}

func (geocoderUseCaseStub) RejectLocalPoint(context.Context, uuid.UUID, *uuid.UUID) (geodomain.LocalGeoPoint, error) {
	return geodomain.LocalGeoPoint{}, nil
}

func (geocoderUseCaseStub) ExportTrustedLocalPoints(context.Context) ([]geodomain.LocalGeoPoint, error) {
	return nil, nil
}

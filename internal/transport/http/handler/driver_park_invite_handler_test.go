package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

func TestInviteFailureMapsDomainErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantCode   response.ErrorCode
	}{
		{name: "not found", err: domain.ErrParkInviteNotFound, wantStatus: http.StatusNotFound, wantCode: response.CodeNotFound},
		{name: "wrapped not found", err: fmt.Errorf("lookup: %w", domain.ErrParkInviteNotFound), wantStatus: http.StatusNotFound, wantCode: response.CodeNotFound},
		{name: "forbidden", err: domain.ErrParkInviteForbidden, wantStatus: http.StatusForbidden, wantCode: response.CodeForbidden},
		{name: "closed", err: domain.ErrParkInviteClosed, wantStatus: http.StatusConflict, wantCode: "PARK_INVITE_CONFLICT"},
		{name: "expired", err: domain.ErrParkInviteExpired, wantStatus: http.StatusConflict, wantCode: "PARK_INVITE_CONFLICT"},
		{name: "same park", err: domain.ErrParkInviteSamePark, wantStatus: http.StatusConflict, wantCode: "PARK_INVITE_CONFLICT"},
		{name: "inactive park", err: domain.ErrParkInviteInactivePark, wantStatus: http.StatusConflict, wantCode: "PARK_INVITE_CONFLICT"},
		{name: "driver working", err: domain.ErrParkInviteDriverWorking, wantStatus: http.StatusConflict, wantCode: "PARK_INVITE_CONFLICT"},
		{name: "changed context", err: domain.ErrParkInviteContextChanged, wantStatus: http.StatusConflict, wantCode: "PARK_INVITE_CONFLICT"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			context, _ := gin.CreateTestContext(recorder)
			inviteFailure(context, test.err)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status=%d want=%d body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			var body response.Error
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != test.wantCode || body.Error.Message != test.err.Error() {
				t.Fatalf("response=%+v", body.Error)
			}
		})
	}
}

func TestDriverParkInviteHandlerRejectsInvalidInputBeforeService(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewDriverParkInviteHandler(nil)
	router := gin.New()
	router.Use(func(context *gin.Context) {
		context.Set("user_id", uuid.New())
		context.Next()
	})
	router.POST("/create", handler.create)
	router.POST("/respond/:id", handler.respond(domain.DriverParkInviteAccepted))
	router.PUT("/push-token", handler.savePushToken)

	for _, test := range []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{name: "missing phone", method: http.MethodPost, path: "/create", body: `{}`},
		{name: "invalid invite id", method: http.MethodPost, path: "/respond/not-a-uuid", body: `{}`},
		{name: "invalid push platform", method: http.MethodPut, path: "/push-token", body: `{"token":"token","platform":"desktop"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := performJSON(router, test.method, test.path, test.body)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			var body response.Error
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body.Error.Code != "VALIDATION_ERROR" {
				t.Fatalf("code=%s", body.Error.Code)
			}
		})
	}
}

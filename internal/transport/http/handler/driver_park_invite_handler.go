package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/driverinvite"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
	"net/http"
)

type DriverParkInviteHandler struct{ service *driverinvite.Service }

func NewDriverParkInviteHandler(service *driverinvite.Service) *DriverParkInviteHandler {
	return &DriverParkInviteHandler{service}
}
func (handler *DriverParkInviteHandler) RegisterRoutes(router gin.IRouter) {
	driver := router.Group("/driver", middleware.RequireRole(domain.UserRoleDriver))
	driver.GET("/park-invites", handler.list(false))
	driver.POST("/park-invites/:id/accept", handler.respond(domain.DriverParkInviteAccepted))
	driver.POST("/park-invites/:id/decline", handler.respond(domain.DriverParkInviteDeclined))
	driver.PUT("/push-token", handler.savePushToken)
	driver.DELETE("/push-token", handler.deletePushToken)
	park := router.Group("/taxi-park/driver-invites", middleware.RequireRole(domain.UserRoleTaxiPark, domain.UserRoleDispatcher), middleware.RequirePermission(domain.PermissionTaxiParkInviteDrivers))
	park.GET("", handler.list(true))
	park.POST("", handler.create)
	park.POST("/:id/cancel", handler.respond(domain.DriverParkInviteCancelled))
}
func inviteFailure(context *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrParkInviteNotFound):
		response.Fail(context, http.StatusNotFound, "NOT_FOUND", err.Error(), nil)
	case errors.Is(err, domain.ErrParkInviteForbidden):
		response.Fail(context, http.StatusForbidden, "FORBIDDEN", err.Error(), nil)
	case errors.Is(err, domain.ErrParkInviteClosed), errors.Is(err, domain.ErrParkInviteExpired), errors.Is(err, domain.ErrParkInviteSamePark), errors.Is(err, domain.ErrParkInviteInactivePark), errors.Is(err, domain.ErrParkInviteDriverWorking), errors.Is(err, domain.ErrParkInviteContextChanged):
		response.Fail(context, http.StatusConflict, "PARK_INVITE_CONFLICT", err.Error(), nil)
	default:
		failByError(context, err)
	}
}
func (handler *DriverParkInviteHandler) create(context *gin.Context) {
	actor, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var input struct {
		Phone string `json:"phone" binding:"required,max=32"`
	}
	if err := context.ShouldBindJSON(&input); err != nil {
		response.Fail(context, 400, "VALIDATION_ERROR", "Укажите телефон водителя", nil)
		return
	}
	result, err := handler.service.Create(context.Request.Context(), actor, input.Phone)
	if err != nil {
		inviteFailure(context, err)
		return
	}
	result.FromTaxiParkID = nil
	result.FromTaxiParkName = ""
	response.OK(context, result)
}
func (handler *DriverParkInviteHandler) list(park bool) gin.HandlerFunc {
	return func(context *gin.Context) {
		actor, ok := userIDFromContext(context)
		if !ok {
			failUnauthorized(context, "User id is missing")
			return
		}
		result, err := handler.service.List(context.Request.Context(), actor, park)
		if err != nil {
			inviteFailure(context, err)
			return
		}
		response.OK(context, result)
	}
}
func (handler *DriverParkInviteHandler) respond(action domain.DriverParkInviteStatus) gin.HandlerFunc {
	return func(context *gin.Context) {
		actor, ok := userIDFromContext(context)
		if !ok {
			failUnauthorized(context, "User id is missing")
			return
		}
		id, err := uuid.Parse(context.Param("id"))
		if err != nil {
			response.Fail(context, 400, "VALIDATION_ERROR", "Некорректный идентификатор приглашения", nil)
			return
		}
		result, err := handler.service.Respond(context.Request.Context(), actor, id, action)
		if err != nil {
			inviteFailure(context, err)
			return
		}
		if action == domain.DriverParkInviteCancelled {
			result.FromTaxiParkID = nil
			result.FromTaxiParkName = ""
		}
		response.OK(context, result)
	}
}
func (handler *DriverParkInviteHandler) savePushToken(context *gin.Context) {
	actor, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var input struct {
		Token    string `json:"token" binding:"required,max=4096"`
		Platform string `json:"platform" binding:"required,oneof=android ios web"`
	}
	if err := context.ShouldBindJSON(&input); err != nil {
		response.Fail(context, 400, "VALIDATION_ERROR", "Некорректный push-токен", nil)
		return
	}
	if err := handler.service.SavePushToken(context.Request.Context(), actor, input.Token, input.Platform); err != nil {
		inviteFailure(context, err)
		return
	}
	response.OK(context, struct{}{})
}
func (handler *DriverParkInviteHandler) deletePushToken(context *gin.Context) {
	actor, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var input struct {
		Token string `json:"token" binding:"required,max=4096"`
	}
	if err := context.ShouldBindJSON(&input); err != nil {
		response.Fail(context, 400, "VALIDATION_ERROR", "Некорректный push-токен", nil)
		return
	}
	if err := handler.service.DeletePushToken(context.Request.Context(), actor, input.Token); err != nil {
		inviteFailure(context, err)
		return
	}
	response.OK(context, struct{}{})
}

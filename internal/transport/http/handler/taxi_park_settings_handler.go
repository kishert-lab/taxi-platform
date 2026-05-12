package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type TaxiParkSettingsUseCase interface {
	GetSettings(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkSettings, error)
	UpdateSettings(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error)
	ListTariffs(ctx context.Context, ownerUserID uuid.UUID) ([]domain.TaxiParkTariff, error)
	CreateTariff(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error)
	UpdateTariff(ctx context.Context, ownerUserID uuid.UUID, tariffID uuid.UUID, request dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error)
}

type TaxiParkSettingsHandler struct {
	useCase TaxiParkSettingsUseCase
}

func NewTaxiParkSettingsHandler(useCase TaxiParkSettingsUseCase) *TaxiParkSettingsHandler {
	return &TaxiParkSettingsHandler{useCase: useCase}
}

func (handler *TaxiParkSettingsHandler) RegisterRoutes(router gin.IRouter) {
	taxiPark := router.Group("/taxi-park", middleware.RequireRole(domain.UserRoleTaxiPark))
	taxiPark.GET("/settings", handler.GetSettings)
	taxiPark.PATCH("/settings", handler.UpdateSettings)
	taxiPark.GET("/tariffs", handler.ListTariffs)
	taxiPark.POST("/tariffs", handler.CreateTariff)
	taxiPark.PATCH("/tariffs/:id", handler.UpdateTariff)
}

// GetSettings godoc
// @Summary Get taxi park settings
// @Tags taxi-park-settings
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TaxiParkSettingsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/settings [get]
func (handler *TaxiParkSettingsHandler) GetSettings(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	settings, err := handler.useCase.GetSettings(context.Request.Context(), ownerUserID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.TaxiParkSettingsFromDomain(settings))
}

// UpdateSettings godoc
// @Summary Update taxi park settings and branding
// @Tags taxi-park-settings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.TaxiParkSettingsPatchRequest true "Taxi park settings patch"
// @Success 200 {object} TaxiParkSettingsSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/settings [patch]
func (handler *TaxiParkSettingsHandler) UpdateSettings(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var request dto.TaxiParkSettingsPatchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid taxi park settings request")
		return
	}
	settings, err := handler.useCase.UpdateSettings(context.Request.Context(), ownerUserID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.TaxiParkSettingsFromDomain(settings))
}

// ListTariffs godoc
// @Summary List taxi park tariffs
// @Tags taxi-park-tariffs
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TaxiParkTariffsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/tariffs [get]
func (handler *TaxiParkSettingsHandler) ListTariffs(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	tariffs, err := handler.useCase.ListTariffs(context.Request.Context(), ownerUserID)
	if err != nil {
		failByError(context, err)
		return
	}
	responseBody := dto.TaxiParkTariffsResponse{Tariffs: make([]dto.TaxiParkTariffResponse, 0, len(tariffs))}
	for _, tariff := range tariffs {
		responseBody.Tariffs = append(responseBody.Tariffs, dto.TaxiParkTariffFromDomain(tariff))
	}
	response.OK(context, responseBody)
}

// CreateTariff godoc
// @Summary Create taxi park tariff
// @Tags taxi-park-tariffs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.TaxiParkTariffRequest true "Taxi park tariff"
// @Success 201 {object} TaxiParkTariffSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/tariffs [post]
func (handler *TaxiParkSettingsHandler) CreateTariff(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var request dto.TaxiParkTariffRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid taxi park tariff request")
		return
	}
	tariff, err := handler.useCase.CreateTariff(context.Request.Context(), ownerUserID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.Created(context, dto.TaxiParkTariffFromDomain(tariff))
}

// UpdateTariff godoc
// @Summary Update taxi park tariff
// @Tags taxi-park-tariffs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Tariff ID"
// @Param request body dto.TaxiParkTariffPatchRequest true "Taxi park tariff patch"
// @Success 200 {object} TaxiParkTariffSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/tariffs/{id} [patch]
func (handler *TaxiParkSettingsHandler) UpdateTariff(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	tariffID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid tariff id")
		return
	}
	var request dto.TaxiParkTariffPatchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid taxi park tariff patch")
		return
	}
	tariff, err := handler.useCase.UpdateTariff(context.Request.Context(), ownerUserID, tariffID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.TaxiParkTariffFromDomain(tariff))
}

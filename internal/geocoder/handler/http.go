// Package handler exposes geocoder use cases over HTTP.
package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/geocoder/exporter"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type UseCase interface {
	Search(ctx context.Context, request geodomain.SearchRequest) ([]geodomain.SearchResult, error)
	ConfirmPoint(ctx context.Context, request geoservice.ConfirmPointRequest) (geodomain.LocalGeoPoint, error)
	CreateLocalPoint(ctx context.Context, request geoservice.AdminLocalPointRequest) (geodomain.LocalGeoPoint, error)
	ListLocalPoints(ctx context.Context, filter geoservice.LocalPointFilter) ([]geodomain.LocalGeoPoint, error)
	ApproveLocalPoint(ctx context.Context, id uuid.UUID, adminUserID *uuid.UUID) (geodomain.LocalGeoPoint, error)
	RejectLocalPoint(ctx context.Context, id uuid.UUID, adminUserID *uuid.UUID) (geodomain.LocalGeoPoint, error)
	ExportTrustedLocalPoints(ctx context.Context) ([]geodomain.LocalGeoPoint, error)
}

type Handler struct {
	useCase UseCase
}

func New(useCase UseCase) *Handler {
	return &Handler{useCase: useCase}
}

func (handler *Handler) RegisterRoutes(router gin.IRouter) {
	router.GET("/geocoder/search", handler.Search)
	router.POST("/geocoder/points/confirm", handler.ConfirmPoint)

	admin := router.Group("/admin/geocoder")
	admin.POST("/local-points", handler.CreateLocalPoint)
	admin.GET("/local-points", handler.ListLocalPoints)
	admin.POST("/local-points/:id/approve", handler.ApproveLocalPoint)
	admin.POST("/local-points/:id/reject", handler.RejectLocalPoint)
	admin.GET("/export/pelias-csv", handler.ExportPeliasCSV)
}

// Search godoc
// @Summary Search address with hybrid geocoder
// @Description Searches trusted local points first, then local Pelias, then temporary DaData fallback, then Yandex fallback.
// @Tags geocoder
// @Produce json
// @Param q query string true "Address query"
// @Param city_id query string false "City UUID"
// @Param lat query number false "Focus latitude"
// @Param lon query number false "Focus longitude"
// @Param limit query int false "Result limit"
// @Success 200 {object} SearchSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /geocoder/search [get]
func (handler *Handler) Search(context *gin.Context) {
	request, err := searchRequestFromQuery(context)
	if err != nil {
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid geocoder search request", err)
		return
	}
	results, err := handler.useCase.Search(context.Request.Context(), request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, SearchResponse{Results: searchResultsToResponse(results)})
}

// ConfirmPoint godoc
// @Summary Confirm local geocoder point
// @Description Persists a user, driver, or dispatcher confirmed point as platform-owned local data.
// @Tags geocoder
// @Accept json
// @Produce json
// @Param request body ConfirmPointRequest true "Confirmation request"
// @Success 200 {object} LocalPointSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /geocoder/points/confirm [post]
func (handler *Handler) ConfirmPoint(context *gin.Context) {
	var request ConfirmPointRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid geocoder confirmation request", err)
		return
	}
	confirmRequest, err := request.toService(context)
	if err != nil {
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid geocoder confirmation request", err)
		return
	}
	point, err := handler.useCase.ConfirmPoint(context.Request.Context(), confirmRequest)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, localPointToResponse(point))
}

// CreateLocalPoint godoc
// @Summary Create admin local geocoder point
// @Tags admin-geocoder
// @Accept json
// @Produce json
// @Param request body AdminLocalPointRequest true "Local point"
// @Success 201 {object} LocalPointSuccessResponse
// @Failure 400 {object} response.Error
// @Router /admin/geocoder/local-points [post]
func (handler *Handler) CreateLocalPoint(context *gin.Context) {
	var request AdminLocalPointRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid local geocoder point request", err)
		return
	}
	serviceRequest, err := request.toService()
	if err != nil {
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid local geocoder point request", err)
		return
	}
	point, err := handler.useCase.CreateLocalPoint(context.Request.Context(), serviceRequest)
	if err != nil {
		failByError(context, err)
		return
	}
	response.Created(context, localPointToResponse(point))
}

// ListLocalPoints godoc
// @Summary List local geocoder points
// @Tags admin-geocoder
// @Produce json
// @Param city_id query string false "City UUID"
// @Param trust_level query string false "confirmed|trusted|rejected"
// @Param limit query int false "Limit"
// @Success 200 {object} LocalPointsSuccessResponse
// @Router /admin/geocoder/local-points [get]
func (handler *Handler) ListLocalPoints(context *gin.Context) {
	filter, err := localPointFilterFromQuery(context)
	if err != nil {
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid local geocoder point filter", err)
		return
	}
	points, err := handler.useCase.ListLocalPoints(context.Request.Context(), filter)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, LocalPointsResponse{Points: localPointsToResponse(points)})
}

// ApproveLocalPoint godoc
// @Summary Approve local geocoder point
// @Tags admin-geocoder
// @Produce json
// @Param id path string true "Local point UUID"
// @Success 200 {object} LocalPointSuccessResponse
// @Failure 404 {object} response.Error
// @Router /admin/geocoder/local-points/{id}/approve [post]
func (handler *Handler) ApproveLocalPoint(context *gin.Context) {
	id, err := uuid.Parse(context.Param("id"))
	if err != nil {
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid local geocoder point id", err)
		return
	}
	userID, _ := userIDFromContext(context)
	point, err := handler.useCase.ApproveLocalPoint(context.Request.Context(), id, userID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, localPointToResponse(point))
}

// RejectLocalPoint godoc
// @Summary Reject local geocoder point
// @Tags admin-geocoder
// @Produce json
// @Param id path string true "Local point UUID"
// @Success 200 {object} LocalPointSuccessResponse
// @Failure 404 {object} response.Error
// @Router /admin/geocoder/local-points/{id}/reject [post]
func (handler *Handler) RejectLocalPoint(context *gin.Context) {
	id, err := uuid.Parse(context.Param("id"))
	if err != nil {
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid local geocoder point id", err)
		return
	}
	userID, _ := userIDFromContext(context)
	point, err := handler.useCase.RejectLocalPoint(context.Request.Context(), id, userID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, localPointToResponse(point))
}

// ExportPeliasCSV godoc
// @Summary Export trusted local points as Pelias CSV
// @Description Exports only platform-owned trusted local_geo_points; temporary external geocoder cache is never exported.
// @Tags admin-geocoder
// @Produce text/csv
// @Success 200 {file} file
// @Failure 500 {object} response.Error
// @Router /admin/geocoder/export/pelias-csv [get]
func (handler *Handler) ExportPeliasCSV(context *gin.Context) {
	points, err := handler.useCase.ExportTrustedLocalPoints(context.Request.Context())
	if err != nil {
		failByError(context, err)
		return
	}
	buffer := bytes.NewBuffer(nil)
	if err := exporter.WritePeliasCSV(buffer, points); err != nil {
		failByError(context, err)
		return
	}
	context.Header("Content-Disposition", `attachment; filename="pelias-local-points.csv"`)
	context.Data(http.StatusOK, "text/csv; charset=utf-8", buffer.Bytes())
}

func searchRequestFromQuery(context *gin.Context) (geodomain.SearchRequest, error) {
	query := context.Query("q")
	actorUserID, _ := userIDFromContext(context)
	var cityID *uuid.UUID
	if value := context.Query("city_id"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return geodomain.SearchRequest{}, err
		}
		cityID = &parsed
	}
	var focus *geodomain.Coordinates
	if context.Query("lat") != "" || context.Query("lon") != "" {
		latitude, err := strconv.ParseFloat(context.Query("lat"), 64)
		if err != nil {
			return geodomain.SearchRequest{}, err
		}
		longitude, err := strconv.ParseFloat(context.Query("lon"), 64)
		if err != nil {
			return geodomain.SearchRequest{}, err
		}
		coordinates, err := geodomain.NewCoordinates(latitude, longitude)
		if err != nil {
			return geodomain.SearchRequest{}, err
		}
		focus = &coordinates
	}
	limit := 10
	if value := context.Query("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return geodomain.SearchRequest{}, err
		}
		limit = parsed
	}
	return geodomain.SearchRequest{
		Query:       query,
		CityID:      cityID,
		Focus:       focus,
		ActorUserID: actorUserID,
		ActorRole:   roleStringFromContext(context),
		Limit:       limit,
	}, nil
}

func localPointFilterFromQuery(context *gin.Context) (geoservice.LocalPointFilter, error) {
	filter := geoservice.LocalPointFilter{TrustLevel: geodomain.TrustLevel(context.Query("trust_level")), Limit: 100}
	if value := context.Query("city_id"); value != "" {
		cityID, err := uuid.Parse(value)
		if err != nil {
			return geoservice.LocalPointFilter{}, err
		}
		filter.CityID = &cityID
	}
	if value := context.Query("limit"); value != "" {
		limit, err := strconv.Atoi(value)
		if err != nil {
			return geoservice.LocalPointFilter{}, err
		}
		filter.Limit = limit
	}
	return filter, nil
}

func failByError(context *gin.Context, err error) {
	switch {
	case errors.Is(err, geodomain.ErrInvalidQuery), errors.Is(err, geodomain.ErrInvalidCoordinates), errors.Is(err, geodomain.ErrInvalidConfidence), errors.Is(err, geodomain.ErrPromotionForbidden):
		fail(context, http.StatusBadRequest, response.CodeValidationError, "Invalid geocoder request", err)
	case errors.Is(err, geodomain.ErrPointNotFound):
		fail(context, http.StatusNotFound, response.CodeNotFound, "Local geocoder point not found", err)
	case errors.Is(err, geodomain.ErrExternalUnavailable):
		fail(context, http.StatusBadGateway, response.CodeInternalError, "External geocoder is unavailable", err)
	default:
		fail(context, http.StatusInternalServerError, response.CodeInternalError, "Internal error", err)
	}
}

func fail(context *gin.Context, status int, code response.ErrorCode, message string, err error) {
	if err != nil {
		_ = context.Error(err)
	}
	details := map[string]any(nil)
	if err != nil && status < http.StatusInternalServerError {
		details = map[string]any{"message": err.Error()}
	}
	response.Fail(context, status, code, message, details)
}

func userIDFromContext(context *gin.Context) (*uuid.UUID, bool) {
	value, exists := context.Get("user_id")
	if !exists {
		return nil, false
	}
	switch typedValue := value.(type) {
	case uuid.UUID:
		return &typedValue, true
	case string:
		parsed, err := uuid.Parse(typedValue)
		if err != nil {
			return nil, false
		}
		return &parsed, true
	default:
		return nil, false
	}
}

func roleStringFromContext(context *gin.Context) string {
	value, exists := context.Get("user_role")
	if !exists {
		return ""
	}
	switch typedValue := value.(type) {
	case string:
		return typedValue
	case fmt.Stringer:
		return typedValue.String()
	default:
		return fmt.Sprint(typedValue)
	}
}

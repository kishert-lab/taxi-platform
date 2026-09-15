package handler

import (
	"errors"
	"github.com/gin-gonic/gin"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	mapsapp "github.com/kishert-lab/taxi-platform/internal/maps"
	"github.com/kishert-lab/taxi-platform/internal/routing"
	"github.com/kishert-lab/taxi-platform/pkg/response"
	"net/http"
	"strconv"
)

type MapHandler struct{ service *mapsapp.Service }

func NewMapHandler(service *mapsapp.Service) *MapHandler { return &MapHandler{service} }
func (handler *MapHandler) RegisterRoutes(router gin.IRouter, passengerAuth gin.HandlerFunc) {
	router.GET("/public/map/config", handler.Configuration)
	router.POST("/map/routes", handler.Route)
	router.POST("/passenger/map/routes", passengerAuth, handler.Route)
	router.GET("/geocoder/reverse", handler.Reverse)
	router.GET("/passenger/address/reverse", passengerAuth, handler.Reverse)
}

// Configuration godoc
// @Summary Shared MapLibre configuration for all clients
// @Description Availability indicates a published, validated dataset, not a live SLA. Bounds are west,south,east,north; west > east crosses the antimeridian.
// @Tags maps
// @Produce json
// @Success 200 {object} MapConfigurationResponse
// @Failure 503 {object} response.Error
// @Router /public/map/config [get]
func (handler *MapHandler) Configuration(context *gin.Context) {
	result, err := handler.service.Configuration(context.Request.Context())
	if err != nil {
		mapFailure(context, err)
		return
	}
	response.OK(context, result)
}

// Route godoc
// @Summary Calculate a road route between two supplied points
// @Description Points use latitude/longitude fields. GeoJSON geometry uses [longitude,latitude]. Original points are preserved; snapped points are separate. No live traffic. Does not grant access to order or driver coordinates; obtain those through existing authorized order and WebSocket APIs.
// @Tags maps
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body mapsapp.RouteRequest true "Exactly two points; order model has no intermediate stops"
// @Success 200 {object} MapRouteResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 422 {object} response.Error
// @Failure 503 {object} response.Error
// @Router /map/routes [post]
// @Router /passenger/map/routes [post]
func (handler *MapHandler) Route(context *gin.Context) {
	var request mapsapp.RouteRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		mapFailure(context, routing.ErrInvalidRequest)
		return
	}
	result, err := handler.service.Route(context.Request.Context(), request)
	if err != nil {
		mapFailure(context, err)
		return
	}
	response.OK(context, result)
}
func mapFailure(context *gin.Context, err error) {
	status := http.StatusServiceUnavailable
	code := response.CodeInternalError
	message := "Map service unavailable"
	if errors.Is(err, routing.ErrInvalidRequest) {
		status = http.StatusBadRequest
		code = response.CodeValidationError
		message = "Invalid route coordinates or point count"
	}
	if errors.Is(err, routing.ErrOutsideCoverage) || errors.Is(err, routing.ErrRouteNotFound) {
		status = http.StatusUnprocessableEntity
		code = response.CodeValidationError
		message = "No road route within coverage and snapping radius"
	}
	if attached := context.Error(err); attached != nil {
		attached.Type = gin.ErrorTypePrivate
	}
	response.Fail(context, status, code, message, nil)
}

type MapConfigurationResponse struct {
	Data mapsapp.Configuration `json:"data"`
	Meta response.Meta         `json:"meta"`
}
type MapRouteResponse struct {
	Data routing.Route `json:"data"`
	Meta response.Meta `json:"meta"`
}

// Reverse godoc
// @Summary Reverse geocode without changing the selected pickup point
// @Tags maps
// @Security BearerAuth
// @Produce json
// @Param lat query number true "Latitude"
// @Param lon query number true "Longitude"
// @Success 200 {object} MapReverseResponse
// @Failure 400 {object} response.Error
// @Failure 503 {object} response.Error
// @Router /geocoder/reverse [get]
// @Router /passenger/address/reverse [get]
func (handler *MapHandler) Reverse(context *gin.Context) {
	latitude, err := strconv.ParseFloat(context.Query("lat"), 64)
	if err != nil {
		mapFailure(context, routing.ErrInvalidRequest)
		return
	}
	longitude, err := strconv.ParseFloat(context.Query("lon"), 64)
	if err != nil {
		mapFailure(context, routing.ErrInvalidRequest)
		return
	}
	result, err := handler.service.Reverse(context.Request.Context(), geodomain.Coordinates{Latitude: latitude, Longitude: longitude})
	if err != nil {
		mapFailure(context, err)
		return
	}
	response.OK(context, result)
}

type MapReverseResponse struct {
	Data mapsapp.ReverseResponse `json:"data"`
	Meta response.Meta           `json:"meta"`
}

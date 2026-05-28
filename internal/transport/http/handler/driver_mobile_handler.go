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

type DriverMobileUseCase interface {
	GetDriverProfile(ctx context.Context, driverID uuid.UUID) (dto.DriverProfileResponse, error)
	ListDriverCars(ctx context.Context, driverID uuid.UUID) (dto.TaxiParkCarsResponse, error)
	UpdateDriverProfile(ctx context.Context, driverID uuid.UUID, request dto.DriverProfilePatchRequest) (dto.DriverProfileResponse, error)
	UploadDriverProfilePhoto(ctx context.Context, driverID uuid.UUID, request dto.ProfilePhotoUploadRequest) (dto.ProfilePhotoUploadResponse, error)
	MarkDriverOnline(ctx context.Context, driverID uuid.UUID) (dto.DriverProfileResponse, error)
	MarkDriverOffline(ctx context.Context, driverID uuid.UUID) (dto.DriverProfileResponse, error)
	UpdateDriverLocation(ctx context.Context, driverID uuid.UUID, request dto.DriverLocationRequest) error
	UpdateDriverLocationBatch(ctx context.Context, driverID uuid.UUID, request dto.DriverLocationBatchRequest) error
	GetCurrentDriverOrder(ctx context.Context, driverID uuid.UUID) (dto.DriverOrderResponse, error)
	ListDriverOrderHistory(ctx context.Context, driverID uuid.UUID) (dto.DriverOrderHistoryResponse, error)
	ListDriverOrderOffers(ctx context.Context, driverID uuid.UUID) (dto.DriverOrderOffersResponse, error)
	GetDriverOrderRoute(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID) (dto.OrderRouteResponse, error)
	AcceptDriverOrder(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error)
	RejectDriverOrder(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID, request dto.RejectOrderRequest) error
	MarkDriverArriving(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error)
	MarkDriverArrived(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error)
	CancelDriverOrder(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID, reason string) (dto.DriverOrderResponse, error)
	StartDriverTrip(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID) (dto.DriverOrderResponse, error)
	CompleteDriverTrip(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID, request dto.CompleteOrderRequest) (dto.DriverOrderResponse, error)
	RatePassenger(ctx context.Context, driverID uuid.UUID, orderID uuid.UUID, request dto.RateOrderRequest) (dto.DriverOrderResponse, error)
}

type DriverMobileHandler struct {
	useCase DriverMobileUseCase
}

func NewDriverMobileHandler(useCase DriverMobileUseCase) *DriverMobileHandler {
	return &DriverMobileHandler{useCase: useCase}
}

func (handler *DriverMobileHandler) RegisterRoutes(router gin.IRouter) {
	driver := router.Group("/driver", middleware.RequireRole(domain.UserRoleDriver))
	driver.GET("/profile", handler.GetProfile)
	driver.GET("/cars", handler.ListCars)
	driver.PATCH("/profile", handler.UpdateProfile)
	driver.POST("/profile/photo", handler.UploadProfilePhoto)
	driver.POST("/online", handler.Online)
	driver.POST("/offline", handler.Offline)
	driver.POST("/location", handler.UpdateLocation)
	driver.POST("/location/batch", handler.UpdateLocationBatch)
	driver.GET("/orders/current", handler.CurrentOrder)
	driver.GET("/orders/history", handler.OrderHistory)
	driver.GET("/orders/offers", handler.OrderOffers)
	driver.GET("/orders/:id/route", handler.OrderRoute)
	driver.POST("/orders/:id/accept", handler.AcceptOrder)
	driver.POST("/orders/:id/reject", handler.RejectOrder)
	driver.POST("/orders/:id/arriving", handler.Arriving)
	driver.POST("/orders/:id/arrived", handler.Arrived)
	driver.POST("/orders/:id/cancel", handler.CancelOrder)
	driver.POST("/orders/:id/start", handler.StartTrip)
	driver.POST("/orders/:id/complete", handler.CompleteTrip)
	driver.POST("/orders/:id/rate-passenger", handler.RatePassenger)
}

// GetProfile godoc
// @Summary Get driver profile
// @Tags driver
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DriverProfileSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/profile [get]
func (handler *DriverMobileHandler) GetProfile(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.useCase.GetDriverProfile(context.Request.Context(), driverID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// ListCars godoc
// @Summary List cars attached to current driver
// @Tags driver
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TaxiParkCarsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/cars [get]
func (handler *DriverMobileHandler) ListCars(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.useCase.ListDriverCars(context.Request.Context(), driverID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// UpdateProfile godoc
// @Summary Update driver profile
// @Tags driver
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.DriverProfilePatchRequest true "Driver profile patch"
// @Success 200 {object} DriverProfileSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/profile [patch]
func (handler *DriverMobileHandler) UpdateProfile(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.DriverProfilePatchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid driver profile patch")
		return
	}

	result, err := handler.useCase.UpdateDriverProfile(context.Request.Context(), driverID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// UploadProfilePhoto godoc
// @Summary Upload driver profile photo
// @Description Attaches driver avatar/photo. The file must be jpeg, png, or webp and no larger than 5 MB.
// @Tags driver
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param photo formData file true "Profile photo"
// @Success 200 {object} ProfilePhotoUploadSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/profile/photo [post]
func (handler *DriverMobileHandler) UploadProfilePhoto(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	request, closeFile, ok := profilePhotoUploadRequest(context)
	if !ok {
		return
	}
	defer closeFile()

	result, err := handler.useCase.UploadDriverProfilePhoto(context.Request.Context(), driverID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// Online godoc
// @Summary Put driver online
// @Tags driver
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DriverProfileSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /driver/online [post]
func (handler *DriverMobileHandler) Online(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.useCase.MarkDriverOnline(context.Request.Context(), driverID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// Offline godoc
// @Summary Put driver offline
// @Tags driver
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DriverProfileSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/offline [post]
func (handler *DriverMobileHandler) Offline(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.useCase.MarkDriverOffline(context.Request.Context(), driverID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// UpdateLocation godoc
// @Summary Update driver location
// @Description Driver mobile clients must send no more than one update per two seconds unless using batch endpoint.
// @Tags driver-location
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.DriverLocationRequest true "Driver location"
// @Success 200 {object} AcceptedSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 429 {object} response.Error
// @Router /driver/location [post]
func (handler *DriverMobileHandler) UpdateLocation(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.DriverLocationRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid driver location request")
		return
	}

	if err := handler.useCase.UpdateDriverLocation(context.Request.Context(), driverID, request); err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, gin.H{"accepted": true})
}

// UpdateLocationBatch godoc
// @Summary Update driver locations in batch
// @Tags driver-location
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.DriverLocationBatchRequest true "Driver location batch"
// @Success 200 {object} AcceptedSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/location/batch [post]
func (handler *DriverMobileHandler) UpdateLocationBatch(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.DriverLocationBatchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid driver location batch")
		return
	}

	if err := handler.useCase.UpdateDriverLocationBatch(context.Request.Context(), driverID, request); err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, gin.H{"accepted": true})
}

// CurrentOrder godoc
// @Summary Get current driver order
// @Description Mobile reconnect sync endpoint. Response includes allowed_actions for driver UI.
// @Tags driver-orders
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DriverOrderSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /driver/orders/current [get]
func (handler *DriverMobileHandler) CurrentOrder(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.useCase.GetCurrentDriverOrder(context.Request.Context(), driverID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// OrderHistory godoc
// @Summary Get driver order history
// @Tags driver-orders
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DriverOrderHistorySuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/orders/history [get]
func (handler *DriverMobileHandler) OrderHistory(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.useCase.ListDriverOrderHistory(context.Request.Context(), driverID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// OrderOffers godoc
// @Summary List active order offers for current driver
// @Description Fallback sync endpoint for reconnect or weak internet. Returns Redis-backed active offers still available for accept/reject.
// @Tags driver-orders
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DriverOrderOffersSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/orders/offers [get]
func (handler *DriverMobileHandler) OrderOffers(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.useCase.ListDriverOrderOffers(context.Request.Context(), driverID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// OrderRoute godoc
// @Summary Get recorded route points for driver order
// @Description Route points are recorded from trip start until completion while driver location updates are received.
// @Tags driver-orders
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} DriverOrderRouteSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /driver/orders/{id}/route [get]
func (handler *DriverMobileHandler) OrderRoute(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	result, err := handler.useCase.GetDriverOrderRoute(context.Request.Context(), driverID, orderID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// AcceptOrder godoc
// @Summary Accept offered order
// @Tags driver-orders
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} DriverOrderSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /driver/orders/{id}/accept [post]
func (handler *DriverMobileHandler) AcceptOrder(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	result, err := handler.useCase.AcceptDriverOrder(context.Request.Context(), driverID, orderID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// RejectOrder godoc
// @Summary Reject offered order
// @Tags driver-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.RejectOrderRequest true "Reject reason"
// @Success 200 {object} AcceptedSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/orders/{id}/reject [post]
func (handler *DriverMobileHandler) RejectOrder(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	var request dto.RejectOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid reject request")
		return
	}

	if err := handler.useCase.RejectDriverOrder(context.Request.Context(), driverID, orderID, request); err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, gin.H{"accepted": true})
}

// Arriving godoc
// @Summary Mark driver is going to pickup point
// @Tags driver-orders
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} DriverOrderSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /driver/orders/{id}/arriving [post]
func (handler *DriverMobileHandler) Arriving(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	result, err := handler.useCase.MarkDriverArriving(context.Request.Context(), driverID, orderID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// Arrived godoc
// @Summary Mark driver arrived
// @Tags driver-orders
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} DriverOrderSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /driver/orders/{id}/arrived [post]
func (handler *DriverMobileHandler) Arrived(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	result, err := handler.useCase.MarkDriverArrived(context.Request.Context(), driverID, orderID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// CancelOrder godoc
// @Summary Cancel assigned order by driver
// @Description Driver can cancel before trip starts, for example after waiting too long.
// @Tags driver-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.CancelOrderRequest true "Cancellation reason"
// @Success 200 {object} DriverOrderSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /driver/orders/{id}/cancel [post]
func (handler *DriverMobileHandler) CancelOrder(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	var request dto.CancelOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid cancellation request")
		return
	}

	result, err := handler.useCase.CancelDriverOrder(context.Request.Context(), driverID, orderID, request.Reason)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// StartTrip godoc
// @Summary Start driver trip
// @Tags driver-orders
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Success 200 {object} DriverOrderSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /driver/orders/{id}/start [post]
func (handler *DriverMobileHandler) StartTrip(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	result, err := handler.useCase.StartDriverTrip(context.Request.Context(), driverID, orderID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// CompleteTrip godoc
// @Summary Complete driver trip
// @Tags driver-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.CompleteOrderRequest true "Completion request"
// @Success 200 {object} DriverOrderSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /driver/orders/{id}/complete [post]
func (handler *DriverMobileHandler) CompleteTrip(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	var request dto.CompleteOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid completion request")
		return
	}

	result, err := handler.useCase.CompleteDriverTrip(context.Request.Context(), driverID, orderID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// RatePassenger godoc
// @Summary Rate passenger after completed trip
// @Tags driver-orders
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Order ID"
// @Param request body dto.RateOrderRequest true "Passenger rating"
// @Success 200 {object} DriverOrderSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /driver/orders/{id}/rate-passenger [post]
func (handler *DriverMobileHandler) RatePassenger(context *gin.Context) {
	driverID, orderID, ok := driverOrderIDs(context)
	if !ok {
		return
	}

	var request dto.RateOrderRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger rating request")
		return
	}

	result, err := handler.useCase.RatePassenger(context.Request.Context(), driverID, orderID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

func driverOrderIDs(context *gin.Context) (uuid.UUID, uuid.UUID, bool) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return uuid.Nil, uuid.Nil, false
	}

	orderID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid order id")
		return uuid.Nil, uuid.Nil, false
	}

	return driverID, orderID, true
}

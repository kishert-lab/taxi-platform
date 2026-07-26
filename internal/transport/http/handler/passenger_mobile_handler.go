package handler

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/dto"
	geodomain "github.com/kishert-lab/taxi-platform/internal/geocoder/domain"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type PassengerProfileUseCase interface {
	CreatePassengerProfile(ctx context.Context, passengerID uuid.UUID, request dto.PassengerProfileRequest) (dto.PassengerProfileResponse, error)
	GetPassengerProfile(ctx context.Context, passengerID uuid.UUID) (dto.PassengerProfileResponse, error)
	UpdatePassengerProfile(ctx context.Context, passengerID uuid.UUID, request dto.PassengerProfilePatchRequest) (dto.PassengerProfileResponse, error)
	UploadPassengerProfilePhoto(ctx context.Context, passengerID uuid.UUID, request dto.ProfilePhotoUploadRequest) (dto.ProfilePhotoUploadResponse, error)
}

type PassengerOrderUseCase interface {
	EstimatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.OrderEstimateRequest) (dto.OrderEstimateResponse, error)
	CreatePassengerOrder(ctx context.Context, passengerID uuid.UUID, request dto.PassengerCreateOrderRequest) (dto.PassengerOrderResponse, error)
	GetCurrentPassengerOrder(ctx context.Context, passengerID uuid.UUID) (dto.PassengerOrderResponse, error)
	ListPassengerOrderHistory(ctx context.Context, passengerID uuid.UUID) (dto.OrderHistoryResponse, error)
	GetPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID) (dto.PassengerOrderResponse, error)
	CancelPassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.CancelOrderRequest) (dto.PassengerOrderResponse, error)
	RatePassengerOrder(ctx context.Context, passengerID uuid.UUID, orderID uuid.UUID, request dto.RateOrderRequest) (dto.PassengerOrderResponse, error)
}

type PassengerAddressUseCase interface {
	SearchPassengerAddresses(
		ctx context.Context,
		passengerID uuid.UUID,
		query string,
		cityID *uuid.UUID,
		focusLatitude *float64,
		focusLongitude *float64,
		limit int,
	) ([]geodomain.SearchResult, error)
}

type PassengerMobileHandler struct {
	profileUseCase PassengerProfileUseCase
}

func NewPassengerMobileHandler(profileUseCase PassengerProfileUseCase, _ PassengerOrderUseCase) *PassengerMobileHandler {
	return &PassengerMobileHandler{profileUseCase: profileUseCase}
}

func (handler *PassengerMobileHandler) RegisterRoutes(router gin.IRouter) {
	passenger := router.Group("/passenger", middleware.RequireAuthenticated())
	passenger.POST("/profile", handler.CreateProfile)
	passenger.GET("/profile", handler.GetProfile)
	passenger.PATCH("/profile", handler.UpdateProfile)
	passenger.POST("/profile/photo", handler.UploadProfilePhoto)
}

type PassengerAddressSearchResultResponse struct {
	ID              string                  `json:"id" example:"pelias:address:123"`
	LocalPointID    *uuid.UUID              `json:"local_point_id,omitempty" example:"11111111-1111-1111-1111-111111111111"`
	Provider        geodomain.Provider      `json:"provider" example:"pelias"`
	Name            string                  `json:"name" example:"Мира 8"`
	Address         string                  `json:"address" example:"Пермь, улица Мира, 8"`
	CityID          *uuid.UUID              `json:"city_id,omitempty" example:"22222222-2222-2222-2222-222222222222"`
	Coordinates     dto.CoordinatesResponse `json:"coordinates"`
	Confidence      float64                 `json:"confidence" example:"0.91"`
	TrustLevel      geodomain.TrustLevel    `json:"trust_level,omitempty" example:"trusted"`
	ExternalPlaceID string                  `json:"external_place_id,omitempty" example:"yandex:123"`
}

type PassengerAddressSearchResponse struct {
	Results []PassengerAddressSearchResultResponse `json:"results"`
}

type PassengerAddressSearchSuccessResponse struct {
	Data PassengerAddressSearchResponse `json:"data"`
	Meta response.Meta                  `json:"meta"`
}

// CreateProfile godoc
// @Summary Create passenger profile
// @Tags passenger
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.PassengerProfileRequest true "Passenger profile"
// @Success 201 {object} PassengerProfileSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/profile [post]
func (handler *PassengerMobileHandler) CreateProfile(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.PassengerProfileRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger profile request")
		return
	}

	result, err := handler.profileUseCase.CreatePassengerProfile(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.Created(context, result)
}

// GetProfile godoc
// @Summary Get passenger profile
// @Tags passenger
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PassengerProfileSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/profile [get]
func (handler *PassengerMobileHandler) GetProfile(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	result, err := handler.profileUseCase.GetPassengerProfile(context.Request.Context(), passengerID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// UpdateProfile godoc
// @Summary Update passenger profile
// @Tags passenger
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.PassengerProfilePatchRequest true "Passenger profile patch"
// @Success 200 {object} PassengerProfileSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/profile [patch]
func (handler *PassengerMobileHandler) UpdateProfile(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.PassengerProfilePatchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid passenger profile patch")
		return
	}

	result, err := handler.profileUseCase.UpdatePassengerProfile(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

// UploadProfilePhoto godoc
// @Summary Upload passenger profile photo
// @Description Attaches passenger avatar/photo. The file must be jpeg, png, or webp and no larger than 5 MB.
// @Tags passenger
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param photo formData file true "Profile photo"
// @Success 200 {object} ProfilePhotoUploadSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /passenger/profile/photo [post]
func (handler *PassengerMobileHandler) UploadProfilePhoto(context *gin.Context) {
	passengerID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	request, closeFile, ok := profilePhotoUploadRequest(context)
	if !ok {
		return
	}
	defer closeFile()

	result, err := handler.profileUseCase.UploadPassengerProfilePhoto(context.Request.Context(), passengerID, request)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, result)
}

func passengerAddressSearchQuery(context *gin.Context) (*uuid.UUID, *float64, *float64, int, error) {
	var cityID *uuid.UUID
	if value := context.Query("city_id"); value != "" {
		parsed, err := uuid.Parse(value)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		cityID = &parsed
	}

	var focusLatitude *float64
	var focusLongitude *float64
	if context.Query("lat") != "" || context.Query("lon") != "" {
		latitude, err := strconv.ParseFloat(context.Query("lat"), 64)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		longitude, err := strconv.ParseFloat(context.Query("lon"), 64)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		focusLatitude = &latitude
		focusLongitude = &longitude
	}

	limit := 10
	if value := context.Query("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		limit = parsed
	}

	return cityID, focusLatitude, focusLongitude, limit, nil
}

func passengerAddressSearchResults(results []geodomain.SearchResult) []PassengerAddressSearchResultResponse {
	responseItems := make([]PassengerAddressSearchResultResponse, 0, len(results))
	for _, result := range results {
		responseItems = append(responseItems, PassengerAddressSearchResultResponse{
			ID:              result.ID,
			LocalPointID:    result.LocalPointID,
			Provider:        result.Provider,
			Name:            result.Name,
			Address:         result.Address,
			CityID:          result.CityID,
			Coordinates:     dto.CoordinatesResponse{Latitude: result.Coordinates.Latitude, Longitude: result.Coordinates.Longitude},
			Confidence:      result.Confidence,
			TrustLevel:      result.TrustLevel,
			ExternalPlaceID: result.ExternalPlaceID,
		})
	}
	return responseItems
}

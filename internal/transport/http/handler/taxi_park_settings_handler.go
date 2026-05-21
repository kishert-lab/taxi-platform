package handler

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	taxiparkapp "github.com/kishert-lab/taxi-platform/internal/taxipark"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type TaxiParkSettingsUseCase interface {
	GetSettings(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkSettings, error)
	UpdateSettings(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkSettingsPatchRequest) (domain.TaxiParkSettings, error)
	ListTariffs(ctx context.Context, ownerUserID uuid.UUID) ([]domain.TaxiParkTariff, error)
	CreateTariff(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkTariffRequest) (domain.TaxiParkTariff, error)
	UpdateTariff(ctx context.Context, ownerUserID uuid.UUID, tariffID uuid.UUID, request dto.TaxiParkTariffPatchRequest) (domain.TaxiParkTariff, error)
	CreateDriver(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkCreateDriverRequest) (taxiparkapp.CreateDriverResult, error)
	UpdateDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, request dto.TaxiParkUpdateDriverRequest) (taxiparkapp.CreateDriverResult, error)
	BlockDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, reason string) error
	ArchiveDriver(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) error
	ListCars(ctx context.Context, ownerUserID uuid.UUID) ([]domain.Car, error)
	CreateCar(ctx context.Context, ownerUserID uuid.UUID, request dto.TaxiParkCarRequest) (domain.Car, error)
	UpdateCar(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID, request dto.TaxiParkCarPatchRequest) (domain.Car, error)
	VerifyCar(ctx context.Context, ownerUserID uuid.UUID, carID uuid.UUID) (domain.Car, error)
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
	taxiPark.POST("/drivers", handler.CreateDriver)
	taxiPark.PATCH("/drivers/:id", handler.UpdateDriver)
	taxiPark.POST("/drivers/:id/block", handler.BlockDriver)
	taxiPark.DELETE("/drivers/:id", handler.ArchiveDriver)
	taxiPark.GET("/cars", handler.ListCars)
	taxiPark.POST("/cars", handler.CreateCar)
	taxiPark.PATCH("/cars/:id", handler.UpdateCar)
	taxiPark.POST("/cars/:id/verify", handler.VerifyCar)
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

// CreateDriver godoc
// @Summary Create taxi park driver
// @Description Creates a driver account under the current taxi park by phone number. If password is omitted, backend generates a temporary password and returns it once.
// @Tags taxi-park-drivers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.TaxiParkCreateDriverRequest true "Driver account"
// @Success 201 {object} TaxiParkCreateDriverSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 409 {object} response.Error
// @Router /taxi-park/drivers [post]
func (handler *TaxiParkSettingsHandler) CreateDriver(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var request dto.TaxiParkCreateDriverRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid taxi park driver request")
		return
	}
	driver, err := handler.useCase.CreateDriver(context.Request.Context(), ownerUserID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.Created(context, taxiParkCreateDriverResponse(driver))
}

// UpdateDriver godoc
// @Summary Update taxi park driver
// @Tags taxi-park-drivers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Driver ID"
// @Param request body dto.TaxiParkUpdateDriverRequest true "Driver patch"
// @Success 200 {object} TaxiParkCreateDriverSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /taxi-park/drivers/{id} [patch]
func (handler *TaxiParkSettingsHandler) UpdateDriver(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	driverID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid driver id")
		return
	}
	var request dto.TaxiParkUpdateDriverRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid taxi park driver patch")
		return
	}
	driver, err := handler.useCase.UpdateDriver(context.Request.Context(), ownerUserID, driverID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, taxiParkCreateDriverResponse(driver))
}

// BlockDriver godoc
// @Summary Block taxi park driver
// @Tags taxi-park-drivers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Driver ID"
// @Param request body dto.TaxiParkBlockDriverRequest true "Block reason"
// @Success 200 {object} response.Success
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /taxi-park/drivers/{id}/block [post]
func (handler *TaxiParkSettingsHandler) BlockDriver(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	driverID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid driver id")
		return
	}
	var request dto.TaxiParkBlockDriverRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid taxi park driver block request")
		return
	}
	if err := handler.useCase.BlockDriver(context.Request.Context(), ownerUserID, driverID, request.Reason); err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, gin.H{"blocked": true})
}

// ArchiveDriver godoc
// @Summary Archive taxi park driver
// @Tags taxi-park-drivers
// @Produce json
// @Security BearerAuth
// @Param id path string true "Driver ID"
// @Success 200 {object} response.Success
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Failure 404 {object} response.Error
// @Router /taxi-park/drivers/{id} [delete]
func (handler *TaxiParkSettingsHandler) ArchiveDriver(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	driverID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid driver id")
		return
	}
	if err := handler.useCase.ArchiveDriver(context.Request.Context(), ownerUserID, driverID); err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, gin.H{"archived": true})
}

// ListCars godoc
// @Summary List taxi park cars
// @Tags taxi-park-cars
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TaxiParkCarsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/cars [get]
func (handler *TaxiParkSettingsHandler) ListCars(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	cars, err := handler.useCase.ListCars(context.Request.Context(), ownerUserID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, taxiParkCarsResponse(cars))
}

// CreateCar godoc
// @Summary Create taxi park car
// @Tags taxi-park-cars
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.TaxiParkCarRequest true "Car card"
// @Success 201 {object} TaxiParkCarSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/cars [post]
func (handler *TaxiParkSettingsHandler) CreateCar(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var request dto.TaxiParkCarRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid taxi park car request")
		return
	}
	car, err := handler.useCase.CreateCar(context.Request.Context(), ownerUserID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.Created(context, taxiParkCarResponse(car))
}

// UpdateCar godoc
// @Summary Update taxi park car
// @Tags taxi-park-cars
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Car ID"
// @Param request body dto.TaxiParkCarPatchRequest true "Car patch"
// @Success 200 {object} TaxiParkCarSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/cars/{id} [patch]
func (handler *TaxiParkSettingsHandler) UpdateCar(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	carID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid car id")
		return
	}
	var request dto.TaxiParkCarPatchRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid taxi park car patch")
		return
	}
	car, err := handler.useCase.UpdateCar(context.Request.Context(), ownerUserID, carID, request)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, taxiParkCarResponse(car))
}

// VerifyCar godoc
// @Summary Verify taxi park car
// @Tags taxi-park-cars
// @Produce json
// @Security BearerAuth
// @Param id path string true "Car ID"
// @Success 200 {object} TaxiParkCarSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/cars/{id}/verify [post]
func (handler *TaxiParkSettingsHandler) VerifyCar(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	carID, err := uuid.Parse(context.Param("id"))
	if err != nil {
		failValidation(context, "Invalid car id")
		return
	}
	car, err := handler.useCase.VerifyCar(context.Request.Context(), ownerUserID, carID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, taxiParkCarResponse(car))
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

func taxiParkCreateDriverResponse(driver taxiparkapp.CreateDriverResult) dto.TaxiParkCreateDriverResponse {
	name := driver.FirstName
	if driver.LastName != "" {
		if name != "" {
			name += " "
		}
		name += driver.LastName
	}

	return dto.TaxiParkCreateDriverResponse{
		UserID:                driver.UserID,
		DriverID:              driver.DriverID,
		TaxiParkID:            driver.TaxiParkID,
		Phone:                 driver.Phone,
		Email:                 driver.Email,
		Name:                  name,
		Status:                driver.Status,
		VerificationStatus:    driver.VerificationStatus,
		Rating:                driver.Rating,
		RatingsCount:          driver.RatingsCount,
		BirthDate:             driver.BirthDate,
		LicenseSeries:         driver.LicenseSeries,
		LicenseNumber:         driver.LicenseNumber,
		LicenseIssuedAt:       driver.LicenseIssuedAt,
		LicenseExpiresAt:      driver.LicenseExpiresAt,
		DrivingExperienceFrom: driver.DrivingExperienceFrom,
		IsVerified:            driver.IsVerified,
		TaxiParkComment:       driver.TaxiParkComment,
		GeneratedPassword:     driver.GeneratedPassword,
		PasswordGenerated:     driver.PasswordGenerated,
	}
}

func taxiParkCarsResponse(cars []domain.Car) dto.TaxiParkCarsResponse {
	responseBody := dto.TaxiParkCarsResponse{Cars: make([]dto.TaxiParkCarResponse, 0, len(cars))}
	for _, car := range cars {
		responseBody.Cars = append(responseBody.Cars, taxiParkCarResponse(car))
	}
	return responseBody
}

func taxiParkCarResponse(car domain.Car) dto.TaxiParkCarResponse {
	return dto.TaxiParkCarResponse{
		ID:                      car.ID,
		TaxiParkID:              car.TaxiParkID,
		PrimaryDriverID:         car.PrimaryDriverID,
		AttachedDriverIDs:       car.AttachedDriverIDs,
		Brand:                   car.Brand,
		Model:                   car.Model,
		Year:                    car.Year,
		PlateNumber:             car.PlateNumber,
		VIN:                     car.VIN,
		STS:                     car.STS,
		PTS:                     car.PTS,
		Color:                   car.Color,
		CarClass:                car.CarClass,
		VerificationStatus:      car.VerificationStatus,
		OwnerDetails:            car.OwnerDetails,
		OSAGOExpiresAt:          car.OSAGOExpiresAt,
		DiagnosticCardExpiresAt: car.DiagnosticCardExpiresAt,
		TaxiPermitNumber:        car.TaxiPermitNumber,
		RegionalRegistryNumber:  car.RegionalRegistryNumber,
		PermitRegion:            car.PermitRegion,
		PermitIssuedAt:          car.PermitIssuedAt,
		PermitExpiresAt:         car.PermitExpiresAt,
		IsActive:                car.IsActive,
		CreatedAt:               car.CreatedAt,
		UpdatedAt:               car.UpdatedAt,
	}
}

package handler

import (
	"context"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kishert-lab/taxi-platform/internal/domain"
	"github.com/kishert-lab/taxi-platform/internal/dto"
	"github.com/kishert-lab/taxi-platform/internal/finance"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

type FinanceUseCase interface {
	GetDriverBalance(ctx context.Context, driverID uuid.UUID) (domain.DriverBalance, error)
	ListDriverTransactions(ctx context.Context, driverID uuid.UUID, limit int) ([]domain.FinancialTransaction, error)
	GetTaxiParkBalance(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkBalance, error)
	ListTaxiParkDrivers(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.TaxiParkDriver, error)
	ListTaxiParkOrders(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.TaxiParkOrder, error)
	ListTaxiParkTransactions(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]domain.FinancialTransaction, error)
	GetAdminOverview(ctx context.Context, periodFrom time.Time, periodTo time.Time) (finance.AdminOverview, error)
}

type FinanceHandler struct {
	useCase FinanceUseCase
}

func NewFinanceHandler(useCase FinanceUseCase) *FinanceHandler {
	return &FinanceHandler{useCase: useCase}
}

func (handler *FinanceHandler) RegisterRoutes(router gin.IRouter) {
	driver := router.Group("/driver", middleware.RequireRole(domain.UserRoleDriver))
	driver.GET("/balance", handler.DriverBalance)
	driver.GET("/transactions", handler.DriverTransactions)

	taxiPark := router.Group("/taxi-park", middleware.RequireRole(domain.UserRoleTaxiPark))
	taxiPark.GET("/balance", handler.TaxiParkBalance)
	taxiPark.GET("/drivers", handler.TaxiParkDrivers)
	taxiPark.GET("/orders", handler.TaxiParkOrders)
	taxiPark.GET("/transactions", handler.TaxiParkTransactions)

	admin := router.Group("/admin", middleware.RequireRole(domain.UserRoleAdmin))
	admin.GET("/finance/overview", handler.AdminFinanceOverview)
}

// DriverBalance godoc
// @Summary Get driver balance
// @Tags finance-driver
// @Produce json
// @Security BearerAuth
// @Success 200 {object} DriverBalanceSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/balance [get]
func (handler *FinanceHandler) DriverBalance(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	balance, err := handler.useCase.GetDriverBalance(context.Request.Context(), driverID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.DriverBalanceFromDomain(balance))
}

// DriverTransactions godoc
// @Summary Get driver financial transactions
// @Tags finance-driver
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} FinancialTransactionsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/transactions [get]
func (handler *FinanceHandler) DriverTransactions(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	transactions, err := handler.useCase.ListDriverTransactions(context.Request.Context(), driverID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.FinancialTransactionsFromDomain(transactions))
}

// TaxiParkBalance godoc
// @Summary Get taxi park balance
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TaxiParkBalanceSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/balance [get]
func (handler *FinanceHandler) TaxiParkBalance(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	balance, err := handler.useCase.GetTaxiParkBalance(context.Request.Context(), ownerUserID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.TaxiParkBalanceFromDomain(balance))
}

// TaxiParkDrivers godoc
// @Summary Get taxi park drivers
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} TaxiParkDriversSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/drivers [get]
func (handler *FinanceHandler) TaxiParkDrivers(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	drivers, err := handler.useCase.ListTaxiParkDrivers(context.Request.Context(), ownerUserID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, taxiParkDriversResponse(drivers))
}

// TaxiParkOrders godoc
// @Summary Get taxi park orders
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} TaxiParkOrdersSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/orders [get]
func (handler *FinanceHandler) TaxiParkOrders(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	orders, err := handler.useCase.ListTaxiParkOrders(context.Request.Context(), ownerUserID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, taxiParkOrdersResponse(orders))
}

// TaxiParkTransactions godoc
// @Summary Get taxi park financial transactions
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} FinancialTransactionsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/transactions [get]
func (handler *FinanceHandler) TaxiParkTransactions(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	transactions, err := handler.useCase.ListTaxiParkTransactions(context.Request.Context(), ownerUserID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.FinancialTransactionsFromDomain(transactions))
}

// AdminFinanceOverview godoc
// @Summary Get admin finance overview
// @Tags finance-admin
// @Produce json
// @Security BearerAuth
// @Param from query string false "Period start RFC3339"
// @Param to query string false "Period end RFC3339"
// @Success 200 {object} AdminFinanceOverviewSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/finance/overview [get]
func (handler *FinanceHandler) AdminFinanceOverview(context *gin.Context) {
	periodFrom, periodTo, ok := financePeriodFromQuery(context)
	if !ok {
		return
	}

	overview, err := handler.useCase.GetAdminOverview(context.Request.Context(), periodFrom, periodTo)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, adminFinanceOverviewResponse(overview))
}

func limitFromQuery(context *gin.Context) int {
	limit, err := strconv.Atoi(context.DefaultQuery("limit", "50"))
	if err != nil {
		return 50
	}
	return limit
}

func financePeriodFromQuery(context *gin.Context) (time.Time, time.Time, bool) {
	var periodFrom time.Time
	var periodTo time.Time
	var err error
	if value := context.Query("from"); value != "" {
		periodFrom, err = time.Parse(time.RFC3339, value)
		if err != nil {
			failValidation(context, "Invalid from period")
			return time.Time{}, time.Time{}, false
		}
	}
	if value := context.Query("to"); value != "" {
		periodTo, err = time.Parse(time.RFC3339, value)
		if err != nil {
			failValidation(context, "Invalid to period")
			return time.Time{}, time.Time{}, false
		}
	}
	return periodFrom, periodTo, true
}

func taxiParkDriversResponse(drivers []finance.TaxiParkDriver) dto.TaxiParkDriversResponse {
	responseBody := dto.TaxiParkDriversResponse{Drivers: make([]dto.TaxiParkDriverResponse, 0, len(drivers))}
	for _, driver := range drivers {
		responseBody.Drivers = append(responseBody.Drivers, dto.TaxiParkDriverResponse{
			ID:        driver.ID,
			UserID:    driver.UserID,
			FullName:  driver.FullName,
			Status:    driver.Status,
			Rating:    driver.Rating,
			CreatedAt: driver.CreatedAt,
		})
	}
	return responseBody
}

func taxiParkOrdersResponse(orders []finance.TaxiParkOrder) dto.TaxiParkOrdersResponse {
	responseBody := dto.TaxiParkOrdersResponse{Orders: make([]dto.TaxiParkOrderResponse, 0, len(orders))}
	for _, order := range orders {
		responseBody.Orders = append(responseBody.Orders, dto.TaxiParkOrderResponse{
			ID:          order.ID,
			DriverID:    order.DriverID,
			Status:      order.Status,
			GrossAmount: dto.MoneyCentsFromDomain(order.GrossAmount),
			CreatedAt:   order.CreatedAt,
			CompletedAt: order.CompletedAt,
		})
	}
	return responseBody
}

func adminFinanceOverviewResponse(overview finance.AdminOverview) dto.AdminFinanceOverviewResponse {
	return dto.AdminFinanceOverviewResponse{
		CompletedOrdersRevenue: dto.MoneyCentsFromDomain(overview.CompletedOrdersRevenue),
		TotalCommissions:       dto.MoneyCentsFromDomain(overview.TotalCommissions),
		DriverPayouts:          dto.MoneyCentsFromDomain(overview.DriverPayouts),
		TaxiParkRevenue:        dto.MoneyCentsFromDomain(overview.TaxiParkRevenue),
		AverageCommission:      dto.MoneyCentsFromDomain(overview.AverageCommission),
		CompletedOrdersCount:   overview.CompletedOrdersCount,
		PeriodFrom:             overview.PeriodFrom,
		PeriodTo:               overview.PeriodTo,
	}
}

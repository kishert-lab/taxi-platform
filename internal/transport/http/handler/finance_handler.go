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
	ListDriverOrderFinances(ctx context.Context, userID uuid.UUID, limit int) ([]finance.OrderFinance, error)
	ListDriverPayouts(ctx context.Context, userID uuid.UUID, limit int) ([]finance.DriverPayout, error)
	ListDriverFinanceDocuments(ctx context.Context, userID uuid.UUID, limit int) ([]finance.FinanceDocument, error)
	GetTaxiParkBalance(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkBalance, error)
	GetTaxiParkFinanceSettings(ctx context.Context, ownerUserID uuid.UUID) (domain.TaxiParkFinanceSettings, error)
	UpdateTaxiParkDriverCommission(ctx context.Context, ownerUserID uuid.UUID, driverCommissionBasisPoints int32) (domain.TaxiParkFinanceSettings, error)
	GetTaxiParkOverview(ctx context.Context, ownerUserID uuid.UUID, periodFrom time.Time, periodTo time.Time) (finance.TaxiParkFinanceOverview, error)
	ListTaxiParkOrderFinances(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.OrderFinance, error)
	GetTaxiParkDriverBalance(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID) (domain.DriverBalance, error)
	ListTaxiParkDriverPayouts(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, limit int) ([]finance.DriverPayout, error)
	CreateDriverPayout(ctx context.Context, ownerUserID uuid.UUID, driverID uuid.UUID, input finance.CreateDriverPayoutInput) (finance.DriverPayout, error)
	ApproveDriverPayout(ctx context.Context, ownerUserID uuid.UUID, payoutID uuid.UUID) (finance.DriverPayout, error)
	MarkDriverPayoutPaid(ctx context.Context, ownerUserID uuid.UUID, payoutID uuid.UUID) (finance.DriverPayout, error)
	GetTaxiParkPlatformFeeDebt(ctx context.Context, ownerUserID uuid.UUID) (domain.Money, error)
	ListTaxiParkPlatformFeeAccruals(ctx context.Context, ownerUserID uuid.UUID, periodFrom time.Time, periodTo time.Time, limit int) ([]finance.OrderFinance, error)
	ListTaxiParkPlatformInvoices(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.PlatformInvoice, error)
	CreateTaxiParkPlatformInvoice(ctx context.Context, ownerUserID uuid.UUID, input finance.CreatePlatformInvoiceInput) (finance.PlatformInvoice, error)
	ListTaxiParkDocuments(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.FinanceDocument, error)
	ListTaxiParkDrivers(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.TaxiParkDriver, error)
	ListTaxiParkOrders(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]finance.TaxiParkOrder, error)
	ListTaxiParkTransactions(ctx context.Context, ownerUserID uuid.UUID, limit int) ([]domain.FinancialTransaction, error)
	GetAdminOverview(ctx context.Context, periodFrom time.Time, periodTo time.Time) (finance.AdminOverview, error)
	GetAdminTaxiParkOverview(ctx context.Context, taxiParkID uuid.UUID, periodFrom time.Time, periodTo time.Time) (finance.TaxiParkFinanceOverview, error)
	GetAdminTaxiParkPlatformFeeDebt(ctx context.Context, taxiParkID uuid.UUID) (domain.Money, error)
	ListAdminPlatformInvoices(ctx context.Context, limit int) ([]finance.PlatformInvoice, error)
	MarkAdminPlatformInvoicePaid(ctx context.Context, invoiceID uuid.UUID) (finance.PlatformInvoice, error)
	ListAdminDocuments(ctx context.Context, limit int) ([]finance.FinanceDocument, error)
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
	driver.GET("/finance/balance", handler.DriverBalance)
	driver.GET("/finance/orders", handler.DriverFinanceOrders)
	driver.GET("/finance/payouts", handler.DriverFinancePayouts)
	driver.GET("/finance/documents", handler.DriverFinanceDocuments)

	taxiPark := router.Group("/taxi-park", middleware.RequireRole(domain.UserRoleTaxiPark))
	taxiPark.GET("/balance", handler.TaxiParkBalance)
	taxiPark.GET("/finance/settings", handler.TaxiParkFinanceSettings)
	taxiPark.PUT("/finance/settings/driver-commission", handler.UpdateTaxiParkDriverCommission)
	taxiPark.GET("/finance/overview", handler.TaxiParkFinanceOverview)
	taxiPark.GET("/finance/orders", handler.TaxiParkFinanceOrders)
	taxiPark.GET("/finance/drivers/:driver_id/balance", handler.TaxiParkDriverBalance)
	taxiPark.GET("/finance/drivers/:driver_id/payouts", handler.TaxiParkDriverPayouts)
	taxiPark.POST("/finance/drivers/:driver_id/payouts", handler.CreateTaxiParkDriverPayout)
	taxiPark.POST("/finance/payouts/:payout_id/approve", handler.ApproveTaxiParkDriverPayout)
	taxiPark.POST("/finance/payouts/:payout_id/mark-paid", handler.MarkTaxiParkDriverPayoutPaid)
	taxiPark.GET("/finance/platform-fee-debt", handler.TaxiParkPlatformFeeDebt)
	taxiPark.GET("/finance/platform-fee-accruals", handler.TaxiParkPlatformFeeAccruals)
	taxiPark.GET("/finance/platform-invoices", handler.TaxiParkPlatformInvoices)
	taxiPark.POST("/finance/platform-invoices", handler.CreateTaxiParkPlatformInvoice)
	taxiPark.GET("/finance/documents", handler.TaxiParkFinanceDocuments)
	taxiPark.GET("/drivers", handler.TaxiParkDrivers)
	taxiPark.GET("/orders", handler.TaxiParkOrders)
	taxiPark.GET("/transactions", handler.TaxiParkTransactions)

	admin := router.Group("/admin", middleware.RequireRole(domain.UserRoleAdmin))
	admin.GET("/finance/overview", handler.AdminFinanceOverview)
	admin.GET("/finance/platform-overview", handler.AdminFinanceOverview)
	admin.GET("/finance/taxi-parks/:taxi_park_id/overview", handler.AdminTaxiParkFinanceOverview)
	admin.GET("/finance/taxi-parks/:taxi_park_id/platform-fee-debt", handler.AdminTaxiParkPlatformFeeDebt)
	admin.GET("/finance/platform-invoices", handler.AdminPlatformInvoices)
	admin.POST("/finance/platform-invoices/:invoice_id/mark-paid", handler.MarkAdminPlatformInvoicePaid)
	admin.GET("/finance/documents", handler.AdminFinanceDocuments)
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
// @Router /driver/finance/balance [get]
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

// DriverFinanceOrders godoc
// @Summary Get driver finance orders
// @Tags finance-driver
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} OrderFinancesSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/finance/orders [get]
func (handler *FinanceHandler) DriverFinanceOrders(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	orders, err := handler.useCase.ListDriverOrderFinances(context.Request.Context(), driverID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.OrderFinancesFromFinance(orders))
}

// DriverFinancePayouts godoc
// @Summary Get driver payouts
// @Tags finance-driver
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} DriverPayoutsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/finance/payouts [get]
func (handler *FinanceHandler) DriverFinancePayouts(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	payouts, err := handler.useCase.ListDriverPayouts(context.Request.Context(), driverID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.DriverPayoutsFromFinance(payouts))
}

// DriverFinanceDocuments godoc
// @Summary Get driver finance documents
// @Tags finance-driver
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} FinanceDocumentsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /driver/finance/documents [get]
func (handler *FinanceHandler) DriverFinanceDocuments(context *gin.Context) {
	driverID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	documents, err := handler.useCase.ListDriverFinanceDocuments(context.Request.Context(), driverID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.FinanceDocumentsFromFinance(documents))
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

// TaxiParkFinanceSettings godoc
// @Summary Get taxi park finance settings
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Success 200 {object} TaxiParkFinanceSettingsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/settings [get]
func (handler *FinanceHandler) TaxiParkFinanceSettings(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	settings, err := handler.useCase.GetTaxiParkFinanceSettings(context.Request.Context(), ownerUserID)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.TaxiParkFinanceSettingsFromDomain(settings))
}

// UpdateTaxiParkDriverCommission godoc
// @Summary Update taxi park driver commission
// @Tags finance-taxi-park
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.UpdateTaxiParkDriverCommissionRequest true "Driver commission update request"
// @Success 200 {object} TaxiParkFinanceSettingsSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/settings/driver-commission [put]
func (handler *FinanceHandler) UpdateTaxiParkDriverCommission(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}

	var request dto.UpdateTaxiParkDriverCommissionRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid request body")
		return
	}

	settings, err := handler.useCase.UpdateTaxiParkDriverCommission(context.Request.Context(), ownerUserID, request.DriverCommissionBasisPoints)
	if err != nil {
		failByError(context, err)
		return
	}

	response.OK(context, dto.TaxiParkFinanceSettingsFromDomain(settings))
}

// TaxiParkFinanceOverview godoc
// @Summary Get taxi park finance overview
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param from query string false "Period start RFC3339"
// @Param to query string false "Period end RFC3339"
// @Success 200 {object} TaxiParkFinanceOverviewSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/overview [get]
func (handler *FinanceHandler) TaxiParkFinanceOverview(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	periodFrom, periodTo, ok := financePeriodFromQuery(context)
	if !ok {
		return
	}
	overview, err := handler.useCase.GetTaxiParkOverview(context.Request.Context(), ownerUserID, periodFrom, periodTo)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.TaxiParkFinanceOverviewFromFinance(overview))
}

// TaxiParkFinanceOrders godoc
// @Summary Get taxi park finance orders
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} OrderFinancesSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/orders [get]
func (handler *FinanceHandler) TaxiParkFinanceOrders(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	orders, err := handler.useCase.ListTaxiParkOrderFinances(context.Request.Context(), ownerUserID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.OrderFinancesFromFinance(orders))
}

// TaxiParkDriverBalance godoc
// @Summary Get taxi park driver balance
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param driver_id path string true "Driver ID"
// @Success 200 {object} DriverBalanceSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/drivers/{driver_id}/balance [get]
func (handler *FinanceHandler) TaxiParkDriverBalance(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	driverID, ok := uuidParam(context, "driver_id")
	if !ok {
		return
	}
	balance, err := handler.useCase.GetTaxiParkDriverBalance(context.Request.Context(), ownerUserID, driverID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.DriverBalanceFromDomain(balance))
}

// TaxiParkDriverPayouts godoc
// @Summary Get taxi park driver payouts
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param driver_id path string true "Driver ID"
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} DriverPayoutsSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/drivers/{driver_id}/payouts [get]
func (handler *FinanceHandler) TaxiParkDriverPayouts(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	driverID, ok := uuidParam(context, "driver_id")
	if !ok {
		return
	}
	payouts, err := handler.useCase.ListTaxiParkDriverPayouts(context.Request.Context(), ownerUserID, driverID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.DriverPayoutsFromFinance(payouts))
}

// CreateTaxiParkDriverPayout godoc
// @Summary Create taxi park driver payout
// @Tags finance-taxi-park
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param driver_id path string true "Driver ID"
// @Param request body dto.CreateDriverPayoutRequest true "Create payout request"
// @Success 200 {object} DriverPayoutSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/drivers/{driver_id}/payouts [post]
func (handler *FinanceHandler) CreateTaxiParkDriverPayout(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	driverID, ok := uuidParam(context, "driver_id")
	if !ok {
		return
	}
	var request dto.CreateDriverPayoutRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid request body")
		return
	}
	payout, err := handler.useCase.CreateDriverPayout(context.Request.Context(), ownerUserID, driverID, finance.CreateDriverPayoutInput{
		AmountCents: request.AmountCents,
		PeriodFrom:  request.PeriodFrom,
		PeriodTo:    request.PeriodTo,
		Comment:     request.Comment,
	})
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.DriverPayoutFromFinance(payout))
}

// ApproveTaxiParkDriverPayout godoc
// @Summary Approve taxi park driver payout
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param payout_id path string true "Payout ID"
// @Success 200 {object} DriverPayoutSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/payouts/{payout_id}/approve [post]
func (handler *FinanceHandler) ApproveTaxiParkDriverPayout(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	payoutID, ok := uuidParam(context, "payout_id")
	if !ok {
		return
	}
	payout, err := handler.useCase.ApproveDriverPayout(context.Request.Context(), ownerUserID, payoutID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.DriverPayoutFromFinance(payout))
}

// MarkTaxiParkDriverPayoutPaid godoc
// @Summary Mark taxi park driver payout paid
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param payout_id path string true "Payout ID"
// @Success 200 {object} DriverPayoutSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/payouts/{payout_id}/mark-paid [post]
func (handler *FinanceHandler) MarkTaxiParkDriverPayoutPaid(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	payoutID, ok := uuidParam(context, "payout_id")
	if !ok {
		return
	}
	payout, err := handler.useCase.MarkDriverPayoutPaid(context.Request.Context(), ownerUserID, payoutID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.DriverPayoutFromFinance(payout))
}

// TaxiParkPlatformFeeDebt godoc
// @Summary Get taxi park platform fee debt
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Success 200 {object} PlatformFeeDebtSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/platform-fee-debt [get]
func (handler *FinanceHandler) TaxiParkPlatformFeeDebt(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	amount, err := handler.useCase.GetTaxiParkPlatformFeeDebt(context.Request.Context(), ownerUserID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.PlatformFeeDebtResponse{Amount: dto.MoneyCentsFromDomain(amount)})
}

// TaxiParkPlatformFeeAccruals godoc
// @Summary Get taxi park platform fee accruals
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param from query string false "Period start RFC3339"
// @Param to query string false "Period end RFC3339"
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} OrderFinancesSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/platform-fee-accruals [get]
func (handler *FinanceHandler) TaxiParkPlatformFeeAccruals(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	periodFrom, periodTo, ok := financePeriodFromQuery(context)
	if !ok {
		return
	}
	items, err := handler.useCase.ListTaxiParkPlatformFeeAccruals(context.Request.Context(), ownerUserID, periodFrom, periodTo, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.OrderFinancesFromFinance(items))
}

// TaxiParkPlatformInvoices godoc
// @Summary Get taxi park platform invoices
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} PlatformInvoicesSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/platform-invoices [get]
func (handler *FinanceHandler) TaxiParkPlatformInvoices(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	invoices, err := handler.useCase.ListTaxiParkPlatformInvoices(context.Request.Context(), ownerUserID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.PlatformInvoicesFromFinance(invoices))
}

// CreateTaxiParkPlatformInvoice godoc
// @Summary Create taxi park platform invoice
// @Tags finance-taxi-park
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body dto.CreatePlatformInvoiceRequest true "Create platform invoice request"
// @Success 200 {object} PlatformInvoiceSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/platform-invoices [post]
func (handler *FinanceHandler) CreateTaxiParkPlatformInvoice(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	var request dto.CreatePlatformInvoiceRequest
	if err := context.ShouldBindJSON(&request); err != nil {
		failValidation(context, "Invalid request body")
		return
	}
	invoice, err := handler.useCase.CreateTaxiParkPlatformInvoice(context.Request.Context(), ownerUserID, finance.CreatePlatformInvoiceInput{
		PeriodFrom: request.PeriodFrom,
		PeriodTo:   request.PeriodTo,
	})
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.PlatformInvoiceFromFinance(invoice))
}

// TaxiParkFinanceDocuments godoc
// @Summary Get taxi park finance documents
// @Tags finance-taxi-park
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} FinanceDocumentsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /taxi-park/finance/documents [get]
func (handler *FinanceHandler) TaxiParkFinanceDocuments(context *gin.Context) {
	ownerUserID, ok := userIDFromContext(context)
	if !ok {
		failUnauthorized(context, "User id is missing")
		return
	}
	documents, err := handler.useCase.ListTaxiParkDocuments(context.Request.Context(), ownerUserID, limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.FinanceDocumentsFromFinance(documents))
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
// @Router /admin/finance/platform-overview [get]
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

// AdminTaxiParkFinanceOverview godoc
// @Summary Get admin taxi park finance overview
// @Tags finance-admin
// @Produce json
// @Security BearerAuth
// @Param taxi_park_id path string true "Taxi park ID"
// @Param from query string false "Period start RFC3339"
// @Param to query string false "Period end RFC3339"
// @Success 200 {object} TaxiParkFinanceOverviewSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/finance/taxi-parks/{taxi_park_id}/overview [get]
func (handler *FinanceHandler) AdminTaxiParkFinanceOverview(context *gin.Context) {
	taxiParkID, ok := uuidParam(context, "taxi_park_id")
	if !ok {
		return
	}
	periodFrom, periodTo, ok := financePeriodFromQuery(context)
	if !ok {
		return
	}
	overview, err := handler.useCase.GetAdminTaxiParkOverview(context.Request.Context(), taxiParkID, periodFrom, periodTo)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.TaxiParkFinanceOverviewFromFinance(overview))
}

// AdminTaxiParkPlatformFeeDebt godoc
// @Summary Get admin taxi park platform fee debt
// @Tags finance-admin
// @Produce json
// @Security BearerAuth
// @Param taxi_park_id path string true "Taxi park ID"
// @Success 200 {object} PlatformFeeDebtSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/finance/taxi-parks/{taxi_park_id}/platform-fee-debt [get]
func (handler *FinanceHandler) AdminTaxiParkPlatformFeeDebt(context *gin.Context) {
	taxiParkID, ok := uuidParam(context, "taxi_park_id")
	if !ok {
		return
	}
	amount, err := handler.useCase.GetAdminTaxiParkPlatformFeeDebt(context.Request.Context(), taxiParkID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.PlatformFeeDebtResponse{Amount: dto.MoneyCentsFromDomain(amount)})
}

// AdminPlatformInvoices godoc
// @Summary Get admin platform invoices
// @Tags finance-admin
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} PlatformInvoicesSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/finance/platform-invoices [get]
func (handler *FinanceHandler) AdminPlatformInvoices(context *gin.Context) {
	invoices, err := handler.useCase.ListAdminPlatformInvoices(context.Request.Context(), limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.PlatformInvoicesFromFinance(invoices))
}

// MarkAdminPlatformInvoicePaid godoc
// @Summary Mark admin platform invoice paid
// @Tags finance-admin
// @Produce json
// @Security BearerAuth
// @Param invoice_id path string true "Invoice ID"
// @Success 200 {object} PlatformInvoiceSuccessResponse
// @Failure 400 {object} response.Error
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/finance/platform-invoices/{invoice_id}/mark-paid [post]
func (handler *FinanceHandler) MarkAdminPlatformInvoicePaid(context *gin.Context) {
	invoiceID, ok := uuidParam(context, "invoice_id")
	if !ok {
		return
	}
	invoice, err := handler.useCase.MarkAdminPlatformInvoicePaid(context.Request.Context(), invoiceID)
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.PlatformInvoiceFromFinance(invoice))
}

// AdminFinanceDocuments godoc
// @Summary Get admin finance documents
// @Tags finance-admin
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit, max 100"
// @Success 200 {object} FinanceDocumentsSuccessResponse
// @Failure 401 {object} response.Error
// @Failure 403 {object} response.Error
// @Router /admin/finance/documents [get]
func (handler *FinanceHandler) AdminFinanceDocuments(context *gin.Context) {
	documents, err := handler.useCase.ListAdminDocuments(context.Request.Context(), limitFromQuery(context))
	if err != nil {
		failByError(context, err)
		return
	}
	response.OK(context, dto.FinanceDocumentsFromFinance(documents))
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

func uuidParam(context *gin.Context, name string) (uuid.UUID, bool) {
	value, err := uuid.Parse(context.Param(name))
	if err != nil {
		failValidation(context, "Invalid "+name)
		return uuid.Nil, false
	}
	return value, true
}

func taxiParkDriversResponse(drivers []finance.TaxiParkDriver) dto.TaxiParkDriversResponse {
	responseBody := dto.TaxiParkDriversResponse{Drivers: make([]dto.TaxiParkDriverResponse, 0, len(drivers))}
	for _, driver := range drivers {
		responseBody.Drivers = append(responseBody.Drivers, dto.TaxiParkDriverResponse{
			ID:                            driver.ID,
			UserID:                        driver.UserID,
			Phone:                         driver.Phone,
			Email:                         driver.Email,
			FirstName:                     driver.FirstName,
			LastName:                      driver.LastName,
			FullName:                      driver.FullName,
			Status:                        driver.Status,
			VerificationStatus:            driver.VerificationStatus,
			Rating:                        driver.Rating,
			RatingsCount:                  driver.RatingsCount,
			BirthDate:                     driver.BirthDate,
			LicenseSeries:                 driver.LicenseSeries,
			LicenseNumber:                 driver.LicenseNumber,
			LicenseCategory:               driver.LicenseCategory,
			LicenseIssuedAt:               driver.LicenseIssuedAt,
			LicenseExpiresAt:              driver.LicenseExpiresAt,
			DrivingExperienceFrom:         driver.DrivingExperienceFrom,
			HasNoTaxiWorkRestrictions:     driver.HasNoTaxiWorkRestrictions,
			FederalLaw580Compliant:        driver.FederalLaw580Compliant,
			RegionalRequirementsCompliant: driver.RegionalRequirementsCompliant,
			MedicalCheckPassed:            driver.MedicalCheckPassed,
			PretripControlRequired:        driver.PretripControlRequired,
			PretripControlPassed:          driver.PretripControlPassed,
			NoTransportBan:                driver.NoTransportBan,
			VerificationCheckedAt:         driver.VerificationCheckedAt,
			VerificationCheckedBy:         driver.VerificationCheckedBy,
			IsVerified:                    driver.IsVerified,
			BlockedReason:                 driver.BlockedReason,
			TaxiParkComment:               driver.TaxiParkComment,
			CreatedAt:                     driver.CreatedAt,
			UpdatedAt:                     driver.UpdatedAt,
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

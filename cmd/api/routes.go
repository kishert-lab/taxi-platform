package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	financeapp "github.com/kishert-lab/taxi-platform/internal/finance"
	legalapp "github.com/kishert-lab/taxi-platform/internal/legal"
	"github.com/kishert-lab/taxi-platform/internal/repository"
	"github.com/kishert-lab/taxi-platform/internal/service"
	taxiparkapp "github.com/kishert-lab/taxi-platform/internal/taxipark"
	"github.com/kishert-lab/taxi-platform/internal/transport/http/handler"
)

type applicationRoutes struct {
	auth       *handler.AuthHandler
	mobileAuth *handler.MobileAuthHandler
	order      *handler.OrderHandler
	passenger  *handler.PassengerMobileHandler
	driver     *handler.DriverMobileHandler
	finance    *handler.FinanceHandler
	taxiPark   *handler.TaxiParkSettingsHandler
	legal      *handler.LegalHandler
	websocket  *handler.WebSocketHandler
}

func newApplicationRoutes(postgresPool *pgxpool.Pool, logger *zap.Logger) applicationRoutes {
	unavailableUseCase := service.NewUnavailableUseCase()

	financeRepository := repository.NewPostgresFinanceRepository(postgresPool)
	financeService := financeapp.NewService(financeRepository, logger)
	taxiParkSettingsRepository := repository.NewPostgresTaxiParkSettingsRepository(postgresPool)
	taxiParkSettingsService := taxiparkapp.NewService(taxiParkSettingsRepository)
	legalRepository := repository.NewPostgresLegalRepository(postgresPool)
	legalService := legalapp.NewService(legalRepository)

	return applicationRoutes{
		auth:       handler.NewAuthHandler(unavailableUseCase),
		mobileAuth: handler.NewMobileAuthHandler(unavailableUseCase),
		order:      handler.NewOrderHandler(unavailableUseCase),
		passenger:  handler.NewPassengerMobileHandler(unavailableUseCase, unavailableUseCase),
		driver:     handler.NewDriverMobileHandler(unavailableUseCase),
		finance:    handler.NewFinanceHandler(financeService),
		taxiPark:   handler.NewTaxiParkSettingsHandler(taxiParkSettingsService),
		legal:      handler.NewLegalHandler(legalService),
		websocket:  handler.NewWebSocketHandler(unavailableUseCase),
	}
}

func (routes applicationRoutes) Register(api gin.IRouter) {
	routes.auth.RegisterRoutes(api)
	routes.mobileAuth.RegisterRoutes(api)
	routes.order.RegisterRoutes(api)
	routes.passenger.RegisterRoutes(api)
	routes.driver.RegisterRoutes(api)
	routes.finance.RegisterRoutes(api)
	routes.taxiPark.RegisterRoutes(api)
	routes.legal.RegisterRoutes(api)
	routes.websocket.RegisterRoutes(api)
}

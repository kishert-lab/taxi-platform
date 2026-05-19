package main

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/configs"
	authapp "github.com/kishert-lab/taxi-platform/internal/auth"
	financeapp "github.com/kishert-lab/taxi-platform/internal/finance"
	legalapp "github.com/kishert-lab/taxi-platform/internal/legal"
	"github.com/kishert-lab/taxi-platform/internal/repository"
	"github.com/kishert-lab/taxi-platform/internal/security"
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

func newApplicationRoutes(postgresPool *pgxpool.Pool, config *configs.Config, logger *zap.Logger) applicationRoutes {
	unavailableUseCase := service.NewUnavailableUseCase()

	userRepository := repository.NewPostgresUserRepository(postgresPool)
	taxiParkRepository := repository.NewPostgresTaxiParkRepository(postgresPool)
	verificationCodeRepository := repository.NewPostgresVerificationCodeRepository(postgresPool)
	refreshTokenRepository := repository.NewPostgresRefreshTokenRepository(postgresPool)
	userConsentEventRepository := repository.NewPostgresUserConsentEventRepository(postgresPool)
	passwordHasher := security.NewBCryptPasswordHasher(config.Security.BCryptCost)
	codeHasher := security.NewBCryptCodeHasher(config.Security.BCryptCost)
	codeGenerator := security.NewNumericCodeGenerator()
	registrationService := authapp.NewRegistrationService(authapp.NewRegistrationServiceParams{
		UserRepository:             userRepository,
		TaxiParkRepository:         taxiParkRepository,
		VerificationCodeRepository: verificationCodeRepository,
		UserConsentEventRepository: userConsentEventRepository,
		SMSProvider:                authapp.NewLoggingSMSProvider(logger),
		EmailProvider:              authapp.NewLoggingEmailProvider(logger),
		PasswordHasher:             passwordHasher,
		CodeHasher:                 codeHasher,
		CodeGenerator:              codeGenerator,
		Logger:                     logger,
		Config: authapp.RegistrationServiceConfig{
			PhoneCodeTTL:    config.Auth.PhoneCodeTTL,
			EmailCodeTTL:    config.Auth.EmailCodeTTL,
			MaxCodeAttempts: config.Auth.MaxCodeAttempts,
		},
	})
	mobileAuthService := authapp.NewMobileService(authapp.NewMobileServiceParams{
		UserRepository:             userRepository,
		VerificationCodeRepository: verificationCodeRepository,
		RefreshTokenRepository:     refreshTokenRepository,
		CodeGenerator:              codeGenerator,
		CodeHasher:                 codeHasher,
		PasswordHasher:             passwordHasher,
		TokenManager: authapp.NewTokenManager(authapp.TokenManagerConfig{
			AccessSecret:  config.JWT.AccessSecret,
			RefreshSecret: config.JWT.RefreshSecret,
			Issuer:        config.JWT.Issuer,
			AccessTTL:     config.JWT.AccessTTL,
			RefreshTTL:    config.JWT.RefreshTTL,
		}),
		Logger:          logger,
		CodeLength:      config.Auth.CodeLength,
		PhoneCodeTTL:    config.Auth.PhoneCodeTTL,
		EmailCodeTTL:    config.Auth.EmailCodeTTL,
		MaxCodeAttempts: config.Auth.MaxCodeAttempts,
	})
	financeRepository := repository.NewPostgresFinanceRepository(postgresPool)
	financeService := financeapp.NewService(financeRepository, logger)
	taxiParkSettingsRepository := repository.NewPostgresTaxiParkSettingsRepository(postgresPool)
	taxiParkSettingsService := taxiparkapp.NewService(taxiParkSettingsRepository, passwordHasher)
	legalRepository := repository.NewPostgresLegalRepository(postgresPool)
	legalService := legalapp.NewService(legalRepository)

	return applicationRoutes{
		auth:       handler.NewAuthHandler(registrationService),
		mobileAuth: handler.NewMobileAuthHandler(mobileAuthService),
		order:      handler.NewOrderHandler(unavailableUseCase),
		passenger:  handler.NewPassengerMobileHandler(unavailableUseCase, unavailableUseCase),
		driver:     handler.NewDriverMobileHandler(unavailableUseCase),
		finance:    handler.NewFinanceHandler(financeService),
		taxiPark:   handler.NewTaxiParkSettingsHandler(taxiParkSettingsService),
		legal:      handler.NewLegalHandler(legalService),
		websocket:  handler.NewWebSocketHandler(mobileAuthService, config.HTTP.CORS.AllowedOrigins),
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

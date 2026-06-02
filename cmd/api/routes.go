package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/configs"
	authapp "github.com/kishert-lab/taxi-platform/internal/auth"
	chatapp "github.com/kishert-lab/taxi-platform/internal/chat"
	dispatchapp "github.com/kishert-lab/taxi-platform/internal/dispatch"
	driverapp "github.com/kishert-lab/taxi-platform/internal/driver"
	financeapp "github.com/kishert-lab/taxi-platform/internal/finance"
	geoapp "github.com/kishert-lab/taxi-platform/internal/geo"
	dadataclient "github.com/kishert-lab/taxi-platform/internal/geocoder/client/dadata"
	peliasclient "github.com/kishert-lab/taxi-platform/internal/geocoder/client/pelias"
	yandexclient "github.com/kishert-lab/taxi-platform/internal/geocoder/client/yandex"
	geocoderhandler "github.com/kishert-lab/taxi-platform/internal/geocoder/handler"
	geocoderrepository "github.com/kishert-lab/taxi-platform/internal/geocoder/repository"
	geocoderservice "github.com/kishert-lab/taxi-platform/internal/geocoder/service"
	legalapp "github.com/kishert-lab/taxi-platform/internal/legal"
	redisinfra "github.com/kishert-lab/taxi-platform/internal/redis"
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
	chat       *handler.ChatHandler
	geocoder   *geocoderhandler.Handler
	websocket  *handler.WebSocketHandler
	dispatch   *dispatchapp.Worker
}

func newApplicationRoutes(postgresPool *pgxpool.Pool, redisClient *goredis.Client, config *configs.Config, logger *zap.Logger) applicationRoutes {
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
	driverLocationRepository := repository.NewPostgresDriverLocationRepository(postgresPool)
	driverLocationThrottle := redisinfra.NewLocationThrottle(redisClient)
	driverLocationService := geoapp.NewLocationService(driverLocationRepository, driverLocationThrottle)
	dispatchOrderRepository := repository.NewPostgresDispatchOrderRepository(postgresPool)
	dispatchQueue := redisinfra.NewDispatchQueue(redisClient)
	dispatchTimeoutQueue := redisinfra.NewDispatchTimeoutQueue(redisClient)
	dispatchStateStore := redisinfra.NewDispatchStateStore(redisClient)
	offerStore := redisinfra.NewOfferStore(redisClient)
	lockManager := redisinfra.NewLockManager(redisClient)
	realtimeGateway := redisinfra.NewRealtimeGateway(redisClient, postgresPool)
	dispatchConfig := dispatchConfigFromApplication(config.Dispatch)
	dispatchService := dispatchapp.NewService(dispatchapp.NewServiceParams{
		OrderRepository:        dispatchOrderRepository,
		DriverSearchRepository: repository.NewPostgresDriverSearchRepository(postgresPool),
		DriverStateRepository:  repository.NewPostgresDriverStateRepository(postgresPool),
		OfferStore:             offerStore,
		DispatchStateStore:     dispatchStateStore,
		TaskQueue:              dispatchQueue,
		TimeoutQueue:           dispatchTimeoutQueue,
		LockManager:            lockManager,
		RealtimeGateway:        realtimeGateway,
		Logger:                 logger,
		Config:                 dispatchConfig,
	})
	dispatchWorker := dispatchapp.NewWorker(dispatchapp.NewWorkerParams{
		Service:            dispatchService,
		TaskQueue:          dispatchQueue,
		TimeoutQueue:       dispatchTimeoutQueue,
		RecoveryRepository: dispatchOrderRepository,
		Logger:             logger,
		Config:             dispatchConfig,
	})
	driverMobileRepository := repository.NewPostgresDriverMobileRepository(postgresPool)
	driverPresenceStore := redisinfra.NewDriverPresenceStore(redisClient)
	financeRepository := repository.NewPostgresFinanceRepository(postgresPool)
	financeService := financeapp.NewService(financeRepository, logger)
	driverMobileService := driverapp.NewMobileServiceWithDispatch(driverMobileRepository, driverPresenceStore, driverLocationService, dispatchService, logger, realtimeGateway).
		WithFinanceProcessor(financeService)
	taxiParkSettingsRepository := repository.NewPostgresTaxiParkSettingsRepository(postgresPool)
	taxiParkSettingsService := taxiparkapp.NewServiceWithDispatch(taxiParkSettingsRepository, passwordHasher, dispatchService, realtimeGateway).
		WithFinanceProcessor(financeService)
	legalRepository := repository.NewPostgresLegalRepository(postgresPool)
	legalService := legalapp.NewService(legalRepository)
	chatRepository := repository.NewPostgresChatRepository(postgresPool)
	chatService := chatapp.NewService(chatRepository, realtimeGateway, logger)
	geocoderRepository := geocoderrepository.NewPostgresRepository(postgresPool)
	geocoderService := geocoderservice.New(
		geocoderRepository,
		peliasclient.New(config.Geocoder.PeliasURL, &http.Client{Timeout: 3 * time.Second}),
		yandexclient.New(config.Geocoder.YandexAPIKey, &http.Client{Timeout: 4 * time.Second}),
		dadataclient.New(config.Geocoder.DaDataAPIKey, config.Geocoder.DaDataSecretKey, config.Geocoder.DaDataURL, config.Geocoder.DaDataSuggestURL, &http.Client{Timeout: 4 * time.Second}),
		logger,
		geocoderservice.Config{
			YandexEnabled:             config.Geocoder.YandexEnabled,
			DaDataEnabled:             config.Geocoder.DaDataEnabled,
			ExternalCacheTTL:          time.Duration(config.Geocoder.ExternalCacheTTLDays) * 24 * time.Hour,
			PeliasConfidenceThreshold: config.Geocoder.PeliasConfidenceThreshold,
			DefaultLimit:              10,
		},
	)

	return applicationRoutes{
		auth:       handler.NewAuthHandler(registrationService),
		mobileAuth: handler.NewMobileAuthHandler(mobileAuthService),
		order:      handler.NewOrderHandler(unavailableUseCase),
		passenger:  handler.NewPassengerMobileHandler(unavailableUseCase, unavailableUseCase),
		driver:     handler.NewDriverMobileHandler(driverMobileService),
		finance:    handler.NewFinanceHandler(financeService),
		taxiPark:   handler.NewTaxiParkSettingsHandler(taxiParkSettingsService),
		legal:      handler.NewLegalHandler(legalService),
		chat:       handler.NewChatHandler(chatService),
		geocoder:   geocoderhandler.New(geocoderService),
		websocket:  handler.NewWebSocketHandler(mobileAuthService, config.HTTP.CORS.AllowedOrigins, redisClient),
		dispatch:   dispatchWorker,
	}
}

func dispatchConfigFromApplication(config configs.DispatchConfig) dispatchapp.Config {
	return dispatchapp.Config{
		InitialRadiusMeters:  config.InitialRadiusMeters,
		MaxRadiusMeters:      config.MaxRadiusMeters,
		RadiusStepMeters:     config.RadiusStepMeters,
		RadiusAttemptsMeters: config.RadiusAttemptsMeters,
		MaxDriversPerOffer:   config.MaxDriversPerOffer,
		DriverLocationMaxAge: config.DriverLocationMaxAge,
		OfferTTL:             config.OfferTTL,
		AcceptLockTTL:        config.AcceptLockTTL,
		WorkerPollTimeout:    config.WorkerPollTimeout,
		RecoveryInterval:     config.RecoveryInterval,
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
	routes.chat.RegisterRoutes(api)
	routes.geocoder.RegisterRoutes(api)
	routes.websocket.RegisterRoutes(api)
}

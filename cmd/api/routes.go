package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/configs"
	auditapp "github.com/kishert-lab/taxi-platform/internal/audit"
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
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	passengerapp "github.com/kishert-lab/taxi-platform/internal/passenger"
	pushapp "github.com/kishert-lab/taxi-platform/internal/push"
	redisinfra "github.com/kishert-lab/taxi-platform/internal/redis"
	"github.com/kishert-lab/taxi-platform/internal/repository"
	scheduledapp "github.com/kishert-lab/taxi-platform/internal/scheduled"
	"github.com/kishert-lab/taxi-platform/internal/security"
	"github.com/kishert-lab/taxi-platform/internal/service"
	taxiparkapp "github.com/kishert-lab/taxi-platform/internal/taxipark"
	"github.com/kishert-lab/taxi-platform/internal/transport/http/handler"
)

type applicationRoutes struct {
	auth                    *handler.AuthHandler
	mobileAuth              *handler.MobileAuthHandler
	passengerAuth           *handler.PassengerAuthHandler
	passengerMe             *handler.PassengerMeHandler
	passengerAddress        *handler.PassengerAddressHandler
	passengerCarClasses     *handler.PassengerCarClassHandler
	passengerOrders         *handler.PassengerOrdersHandler
	passengerPush           *handler.PassengerPushHandler
	passengerAuthMiddleware gin.HandlerFunc
	order                   *handler.OrderHandler
	passenger               *handler.PassengerMobileHandler
	driver                  *handler.DriverMobileHandler
	finance                 *handler.FinanceHandler
	taxiPark                *handler.TaxiParkSettingsHandler
	legal                   *handler.LegalHandler
	chat                    *handler.ChatHandler
	geocoder                *geocoderhandler.Handler
	websocket               *handler.WebSocketHandler
	requestAudit            *auditapp.Service
	dispatch                *dispatchapp.Worker
	scheduled               *scheduledapp.Worker
}

func newApplicationRoutes(postgresPool *pgxpool.Pool, redisClient *goredis.Client, config *configs.Config, logger *zap.Logger) applicationRoutes {
	unavailableUseCase := service.NewUnavailableUseCase()

	userRepository := repository.NewPostgresUserRepository(postgresPool)
	passengerRepository := repository.NewPostgresPassengerRepository(postgresPool)
	transportRequestLogRepository := repository.NewPostgresTransportRequestLogRepository(postgresPool)
	taxiParkRepository := repository.NewPostgresTaxiParkRepository(postgresPool)
	verificationCodeRepository := repository.NewPostgresVerificationCodeRepository(postgresPool)
	refreshTokenRepository := repository.NewPostgresRefreshTokenRepository(postgresPool)
	passengerAuthCodeRepository := repository.NewPostgresPassengerAuthCodeRepository(postgresPool)
	passengerRefreshTokenRepository := repository.NewPostgresPassengerRefreshTokenRepository(postgresPool)
	passengerPushTokenRepository := repository.NewPostgresPassengerPushTokenRepository(postgresPool)
	userConsentEventRepository := repository.NewPostgresUserConsentEventRepository(postgresPool)
	passwordHasher := security.NewBCryptPasswordHasher(config.Security.BCryptCost)
	codeHasher := security.NewBCryptCodeHasher(config.Security.BCryptCost)
	codeGenerator := security.NewNumericCodeGenerator()
	requestAuditService := auditapp.NewService(transportRequestLogRepository, logger)
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
	passengerTokenManager := passengerapp.NewTokenManager(passengerapp.TokenManagerConfig{
		AccessSecret:  config.JWT.AccessSecret,
		RefreshSecret: config.JWT.RefreshSecret,
		Issuer:        config.JWT.Issuer,
		AccessTTL:     config.JWT.AccessTTL,
		RefreshTTL:    config.JWT.RefreshTTL,
	})
	passengerAuthService := passengerapp.NewAuthService(passengerapp.AuthServiceParams{
		Repository:             passengerRepository,
		AuthCodeRepository:     passengerAuthCodeRepository,
		RefreshTokenRepository: passengerRefreshTokenRepository,
		SMSService:             passengerapp.NewLoggingSMSService(logger),
		CodeGenerator:          codeGenerator,
		CodeHasher:             codeHasher,
		TokenManager:           passengerTokenManager,
		Logger:                 logger,
		CodeLength:             config.Auth.CodeLength,
		CodeTTL:                config.Auth.PhoneCodeTTL,
		MaxCodeAttempts:        config.Auth.MaxCodeAttempts,
		DevCode:                config.Auth.PassengerDevCode,
	})
	passengerProfileService := passengerapp.NewProfileService(passengerRepository)
	passengerPushTokenService := passengerapp.NewPushTokenService(passengerPushTokenRepository)
	passengerAuthMiddleware := middleware.AuthenticatePassengerAccessToken(passengerTokenManager, passengerRepository)
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
	passengerPushService := buildPassengerPushService(config, logger, passengerPushTokenRepository)
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
		PassengerNotifier:      newDispatchPassengerNotifier(passengerPushService),
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
		WithFinanceProcessor(financeService).
		WithPassengerNotifier(newDriverPassengerNotifier(passengerPushService))
	taxiParkSettingsRepository := repository.NewPostgresTaxiParkSettingsRepository(postgresPool)
	taxiParkSettingsService := taxiparkapp.NewServiceWithDispatch(taxiParkSettingsRepository, passwordHasher, dispatchService, realtimeGateway).
		WithFinanceProcessor(financeService)
	scheduledRepository := repository.NewPostgresScheduledOrderRepository(postgresPool)
	scheduledService := scheduledapp.NewService(
		scheduledRepository,
		dispatchService,
		realtimeGateway,
		logger,
		scheduledapp.Config{BatchSize: config.Scheduled.BatchSize},
	)
	scheduledWorker := scheduledapp.NewWorker(scheduledService, logger, scheduledapp.WorkerConfig{
		Enabled:     config.Scheduled.WorkerEnabled,
		TickSeconds: config.Scheduled.TickSeconds,
	})
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
			YandexCacheTTL:            time.Duration(config.Geocoder.YandexCacheTTLDays) * 24 * time.Hour,
			DaDataCacheTTL:            time.Duration(config.Geocoder.DaDataCacheTTLDays) * 24 * time.Hour,
			PeliasCacheTTL:            time.Duration(config.Geocoder.PeliasCacheTTLDays) * 24 * time.Hour,
			PeliasConfidenceThreshold: config.Geocoder.PeliasConfidenceThreshold,
			DefaultLimit:              10,
		},
	)
	passengerAddressSearchService := passengerapp.NewAddressSearchService(geocoderService)
	passengerOrderRepository := repository.NewPostgresPassengerOrderRepository(postgresPool)
	passengerOrderService := passengerapp.NewOrderService(passengerRepository, passengerOrderRepository, dispatchService, geocoderService)

	return applicationRoutes{
		auth:                    handler.NewAuthHandler(registrationService),
		mobileAuth:              handler.NewMobileAuthHandler(mobileAuthService),
		passengerAuth:           handler.NewPassengerAuthHandler(passengerAuthService),
		passengerMe:             handler.NewPassengerMeHandler(passengerProfileService),
		passengerAddress:        handler.NewPassengerAddressHandler(passengerAddressSearchService),
		passengerCarClasses:     handler.NewPassengerCarClassHandler(passengerOrderService),
		passengerOrders:         handler.NewPassengerOrdersHandler(passengerOrderService),
		passengerPush:           handler.NewPassengerPushHandler(passengerPushTokenService),
		passengerAuthMiddleware: passengerAuthMiddleware,
		order:                   handler.NewOrderHandler(unavailableUseCase),
		passenger:               handler.NewPassengerMobileHandler(unavailableUseCase, unavailableUseCase),
		driver:                  handler.NewDriverMobileHandler(driverMobileService),
		finance:                 handler.NewFinanceHandler(financeService),
		taxiPark:                handler.NewTaxiParkSettingsHandler(taxiParkSettingsService),
		legal:                   handler.NewLegalHandler(legalService),
		chat:                    handler.NewChatHandler(chatService),
		geocoder:                geocoderhandler.New(geocoderService),
		websocket:               handler.NewWebSocketHandler(mobileAuthService, requestAuditService, config.HTTP.CORS.AllowedOrigins, redisClient),
		requestAudit:            requestAuditService,
		dispatch:                dispatchWorker,
		scheduled:               scheduledWorker,
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
	routes.passengerAuth.RegisterRoutes(api, routes.passengerAuthMiddleware)
	routes.passengerMe.RegisterRoutes(api, routes.passengerAuthMiddleware)
	routes.passengerAddress.RegisterRoutes(api, routes.passengerAuthMiddleware)
	if routes.passengerCarClasses != nil {
		routes.passengerCarClasses.RegisterRoutes(api, routes.passengerAuthMiddleware)
	}
	routes.passengerOrders.RegisterRoutes(api, routes.passengerAuthMiddleware)
	routes.passengerPush.RegisterRoutes(api, routes.passengerAuthMiddleware)
	routes.order.RegisterRoutes(api)
	routes.passenger.RegisterRoutes(api)
	routes.driver.RegisterRoutes(api)
	routes.finance.RegisterRoutes(api)
	routes.taxiPark.RegisterRoutes(api)
	routes.legal.RegisterRoutes(api)
	routes.chat.RegisterRoutes(api, routes.passengerAuthMiddleware)
	routes.geocoder.RegisterRoutes(api)
	routes.websocket.RegisterRoutes(api)
}

func buildPassengerPushService(
	config *configs.Config,
	logger *zap.Logger,
	tokenRepository *repository.PostgresPassengerPushTokenRepository,
) *pushapp.Service {
	if config == nil || !config.Push.Enabled {
		return nil
	}

	projectID := strings.TrimSpace(config.Push.FirebaseProjectID)
	if projectID == "" && strings.TrimSpace(config.Push.FirebaseGoogleServicesFile) != "" {
		derivedProjectID, err := pushapp.ProjectIDFromGoogleServicesFile(config.Push.FirebaseGoogleServicesFile)
		if err != nil {
			logger.Warn("derive firebase project id from google-services", zap.Error(err))
		} else {
			projectID = derivedProjectID
		}
	}

	provider, err := pushapp.NewFirebaseProvider(projectID, config.Push.FirebaseCredentialsFile)
	if err != nil {
		logger.Warn("initialize firebase push provider", zap.Error(err))
		return nil
	}

	return pushapp.NewService(pushapp.ServiceParams{
		Provider:   provider,
		Repository: tokenRepository,
		Logger:     logger,
		Enabled:    true,
	})
}

type dispatchPassengerNotifier struct {
	service *pushapp.Service
}

func newDispatchPassengerNotifier(service *pushapp.Service) *dispatchPassengerNotifier {
	if service == nil {
		return nil
	}
	return &dispatchPassengerNotifier{service: service}
}

func (notifier *dispatchPassengerNotifier) NotifyPassenger(ctx context.Context, passengerID uuid.UUID, notification dispatchapp.PassengerNotification) error {
	return notifier.service.NotifyPassenger(ctx, passengerID, pushapp.Notification{
		Title: notification.Title,
		Body:  notification.Body,
		Data:  notification.Data,
	})
}

type driverPassengerNotifier struct {
	service *pushapp.Service
}

func newDriverPassengerNotifier(service *pushapp.Service) *driverPassengerNotifier {
	if service == nil {
		return nil
	}
	return &driverPassengerNotifier{service: service}
}

func (notifier *driverPassengerNotifier) NotifyPassenger(ctx context.Context, passengerID uuid.UUID, notification driverapp.PassengerNotification) error {
	return notifier.service.NotifyPassenger(ctx, passengerID, pushapp.Notification{
		Title: notification.Title,
		Body:  notification.Body,
		Data:  notification.Data,
	})
}

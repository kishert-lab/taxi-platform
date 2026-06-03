package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"

	"github.com/kishert-lab/taxi-platform/configs"
	_ "github.com/kishert-lab/taxi-platform/docs"
	authapp "github.com/kishert-lab/taxi-platform/internal/auth"
	"github.com/kishert-lab/taxi-platform/internal/database"
	geoservice "github.com/kishert-lab/taxi-platform/internal/geo"
	"github.com/kishert-lab/taxi-platform/internal/middleware"
	redisclient "github.com/kishert-lab/taxi-platform/internal/redis"
	"github.com/kishert-lab/taxi-platform/internal/repository"
	"github.com/kishert-lab/taxi-platform/pkg/logger"
	"github.com/kishert-lab/taxi-platform/pkg/response"
)

// @title Taxi Platform API
// @version 0.1.0
// @description Production-grade taxi backend API for small cities and rural areas.
// @termsOfService http://swagger.io/terms/
// @contact.name Taxi Platform API Support
// @contact.email support@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @schemes http https
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter the token with the `Bearer ` prefix, for example: Bearer eyJhbGciOi...
func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	config, err := configs.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(config.Logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		_ = log.Sync()
	}()

	postgresPool, err := database.NewPostgres(ctx, config.Database)
	if err != nil {
		log.Fatal("connect postgres", zap.Error(err))
	}
	defer postgresPool.Close()

	redisClient, err := redisclient.New(ctx, config.Redis)
	if err != nil {
		log.Fatal("connect redis", zap.Error(err))
	}
	defer func() {
		if closeErr := redisClient.Close(); closeErr != nil {
			log.Error("close redis", zap.Error(closeErr))
		}
	}()

	routes := newApplicationRoutes(postgresPool, redisClient, config, log)
	if routes.dispatch != nil {
		go func() {
			if dispatchErr := routes.dispatch.Run(ctx); dispatchErr != nil {
				log.Error("dispatch worker stopped", zap.Error(dispatchErr))
			}
		}()
	}
	staleDriverService := geoservice.NewLocationService(
		repository.NewPostgresDriverLocationRepository(postgresPool),
		redisclient.NewLocationThrottle(redisClient),
	)
	startStaleDriverOfflineWorker(ctx, staleDriverService, log)

	router := buildRouter(config, log, routes)
	server := &http.Server{
		Addr:              config.Server.Address(),
		Handler:           router,
		ReadHeaderTimeout: config.Server.ReadHeaderTimeout,
		ReadTimeout:       config.Server.ReadTimeout,
		WriteTimeout:      config.Server.WriteTimeout,
		IdleTimeout:       config.Server.IdleTimeout,
	}

	go func() {
		log.Info("starting http server", zap.String("address", server.Addr))
		if listenErr := server.ListenAndServe(); listenErr != nil && !errors.Is(listenErr, http.ErrServerClosed) {
			log.Fatal("http server failed", zap.Error(listenErr))
		}
	}()

	<-ctx.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), config.Server.ShutdownTimeout)
	defer cancel()

	log.Info("shutting down http server")
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Error("shutdown http server", zap.Error(err))
	}
}

type staleDriverOfflineMarker interface {
	MarkStaleDriversOffline(ctx context.Context, limit int) (int, error)
}

func startStaleDriverOfflineWorker(ctx context.Context, marker staleDriverOfflineMarker, log *zap.Logger) {
	const (
		interval = 10 * time.Second
		limit    = 500
	)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			count, err := marker.MarkStaleDriversOffline(ctx, limit)
			if err != nil && !errors.Is(err, context.Canceled) {
				log.Error("mark stale drivers offline", zap.Error(err))
			}
			if count > 0 {
				log.Info("stale drivers marked offline", zap.Int("count", count))
			}

			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func debugHTTPLoggingEnabled(config *configs.Config) bool {
	if config == nil {
		return false
	}
	return config.Server.Mode == gin.DebugMode || strings.EqualFold(config.App.Env, "debug") || strings.EqualFold(config.Logger.Level, "debug")
}

func buildRouter(config *configs.Config, log *zap.Logger, routes applicationRoutes) *gin.Engine {
	if config.Server.Mode == gin.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.DebugRequestLogger(log, debugHTTPLoggingEnabled(config)))
	router.Use(middleware.JSONCharset())
	router.Use(gin.Recovery())

	router.Use(cors.New(cors.Config{
		AllowOrigins:     config.HTTP.CORS.AllowedOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Authorization", "Content-Type", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	router.GET("/health/live", func(context *gin.Context) {
		handleLiveHealth(context)
	})
	router.GET("/health/ready", func(context *gin.Context) {
		handleReadyHealth(context)
	})
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	api := router.Group("/api/v1")
	api.GET("/health/live", handleLiveHealth)
	api.GET("/health/ready", handleReadyHealth)
	api.GET("/health", func(context *gin.Context) {
		handleAPIHealth(context, config)
	})
	api.Use(middleware.AuthenticateAccessToken(
		authapp.NewTokenManager(authapp.TokenManagerConfig{
			AccessSecret:  config.JWT.AccessSecret,
			RefreshSecret: config.JWT.RefreshSecret,
			Issuer:        config.JWT.Issuer,
			AccessTTL:     config.JWT.AccessTTL,
			RefreshTTL:    config.JWT.RefreshTTL,
		}),
		"/api/v1/auth",
		"/api/v1/public",
		"/api/v1/ws",
	))
	routes.Register(api)

	router.NoRoute(func(context *gin.Context) {
		log.Debug("route not found", zap.String("path", context.Request.URL.Path))
		response.Fail(context, http.StatusNotFound, response.CodeNotFound, "Route not found", nil)
	})

	return router
}

type healthResponse struct {
	Status string `json:"status" example:"ok"`
}

type apiHealthResponse struct {
	Status  string `json:"status" example:"ok"`
	Service string `json:"service" example:"taxi-platform"`
}

type errorResponse struct {
	Error string `json:"error" example:"route not found"`
}

// handleLiveHealth godoc
// @Summary Liveness probe
// @Description Returns process liveness status for orchestrators.
// @Tags health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health/live [get]
func handleLiveHealth(context *gin.Context) {
	context.JSON(http.StatusOK, healthResponse{Status: "ok"})
}

// handleReadyHealth godoc
// @Summary Readiness probe
// @Description Returns readiness status for traffic routing.
// @Tags health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health/ready [get]
func handleReadyHealth(context *gin.Context) {
	context.JSON(http.StatusOK, healthResponse{Status: "ok"})
}

// handleAPIHealth godoc
// @Summary API health
// @Description Returns public API health status.
// @Tags health
// @Produce json
// @Success 200 {object} apiHealthResponse
// @Router /health [get]
func handleAPIHealth(context *gin.Context, config *configs.Config) {
	context.JSON(http.StatusOK, apiHealthResponse{Status: "ok", Service: config.App.Name})
}

package configs

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App      AppConfig      `mapstructure:"app"`
	Server   ServerConfig   `mapstructure:"server"`
	HTTP     HTTPConfig     `mapstructure:"http"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Logger   LoggerConfig   `mapstructure:"logger"`
	Security SecurityConfig `mapstructure:"security"`
	Dispatch DispatchConfig `mapstructure:"dispatch"`
}

type AppConfig struct {
	Name        string `mapstructure:"name"`
	Env         string `mapstructure:"env"`
	Version     string `mapstructure:"version"`
	ServiceID   string `mapstructure:"service_id"`
	Timezone    string `mapstructure:"timezone"`
	MetricsPath string `mapstructure:"metrics_path"`
}

type ServerConfig struct {
	Host              string        `mapstructure:"host"`
	Port              int           `mapstructure:"port"`
	Mode              string        `mapstructure:"mode"`
	ReadHeaderTimeout time.Duration `mapstructure:"read_header_timeout"`
	ReadTimeout       time.Duration `mapstructure:"read_timeout"`
	WriteTimeout      time.Duration `mapstructure:"write_timeout"`
	IdleTimeout       time.Duration `mapstructure:"idle_timeout"`
	ShutdownTimeout   time.Duration `mapstructure:"shutdown_timeout"`
}

func (config ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", config.Host, config.Port)
}

type HTTPConfig struct {
	CORS CORSConfig `mapstructure:"cors"`
}

type CORSConfig struct {
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}

type DatabaseConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	User            string        `mapstructure:"user"`
	Password        string        `mapstructure:"password"`
	Name            string        `mapstructure:"name"`
	SSLMode         string        `mapstructure:"ssl_mode"`
	MaxConns        int32         `mapstructure:"max_conns"`
	MinConns        int32         `mapstructure:"min_conns"`
	MaxConnLifetime time.Duration `mapstructure:"max_conn_lifetime"`
	MaxConnIdleTime time.Duration `mapstructure:"max_conn_idle_time"`
}

func (config DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		config.User,
		config.Password,
		config.Host,
		config.Port,
		config.Name,
		config.SSLMode,
	)
}

type RedisConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	Password     string        `mapstructure:"password"`
	DB           int           `mapstructure:"db"`
	DialTimeout  time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

func (config RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", config.Host, config.Port)
}

type JWTConfig struct {
	AccessSecret      string        `mapstructure:"access_secret"`
	RefreshSecret     string        `mapstructure:"refresh_secret"`
	AccessTTL         time.Duration `mapstructure:"access_ttl"`
	RefreshTTL        time.Duration `mapstructure:"refresh_ttl"`
	DeviceAccessTTL   time.Duration `mapstructure:"device_access_ttl"`
	SigningMethod     string        `mapstructure:"signing_method"`
	Issuer            string        `mapstructure:"issuer"`
	RefreshCookieName string        `mapstructure:"refresh_cookie_name"`
}

type AuthConfig struct {
	PhoneCodeTTL       time.Duration `mapstructure:"phone_code_ttl"`
	EmailCodeTTL       time.Duration `mapstructure:"email_code_ttl"`
	CodeLength         int           `mapstructure:"code_length"`
	MaxCodeAttempts    int           `mapstructure:"max_code_attempts"`
	ResendCooldown     time.Duration `mapstructure:"resend_cooldown"`
	RequireEmailVerify bool          `mapstructure:"require_email_verify"`
	RequirePhoneVerify bool          `mapstructure:"require_phone_verify"`
}

type LoggerConfig struct {
	Level       string `mapstructure:"level"`
	Encoding    string `mapstructure:"encoding"`
	Development bool   `mapstructure:"development"`
}

type SecurityConfig struct {
	BCryptCost            int           `mapstructure:"bcrypt_cost"`
	RateLimitPerMinute    int           `mapstructure:"rate_limit_per_minute"`
	LoginAttemptTTL       time.Duration `mapstructure:"login_attempt_ttl"`
	MaxLoginAttempts      int           `mapstructure:"max_login_attempts"`
	RefreshTokenReuseLock time.Duration `mapstructure:"refresh_token_reuse_lock"`
}

type DispatchConfig struct {
	InitialRadiusMeters  int           `mapstructure:"initial_radius_meters"`
	MaxRadiusMeters      int           `mapstructure:"max_radius_meters"`
	RadiusStepMeters     int           `mapstructure:"radius_step_meters"`
	RadiusAttemptsMeters []int         `mapstructure:"radius_attempts_meters"`
	MaxDriversPerOffer   int           `mapstructure:"max_drivers_per_offer"`
	DriverLocationMaxAge time.Duration `mapstructure:"driver_location_max_age"`
	OfferTTL             time.Duration `mapstructure:"offer_ttl"`
	AcceptLockTTL        time.Duration `mapstructure:"accept_lock_ttl"`
	WorkerPollTimeout    time.Duration `mapstructure:"worker_poll_timeout"`
	RecoveryInterval     time.Duration `mapstructure:"recovery_interval"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath("/app/configs")
	viper.SetEnvPrefix("TAXI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	if err := bindEnvironmentAliases(); err != nil {
		return nil, err
	}
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("read config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

func bindEnvironmentAliases() error {
	aliases := map[string][]string{
		"app.env":                        {"APP_ENV"},
		"server.port":                    {"HTTP_PORT"},
		"database.host":                  {"DB_HOST"},
		"database.port":                  {"DB_PORT"},
		"database.user":                  {"DB_USER"},
		"database.password":              {"DB_PASSWORD"},
		"database.name":                  {"DB_NAME"},
		"redis.host":                     {"REDIS_HOST"},
		"redis.port":                     {"REDIS_PORT"},
		"jwt.access_secret":              {"JWT_SECRET"},
		"jwt.refresh_secret":             {"JWT_REFRESH_SECRET"},
		"dispatch.initial_radius_meters": {"DISPATCH_INITIAL_RADIUS"},
		"dispatch.max_radius_meters":     {"DISPATCH_MAX_RADIUS"},
	}

	for key, envNames := range aliases {
		bindArgs := append([]string{key}, envNames...)
		if err := viper.BindEnv(bindArgs...); err != nil {
			return fmt.Errorf("bind env %s: %w", key, err)
		}
	}

	if value := os.Getenv("DISPATCH_TIMEOUT_SECONDS"); value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse DISPATCH_TIMEOUT_SECONDS: %w", err)
		}
		viper.Set("dispatch.offer_ttl", fmt.Sprintf("%ds", seconds))
	}

	return nil
}

func (config Config) Validate() error {
	if config.Server.Port <= 0 {
		return errors.New("server port must be positive")
	}
	if config.Database.Host == "" {
		return errors.New("database host is required")
	}
	if config.Database.User == "" {
		return errors.New("database user is required")
	}
	if config.Database.Name == "" {
		return errors.New("database name is required")
	}
	if config.Redis.Host == "" {
		return errors.New("redis host is required")
	}
	if len(config.JWT.AccessSecret) < 32 {
		return errors.New("jwt access secret must contain at least 32 characters")
	}
	if len(config.JWT.RefreshSecret) < 32 {
		return errors.New("jwt refresh secret must contain at least 32 characters")
	}
	if config.JWT.AccessSecret == config.JWT.RefreshSecret {
		return errors.New("jwt access and refresh secrets must be different")
	}
	return nil
}

func setDefaults() {
	viper.SetDefault("app.name", "taxi-platform")
	viper.SetDefault("app.env", "local")
	viper.SetDefault("app.version", "0.1.0")
	viper.SetDefault("app.service_id", "taxi-api")
	viper.SetDefault("app.timezone", "Asia/Yekaterinburg")
	viper.SetDefault("app.metrics_path", "/metrics")
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_header_timeout", "5s")
	viper.SetDefault("server.read_timeout", "15s")
	viper.SetDefault("server.write_timeout", "15s")
	viper.SetDefault("server.idle_timeout", "60s")
	viper.SetDefault("server.shutdown_timeout", "15s")
	viper.SetDefault("http.cors.allowed_origins", []string{"http://localhost:3000", "http://localhost:5173"})
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.user", "taxi")
	viper.SetDefault("database.password", "taxi_password")
	viper.SetDefault("database.name", "taxi")
	viper.SetDefault("database.ssl_mode", "disable")
	viper.SetDefault("database.max_conns", 20)
	viper.SetDefault("database.min_conns", 2)
	viper.SetDefault("database.max_conn_lifetime", "1h")
	viper.SetDefault("database.max_conn_idle_time", "30m")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)
	viper.SetDefault("redis.dial_timeout", "5s")
	viper.SetDefault("redis.read_timeout", "3s")
	viper.SetDefault("redis.write_timeout", "3s")
	viper.SetDefault("jwt.access_ttl", "15m")
	viper.SetDefault("jwt.refresh_ttl", "720h")
	viper.SetDefault("jwt.device_access_ttl", "24h")
	viper.SetDefault("jwt.signing_method", "HS256")
	viper.SetDefault("jwt.issuer", "taxi-platform")
	viper.SetDefault("jwt.refresh_cookie_name", "taxi_refresh_token")
	viper.SetDefault("auth.phone_code_ttl", "5m")
	viper.SetDefault("auth.email_code_ttl", "15m")
	viper.SetDefault("auth.code_length", 6)
	viper.SetDefault("auth.max_code_attempts", 5)
	viper.SetDefault("auth.resend_cooldown", "60s")
	viper.SetDefault("auth.require_email_verify", true)
	viper.SetDefault("auth.require_phone_verify", true)
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.encoding", "json")
	viper.SetDefault("logger.development", false)
	viper.SetDefault("security.bcrypt_cost", 12)
	viper.SetDefault("security.rate_limit_per_minute", 120)
	viper.SetDefault("security.login_attempt_ttl", "15m")
	viper.SetDefault("security.max_login_attempts", 5)
	viper.SetDefault("security.refresh_token_reuse_lock", "30s")
	viper.SetDefault("dispatch.initial_radius_meters", 1000)
	viper.SetDefault("dispatch.max_radius_meters", 10000)
	viper.SetDefault("dispatch.radius_step_meters", 1000)
	viper.SetDefault("dispatch.radius_attempts_meters", []int{1000, 3000, 5000, 10000})
	viper.SetDefault("dispatch.max_drivers_per_offer", 5)
	viper.SetDefault("dispatch.driver_location_max_age", "30s")
	viper.SetDefault("dispatch.offer_ttl", "15s")
	viper.SetDefault("dispatch.accept_lock_ttl", "30s")
	viper.SetDefault("dispatch.worker_poll_timeout", "5s")
	viper.SetDefault("dispatch.recovery_interval", "30s")
}

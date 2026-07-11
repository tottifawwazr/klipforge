// Package config loads and validates runtime configuration from environment
// variables. It intentionally has no dependency on a .env file so the same
// binary behaves consistently in containers and production runtimes.
package config

import (
	"fmt"
	"math"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const ServiceName = "klipforge-api"

type Config struct {
	Environment string
	Version     string
	LogLevel    string
	HTTP        HTTPConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	Health      HealthConfig
	Auth        AuthConfig
}

type AuthConfig struct {
	JWTSecret          string
	JWTIssuer          string
	JWTAudience        string
	AccessTokenTTL     time.Duration
	RefreshTokenTTL    time.Duration
	RefreshTokenPepper string
	BcryptCost         int
	CookieName         string
	CookieDomain       string
	CookieSecure       bool
	LoginRateLimit     int64
	RegisterRateLimit  int64
	RefreshRateLimit   int64
	RateLimitWindow    time.Duration
}

type HTTPConfig struct {
	Address             string
	AllowedOrigins      []string
	AllowCredentials    bool
	ReadHeaderTimeout   time.Duration
	ReadTimeout         time.Duration
	WriteTimeout        time.Duration
	IdleTimeout         time.Duration
	RequestTimeout      time.Duration
	ShutdownTimeout     time.Duration
	MaxHeaderBytes      int
	MaxRequestBodyBytes int64
}

type DatabaseConfig struct {
	URL                   string
	MaxConnections        int32
	MinConnections        int32
	MaxConnectionLifetime time.Duration
	MaxConnectionIdleTime time.Duration
	HealthCheckPeriod     time.Duration
}

type RedisConfig struct {
	URL                string
	PoolSize           int
	MinIdleConnections int
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
}

type HealthConfig struct {
	DependencyTimeout time.Duration
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	return load(os.LookupEnv)
}

type lookupFunc func(string) (string, bool)

type envReader struct {
	lookup lookupFunc
}

func load(lookup lookupFunc) (Config, error) {
	r := envReader{lookup: lookup}

	readHeaderTimeout, err := r.positiveDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	readTimeout, err := r.positiveDuration("HTTP_READ_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}
	writeTimeout, err := r.positiveDuration("HTTP_WRITE_TIMEOUT", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	idleTimeout, err := r.positiveDuration("HTTP_IDLE_TIMEOUT", 60*time.Second)
	if err != nil {
		return Config{}, err
	}
	requestTimeout, err := r.positiveDuration("HTTP_REQUEST_TIMEOUT", 15*time.Second)
	if err != nil {
		return Config{}, err
	}
	shutdownTimeout, err := r.positiveDuration("HTTP_SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	dependencyTimeout, err := r.positiveDuration("HEALTH_DEPENDENCY_TIMEOUT", 2*time.Second)
	if err != nil {
		return Config{}, err
	}
	accessTokenTTL, err := r.positiveDuration("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Config{}, err
	}
	refreshTokenTTL, err := r.positiveDuration("REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return Config{}, err
	}
	if refreshTokenTTL <= accessTokenTTL {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_TTL must exceed ACCESS_TOKEN_TTL")
	}
	rateLimitWindow, err := r.positiveDuration("AUTH_RATE_LIMIT_WINDOW", time.Minute)
	if err != nil {
		return Config{}, err
	}
	bcryptCost, err := r.positiveInt64("BCRYPT_COST", 12)
	if err != nil || bcryptCost < 10 || bcryptCost > 15 {
		return Config{}, fmt.Errorf("BCRYPT_COST must be between 10 and 15")
	}
	loginRateLimit, err := r.positiveInt64("AUTH_LOGIN_RATE_LIMIT", 10)
	if err != nil {
		return Config{}, err
	}
	registerRateLimit, err := r.positiveInt64("AUTH_REGISTER_RATE_LIMIT", 5)
	if err != nil {
		return Config{}, err
	}
	refreshRateLimit, err := r.positiveInt64("AUTH_REFRESH_RATE_LIMIT", 30)
	if err != nil {
		return Config{}, err
	}

	maxHeaderBytes, err := r.positiveInt64("HTTP_MAX_HEADER_BYTES", 1<<20)
	if err != nil {
		return Config{}, err
	}
	if maxHeaderBytes > int64(math.MaxInt) {
		return Config{}, fmt.Errorf("HTTP_MAX_HEADER_BYTES exceeds the platform integer limit")
	}
	maxRequestBodyBytes, err := r.positiveInt64("HTTP_MAX_REQUEST_BODY_BYTES", 1<<20)
	if err != nil {
		return Config{}, err
	}

	maxConnections, err := r.positiveInt64("POSTGRES_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	if maxConnections > math.MaxInt32 {
		return Config{}, fmt.Errorf("POSTGRES_MAX_CONNS must not exceed %d", int64(math.MaxInt32))
	}
	minConnections, err := r.nonNegativeInt64("POSTGRES_MIN_CONNS", 0)
	if err != nil {
		return Config{}, err
	}
	if minConnections > maxConnections {
		return Config{}, fmt.Errorf("POSTGRES_MIN_CONNS must not exceed POSTGRES_MAX_CONNS")
	}
	maxConnectionLifetime, err := r.positiveDuration("POSTGRES_MAX_CONN_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}
	maxConnectionIdleTime, err := r.positiveDuration("POSTGRES_MAX_CONN_IDLE_TIME", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	healthCheckPeriod, err := r.positiveDuration("POSTGRES_HEALTH_CHECK_PERIOD", time.Minute)
	if err != nil {
		return Config{}, err
	}

	redisPoolSize, err := r.positiveInt64("REDIS_POOL_SIZE", 10)
	if err != nil {
		return Config{}, err
	}
	if redisPoolSize > int64(math.MaxInt) {
		return Config{}, fmt.Errorf("REDIS_POOL_SIZE exceeds the platform integer limit")
	}
	redisMinIdleConnections, err := r.nonNegativeInt64("REDIS_MIN_IDLE_CONNS", 1)
	if err != nil {
		return Config{}, err
	}
	if redisMinIdleConnections > redisPoolSize {
		return Config{}, fmt.Errorf("REDIS_MIN_IDLE_CONNS must not exceed REDIS_POOL_SIZE")
	}
	redisDialTimeout, err := r.positiveDuration("REDIS_DIAL_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	redisReadTimeout, err := r.positiveDuration("REDIS_READ_TIMEOUT", 3*time.Second)
	if err != nil {
		return Config{}, err
	}
	redisWriteTimeout, err := r.positiveDuration("REDIS_WRITE_TIMEOUT", 3*time.Second)
	if err != nil {
		return Config{}, err
	}

	allowedOrigins, err := parseAllowedOrigins(r.stringValue("http://localhost:3000", "CORS_ALLOWED_ORIGINS"))
	if err != nil {
		return Config{}, err
	}
	allowCredentials, err := r.boolValue("CORS_ALLOW_CREDENTIALS", true)
	if err != nil {
		return Config{}, err
	}
	if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" && allowCredentials {
		return Config{}, fmt.Errorf("CORS_ALLOW_CREDENTIALS must be false when CORS_ALLOWED_ORIGINS is '*'")
	}

	logLevel := strings.ToLower(r.stringValue("info", "LOG_LEVEL"))
	switch logLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL must be one of debug, info, warn, or error")
	}

	address, err := httpAddress(r)
	if err != nil {
		return Config{}, err
	}

	environment := r.stringValue("development", "APP_ENV", "ENVIRONMENT")
	jwtSecret := r.stringValue("development-only-jwt-secret-change-before-production", "JWT_SECRET")
	refreshPepper := r.stringValue("development-only-refresh-pepper-change-before-production", "REFRESH_TOKEN_PEPPER")
	if len(jwtSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if len(refreshPepper) < 32 {
		return Config{}, fmt.Errorf("REFRESH_TOKEN_PEPPER must contain at least 32 characters")
	}
	if strings.EqualFold(environment, "production") &&
		(strings.HasPrefix(jwtSecret, "development-only-") || strings.HasPrefix(refreshPepper, "development-only-")) {
		return Config{}, fmt.Errorf("production authentication secrets must be explicitly configured")
	}
	cookieSecure, err := r.boolValue("REFRESH_COOKIE_SECURE", strings.EqualFold(environment, "production"))
	if err != nil {
		return Config{}, err
	}
	if strings.EqualFold(environment, "production") && !cookieSecure {
		return Config{}, fmt.Errorf("REFRESH_COOKIE_SECURE must be true in production")
	}

	return Config{
		Environment: environment,
		Version:     r.stringValue("dev", "APP_VERSION"),
		LogLevel:    logLevel,
		HTTP: HTTPConfig{
			Address:             address,
			AllowedOrigins:      allowedOrigins,
			AllowCredentials:    allowCredentials,
			ReadHeaderTimeout:   readHeaderTimeout,
			ReadTimeout:         readTimeout,
			WriteTimeout:        writeTimeout,
			IdleTimeout:         idleTimeout,
			RequestTimeout:      requestTimeout,
			ShutdownTimeout:     shutdownTimeout,
			MaxHeaderBytes:      int(maxHeaderBytes),
			MaxRequestBodyBytes: maxRequestBodyBytes,
		},
		Database: DatabaseConfig{
			URL:                   r.stringValue("postgres://klipforge:klipforge_dev_password@localhost:5432/klipforge?sslmode=disable", "DATABASE_URL"),
			MaxConnections:        int32(maxConnections),
			MinConnections:        int32(minConnections),
			MaxConnectionLifetime: maxConnectionLifetime,
			MaxConnectionIdleTime: maxConnectionIdleTime,
			HealthCheckPeriod:     healthCheckPeriod,
		},
		Redis: RedisConfig{
			URL:                r.stringValue("redis://localhost:6379/0", "REDIS_URL"),
			PoolSize:           int(redisPoolSize),
			MinIdleConnections: int(redisMinIdleConnections),
			DialTimeout:        redisDialTimeout,
			ReadTimeout:        redisReadTimeout,
			WriteTimeout:       redisWriteTimeout,
		},
		Health: HealthConfig{DependencyTimeout: dependencyTimeout},
		Auth: AuthConfig{
			JWTSecret:          jwtSecret,
			JWTIssuer:          r.stringValue("klipforge-api", "JWT_ISSUER"),
			JWTAudience:        r.stringValue("klipforge-web", "JWT_AUDIENCE"),
			AccessTokenTTL:     accessTokenTTL,
			RefreshTokenTTL:    refreshTokenTTL,
			RefreshTokenPepper: refreshPepper,
			BcryptCost:         int(bcryptCost),
			CookieName:         r.stringValue("klipforge_refresh_token", "REFRESH_COOKIE_NAME"),
			CookieDomain:       r.stringValue("", "REFRESH_COOKIE_DOMAIN"),
			CookieSecure:       cookieSecure,
			LoginRateLimit:     loginRateLimit,
			RegisterRateLimit:  registerRateLimit,
			RefreshRateLimit:   refreshRateLimit,
			RateLimitWindow:    rateLimitWindow,
		},
	}, nil
}

func httpAddress(r envReader) (string, error) {
	if address := r.stringValue("", "HTTP_ADDR"); address != "" {
		if _, _, err := net.SplitHostPort(address); err != nil {
			return "", fmt.Errorf("HTTP_ADDR must be a host:port pair: %w", err)
		}
		return address, nil
	}

	portValue := r.stringValue("8080", "HTTP_PORT", "API_PORT", "PORT")
	port, err := strconv.Atoi(portValue)
	if err != nil || port < 1 || port > 65535 {
		return "", fmt.Errorf("HTTP_PORT must be an integer between 1 and 65535")
	}
	host := r.stringValue("0.0.0.0", "HTTP_HOST", "API_HOST")
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}

func parseAllowedOrigins(raw string) ([]string, error) {
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))

	for _, part := range parts {
		origin := strings.TrimSuffix(strings.TrimSpace(part), "/")
		if origin == "" {
			continue
		}
		if origin == "*" {
			if len(parts) != 1 {
				return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS cannot combine '*' with explicit origins")
			}
			return []string{"*"}, nil
		}

		parsed, err := url.Parse(origin)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
			return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS contains invalid origin %q", origin)
		}
		origin = parsed.Scheme + "://" + parsed.Host
		if _, exists := seen[origin]; exists {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}

	if len(origins) == 0 {
		return nil, fmt.Errorf("CORS_ALLOWED_ORIGINS must contain at least one origin")
	}
	return origins, nil
}

func (r envReader) stringValue(fallback string, keys ...string) string {
	for _, key := range keys {
		if value, ok := r.lookup(key); ok && strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return fallback
}

func (r envReader) positiveDuration(key string, fallback time.Duration) (time.Duration, error) {
	raw := r.stringValue(fallback.String(), key)
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return value, nil
}

func (r envReader) positiveInt64(key string, fallback int64) (int64, error) {
	raw := r.stringValue(strconv.FormatInt(fallback, 10), key)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return value, nil
}

func (r envReader) nonNegativeInt64(key string, fallback int64) (int64, error) {
	raw := r.stringValue(strconv.FormatInt(fallback, 10), key)
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return value, nil
}

func (r envReader) boolValue(key string, fallback bool) (bool, error) {
	raw := r.stringValue(strconv.FormatBool(fallback), key)
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return false, fmt.Errorf("%s must be a boolean", key)
	}
	return value, nil
}

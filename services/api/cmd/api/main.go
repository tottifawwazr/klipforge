package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/config"
	"github.com/klipforge/klipforge/services/api/internal/dependency"
	"github.com/klipforge/klipforge/services/api/internal/health"
	"github.com/klipforge/klipforge/services/api/internal/httpapi"
	"github.com/klipforge/klipforge/services/api/internal/server"
)

func main() {
	bootstrapLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(); err != nil {
		bootstrapLogger.Error("API stopped with an error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	postgres, err := dependency.NewPostgres(context.Background(), cfg.Database)
	if err != nil {
		return err
	}
	defer postgres.Close()

	redis, err := dependency.NewRedis(cfg.Redis)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := redis.Close(); closeErr != nil {
			logger.Warn("failed to close Redis client", "error", closeErr)
		}
	}()

	healthService, err := health.NewService(
		config.ServiceName,
		cfg.Version,
		postgres,
		redis,
		cfg.Health.DependencyTimeout,
	)
	if err != nil {
		return fmt.Errorf("create health service: %w", err)
	}
	healthHandler := httpapi.NewHealthHandler(healthService, config.ServiceName, cfg.Version)
	authRepository := auth.NewRepository(postgres.Pool())
	tokenService := auth.NewTokenService(cfg.Auth.JWTSecret, cfg.Auth.RefreshTokenPepper, cfg.Auth.JWTIssuer, cfg.Auth.JWTAudience, cfg.Auth.AccessTokenTTL, cfg.Auth.RefreshTokenTTL)
	authService := auth.NewService(authRepository, auth.NewPasswordService(cfg.Auth.BcryptCost), tokenService)
	authHandler := httpapi.NewAuthHandler(authService, redis, cfg.Auth, logger)
	router := httpapi.NewRouter(cfg, logger, healthHandler, authHandler)
	httpServer := server.New(cfg.HTTP, router, logger)

	serveErrors := make(chan error, 1)
	go func() {
		logger.Info("API listening",
			"address", cfg.HTTP.Address,
			"environment", cfg.Environment,
			"service", config.ServiceName,
			"version", cfg.Version,
		)
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serveErrors <- err
	}()

	signalContext, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	select {
	case err := <-serveErrors:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-signalContext.Done():
		logger.Info("shutdown signal received")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(context.Background(), cfg.HTTP.ShutdownTimeout)
	defer cancelShutdown()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		_ = httpServer.Close()
		return fmt.Errorf("gracefully shut down HTTP server: %w", err)
	}

	if err := <-serveErrors; err != nil {
		return fmt.Errorf("serve HTTP during shutdown: %w", err)
	}
	logger.Info("API shutdown complete")
	return nil
}

func newLogger(level string) *slog.Logger {
	var slogLevel slog.Level
	switch level {
	case "debug":
		slogLevel = slog.LevelDebug
	case "warn":
		slogLevel = slog.LevelWarn
	case "error":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slogLevel}))
}

package server

import (
	"log/slog"
	"net/http"

	"github.com/klipforge/klipforge/services/api/internal/config"
)

func New(cfg config.HTTPConfig, handler http.Handler, logger *slog.Logger) *http.Server {
	return &http.Server{
		Addr:              cfg.Address,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}

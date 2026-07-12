package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/config"
)

func NewRouter(cfg config.Config, logger *slog.Logger, healthHandler *HealthHandler, authHandlers ...*AuthHandler) http.Handler {
	var authHandler *AuthHandler
	if len(authHandlers) > 0 {
		authHandler = authHandlers[0]
	}
	return newRouter(cfg, logger, healthHandler, authHandler, nil)
}

func NewRouterWithCampaigns(cfg config.Config, logger *slog.Logger, healthHandler *HealthHandler, authHandler *AuthHandler, campaignHandler *CampaignHandler, participationHandlers ...*ParticipationHandler) http.Handler {
	var participationHandler *ParticipationHandler
	if len(participationHandlers) > 0 {
		participationHandler = participationHandlers[0]
	}
	return newRouter(cfg, logger, healthHandler, authHandler, campaignHandler, participationHandler)
}

func newRouter(cfg config.Config, logger *slog.Logger, healthHandler *HealthHandler, authHandler *AuthHandler, campaignHandler *CampaignHandler, participationHandlers ...*ParticipationHandler) http.Handler {
	var participationHandler *ParticipationHandler
	if len(participationHandlers) > 0 {
		participationHandler = participationHandlers[0]
	}
	router := chi.NewRouter()

	router.Use(requestIDMiddleware)
	router.Use(requestLogger(logger))
	router.Use(recoverer(logger))
	router.Use(securityHeaders(strings.EqualFold(cfg.Environment, "production")))
	router.Use(newCORS(cfg.HTTP.AllowedOrigins, cfg.HTTP.AllowCredentials))
	router.Use(chimiddleware.RequestSize(cfg.HTTP.MaxRequestBodyBytes))
	router.Use(chimiddleware.Timeout(cfg.HTTP.RequestTimeout))

	router.Get("/healthz", healthHandler.Liveness)
	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/health", healthHandler.Readiness)
		if authHandler != nil {
			api.Mount("/auth", authHandler.Routes())
		}
		if campaignHandler != nil {
			campaignRoutes := chi.NewRouter()
			campaignHandler.RegisterPublicRoutes(campaignRoutes)
			brandCampaignRoutes := chi.NewRouter()
			brandCampaignRoutes.Use(authHandler.Authorization().RequireAuthentication, authHandler.Authorization().RequireActiveUser, authHandler.Authorization().RequireActiveSession, authHandler.Authorization().RequireRole(auth.RoleBrand))
			campaignHandler.RegisterBrandRoutes(brandCampaignRoutes)
			if participationHandler != nil {
				participationHandler.RegisterCampaignRoutes(campaignRoutes)
				participationHandler.RegisterBrandRoutes(brandCampaignRoutes)
			}
			api.Mount("/campaigns", campaignRoutes)
			api.Mount("/brand/campaigns", brandCampaignRoutes)
			api.Mount("/admin/campaigns", campaignHandler.AdminRoutes())
		}
		if participationHandler != nil {
			api.Mount("/clipper", participationHandler.ClipperRoutes())
			api.Mount("/admin", participationHandler.AdminRoutes())
		}
	})

	router.NotFound(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, r, http.StatusNotFound, "ROUTE_NOT_FOUND", "The requested route does not exist.")
	})
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		writeAPIError(w, r, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "The request method is not allowed for this route.")
	})

	return router
}

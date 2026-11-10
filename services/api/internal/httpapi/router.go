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
	return newRouter(cfg, logger, healthHandler, authHandler, nil, nil, nil)
}

func NewRouterWithCampaigns(cfg config.Config, logger *slog.Logger, healthHandler *HealthHandler, authHandler *AuthHandler, campaignHandler *CampaignHandler, participationHandlers ...*ParticipationHandler) http.Handler {
	var participationHandler *ParticipationHandler
	if len(participationHandlers) > 0 {
		participationHandler = participationHandlers[0]
	}
	return newRouter(cfg, logger, healthHandler, authHandler, campaignHandler, participationHandler, nil)
}

func NewRouterWithModeration(cfg config.Config, logger *slog.Logger, healthHandler *HealthHandler, authHandler *AuthHandler, campaignHandler *CampaignHandler, participationHandler *ParticipationHandler, moderationHandler *ModerationHandler) http.Handler {
	return newRouter(cfg, logger, healthHandler, authHandler, campaignHandler, participationHandler, moderationHandler)
}

func NewRouterWithAnalytics(cfg config.Config, logger *slog.Logger, healthHandler *HealthHandler, authHandler *AuthHandler, campaignHandler *CampaignHandler, participationHandler *ParticipationHandler, moderationHandler *ModerationHandler, analyticsHandler *AnalyticsHandler) http.Handler {
	return newRouter(cfg, logger, healthHandler, authHandler, campaignHandler, participationHandler, moderationHandler, ProductHandlers{Analytics: analyticsHandler})
}

type ProductHandlers struct {
	YouTube    *YouTubeHandler
	Analytics  *AnalyticsHandler
	Operations *OperationsHandler
}

func NewRouterWithOperations(cfg config.Config, logger *slog.Logger, healthHandler *HealthHandler, authHandler *AuthHandler, campaignHandler *CampaignHandler, participationHandler *ParticipationHandler, moderationHandler *ModerationHandler, analyticsHandler *AnalyticsHandler, operationsHandler *OperationsHandler, youtubeHandlers ...*YouTubeHandler) http.Handler {
	var youtubeHandler *YouTubeHandler
	if len(youtubeHandlers) > 0 {
		youtubeHandler = youtubeHandlers[0]
	}
	return newRouter(cfg, logger, healthHandler, authHandler, campaignHandler, participationHandler, moderationHandler, ProductHandlers{Analytics: analyticsHandler, Operations: operationsHandler, YouTube: youtubeHandler})
}

func newRouter(cfg config.Config, logger *slog.Logger, healthHandler *HealthHandler, authHandler *AuthHandler, campaignHandler *CampaignHandler, participationHandler *ParticipationHandler, moderationHandler *ModerationHandler, features ...ProductHandlers) http.Handler {
	var analyticsHandler *AnalyticsHandler
	var operationsHandler *OperationsHandler
	var youtubeHandler *YouTubeHandler
	if len(features) > 0 {
		analyticsHandler = features[0].Analytics
		operationsHandler = features[0].Operations
		youtubeHandler = features[0].YouTube
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
		if analyticsHandler != nil && authHandler != nil {
			api.With(authHandler.Authorization().RequireAuthentication, authHandler.Authorization().RequireActiveUser, authHandler.Authorization().RequireActiveSession).Get("/analytics", analyticsHandler.overview)
		}
		if authHandler != nil {
			api.Mount("/auth", authHandler.Routes())
		}
		if youtubeHandler != nil && authHandler != nil {
			guards := []func(http.Handler) http.Handler{authHandler.Authorization().RequireAuthentication, authHandler.Authorization().RequireActiveUser, authHandler.Authorization().RequireActiveSession}
			api.With(guards...).Get("/integrations/youtube", youtubeHandler.index)
			api.With(guards...).Get("/integrations/youtube/submissions/{submissionID}", youtubeHandler.metrics)
		}
		if operationsHandler != nil && authHandler != nil {
			protected := chi.NewRouter()
			protected.Use(authHandler.Authorization().RequireAuthentication, authHandler.Authorization().RequireActiveUser, authHandler.Authorization().RequireActiveSession)
			protected.Get("/payouts", operationsHandler.list)
			protected.Get("/notifications", operationsHandler.notifications)
			protected.Post("/notifications/{notificationID}/read", operationsHandler.read)
			api.Mount("/account", protected)
		}
		if campaignHandler != nil {
			campaignRoutes := chi.NewRouter()
			campaignHandler.RegisterPublicRoutes(campaignRoutes)
			if analyticsHandler != nil {
				campaignRoutes.With(authHandler.Authorization().RequireAuthentication, authHandler.Authorization().RequireActiveUser, authHandler.Authorization().RequireActiveSession).Get("/{campaignID}/analytics", analyticsHandler.overview)
				campaignRoutes.With(authHandler.Authorization().RequireAuthentication, authHandler.Authorization().RequireActiveUser, authHandler.Authorization().RequireActiveSession).Get("/{campaignID}/leaderboard", analyticsHandler.leaderboard)
			}
			brandCampaignRoutes := chi.NewRouter()
			brandCampaignRoutes.Use(authHandler.Authorization().RequireAuthentication, authHandler.Authorization().RequireActiveUser, authHandler.Authorization().RequireActiveSession, authHandler.Authorization().RequireRole(auth.RoleBrand))
			campaignHandler.RegisterBrandRoutes(brandCampaignRoutes)
			if participationHandler != nil {
				participationHandler.RegisterCampaignRoutes(campaignRoutes)
				participationHandler.RegisterBrandRoutes(brandCampaignRoutes)
			}
			if moderationHandler != nil {
				moderationHandler.RegisterBrandRoutes(brandCampaignRoutes)
			}
			api.Mount("/campaigns", campaignRoutes)
			api.Mount("/brand/campaigns", brandCampaignRoutes)
			api.Mount("/admin/campaigns", campaignHandler.AdminRoutes())
		}
		if participationHandler != nil || moderationHandler != nil || analyticsHandler != nil || operationsHandler != nil || youtubeHandler != nil {
			adminRoutes := chi.NewRouter()
			adminRoutes.Use(authHandler.Authorization().RequireAuthentication, authHandler.Authorization().RequireActiveUser, authHandler.Authorization().RequireActiveSession, authHandler.Authorization().RequireRole(auth.RoleAdmin))
			if participationHandler != nil {
				participationHandler.RegisterAdminRoutes(adminRoutes)
			}
			if moderationHandler != nil {
				moderationHandler.RegisterAdminRoutes(adminRoutes)
			}
			if analyticsHandler != nil {
				adminRoutes.Post("/metrics/collect", analyticsHandler.collect)
			}
			if operationsHandler != nil {
				adminRoutes.Post("/payouts", operationsHandler.create)
				adminRoutes.Post("/payout-quotes", operationsHandler.quote)
				adminRoutes.Post("/payouts/{payoutID}/{action}", operationsHandler.transition)
				adminRoutes.Get("/audit", operationsHandler.audit)
			}
			if youtubeHandler != nil {
				adminRoutes.Post("/youtube/submissions/{submissionID}/refresh", youtubeHandler.refresh)
			}
			api.Mount("/admin", adminRoutes)
		}
		if participationHandler != nil {
			api.Mount("/clipper", participationHandler.ClipperRoutes())
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

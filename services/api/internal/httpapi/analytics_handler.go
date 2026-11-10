package httpapi

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/klipforge/klipforge/services/api/internal/analytics"
	"log/slog"
	"net/http"
)

type AnalyticsHandler struct {
	service   *analytics.Service
	collector *analytics.Collector
	logger    *slog.Logger
}

func NewAnalyticsHandler(service *analytics.Service, collector *analytics.Collector, logger *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{service, collector, logger}
}
func (h *AnalyticsHandler) overview(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Overview(r.Context(), principal(r), chi.URLParam(r, "campaignID"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"analytics": data})
}
func (h *AnalyticsHandler) leaderboard(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.Leaderboard(r.Context(), principal(r), chi.URLParam(r, "campaignID"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, 200, map[string]any{"leaderboard": data, "metrics_mode": "SIMULATED"})
}
func (h *AnalyticsHandler) collect(w http.ResponseWriter, r *http.Request) {
	if h.collector == nil {
		writeAPIError(w, r, 503, "METRICS_DISABLED", "The metric collector is disabled.")
		return
	}
	data, err := h.collector.Collect(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	writeJSON(w, 200, data)
}
func (h *AnalyticsHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	status, code := 500, "ANALYTICS_UNAVAILABLE"
	if errors.Is(err, analytics.ErrNotFound) {
		status, code = 404, "RESOURCE_NOT_FOUND"
	} else if errors.Is(err, analytics.ErrForbidden) {
		status, code = 403, "FORBIDDEN"
	} else {
		h.logger.ErrorContext(r.Context(), "analytics operation failed", "request_id", requestIDFromContext(r.Context()), "error", err)
	}
	writeAPIError(w, r, status, code, analytics.ErrorMessage(err))
}

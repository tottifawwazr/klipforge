package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/klipforge/klipforge/services/api/internal/health"
)

type readinessService interface {
	Readiness(context.Context) health.Report
}

type HealthHandler struct {
	service     readinessService
	serviceName string
	version     string
	now         func() time.Time
}

func NewHealthHandler(service readinessService, serviceName, version string) *HealthHandler {
	return &HealthHandler{
		service:     service,
		serviceName: serviceName,
		version:     version,
		now:         time.Now,
	}
}

type livenessResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, livenessResponse{
		Status:    health.StatusOK,
		Service:   h.serviceName,
		Version:   h.version,
		Timestamp: h.now().UTC().Format(time.RFC3339),
	})
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	report := h.service.Readiness(r.Context())
	status := http.StatusOK
	if report.Status != health.StatusOK {
		status = http.StatusServiceUnavailable
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, status, report)
}

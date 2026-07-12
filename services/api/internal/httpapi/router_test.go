package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/klipforge/klipforge/services/api/internal/config"
	"github.com/klipforge/klipforge/services/api/internal/health"
)

type readinessStub struct {
	report health.Report
}

func (stub readinessStub) Readiness(context.Context) health.Report {
	return stub.report
}

func TestReadinessRouteReturnsHealthyDependencies(t *testing.T) {
	report := testHealthReport(health.StatusOK, health.CheckUp, health.CheckUp)
	recorder := serveTestRequest(report, http.MethodGet, "/api/v1/health", nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if recorder.Header().Get("X-Request-ID") == "" {
		t.Fatal("X-Request-ID header is missing")
	}
	if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("security headers are missing")
	}

	var response health.Report
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Service != config.ServiceName || response.Checks.Postgres.Status != health.CheckUp || response.Checks.Redis.Status != health.CheckUp {
		t.Fatalf("unexpected health response: %#v", response)
	}
}

func TestReadinessRouteReturnsServiceUnavailableWhenDependencyIsDown(t *testing.T) {
	report := testHealthReport(health.StatusDegraded, health.CheckDown, health.CheckUp)
	recorder := serveTestRequest(report, http.MethodGet, "/api/v1/health", nil)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
	}
}

func TestRouterUsesConsistentErrorEnvelope(t *testing.T) {
	recorder := serveTestRequest(testHealthReport(health.StatusOK, health.CheckUp, health.CheckUp), http.MethodGet, "/missing", func(request *http.Request) {
		request.Header.Set("X-Request-ID", "test-request-123")
	})

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusNotFound)
	}
	var response errorEnvelope
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Error.Code != "ROUTE_NOT_FOUND" || response.Error.RequestID != "test-request-123" {
		t.Fatalf("unexpected error response: %#v", response)
	}
}

func TestRouterHandlesAllowedCORSPreflight(t *testing.T) {
	recorder := serveTestRequest(testHealthReport(health.StatusOK, health.CheckUp, health.CheckUp), http.MethodOptions, "/api/v1/health", func(request *http.Request) {
		request.Header.Set("Origin", "http://localhost:3000")
		request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	})

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusNoContent, recorder.Body.String())
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Fatalf("allow origin = %q", recorder.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestSharedBrandRoutesRegisterAfterExistingRoutes(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/existing", func(http.ResponseWriter, *http.Request) {})
	authorization := &AuthorizationMiddleware{}

	NewCampaignHandler(nil, authorization, nil).RegisterBrandRoutes(router)
	NewParticipationHandler(nil, authorization).RegisterBrandRoutes(router)
	NewModerationHandler(nil, authorization, slog.Default()).RegisterBrandRoutes(router)
}

func TestSharedAdminRoutesRegisterTogether(t *testing.T) {
	router := chi.NewRouter()
	authorization := &AuthorizationMiddleware{}

	NewParticipationHandler(nil, authorization).RegisterAdminRoutes(router)
	NewModerationHandler(nil, authorization, slog.Default()).RegisterAdminRoutes(router)
}

func serveTestRequest(report health.Report, method, target string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	cfg := config.Config{
		Environment: "test",
		HTTP: config.HTTPConfig{
			AllowedOrigins:      []string{"http://localhost:3000"},
			AllowCredentials:    true,
			RequestTimeout:      time.Second,
			MaxRequestBodyBytes: 1 << 20,
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHealthHandler(readinessStub{report: report}, config.ServiceName, "test-version")
	router := NewRouter(cfg, logger, handler)
	request := httptest.NewRequest(method, target, nil)
	if mutate != nil {
		mutate(request)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	return recorder
}

func testHealthReport(status, postgresStatus, redisStatus string) health.Report {
	return health.Report{
		Status:    status,
		Service:   config.ServiceName,
		Version:   "test-version",
		Timestamp: "2026-07-11T03:04:05Z",
		Checks: health.Checks{
			Postgres: health.DependencyCheck{Status: postgresStatus, LatencyMS: 2},
			Redis:    health.DependencyCheck{Status: redisStatus, LatencyMS: 1},
		},
	}
}

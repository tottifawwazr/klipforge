package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/klipforge/klipforge/services/api/internal/health"
)

type staticReadinessService struct {
	report health.Report
}

func (service staticReadinessService) Readiness(context.Context) health.Report {
	return service.report
}

func TestHealthHandlerLiveness(t *testing.T) {
	handler := NewHealthHandler(staticReadinessService{}, "klipforge-api", "test-version")
	handler.now = func() time.Time {
		return time.Date(2026, 7, 11, 3, 4, 5, 0, time.UTC)
	}

	response := httptest.NewRecorder()
	handler.Liveness(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", response.Header().Get("Cache-Control"))
	}
	var payload livenessResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Status != "ok" || payload.Service != "klipforge-api" || payload.Version != "test-version" || payload.Timestamp != "2026-07-11T03:04:05Z" {
		t.Fatalf("unexpected response: %#v", payload)
	}
}

func TestHealthHandlerReadinessStatus(t *testing.T) {
	tests := []struct {
		name       string
		report     health.Report
		wantStatus int
	}{
		{
			name: "ready",
			report: health.Report{
				Status: "ok", Service: "klipforge-api", Version: "dev", Timestamp: "2026-07-11T03:04:05Z",
				Checks: health.Checks{
					Postgres: health.DependencyCheck{Status: "up", LatencyMS: 1},
					Redis:    health.DependencyCheck{Status: "up", LatencyMS: 2},
				},
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "degraded",
			report: health.Report{
				Status: "degraded", Service: "klipforge-api", Version: "dev", Timestamp: "2026-07-11T03:04:05Z",
				Checks: health.Checks{
					Postgres: health.DependencyCheck{Status: "down", LatencyMS: 5},
					Redis:    health.DependencyCheck{Status: "up", LatencyMS: 1},
				},
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := NewHealthHandler(staticReadinessService{report: test.report}, "klipforge-api", "dev")
			response := httptest.NewRecorder()
			handler.Readiness(response, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, test.wantStatus)
			}
			var payload health.Report
			if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if payload != test.report {
				t.Fatalf("payload = %#v, want %#v", payload, test.report)
			}
		})
	}
}

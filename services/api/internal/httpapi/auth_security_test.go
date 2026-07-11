package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/klipforge/klipforge/services/api/internal/config"
)

type limiterStub struct {
	allowed bool
	err     error
}

func (s limiterStub) Allow(context.Context, string, int64, time.Duration) (bool, error) {
	return s.allowed, s.err
}

func TestAuthenticationRateLimitErrorsUseStructuredEnvelope(t *testing.T) {
	for _, test := range []struct {
		name    string
		limiter limiterStub
		status  int
		code    string
	}{
		{"limited", limiterStub{allowed: false}, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED"},
		{"redis unavailable", limiterStub{err: errors.New("redis unavailable")}, http.StatusServiceUnavailable, "AUTH_SERVICE_UNAVAILABLE"},
	} {
		t.Run(test.name, func(t *testing.T) {
			logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
			handler := NewAuthHandler(nil, test.limiter, config.AuthConfig{RegisterRateLimit: 1, RateLimitWindow: time.Minute}, logger)
			router := requestIDMiddleware(handler.Routes())
			request := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(`{}`))
			request.RemoteAddr = "127.0.0.1:1234"
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, request)
			if recorder.Code != test.status {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			var envelope errorEnvelope
			if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil {
				t.Fatal(err)
			}
			if envelope.Error.Code != test.code || envelope.Error.RequestID == "" {
				t.Fatalf("unexpected error=%#v", envelope.Error)
			}
		})
	}
}

func TestRequestLoggerDoesNotLogCredentialsOrTokens(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := requestIDMiddleware(requestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) })))
	secretValues := []string{"StrongPass1", "raw-access-token", "raw-refresh-token"}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"password":"StrongPass1"}`))
	request.Header.Set("Authorization", "Bearer raw-access-token")
	request.Header.Set("Cookie", "klipforge_refresh_token=raw-refresh-token")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	for _, secret := range secretValues {
		if strings.Contains(output.String(), secret) {
			t.Fatalf("logger exposed secret %q: %s", secret, output.String())
		}
	}
}

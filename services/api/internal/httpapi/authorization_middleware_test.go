package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/config"
)

type authenticatorStub struct {
	principal auth.Principal
	err       error
}

func (s authenticatorStub) Authenticate(context.Context, string) (auth.Principal, error) {
	return s.principal, s.err
}

func activePrincipal(role string) auth.Principal {
	return auth.Principal{UserID: "00000000-0000-0000-0000-000000000001", Email: "user@example.test", Role: role, SessionID: "00000000-0000-0000-0000-000000000002", TokenID: "00000000-0000-0000-0000-000000000003", AccountActive: true, SessionActive: true}
}

func testAuthorizationMiddleware(principal auth.Principal) *AuthorizationMiddleware {
	return NewAuthorizationMiddleware(authenticatorStub{principal: principal}, slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
}

func TestRequireAuthenticationAttachesTypedPrincipal(t *testing.T) {
	principal := activePrincipal(auth.RoleBrand)
	middleware := testAuthorizationMiddleware(principal)
	reached := false
	handler := requestIDMiddleware(middleware.RequireAuthentication(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actual, ok := auth.PrincipalFromContext(r.Context())
		if !ok || actual.TokenID != principal.TokenID {
			t.Fatalf("principal=%#v ok=%v", actual, ok)
		}
		reached = true
		w.WriteHeader(http.StatusNoContent)
	})))
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "Bearer safe-test-token")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent || !reached {
		t.Fatalf("status=%d reached=%v body=%s", recorder.Code, reached, recorder.Body.String())
	}
}

func TestRequireAuthenticationRejectsInvalidBoundary(t *testing.T) {
	tests := []struct {
		name, header  string
		authenticator authenticatorStub
		status        int
		code          string
	}{
		{"missing", "", authenticatorStub{}, http.StatusUnauthorized, "ACCESS_TOKEN_MISSING"},
		{"malformed", "Basic value", authenticatorStub{}, http.StatusUnauthorized, "ACCESS_TOKEN_INVALID"},
		{"revoked", "Bearer token", authenticatorStub{err: auth.ErrSessionRevoked}, http.StatusUnauthorized, "SESSION_REVOKED"},
		{"inactive", "Bearer token", authenticatorStub{err: auth.ErrUserInactive}, http.StatusForbidden, "USER_INACTIVE"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			middleware := NewAuthorizationMiddleware(test.authenticator, slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil)))
			handler := requestIDMiddleware(middleware.RequireAuthentication(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("protected handler reached") })))
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.header != "" {
				request.Header.Set("Authorization", test.header)
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			assertErrorCode(t, recorder, test.status, test.code)
		})
	}
}

func TestRoleGuards(t *testing.T) {
	tests := []struct {
		name, role string
		guard      func(*AuthorizationMiddleware) func(http.Handler) http.Handler
		status     int
		code       string
	}{
		{"admin accepted", auth.RoleAdmin, func(m *AuthorizationMiddleware) func(http.Handler) http.Handler { return m.RequireRole(auth.RoleAdmin) }, http.StatusNoContent, ""},
		{"brand rejected admin", auth.RoleBrand, func(m *AuthorizationMiddleware) func(http.Handler) http.Handler { return m.RequireRole(auth.RoleAdmin) }, http.StatusForbidden, "ROLE_NOT_ALLOWED"},
		{"clipper rejected admin", auth.RoleClipper, func(m *AuthorizationMiddleware) func(http.Handler) http.Handler { return m.RequireRole(auth.RoleAdmin) }, http.StatusForbidden, "ROLE_NOT_ALLOWED"},
		{"brand accepted any", auth.RoleBrand, func(m *AuthorizationMiddleware) func(http.Handler) http.Handler {
			return m.RequireAnyRole(auth.RoleBrand, auth.RoleAdmin)
		}, http.StatusNoContent, ""},
		{"clipper accepted", auth.RoleClipper, func(m *AuthorizationMiddleware) func(http.Handler) http.Handler {
			return m.RequireRole(auth.RoleClipper)
		}, http.StatusNoContent, ""},
		{"invalid principal role", "OWNER", func(m *AuthorizationMiddleware) func(http.Handler) http.Handler {
			return m.RequireAnyRole(auth.RoleAdmin, auth.RoleBrand)
		}, http.StatusForbidden, "ROLE_NOT_ALLOWED"},
		{"invalid configured role", auth.RoleAdmin, func(m *AuthorizationMiddleware) func(http.Handler) http.Handler { return m.RequireRole("OWNER") }, http.StatusForbidden, "ROLE_NOT_ALLOWED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			middleware := testAuthorizationMiddleware(activePrincipal(test.role))
			reached := false
			handler := requestIDMiddleware(test.guard(middleware)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { reached = true; w.WriteHeader(http.StatusNoContent) })))
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request = request.WithContext(auth.WithPrincipal(request.Context(), activePrincipal(test.role)))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if test.status == http.StatusNoContent {
				if recorder.Code != test.status || !reached {
					t.Fatalf("status=%d reached=%v", recorder.Code, reached)
				}
				return
			}
			assertErrorCode(t, recorder, test.status, test.code)
		})
	}
}

func TestActiveStateGuards(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*auth.Principal)
		guard  func(*AuthorizationMiddleware) func(http.Handler) http.Handler
		status int
		code   string
	}{
		{"inactive user", func(p *auth.Principal) { p.AccountActive = false }, func(m *AuthorizationMiddleware) func(http.Handler) http.Handler { return m.RequireActiveUser }, http.StatusForbidden, "USER_INACTIVE"},
		{"revoked session", func(p *auth.Principal) { p.SessionActive = false }, func(m *AuthorizationMiddleware) func(http.Handler) http.Handler { return m.RequireActiveSession }, http.StatusUnauthorized, "SESSION_REVOKED"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			principal := activePrincipal(auth.RoleClipper)
			test.mutate(&principal)
			middleware := testAuthorizationMiddleware(principal)
			handler := requestIDMiddleware(test.guard(middleware)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("guarded handler reached") })))
			request := httptest.NewRequest(http.MethodGet, "/protected", nil).WithContext(auth.WithPrincipal(context.Background(), principal))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			assertErrorCode(t, recorder, test.status, test.code)
		})
	}
}

func TestAuthenticationRouteBoundary(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(&bytes.Buffer{}, nil))
	handler := requestIDMiddleware(NewAuthHandler(nil, limiterStub{allowed: false}, config.AuthConfig{LoginRateLimit: 1, RegisterRateLimit: 1, RefreshRateLimit: 1}, logger).Routes())
	for _, path := range []string{"/register", "/login"} {
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{}`))
		request.RemoteAddr = "127.0.0.1:1234"
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusTooManyRequests {
			t.Fatalf("public %s unexpectedly protected: status=%d", path, recorder.Code)
		}
	}
	for _, path := range []string{"/me", "/logout-all"} {
		method := http.MethodGet
		if path == "/logout-all" {
			method = http.MethodPost
		}
		request := httptest.NewRequest(method, path, nil)
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		assertErrorCode(t, recorder, http.StatusUnauthorized, "ACCESS_TOKEN_MISSING")
	}
}

func TestOwnershipGuardMapsDecisionsAndHidesInternalErrors(t *testing.T) {
	principal := activePrincipal(auth.RoleBrand)
	middleware := testAuthorizationMiddleware(principal)
	tests := []struct {
		name   string
		check  PolicyCheck
		status int
		code   string
	}{
		{"inaccessible resource", func(context.Context, auth.Principal, *http.Request) error { return auth.ErrResourceNotFound }, http.StatusNotFound, "RESOURCE_NOT_FOUND"},
		{"database failure", func(context.Context, auth.Principal, *http.Request) error {
			return errors.New("driver secret: relation details")
		}, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := requestIDMiddleware(middleware.RequireOwnership("campaign.manage", test.check)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("protected handler reached") })))
			request := httptest.NewRequest(http.MethodGet, "/campaigns/secret", nil).WithContext(auth.WithPrincipal(context.Background(), principal))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			assertErrorCode(t, recorder, test.status, test.code)
			if bytes.Contains(recorder.Body.Bytes(), []byte("driver secret")) {
				t.Fatalf("internal error leaked: %s", recorder.Body.String())
			}
		})
	}
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("status=%d want=%d body=%s", recorder.Code, status, recorder.Body.String())
	}
	var envelope errorEnvelope
	if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil {
		t.Fatal(err)
	}
	if envelope.Error.Code != code || envelope.Error.RequestID == "" {
		t.Fatalf("error=%#v", envelope.Error)
	}
}

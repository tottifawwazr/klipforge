package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

func TestModerationRoutesEnforceAuthenticationRoleAndState(t *testing.T) {
	tests := []struct {
		name       string
		admin      bool
		principal  auth.Principal
		authorize  bool
		wantStatus int
		wantCode   string
	}{
		{"brand missing token", false, activePrincipal(auth.RoleBrand), false, http.StatusUnauthorized, "ACCESS_TOKEN_MISSING"},
		{"brand rejects clipper", false, activePrincipal(auth.RoleClipper), true, http.StatusForbidden, "ROLE_NOT_ALLOWED"},
		{"brand rejects inactive account", false, func() auth.Principal { p := activePrincipal(auth.RoleBrand); p.AccountActive = false; return p }(), true, http.StatusForbidden, "USER_INACTIVE"},
		{"brand rejects revoked session", false, func() auth.Principal { p := activePrincipal(auth.RoleBrand); p.SessionActive = false; return p }(), true, http.StatusUnauthorized, "SESSION_REVOKED"},
		{"admin rejects brand", true, activePrincipal(auth.RoleBrand), true, http.StatusForbidden, "ROLE_NOT_ALLOWED"},
		{"admin rejects clipper", true, activePrincipal(auth.RoleClipper), true, http.StatusForbidden, "ROLE_NOT_ALLOWED"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
			authorization := testAuthorizationMiddleware(test.principal)
			handler := NewModerationHandler(nil, authorization, logger)
			router := chi.NewRouter()
			router.Use(authorization.RequireAuthentication, authorization.RequireActiveUser, authorization.RequireActiveSession)
			if test.admin {
				router.Use(authorization.RequireRole(auth.RoleAdmin))
				handler.RegisterAdminRoutes(router)
			} else {
				router.Use(authorization.RequireRole(auth.RoleBrand))
				handler.RegisterBrandRoutes(router)
			}
			target := "/00000000-0000-0000-0000-000000000301/moderation/submissions"
			if test.admin {
				target = "/moderation/submissions"
			}
			request := httptest.NewRequest(http.MethodGet, target, bytes.NewReader(nil))
			if test.authorize {
				request.Header.Set("Authorization", "Bearer test-token")
			}
			recorder := httptest.NewRecorder()
			requestIDMiddleware(router).ServeHTTP(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, test.wantStatus, recorder.Body.String())
			}
			var envelope errorEnvelope
			if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil || envelope.Error.Code != test.wantCode {
				t.Fatalf("error = %#v, decode=%v", envelope, err)
			}
		})
	}
}

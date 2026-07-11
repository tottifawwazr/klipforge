package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

type principalAuthenticator interface {
	Authenticate(context.Context, string) (auth.Principal, error)
}

type AuthorizationMiddleware struct {
	authenticator principalAuthenticator
	logger        *slog.Logger
}

type PolicyCheck func(context.Context, auth.Principal, *http.Request) error

func NewAuthorizationMiddleware(authenticator principalAuthenticator, logger *slog.Logger) *AuthorizationMiddleware {
	return &AuthorizationMiddleware{authenticator: authenticator, logger: logger}
}

func (m *AuthorizationMiddleware) RequireAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := strings.TrimSpace(r.Header.Get("Authorization"))
		if header == "" {
			writeAuthError(w, r, auth.ErrAccessMissing)
			return
		}
		parts := strings.Fields(header)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeAuthError(w, r, auth.ErrAccessInvalid)
			return
		}
		principal, err := m.authenticator.Authenticate(r.Context(), parts[1])
		if err != nil {
			if domain := auth.AsError(err); domain != nil {
				writeAuthError(w, r, domain)
				return
			}
			m.logger.ErrorContext(r.Context(), "authentication middleware failed", "request_id", requestIDFromContext(r.Context()), "error", err)
			writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred.")
			return
		}
		if !auth.ValidRole(principal.Role) {
			m.deny(w, r, principal, "require_authentication", auth.ErrAccessInvalid, "invalid_principal_role")
			return
		}
		next.ServeHTTP(w, r.WithContext(auth.WithPrincipal(r.Context(), principal)))
	})
}

func (m *AuthorizationMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return m.RequireAnyRole(role)
}

func (m *AuthorizationMiddleware) RequireAnyRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	validConfiguration := len(roles) > 0
	for _, role := range roles {
		if !auth.ValidRole(role) {
			validConfiguration = false
			continue
		}
		allowed[role] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := auth.PrincipalFromContext(r.Context())
			if !ok {
				writeAuthError(w, r, auth.ErrAuthenticationRequired)
				return
			}
			if !validConfiguration || !auth.ValidRole(principal.Role) {
				m.deny(w, r, principal, "require_any_role", auth.ErrRoleNotAllowed, "invalid_role")
				return
			}
			if _, ok := allowed[principal.Role]; !ok {
				m.deny(w, r, principal, "require_any_role", auth.ErrRoleNotAllowed, "role_not_allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (m *AuthorizationMiddleware) RequireActiveUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			writeAuthError(w, r, auth.ErrAuthenticationRequired)
			return
		}
		if !principal.AccountActive {
			m.deny(w, r, principal, "require_active_user", auth.ErrUserInactive, "user_inactive")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthorizationMiddleware) RequireActiveSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			writeAuthError(w, r, auth.ErrAuthenticationRequired)
			return
		}
		if !principal.SessionActive {
			m.deny(w, r, principal, "require_active_session", auth.ErrSessionRevoked, "session_revoked")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (m *AuthorizationMiddleware) RequirePolicy(policyName string, check PolicyCheck) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := auth.PrincipalFromContext(r.Context())
			if !ok {
				writeAuthError(w, r, auth.ErrAuthenticationRequired)
				return
			}
			if check == nil {
				m.logger.ErrorContext(r.Context(), "authorization policy is not configured", "request_id", requestIDFromContext(r.Context()), "policy", policyName)
				writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred.")
				return
			}
			err := check(r.Context(), principal, r)
			if err == nil {
				next.ServeHTTP(w, r)
				return
			}
			if domain := auth.AsError(err); domain != nil {
				m.deny(w, r, principal, policyName, domain, domain.Code)
				return
			}
			m.logger.ErrorContext(r.Context(), "authorization policy failed",
				"request_id", requestIDFromContext(r.Context()), "route", r.URL.Path,
				"method", r.Method, "user_id", principal.UserID, "role", principal.Role,
				"policy", policyName, "decision", "error",
			)
			writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred.")
		})
	}
}

func (m *AuthorizationMiddleware) RequireOwnership(policyName string, check PolicyCheck) func(http.Handler) http.Handler {
	return m.RequirePolicy(policyName, check)
}

func (m *AuthorizationMiddleware) deny(w http.ResponseWriter, r *http.Request, principal auth.Principal, policy string, domain *auth.Error, reason string) {
	m.logger.WarnContext(r.Context(), "authorization denied",
		"request_id", requestIDFromContext(r.Context()),
		"route", r.URL.Path,
		"method", r.Method,
		"user_id", principal.UserID,
		"role", principal.Role,
		"policy", policy,
		"decision", "deny",
		"reason_code", reason,
	)
	writeAuthError(w, r, domain)
}

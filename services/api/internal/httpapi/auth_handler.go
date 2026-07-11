package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/klipforge/klipforge/services/api/internal/auth"
	"github.com/klipforge/klipforge/services/api/internal/config"
)

type authRateLimiter interface {
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error)
}

type AuthHandler struct {
	service *auth.Service
	limiter authRateLimiter
	cfg     config.AuthConfig
	logger  *slog.Logger
}

func NewAuthHandler(service *auth.Service, limiter authRateLimiter, cfg config.AuthConfig, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{service, limiter, cfg, logger}
}

func (h *AuthHandler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/register", h.rateLimited("register", h.cfg.RegisterRateLimit, h.register))
	r.Post("/login", h.rateLimited("login", h.cfg.LoginRateLimit, h.login))
	r.Post("/refresh", h.rateLimited("refresh", h.cfg.RefreshRateLimit, h.refresh))
	r.Post("/logout", h.logout)
	r.Group(func(protected chi.Router) {
		protected.Use(h.authenticate)
		protected.Post("/logout-all", h.logoutAll)
		protected.Get("/me", h.me)
	})
	return r
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		FullName string `json:"full_name"`
		Role     string `json:"role"`
	}
	if !decodeJSON(r, &body) {
		writeAuthError(w, r, auth.ErrValidation)
		return
	}
	pair, err := h.service.Register(r.Context(), auth.RegisterInput{
		Email: body.Email, Password: body.Password, FullName: body.FullName, Role: body.Role,
		RequestID: requestIDFromContext(r.Context()), UserAgent: r.UserAgent(), IPAddress: remoteIP(r.RemoteAddr),
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	h.setCookie(w, pair.RefreshToken, pair.RefreshExpiresAt)
	writeTokenResponse(w, http.StatusCreated, pair, h.cfg.AccessTokenTTL)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decodeJSON(r, &body) {
		writeAuthError(w, r, auth.ErrValidation)
		return
	}
	pair, err := h.service.Login(r.Context(), auth.LoginInput{
		Email: body.Email, Password: body.Password, RequestID: requestIDFromContext(r.Context()),
		UserAgent: r.UserAgent(), IPAddress: remoteIP(r.RemoteAddr),
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	h.setCookie(w, pair.RefreshToken, pair.RefreshExpiresAt)
	writeTokenResponse(w, http.StatusOK, pair, h.cfg.AccessTokenTTL)
}

func (h *AuthHandler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(h.cfg.CookieName)
	if err != nil {
		h.clearCookie(w)
		writeAuthError(w, r, auth.ErrRefreshMissing)
		return
	}
	pair, err := h.service.Refresh(r.Context(), cookie.Value, requestIDFromContext(r.Context()), r.UserAgent(), remoteIP(r.RemoteAddr))
	if err != nil {
		h.clearCookie(w)
		h.handleError(w, r, err)
		return
	}
	h.setCookie(w, pair.RefreshToken, pair.RefreshExpiresAt)
	writeTokenResponse(w, http.StatusOK, pair, h.cfg.AccessTokenTTL)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	var raw string
	if cookie, err := r.Cookie(h.cfg.CookieName); err == nil {
		raw = cookie.Value
	}
	err := h.service.Logout(r.Context(), raw, requestIDFromContext(r.Context()), r.UserAgent(), remoteIP(r.RemoteAddr))
	h.clearCookie(w)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) logoutAll(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFromContext(r.Context())
	err := h.service.LogoutAll(r.Context(), p, requestIDFromContext(r.Context()), r.UserAgent(), remoteIP(r.RemoteAddr))
	h.clearCookie(w)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	p, _ := principalFromContext(r.Context())
	user, err := h.service.CurrentUser(r.Context(), p)
	if err != nil {
		h.handleError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

func (h *AuthHandler) rateLimited(scope string, limit int64, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allowed, err := h.limiter.Allow(r.Context(), "auth:"+scope+":"+remoteIP(r.RemoteAddr), limit, h.cfg.RateLimitWindow)
		if err != nil {
			writeAuthError(w, r, auth.ErrServiceUnavailable)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", strconv.Itoa(int(h.cfg.RateLimitWindow.Seconds())))
			writeAuthError(w, r, auth.ErrRateLimited)
			return
		}
		next(w, r)
	}
}

func (h *AuthHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	if domain := auth.AsError(err); domain != nil {
		writeAuthError(w, r, domain)
		return
	}
	h.logger.ErrorContext(r.Context(), "authentication operation failed", "request_id", requestIDFromContext(r.Context()), "error", err)
	writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred.")
}

func writeAuthError(w http.ResponseWriter, r *http.Request, err *auth.Error) {
	writeAPIError(w, r, err.HTTPStatus, err.Code, err.Message)
}

func writeTokenResponse(w http.ResponseWriter, status int, p auth.TokenPair, ttl time.Duration) {
	writeJSON(w, status, map[string]any{"access_token": p.AccessToken, "token_type": "Bearer", "expires_in": int(ttl.Seconds()), "user": p.User, "session": p.Session})
}

func decodeJSON(r *http.Request, destination any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return false
	}
	var extra any
	return errors.Is(decoder.Decode(&extra), io.EOF)
}

func (h *AuthHandler) setCookie(w http.ResponseWriter, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{Name: h.cfg.CookieName, Value: value, Path: "/api/v1/auth", Domain: h.cfg.CookieDomain, Expires: expires, MaxAge: int(time.Until(expires).Seconds()), HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
}
func (h *AuthHandler) clearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: h.cfg.CookieName, Value: "", Path: "/api/v1/auth", Domain: h.cfg.CookieDomain, Expires: time.Unix(1, 0), MaxAge: -1, HttpOnly: true, Secure: h.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
}

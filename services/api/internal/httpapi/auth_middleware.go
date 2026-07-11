package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/klipforge/klipforge/services/api/internal/auth"
)

type principalContextKey struct{}

func (h *AuthHandler) authenticate(next http.Handler) http.Handler {
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
		principal, err := h.service.Authenticate(r.Context(), parts[1])
		if err != nil {
			h.handleError(w, r, err)
			return
		}
		ctx := context.WithValue(r.Context(), principalContextKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func principalFromContext(ctx context.Context) (auth.Principal, bool) {
	p, ok := ctx.Value(principalContextKey{}).(auth.Principal)
	return p, ok
}

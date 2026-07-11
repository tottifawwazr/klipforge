package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get(requestIDHeader))
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}

		w.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

func validRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, character := range value {
		if (character >= 'a' && character <= 'z') ||
			(character >= 'A' && character <= 'Z') ||
			(character >= '0' && character <= '9') ||
			character == '-' || character == '_' || character == '.' {
			continue
		}
		return false
	}
	return true
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err == nil {
		return hex.EncodeToString(bytes[:])
	}
	return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
}

func requestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			wrapped := chimiddleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(wrapped, r)

			status := wrapped.Status()
			if status == 0 {
				status = http.StatusOK
			}
			level := slog.LevelInfo
			if r.URL.Path == "/healthz" || r.URL.Path == "/api/v1/health" {
				level = slog.LevelDebug
			}
			logger.LogAttrs(r.Context(), level, "HTTP request completed",
				slog.String("request_id", requestIDFromContext(r.Context())),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", status),
				slog.Int("bytes", wrapped.BytesWritten()),
				slog.Int64("duration_ms", time.Since(started).Milliseconds()),
				slog.String("remote_ip", remoteIP(r.RemoteAddr)),
			)
		})
	}
}

func recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.ErrorContext(r.Context(), "panic recovered",
						"request_id", requestIDFromContext(r.Context()),
						"panic", fmt.Sprint(recovered),
						"stack", string(debug.Stack()),
					)
					if statusWriter, ok := w.(interface{ Status() int }); !ok || statusWriter.Status() == 0 {
						writeAPIError(w, r, http.StatusInternalServerError, "INTERNAL_SERVER_ERROR", "An unexpected error occurred.")
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func securityHeaders(production bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			headers := w.Header()
			headers.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			headers.Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
			headers.Set("Referrer-Policy", "no-referrer")
			headers.Set("X-Content-Type-Options", "nosniff")
			headers.Set("X-Frame-Options", "DENY")
			if production {
				headers.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

type corsOptions struct {
	allowedOrigins   map[string]struct{}
	allowAnyOrigin   bool
	allowCredentials bool
}

func newCORS(allowedOrigins []string, allowCredentials bool) func(http.Handler) http.Handler {
	options := corsOptions{
		allowedOrigins:   make(map[string]struct{}, len(allowedOrigins)),
		allowCredentials: allowCredentials,
	}
	for _, origin := range allowedOrigins {
		if origin == "*" {
			options.allowAnyOrigin = true
			continue
		}
		options.allowedOrigins[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Add("Vary", "Origin")
			_, explicitlyAllowed := options.allowedOrigins[origin]
			if !options.allowAnyOrigin && !explicitlyAllowed {
				if isPreflight(r) {
					writeAPIError(w, r, http.StatusForbidden, "CORS_ORIGIN_DENIED", "The request origin is not allowed.")
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			if options.allowAnyOrigin {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			if options.allowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if isPreflight(r) {
				w.Header().Add("Vary", "Access-Control-Request-Method")
				w.Header().Add("Vary", "Access-Control-Request-Headers")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Idempotency-Key, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != ""
}

func remoteIP(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return address
	}
	return host
}

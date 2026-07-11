package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := load(mapLookup(nil))
	if err != nil {
		t.Fatalf("load defaults: %v", err)
	}

	if cfg.Environment != "development" {
		t.Fatalf("Environment = %q, want development", cfg.Environment)
	}
	if cfg.Version != "dev" {
		t.Fatalf("Version = %q, want dev", cfg.Version)
	}
	if cfg.HTTP.Address != "0.0.0.0:8080" {
		t.Fatalf("HTTP address = %q, want 0.0.0.0:8080", cfg.HTTP.Address)
	}
	if !reflect.DeepEqual(cfg.HTTP.AllowedOrigins, []string{"http://localhost:3000"}) {
		t.Fatalf("allowed origins = %#v", cfg.HTTP.AllowedOrigins)
	}
	if !cfg.HTTP.AllowCredentials {
		t.Fatal("credentials should be allowed by default")
	}
	if cfg.Health.DependencyTimeout != 2*time.Second {
		t.Fatalf("dependency timeout = %s, want 2s", cfg.Health.DependencyTimeout)
	}
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := load(mapLookup(map[string]string{
		"APP_ENV":                   "production",
		"APP_VERSION":               "1.2.3",
		"LOG_LEVEL":                 "warn",
		"HTTP_ADDR":                 "127.0.0.1:9090",
		"HTTP_REQUEST_TIMEOUT":      "9s",
		"CORS_ALLOWED_ORIGINS":      "https://app.example.com/, https://admin.example.com",
		"CORS_ALLOW_CREDENTIALS":    "false",
		"POSTGRES_MAX_CONNS":        "20",
		"POSTGRES_MIN_CONNS":        "2",
		"REDIS_POOL_SIZE":           "25",
		"REDIS_MIN_IDLE_CONNS":      "3",
		"HEALTH_DEPENDENCY_TIMEOUT": "750ms",
	}))
	if err != nil {
		t.Fatalf("load overrides: %v", err)
	}

	if cfg.Environment != "production" || cfg.Version != "1.2.3" || cfg.LogLevel != "warn" {
		t.Fatalf("unexpected application config: %#v", cfg)
	}
	if cfg.HTTP.Address != "127.0.0.1:9090" || cfg.HTTP.RequestTimeout != 9*time.Second {
		t.Fatalf("unexpected HTTP config: %#v", cfg.HTTP)
	}
	wantOrigins := []string{"https://app.example.com", "https://admin.example.com"}
	if !reflect.DeepEqual(cfg.HTTP.AllowedOrigins, wantOrigins) {
		t.Fatalf("allowed origins = %#v, want %#v", cfg.HTTP.AllowedOrigins, wantOrigins)
	}
	if cfg.Database.MaxConnections != 20 || cfg.Database.MinConnections != 2 {
		t.Fatalf("unexpected database pool config: %#v", cfg.Database)
	}
	if cfg.Redis.PoolSize != 25 || cfg.Redis.MinIdleConnections != 3 {
		t.Fatalf("unexpected Redis pool config: %#v", cfg.Redis)
	}
	if cfg.Health.DependencyTimeout != 750*time.Millisecond {
		t.Fatalf("dependency timeout = %s, want 750ms", cfg.Health.DependencyTimeout)
	}
}

func TestLoadRejectsInvalidConfiguration(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
	}{
		{name: "invalid port", env: map[string]string{"HTTP_PORT": "70000"}},
		{name: "invalid duration", env: map[string]string{"HTTP_READ_TIMEOUT": "soon"}},
		{name: "invalid log level", env: map[string]string{"LOG_LEVEL": "verbose"}},
		{name: "invalid origin", env: map[string]string{"CORS_ALLOWED_ORIGINS": "example.com"}},
		{name: "wildcard credentials", env: map[string]string{"CORS_ALLOWED_ORIGINS": "*"}},
		{name: "minimum database pool exceeds maximum", env: map[string]string{"POSTGRES_MIN_CONNS": "11"}},
		{name: "minimum Redis pool exceeds maximum", env: map[string]string{"REDIS_MIN_IDLE_CONNS": "11"}},
		{name: "zero body limit", env: map[string]string{"HTTP_MAX_REQUEST_BODY_BYTES": "0"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := load(mapLookup(test.env)); err == nil {
				t.Fatal("load() error = nil, want validation error")
			}
		})
	}
}

func TestLoadAllowsWildcardWithoutCredentials(t *testing.T) {
	cfg, err := load(mapLookup(map[string]string{
		"CORS_ALLOWED_ORIGINS":   "*",
		"CORS_ALLOW_CREDENTIALS": "false",
	}))
	if err != nil {
		t.Fatalf("load wildcard CORS: %v", err)
	}
	if !reflect.DeepEqual(cfg.HTTP.AllowedOrigins, []string{"*"}) {
		t.Fatalf("allowed origins = %#v", cfg.HTTP.AllowedOrigins)
	}
}

func mapLookup(values map[string]string) lookupFunc {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}

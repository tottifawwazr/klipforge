package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

type checkerFunc func(context.Context) error

func (check checkerFunc) Ping(ctx context.Context) error {
	return check(ctx)
}

func TestReadinessAllDependenciesUp(t *testing.T) {
	service := newTestService(t,
		checkerFunc(func(context.Context) error { return nil }),
		checkerFunc(func(context.Context) error { return nil }),
		50*time.Millisecond,
	)

	report := service.Readiness(context.Background())

	if report.Status != StatusOK {
		t.Fatalf("status = %q, want %q", report.Status, StatusOK)
	}
	if report.Service != "klipforge-api" || report.Version != "test-version" {
		t.Fatalf("unexpected identity: service=%q version=%q", report.Service, report.Version)
	}
	if report.Timestamp != "2026-07-11T03:04:05Z" {
		t.Fatalf("timestamp = %q", report.Timestamp)
	}
	if report.Checks.Postgres.Status != CheckUp || report.Checks.Redis.Status != CheckUp {
		t.Fatalf("checks = %#v, want both up", report.Checks)
	}
	if report.Checks.Postgres.LatencyMS < 0 || report.Checks.Redis.LatencyMS < 0 {
		t.Fatalf("latencies must be non-negative: %#v", report.Checks)
	}
}

func TestReadinessDegradedWhenDependencyFails(t *testing.T) {
	service := newTestService(t,
		checkerFunc(func(context.Context) error { return errors.New("database unavailable") }),
		checkerFunc(func(context.Context) error { return nil }),
		50*time.Millisecond,
	)

	report := service.Readiness(context.Background())

	if report.Status != StatusDegraded {
		t.Fatalf("status = %q, want %q", report.Status, StatusDegraded)
	}
	if report.Checks.Postgres.Status != CheckDown {
		t.Fatalf("PostgreSQL status = %q, want down", report.Checks.Postgres.Status)
	}
	if report.Checks.Redis.Status != CheckUp {
		t.Fatalf("Redis status = %q, want up", report.Checks.Redis.Status)
	}
}

func TestReadinessBoundsBlockingChecksByTimeout(t *testing.T) {
	blocking := checkerFunc(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	})
	service := newTestService(t, blocking, blocking, 10*time.Millisecond)

	started := time.Now()
	report := service.Readiness(context.Background())
	elapsed := time.Since(started)

	if elapsed > 500*time.Millisecond {
		t.Fatalf("readiness took %s, want a bounded check", elapsed)
	}
	if report.Status != StatusDegraded || report.Checks.Postgres.Status != CheckDown || report.Checks.Redis.Status != CheckDown {
		t.Fatalf("unexpected timed-out report: %#v", report)
	}
}

func TestNewServiceValidatesDependencies(t *testing.T) {
	valid := checkerFunc(func(context.Context) error { return nil })
	tests := []struct {
		name     string
		service  string
		version  string
		postgres Checker
		redis    Checker
		timeout  time.Duration
	}{
		{name: "service name", version: "dev", postgres: valid, redis: valid, timeout: time.Second},
		{name: "version", service: "api", postgres: valid, redis: valid, timeout: time.Second},
		{name: "PostgreSQL", service: "api", version: "dev", redis: valid, timeout: time.Second},
		{name: "Redis", service: "api", version: "dev", postgres: valid, timeout: time.Second},
		{name: "timeout", service: "api", version: "dev", postgres: valid, redis: valid},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := NewService(test.service, test.version, test.postgres, test.redis, test.timeout); err == nil {
				t.Fatal("NewService() error = nil, want validation error")
			}
		})
	}
}

func newTestService(t *testing.T, postgres, redis Checker, timeout time.Duration) *Service {
	t.Helper()
	service, err := NewService("klipforge-api", "test-version", postgres, redis, timeout)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}
	service.now = func() time.Time {
		return time.Date(2026, 7, 11, 3, 4, 5, 0, time.UTC)
	}
	return service
}

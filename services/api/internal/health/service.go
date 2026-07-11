// Package health contains dependency readiness logic independently of HTTP.
package health

import (
	"context"
	"fmt"
	"time"
)

const (
	StatusOK       = "ok"
	StatusDegraded = "degraded"
	CheckUp        = "up"
	CheckDown      = "down"
)

type Checker interface {
	Ping(context.Context) error
}

type DependencyCheck struct {
	Status    string `json:"status"`
	LatencyMS int64  `json:"latency_ms"`
}

type Checks struct {
	Postgres DependencyCheck `json:"postgres"`
	Redis    DependencyCheck `json:"redis"`
}

type Report struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
	Checks    Checks `json:"checks"`
}

type Service struct {
	serviceName string
	version     string
	postgres    Checker
	redis       Checker
	timeout     time.Duration
	now         func() time.Time
}

func NewService(serviceName, version string, postgres, redis Checker, timeout time.Duration) (*Service, error) {
	if serviceName == "" {
		return nil, fmt.Errorf("health service name is required")
	}
	if version == "" {
		return nil, fmt.Errorf("health service version is required")
	}
	if postgres == nil {
		return nil, fmt.Errorf("PostgreSQL health checker is required")
	}
	if redis == nil {
		return nil, fmt.Errorf("Redis health checker is required")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("health dependency timeout must be positive")
	}

	return &Service{
		serviceName: serviceName,
		version:     version,
		postgres:    postgres,
		redis:       redis,
		timeout:     timeout,
		now:         time.Now,
	}, nil
}

type checkResult struct {
	name      string
	status    string
	latencyMS int64
}

// Readiness checks PostgreSQL and Redis concurrently so the endpoint's worst
// case duration is bounded by one dependency timeout rather than their sum.
func (s *Service) Readiness(ctx context.Context) Report {
	checkCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	started := time.Now()
	results := make(chan checkResult, 2)
	go runCheck(checkCtx, "postgres", s.postgres, results)
	go runCheck(checkCtx, "redis", s.redis, results)

	checks := Checks{
		Postgres: DependencyCheck{Status: CheckDown},
		Redis:    DependencyCheck{Status: CheckDown},
	}
	received := make(map[string]bool, 2)

	for len(received) < 2 {
		select {
		case result := <-results:
			if received[result.name] {
				continue
			}
			received[result.name] = true
			switch result.name {
			case "postgres":
				checks.Postgres = DependencyCheck{Status: result.status, LatencyMS: result.latencyMS}
			case "redis":
				checks.Redis = DependencyCheck{Status: result.status, LatencyMS: result.latencyMS}
			}
		case <-checkCtx.Done():
			elapsed := nonNegativeMilliseconds(time.Since(started))
			if !received["postgres"] {
				checks.Postgres.LatencyMS = elapsed
			}
			if !received["redis"] {
				checks.Redis.LatencyMS = elapsed
			}
			return s.report(checks)
		}
	}

	return s.report(checks)
}

func runCheck(ctx context.Context, name string, checker Checker, results chan<- checkResult) {
	started := time.Now()
	err := checker.Ping(ctx)
	status := CheckUp
	if err != nil {
		status = CheckDown
	}
	results <- checkResult{
		name:      name,
		status:    status,
		latencyMS: nonNegativeMilliseconds(time.Since(started)),
	}
}

func nonNegativeMilliseconds(duration time.Duration) int64 {
	if duration < 0 {
		return 0
	}
	return duration.Milliseconds()
}

func (s *Service) report(checks Checks) Report {
	status := StatusOK
	if checks.Postgres.Status != CheckUp || checks.Redis.Status != CheckUp {
		status = StatusDegraded
	}
	return Report{
		Status:    status,
		Service:   s.serviceName,
		Version:   s.version,
		Timestamp: s.now().UTC().Format(time.RFC3339),
		Checks:    checks,
	}
}

// Package migration applies ordered SQL migrations and records each applied version.
package migration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var migrationFilename = regexp.MustCompile(`^(\d+)_(.+)\.(up|down)\.sql$`)

type Migration struct {
	Version  int64
	Name     string
	UpPath   string
	DownPath string
}

type Status struct {
	Version int64
	Name    string
	Applied bool
}

type Runner struct {
	pool          *pgxpool.Pool
	migrationsDir string
}

func New(ctx context.Context, databaseURL, migrationsDir string) (*Runner, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create migration PostgreSQL pool: %w", err)
	}

	return &Runner{pool: pool, migrationsDir: migrationsDir}, nil
}

func (r *Runner) Close() {
	r.pool.Close()
}

func (r *Runner) Up(ctx context.Context) (int, error) {
	if err := r.ensureLedger(ctx); err != nil {
		return 0, err
	}

	migrations, err := load(r.migrationsDir)
	if err != nil {
		return 0, err
	}
	applied, err := r.appliedVersions(ctx)
	if err != nil {
		return 0, err
	}

	count := 0
	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}
		if err := r.applyUp(ctx, migration); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

func (r *Runner) Down(ctx context.Context) (*Migration, error) {
	if err := r.ensureLedger(ctx); err != nil {
		return nil, err
	}

	var version int64
	var name string
	err := r.pool.QueryRow(ctx, `SELECT version, name FROM schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version, &name)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read latest migration: %w", err)
	}

	migrations, err := load(r.migrationsDir)
	if err != nil {
		return nil, err
	}
	for _, migration := range migrations {
		if migration.Version == version {
			if migration.Name != name {
				return nil, fmt.Errorf("migration %d name mismatch: database=%q files=%q", version, name, migration.Name)
			}
			if err := r.applyDown(ctx, migration); err != nil {
				return nil, err
			}
			return &migration, nil
		}
	}

	return nil, fmt.Errorf("applied migration %d_%s is missing from %s", version, name, r.migrationsDir)
}

func (r *Runner) Status(ctx context.Context) ([]Status, error) {
	if err := r.ensureLedger(ctx); err != nil {
		return nil, err
	}

	migrations, err := load(r.migrationsDir)
	if err != nil {
		return nil, err
	}
	applied, err := r.appliedVersions(ctx)
	if err != nil {
		return nil, err
	}

	statuses := make([]Status, 0, len(migrations))
	for _, migration := range migrations {
		statuses = append(statuses, Status{
			Version: migration.Version,
			Name:    migration.Name,
			Applied: applied[migration.Version],
		})
	}
	return statuses, nil
}

func (r *Runner) ensureLedger(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY CHECK (version > 0),
			name text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT NOW()
		)`)
	if err != nil {
		return fmt.Errorf("create schema migration ledger: %w", err)
	}
	return nil
}

func (r *Runner) appliedVersions(ctx context.Context) (map[int64]bool, error) {
	rows, err := r.pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan applied migration: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applied migrations: %w", err)
	}
	return applied, nil
}

func (r *Runner) applyUp(ctx context.Context, migration Migration) error {
	sql, err := os.ReadFile(migration.UpPath)
	if err != nil {
		return fmt.Errorf("read up migration %d: %w", migration.Version, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin up migration %d: %w", migration.Version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("execute up migration %d_%s: %w", migration.Version, migration.Name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, migration.Version, migration.Name); err != nil {
		return fmt.Errorf("record up migration %d: %w", migration.Version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit up migration %d: %w", migration.Version, err)
	}
	return nil
}

func (r *Runner) applyDown(ctx context.Context, migration Migration) error {
	sql, err := os.ReadFile(migration.DownPath)
	if err != nil {
		return fmt.Errorf("read down migration %d: %w", migration.Version, err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin down migration %d: %w", migration.Version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("execute down migration %d_%s: %w", migration.Version, migration.Name, err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, migration.Version); err != nil {
		return fmt.Errorf("remove migration ledger entry %d: %w", migration.Version, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit down migration %d: %w", migration.Version, err)
	}
	return nil
}

func load(dir string) ([]Migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations directory %s: %w", dir, err)
	}

	byVersion := make(map[int64]*Migration)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		matches := migrationFilename.FindStringSubmatch(entry.Name())
		if matches == nil {
			continue
		}

		version, err := strconv.ParseInt(matches[1], 10, 64)
		if err != nil || version < 1 {
			return nil, fmt.Errorf("invalid migration version in %s", entry.Name())
		}
		name := matches[2]
		direction := matches[3]

		migration, ok := byVersion[version]
		if !ok {
			migration = &Migration{Version: version, Name: name}
			byVersion[version] = migration
		}
		if migration.Name != name {
			return nil, fmt.Errorf("migration version %d has conflicting names %q and %q", version, migration.Name, name)
		}

		path := filepath.Join(dir, entry.Name())
		if direction == "up" {
			if migration.UpPath != "" {
				return nil, fmt.Errorf("migration %d has more than one up file", version)
			}
			migration.UpPath = path
		} else {
			if migration.DownPath != "" {
				return nil, fmt.Errorf("migration %d has more than one down file", version)
			}
			migration.DownPath = path
		}
	}

	versions := make([]int64, 0, len(byVersion))
	for version, migration := range byVersion {
		if migration.UpPath == "" || migration.DownPath == "" {
			return nil, fmt.Errorf("migration %d_%s must have both up and down files", version, migration.Name)
		}
		versions = append(versions, version)
	}
	sort.Slice(versions, func(i, j int) bool { return versions[i] < versions[j] })

	migrations := make([]Migration, 0, len(versions))
	for _, version := range versions {
		migrations = append(migrations, *byVersion[version])
	}
	if len(migrations) == 0 {
		return nil, fmt.Errorf("no migration files found in %s", dir)
	}

	return migrations, nil
}

func FormatStatuses(statuses []Status) string {
	lines := make([]string, 0, len(statuses))
	for _, status := range statuses {
		state := "pending"
		if status.Applied {
			state = "applied"
		}
		lines = append(lines, fmt.Sprintf("%06d %-8s %s", status.Version, state, status.Name))
	}
	return strings.Join(lines, "\n")
}

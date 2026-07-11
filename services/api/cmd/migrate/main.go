package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/klipforge/klipforge/services/api/internal/migration"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: klipforge-migrate <up|down|status>")
	}

	databaseURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	migrationsDir := strings.TrimSpace(os.Getenv("MIGRATIONS_DIR"))
	if migrationsDir == "" {
		migrationsDir = "/migrations"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	runner, err := migration.New(ctx, databaseURL, migrationsDir)
	if err != nil {
		return err
	}
	defer runner.Close()

	switch os.Args[1] {
	case "up":
		count, err := runner.Up(ctx)
		if err != nil {
			return err
		}
		fmt.Printf("Applied %d migration(s).\n", count)
	case "down":
		applied, err := runner.Down(ctx)
		if err != nil {
			return err
		}
		if applied == nil {
			fmt.Println("No applied migrations to roll back.")
			return nil
		}
		fmt.Printf("Rolled back %06d_%s.\n", applied.Version, applied.Name)
	case "status":
		statuses, err := runner.Status(ctx)
		if err != nil {
			return err
		}
		fmt.Println(migration.FormatStatuses(statuses))
	default:
		return fmt.Errorf("unsupported migration action %q; use up, down, or status", os.Args[1])
	}

	return nil
}

// Command backfill populates jobs.current_stage_template_id for the
// single-pipeline (Status/Stage) refactor introduced by migration 000040.
//
// It is a thin CLI wrapper around internal/backfill, which holds the shared,
// idempotent core (also invoked as a guarded step during cmd/api startup).
//
// Usage:
//
//	go run ./cmd/backfill                # apply and commit
//	go run ./cmd/backfill --dry-run      # run everything, then ROLL BACK
//
// Connection follows the same pattern as cmd/seed and internal/platform/postgres:
// config.Load() -> DatabaseConfig.DSN() -> pgxpool. A .env file is loaded if
// present so the command works in local dev the same way the API does.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/andreypavlenko/jobber/internal/backfill"
	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "run the full backfill inside a transaction and roll it back, printing what WOULD change")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(context.Background(), logger, *dryRun); err != nil {
		logger.Error("backfill failed", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, logger *slog.Logger, dryRun bool) error {
	// Load a local .env if present (mirrors cmd/seed), then read config.
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	pool, err := pgxpool.New(ctx, cfg.Database.DSN())
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}
	logger.Info("connected to database",
		"host", cfg.Database.Host, "db", cfg.Database.DBName, "dry_run", dryRun)

	if dryRun {
		_, err = backfill.RunDry(ctx, logger, pool)
		return err
	}
	_, err = backfill.Run(ctx, logger, pool)
	return err
}

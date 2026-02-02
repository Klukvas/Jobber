package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/andreypavlenko/jobber/internal/config"
	"github.com/andreypavlenko/jobber/internal/platform/logger"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"go.uber.org/zap"
)

// RunMigrations executes database migrations
func RunMigrations(ctx context.Context, cfg config.DatabaseConfig, log *logger.Logger, migrationsPath string) error {
	log.Info("Starting database migrations", zap.String("path", migrationsPath))

	// Build migration source URL
	sourceURL := fmt.Sprintf("file://%s", migrationsPath)
	
	// Build database URL for migrations
	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.Port,
		cfg.DBName,
		cfg.SSLMode,
	)

	// Create migrator instance
	m, err := migrate.New(sourceURL, databaseURL)
	if err != nil {
		log.Error("Failed to create migrator", zap.Error(err))
		return fmt.Errorf("failed to create migrator: %w", err)
	}
	defer m.Close()

	// Run migrations
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Info("Database schema is already up to date")
			return nil
		}
		
		// Get current version for debugging
		version, dirty, vErr := m.Version()
		if vErr != nil {
			log.Error("Failed to get migration version", zap.Error(vErr))
		} else {
			log.Error("Migration failed",
				zap.Error(err),
				zap.Uint("version", version),
				zap.Bool("dirty", dirty),
			)
		}
		
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Get final version
	version, dirty, err := m.Version()
	if err != nil {
		log.Warn("Could not get migration version after completion", zap.Error(err))
	} else {
		log.Info("Database migrations completed successfully",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	return nil
}

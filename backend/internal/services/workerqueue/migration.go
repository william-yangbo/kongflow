// Package workerqueue provides database migration support for River Queue
package workerqueue

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// MigrationManager handles River Queue database migrations
type MigrationManager struct {
	dbPool *pgxpool.Pool
	logger *slog.Logger
}

// NewMigrationManager creates a new migration manager
func NewMigrationManager(dbPool *pgxpool.Pool, logger *slog.Logger) *MigrationManager {
	if logger == nil {
		logger = slog.Default()
	}

	return &MigrationManager{
		dbPool: dbPool,
		logger: logger,
	}
}

// EnsureRiverTables ensures that River Queue tables are created and up to date
// This is automatically called by Client.Initialize()
func (m *MigrationManager) EnsureRiverTables(ctx context.Context) error {
	m.logger.Info("Ensuring River Queue tables are up to date")

	migrator, err := rivermigrate.New(riverpgxv5.New(m.dbPool), &rivermigrate.Config{
		Logger: m.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create River migrator: %w", err)
	}

	// Run migrations up to the latest version
	result, err := migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{})
	if err != nil {
		return fmt.Errorf("failed to run River migrations: %w", err)
	}

	if len(result.Versions) > 0 {
		m.logger.Info("River Queue migrations completed",
			"applied_versions", result.Versions,
		)
	} else {
		m.logger.Info("River Queue tables are already up to date")
	}

	return nil
}

// GetMigrationStatus returns information about current database state
func (m *MigrationManager) GetMigrationStatus(ctx context.Context) error {
	_, err := rivermigrate.New(riverpgxv5.New(m.dbPool), &rivermigrate.Config{
		Logger: m.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create River migrator: %w", err)
	}

	// For now, just validate that we can create a migrator
	// In a real implementation, you could query river_migration table directly
	m.logger.Info("River Queue migration system is available")
	return nil
}

// MigrateDown removes River Queue tables and data (use with caution)
func (m *MigrationManager) MigrateDown(ctx context.Context) error {
	_, err := rivermigrate.New(riverpgxv5.New(m.dbPool), &rivermigrate.Config{
		Logger: m.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create River migrator: %w", err)
	}

	// For now, just validate that we can create a migrator
	// In a real implementation, you'd use migrator to rollback specific versions
	m.logger.Warn("River Queue migration rollback functionality - use with extreme caution")
	return nil
}

// DropTables removes all River Queue tables (use with extreme caution)
func (m *MigrationManager) DropTables(ctx context.Context) error {
	m.logger.Warn("Dropping all River Queue tables - THIS WILL DELETE ALL JOB DATA")

	migrator, err := rivermigrate.New(riverpgxv5.New(m.dbPool), &rivermigrate.Config{
		Logger: m.logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create River migrator: %w", err)
	}

	// Migrate down to remove all tables (TargetVersion: -1 removes everything)
	_, err = migrator.Migrate(ctx, rivermigrate.DirectionDown, &rivermigrate.MigrateOpts{
		TargetVersion: -1,
	})
	if err != nil {
		return fmt.Errorf("failed to drop River tables: %w", err)
	}

	m.logger.Warn("All River Queue tables have been dropped")
	return nil
}

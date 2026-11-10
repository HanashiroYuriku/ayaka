package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"be-ayaka/migrations"

	"github.com/pressly/goose/v3"
)

// Migrator runs the embedded SQL migrations.
type Migrator struct {
	provider *goose.Provider
}

func NewMigrator(db *sql.DB) (*Migrator, error) {
	p, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		return nil, fmt.Errorf("create migration provider: %w", err)
	}
	return &Migrator{provider: p}, nil
}

// Up applies all pending migrations.
func (m *Migrator) Up(ctx context.Context, log *slog.Logger) error {
	results, err := m.provider.Up(ctx)
	for _, r := range results {
		log.Info("migration applied", "migration", r.Source.Version, "file", r.Source.Path, "duration", r.Duration)
	}
	if err != nil {
		return fmt.Errorf("migrate up: %w", err)
	}
	if len(results) == 0 {
		log.Info("database schema is up to date")
	}
	return nil
}

// Down rolls back the most recent migration.
func (m *Migrator) Down(ctx context.Context, log *slog.Logger) error {
	r, err := m.provider.Down(ctx)
	if err != nil {
		return fmt.Errorf("migrate down: %w", err)
	}
	if r != nil {
		log.Info("migration rolled back", "migration", r.Source.Version, "file", r.Source.Path)
	}
	return nil
}

// Status prints the state of every migration.
func (m *Migrator) Status(ctx context.Context) ([]*goose.MigrationStatus, error) {
	return m.provider.Status(ctx)
}

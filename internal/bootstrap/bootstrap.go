// Package bootstrap wires the application together and manages its lifecycle.
package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"be-ayaka/config"
	"be-ayaka/internal/adapter/database"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"
)

// App is a fully wired, ready-to-run server.
type App struct {
	cfg   *config.Config
	log   *slog.Logger
	db    *gorm.DB
	fiber *fiber.App
}

// Options tweak start-up behaviour.
type Options struct {
	// Migrate applies pending migrations before serving traffic.
	Migrate bool
}

// New connects to dependencies and builds the HTTP server.
func New(ctx context.Context, cfg *config.Config, log *slog.Logger, opts Options) (*App, error) {
	db, err := database.NewPostgres(ctx, cfg.Database, log)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		_ = database.Close(db)
		return nil, err
	}

	if opts.Migrate {
		m, err := database.NewMigrator(sqlDB)
		if err == nil {
			err = m.Up(ctx, log)
		}
		if err != nil {
			_ = database.Close(db)
			return nil, err
		}
	}

	app := NewFiber(cfg, log)
	RegisterRoutes(app, cfg, NewContainer(cfg, db, sqlDB))

	return &App{cfg: cfg, log: log, db: db, fiber: app}, nil
}

// Run serves HTTP until ctx is cancelled (e.g. SIGINT/SIGTERM) or the
// listener fails, then shuts down gracefully.
func (a *App) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", a.cfg.Server.Port)
	serveErr := make(chan error, 1)
	go func() {
		a.log.Info("http server listening", "addr", addr)
		serveErr <- a.fiber.Listen(addr)
	}()

	var runErr error
	select {
	case err := <-serveErr:
		runErr = fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
		a.log.Info("shutdown signal received")
	}

	return errors.Join(runErr, a.shutdown())
}

func (a *App) shutdown() error {
	var errs []error

	a.log.Info("stopping http server", "timeout", a.cfg.Server.ShutdownTimeout)
	if err := a.fiber.ShutdownWithTimeout(a.cfg.Server.ShutdownTimeout); err != nil {
		errs = append(errs, fmt.Errorf("shutdown http server: %w", err))
	}

	if err := database.Close(a.db); err != nil {
		errs = append(errs, fmt.Errorf("close database: %w", err))
	}

	if len(errs) == 0 {
		a.log.Info("shutdown complete")
	}
	return errors.Join(errs...)
}

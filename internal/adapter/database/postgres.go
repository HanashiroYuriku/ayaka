// Package database opens and configures the PostgreSQL connection pool.
package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"be-ayaka/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const (
	connectAttempts = 5
	connectBackoff  = 2 * time.Second
	pingTimeout     = 5 * time.Second
)

// NewPostgres opens a GORM connection, configures the pool and verifies it,
// retrying a few times so the app tolerates a database that starts slowly
// (e.g. docker compose).
func NewPostgres(ctx context.Context, cfg config.DatabaseConfig, log *slog.Logger) (*gorm.DB, error) {
	gormLog := gormlogger.NewSlogLogger(log.With("component", "gorm"), gormlogger.Config{
		SlowThreshold:             cfg.SlowQuery,
		LogLevel:                  gormlogger.Warn,
		IgnoreRecordNotFoundError: true,
		ParameterizedQueries:      true, // never log query parameters (may contain PII)
	})

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger:         gormLog,
		TranslateError: true, // map driver errors to gorm.ErrDuplicatedKey, ...
		NowFunc:        func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	for attempt := 1; ; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
		err = sqlDB.PingContext(pingCtx)
		cancel()
		if err == nil {
			break
		}
		if attempt == connectAttempts {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("ping database after %d attempts: %w", attempt, err)
		}
		log.Warn("database not ready, retrying", "attempt", attempt, "error", err)
		select {
		case <-ctx.Done():
			_ = sqlDB.Close()
			return nil, ctx.Err()
		case <-time.After(connectBackoff * time.Duration(attempt)):
		}
	}

	log.Info("database connected", "host", cfg.Host, "name", cfg.Name)
	return db, nil
}

// Close closes the underlying connection pool.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

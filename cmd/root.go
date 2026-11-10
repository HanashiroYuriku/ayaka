// Package cmd defines the CLI entry points (serve, migrate, ...).
package cmd

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"be-ayaka/config"
	"be-ayaka/pkg/logger"

	"github.com/spf13/cobra"
)

var configPath string

var rootCmd = &cobra.Command{
	Use:           "ayaka",
	Short:         "Ayaka Backend Service",
	Long:          "Ayaka is a production-ready Go backend template (Fiber + GORM + PostgreSQL).",
	SilenceUsage:  true, // don't print usage on runtime errors
	SilenceErrors: true, // main prints the error once
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.ExecuteContext(context.Background())
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "config.yaml", "path to the config file")
	rootCmd.AddCommand(serveCmd, migrateCmd, userCmd, tokensCmd)
}

// setup loads the configuration and builds the global logger.
func setup() (*config.Config, *slog.Logger, io.Closer, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("load config: %w", err)
	}

	log, closer, err := logger.New(logger.Options{
		Level:      cfg.Log.Level,
		Format:     cfg.Log.Format,
		FilePath:   cfg.Log.FilePath,
		MaxSizeMB:  cfg.Log.MaxSize,
		MaxBackups: cfg.Log.MaxBackups,
		MaxAgeDays: cfg.Log.MaxAge,
		Compress:   cfg.Log.Compress,
		Attrs: []slog.Attr{
			slog.String("app", cfg.App.Name),
			slog.String("version", cfg.App.Version),
			slog.String("env", cfg.App.Env),
		},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("init logger: %w", err)
	}
	slog.SetDefault(log)

	return cfg, log, closer, nil
}

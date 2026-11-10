package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	"be-ayaka/internal/adapter/database"

	"github.com/spf13/cobra"
)

const migrationsDir = "migrations"

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Manage database schema migrations",
}

func init() {
	migrateCmd.AddCommand(
		&cobra.Command{
			Use:   "up",
			Short: "Apply all pending migrations",
			RunE:  withMigrator(func(cmd *cobra.Command, m *database.Migrator) error { return m.Up(cmd.Context(), slog.Default()) }),
		},
		&cobra.Command{
			Use:   "down",
			Short: "Roll back the most recent migration",
			RunE:  withMigrator(func(cmd *cobra.Command, m *database.Migrator) error { return m.Down(cmd.Context(), slog.Default()) }),
		},
		&cobra.Command{
			Use:   "status",
			Short: "Show the status of every migration",
			RunE: withMigrator(func(cmd *cobra.Command, m *database.Migrator) error {
				statuses, err := m.Status(cmd.Context())
				if err != nil {
					return err
				}
				w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
				fmt.Fprintln(w, "VERSION\tSTATE\tAPPLIED AT\tFILE")
				for _, s := range statuses {
					applied := "-"
					if !s.AppliedAt.IsZero() {
						applied = s.AppliedAt.Format(time.RFC3339)
					}
					fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", s.Source.Version, s.State, applied, filepath.Base(s.Source.Path))
				}
				return w.Flush()
			}),
		},
		&cobra.Command{
			Use:   "create <name>",
			Short: "Create a new empty SQL migration file",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				return createMigration(args[0])
			},
		},
	)
}

// withMigrator connects to the database and runs fn with a migrator.
func withMigrator(fn func(*cobra.Command, *database.Migrator) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cfg, log, closer, err := setup()
		if err != nil {
			return err
		}
		defer closer.Close()

		db, err := database.NewPostgres(cmd.Context(), cfg.Database, log)
		if err != nil {
			return err
		}
		defer database.Close(db)

		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		m, err := database.NewMigrator(sqlDB)
		if err != nil {
			return err
		}
		return fn(cmd, m)
	}
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)

func createMigration(name string) error {
	slug := strings.Trim(nonAlnum.ReplaceAllString(strings.ToLower(name), "_"), "_")
	if slug == "" {
		return fmt.Errorf("invalid migration name %q", name)
	}

	file := filepath.Join(migrationsDir, fmt.Sprintf("%s_%s.sql", time.Now().UTC().Format("20060102150405"), slug))
	content := "-- +goose Up\n\n\n-- +goose Down\n\n"
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		return fmt.Errorf("create migration: %w", err)
	}
	fmt.Println("created", file)
	return nil
}

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"be-ayaka/internal/adapter/database"
	"be-ayaka/internal/bootstrap"
	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/delivery/http/dto"

	"github.com/spf13/cobra"
)

// adminPasswordEnv lets scripts pass the password without exposing it in the
// process list or shell history (prefer it over --password).
const adminPasswordEnv = "ADMIN_PASSWORD"

var (
	adminEmail       string
	adminUsername    string
	adminDisplayName string
	adminPassword    string
)

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "Manage users",
}

var createAdminCmd = &cobra.Command{
	Use:   "create-admin",
	Short: "Create a verified administrator account",
	Example: `  ADMIN_PASSWORD='S3cure!pass' ayaka user create-admin --email admin@example.com --username admin
  make create-admin email=admin@example.com username=admin`,
	RunE: withContainer(func(ctx context.Context, c *bootstrap.Container) error {
		password := adminPassword
		if password == "" {
			password = os.Getenv(adminPasswordEnv)
		}
		if password == "" {
			return errors.New("password required: set " + adminPasswordEnv + " (recommended) or --password")
		}

		req := dto.RegisterRequest{
			Username: adminUsername, Email: adminEmail, DisplayName: adminDisplayName, Password: password,
		}
		req.Normalize()
		// Same rules as the public registration endpoint.
		if err := c.Validator.Validate(ctx, &req); err != nil {
			return describe(err)
		}

		admin, err := c.AuthService.CreateAdmin(ctx, req.ToInput())
		if err != nil {
			return describe(err)
		}
		fmt.Printf("admin created: id=%s email=%s username=%s\n", admin.ID, admin.Email, admin.Username)
		return nil
	}),
}

var tokensCmd = &cobra.Command{
	Use:   "tokens",
	Short: "Maintain authentication tokens",
}

var pruneRetention time.Duration

var pruneTokensCmd = &cobra.Command{
	Use:   "prune",
	Short: "Delete expired, revoked and used tokens (run periodically, e.g. daily cron)",
	RunE: withContainer(func(ctx context.Context, c *bootstrap.Container) error {
		before := time.Now().Add(-pruneRetention)
		refresh, err := c.RefreshTokens.DeleteStale(ctx, before)
		if err != nil {
			return err
		}
		oneTime, err := c.UserTokens.DeleteStale(ctx, before)
		if err != nil {
			return err
		}
		fmt.Printf("deleted %d refresh tokens and %d one-time tokens older than %s\n", refresh, oneTime, before.Format(time.RFC3339))
		return nil
	}),
}

func init() {
	createAdminCmd.Flags().StringVar(&adminEmail, "email", "", "admin email (required)")
	createAdminCmd.Flags().StringVar(&adminUsername, "username", "", "admin username (required)")
	createAdminCmd.Flags().StringVar(&adminDisplayName, "display-name", "", "display name")
	createAdminCmd.Flags().StringVar(&adminPassword, "password", "", "password (prefer the "+adminPasswordEnv+" env var)")
	_ = createAdminCmd.MarkFlagRequired("email")
	_ = createAdminCmd.MarkFlagRequired("username")
	userCmd.AddCommand(createAdminCmd)

	// Keep revoked tokens for a while: useful when investigating a reuse alert.
	pruneTokensCmd.Flags().DurationVar(&pruneRetention, "retention", 7*24*time.Hour, "keep tokens that became stale more recently than this")
	tokensCmd.AddCommand(pruneTokensCmd)
}

// withContainer connects to the database, builds the container and runs fn.
func withContainer(fn func(context.Context, *bootstrap.Container) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, _ []string) error {
		cfg, log, closer, err := setup()
		if err != nil {
			return err
		}
		defer closer.Close()

		ctx := cmd.Context()
		db, err := database.NewPostgres(ctx, cfg.Database, log)
		if err != nil {
			return err
		}
		defer database.Close(db)

		sqlDB, err := db.DB()
		if err != nil {
			return err
		}
		c, err := bootstrap.NewContainer(cfg, db, sqlDB, log)
		if err != nil {
			return err
		}
		defer c.Close(context.Background())

		return fn(ctx, c)
	}
}

// describe turns an application error into a readable CLI error, including field details.
func describe(err error) error {
	ae := apperror.As(err)
	if len(ae.Fields) == 0 {
		return err
	}
	msg := ae.PublicMessage()
	for field, detail := range ae.Fields {
		msg += fmt.Sprintf("\n  - %s: %s", field, detail)
	}
	return errors.New(msg)
}

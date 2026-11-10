package cmd

import (
	"fmt"
	"os/signal"
	"syscall"

	"be-ayaka/internal/bootstrap"

	"github.com/spf13/cobra"
)

var migrateOnStart bool

var serveCmd = &cobra.Command{
	Use:     "serve",
	Aliases: []string{"svc", "service"},
	Short:   "Start the HTTP server",
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, log, closer, err := setup()
		if err != nil {
			return err
		}
		defer closer.Close()

		// Cancelled on SIGINT/SIGTERM -> graceful shutdown.
		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()

		app, err := bootstrap.New(ctx, cfg, log, bootstrap.Options{Migrate: migrateOnStart})
		if err != nil {
			return err
		}

		printBanner(cfg.Server.Port)
		return app.Run(ctx)
	},
}

func init() {
	serveCmd.Flags().BoolVar(&migrateOnStart, "migrate", false, "apply pending database migrations before starting")
}

func printBanner(port int) {
	fmt.Printf(`
     ░█▀▀█ ░█──░█ ░█▀▀█ ░█─▄▀ ░█▀▀█
     ▒█▄▄█ ░█▄▄▄█ ▒█▄▄█ ░█▀▄─ ▒█▄▄█
     ▒█─▒█ ──░█── ▒█─▒█ ░█─░█ ▒█─▒█
    Running on port %d
`+"\n", port)
}

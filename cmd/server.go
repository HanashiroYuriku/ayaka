package cmd

import (
	"fmt"
	"log"

	"be-ayaka/config"
	"be-ayaka/internal/bootstrap"
	ayaka "be-ayaka/pkg/logger"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:     "service",
	Aliases: []string{"svc"},
	Short: "Start the Ayaka backend service",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🌸 Preparing Ayaka Server...")

		// Step 1: Load configuration
		cfg, err := config.LoadConfig()
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		
		// Step 2: Set up logger
		ayaka.SetUp(cfg)	
		ayaka.Log("SYSTEM", "INFO", "Config & Logger Already Loaded", "unknown-request-id")

		// Step 3: Run bootstrap
		ayaka.Log("SYSTEM", "WARN", "Preparing Bootstrap & Fiber", "unknown-request-id")
		bootstrap.Run(cfg)
	},
}

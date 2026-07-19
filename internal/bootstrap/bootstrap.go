package bootstrap

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"be-ayaka/config"
	"be-ayaka/internal/adapter/database"
	"be-ayaka/internal/middleware"
	ayaka "be-ayaka/pkg/logger"
	"be-ayaka/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

func Run(cfg *config.Config) {
	// init database connection
	db := database.NewPostgresConnection(cfg)

	// init validator
	validator := validator.NewGoValidator(db)
	_ = validator
	ayaka.Log("SYSTEM", "INFO", "Validator System loaded!", "unknown-request-id")

	// init fiber
	app := fiber.New(fiber.Config{
		AppName:               cfg.App.Name,
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		DisableStartupMessage: true,
		ErrorHandler:          middleware.GlobalErrorHandler,
	})
	SetupRoutes(app, cfg, db)

	// run server in a goroutine
	go func() {
		port := fmt.Sprintf(":%d", cfg.Server.Port)
		ayaka.Log("SYSTEM", "INFO", fmt.Sprintf("Running on port %d", cfg.Server.Port), "unknown-request-id")

		if err := app.Listen(port); err != nil {
			ayaka.Log("SYSTEM", "ERROR", fmt.Sprintf("Failed to start server: %v", err), "unknown-request-id")
		}
	}()
	logo(cfg.Server.Port)

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ayaka.Log("SYSTEM", "WARN", "Start Graceful Shutdown process", "unknown-request-id")

	if err := app.ShutdownWithTimeout(5 * time.Second); err != nil {
		ayaka.Log("SYSTEM", "ERROR", fmt.Sprintf("Fiber Shutdown Failed [!]: %v", err), "unknown-request-id")
	} else {
		ayaka.Log("SYSTEM", "INFO", "Fiber server stopped cleanly", "unknown-request-id")
	}

	if db != nil {
		ayaka.Log("SYSTEM", "INFO", "Closing database connection...", "unknown-request-id")
		
		sqlDB, err := db.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				ayaka.Log("SYSTEM", "ERROR", fmt.Sprintf("Failed to close database: %v", err), "unknown-request-id")
			} else {
				ayaka.Log("SYSTEM", "INFO", "Database connection closed successfully", "unknown-request-id")
			}
		}
	}

	ayaka.Log("SYSTEM", "INFO", "🌸 Ayaka Server shutdown complete", "unknown-request-id")
}

func logo(port int) {
	fmt.Printf(`
     ░█▀▀▀ ░█──░█  ░█▀▀█ ─░█─ ░█▄─░█ ░█▀▀█
     ▒█▀▀─ ▒█──▒█  ▒█▄▄▀ ─▒█─ ░█▀█░█ ▒█▄▄█
     ▒█─── ─▀▄▄▄▀─ ▒█─░█ ─▒█─ ░█──▀█ ▒█─▒█
    Running on port %d
`+"\n", port)
}

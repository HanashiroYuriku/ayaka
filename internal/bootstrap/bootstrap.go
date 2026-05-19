package bootstrap

import (
	"context"
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

	_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Shutdown(); err != nil {
		ayaka.Log("SYSTEM", "ERROR", fmt.Sprintf("Graceful Shutdown Failed [!]: %v", err), "unknown-request-id")
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

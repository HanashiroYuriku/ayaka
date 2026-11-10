package bootstrap

import (
	"log/slog"

	"be-ayaka/config"
	"be-ayaka/internal/middleware"

	"github.com/gofiber/fiber/v2"
)

// NewFiber creates the Fiber app with the global middleware chain.
// Order matters: request id -> logging/error rendering -> panic recovery -> security.
func NewFiber(cfg *config.Config, log *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:                 cfg.App.Name,
		ReadTimeout:             cfg.Server.ReadTimeout,
		WriteTimeout:            cfg.Server.WriteTimeout,
		IdleTimeout:             cfg.Server.IdleTimeout,
		BodyLimit:               cfg.Server.BodyLimitMB * 1024 * 1024,
		DisableStartupMessage:   true,
		ErrorHandler:            middleware.ErrorHandler,
		EnableTrustedProxyCheck: len(cfg.Server.TrustedProxies) > 0,
		TrustedProxies:          cfg.Server.TrustedProxies,
		ProxyHeader:             proxyHeader(cfg),
	})

	app.Use(
		middleware.RequestIDMiddleware(),
		middleware.RequestContext(log),
		middleware.Recover(),
		middleware.SecurityHeaders(),
		middleware.CORS(cfg.Server.CORSAllowedOrigins),
	)
	return app
}

func proxyHeader(cfg *config.Config) string {
	if len(cfg.Server.TrustedProxies) > 0 {
		return fiber.HeaderXForwardedFor
	}
	return ""
}

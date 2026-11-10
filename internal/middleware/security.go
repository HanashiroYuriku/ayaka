package middleware

import (
	"strings"
	"time"

	"be-ayaka/internal/core/apperror"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
)

// CORS allows the configured origins to call the API from a browser.
func CORS(allowedOrigins []string) fiber.Handler {
	return cors.New(cors.Config{
		AllowOrigins:     strings.Join(allowedOrigins, ","),
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID",
		ExposeHeaders:    "X-Request-ID",
		AllowCredentials: !containsWildcard(allowedOrigins),
		MaxAge:           int((12 * time.Hour).Seconds()),
	})
}

// SecurityHeaders sets common security headers (X-Frame-Options, CSP, ...).
// Swagger UI needs inline scripts, so it is served without a strict CSP.
func SecurityHeaders() fiber.Handler {
	return helmet.New(helmet.Config{
		Next: func(c *fiber.Ctx) bool { return strings.HasPrefix(c.Path(), "/swagger") },
	})
}

// RateLimit limits requests per client IP. Uses in-memory storage; for
// multiple instances plug a shared storage (e.g. Redis) into limiter.Config.
func RateLimit(max int, window time.Duration) fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        max,
		Expiration: window,
		LimitReached: func(*fiber.Ctx) error {
			return apperror.TooManyRequests("too many requests, please try again later")
		},
	})
}

func containsWildcard(origins []string) bool {
	for _, o := range origins {
		if o == "*" {
			return true
		}
	}
	return false
}

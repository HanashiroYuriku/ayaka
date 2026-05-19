package testingutils

import (
	"be-ayaka/config"

	"github.com/gofiber/fiber/v2"
)

func GetDummyConfig() *config.Config {
	return &config.Config{
		Frontend: config.FrontendConfig{
			URL: "http://localhost:3000",
		},
	}
}

func StringPtr(s string) *string {
	return &s
}

func MockAuthMiddleware(defaultUserID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Get("X-Test-User-ID", defaultUserID)

		c.Locals("userID", userID)
		c.Locals("request_id", "REQ-TEST-123")
		return c.Next()
	}
}

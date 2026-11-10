// Package testingutils contains helpers shared by tests.
package testingutils

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/middleware"
	"be-ayaka/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/require"
)

// NewTestApp returns a Fiber app with the same error handling and request
// context middleware as production, but with a silent logger.
func NewTestApp() *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: middleware.ErrorHandler})
	app.Use(
		middleware.RequestIDMiddleware(),
		middleware.RequestContext(slog.New(slog.DiscardHandler)),
		middleware.Recover(),
	)
	return app
}

// FakeAuth authenticates every request as userID with role, unless the
// X-Test-User-ID header overrides the user.
func FakeAuth(userID string, role entity.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		middleware.SetIdentity(c, c.Get("X-Test-User-ID", userID), role)
		return c.Next()
	}
}

// DecodeResponse decodes the standard JSON envelope from an HTTP response.
func DecodeResponse(t *testing.T, resp *http.Response) response.Response {
	t.Helper()
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var out response.Response
	require.NoError(t, json.Unmarshal(body, &out), "body: %s", body)
	return out
}

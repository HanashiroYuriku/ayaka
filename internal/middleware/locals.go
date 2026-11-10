// Package middleware contains the HTTP middlewares and the helpers to read the
// values they store on the request.
package middleware

import (
	"be-ayaka/internal/core/entity"

	"github.com/gofiber/fiber/v2"
)

// Typed keys avoid collisions with other packages using c.Locals.
type localsKey string

const (
	localsRequestID localsKey = "requestid"
	localsUserID    localsKey = "userID"
	localsRole      localsKey = "role"
)

// RequestID returns the ID assigned to the current request.
func RequestID(c *fiber.Ctx) string {
	if id, ok := c.Locals(localsRequestID).(string); ok {
		return id
	}
	return ""
}

// UserID returns the authenticated user's ID. ok is false on public routes.
func UserID(c *fiber.Ctx) (string, bool) {
	id, ok := c.Locals(localsUserID).(string)
	return id, ok && id != ""
}

// Role returns the authenticated user's role. ok is false on public routes.
func Role(c *fiber.Ctx) (entity.Role, bool) {
	r, ok := c.Locals(localsRole).(entity.Role)
	return r, ok && r != ""
}

// SetIdentity stores the authenticated identity. Exposed for tests.
func SetIdentity(c *fiber.Ctx, userID string, role entity.Role) {
	c.Locals(localsUserID, userID)
	c.Locals(localsRole, role)
}

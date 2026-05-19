package middleware

import (
	"be-ayaka/pkg/requestid"
	"be-ayaka/pkg/response"

	"github.com/gofiber/fiber/v2"
)

func OnlyRole(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestId := requestid.GetRequestID(c)

		userRole, ok := c.Locals("role").(string)
		if !ok || userRole == "" {
			return c.Status(fiber.StatusForbidden).JSON(response.NewErrorResponse(
				response.Forbidden,
				"Access Denied: Role not found",
				requestId,
			))
		}

		isAllowed := false
		for _, role := range allowedRoles {
			if role == userRole {
				isAllowed = true
				break
			}
		}

		if !isAllowed {
			return c.Status(fiber.StatusForbidden).JSON(response.NewErrorResponse(
				response.Forbidden,
				"Access Denied",
				requestId,
			))
		}

		return c.Next()
	}
}

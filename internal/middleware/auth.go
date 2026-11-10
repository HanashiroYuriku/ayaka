package middleware

import (
	"log/slog"
	"slices"
	"strings"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/port"
	"be-ayaka/pkg/logger"

	"github.com/gofiber/fiber/v2"
)

var (
	errMissingToken = apperror.Unauthorized("missing or malformed bearer token")
	errForbidden    = apperror.Forbidden("you do not have permission to access this resource")
)

// RequireAuth verifies the Bearer access token and stores the identity on the request.
func RequireAuth(tokens port.TokenProvider) fiber.Handler {
	return func(c *fiber.Ctx) error {
		scheme, token, ok := strings.Cut(c.Get(fiber.HeaderAuthorization), " ")
		if !ok || !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(token) == "" {
			return errMissingToken
		}

		claims, err := tokens.Parse(strings.TrimSpace(token), model.TokenTypeAccess)
		if err != nil {
			return err
		}

		SetIdentity(c, claims.UserID, claims.Role)

		ctx := c.UserContext()
		c.SetUserContext(logger.WithContext(ctx, logger.FromContext(ctx).With(slog.String("user_id", claims.UserID))))

		return c.Next()
	}
}

// RequireRole allows the request only if the authenticated user has one of roles.
// It must be used after RequireAuth.
func RequireRole(roles ...entity.Role) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := Role(c)
		if !ok {
			return errMissingToken
		}
		if !slices.Contains(roles, role) {
			return errForbidden
		}
		return c.Next()
	}
}

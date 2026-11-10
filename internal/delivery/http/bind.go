// Package http contains the Fiber HTTP handlers.
//
// Handlers are thin: bind + validate input, call a service, map the result to
// a DTO. They never log errors themselves; they return them and the global
// ErrorHandler logs and renders them consistently.
package http

import (
	"be-ayaka/internal/core/apperror"
	"be-ayaka/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

type normalizer interface{ Normalize() }

// bindBody parses the request body into dst, normalizes and validates it.
func bindBody(c *fiber.Ctx, v validator.Validator, dst any) error {
	if err := c.BodyParser(dst); err != nil {
		return apperror.BadRequest("invalid request body").Wrap(err)
	}
	return normalizeAndValidate(c, v, dst)
}

// bindQuery parses the query string into dst, normalizes and validates it.
func bindQuery(c *fiber.Ctx, v validator.Validator, dst any) error {
	if err := c.QueryParser(dst); err != nil {
		return apperror.BadRequest("invalid query parameters").Wrap(err)
	}
	return normalizeAndValidate(c, v, dst)
}

func normalizeAndValidate(c *fiber.Ctx, v validator.Validator, dst any) error {
	if n, ok := dst.(normalizer); ok {
		n.Normalize()
	}
	return v.Validate(c.UserContext(), dst)
}

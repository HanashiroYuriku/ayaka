package middleware

import (
	"errors"
	"log/slog"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/pkg/logger"
	"be-ayaka/pkg/response"

	"github.com/gofiber/fiber/v2"
)

// kindToHTTP is the single place where application errors are mapped to HTTP.
var kindToHTTP = map[apperror.Kind]int{
	apperror.KindInternal:        fiber.StatusInternalServerError,
	apperror.KindBadRequest:      fiber.StatusBadRequest,
	apperror.KindValidation:      fiber.StatusUnprocessableEntity,
	apperror.KindUnauthorized:    fiber.StatusUnauthorized,
	apperror.KindForbidden:       fiber.StatusForbidden,
	apperror.KindNotFound:        fiber.StatusNotFound,
	apperror.KindConflict:        fiber.StatusConflict,
	apperror.KindTooManyRequests: fiber.StatusTooManyRequests,
}

// ErrorHandler is the Fiber ErrorHandler. Handlers simply `return err`.
func ErrorHandler(c *fiber.Ctx, err error) error {
	var (
		code    int
		message string
		fields  map[string]string
	)

	var fe *fiber.Error
	if errors.As(err, &fe) {
		// Errors raised by Fiber itself: 404 route, 405, 413 body too large, ...
		code, message = fe.Code, fe.Message
	} else {
		ae := apperror.As(err)
		code = kindToHTTP[ae.Kind]
		if code == 0 {
			code = fiber.StatusInternalServerError
		}
		message, fields = ae.PublicMessage(), ae.Fields
	}

	log := logger.FromContext(c.UserContext())
	if code >= fiber.StatusInternalServerError {
		// The cause is logged but never returned to the client.
		log.Error("request failed", slog.Any("error", err))
	} else {
		log.Debug("request rejected", slog.Int("status", code), slog.Any("error", err))
	}

	return c.Status(code).JSON(response.Error(response.StatusFromHTTP(code), message, fields, RequestID(c)))
}

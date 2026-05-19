package middleware

import (
	"be-ayaka/internal/core/customerrors"
	"be-ayaka/pkg/requestid"
	"be-ayaka/pkg/response"
	"errors"

	"github.com/gofiber/fiber/v2"
)

func GlobalErrorHandler(c *fiber.Ctx, err error) error {
	requestId := requestid.GetRequestID(c)

	// unprocess 422
	var valErr *customerrors.ValidationError
	if errors.As(err, &valErr) {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(
			response.NewErrorFieldResponse(
				response.UnprocessableEntity,
				valErr.Error(),
				valErr.Detail,
				requestId,
			),
		)
	}

	// data conflict 409
	var valErrConflict *customerrors.ConflictError
	if errors.As(err, &valErrConflict) {
		return c.Status(fiber.StatusConflict).JSON(
			response.NewErrorFieldResponse(
				response.DataConflict,
				valErrConflict.Error(),
				valErrConflict.Detail,
				requestId,
			),
		)
	}

	// not found 404
	var valErrNotFound *customerrors.NotFoundError
	if errors.As(err, &valErrNotFound) {
		return c.Status(fiber.StatusNotFound).JSON(
			response.NewErrorFieldResponse(
				response.DataNotFound,
				valErrNotFound.Error(),
				valErrNotFound.Detail,
				requestId,
			),
		)
	}

	code := fiber.StatusInternalServerError
	statusString := response.InternalServerError
	message := "Internal Server Error"

	var fiberErr *fiber.Error
	if errors.As(err, &fiberErr) {
		code = fiberErr.Code
		message = fiberErr.Message
	}

	switch {
	case errors.Is(err, customerrors.ErrDataNotFound):
		code = fiber.StatusNotFound
		statusString = response.DataNotFound
		message = err.Error()

	case errors.Is(err, customerrors.ErrInvalidPassword),
		errors.Is(err, customerrors.ErrTokenExpired):
		code = fiber.StatusUnauthorized
		statusString = response.Unauthorized
		message = err.Error()

	case errors.Is(err, customerrors.ErrBadRequest):
		code = fiber.StatusBadRequest
		statusString = response.BadRequest
		message = err.Error()

	case errors.Is(err, customerrors.ErrCooldownActive):
		code = fiber.StatusTooManyRequests
		statusString = response.TooManyRequests
		message = err.Error()

	case errors.Is(err, customerrors.ErrTokenExpired):
		code = fiber.StatusUnauthorized
		statusString = response.Unauthorized
		message = err.Error()

	case errors.Is(err, customerrors.ErrInvalidCredentials):
		code = fiber.StatusUnauthorized
		statusString = response.Unauthorized
		message = err.Error()

	case errors.Is(err, customerrors.ErrAccountInactive):
		code = fiber.StatusUnauthorized
		statusString = response.Unauthorized
		message = err.Error()

	case errors.Is(err, customerrors.ErrAccountAlreadyVerified):
		code = fiber.StatusConflict
		statusString = response.DataConflict
		message = err.Error()

	case errors.Is(err, customerrors.ErrUnauthorized):
		code = fiber.StatusUnauthorized
		statusString = response.Unauthorized
		message = err.Error()

	case errors.Is(err, customerrors.ErrFailHash):
		code = fiber.StatusInternalServerError
		statusString = response.InternalServerError
		message = err.Error()
	}

	return c.Status(code).JSON(response.NewErrorResponse(
		statusString,
		message,
		requestId,
	))
}

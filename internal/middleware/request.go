package middleware

import (
	"fmt"
	"log/slog"
	"runtime/debug"
	"time"

	"be-ayaka/pkg/generateid"
	"be-ayaka/pkg/logger"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
)

// RequestIDMiddleware assigns an ID to every request (or reuses the incoming
// X-Request-ID, so IDs propagate across services) and echoes it in the response.
func RequestIDMiddleware() fiber.Handler {
	return requestid.New(requestid.Config{
		Header:     fiber.HeaderXRequestID,
		ContextKey: localsRequestID,
		Generator:  func() string { return generateid.New("REQ") },
	})
}

// RequestContext attaches a request-scoped logger (with request_id) to the
// request's user context, converts returned errors into responses with the
// app's ErrorHandler, and writes one access-log line per request.
//
// Handlers and services should log with logger.FromContext(ctx).
func RequestContext(base *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		reqLog := base.With(slog.String("request_id", RequestID(c)))
		c.SetUserContext(logger.WithContext(c.UserContext(), reqLog))

		if err := c.Next(); err != nil {
			if herr := c.App().ErrorHandler(c, err); herr != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		status := c.Response().StatusCode()
		level := slog.LevelInfo
		switch {
		case status >= 500:
			level = slog.LevelError
		case status >= 400:
			level = slog.LevelWarn
		}

		// Re-read the logger: auth may have enriched it with user_id.
		logger.FromContext(c.UserContext()).LogAttrs(c.UserContext(), level, "http request",
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.String("route", c.Route().Path),
			slog.Int("status", status),
			slog.Duration("latency", time.Since(start)),
			slog.String("ip", c.IP()),
			slog.String("user_agent", c.Get(fiber.HeaderUserAgent)),
		)
		return nil
	}
}

// Recover turns panics into 500 responses and logs the stack trace.
func Recover() fiber.Handler {
	return recover.New(recover.Config{
		EnableStackTrace: true,
		StackTraceHandler: func(c *fiber.Ctx, e any) {
			logger.FromContext(c.UserContext()).Error("panic recovered",
				slog.String("panic", fmt.Sprint(e)),
				slog.String("stack", string(debug.Stack())),
			)
		},
	})
}

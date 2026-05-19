package middleware

import (
	"strings"

	"be-ayaka/config"
	customjwt "be-ayaka/pkg/jwt"
	"be-ayaka/pkg/logger"
	"be-ayaka/pkg/requestid"
	"be-ayaka/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

func RequireAuth(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		requestId := requestid.GetRequestID(c)

		authHeader := c.Get("Authorization")

		if authHeader == "" {
			logger.Log("AUTH", "WARN", "Authorization header is missing", requestId)
			return c.Status(fiber.StatusUnauthorized).JSON(response.NewErrorResponse(
				response.Unauthorized,
				"Authorization header is missing",
				requestId,
			))
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Log("AUTH", "WARN", "Invalid Token Format", requestId)
			return c.Status(fiber.StatusUnauthorized).JSON(response.NewErrorResponse(
				response.Unauthorized,
				"Invalid Token Format",
				requestId))
		}

		tokenString := parts[1]

		claims := &customjwt.CustomClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				logger.Log("AUTH", "WARN", "Unexpected signing method", requestId)
				return nil, fiber.ErrUnauthorized
			}
			return []byte(cfg.JWT.Secret), nil
		})

		if err != nil || !token.Valid {
			logger.Log("AUTH", "WARN", "Token Invalid or Expired", requestId)
			return c.Status(fiber.StatusUnauthorized).JSON(response.NewErrorResponse(
				response.Unauthorized,
				"Token Invalid or Expired",
				requestId,
			))
		}

		c.Locals("userID", claims.UserID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

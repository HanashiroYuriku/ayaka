package http

import (
	"be-ayaka/internal/core/service"
	"be-ayaka/internal/delivery/http/dto"
	"be-ayaka/internal/middleware"
	"be-ayaka/pkg/logger"
	"be-ayaka/pkg/response"
	"be-ayaka/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	auth      service.AuthService
	validator validator.Validator
}

func NewAuthHandler(auth service.AuthService, v validator.Validator) *AuthHandler {
	return &AuthHandler{auth: auth, validator: v}
}

// Register creates a new user account.
//
//	@Summary		Register
//	@Description	Create a new user account.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest								true	"Register payload"
//	@Success		201		{object}	response.Response{data=dto.UserResponse}
//	@Failure		400		{object}	response.Response	"Malformed body"
//	@Failure		409		{object}	response.Response	"Username or email already exists"
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Failure		429		{object}	response.Response	"Rate limited"
//	@Router			/api/v1/auth/register [post]
func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}

	user, err := h.auth.Register(c.UserContext(), req.ToInput())
	if err != nil {
		return err
	}

	logger.FromContext(c.UserContext()).Info("user registered", "user_id", user.ID)
	return c.Status(fiber.StatusCreated).JSON(
		response.Success("user registered successfully", dto.NewUserResponse(*user), middleware.RequestID(c)),
	)
}

// Login authenticates with email and password.
//
//	@Summary		Login
//	@Description	Exchange email and password for an access/refresh token pair.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest								true	"Login payload"
//	@Success		200		{object}	response.Response{data=dto.TokenResponse}
//	@Failure		401		{object}	response.Response	"Invalid credentials"
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Failure		429		{object}	response.Response	"Rate limited"
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}

	tokens, err := h.auth.Login(c.UserContext(), req.ToInput())
	if err != nil {
		return err
	}

	return c.JSON(response.Success("login successful", dto.NewTokenResponse(tokens), middleware.RequestID(c)))
}

// Refresh issues a new token pair from a refresh token.
//
//	@Summary		Refresh token
//	@Description	Exchange a valid refresh token for a new access/refresh token pair.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RefreshRequest								true	"Refresh payload"
//	@Success		200		{object}	response.Response{data=dto.TokenResponse}
//	@Failure		401		{object}	response.Response	"Invalid or expired refresh token"
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Router			/api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req dto.RefreshRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}

	tokens, err := h.auth.Refresh(c.UserContext(), req.RefreshToken)
	if err != nil {
		return err
	}

	return c.JSON(response.Success("token refreshed", dto.NewTokenResponse(tokens), middleware.RequestID(c)))
}

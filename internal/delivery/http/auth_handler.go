package http

import (
	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/service"
	"be-ayaka/internal/delivery/http/dto"
	"be-ayaka/internal/middleware"
	"be-ayaka/pkg/logger"
	"be-ayaka/pkg/response"
	"be-ayaka/pkg/validator"

	"github.com/gofiber/fiber/v2"
)

// Generic message for endpoints that must not reveal whether an email exists.
const msgEmailSentIfExists = "if the email is registered, a message has been sent"

type AuthHandler struct {
	auth      service.AuthService
	validator validator.Validator
}

func NewAuthHandler(auth service.AuthService, v validator.Validator) *AuthHandler {
	return &AuthHandler{auth: auth, validator: v}
}

// Register creates a new user account and sends a verification email.
//
//	@Summary		Register
//	@Description	Create a new user account. A verification link is emailed to the user.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RegisterRequest	true	"Register payload"
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
		response.Success("user registered successfully, please verify your email", dto.NewUserResponse(*user), middleware.RequestID(c)),
	)
}

// Login authenticates with email and password.
//
//	@Summary		Login
//	@Description	Exchange email and password for an access/refresh token pair (starts a new session).
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.LoginRequest	true	"Login payload"
//	@Success		200		{object}	response.Response{data=dto.TokenResponse}
//	@Failure		401		{object}	response.Response	"Invalid credentials"
//	@Failure		403		{object}	response.Response	"Email not verified"
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Failure		429		{object}	response.Response	"Rate limited or account temporarily locked"
//	@Router			/api/v1/auth/login [post]
func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req dto.LoginRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}

	tokens, err := h.auth.Login(c.UserContext(), req.ToInput(clientInfo(c)))
	if err != nil {
		return err
	}

	return c.JSON(response.Success("login successful", dto.NewTokenResponse(tokens), middleware.RequestID(c)))
}

// Refresh rotates the refresh token and issues a new token pair.
//
//	@Summary		Refresh token
//	@Description	Exchange a refresh token for a new access/refresh token pair. The old refresh token becomes invalid; reusing it revokes the whole session.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RefreshTokenRequest	true	"Refresh payload"
//	@Success		200		{object}	response.Response{data=dto.TokenResponse}
//	@Failure		401		{object}	response.Response	"Invalid, expired or reused refresh token"
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Router			/api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req dto.RefreshTokenRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}

	tokens, err := h.auth.Refresh(c.UserContext(), req.RefreshToken, clientInfo(c))
	if err != nil {
		return err
	}

	return c.JSON(response.Success("token refreshed", dto.NewTokenResponse(tokens), middleware.RequestID(c)))
}

// Logout ends the session of the given refresh token.
//
//	@Summary		Logout
//	@Description	Revoke the session the refresh token belongs to. Idempotent. The access token stays valid until it expires (keep it short-lived).
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.RefreshTokenRequest	true	"Refresh token of the session"
//	@Success		200		{object}	response.Response
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Router			/api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req dto.RefreshTokenRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}
	if err := h.auth.Logout(c.UserContext(), req.RefreshToken); err != nil {
		return err
	}
	return c.JSON(response.Success("logged out", nil, middleware.RequestID(c)))
}

// LogoutAll ends every session of the authenticated user.
//
//	@Summary		Logout from all devices
//	@Description	Revoke every refresh token of the authenticated user.
//	@Tags			Auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response	"Missing or invalid token"
//	@Router			/api/v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *fiber.Ctx) error {
	userID, ok := middleware.UserID(c)
	if !ok {
		return apperror.ErrUnauthorized
	}
	if err := h.auth.LogoutAll(c.UserContext(), userID); err != nil {
		return err
	}
	return c.JSON(response.Success("logged out from all devices", nil, middleware.RequestID(c)))
}

// ChangePassword changes the password of the authenticated user.
//
//	@Summary		Change password
//	@Description	Change the password of the authenticated user. All sessions are revoked; log in again afterwards.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		dto.ChangePasswordRequest	true	"Passwords"
//	@Success		200		{object}	response.Response
//	@Failure		401		{object}	response.Response	"Missing or invalid token"
//	@Failure		422		{object}	response.Response	"Wrong current password or weak new password"
//	@Router			/api/v1/auth/password [put]
func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	userID, ok := middleware.UserID(c)
	if !ok {
		return apperror.ErrUnauthorized
	}
	var req dto.ChangePasswordRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}

	err := h.auth.ChangePassword(c.UserContext(), model.ChangePasswordInput{
		UserID: userID, CurrentPassword: req.CurrentPassword, NewPassword: req.NewPassword,
	})
	if err != nil {
		return err
	}
	return c.JSON(response.Success("password changed, please log in again", nil, middleware.RequestID(c)))
}

// ForgotPassword emails a password reset link.
//
//	@Summary		Forgot password
//	@Description	Email a password reset link. Always returns 200 so it cannot be used to discover registered emails.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.EmailRequest	true	"Account email"
//	@Success		200		{object}	response.Response
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Failure		429		{object}	response.Response	"Rate limited"
//	@Router			/api/v1/auth/password/forgot [post]
func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req dto.EmailRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}
	if err := h.auth.RequestPasswordReset(c.UserContext(), req.Email); err != nil {
		return err
	}
	return c.JSON(response.Success(msgEmailSentIfExists, nil, middleware.RequestID(c)))
}

// ResetPassword sets a new password using the token from the email.
//
//	@Summary		Reset password
//	@Description	Set a new password with the token from the reset email. Unlocks the account and revokes all sessions.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.ResetPasswordRequest	true	"Token and new password"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response	"Invalid or expired token"
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Router			/api/v1/auth/password/reset [post]
func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req dto.ResetPasswordRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}
	err := h.auth.ResetPassword(c.UserContext(), model.ResetPasswordInput{Token: req.Token, NewPassword: req.NewPassword})
	if err != nil {
		return err
	}
	return c.JSON(response.Success("password has been reset, please log in", nil, middleware.RequestID(c)))
}

// VerifyEmail confirms the email address.
//
//	@Summary		Verify email
//	@Description	Confirm the email address with the token from the verification email.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.TokenRequest	true	"Verification token"
//	@Success		200		{object}	response.Response
//	@Failure		400		{object}	response.Response	"Invalid or expired token"
//	@Router			/api/v1/auth/email/verify [post]
func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var req dto.TokenRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}
	if err := h.auth.VerifyEmail(c.UserContext(), req.Token); err != nil {
		return err
	}
	return c.JSON(response.Success("email verified", nil, middleware.RequestID(c)))
}

// ResendVerification sends a new verification email.
//
//	@Summary		Resend verification email
//	@Description	Send a new verification link. Always returns 200 so it cannot be used to discover registered emails.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		dto.EmailRequest	true	"Account email"
//	@Success		200		{object}	response.Response
//	@Failure		422		{object}	response.Response	"Validation failed"
//	@Failure		429		{object}	response.Response	"Rate limited"
//	@Router			/api/v1/auth/email/resend [post]
func (h *AuthHandler) ResendVerification(c *fiber.Ctx) error {
	var req dto.EmailRequest
	if err := bindBody(c, h.validator, &req); err != nil {
		return err
	}
	if err := h.auth.ResendVerification(c.UserContext(), req.Email); err != nil {
		return err
	}
	return c.JSON(response.Success(msgEmailSentIfExists, nil, middleware.RequestID(c)))
}

func clientInfo(c *fiber.Ctx) model.ClientInfo {
	return model.ClientInfo{IP: c.IP(), UserAgent: c.Get(fiber.HeaderUserAgent)}
}

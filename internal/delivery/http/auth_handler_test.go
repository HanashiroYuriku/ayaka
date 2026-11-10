package http_test

import (
	"testing"
	"time"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/service"
	httpdelivery "be-ayaka/internal/delivery/http"
	"be-ayaka/internal/testingutils"
	"be-ayaka/internal/testingutils/mocks"
	"be-ayaka/pkg/response"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AuthHandlerSuite struct {
	suite.Suite
	app       *fiber.App
	svc       *mocks.AuthService
	validator *mocks.Validator
}

func TestAuthHandlerSuite(t *testing.T) {
	suite.Run(t, new(AuthHandlerSuite))
}

func (s *AuthHandlerSuite) SetupTest() {
	s.svc = new(mocks.AuthService)
	s.validator = new(mocks.Validator)
	h := httpdelivery.NewAuthHandler(s.svc, s.validator)

	s.app = testingutils.NewTestApp()
	s.app.Post("/auth/register", h.Register)
	s.app.Post("/auth/login", h.Login)
	s.app.Post("/auth/refresh", h.Refresh)
	s.app.Post("/auth/logout", h.Logout)
	s.app.Post("/auth/password/forgot", h.ForgotPassword)
	s.app.Post("/auth/password/reset", h.ResetPassword)
	s.app.Post("/auth/email/verify", h.VerifyEmail)
	s.app.Post("/auth/email/resend", h.ResendVerification)
	authed := s.app.Group("", testingutils.FakeAuth("USR-1", entity.RoleUser))
	authed.Post("/auth/logout-all", h.LogoutAll)
	authed.Put("/auth/password", h.ChangePassword)
}

func (s *AuthHandlerSuite) TearDownTest() {
	s.svc.AssertExpectations(s.T())
	s.validator.AssertExpectations(s.T())
}

// ============================================================ Register

func (s *AuthHandlerSuite) TestRegister_Success_NormalizesInput() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("Register", mock.Anything, model.RegisterInput{
		Username: "riku", Email: "riku@mail.com", DisplayName: "Riku", Password: "P4$$word",
	}).Return(&model.UserProfile{ID: "USR-1", Username: "riku", Email: "riku@mail.com"}, nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/register", map[string]string{
		"username": " riku ", "email": "RIKU@mail.com", "displayName": "Riku", "password": "P4$$word",
	}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusCreated, resp.StatusCode)
	body := testingutils.DecodeResponse(s.T(), resp)
	s.Equal(response.StatusSuccess.String(), body.Status)
	s.Equal("TEST-REQUEST-ID", body.RequestID)
}

func (s *AuthHandlerSuite) TestRegister_MalformedBody_Returns400() {
	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/register", []byte(`{"username":`)))
	s.Require().NoError(err)

	s.Equal(fiber.StatusBadRequest, resp.StatusCode)
	s.Equal(response.BadRequest.String(), testingutils.DecodeResponse(s.T(), resp).Status)
}

func (s *AuthHandlerSuite) TestRegister_ValidationError_Returns422WithFields() {
	s.validator.On("Validate", mock.Anything, mock.Anything).
		Return(apperror.Validation(map[string]string{"username": "Username must be at least 3 characters in length"})).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/register", map[string]string{"username": "ri"}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusUnprocessableEntity, resp.StatusCode)
	body := testingutils.DecodeResponse(s.T(), resp)
	s.Equal(response.UnprocessableEntity.String(), body.Status)
	s.Contains(body.ErrorField, "username")
}

func (s *AuthHandlerSuite) TestRegister_Conflict_Returns409() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("Register", mock.Anything, mock.Anything).Return(nil, apperror.Conflict("username or email already exists")).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/register", map[string]string{"username": "riku"}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusConflict, resp.StatusCode)
}

func (s *AuthHandlerSuite) TestRegister_InternalError_HidesDetails() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("Register", mock.Anything, mock.Anything).
		Return(nil, apperror.Internal(assertError("pq: password authentication failed"))).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/register", map[string]string{"username": "riku"}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
	body := testingutils.DecodeResponse(s.T(), resp)
	s.Equal("internal server error", body.Message)
	s.NotContains(body.Message, "pq:")
}

// ============================================================ Login

func (s *AuthHandlerSuite) TestLogin_Success() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("Login", mock.Anything, mock.MatchedBy(func(in model.LoginInput) bool {
		return in.Email == "riku@mail.com" && in.Password == "P4$$word" && in.Client.IP != ""
	})).
		Return(&model.AuthTokens{
			AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer",
			AccessExpiresAt: time.Now().Add(15 * time.Minute),
		}, nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/login", map[string]string{
		"email": "riku@mail.com", "password": "P4$$word",
	}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusOK, resp.StatusCode)
	data := testingutils.DecodeResponse(s.T(), resp).Data.(map[string]any)
	s.Equal("access", data["accessToken"])
	s.Equal("Bearer", data["tokenType"])
}

func (s *AuthHandlerSuite) TestLogin_InvalidCredentials_Returns401() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("Login", mock.Anything, mock.Anything).Return(nil, service.ErrInvalidCredentials).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/login", map[string]string{
		"email": "riku@mail.com", "password": "wrong",
	}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
	s.Equal("invalid email or password", testingutils.DecodeResponse(s.T(), resp).Message)
}

// ============================================================ Refresh

func (s *AuthHandlerSuite) TestRefresh_InvalidToken_Returns401() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("Refresh", mock.Anything, "bad-token", mock.Anything).Return(nil, service.ErrInvalidToken).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/refresh", map[string]string{
		"refreshToken": "bad-token",
	}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

type assertError string

func (e assertError) Error() string { return string(e) }

// ============================================================ Logout

func (s *AuthHandlerSuite) TestLogout_Success() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("Logout", mock.Anything, "rt").Return(nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/logout", map[string]string{"refreshToken": "rt"}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *AuthHandlerSuite) TestLogoutAll_UsesAuthenticatedUser() {
	s.svc.On("LogoutAll", mock.Anything, "USR-1").Return(nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/logout-all", nil))
	s.Require().NoError(err)

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

// ============================================================ Passwords

func (s *AuthHandlerSuite) TestChangePassword_WrongCurrent_Returns422() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("ChangePassword", mock.Anything, model.ChangePasswordInput{
		UserID: "USR-1", CurrentPassword: "old", NewPassword: "New$pass1",
	}).Return(service.ErrWrongCurrentPassword).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPut, "/auth/password", map[string]string{
		"currentPassword": "old", "newPassword": "New$pass1",
	}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusUnprocessableEntity, resp.StatusCode)
	s.Contains(testingutils.DecodeResponse(s.T(), resp).ErrorField, "currentPassword")
}

func (s *AuthHandlerSuite) TestForgotPassword_AlwaysGenericResponse() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("RequestPasswordReset", mock.Anything, "ghost@mail.com").Return(nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/password/forgot", map[string]string{
		"email": " Ghost@Mail.com ",
	}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.Equal("if the email is registered, a message has been sent", testingutils.DecodeResponse(s.T(), resp).Message)
}

func (s *AuthHandlerSuite) TestResetPassword_InvalidToken_Returns400() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("ResetPassword", mock.Anything, model.ResetPasswordInput{Token: "bad", NewPassword: "New$pass1"}).
		Return(service.ErrInvalidResetToken).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/password/reset", map[string]string{
		"token": "bad", "newPassword": "New$pass1",
	}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusBadRequest, resp.StatusCode)
}

// ============================================================ Email verification

func (s *AuthHandlerSuite) TestVerifyEmail_Success() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("VerifyEmail", mock.Anything, "tok").Return(nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/email/verify", map[string]string{"token": "tok"}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

func (s *AuthHandlerSuite) TestResendVerification_Success() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.svc.On("ResendVerification", mock.Anything, "riku@mail.com").Return(nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/email/resend", map[string]string{"email": "riku@mail.com"}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusOK, resp.StatusCode)
}

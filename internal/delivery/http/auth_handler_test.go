package http_test

import (
	"testing"
	"time"

	"be-ayaka/internal/core/apperror"
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
	s.svc.On("Login", mock.Anything, model.LoginInput{Email: "riku@mail.com", Password: "P4$$word"}).
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
	s.svc.On("Refresh", mock.Anything, "bad-token").Return(nil, service.ErrInvalidToken).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodPost, "/auth/refresh", map[string]string{
		"refreshToken": "bad-token",
	}))
	s.Require().NoError(err)

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

type assertError string

func (e assertError) Error() string { return string(e) }

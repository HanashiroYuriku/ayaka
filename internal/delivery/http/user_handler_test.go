package http_test

import (
	"be-ayaka/internal/core/customerrors"
	"be-ayaka/internal/core/entity"
	httpDelivery "be-ayaka/internal/delivery/http"
	"be-ayaka/internal/middleware"
	"be-ayaka/internal/testingutils"
	mocksPkg "be-ayaka/internal/testingutils/mocks/pkg"
	mocksService "be-ayaka/internal/testingutils/mocks/service"
	"errors"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type UserHandlerSuite struct {
	suite.Suite
	app           *fiber.App
	mockService   *mocksService.MockUserService
	mockValidator *mocksPkg.MockValidator
	handler       *httpDelivery.UserHandler
}

func (s *UserHandlerSuite) SetupTest() {
	s.app = fiber.New(fiber.Config{
		ErrorHandler: middleware.GlobalErrorHandler,
	})
	s.mockService = new(mocksService.MockUserService)
	s.mockValidator = new(mocksPkg.MockValidator)
	s.handler = httpDelivery.NewUserHandler(s.mockService, s.mockValidator)

	api := s.app.Group("/api/v1")
	api.Use(testingutils.MockAuthMiddleware("USER-123"))

	api.Post("/auth/register", s.handler.RegisterUser)
	api.Get("/users/profile", s.handler.GetProfile)
	// add more routes if needed
}

func TestUserHandlerSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerSuite))
}

// ///////////////// TEST REGISTER USER ///////////////////
// 1. success scenario
func (s *UserHandlerSuite) TestRegisterUser_Success() {
	requestBody := entity.UserRequest{
		Username:    "riku",
		Email:       "riku@mail.com",
		DisplayName: "Riku",
		Password:    "P4$$w0rd",
	}

	s.mockValidator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.mockService.On("Create", mock.Anything, mock.Anything).Return(nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest("POST", "/api/v1/auth/register", requestBody))

	s.Require().NoError(err)
	s.Equal(fiber.StatusCreated, resp.StatusCode)

	s.mockValidator.AssertExpectations(s.T())
	s.mockService.AssertExpectations(s.T())
}

// 2. failed scenario: bad request
func (s *UserHandlerSuite) TestRegisterUser_Failed_BadRequest() {
	body := []byte(`{"username": "riku", "email": "riku@mail.com"`)

	resp, err := s.app.Test(testingutils.MakeJSONRequest("POST", "/api/v1/auth/register", body))

	s.Require().NoError(err)
	s.Equal(fiber.StatusBadRequest, resp.StatusCode)

	s.mockValidator.AssertNotCalled(s.T(), "Validate", mock.Anything, mock.Anything)
	s.mockService.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

// 3. failed scenario: error validation
func (s *UserHandlerSuite) TestRegisterUser_Failed_ValidationError() {
	requestBody := entity.UserRequest{Username: "ri"}

	expectedErr := customerrors.NewValidationError(
		`"username": "username must be at least 3 characters"`,
	)

	s.mockValidator.On("Validate", mock.Anything, mock.Anything).
		Return(expectedErr).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest("POST", "/api/v1/auth/register", requestBody))

	s.Require().NoError(err)
	s.Equal(fiber.StatusUnprocessableEntity, resp.StatusCode)

	s.mockService.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
	s.mockValidator.AssertExpectations(s.T())
}

// 4. failed scenario: internal server error
func (s *UserHandlerSuite) TestRegisterUser_Failed_InternalServerError() {
	requestBody := entity.UserRequest{
		Username: "riku",
		Email:    "riku@mail.com",
	}

	s.mockValidator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	s.mockService.On("Create", mock.Anything, mock.Anything).
		Return(errors.New("failed to create user")).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest("POST", "/api/v1/auth/register", requestBody))

	s.Require().NoError(err)
	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)

	s.mockValidator.AssertExpectations(s.T())
	s.mockService.AssertExpectations(s.T())
}

/////////////////// TEST REGISTER USER ///////////////////

/////////////////// TEST GET PROFILE ///////////////////

// 1. Success scenario
func (s *UserHandlerSuite) TestGetProfile_Success() {
	userID := "USER-123"
	dummyUser := &entity.User{
		Username: "riku",
		Email:    "riku@mail.com",
	}
	dummyUser.ID = userID

	s.mockService.On("GetProfile", mock.Anything, userID).Return(dummyUser, nil).Once()

	req := testingutils.MakeJSONRequest("GET", "/api/v1/users/profile", nil)
	req.Header.Set("X-Test-User-ID", userID)

	resp, err := s.app.Test(req)

	s.Require().NoError(err)
	s.Equal(fiber.StatusOK, resp.StatusCode)
	s.mockService.AssertExpectations(s.T())
}

// 2. failed scenario: user not found
func (s *UserHandlerSuite) TestGetProfile_Failed_NotFound() {
	userID := "USER-NOTFOUND"

	s.mockService.On("GetProfile", mock.Anything, userID).Return(nil, customerrors.ErrDataNotFound).Once()

	req := testingutils.MakeJSONRequest("GET", "/api/v1/users/profile", nil)
	req.Header.Set("X-Test-User-ID", userID)

	resp, err := s.app.Test(req)

	s.Require().NoError(err)
	s.Equal(fiber.StatusNotFound, resp.StatusCode)
	s.mockService.AssertExpectations(s.T())
}

// 3. failed scneario: internal server error
func (s *UserHandlerSuite) TestGetProfile_Failed_InternalServerError() {
	userID := "USER-123"
	systemError := errors.New("database connection lost")

	s.mockService.On("GetProfile", mock.Anything, userID).Return(nil, systemError).Once()

	req := testingutils.MakeJSONRequest("GET", "/api/v1/users/profile", nil)
	req.Header.Set("X-Test-User-ID", userID)

	resp, err := s.app.Test(req)

	s.Require().NoError(err)
	s.Equal(fiber.StatusInternalServerError, resp.StatusCode)
	s.mockService.AssertExpectations(s.T())
}

/////////////////// TEST GET PROFILE USER ///////////////////

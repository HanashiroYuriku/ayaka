package http_test

import (
	"testing"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	httpdelivery "be-ayaka/internal/delivery/http"
	"be-ayaka/internal/testingutils"
	"be-ayaka/internal/testingutils/mocks"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type UserHandlerSuite struct {
	suite.Suite
	app       *fiber.App
	svc       *mocks.UserService
	validator *mocks.Validator
}

func TestUserHandlerSuite(t *testing.T) {
	suite.Run(t, new(UserHandlerSuite))
}

func (s *UserHandlerSuite) SetupTest() {
	s.svc = new(mocks.UserService)
	s.validator = new(mocks.Validator)
	h := httpdelivery.NewUserHandler(s.svc, s.validator)

	s.app = testingutils.NewTestApp()
	s.app.Get("/public/me", h.GetMe) // no auth middleware on purpose
	authed := s.app.Group("/users", testingutils.FakeAuth("USR-1", entity.RoleAdmin))
	authed.Get("/me", h.GetMe)
	authed.Get("/", h.List)
}

func (s *UserHandlerSuite) TearDownTest() {
	s.svc.AssertExpectations(s.T())
	s.validator.AssertExpectations(s.T())
}

// ============================================================ GetMe

func (s *UserHandlerSuite) TestGetMe_Success() {
	s.svc.On("GetProfile", mock.Anything, "USR-1").
		Return(&model.UserProfile{ID: "USR-1", Username: "riku", Role: entity.RoleAdmin}, nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/users/me", nil))
	s.Require().NoError(err)

	s.Equal(fiber.StatusOK, resp.StatusCode)
	data := testingutils.DecodeResponse(s.T(), resp).Data.(map[string]any)
	s.Equal("riku", data["username"])
	s.NotContains(data, "password")
}

func (s *UserHandlerSuite) TestGetMe_NotFound() {
	req := testingutils.MakeJSONRequest(fiber.MethodGet, "/users/me", nil)
	req.Header.Set("X-Test-User-ID", "USR-GONE")
	s.svc.On("GetProfile", mock.Anything, "USR-GONE").Return(nil, apperror.NotFound("user not found")).Once()

	resp, err := s.app.Test(req)
	s.Require().NoError(err)

	s.Equal(fiber.StatusNotFound, resp.StatusCode)
}

func (s *UserHandlerSuite) TestGetMe_WithoutIdentity_Returns401InsteadOfPanic() {
	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/public/me", nil))
	s.Require().NoError(err)

	s.Equal(fiber.StatusUnauthorized, resp.StatusCode)
}

// ============================================================ List

func (s *UserHandlerSuite) TestList_Success_WithMeta() {
	s.validator.On("Validate", mock.Anything, mock.Anything).Return(nil).Once()
	page := model.NewPage([]model.UserProfile{{ID: "USR-1"}, {ID: "USR-2"}}, 3, model.PageRequest{Page: 1, PageSize: 2})
	s.svc.On("List", mock.Anything, model.PageRequest{Page: 1, PageSize: 2}).Return(&page, nil).Once()

	resp, err := s.app.Test(testingutils.MakeJSONRequest(fiber.MethodGet, "/users?page=1&pageSize=2", nil))
	s.Require().NoError(err)

	s.Equal(fiber.StatusOK, resp.StatusCode)
	body := testingutils.DecodeResponse(s.T(), resp)
	s.Len(body.Data, 2)
	meta := body.Meta.(map[string]any)
	s.EqualValues(3, meta["totalRecords"])
	s.EqualValues(2, meta["totalPages"])
	s.Equal(true, meta["hasNext"])
}

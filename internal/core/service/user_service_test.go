package service_test

import (
	"context"
	"errors"
	"testing"

	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"
	"be-ayaka/internal/core/service"
	"be-ayaka/internal/testingutils/mocks"

	"github.com/stretchr/testify/suite"
)

type UserServiceSuite struct {
	suite.Suite
	ctx   context.Context
	users *mocks.UserRepo
	svc   service.UserService
}

func TestUserServiceSuite(t *testing.T) {
	suite.Run(t, new(UserServiceSuite))
}

func (s *UserServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.users = new(mocks.UserRepo)
	s.svc = service.NewUserService(s.users)
}

func (s *UserServiceSuite) TearDownTest() {
	s.users.AssertExpectations(s.T())
}

func (s *UserServiceSuite) TestGetProfile_Success() {
	user := &entity.User{Username: "riku", Email: "riku@mail.com", Role: entity.RoleUser}
	user.ID = "USR-1"
	s.users.On("FindByID", s.ctx, "USR-1").Return(user, nil).Once()

	profile, err := s.svc.GetProfile(s.ctx, "USR-1")

	s.Require().NoError(err)
	s.Equal("riku", profile.Username)
	s.Equal(entity.RoleUser, profile.Role)
}

func (s *UserServiceSuite) TestGetProfile_NotFound() {
	s.users.On("FindByID", s.ctx, "USR-X").Return(nil, apperror.NotFound("user not found")).Once()

	_, err := s.svc.GetProfile(s.ctx, "USR-X")

	s.ErrorIs(err, apperror.ErrNotFound)
}

func (s *UserServiceSuite) TestList_NormalizesPageAndMaps() {
	users := []entity.User{{Username: "a"}, {Username: "b"}}
	// Page 0 / size 1000 must be clamped before reaching the repository.
	s.users.On("List", s.ctx, model.PageRequest{Page: 1, PageSize: model.MaxPageSize}).
		Return(users, int64(250), nil).Once()

	page, err := s.svc.List(s.ctx, model.PageRequest{Page: 0, PageSize: 1000})

	s.Require().NoError(err)
	s.Len(page.Items, 2)
	s.Equal("b", page.Items[1].Username)
	s.Equal(3, page.TotalPages())
	s.True(page.HasNext())
}

func (s *UserServiceSuite) TestList_RepositoryError() {
	s.users.On("List", s.ctx, model.PageRequest{Page: 1, PageSize: model.DefaultPageSize}).
		Return(nil, int64(0), errors.New("db down")).Once()

	_, err := s.svc.List(s.ctx, model.PageRequest{})

	s.Error(err)
}

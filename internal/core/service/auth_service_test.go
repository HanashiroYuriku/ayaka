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

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type AuthServiceSuite struct {
	suite.Suite
	ctx    context.Context
	users  *mocks.UserRepo
	hasher *mocks.Hasher
	tokens *mocks.TokenProvider
	tx     *mocks.FakeTx
	svc    service.AuthService
}

func TestAuthServiceSuite(t *testing.T) {
	suite.Run(t, new(AuthServiceSuite))
}

func (s *AuthServiceSuite) SetupTest() {
	s.ctx = context.Background()
	s.users = new(mocks.UserRepo)
	s.hasher = new(mocks.Hasher)
	s.tokens = new(mocks.TokenProvider)
	s.tx = new(mocks.FakeTx)

	// NewAuthService pre-computes a dummy hash for timing-safe logins.
	s.hasher.On("Hash", "ayaka-dummy-password").Return("DUMMY-HASH", nil).Once()
	s.svc = service.NewAuthService(s.users, s.hasher, s.tokens, s.tx)
}

func (s *AuthServiceSuite) TearDownTest() {
	s.users.AssertExpectations(s.T())
	s.hasher.AssertExpectations(s.T())
	s.tokens.AssertExpectations(s.T())
}

// ============================================================ Register

func (s *AuthServiceSuite) TestRegister_Success() {
	in := model.RegisterInput{Username: " riku ", Email: " Riku@Mail.com ", Password: "P4$$word"}

	s.hasher.On("Hash", "P4$$word").Return("HASHED", nil).Once()
	s.users.On("Create", mock.Anything, mock.MatchedBy(func(u *entity.User) bool {
		return u.Username == "riku" &&
			u.Email == "riku@mail.com" &&
			u.Password == "HASHED" &&
			u.Role == entity.RoleUser &&
			u.ID != ""
	})).Return(nil).Once()

	profile, err := s.svc.Register(s.ctx, in)

	s.Require().NoError(err)
	s.Equal("riku@mail.com", profile.Email)
	s.Equal(1, s.tx.Count, "register should run inside a transaction")
}

func (s *AuthServiceSuite) TestRegister_HashFailure_IsInternal() {
	s.hasher.On("Hash", "P4$$word").Return("", errors.New("boom")).Once()

	_, err := s.svc.Register(s.ctx, model.RegisterInput{Password: "P4$$word"})

	s.ErrorIs(err, apperror.ErrInternal)
	s.users.AssertNotCalled(s.T(), "Create", mock.Anything, mock.Anything)
}

func (s *AuthServiceSuite) TestRegister_Conflict_IsPropagated() {
	conflict := apperror.Conflict("username or email already exists")
	s.hasher.On("Hash", mock.Anything).Return("HASHED", nil).Once()
	s.users.On("Create", mock.Anything, mock.Anything).Return(conflict).Once()

	_, err := s.svc.Register(s.ctx, model.RegisterInput{Password: "x"})

	s.ErrorIs(err, apperror.ErrConflict)
}

// ============================================================ Login

func (s *AuthServiceSuite) TestLogin_Success() {
	user := &entity.User{Email: "riku@mail.com", Password: "HASHED", Role: entity.RoleAdmin}
	user.ID = "USR-1"
	tokens := &model.AuthTokens{AccessToken: "a", RefreshToken: "r"}

	s.users.On("FindByEmail", s.ctx, "riku@mail.com").Return(user, nil).Once()
	s.hasher.On("Compare", "HASHED", "P4$$word").Return(nil).Once()
	s.tokens.On("Issue", "USR-1", entity.RoleAdmin).Return(tokens, nil).Once()

	got, err := s.svc.Login(s.ctx, model.LoginInput{Email: "RIKU@mail.com", Password: "P4$$word"})

	s.Require().NoError(err)
	s.Equal(tokens, got)
}

func (s *AuthServiceSuite) TestLogin_UnknownEmail_ReturnsGenericError() {
	s.users.On("FindByEmail", s.ctx, "ghost@mail.com").Return(nil, apperror.NotFound("user not found")).Once()
	// The dummy hash is still compared to keep timing constant.
	s.hasher.On("Compare", "DUMMY-HASH", "whatever").Return(errors.New("mismatch")).Once()

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "ghost@mail.com", Password: "whatever"})

	s.ErrorIs(err, service.ErrInvalidCredentials)
}

func (s *AuthServiceSuite) TestLogin_WrongPassword_ReturnsGenericError() {
	user := &entity.User{Password: "HASHED"}
	s.users.On("FindByEmail", s.ctx, "riku@mail.com").Return(user, nil).Once()
	s.hasher.On("Compare", "HASHED", "wrong").Return(errors.New("mismatch")).Once()

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "riku@mail.com", Password: "wrong"})

	s.ErrorIs(err, service.ErrInvalidCredentials)
	s.tokens.AssertNotCalled(s.T(), "Issue", mock.Anything, mock.Anything)
}

func (s *AuthServiceSuite) TestLogin_RepositoryFailure_IsPropagated() {
	dbErr := apperror.Internal(errors.New("connection refused"))
	s.users.On("FindByEmail", s.ctx, "riku@mail.com").Return(nil, dbErr).Once()

	_, err := s.svc.Login(s.ctx, model.LoginInput{Email: "riku@mail.com", Password: "x"})

	s.ErrorIs(err, apperror.ErrInternal)
	s.NotErrorIs(err, service.ErrInvalidCredentials)
}

// ============================================================ Refresh

func (s *AuthServiceSuite) TestRefresh_Success_UsesCurrentRole() {
	user := &entity.User{Role: entity.RoleAdmin}
	user.ID = "USR-1"
	claims := &model.TokenClaims{UserID: "USR-1", Role: entity.RoleUser, Type: model.TokenTypeRefresh}
	tokens := &model.AuthTokens{AccessToken: "a2", RefreshToken: "r2"}

	s.tokens.On("Parse", "refresh-token", model.TokenTypeRefresh).Return(claims, nil).Once()
	s.users.On("FindByID", s.ctx, "USR-1").Return(user, nil).Once()
	s.tokens.On("Issue", "USR-1", entity.RoleAdmin).Return(tokens, nil).Once()

	got, err := s.svc.Refresh(s.ctx, "refresh-token")

	s.Require().NoError(err)
	s.Equal(tokens, got)
}

func (s *AuthServiceSuite) TestRefresh_InvalidToken() {
	s.tokens.On("Parse", "bad", model.TokenTypeRefresh).Return(nil, apperror.Unauthorized("token expired")).Once()

	_, err := s.svc.Refresh(s.ctx, "bad")

	s.ErrorIs(err, service.ErrInvalidToken)
}

func (s *AuthServiceSuite) TestRefresh_DeletedUser() {
	claims := &model.TokenClaims{UserID: "USR-GONE"}
	s.tokens.On("Parse", "t", model.TokenTypeRefresh).Return(claims, nil).Once()
	s.users.On("FindByID", s.ctx, "USR-GONE").Return(nil, apperror.NotFound("user not found")).Once()

	_, err := s.svc.Refresh(s.ctx, "t")

	s.ErrorIs(err, service.ErrInvalidToken)
}

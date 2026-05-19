package repository_test

import (
	"be-ayaka/internal/adapter/repository"
	"be-ayaka/internal/core/customerrors"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/port"
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type UserRepoSuite struct {
	BaseRepoSuite
	repo port.UserRepository
}

func (s *UserRepoSuite) SetupSuite() {
	s.BaseRepoSuite.SetupSuite()
	s.repo = repository.NewUserRepo(s.DB)
}

func TestUserRepoSuite(t *testing.T) {
	suite.Run(t, &UserRepoSuite{
		BaseRepoSuite: NewBaseRepoSuite(&entity.User{}),
	})
}

// =============================================================================
// TEST FIND BY ID
// =============================================================================
func (s *UserRepoSuite) TestFindByID() {
	ctx := context.Background()

	dummy := &entity.User{
		Username: "riku",
		Email:    "riku@mail.com",
	}
	dummy.ID = "USER-1"
	s.DB.Create(dummy)

	// 1. success scenario
	s.Run("Success - Success find user", func() {
		res, err := s.repo.FindByID(ctx, dummy.ID)
		s.NoError(err)
		s.Equal("riku", res.Username)
	})

	// 2. failed scenario: user not found
	s.Run("Failed - ID User not found", func() {
		res, err := s.repo.FindByID(ctx, "unknown")
		s.Error(err)
		s.Nil(res)
		s.ErrorIs(err, customerrors.ErrDataNotFound)
	})
}

// =============================================================================
// TEST FIND BY EMAIL OR Username
// =============================================================================

func (s *UserRepoSuite) TestFindByEmail() {
	ctx := context.Background()

	dummy := &entity.User{
		Username: "riku",
		Email:    "riku@mail.com",
	}
	dummy.ID = "USER-001"
	s.DB.Create(dummy)

	// 1. success scenario using email
	s.Run("Success - Find by email", func() {
		res, err := s.repo.FindByEmail(ctx, "riku@mail.com")
		s.NoError(err)
		s.Equal("riku", res.Username)
	})

	// 2. failed scenario: user not found
	s.Run("Failed - Not found", func() {
		res, err := s.repo.FindByEmail(ctx, "unknown")
		s.Error(err)
		s.Nil(res)
		s.ErrorIs(err, customerrors.ErrInvalidCredentials)
	})
}

// =============================================================================
// TEST CREATE USER
// =============================================================================
// 1. success scenario
func (s *UserRepoSuite) TestCreate_Success() {
	ctx := context.Background()
	newUser := &entity.User{
		Username: "hanashiroyuriku",
		Email:    "yuriku@mail.com",
		Password: "hashedpassword",
	}
	newUser.ID = "USER-123"

	err := s.repo.Create(ctx, newUser)
	s.NoError(err)

	var res entity.User
	s.DB.First(&res, "id = ?", "USER-123")
	s.Equal("hanashiroyuriku", res.Username)
	s.Equal("yuriku@mail.com", res.Email)
}

// 2. success scenario with Tx
func (s *UserRepoSuite) TestCreate_SuccessWithTx() {
	tx := s.DB.Begin()
	ctxWithTx := context.WithValue(context.Background(), repository.TxKey{}, tx)

	newUser := &entity.User{
		Username: "riku",
		Email:    "riku@mail.com",
	}
	newUser.ID = "USER-1"

	err := s.repo.Create(ctxWithTx, newUser)
	s.NoError(err)

	tx.Rollback()

	var res entity.User
	errFind := s.DB.First(&res, "id = ?", "USER-1").Error
	s.Error(errFind)
	s.Equal(gorm.ErrRecordNotFound, errFind)
}

// 3. failed scenario: email duplicate
func (s *UserRepoSuite) TestCreate_Failed_DuplicateEmail() {
	ctx := context.Background()

	user1 := &entity.User{
		Username: "user1",
		Email:    "test@mail.com",
	}
	user1.ID = "U1"
	s.DB.Create(user1)

	user2 := &entity.User{
		Username: "user2",
		Email:    "test@mail.com",
	}
	user2.ID = "U2"

	err := s.repo.Create(ctx, user2)
	s.Error(err)
	s.Contains(err.Error(), "UNIQUE constraint failed")
}

// 4. failed scenario: context canceled
func (s *UserRepoSuite) TestCreate_Failed_ContextCanceled() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	newUser := &entity.User{
		Username: "canceled",
		Email:    "canceled@mail.com",
	}
	newUser.ID = "USER-CANCEL"

	err := s.repo.Create(ctx, newUser)
	s.Error(err)
}

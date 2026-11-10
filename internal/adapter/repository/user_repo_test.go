package repository_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"be-ayaka/internal/adapter/repository"
	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/model"

	"github.com/stretchr/testify/suite"
)

type UserRepoSuite struct {
	BaseRepoSuite
	repo *repository.UserRepo
	ctx  context.Context
}

func TestUserRepoSuite(t *testing.T) {
	suite.Run(t, &UserRepoSuite{BaseRepoSuite: NewBaseRepoSuite(&entity.User{})})
}

func (s *UserRepoSuite) SetupTest() {
	s.BaseRepoSuite.SetupTest()
	s.repo = repository.NewUserRepo(s.DB)
	s.ctx = context.Background()
}

func (s *UserRepoSuite) seed(users ...*entity.User) {
	for _, u := range users {
		s.Require().NoError(s.DB.Create(u).Error)
	}
}

// ============================================================ FindByID

func (s *UserRepoSuite) TestFindByID() {
	s.seed(newUser("U-1", "riku"))

	s.Run("found", func() {
		res, err := s.repo.FindByID(s.ctx, "U-1")
		s.Require().NoError(err)
		s.Equal("riku", res.Username)
	})

	s.Run("not found", func() {
		res, err := s.repo.FindByID(s.ctx, "unknown")
		s.Nil(res)
		s.ErrorIs(err, repository.ErrUserNotFound)
		s.ErrorIs(err, apperror.ErrNotFound)
	})

	s.Run("soft deleted is not found", func() {
		s.seed(newUser("U-DEL", "deleted"))
		s.Require().NoError(s.DB.Delete(&entity.User{}, "id = ?", "U-DEL").Error)

		_, err := s.repo.FindByID(s.ctx, "U-DEL")
		s.ErrorIs(err, apperror.ErrNotFound)
	})
}

// ============================================================ FindByEmail

func (s *UserRepoSuite) TestFindByEmail() {
	s.seed(newUser("U-1", "riku"))

	res, err := s.repo.FindByEmail(s.ctx, "riku@mail.com")
	s.Require().NoError(err)
	s.Equal("U-1", res.ID)

	_, err = s.repo.FindByEmail(s.ctx, "ghost@mail.com")
	s.ErrorIs(err, apperror.ErrNotFound)
}

// ============================================================ Create

func (s *UserRepoSuite) TestCreate_Success() {
	s.Require().NoError(s.repo.Create(s.ctx, newUser("U-1", "yuriku")))

	res, err := s.repo.FindByID(s.ctx, "U-1")
	s.Require().NoError(err)
	s.Equal("yuriku@mail.com", res.Email)
}

func (s *UserRepoSuite) TestCreate_DuplicateEmail_ReturnsConflict() {
	s.seed(newUser("U-1", "riku"))
	dup := newUser("U-2", "other")
	dup.Email = "riku@mail.com"

	err := s.repo.Create(s.ctx, dup)

	s.ErrorIs(err, repository.ErrUserConflict)
	s.ErrorIs(err, apperror.ErrConflict)
}

func (s *UserRepoSuite) TestCreate_RespectsTransactionInContext() {
	tx := s.DB.Begin()
	ctx := repository.ContextWithTx(s.ctx, tx)

	s.Require().NoError(s.repo.Create(ctx, newUser("U-1", "riku")))
	tx.Rollback()

	_, err := s.repo.FindByID(s.ctx, "U-1")
	s.ErrorIs(err, apperror.ErrNotFound)
}

func (s *UserRepoSuite) TestCreate_ContextCanceled() {
	ctx, cancel := context.WithCancel(s.ctx)
	cancel()

	s.Error(s.repo.Create(ctx, newUser("U-1", "canceled")))
}

// ============================================================ List

func (s *UserRepoSuite) TestList_PaginatesNewestFirst() {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 1; i <= 5; i++ {
		u := newUser(fmt.Sprintf("U-%d", i), fmt.Sprintf("user%d", i))
		u.CreatedAt = base.Add(time.Duration(i) * time.Hour)
		s.seed(u)
	}

	users, total, err := s.repo.List(s.ctx, model.PageRequest{Page: 1, PageSize: 2})
	s.Require().NoError(err)
	s.EqualValues(5, total)
	s.Require().Len(users, 2)
	s.Equal("U-5", users[0].ID)
	s.Equal("U-4", users[1].ID)

	users, _, err = s.repo.List(s.ctx, model.PageRequest{Page: 3, PageSize: 2})
	s.Require().NoError(err)
	s.Require().Len(users, 1)
	s.Equal("U-1", users[0].ID)
}

func (s *UserRepoSuite) TestList_Empty() {
	users, total, err := s.repo.List(s.ctx, model.PageRequest{Page: 1, PageSize: 10})

	s.Require().NoError(err)
	s.Zero(total)
	s.NotNil(users)
	s.Empty(users)
}

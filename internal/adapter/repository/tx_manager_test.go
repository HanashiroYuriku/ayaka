package repository_test

import (
	"context"
	"errors"
	"testing"

	"be-ayaka/internal/adapter/repository"
	"be-ayaka/internal/core/apperror"
	"be-ayaka/internal/core/entity"

	"github.com/stretchr/testify/suite"
)

type TxManagerSuite struct {
	BaseRepoSuite
	tx    *repository.TxManager
	users *repository.UserRepo
}

func TestTxManagerSuite(t *testing.T) {
	suite.Run(t, &TxManagerSuite{BaseRepoSuite: NewBaseRepoSuite(&entity.User{})})
}

func (s *TxManagerSuite) SetupTest() {
	s.BaseRepoSuite.SetupTest()
	s.tx = repository.NewTxManager(s.DB)
	s.users = repository.NewUserRepo(s.DB)
}

func newUser(id, username string) *entity.User {
	u := &entity.User{Username: username, Email: username + "@mail.com", Password: "x", Role: entity.RoleUser}
	u.ID = id
	return u
}

func (s *TxManagerSuite) TestWithTx_Commit() {
	ctx := context.Background()

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		return s.users.Create(ctx, newUser("U-1", "commit"))
	})

	s.Require().NoError(err)
	_, err = s.users.FindByID(ctx, "U-1")
	s.NoError(err)
}

func (s *TxManagerSuite) TestWithTx_RollbackOnError() {
	ctx := context.Background()
	forced := errors.New("simulated failure")

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		s.Require().NoError(s.users.Create(ctx, newUser("U-2", "rollback")))
		return forced
	})

	s.ErrorIs(err, forced)
	_, err = s.users.FindByID(ctx, "U-2")
	s.ErrorIs(err, apperror.ErrNotFound)
}

func (s *TxManagerSuite) TestWithTx_NestedCallsShareTransaction() {
	ctx := context.Background()
	forced := errors.New("outer failure")

	err := s.tx.WithTx(ctx, func(ctx context.Context) error {
		inner := s.tx.WithTx(ctx, func(ctx context.Context) error {
			return s.users.Create(ctx, newUser("U-3", "nested"))
		})
		s.Require().NoError(inner)
		return forced
	})

	s.ErrorIs(err, forced)
	_, err = s.users.FindByID(ctx, "U-3")
	s.ErrorIs(err, apperror.ErrNotFound, "inner write must roll back with the outer transaction")
}

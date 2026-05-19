package repository_test

import (
	"be-ayaka/internal/adapter/repository"
	"be-ayaka/internal/core/entity"
	"be-ayaka/internal/core/port"
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

type TxManagerSuite struct {
	BaseRepoSuite
	manager port.TxManager
}

func (s *TxManagerSuite) SetupSuite() {
	s.BaseRepoSuite.SetupSuite()
	s.manager = repository.NewTxManager(s.DB)
}

func TestTxManagerSuite(t *testing.T) {
	suite.Run(t, &TxManagerSuite{
		BaseRepoSuite: NewBaseRepoSuite(&entity.User{}),
	})
}

// =============================================================================
// 1. success scenario
// =============================================================================
func (s *TxManagerSuite) TestWithTx_SuccessCommit() {
	ctx := context.Background()

	err := s.manager.WithTx(ctx, func(txCtx context.Context) error {
		// Pastikan ExtractTx berhasil mengambil instance transaksi dari context
		txDB := repository.ExtractTx(txCtx, s.DB)

		dummyUser := &entity.User{Name: "tx_commit_user"}
		dummyUser.ID = "TX-COMMIT-123"

		return txDB.Create(dummyUser).Error
	})

	s.NoError(err)

	var res entity.User
	errFind := s.DB.First(&res, "id = ?", "TX-COMMIT-123").Error
	s.NoError(errFind)
	s.Equal("tx_commit_user", res.Name)
}

// =============================================================================
// 2. failed scenario
// =============================================================================
func (s *TxManagerSuite) TestWithTx_ForcedRollback() {
	ctx := context.Background()
	forcedError := errors.New("simulated database error")

	err := s.manager.WithTx(ctx, func(txCtx context.Context) error {
		txDB := repository.ExtractTx(txCtx, s.DB)

		dummyUser := &entity.User{Name: "tx_rollback_user"}
		dummyUser.ID = "TX-ROLLBACK-123"
		
		txDB.Create(dummyUser)

		return forcedError
	})

	s.ErrorIs(err, forcedError)

	var res entity.User
	errFind := s.DB.First(&res, "id = ?", "TX-ROLLBACK-123").Error
	s.ErrorIs(errFind, gorm.ErrRecordNotFound)
}

// =============================================================================
// 3. ExtractTx returns default DB when no transaction in context
// =============================================================================
func (s *TxManagerSuite) TestExtractTx_ReturnDefaultDB() {
	ctx := context.Background()

	db := repository.ExtractTx(ctx, s.DB)
	
	s.Equal(s.DB, db)
}